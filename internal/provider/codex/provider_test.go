package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/process"
	"gaebokchi/internal/rpc"
	"gaebokchi/internal/testutil"
)

func TestFakeHelperProcess(t *testing.T) {
	if os.Getenv("GAEBOKCHI_FAKE") != "1" {
		return
	}
	if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// newTestConn wires a fake child into an rpc connection with the Codex
// codec and the child teardown hook.
func newTestConn(child *process.Child) *rpc.Conn {
	return rpc.NewConn(child.Stdin(), child.Stdout(), rpc.CodexCodec{}, func() { _ = child.Close() })
}

func fixture(t *testing.T, name string) string {
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
func handshake(t *testing.T, conn *rpc.Conn) {
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

func TestHandshakeOrderAndOutgoingSurface(t *testing.T) {
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
	child.Close()

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
}

func TestAccountMappingChatGPTFromFixture(t *testing.T) {
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
	child.Close()
}

func TestAccountMappingVariants(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
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
			child.Close()
		})
	}
}

func TestAccountEmailNormalizationPolicy(t *testing.T) {
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
		{"format-character-only email", "\u200e\u202e", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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
			child.Close()
		})
	}
}

// Two sessions over two homes keep independent identity: each
// readAccount answer comes from its own fake child and connection.
func TestTwoHomesIsolatedIdentity(t *testing.T) {
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
	defer childA.Close()
	defer childB.Close()

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
}

func TestRemoteAuthErrorIsTypedNotGuessed(t *testing.T) {
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
	child.Close()
}

func TestInspectHome(t *testing.T) {
	dir := t.TempDir()
	if InspectHome(dir) != HomeOK {
		t.Fatal("existing dir not OK")
	}
	if InspectHome(dir + "/missing") != HomeMissing {
		t.Fatal("missing path should be HomeMissing")
	}
	file := dir + "/plainfile"
	os.WriteFile(file, []byte("x"), 0o644)
	if InspectHome(file) != HomeInvalid {
		t.Fatal("non-directory should be HomeInvalid")
	}
}

func intPtr(i int) *int { return &i }

func asRPCCallError(err error, target **rpcCallError) bool {
	rce, ok := err.(*rpcCallError)
	if ok {
		*target = rce
	}
	return ok
}
