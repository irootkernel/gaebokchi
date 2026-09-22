package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/testutil"
)

// fullScript builds a fake transcript answering the complete metadata
// sequence: initialize, account/read (chatgpt+email), rateLimits/read.
func fullScript(t *testing.T, accountFixture, quotaFixture string) []testutil.Step {
	t.Helper()
	return []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: compactFixture(t, accountFixture) + "\n"},
		{Wait: 1},
		{Emit: compactFixture(t, quotaFixture) + "\n"},
	}
}

func compactFixture(t *testing.T, name string) string {
	t.Helper()
	var envelope map[string]json.RawMessage
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	// Renumber ids to the per-connection sequence 1,2,3.
	var compact map[string]any
	if err := json.Unmarshal(b, &compact); err != nil {
		t.Fatal(err)
	}
	delete(compact, "id")
	switch name {
	case "account_chatgpt_home1.json":
		compact["id"] = 2
	case "account_chatgpt_home2.json":
		compact["id"] = 2
	case "ratelimits_weekly_home1.json", "ratelimits_weekly_home2.json":
		compact["id"] = 3
	}
	out, err := json.Marshal(compact)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func codexTarget(home string) domain.Target {
	return domain.Target{
		ID: "codex:0", Index: 0, Provider: domain.ProviderCodex,
		Home: home, FallbackAccount: filepath.Base(home), Bin: os.Args[0],
	}
}

// providerOver returns a Provider whose process factory always yields
// the pre-started fake child, so Query's lifecycle runs against it.
func providerOver(child *process.Child) *Provider {
	return &Provider{Start: func(process.Config) (*process.Child, error) { return child, nil }}
}

func TestQueryFullLifecycleFromFixtures(t *testing.T) {
	home := t.TempDir()
	child := testutil.StartFake(t, fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json"))
	p := providerOver(child)
	res := p.Query(context.Background(), codexTarget(home))
	if res.Status != domain.StatusOK {
		t.Fatalf("status = %v (%s)", res.Status, res.Diagnostic.Message)
	}
	if res.UsedPercent == nil || *res.UsedPercent != 37 {
		t.Fatalf("percent = %v", res.UsedPercent)
	}
	if res.Account != "account1@example.com" || res.AccountSource != domain.AccountSourceProtocolEmail {
		t.Fatalf("account = %q (%v)", res.Account, res.AccountSource)
	}
	if res.Period != domain.PeriodWeekly || res.QuotaID != "codex" || res.Source != domain.SourceCodexAppServer {
		t.Fatalf("meta = %v/%q/%q", res.Period, res.QuotaID, res.Source)
	}
	if res.ResetAt == nil {
		t.Fatal("reset metadata not preserved")
	}
	child.Close()
	kept := string(child.StderrKept())
	if !containsStr(kept, `"method":"account/rateLimits/read"`) {
		t.Fatalf("quota read missing from transcript: %q", kept)
	}
}

func TestQueryTwoHomesDifferentValues(t *testing.T) {
	homeA := t.TempDir()
	homeB := t.TempDir()
	childA := testutil.StartFake(t, fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json"))
	childB := testutil.StartFake(t, fullScript(t, "account_chatgpt_home2.json", "ratelimits_weekly_home2.json"))
	pa := providerOver(childA)
	pb := providerOver(childB)
	tgtA := codexTarget(homeA)
	tgtB := codexTarget(homeB)
	tgtB.ID = "codex:1"
	tgtB.Index = 1

	resB := pb.Query(context.Background(), tgtB)
	resA := pa.Query(context.Background(), tgtA)
	if resA.UsedPercent == nil || resB.UsedPercent == nil || *resA.UsedPercent == *resB.UsedPercent {
		t.Fatalf("values must differ per home: %v / %v", resA.UsedPercent, resB.UsedPercent)
	}
	if resA.Account == resB.Account {
		t.Fatalf("accounts must differ per home: %q", resA.Account)
	}
}

func TestQueryHomePreflight(t *testing.T) {
	p := New()
	missing := p.Query(context.Background(), codexTarget("/definitely/not/here"))
	if missing.Status != domain.StatusHomeNotFound {
		t.Fatalf("missing home status = %v", missing.Status)
	}
	file := filepath.Join(t.TempDir(), "plainfile")
	os.WriteFile(file, []byte("x"), 0o644)
	invalid := p.Query(context.Background(), codexTarget(file))
	if invalid.Status != domain.StatusHomeError {
		t.Fatalf("invalid home status = %v", invalid.Status)
	}
}

func TestQueryMissingExecutable(t *testing.T) {
	p := New()
	tgt := codexTarget(t.TempDir())
	tgt.Bin = "/nonexistent/gaebokchi-codex-missing"
	res := p.Query(context.Background(), tgt)
	if res.Status != domain.StatusCLINotFound {
		t.Fatalf("status = %v", res.Status)
	}
	if res.Account == "" {
		t.Fatal("fallback alias must be preserved")
	}
}

func TestQueryAuthOutcomesShortCircuit(t *testing.T) {
	accountPayload := func(account string) []testutil.Step {
		return []testutil.Step{
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
			{Wait: 1},
			{Emit: `{"id":2,"result":{"account":` + account + `}}` + "\n"},
		}
	}
	cases := []struct {
		name    string
		account string
		want    domain.Status
	}{
		{"absent", "null", domain.StatusAuthRequired},
		{"api key", `{"type":"apiKey"}`, domain.StatusNotApplicableAPIKey},
		{"unknown", `{"type":"device"}`, domain.StatusUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			child := testutil.StartFake(t, accountPayload(tc.account))
			p := providerOver(child)
			res := p.Query(context.Background(), codexTarget(t.TempDir()))
			if res.Status != tc.want {
				t.Fatalf("status = %v, want %v", res.Status, tc.want)
			}
			if res.AccountSource != domain.AccountSourceConfiguredAlias {
				t.Fatalf("identity source = %v", res.AccountSource)
			}
			// No quota read occurs after these outcomes.
			if containsStr(string(child.StderrKept()), "rateLimits") {
				t.Fatal("quota read must not run without a chatgpt account")
			}
		})
	}
}

// TestQueryValidResultSurvivesSlowCleanup reproduces the audit's
// executable-level finding at the provider boundary: every response is
// routed well inside the query budget, but the provider lingers after
// its final reply, so deferred cleanup crosses the deadline inside
// Query. The numeric result must survive; only a response established
// after the deadline may time out.
func TestQueryValidResultSurvivesSlowCleanup(t *testing.T) {
	steps := fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json")
	steps = append(steps, testutil.Step{SleepMs: 600}) // linger like a slow provider
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res := p.Query(ctx, codexTarget(t.TempDir()))
	if res.Status != domain.StatusOK {
		t.Fatalf("status = %v (%s); timely result discarded for slow cleanup", res.Status, res.Diagnostic.Message)
	}
	if res.UsedPercent == nil || *res.UsedPercent != 37 {
		t.Fatalf("percent = %v", res.UsedPercent)
	}
}

func TestQueryTimeoutClassification(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{SleepMs: 30000}, // never answers initialize
	}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res := p.Query(ctx, codexTarget(t.TempDir()))
	if res.Status != domain.StatusTimeout {
		t.Fatalf("status = %v (%s)", res.Status, res.Diagnostic.Message)
	}
	if res.Account == "" {
		t.Fatal("fallback alias lost on timeout")
	}
	if err := child.Close(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

func TestQueryProtocolErrorClassification(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: "banner-not-json\n"},
	}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	res := p.Query(context.Background(), codexTarget(t.TempDir()))
	if res.Status != domain.StatusProtocolError {
		t.Fatalf("status = %v", res.Status)
	}
	if containsStr(res.Diagnostic.Message, "banner") {
		t.Fatalf("raw provider data leaked into diagnostic: %q", res.Diagnostic.Message)
	}
}

func containsStr(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// TestResolveBinRelativeMadeAbsolute pins the audit fix: an explicit
// relative executable path is absolutized against the invocation cwd so
// it survives the child's private working directory.
func TestResolveBinRelativeMadeAbsolute(t *testing.T) {
	dir := t.TempDir()
	name := "fake provider" // includes a space
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
	if got != want {
		t.Fatalf("resolved = %q, want absolute %q", got, want)
	}
}

// TestResolveBinErrDotProtectionPreserved pins that an implicit
// current-directory PATH lookup stays an error rather than being
// silently absolutized.
func TestResolveBinErrDotProtectionPreserved(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dotbin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("PATH", ".")
	if _, err := ResolveBin("dotbin"); !errors.Is(err, exec.ErrDot) {
		t.Fatalf("err = %v, want exec.ErrDot", err)
	}
}

func TestResolveBinMissingAndUnexecutable(t *testing.T) {
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
}

// TestInitializeResultShapeValidation pins the audit's finding that any
// shapeless initialize response used to complete the handshake: a
// malformed or shape-missing reply must yield a protocol error, while
// the qualified shape (with additive members) proceeds.
func TestInitializeResultShapeValidation(t *testing.T) {
	cases := []struct {
		name string
		init string
		want domain.Status
	}{
		{"error null without result", `{"id":1,"error":null}`, domain.StatusProtocolError},
		{"result null", `{"id":1,"result":null}`, domain.StatusProtocolError},
		{"empty object", `{"id":1,"result":{}}`, domain.StatusProtocolError},
		{"bare scalar", `{"id":1,"result":"macos"}`, domain.StatusProtocolError},
		{"array result", `{"id":1,"result":[{"platformOs":"macos"}]}`, domain.StatusProtocolError},
		{"platformOs not a string", `{"id":1,"result":{"platformOs":7}}`, domain.StatusProtocolError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := []testutil.Step{{Wait: 1}, {Emit: tc.init + "\n"}}
			child := testutil.StartFake(t, steps)
			res := providerOver(child).Query(context.Background(), codexTarget(t.TempDir()))
			if res.Status != tc.want {
				t.Fatalf("status = %v (%s), want %v", res.Status, res.Diagnostic.Message, tc.want)
			}
			if containsStr(res.Diagnostic.Message, "macos") {
				t.Fatalf("raw provider data leaked into diagnostic: %q", res.Diagnostic.Message)
			}
		})
	}
	t.Run("qualified shape with additive members proceeds", func(t *testing.T) {
		init := `{"id":1,"result":{"platformOs":"macos","futureField":{"x":1}}}`
		steps := []testutil.Step{
			{Wait: 1}, {Emit: init + "\n"},
			{Capture: 1}, // initialized notification
			{Wait: 1},    // account/read
			{Emit: `{"id":2,"result":{"account":{"type":"apiKey"}}}` + "\n"},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), codexTarget(t.TempDir()))
		if res.Status != domain.StatusNotApplicableAPIKey {
			t.Fatalf("status = %v (%s); qualified shape must complete the handshake", res.Status, res.Diagnostic.Message)
		}
	})
}

