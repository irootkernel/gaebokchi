package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"strings"
	"time"

	"gaebokchi/internal/process"
	"gaebokchi/internal/rpc"
	"gaebokchi/internal/testutil"
)

// newTestConn wires a fake child into an rpc connection with the Codex
// codec and the child teardown hook.
func newTestConn(child *process.Child) *rpc.Conn {
	return rpc.NewConn(child.Stdin(), child.Stdout(), rpc.CodexCodec{}, func() { _ = child.Close() })
}

func fixture(t testT, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Fixtures are stored pretty-printed; the wire needs one compact
	// line per message.
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		t.Fatalf("compact fixture %s: %v", name, err)
	}
	return compact.String()
}

// handshake performs the qualified initialize -> initialized sequence
// against a fake that replies to initialize, leaving the connection
// ready for account/read.
func handshake(t testT, conn *rpc.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	initParams := map[string]any{
		"clientInfo": map[string]any{"name": "gaebokchi", "title": "Gaebokchi", "version": clientVersion},
	}
	resp, err := conn.Call(ctx, "initialize", initParams)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Err != nil {
		t.Fatalf("initialize rejected: %v", resp.Err)
	}
	if err := conn.Notify(ctx, "initialized", struct{}{}); err != nil {
		t.Fatalf("initialized: %v", err)
	}
}

var _ = ginkgo.It("TestHandshakeOrderAndOutgoingSurface", func() {
	t := testT{ginkgo.GinkgoT()}
	steps := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Capture: 1}, // initialized notification
		{Wait: 1},    // account/read request
		{Reply: []testutil.ReplySpec{{Slot: 1, RawResult: `{"account":null}`}}},
	}
	child := testutil.StartFake(t, steps)
	conn := newTestConn(child)
	handshake(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := readAccount(ctx, &session{child: child, conn: conn})
	if err != nil {
		t.Fatalf("readAccount: %v", err)
	}
	if info.AuthStatus != AuthAbsent {
		t.Fatalf("auth status = %v, want AuthAbsent for null account", info.AuthStatus)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}

	kept := string(child.StderrKept())
	if !strings.Contains(kept, `"method":"initialize"`) {
		t.Fatalf("initialize request not observed in transcript: %q", kept)
	}
	if !strings.Contains(kept, `"method":"initialized"`) {
		t.Fatalf("initialized notification not observed: %q", kept)
	}
	if !strings.Contains(kept, `"method":"account/read"`) {
		t.Fatalf("account/read not observed: %q", kept)
	}
	if !strings.Contains(kept, `"refreshToken":false`) {
		t.Fatalf("account/read must send refreshToken:false: %q", kept)
	}
	if !strings.Contains(kept, `"title":"Gaebokchi"`) {
		t.Fatalf("initialize clientInfo missing title: %q", kept)
	}
	for _, forbidden := range []string{"session/new", "session/load", "thread/", "turn/", "prompt", "resumed"} {
		if strings.Contains(kept, forbidden) {
			t.Fatalf("forbidden outbound method %q observed: %q", forbidden, kept)
		}
	}
	if strings.Contains(kept, `"jsonrpc"`) {
		t.Fatalf("codex wire must not carry jsonrpc: %q", kept)
	}
})

var _ = ginkgo.It("TestAccountMappingChatGPTFromFixture", func() {
	t := testT{ginkgo.GinkgoT()}
	acc := fixture(t, "account_chatgpt_home1.json")
	steps := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: acc + "\n"},
	}
	child := testutil.StartFake(t, steps)
	conn := newTestConn(child)
	handshake(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := readAccount(ctx, &session{child: child, conn: conn})
	if err != nil {
		t.Fatalf("readAccount: %v", err)
	}
	if info.AuthStatus != AuthChatGPT || info.Email != "account1@example.com" {
		t.Fatalf("info = %+v", info)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
})

