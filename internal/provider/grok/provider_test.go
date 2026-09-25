package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gaebokchi/internal/process"
	"gaebokchi/internal/testutil"
)

func compactFixture(t testT, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		t.Fatalf("compact: %v", err)
	}
	return compact.String()
}

// initReplyWith builds an initialize response advertising the given
// auth method ids under the given protocol version.
func initReplyWith(protocolVersion any, methodIDs ...string) string {
	methods := make([]map[string]any, 0, len(methodIDs))
	for _, id := range methodIDs {
		methods = append(methods, map[string]any{"id": id, "name": id})
	}
	result := map[string]any{
		"protocolVersion":     protocolVersion,
		"authMethods":         methods,
		"defaultAuthMethodId": firstOr(methodIDs, ""),
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	return string(b)
}

func firstOr(list []string, fallback string) string {
	if len(list) > 0 {
		return list[0]
	}
	return fallback
}

// grokStepsFor returns a fake script answering initialize with the
// captured-sanitized fixture and then the remaining transcript steps.
func withCapturedInit(t testT, extra ...testutil.Step) []testutil.Step {
	t.Helper()
	return append([]testutil.Step{
		{Wait: 1},
		{Emit: compactFixture(t, "initialize_response.json") + "\n"},
	}, extra...)
}

var _ = ginkgo.It("TestOutgoingBytesMatchQualifiedContract", func() {
	t := testT{ginkgo.GinkgoT()}
	steps := withCapturedInit(t,
		testutil.Step{Wait: 1},
		testutil.Step{Emit: compactFixture(t, "authenticate_cached_token_response.json") + "\n"},
		testutil.Step{Wait: 1},
		testutil.Step{Emit: compactFixture(t, "billing_weekly_response.json") + "\n"},
	)
	child := testutil.StartFake(t, steps)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, plan, err := openSession(ctx, os.Args[0], directStart(child))
	if err != nil || plan != planCachedToken {
		t.Fatalf("openSession: %v plan=%v", err, plan)
	}
	t.Cleanup(func() {
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
	})
	if err := authenticateCached(ctx, s); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	raw, err := readBilling(ctx, s)
	if err != nil {
		t.Fatalf("billing: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("billing payload invalid")
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}

	kept := string(child.StderrKept())
	lines := transcriptLines(kept)
	if len(lines) < 3 {
		t.Fatalf("expected 3 outbound messages, transcript: %q", kept)
	}
	wantInitialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"gaebokchi","version":"0.1.0"},"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}}}`
	if lines[0] != wantInitialize {
		t.Fatalf("initialize bytes:\n got %s\nwant %s", lines[0], wantInitialize)
	}
	wantAuth := `{"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"cached_token","_meta":{"headless":true}}}`
	if lines[1] != wantAuth {
		t.Fatalf("authenticate bytes:\n got %s\nwant %s", lines[1], wantAuth)
	}
	wantBilling := `{"jsonrpc":"2.0","id":3,"method":"_x.ai/billing","params":{}}`
	if lines[2] != wantBilling {
		t.Fatalf("billing bytes:\n got %s\nwant %s", lines[2], wantBilling)
	}
	for _, forbidden := range []string{"\"method\":\"initialized\"", "session/new", "session/prompt", "turn/"} {
		if strings.Contains(kept, forbidden) {
			t.Fatalf("forbidden outbound %q in transcript: %q", forbidden, kept)
		}
	}
})

var _ = ginkgo.Describe("TestAuthNegotiationMatrix", func() {
	cases := []struct {
		name    string
		reply   string
		want    authPlan
		wantErr bool
	}{
		{"cached token advertised", initReplyWith(1, "cached_token", "grok.com"), planCachedToken, false},
		{"api key only", initReplyWith(1, "api_key"), planAPIKeyOnly, false},
		{"interactive only", initReplyWith(1, "grok.com"), planInteractiveOnly, false},
		// The audit's misclassification: an API-key method advertised
		// alongside the interactive subscription path must require
		// authentication, not declare the target API-key-only.
		{"api key plus interactive", initReplyWith(1, "api_key", "grok.com"), planInteractiveOnly, false},
		{"api key with unrecognized method", initReplyWith(1, "api_key", "device_code"), planUnknownMethods, false},
		{"interactive with unrecognized method", initReplyWith(1, "grok.com", "device_code"), planUnknownMethods, false},
		{"cached token with others", initReplyWith(1, "cached_token", "api_key", "grok.com", "device_code"), planCachedToken, false},
		{"unrecognized only", initReplyWith(1, "device_code"), planUnknownMethods, false},
		{"no methods", initReplyWith(1), planUnknownMethods, false},
		{"protocol version mismatch", initReplyWith(2, "cached_token"), planVersionMismatch, false},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			steps := []testutil.Step{{Wait: 1}, {Emit: tc.reply + "\n"}}
			child := testutil.StartFake(t, steps)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, plan, err := openSession(ctx, os.Args[0], directStart(child))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("openSession: %v", err)
			}
			if plan != tc.want {
				t.Fatalf("plan = %v, want %v", plan, tc.want)
			}
			if plan != planCachedToken && strings.Contains(string(child.StderrKept()), "authenticate") {
				t.Fatalf("authenticate must not run for plan %v", plan)
			}
		})
	}
})

