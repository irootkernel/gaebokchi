package rpc

import (
	"encoding/json"
	"fmt"
)

// Dialect codecs. Each codec owns its provider's wire envelope and the
// outbound method allowlist enforced at this boundary (ARCH-06).

// Codec renders outgoing messages for one provider dialect.
type Codec interface {
	// Name identifies the dialect in diagnostics.
	Name() string
	// EncodeRequest renders a request with a numeric id.
	EncodeRequest(id int64, method string, params any) ([]byte, error)
	// EncodeNotification renders a notification (no id).
	EncodeNotification(method string, params any) ([]byte, error)
	// EncodeErrorResponse renders a rejection for a server-initiated
	// request, echoing the exact received id.
	EncodeErrorResponse(id json.RawMessage, code int, message string) ([]byte, error)
	// Allowed reports whether the method is on the outbound allowlist.
	Allowed(method string) bool
	// Allowlist lists every permitted outbound method.
	Allowlist() []string
}

// MethodNotFoundCode is the shared JSON-RPC method-not-found code used
// to reject server-initiated requests.
const MethodNotFoundCode = -32601

// wireError is the error member of a response envelope.
type wireError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---- Codex App Server dialect: no jsonrpc marker anywhere ----

var codexAllowlist = []string{
	"initialize",
	"initialized",
	"account/read",
	"account/rateLimits/read",
}

// CodexCodec speaks the Codex App Server envelope: JSON objects with id
// and method, and no "jsonrpc" field.
type CodexCodec struct{}

type codexRequest struct {
	ID     *int64 `json:"id,omitempty"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type codexErrorEnvelope struct {
	ID    json.RawMessage `json:"id"`
	Error wireError       `json:"error"`
}

func (CodexCodec) Name() string { return "codex" }

func (CodexCodec) EncodeRequest(id int64, method string, params any) ([]byte, error) {
	if !methodAllowed(codexAllowlist, method) {
		return nil, fmt.Errorf("rpc: codex codec forbids outbound method %q", method)
	}
	return json.Marshal(codexRequest{ID: &id, Method: method, Params: params})
}

func (CodexCodec) EncodeNotification(method string, params any) ([]byte, error) {
	if !methodAllowed(codexAllowlist, method) {
		return nil, fmt.Errorf("rpc: codex codec forbids outbound method %q", method)
	}
	return json.Marshal(codexRequest{Method: method, Params: params})
}

func (CodexCodec) EncodeErrorResponse(id json.RawMessage, code int, message string) ([]byte, error) {
	return json.Marshal(codexErrorEnvelope{ID: id, Error: wireError{Code: code, Message: message}})
}

func (CodexCodec) Allowed(method string) bool { return methodAllowed(codexAllowlist, method) }

func (CodexCodec) Allowlist() []string { return append([]string(nil), codexAllowlist...) }

// ---- ACP dialect (Grok): jsonrpc:"2.0" on every message ----

var acpAllowlist = []string{
	"initialize",
	"authenticate",
	"_x.ai/billing",
}

// ACPCodec speaks the Agent Client Protocol envelope used by Grok:
// JSON-RPC 2.0 objects carrying "jsonrpc":"2.0". Underscore-prefixed
// extension methods (e.g. _x.ai/billing) are normal ACP extension wire
// forms.
type ACPCodec struct{}

type acpRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type acpErrorEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   wireError       `json:"error"`
}

func (ACPCodec) Name() string { return "acp" }

func (ACPCodec) EncodeRequest(id int64, method string, params any) ([]byte, error) {
	if !methodAllowed(acpAllowlist, method) {
		return nil, fmt.Errorf("rpc: acp codec forbids outbound method %q", method)
	}
	return json.Marshal(acpRequest{JSONRPC: "2.0", ID: &id, Method: method, Params: params})
}

func (ACPCodec) EncodeNotification(method string, params any) ([]byte, error) {
	if !methodAllowed(acpAllowlist, method) {
		return nil, fmt.Errorf("rpc: acp codec forbids outbound method %q", method)
	}
	return json.Marshal(acpRequest{JSONRPC: "2.0", Method: method, Params: params})
}

func (ACPCodec) EncodeErrorResponse(id json.RawMessage, code int, message string) ([]byte, error) {
	return json.Marshal(acpErrorEnvelope{JSONRPC: "2.0", ID: id, Error: wireError{Code: code, Message: message}})
}

func (ACPCodec) Allowed(method string) bool { return methodAllowed(acpAllowlist, method) }

func (ACPCodec) Allowlist() []string { return append([]string(nil), acpAllowlist...) }

func methodAllowed(list []string, method string) bool {
	for _, m := range list {
		if m == method {
			return true
		}
	}
	return false
}