// startFakeIn starts a scripted fake child whose private directory
// lives under the given parent, so a test that owns the parent can make
// removal fail deterministically.
func startFakeIn(t *testing.T, parent string, steps []testutil.Step) *process.Child {
	t.Helper()
	script, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("marshal fake script: %v", err)
	}
	c, err := process.Start(process.Config{
		Bin:        os.Args[0],
		Args:       []string{"-test.run=TestFakeHelperProcess", "--", "fake"},
		Env:        append(os.Environ(), "GAEBOKCHI_FAKE=1", "GAEBOKCHI_FAKE_STEPS="+string(script)),
		TempParent: parent,
	})
	if err != nil {
		t.Fatalf("start fake child: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestQueryCleanupFailureFlaggedOnSuccess pins the propagation contract
// on a successful query: the measured percentage stays, CleanupFailed
// is set, and the safe fixed diagnostic replaces the empty one.
func TestQueryCleanupFailureFlaggedOnSuccess(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based removal failure does not apply as root")
	}
	parent := t.TempDir()
	child := startFakeIn(t, parent, fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json"))
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Skipf("cannot chmod temp parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	res := providerOver(child).Query(context.Background(), codexTarget(t.TempDir()))
	if res.Status != domain.StatusOK || res.UsedPercent == nil || *res.UsedPercent != 37 {
		t.Fatalf("measurement lost: status=%v percent=%v", res.Status, res.UsedPercent)
	}
	if !res.CleanupFailed {
		t.Fatal("cleanup failure not propagated to the result")
	}
	if res.Diagnostic.Code != "cleanup_failed" || res.Diagnostic.Message != "provider cleanup incomplete" {
		t.Fatalf("diagnostic = %+v", res.Diagnostic)
	}
}

// TestQueryCleanupFailureFlaggedViaRPCAbort pins that cleanup failures
// initiated by the RPC abort path (a timed-out query whose transport
// teardown cannot complete) still reach the result: the timeout keeps
// its primary diagnostic and gains the CleanupFailed flag.
func TestQueryCleanupFailureFlaggedViaRPCAbort(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based removal failure does not apply as root")
	}
	parent := t.TempDir()
	child := startFakeIn(t, parent, []testutil.Step{{Wait: 1}, {SleepMs: 60000}})
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Skipf("cannot chmod temp parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res := providerOver(child).Query(ctx, codexTarget(t.TempDir()))
	if res.Status != domain.StatusTimeout {
		t.Fatalf("status = %v, want timeout", res.Status)
	}
	if res.Diagnostic.Code != "timeout" {
		t.Fatalf("primary diagnostic lost: %+v", res.Diagnostic)
	}
	if !res.CleanupFailed {
		t.Fatal("RPC-abort cleanup failure not propagated to the result")
	}
}
