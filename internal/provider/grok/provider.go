// Package grok implements the xAI Grok ACP adapter: the qualified
// headless stdio invocation, cached-token authentication, and the
// billing extension read. It never creates sessions or prompts.
package grok

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

// Wire constants frozen from the TASK-001 installed-binary probes.
const (
	// acpProtocolVersion is the verified protocol version.
	acpProtocolVersion = 1
	// billingMethod is the underscore-prefixed wire name of the
	// internal x.ai/billing extension.
	billingMethod = "_x.ai/billing"
	// cachedTokenMethod is the only authentication mode gaebokchi uses.
	cachedTokenMethod = "cached_token"
	// knownInteractiveMethod is the browser-login method id observed
	// during qualification; it is never a fallback.
	knownInteractiveMethod = "grok.com"
	// knownAPIKeyMethods are advertised method ids that establish
	// API-key-only non-applicability.
	knownAPIKeyMethods = "api_key"
	clientVersion      = "0.1.0"
)

// Provider queries the single Grok target.
type Provider struct {
	Start process.Factory

	resolveOnce sync.Once
	binName     string
	binPath     string
	binErr      error
}

// New builds a provider using the real process factory.
func New() *Provider { return &Provider{Start: process.Start} }

func (p *Provider) resolveBin(bin string) (string, error) {
	p.resolveOnce.Do(func() {
		p.binName = bin
		p.binPath, p.binErr = ResolveBin(bin)
	})
	if p.binName != bin {
		p.resolveOnce = sync.Once{}
		return p.resolveBin(bin)
	}
	return p.binPath, p.binErr
}

// ResolveBin resolves the configured executable once per invocation.
// An explicit relative path is made absolute against the invocation
// environment before the child's private working directory can change
// its meaning; ErrDot protection for implicit current-directory PATH
// lookups is respected and shell aliases are not executable.
func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("grok: empty executable")
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

// session is one Grok ACP connection.
type session struct {
	child *process.Child
	conn  *rpc.Conn
}

func (s *session) close() {
	s.conn.Close()
	_ = s.child.Close()
}

// childEnv composes the filtered environment for Grok children from
// the parent environment.
func childEnv() []string {
	return envfilter.Apply(os.Environ(), envfilter.Grok())
}

// startChild launches the qualified headless invocation.
func startChild(bin string, start process.Factory) (*process.Child, error) {
	return start(process.Config{
		Bin:  bin,
		Args: []string{"--no-auto-update", "agent", "stdio"},
		Env:  childEnv(),
	})
}

// rpcCallError marks a well-formed remote rejection.
type rpcCallError struct {
	method string
	remote *rpc.RPCError
}

func (e *rpcCallError) Error() string {
	return fmt.Sprintf("grok: %s rejected with code %d", e.method, e.remote.Code)
}

// authPlan is the negotiated authentication outcome.
type authPlan int

const (
	planCachedToken authPlan = iota
	planAPIKeyOnly
	planInteractiveOnly
	planUnknownMethods
	planVersionMismatch
)

