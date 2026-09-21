package rpc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodexCodecExactBytes(t *testing.T) {
	c := CodexCodec{}
	got, err := c.EncodeRequest(1, "initialize", map[string]any{"name": "gaebokchi"})
	if err != nil {
		t.Fatal(err)
	}
	// Struct field order is contractual for transcript assertions.
	want := `{"id":1,"method":"initialize","params":{"name":"gaebokchi"}}`
	if string(got) != want {
		t.Fatalf("request = %s, want %s", got, want)
	}
	got, err = c.EncodeRequest(3, "account/rateLimits/read", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"id":3,"method":"account/rateLimits/read"}` {
		t.Fatalf("params-less request = %s", got)
	}
	got, err = c.EncodeNotification("initialized", struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"method":"initialized","params":{}}` {
		t.Fatalf("notification = %s", got)
	}
	if strings.Contains(string(got), "jsonrpc") {
		t.Fatal("codex wire must not carry jsonrpc")
	}
	resp, err := c.EncodeErrorResponse(json.RawMessage(`900`), MethodNotFoundCode, "nope")
	if err != nil {
		t.Fatal(err)
	}
	if string(resp) != `{"id":900,"error":{"code":-32601,"message":"nope"}}` {
		t.Fatalf("error response = %s", resp)
	}
}

func TestACPCodecExactBytes(t *testing.T) {
	c := ACPCodec{}
	got, err := c.EncodeRequest(1, "initialize", map[string]any{"protocolVersion": 1})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}`
	if string(got) != want {
		t.Fatalf("request = %s, want %s", got, want)
	}
	got, err = c.EncodeRequest(3, "_x.ai/billing", struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"jsonrpc":"2.0","id":3,"method":"_x.ai/billing","params":{}}` {
		t.Fatalf("billing request = %s", got)
	}
	resp, err := c.EncodeErrorResponse(json.RawMessage(`"req-7"`), MethodNotFoundCode, "nope")
	if err != nil {
		t.Fatal(err)
	}
	if string(resp) != `{"jsonrpc":"2.0","id":"req-7","error":{"code":-32601,"message":"nope"}}` {
		t.Fatalf("error response = %s", resp)
	}
}

func TestCodecAllowlists(t *testing.T) {
	for _, c := range []Codec{CodexCodec{}, ACPCodec{}} {
		for _, m := range c.Allowlist() {
			if !c.Allowed(m) {
				t.Fatalf("%s allowlist lists %q but Allowed denies it", c.Name(), m)
			}
			if _, err := c.EncodeRequest(1, m, nil); err != nil {
				t.Fatalf("%s rejected allowlisted method %q: %v", c.Name(), m, err)
			}
		}
	}
	codexForbidden := []string{"session/new", "turn/start", "account/usage/read", "thread/start"}
	for _, m := range codexForbidden {
		if _, err := (CodexCodec{}).EncodeRequest(1, m, nil); err == nil {
			t.Fatalf("codex codec allowed forbidden method %q", m)
		}
		if _, err := (CodexCodec{}).EncodeNotification(m, nil); err == nil {
			t.Fatalf("codex codec allowed forbidden notification %q", m)
		}
	}
	acpForbidden := []string{"session/new", "session/prompt", "x.ai/billing", "authenticate/browser", "initialized"}
	for _, m := range acpForbidden {
		if _, err := (ACPCodec{}).EncodeRequest(1, m, nil); err == nil {
			t.Fatalf("acp codec allowed forbidden method %q", m)
		}
		if _, err := (ACPCodec{}).EncodeNotification(m, nil); err == nil {
			t.Fatalf("acp codec allowed forbidden notification %q", m)
		}
	}
}
