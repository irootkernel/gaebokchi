// Package codex implements the Codex (ChatGPT) App Server adapter:
// one isolated child per home, metadata-only handshake, account and
// rate-limit reads. It never starts threads or turns.
package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/provider/envfilter"
	"gaebokchi/internal/rpc"
)

// Provider queries Codex targets. Bin is the configured executable;
// resolution happens once per invocation and is reused for every target.
type Provider struct {
	Start process.Factory

	resolveOnce sync.Once
	binName     string
	binPath     string
	binErr      error
}

// New builds a provider using the real process factory.
func New() *Provider { return &Provider{Start: process.Start} }

// resolveBin resolves the executable once per provider instance.
func (p *Provider) resolveBin(bin string) (string, error) {
	p.resolveOnce.Do(func() {
		p.binName = bin
		p.binPath, p.binErr = ResolveBin(bin)
	})
	if p.binName != bin {
		// Different names cannot occur within one invocation, but never
		// reuse a resolution made for another executable.
		p.resolveOnce = sync.Once{}
		return p.resolveBin(bin)
	}
	return p.binPath, p.binErr
}

// Query runs the complete metadata-only lifecycle for one Codex target
// and returns exactly one terminal result. It never renders, exits, or
// touches other targets; a planned cleanup after a valid exchange never
// overwrites the collected outcome.
func (p *Provider) Query(ctx context.Context, tgt domain.Target) domain.Result {
	res := domain.Result{
		TargetID:      tgt.ID,
		Provider:      tgt.Provider,
		Account:       tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias,
		ObservedAt:    time.Now(),
	}

	switch InspectHome(tgt.Home) {
	case HomeMissing:
		res.Status = domain.StatusHomeNotFound
		res.Diagnostic = domain.Diagnostic{Code: "home_not_found", Message: tgt.ID + ": codex home does not exist"}
		return res
	case HomeInvalid:
		res.Status = domain.StatusHomeError
		res.Diagnostic = domain.Diagnostic{Code: "home_error", Message: tgt.ID + ": codex home is not an accessible directory"}
		return res
	}

	bin, err := p.resolveBin(tgt.Bin)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			res.Status = domain.StatusCLINotFound
			res.Diagnostic = domain.Diagnostic{Code: "cli_not_found", Message: tgt.ID + ": codex executable not found"}
		} else {
			res.Status = domain.StatusProviderError
			res.Diagnostic = domain.Diagnostic{Code: "bin_error", Message: tgt.ID + ": codex executable cannot be used"}
		}
		return res
	}

	sess, err := openSession(ctx, bin, tgt.Home, p.Start)
	if err != nil {
		return classifyTransportFailure(res, err)
	}
	defer sess.close()

	info, err := readAccount(ctx, sess)
	if err != nil {
		out := classifyTransportFailure(res, err)
		// Authentication failed before identity was read: keep the
		// home-alias fallback, never another session's email.
		return out
	}
	switch info.AuthStatus {
	case AuthAbsent:
		res.Status = domain.StatusAuthRequired
		res.Diagnostic = domain.Diagnostic{Code: "auth_required", Message: tgt.ID + ": authenticate using the provider CLI outside gaebokchi"}
		return res
	case AuthAPIKey:
		res.Status = domain.StatusNotApplicableAPIKey
		res.Period = domain.PeriodUnknown
		res.Diagnostic = domain.Diagnostic{Code: "api_key_auth", Message: tgt.ID + ": API-key auth has no ChatGPT subscription quota"}
		return res
	case AuthUnknownType:
		res.Status = domain.StatusUnsupported
		res.Diagnostic = domain.Diagnostic{Code: "auth_unsupported", Message: tgt.ID + ": unrecognized codex authentication mode"}
		return res
	}
	if info.Email != "" {
		res.Account = info.Email
		res.AccountSource = domain.AccountSourceProtocolEmail
	}

	quotaRaw, err := readRateLimits(ctx, sess)
	if err != nil {
		return classifyTransportFailure(res, err)
	}
	out := normalizeQuota(quotaRaw)
	res.Status = out.Status
	res.Source = domain.SourceCodexAppServer
	res.QuotaID = out.QuotaID
	res.ResetAt = out.ResetAt
	if out.Percent != nil {
		v := domain.NormalizePercentSignedZero(*out.Percent)
		res.UsedPercent = &v
		res.Period = domain.PeriodWeekly
	} else {
		res.Period = domain.PeriodUnknown
		res.Diagnostic = domain.Diagnostic{Code: string(out.Status), Message: formatQuotaError(out.Status, tgt.ID)}
	}
	return res
}

