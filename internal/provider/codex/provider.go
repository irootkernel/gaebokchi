// Package codex implements the Codex (ChatGPT) App Server adapter:
// one isolated child per home, metadata-only handshake, account and
// rate-limit reads. It never starts threads or turns.
package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"gaebokchi/internal/process"
	"gaebokchi/internal/provider/envfilter"
	"gaebokchi/internal/rpc"
)

// Provider queries Codex targets. Bin is the configured executable;
// resolution happens once per invocation and is reused for every target.
type Provider struct {
	Start process.Factory
}

// New builds a provider using the real process factory.
func New() *Provider { return &Provider{Start: process.Start} }

// clientVersion identifies gaebokchi to the App Server.
const clientVersion = "0.1.0"

// session is one isolated Codex App Server connection bound to one home.
type session struct {
	child *process.Child
	conn  *rpc.Conn
}

func (s *session) close() {
	s.conn.Close()
	_ = s.child.Close()
}

// startChild launches `codex app-server` for one home with the
// qualified environment policy.
func startChild(bin, home string, start process.Factory) (*process.Child, error) {
	env := envfilter.Apply(os.Environ(), envfilter.Rule{
		Drop: envfilter.Codex().Drop,
		Set:  map[string]string{"CODEX_HOME": home},
	})
	return start(process.Config{
		Bin:  bin,
		Args: []string{"app-server"},
		Env:  env,
	})
}

// openSession starts the child and completes the metadata-only
// initialize/initialized handshake.
func openSession(ctx context.Context, bin, home string, start process.Factory) (*session, error) {
	child, err := startChild(bin, home, start)
	if err != nil {
		return nil, err
	}
	conn := rpc.NewConn(child.Stdin(), child.Stdout(), rpc.CodexCodec{}, func() { _ = child.Close() })
	s := &session{child: child, conn: conn}

	initParams := map[string]any{
		"clientInfo": map[string]any{
			"name":    "gaebokchi",
			"title":   "Gaebokchi",
			"version": clientVersion,
		},
	}
	resp, err := conn.Call(ctx, "initialize", initParams)
	if err != nil {
		s.close()
		return nil, err
	}
	if resp.Err != nil {
		s.close()
		return nil, &rpcCallError{method: "initialize", remote: resp.Err}
	}
	// Wait for the initialize response before sending initialized, per
	// the qualified handshake order.
	if err := conn.Notify("initialized", struct{}{}); err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

// rpcCallError marks a well-formed remote rejection.
type rpcCallError struct {
	method string
	remote *rpc.RPCError
}

func (e *rpcCallError) Error() string {
	return fmt.Sprintf("codex: %s rejected with code %d", e.method, e.remote.Code)
}

// AccountInfo is the normalized identity/auth outcome of account/read.
type AccountInfo struct {
	// Email is the protocol email when the session supplied one.
	Email string
	// AuthStatus classifies the account for downstream mapping.
	AuthStatus AuthStatus
}

// AuthStatus enumerates the qualified account authentication outcomes.
type AuthStatus int

const (
	AuthChatGPT AuthStatus = iota
	AuthAbsent
	AuthAPIKey
	AuthUnknownType
)

// accountWire preserves presence of the account/read result members.
type accountWire struct {
	Account *struct {
		Type    string  `json:"type"`
		Email   *string `json:"email"`
		Plan    *string `json:"planType"`
	} `json:"account"`
}

// readAccount performs account/read with refreshToken:false and maps
// the qualified shapes. Unstructured guesses are never made: an
// unusable email falls back to the caller's home alias at result
// construction.
func readAccount(ctx context.Context, s *session) (*AccountInfo, error) {
	resp, err := s.conn.Call(ctx, "account/read", map[string]any{"refreshToken": false})
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, &rpcCallError{method: "account/read", remote: resp.Err}
	}
	if string(resp.Result) == "null" || len(resp.Result) == 0 {
		return &AccountInfo{AuthStatus: AuthAbsent}, nil
	}
	var w accountWire
	if err := json.Unmarshal(resp.Result, &w); err != nil {
		return nil, fmt.Errorf("codex: account/read result is malformed: %v", err)
	}
	if w.Account == nil {
		return &AccountInfo{AuthStatus: AuthAbsent}, nil
	}
	info := &AccountInfo{}
	switch w.Account.Type {
	case "chatgpt":
		info.AuthStatus = AuthChatGPT
	case "apiKey":
		info.AuthStatus = AuthAPIKey
	default:
		info.AuthStatus = AuthUnknownType
	}
	if w.Account.Email != nil && isUsableEmail(*w.Account.Email) {
		info.Email = *w.Account.Email
	}
	return info, nil
}

// isUsableEmail rejects blank or control-character-laden values that
// cannot serve as a display label.
func isUsableEmail(email string) bool {
	if email == "" {
		return false
	}
	for _, r := range email {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// ResolveBin resolves the configured executable once per invocation.
// Relative explicit paths are honored only when they are not the
// unsafe-current-directory lookup Go protects against.
func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("codex: empty executable")
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

// HomeStatus classifies a Codex home directory without creating it.
type HomeStatus int

const (
	HomeOK HomeStatus = iota
	HomeMissing
	HomeInvalid
)

// InspectHome reports whether the home exists and is a directory.
func InspectHome(home string) HomeStatus {
	fi, err := os.Stat(home)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HomeMissing
		}
		return HomeInvalid // inaccessible: permission or similar
	}
	if !fi.IsDir() {
		return HomeInvalid
	}
	return HomeOK
}
