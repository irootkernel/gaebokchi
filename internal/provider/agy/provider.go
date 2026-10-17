// Package agy reads weekly model-pool allowances (gemini-weekly and 3p-weekly)
// for the user's single default Google Antigravity CLI login through agy's
// built-in /usage command in print mode. It never sends a model prompt.
package agy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/provider/envfilter"
)

const (
	outputLimit     = 256 << 10
	QuotaGemini     = "gemini-weekly"
	QuotaClaudeGPT  = "3p-weekly"
	AccountGemini   = "gemini"
	AccountClaudeGP = "claude/gpt"
)

var (
	versionToken      = regexp.MustCompile(`\b(?:v)?([0-9]+)\.([0-9]+)\.([0-9]+)\b`)
	errOutputTooLarge = errors.New("agy output exceeds limit")
	errAuthRequired   = errors.New("agy authentication required")
)

// Provider queries the default agy login once per binary and projects the two
// weekly model pools onto agy:0 (gemini) and agy:1 (claude/gpt).
type Provider struct {
	Start process.Factory

	mu      sync.Mutex
	queries map[string]*sharedQuery
}

// New builds an agy provider using the standard process factory.
func New() *Provider {
	return &Provider{
		Start:   process.Start,
		queries: make(map[string]*sharedQuery),
	}
}

// ResolveBin resolves the configured agy executable.
func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("agy: empty executable")
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(resolved) {
		return filepath.Abs(resolved)
	}
	return resolved, nil
}

func childEnv() []string {
	return envfilter.Apply(os.Environ(), envfilter.Agy())
}

type commandResult struct {
	output        []byte
	stderr        []byte
	exitCode      int
	cleanupFailed bool
	err           error
}

func hasAuthRequired(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	lower := strings.ToLower(string(b))
	return strings.Contains(lower, "authentication required")
}

func (p *Provider) command(ctx context.Context, bin string, env []string, args ...string) (out commandResult) {
	if err := ctx.Err(); err != nil {
		out.err = err
		return
	}
	start := p.Start
	if start == nil {
		start = process.Start
	}
	child, err := start(process.Config{Bin: bin, Args: args, Env: env})
	if err != nil {
		out.err = err
		return
	}
	defer func() {
		out.cleanupFailed = child.Close() != nil
		out.stderr = child.StderrKept()
		if out.err == nil && hasAuthRequired(out.stderr) {
			out.err = errAuthRequired
		}
	}()
	child.CloseStdin()

	type readResult struct {
		data []byte
		err  error
	}
	readDone := make(chan readResult, 1)
	go func() {
		data, readErr := io.ReadAll(io.LimitReader(child.Stdout(), outputLimit+1))
		if len(data) > outputLimit {
			readErr = errOutputTooLarge
		}
		readDone <- readResult{data: data, err: readErr}
	}()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	childDone := child.Done()
	var stdoutDone, procDone bool
	for !stdoutDone || !procDone {
		if hasAuthRequired(child.StderrKept()) {
			out.err = errAuthRequired
			return
		}
		select {
		case <-ctx.Done():
			out.err = ctx.Err()
			return
		case <-ticker.C:
			if hasAuthRequired(child.StderrKept()) {
				out.err = errAuthRequired
				return
			}
		case read := <-readDone:
			stdoutDone = true
			readDone = nil
			if read.err != nil {
				out.err = read.err
				return
			}
			out.output = read.data
		case <-childDone:
			procDone = true
			childDone = nil
			out.exitCode = child.ExitCode()
		}
	}
	return
}

type poolOutcome struct {
	status  domain.Status
	code    string
	message string
	percent *float64
	resetAt *time.Time
}

type sharedQuery struct {
	once          sync.Once
	cleanupFailed bool
	topStatus     domain.Status
	topCode       string
	topMessage    string
	pools         map[string]poolOutcome
}

func (p *Provider) getQuery(bin string) *sharedQuery {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.queries == nil {
		p.queries = make(map[string]*sharedQuery)
	}
	q, ok := p.queries[bin]
	if !ok {
		q = &sharedQuery{}
		p.queries[bin] = q
	}
	return q
}