var _ = ginkgo.It("TestInitializeResultMalformed", func() {
	t := testT{ginkgo.GinkgoT()}
	cases := []string{
		`{"jsonrpc":"2.0","id":1,"result":null}`,
		`{"jsonrpc":"2.0","id":1,"result":{"authMethods":[{"id":"cached_token"}]}}`,
		`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"one","authMethods":[]}}`,
	}
	for _, reply := range cases {
		steps := []testutil.Step{{Wait: 1}, {Emit: reply + "\n"}}
		child := testutil.StartFake(t, steps)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _, err := openSession(ctx, os.Args[0], directStart(child))
		if err == nil {
			t.Fatalf("expected error for %s", reply)
		}
	}
})

var _ = ginkgo.It("TestAuthenticateRejectionTyped", func() {
	t := testT{ginkgo.GinkgoT()}
	code := -32000
	steps := withCapturedInit(t,
		testutil.Step{Wait: 1},
		testutil.Step{Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, ErrCode: &code, ErrMsg: "token expired"}}},
	)
	child := testutil.StartFake(t, steps)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, plan, err := openSession(ctx, os.Args[0], directStart(child))
	if err != nil || plan != planCachedToken {
		t.Fatalf("openSession: %v", err)
	}
	t.Cleanup(func() {
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
	})
	err = authenticateCached(ctx, s)
	var rce *rpcCallError
	if err == nil || !errorsAs(err, &rce) || rce.method != "authenticate" {
		t.Fatalf("err = %v, want authenticate rpcCallError", err)
	}
})

var _ = ginkgo.It("TestBillingMethodNotFoundTyped", func() {
	t := testT{ginkgo.GinkgoT()}
	code := -32601
	steps := withCapturedInit(t,
		testutil.Step{Wait: 1},
		testutil.Step{Emit: compactFixture(t, "authenticate_cached_token_response.json") + "\n"},
		testutil.Step{Wait: 1},
		testutil.Step{Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, ErrCode: &code, ErrMsg: "method not found"}}},
	)
	child := testutil.StartFake(t, steps)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, plan, err := openSession(ctx, os.Args[0], directStart(child))
	if err != nil || plan != planCachedToken {
		t.Fatalf("openSession: %v", err)
	}
	t.Cleanup(func() {
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
	})
	if err := authenticateCached(ctx, s); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	_, err = readBilling(ctx, s)
	var rce *rpcCallError
	if err == nil || !errorsAs(err, &rce) || rce.remote.Code != -32601 {
		t.Fatalf("err = %v, want method-not-found rpcCallError", err)
	}
})