var _ = ginkgo.Describe("TestAccountMappingVariants", func() {
	cases := []struct {
		name       string
		account    string
		wantStatus AuthStatus
	}{
		{"api key", `{"id":2,"result":{"account":{"type":"apiKey"}}}`, AuthAPIKey},
		{"unknown type", `{"id":2,"result":{"account":{"type":"device"}}}`, AuthUnknownType},
		{"null account", `{"id":2,"result":{"account":null}}`, AuthAbsent},
		{"null result", `{"id":2,"result":null}`, AuthAbsent},
		{"chatgpt without email", `{"id":2,"result":{"account":{"type":"chatgpt"}}}`, AuthChatGPT},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			steps := []testutil.Step{
				{Wait: 1},
				{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
				{Wait: 1},
				{Emit: tc.account + "\n"},
			}
			child := testutil.StartFake(t, steps)
			conn := newTestConn(child)
			handshake(t, conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			info, err := readAccount(ctx, &session{child: child, conn: conn})
			if err != nil {
				t.Fatalf("readAccount: %v", err)
			}
			if info.AuthStatus != tc.wantStatus {
				t.Fatalf("status = %v, want %v", info.AuthStatus, tc.wantStatus)
			}
			if info.Email != "" {
				t.Fatalf("email should be unset, got %q", info.Email)
			}
			if err := child.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
})

var _ = ginkgo.Describe("TestAccountEmailNormalizationPolicy", func() {
	cases := []struct {
		name  string
		email string
		want  string
	}{
		// Trailing/leading control junk normalizes away; the label is
		// usable through the display policy.
		{"trailing control characters", "user@example.com\n\t\x1b", "user@example.com"},
		{"leading and surrounding spaces", "  spaced@example.com  ", "spaced@example.com"},
		{"NBSP becomes a space", "a\u00a0b@example.com", "a b@example.com"},
		// A value with nothing displayable left is unusable and falls
		// back to the home alias (empty Email here).
		{"control-only email", "\t\n\x1b", ""},
		{"whitespace-only email", "   ", ""},
		{"ideographic-space-only email", "\u3000", ""},
		{"mixed unicode whitespace-only email", " \u205f\u1680\u3000", ""},
		{"unicode spaces around email", "\u3000user@example.com\u205f", "user@example.com"},
		{"format-character-only email", "\u200e\u202e", ""},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			payload, _ := json.Marshal(map[string]any{
				"id": 2,
				"result": map[string]any{
					"account": map[string]any{"type": "chatgpt", "email": tc.email},
				},
			})
			steps := []testutil.Step{
				{Wait: 1},
				{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
				{Wait: 1},
				{Emit: string(payload) + "\n"},
			}
			child := testutil.StartFake(t, steps)
			conn := newTestConn(child)
			handshake(t, conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			info, err := readAccount(ctx, &session{child: child, conn: conn})
			if err != nil {
				t.Fatal(err)
			}
			if info.Email != tc.want {
				t.Fatalf("email = %q, want %q", info.Email, tc.want)
			}
			if info.AuthStatus != AuthChatGPT {
				t.Fatalf("auth status = %v", info.AuthStatus)
			}
			if err := child.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
})

// Two sessions over two homes keep independent identity: each
// readAccount answer comes from its own fake child and connection.
var _ = ginkgo.It("TestTwoHomesIsolatedIdentity", func() {
	t := testT{ginkgo.GinkgoT()}
	mk := func(email string) []testutil.Step {
		acc := fmt.Sprintf(`{"id":2,"result":{"account":{"type":"chatgpt","email":%q}}}`, email)
		return []testutil.Step{
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"ok": true}}}},
			{Wait: 1},
			{Emit: acc + "\n"},
		}
	}
	childA := testutil.StartFake(t, mk("a@example.com"))
	childB := testutil.StartFake(t, mk("b@example.com"))
	connA := newTestConn(childA)
	connB := newTestConn(childB)
	handshake(t, connA)
	handshake(t, connB)
	t.Cleanup(func() {
		errA := childA.Close()
		errB := childB.Close()
		if errA != nil || errB != nil {
			t.Fatalf("close child A: %v; close child B: %v", errA, errB)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Interleave: read B's answer first; A's must not be affected.
	infoB, err := readAccount(ctx, &session{child: childB, conn: connB})
	if err != nil {
		t.Fatalf("B: %v", err)
	}
	infoA, err := readAccount(ctx, &session{child: childA, conn: connA})
	if err != nil {
		t.Fatalf("A: %v", err)
	}
	if infoA.Email != "a@example.com" || infoB.Email != "b@example.com" {
		t.Fatalf("identity mixed across homes: %q / %q", infoA.Email, infoB.Email)
	}
})

var _ = ginkgo.It("TestRemoteAuthErrorIsTypedNotGuessed", func() {
	t := testT{ginkgo.GinkgoT()}
	steps := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 1, ErrCode: intPtr(-32000), ErrMsg: "please sign in"}}},
	}
	child := testutil.StartFake(t, steps)
	conn := newTestConn(child)
	handshake(t, conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := readAccount(ctx, &session{child: child, conn: conn})
	var rce *rpcCallError
	if err == nil {
		t.Fatal("expected typed error")
	}
	if ok := asRPCCallError(err, &rce); !ok || rce.method != "account/read" {
		t.Fatalf("err = %v, want rpcCallError for account/read", err)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
})

var _ = ginkgo.It("TestInspectHome", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	if InspectHome(dir) != HomeOK {
		t.Fatal("existing dir not OK")
	}
	if InspectHome(dir+"/missing") != HomeMissing {
		t.Fatal("missing path should be HomeMissing")
	}
	file := dir + "/plainfile"
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if InspectHome(file) != HomeInvalid {
		t.Fatal("non-directory should be HomeInvalid")
	}
})

var _ = ginkgo.It("accepts searchable homes and rejects homes without search permission", func() {
	t := testT{ginkgo.GinkgoT()}
	path := t.TempDir() + "/restricted"
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(path, 0o700) }()
	if err := os.Chmod(path, 0o300); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + "/."); err != nil {
		t.Skip("directory search permission is unavailable")
	}
	if dir, err := os.Open(path); err == nil {
		_ = dir.Close()
		t.Skip("directory read permission checks are bypassed by this user")
	}
	if InspectHome(path) != HomeOK {
		t.Fatal("searchable directory without read permission should be HomeOK")
	}
	for _, mode := range []os.FileMode{0o000, 0o400} {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		if mode == 0o000 {
			if dir, err := os.Open(path); err == nil {
				_ = dir.Close()
				t.Skip("directory permission checks are bypassed by this user")
			}
		} else if _, err := os.Stat(path + "/."); err == nil {
			t.Skip("directory search permission checks are bypassed by this user")
		}
		if InspectHome(path) != HomeInvalid {
			t.Fatalf("mode %04o should be HomeInvalid", mode)
		}
	}
})

func intPtr(i int) *int { return &i }

func asRPCCallError(err error, target **rpcCallError) bool {
	rce, ok := err.(*rpcCallError)
	if ok {
		*target = rce
	}
	return ok
}