func (sq *sharedQuery) execute(ctx context.Context, p *Provider, rawBin string) {
	bin, err := ResolveBin(rawBin)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			sq.topStatus = domain.StatusCLINotFound
			sq.topCode = "cli_not_found"
			sq.topMessage = "agy executable not found"
		} else {
			sq.topStatus = domain.StatusProviderError
			sq.topCode = "bin_error"
			sq.topMessage = "agy executable cannot be used"
		}
		return
	}

	env := childEnv()
	ver := p.command(ctx, bin, env, "--version")
	sq.cleanupFailed = ver.cleanupFailed
	if ver.err != nil {
		if errors.Is(ver.err, errAuthRequired) {
			sq.topStatus = domain.StatusAuthRequired
			sq.topCode = "auth_required"
			sq.topMessage = "authenticate using the agy CLI outside gaebokchi"
			return
		}
		sq.topStatus = commandErrorStatus(ver.err)
		sq.topCode = "version_query_failed"
		sq.topMessage = "agy version check failed"
		return
	}
	if ver.exitCode != 0 || !isSupportedVersion(string(ver.output)) {
		sq.topStatus = domain.StatusUnsupported
		sq.topCode = "version_unsupported"
		sq.topMessage = "agy CLI version must be >= 1.2.15"
		return
	}

	usage := p.command(ctx, bin, env, "--print", "/usage", "--output-format", "json", "--print-timeout", "15s")
	sq.cleanupFailed = sq.cleanupFailed || usage.cleanupFailed
	if usage.err != nil {
		if errors.Is(usage.err, errAuthRequired) {
			sq.topStatus = domain.StatusAuthRequired
			sq.topCode = "auth_required"
			sq.topMessage = "authenticate using the agy CLI outside gaebokchi"
			return
		}
		sq.topStatus = commandErrorStatus(usage.err)
		sq.topCode = "usage_query_failed"
		sq.topMessage = "agy usage query failed"
		return
	}
	if hasAuthRequired(usage.stderr) || hasAuthRequired(usage.output) {
		sq.topStatus = domain.StatusAuthRequired
		sq.topCode = "auth_required"
		sq.topMessage = "authenticate using the agy CLI outside gaebokchi"
		return
	}
	if usage.exitCode != 0 {
		switch usage.exitCode {
		case 3:
			sq.topStatus = domain.StatusTimeout
			sq.topCode = "usage_timeout"
			sq.topMessage = "agy usage query timed out"
		case 2:
			sq.topStatus = domain.StatusUnsupported
			sq.topCode = "usage_unsupported"
			sq.topMessage = "agy CLI rejected /usage flags"
		default:
			sq.topStatus = domain.StatusProviderError
			sq.topCode = "usage_error"
			sq.topMessage = "agy CLI usage command failed"
		}
		return
	}

	pools, status, code, msg := parseUsage(usage.output)
	if status != "" {
		sq.topStatus = status
		sq.topCode = code
		sq.topMessage = msg
		return
	}
	sq.pools = pools
}

// Query executes the shared agy CLI check once per binary and returns the
// target's weekly pool result (gemini-weekly for agy:0 / gemini, 3p-weekly for
// agy:1 / claude/gpt).
func (p *Provider) Query(ctx context.Context, tgt domain.Target) (res domain.Result) {
	res = domain.Result{
		TargetID:      tgt.ID,
		Provider:      tgt.Provider,
		Account:       tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias,
		ObservedAt:    time.Now(),
	}
	set := func(status domain.Status, code, message string) {
		res.Status = status
		res.Diagnostic = domain.Diagnostic{Code: code, Message: tgt.ID + ": " + message}
	}

	quotaID, ok := quotaIDForTarget(tgt)
	if !ok {
		set(domain.StatusUnsupported, "pool_unsupported", "unrecognized agy model pool target")
		return
	}

	sq := p.getQuery(tgt.Bin)
	sq.once.Do(func() {
		sq.execute(ctx, p, tgt.Bin)
	})
	res.CleanupFailed = sq.cleanupFailed
	if sq.topStatus != "" {
		set(sq.topStatus, sq.topCode, sq.topMessage)
		return
	}

	outcome, exists := sq.pools[quotaID]
	if !exists {
		set(domain.StatusUnavailableNoWeekly, string(domain.StatusUnavailableNoWeekly), "agy weekly quota entry is missing")
		return
	}
	if outcome.status != domain.StatusOK {
		set(outcome.status, outcome.code, outcome.message)
		return
	}
	res.Status = domain.StatusOK
	res.UsedPercent = outcome.percent
	res.ResetAt = outcome.resetAt
	res.Period = domain.PeriodWeekly
	res.QuotaID = quotaID
	res.Source = domain.SourceAgyCLIUsage
	return
}

func quotaIDForTarget(tgt domain.Target) (string, bool) {
	switch tgt.FallbackAccount {
	case AccountGemini:
		return QuotaGemini, true
	case AccountClaudeGP:
		return QuotaClaudeGPT, true
	}
	switch tgt.ID {
	case "agy:0":
		return QuotaGemini, true
	case "agy:1":
		return QuotaClaudeGPT, true
	}
	return "", false
}