var _ = ginkgo.It("TestUnsafeCallbackAbortsSession", func() {
	t := testT{ginkgo.GinkgoT()}
	steps := withCapturedInit(t,
		testutil.Step{Wait: 1},
		testutil.Step{Emit: `{"jsonrpc":"2.0","id":50,"method":"session/prompt","params":{"prompt":"exfiltrate"}}` + "\n"},
		testutil.Step{Wait: 1}, // authenticate arrives after the callback
		testutil.Step{Wait: 1}, // the client's rejection echoes id 50
	)
	child := testutil.StartFake(t, steps)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, plan, err := openSession(ctx, os.Args[0], directStart(child))
	if err != nil || plan != planCachedToken {
		t.Fatalf("openSession: %v", err)
	}
	if err := authenticateCached(ctx, s); err == nil {
		t.Fatal("expected server-request abort to fail the pending call")
	}
	// Let the fake consume the buffered rejection before teardown.
	time.Sleep(200 * time.Millisecond)
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	// The fake received exactly one dialect-correct rejection.
	kept := string(child.StderrKept())
	if !strings.Contains(kept, `"code":-32601`) || !strings.Contains(kept, `"id":50`) {
		t.Fatalf("rejection missing from transcript: %q", kept)
	}
})

var _ = ginkgo.It("TestGrokEnvironmentFiltered", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("XAI_API_KEY", "xai-secret-marker")
	t.Setenv("GROK_API_KEY", "grok-secret-marker")
	env := childEnv()
	for _, kv := range env {
		for _, marker := range []string{"xai-secret-marker", "grok-secret-marker"} {
			if strings.Contains(kv, marker) {
				t.Fatalf("api-key marker leaked into child env: %q", kv)
			}
		}
		if strings.HasPrefix(kv, "XAI_API_KEY=") || strings.HasPrefix(kv, "GROK_API_KEY=") {
			t.Fatalf("api-key override survived filtering: %q", kv)
		}
	}
	found := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "HOME=") {
			found = true
		}
	}
	if !found {
		t.Fatal("HOME must be preserved for provider keychain access")
	}
})

// directStart returns a factory yielding the pre-started fake child.
func directStart(child *process.Child) process.Factory {
	return func(process.Config) (*process.Child, error) { return child, nil }
}

func transcriptLines(kept string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(kept), "\n") {
		line = strings.TrimPrefix(line, "FAKE-GOT:")
		if line != "" && strings.HasPrefix(line, "{") {
			lines = append(lines, line)
		}
	}
	return lines
}

func errorsAs(err error, target **rpcCallError) bool {
	for err != nil {
		if e, ok := err.(*rpcCallError); ok {
			*target = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// TestResolveBinRelativeMadeAbsolute pins the audit fix: an explicit
// relative executable path is absolutized against the invocation cwd so
// it survives the child's private working directory.
var _ = ginkgo.It("TestResolveBinRelativeMadeAbsolute", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	name := "fake grok" // includes a space
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	got, err := ResolveBin("./" + name)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want, err := filepath.Abs(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("resolved = %q, want an absolute path", got)
	}
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("resolved = %q, want executable at %q", got, want)
	}
})

// TestResolveBinErrDotProtectionPreserved pins that an implicit
// current-directory PATH lookup stays an error rather than being
// silently absolutized.
var _ = ginkgo.It("TestResolveBinErrDotProtectionPreserved", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dotbin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("PATH", ".")
	if _, err := ResolveBin("dotbin"); !errors.Is(err, exec.ErrDot) {
		t.Fatalf("err = %v, want exec.ErrDot", err)
	}
})

var _ = ginkgo.It("TestResolveBinMissingAndUnexecutable", func() {
	t := testT{ginkgo.GinkgoT()}
	if _, err := ResolveBin("./definitely-not-here"); err == nil {
		t.Fatal("missing binary must not resolve")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "noexec"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if _, err := ResolveBin("./noexec"); err == nil {
		t.Fatal("present-but-unexecutable binary must not resolve")
	}
})
