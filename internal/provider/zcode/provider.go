// Package zcode reads the single default zcode login's 5-hour Coding Plan
// subscription usage through `zcode doctor --json` with a read-only
// NODE_OPTIONS preload hook. It never sends a model prompt or opens an agent
// session.
package zcode

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/provider/envfilter"
)

//go:embed preload.cjs
var preloadScript []byte

const (
	outputLimit    = 256 << 10
	QuotaFiveHour  = "zcode_5h"
	unitHours      = 3
	windowFiveHour = 5
)

var errOutputTooLarge = errors.New("zcode output exceeds limit")

// Provider queries the default zcode login via doctor --json.
type Provider struct {
	Start      process.Factory
	TempParent string
}

// New builds a ZCode provider using the standard process factory.
func New() *Provider {
	return &Provider{Start: process.Start}
}

// ResolveBin resolves the configured zcode executable.
func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("zcode: empty executable")
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

func childEnv(preloadPath string) []string {
	r := envfilter.ZCode()
	r.Set = map[string]string{
		"NODE_OPTIONS": "--require=" + preloadPath,
	}
	return envfilter.Apply(os.Environ(), r)
}

type commandResult struct {
	output        []byte
	exitCode      int
	cleanupFailed bool
	err           error
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

	select {
	case <-ctx.Done():
		out.err = ctx.Err()
		return
	case read := <-readDone:
		if read.err != nil {
			out.err = read.err
			return
		}
		out.output = read.data
	}
	select {
	case <-ctx.Done():
		out.err = ctx.Err()
	case <-child.Done():
		out.exitCode = child.ExitCode()
	}
	return
}

// Query executes <zcode-bin> doctor --json with the embedded read-only preload
// hook and normalizes the 5-hour quota response for zcode:0.
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

	bin, err := ResolveBin(tgt.Bin)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			set(domain.StatusCLINotFound, "cli_not_found", "zcode executable not found")
		} else {
			set(domain.StatusProviderError, "bin_error", "zcode executable cannot be used")
		}
		return
	}

	if err := ctx.Err(); err != nil {
		set(commandErrorStatus(err), "status_query_failed", "zcode usage query failed")
		return
	}

	preloadDir, err := os.MkdirTemp(p.TempParent, "gaebokchi-zcode-")
	if err != nil {
		set(domain.StatusProviderError, "preload_error", "cannot prepare zcode preload directory")
		return
	}
	defer func() {
		if rmErr := os.RemoveAll(preloadDir); rmErr != nil {
			res.CleanupFailed = true
		}
		if res.CleanupFailed && res.Diagnostic.Message == "" {
			res.Diagnostic = domain.Diagnostic{
				Code:    "cleanup_failed",
				Message: tgt.ID + ": provider cleanup incomplete",
			}
		}
	}()
	if err := os.Chmod(preloadDir, 0o700); err != nil {
		set(domain.StatusProviderError, "preload_error", "cannot secure zcode preload directory")
		return
	}
	preloadPath := filepath.Join(preloadDir, "preload.cjs")
	if err := os.WriteFile(preloadPath, preloadScript, 0o600); err != nil {
		set(domain.StatusProviderError, "preload_error", "cannot write zcode preload script")
		return
	}

	env := childEnv(preloadPath)
	cmd := p.command(ctx, bin, env, "doctor", "--json")
	res.CleanupFailed = res.CleanupFailed || cmd.cleanupFailed
	if cmd.err != nil {
		set(commandErrorStatus(cmd.err), "status_query_failed", "zcode usage query failed")
		return
	}
	if cmd.exitCode != 0 {
		set(domain.StatusProviderError, "status_exit_error", "zcode CLI doctor command failed")
		return
	}

	outcome := parseStatus(cmd.output)
	if outcome.period != "" {
		res.Period = outcome.period
		res.QuotaID = QuotaFiveHour
		res.Source = domain.SourceZCodeCLIUsage
	}
	if outcome.status != domain.StatusOK {
		set(outcome.status, outcome.code, outcome.message)
		return
	}
	res.Status = domain.StatusOK
	res.UsedPercent = outcome.percent
	res.ResetAt = outcome.resetAt
	res.Period = domain.PeriodFiveHour
	res.QuotaID = QuotaFiveHour
	res.Source = domain.SourceZCodeCLIUsage
	return
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