// readRateLimits performs the allowlisted quota read.
func readRateLimits(ctx context.Context, s *session) (json.RawMessage, error) {
	resp, err := s.conn.Call(ctx, "account/rateLimits/read", nil)
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, &rpcCallError{method: "account/rateLimits/read", remote: resp.Err}
	}
	return resp.Result, nil
}

// classifyTransportFailure maps transport/process failures to terminal
// statuses with fixed safe messages; raw provider payloads never reach
// diagnostics.
func classifyTransportFailure(res domain.Result, err error) domain.Result {
	var rce *rpcCallError
	switch {
	case errors.Is(err, context.Canceled):
		res.Status = domain.StatusCancelled
		res.Diagnostic = domain.Diagnostic{Code: "cancelled", Message: res.TargetID + ": query cancelled"}
	case errors.Is(err, context.DeadlineExceeded):
		res.Status = domain.StatusTimeout
		res.Diagnostic = domain.Diagnostic{Code: "timeout", Message: res.TargetID + ": query exceeded its deadline"}
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		res.Status = domain.StatusCLINotFound
		res.Diagnostic = domain.Diagnostic{Code: "cli_not_found", Message: res.TargetID + ": codex executable not found"}
	case errors.As(err, &rce):
		res.Status = domain.StatusProviderError
		res.Diagnostic = domain.Diagnostic{Code: "remote_error", Message: res.TargetID + ": provider rejected a metadata read"}
	case errors.Is(err, rpc.ErrEOF), errors.Is(err, rpc.ErrMalformedMessage), errors.Is(err, rpc.ErrServerRequest):
		res.Status = domain.StatusProtocolError
		res.Diagnostic = domain.Diagnostic{Code: "protocol_error", Message: res.TargetID + ": app-server protocol stream failed"}
	default:
		res.Status = domain.StatusProviderError
		res.Diagnostic = domain.Diagnostic{Code: "provider_error", Message: res.TargetID + ": codex query failed"}
	}
	return res
}

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
	if err := validateInitializeResult(resp.Result); err != nil {
		s.close()
		return nil, err
	}
	// Wait for the initialize response before sending initialized, per
	// the qualified handshake order. A blocked write is cancellable so a
	// wedged provider cannot defeat the query deadline.
	if err := conn.Notify(ctx, "initialized", struct{}{}); err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

// validateInitializeResult enforces the qualified Codex initialize
// result shape: a non-null JSON object carrying the qualified
// platformOs member. A malformed or shapeless response (error:null,
// result:null, a bare scalar, an empty object) must not complete the
// handshake; unknown additive members stay tolerated.
func validateInitializeResult(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("codex: initialize result missing: %w", rpc.ErrMalformedMessage)
	}
	var w struct {
		PlatformOs *string `json:"platformOs"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("codex: initialize result is not an object: %w", rpc.ErrMalformedMessage)
	}
	if w.PlatformOs == nil {
		return fmt.Errorf("codex: initialize result lacks the qualified platformOs member: %w", rpc.ErrMalformedMessage)
	}
	return nil
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
// An explicit relative path is made absolute against the invocation
// environment before the child's private working directory can change
// its meaning. Go's unsafe-current-directory PATH lookup protection is
// a different case: an implicit lookup that found the current directory
// fails with exec.ErrDot and stays an error.
func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("codex: empty executable")
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(resolved) {
		abs, err := filepath.Abs(resolved)
		if err != nil {
			return "", err
		}
		resolved = abs
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