func isSupportedVersion(raw string) bool {
	match := versionToken.FindStringSubmatch(raw)
	if match == nil {
		return false
	}
	major, err1 := strconv.Atoi(match[1])
	minor, err2 := strconv.Atoi(match[2])
	patch, err3 := strconv.Atoi(match[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	if major != 1 {
		return major > 1
	}
	if minor != 2 {
		return minor > 2
	}
	return patch >= 15
}

func commandErrorStatus(err error) domain.Status {
	switch {
	case errors.Is(err, context.Canceled):
		return domain.StatusCancelled
	case errors.Is(err, context.DeadlineExceeded):
		return domain.StatusTimeout
	case errors.Is(err, errOutputTooLarge):
		return domain.StatusProtocolError
	default:
		return domain.StatusProviderError
	}
}

type usageEnvelope struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	NumTurns *int   `json:"num_turns"`
	Usage    *struct {
		TotalTokens *int `json:"total_tokens"`
	} `json:"usage"`
	Command *struct {
		Name    string `json:"name"`
		Message string `json:"message"`
		Data    *struct {
			Groups []struct {
				Name    string      `json:"name"`
				Buckets []wireQuota `json:"buckets"`
				Quotas  []wireQuota `json:"quotas"`
			} `json:"groups"`
		} `json:"data"`
	} `json:"command"`
}

type wireQuota struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Label             string          `json:"label"`
	Window            string          `json:"window"`
	RemainingFraction json.RawMessage `json:"remaining_fraction"`
	ResetTime         string          `json:"reset_time"`
}

func parseUsage(raw []byte) (map[string]poolOutcome, domain.Status, string, string) {
	var env usageEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&env); err != nil {
		return nil, domain.StatusProtocolError, "protocol_error", "agy usage output is not valid JSON"
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, domain.StatusProtocolError, "protocol_error", "agy usage output contains trailing data"
	}
	if env.Command != nil && hasAuthRequired([]byte(env.Command.Message)) {
		return nil, domain.StatusAuthRequired, "auth_required", "authenticate using the agy CLI outside gaebokchi"
	}
	if (env.Type != "" && env.Type != "result") || env.Status != "SUCCESS" ||
		env.NumTurns == nil || *env.NumTurns != 0 ||
		(env.Usage != nil && env.Usage.TotalTokens != nil && *env.Usage.TotalTokens != 0) ||
		env.Command == nil || env.Command.Name != "usage" ||
		env.Command.Data == nil || env.Command.Data.Groups == nil {
		return nil, domain.StatusUnsupported, "usage_unsupported", "agy usage output is unsupported"
	}

	pools := make(map[string]poolOutcome, 2)
	seen := make(map[string]bool, 2)

	for _, g := range env.Command.Data.Groups {
		items := g.Buckets
		if len(items) == 0 {
			items = g.Quotas
		}
		for _, q := range items {
			if q.ID != QuotaGemini && q.ID != QuotaClaudeGPT {
				// Ignore non-weekly ("5h") or unrelated quota items.
				continue
			}
			if seen[q.ID] {
				pools[q.ID] = poolOutcome{
					status:  domain.StatusProtocolError,
					code:    "protocol_error",
					message: "duplicate agy weekly quota entry",
				}
				continue
			}
			seen[q.ID] = true

			if q.Window != "weekly" {
				pools[q.ID] = poolOutcome{
					status:  domain.StatusProtocolError,
					code:    "protocol_error",
					message: "agy weekly quota entry has unexpected window",
				}
				continue
			}
			token := strings.TrimSpace(string(q.RemainingFraction))
			if token == "" || token == "null" || strings.HasPrefix(token, `"`) {
				pools[q.ID] = poolOutcome{
					status:  domain.StatusProtocolError,
					code:    "protocol_error",
					message: "agy weekly quota entry is missing remaining_fraction",
				}
				continue
			}
			used, ok := domain.RemainingFractionToUsedPercent(token)
			if !ok {
				pools[q.ID] = poolOutcome{
					status:  domain.StatusProtocolError,
					code:    "protocol_error",
					message: "agy weekly quota remaining_fraction is invalid",
				}
				continue
			}
			v := used
			pools[q.ID] = poolOutcome{
				status:  domain.StatusOK,
				percent: &v,
				resetAt: parseResetTime(q.ResetTime),
			}
		}
	}

	for _, id := range []string{QuotaGemini, QuotaClaudeGPT} {
		if !seen[id] {
			pools[id] = poolOutcome{
				status:  domain.StatusUnavailableNoWeekly,
				code:    string(domain.StatusUnavailableNoWeekly),
				message: "agy weekly quota entry is missing",
			}
		}
	}
	return pools, "", "", ""
}

func parseResetTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}