type quotaLimit struct {
	Type          *string  `json:"type"`
	Unit          *int     `json:"unit"`
	Number        *float64 `json:"number"`
	Percentage    *float64 `json:"percentage"`
	Usage         *float64 `json:"usage"`
	CurrentValue  *float64 `json:"currentValue"`
	Remaining     *float64 `json:"remaining"`
	NextResetTime *string  `json:"nextResetTime"`
}

type statusEnvelope struct {
	Status          *string `json:"status"`
	IsAuthenticated *bool   `json:"isAuthenticated"`
	Family          *string `json:"family"`
	Quota           *struct {
		HTTPStatus *int          `json:"httpStatus"`
		Code       *int          `json:"code"`
		Success    *bool         `json:"success"`
		Limits     *[]quotaLimit `json:"limits"`
	} `json:"quota"`
	QuotaError *struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"quotaError"`
}

type parsedOutcome struct {
	status  domain.Status
	code    string
	message string
	period  domain.PeriodKind
	percent *float64
	resetAt *time.Time
}

func parseStatus(raw []byte) parsedOutcome {
	var env statusEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&env); err != nil {
		return parsedOutcome{
			status:  domain.StatusProtocolError,
			code:    "protocol_error",
			message: "zcode status output is not valid JSON",
		}
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return parsedOutcome{
			status:  domain.StatusProtocolError,
			code:    "protocol_error",
			message: "zcode status output contains trailing data",
		}
	}

	if env.Status == nil || *env.Status == "unsupported" {
		return parsedOutcome{
			status:  domain.StatusUnsupported,
			code:    "status_unsupported",
			message: "zcode status output is unsupported",
		}
	}

	if *env.Status == "unauthenticated" || env.IsAuthenticated == nil || !*env.IsAuthenticated {
		return parsedOutcome{
			status:  domain.StatusAuthRequired,
			code:    "auth_required",
			message: "authenticate using the zcode CLI outside gaebokchi",
		}
	}

	if *env.Status != "authenticated" {
		return parsedOutcome{
			status:  domain.StatusUnsupported,
			code:    "status_unsupported",
			message: "zcode status output is unsupported",
		}
	}

	if env.QuotaError != nil {
		code := strings.ToLower(strings.TrimSpace(env.QuotaError.Code))
		switch code {
		case "auth_required", "http_401", "http_403", "unauthenticated", "unauthorized":
			return parsedOutcome{
				status:  domain.StatusAuthRequired,
				code:    "auth_required",
				message: "authenticate using the zcode CLI outside gaebokchi",
			}
		case "timeout":
			return parsedOutcome{
				status:  domain.StatusTimeout,
				code:    "timeout",
				message: "zcode usage query timed out",
			}
		default:
			return parsedOutcome{
				status:  domain.StatusProviderError,
				code:    "provider_error",
				message: "zcode usage query failed",
			}
		}
	}

	if env.Quota == nil {
		return parsedOutcome{
			status:  domain.StatusUnavailableNoData,
			code:    string(domain.StatusUnavailableNoData),
			message: "zcode 5h usage data is missing",
			period:  domain.PeriodFiveHour,
		}
	}

	if env.Quota.HTTPStatus != nil {
		if *env.Quota.HTTPStatus == 401 || *env.Quota.HTTPStatus == 403 {
			return parsedOutcome{
				status:  domain.StatusAuthRequired,
				code:    "auth_required",
				message: "authenticate using the zcode CLI outside gaebokchi",
			}
		}
		if *env.Quota.HTTPStatus < 200 || *env.Quota.HTTPStatus >= 300 {
			return parsedOutcome{
				status:  domain.StatusProviderError,
				code:    "provider_error",
				message: "zcode usage query failed",
			}
		}
	}

	if env.Quota.Code != nil && *env.Quota.Code != 0 && *env.Quota.Code != 200 {
		if *env.Quota.Code == 401 || *env.Quota.Code == 403 {
			return parsedOutcome{
				status:  domain.StatusAuthRequired,
				code:    "auth_required",
				message: "authenticate using the zcode CLI outside gaebokchi",
			}
		}
		return parsedOutcome{
			status:  domain.StatusProviderError,
			code:    "provider_error",
			message: "zcode usage query failed",
		}
	}

	if env.Quota.Success != nil && !*env.Quota.Success {
		return parsedOutcome{
			status:  domain.StatusProviderError,
			code:    "provider_error",
			message: "zcode usage query failed",
		}
	}

	if env.Quota.Limits == nil {
		return parsedOutcome{
			status:  domain.StatusUnavailableNoData,
			code:    string(domain.StatusUnavailableNoData),
			message: "zcode 5h usage data is missing",
			period:  domain.PeriodFiveHour,
		}
	}

	limit, ok := findFiveHourLimit(*env.Quota.Limits)
	if !ok {
		return parsedOutcome{
			status:  domain.StatusUnavailableNoData,
			code:    string(domain.StatusUnavailableNoData),
			message: "zcode 5h usage data is missing",
			period:  domain.PeriodFiveHour,
		}
	}

	if limit.Usage != nil && *limit.Usage < 0 {
		return parsedOutcome{
			status:  domain.StatusProtocolError,
			code:    "protocol_error",
			message: "zcode 5h usage limit is invalid",
			period:  domain.PeriodFiveHour,
		}
	}

	var used float64
	switch {
	case limit.Percentage != nil:
		if !domain.ValidatePercent(*limit.Percentage) {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "zcode 5h percentage is out of range",
				period:  domain.PeriodFiveHour,
			}
		}
		used = domain.NormalizePercentSignedZero(*limit.Percentage)
	case limit.Usage != nil && *limit.Usage > 0 && limit.CurrentValue != nil:
		if *limit.CurrentValue < 0 || *limit.CurrentValue > *limit.Usage {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "zcode 5h currentValue is out of range",
				period:  domain.PeriodFiveHour,
			}
		}
		pct := (*limit.CurrentValue / *limit.Usage) * 100.0
		if !domain.ValidatePercent(pct) {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "zcode 5h usage ratio is out of range",
				period:  domain.PeriodFiveHour,
			}
		}
		used = domain.NormalizePercentSignedZero(pct)
	case limit.Usage != nil && *limit.Usage > 0 && limit.Remaining != nil:
		if *limit.Remaining < 0 || *limit.Remaining > *limit.Usage {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "zcode 5h remaining value is out of range",
				period:  domain.PeriodFiveHour,
			}
		}
		pct := ((*limit.Usage - *limit.Remaining) / *limit.Usage) * 100.0
		if !domain.ValidatePercent(pct) {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "zcode 5h usage ratio is out of range",
				period:  domain.PeriodFiveHour,
			}
		}
		used = domain.NormalizePercentSignedZero(pct)
	default:
		return parsedOutcome{
			status:  domain.StatusUnavailableNoData,
			code:    string(domain.StatusUnavailableNoData),
			message: "zcode 5h usage percentage is missing",
			period:  domain.PeriodFiveHour,
		}
	}

	var resetAt *time.Time
	if t, ok := parseEpochMs(limit.NextResetTime); ok {
		resetAt = &t
	}
	return parsedOutcome{
		status:  domain.StatusOK,
		period:  domain.PeriodFiveHour,
		percent: &used,
		resetAt: resetAt,
	}
}

func isTokenOrCreditLimit(t *string) bool {
	if t == nil {
		return false
	}
	return *t == "TOKENS_LIMIT" || *t == "CREDIT_LIMIT"
}

func findFiveHourLimit(limits []quotaLimit) (quotaLimit, bool) {
	for _, item := range limits {
		if isTokenOrCreditLimit(item.Type) &&
			item.Unit != nil && *item.Unit == unitHours &&
			item.Number != nil && *item.Number == windowFiveHour {
			return item, true
		}
	}
	return quotaLimit{}, false
}

func parseEpochMs(raw *string) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return time.Time{}, false
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ms <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(ms).UTC().Truncate(time.Second), true
}