// initializeParams is the qualified ACP initialize envelope. Host
// filesystem and terminal capabilities are explicitly disabled.
type initializeParams struct {
	ProtocolVersion    int              `json:"protocolVersion"`
	ClientInfo         clientInfo       `json:"clientInfo"`
	ClientCapabilities clientCapability `json:"clientCapabilities"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type clientCapability struct {
	FS *fsCapability `json:"fs"`
	Terminal bool    `json:"terminal"`
}

type fsCapability struct {
	ReadTextFile  bool `json:"readTextFile"`
	WriteTextFile bool `json:"writeTextFile"`
}

type authenticateParams struct {
	MethodID string       `json:"methodId"`
	Meta     headlessMeta `json:"_meta"`
}

type headlessMeta struct {
	Headless bool `json:"headless"`
}

// openSession starts the child, negotiates the verified protocol
// version, and classifies the advertised authentication methods. No
// `initialized` notification exists in this dialect.
func openSession(ctx context.Context, bin string, start process.Factory) (*session, authPlan, error) {
	child, err := startChild(bin, start)
	if err != nil {
		return nil, 0, err
	}
	conn := rpc.NewConn(child.Stdin(), child.Stdout(), rpc.ACPCodec{}, func() { _ = child.Close() })
	s := &session{child: child, conn: conn}

	initParams := initializeParams{
		ProtocolVersion: acpProtocolVersion,
		ClientInfo:      clientInfo{Name: "gaebokchi", Version: clientVersion},
		ClientCapabilities: clientCapability{
			FS:       &fsCapability{ReadTextFile: false, WriteTextFile: false},
			Terminal: false,
		},
	}

	resp, err := conn.Call(ctx, "initialize", initParams)
	if err != nil {
		s.close()
		return nil, 0, err
	}
	if resp.Err != nil {
		s.close()
		return nil, 0, &rpcCallError{method: "initialize", remote: resp.Err}
	}
	plan, err := classifyInitialize(resp.Result)
	if err != nil || plan != planCachedToken {
		s.close()
		return nil, plan, err
	}
	return s, plan, nil
}

// initResultWire preserves presence of the negotiated members.
type initResultWire struct {
	ProtocolVersion *json.RawMessage `json:"protocolVersion"`
	AuthMethods     []struct {
		ID string `json:"id"`
	} `json:"authMethods"`
}

func classifyInitialize(raw json.RawMessage) (authPlan, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, errors.New("grok: initialize result missing")
	}
	var w initResultWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return 0, fmt.Errorf("grok: initialize result malformed: %v", err)
	}
	if w.ProtocolVersion == nil || string(*w.ProtocolVersion) == "null" {
		return 0, errors.New("grok: initialize result missing protocolVersion")
	}
	var version int
	if err := json.Unmarshal(*w.ProtocolVersion, &version); err != nil {
		return 0, errors.New("grok: protocolVersion is not numeric")
	}
	if version != acpProtocolVersion {
		return planVersionMismatch, nil
	}

	hasCached, hasAPIKey, hasInteractive, hasOther := false, false, false, false
	for _, m := range w.AuthMethods {
		switch m.ID {
		case cachedTokenMethod:
			hasCached = true
		case knownAPIKeyMethods:
			hasAPIKey = true
		case knownInteractiveMethod:
			hasInteractive = true
		default:
			hasOther = true
		}
	}
	switch {
	case hasCached:
		return planCachedToken, nil
	case hasAPIKey:
		return planAPIKeyOnly, nil
	case hasInteractive:
		return planInteractiveOnly, nil
	case hasOther:
		return planUnknownMethods, nil
	default:
		return planUnknownMethods, nil
	}
}

// authenticateCached sends the qualified headless cached-token request.
func authenticateCached(ctx context.Context, s *session) error {
	params := authenticateParams{MethodID: cachedTokenMethod}
	params.Meta.Headless = true
	resp, err := s.conn.Call(ctx, "authenticate", params)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return &rpcCallError{method: "authenticate", remote: resp.Err}
	}
	return nil
}

// readBilling calls the single qualified billing extension method.
func readBilling(ctx context.Context, s *session) (json.RawMessage, error) {
	resp, err := s.conn.Call(ctx, billingMethod, struct{}{})
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, &rpcCallError{method: billingMethod, remote: resp.Err}
	}
	return resp.Result, nil
}

// Query runs the complete metadata-only lifecycle for the Grok target
// and returns exactly one terminal result. The ACCOUNT column is the
// configured alias on every outcome; no identity endpoint exists.
func (p *Provider) Query(ctx context.Context, tgt domain.Target) domain.Result {
	res := domain.Result{
		TargetID:      tgt.ID,
		Provider:      tgt.Provider,
		Account:       tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias,
		ObservedAt:    time.Now(),
	}

	bin, err := p.resolveBin(tgt.Bin)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			res.Status = domain.StatusCLINotFound
			res.Diagnostic = domain.Diagnostic{Code: "cli_not_found", Message: tgt.ID + ": grok executable not found"}
		} else {
			res.Status = domain.StatusProviderError
			res.Diagnostic = domain.Diagnostic{Code: "bin_error", Message: tgt.ID + ": grok executable cannot be used"}
		}
		return res
	}

	s, plan, err := openSession(ctx, bin, p.Start)
	if err != nil {
		return classifyTransportFailure(res, err)
	}
	if s == nil {
		// Negotiation established a terminal plan without a session.
		switch plan {
		case planAPIKeyOnly:
			res.Status = domain.StatusNotApplicableAPIKey
			res.Diagnostic = domain.Diagnostic{Code: "api_key_auth", Message: tgt.ID + ": API-key auth has no subscription billing quota"}
		case planInteractiveOnly:
			res.Status = domain.StatusAuthRequired
			res.Diagnostic = domain.Diagnostic{Code: "auth_required", Message: tgt.ID + ": authenticate using the provider CLI outside gaebokchi"}
		case planUnknownMethods:
			res.Status = domain.StatusUnsupported
			res.Diagnostic = domain.Diagnostic{Code: "auth_unsupported", Message: tgt.ID + ": unrecognized grok authentication advertisement"}
		case planVersionMismatch:
			res.Status = domain.StatusUnsupported
			res.Diagnostic = domain.Diagnostic{Code: "version_unsupported", Message: tgt.ID + ": installed grok speaks an unsupported ACP protocol version"}
		}
		return res
	}
	defer s.close()

	// A live session exists only for the cached-token plan.
	if err := authenticateCached(ctx, s); err != nil {
		var rce *rpcCallError
		if errors.As(err, &rce) {
			res.Status = domain.StatusAuthRequired
			res.Diagnostic = domain.Diagnostic{Code: "auth_required", Message: tgt.ID + ": cached token rejected; re-authenticate with the provider CLI"}
			return res
		}
		return classifyTransportFailure(res, err)
	}

	raw, err := readBilling(ctx, s)
	if err != nil {
		var rce *rpcCallError
		if errors.As(err, &rce) {
			if rce.remote.Code == rpc.MethodNotFoundCode {
				res.Status = domain.StatusUnsupported
				res.Diagnostic = domain.Diagnostic{Code: "method_not_found", Message: formatBillingError(domain.StatusUnsupported, tgt.ID)}
			} else {
				res.Status = domain.StatusProviderError
				res.Diagnostic = domain.Diagnostic{Code: "remote_error", Message: tgt.ID + ": provider rejected the billing read"}
			}
			return res
		}
		return classifyTransportFailure(res, err)
	}

	out := normalizeBilling(raw)
	res.Status = out.Status
	res.Source = domain.SourceGrokACPBilling
	res.Period = out.Period
	res.ResetAt = out.ResetAt
	if out.Percent != nil {
		v := domain.NormalizePercentSignedZero(*out.Percent)
		res.UsedPercent = &v
	} else {
		res.Diagnostic = domain.Diagnostic{Code: string(out.Status), Message: formatBillingError(out.Status, tgt.ID)}
	}
	return res
}

// classifyTransportFailure maps transport failures to terminal statuses
// with fixed safe messages.
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
		res.Diagnostic = domain.Diagnostic{Code: "cli_not_found", Message: res.TargetID + ": grok executable not found"}
	case errors.As(err, &rce):
		res.Status = domain.StatusProviderError
		res.Diagnostic = domain.Diagnostic{Code: "remote_error", Message: res.TargetID + ": provider rejected a metadata read"}
	case errors.Is(err, rpc.ErrEOF), errors.Is(err, rpc.ErrMalformedMessage), errors.Is(err, rpc.ErrServerRequest):
		res.Status = domain.StatusProtocolError
		res.Diagnostic = domain.Diagnostic{Code: "protocol_error", Message: res.TargetID + ": ACP protocol stream failed"}
	default:
		res.Status = domain.StatusProviderError
		res.Diagnostic = domain.Diagnostic{Code: "provider_error", Message: res.TargetID + ": grok query failed"}
	}
	return res
}
