// Package cursor reads the single default cursor-agent login's weekly Included
// subscription usage through `cursor-agent status --format json` with a
// read-only NODE_OPTIONS preload hook. It never sends a model prompt or opens
// an agent session.
package cursor

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
	outputLimit       = 256 << 10
	QuotaIncluded     = "cursor_included"
	periodUnspecified = 0
	periodMonthly     = 1
	periodWeekly      = 2

	minWeeklyCycle  = 6 * 24 * time.Hour
	maxWeeklyCycle  = 8 * 24 * time.Hour
	minMonthlyCycle = 27 * 24 * time.Hour
	maxMonthlyCycle = 32 * 24 * time.Hour
)

var errOutputTooLarge = errors.New("cursor output exceeds limit")

// Provider queries the default cursor-agent login via status --format json.
type Provider struct {
	Start      process.Factory
	TempParent string
}

// New builds a Cursor provider using the standard process factory.
func New() *Provider {
	return &Provider{Start: process.Start}
}

// ResolveBin resolves the configured cursor-agent executable.
func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("cursor: empty executable")
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
	r := envfilter.Cursor()
	r.Set = map[string]string{
		"NODE_OPTIONS": "--require=" + strconv.Quote(preloadPath),
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

// Query executes <cursor-bin> status --format json with the embedded read-only
// preload hook and normalizes the response for cursor:0.
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
			set(domain.StatusCLINotFound, "cli_not_found", "cursor-agent executable not found")
		} else {
			set(domain.StatusProviderError, "bin_error", "cursor-agent executable cannot be used")
		}
		return
	}

	if err := ctx.Err(); err != nil {
		set(commandErrorStatus(err), "status_query_failed", "cursor usage query failed")
		return
	}

	preloadDir, err := os.MkdirTemp(p.TempParent, "gaebokchi-cursor-")
	if err != nil {
		set(domain.StatusProviderError, "preload_error", "cannot prepare cursor preload directory")
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
		set(domain.StatusProviderError, "preload_error", "cannot secure cursor preload directory")
		return
	}
	preloadPath := filepath.Join(preloadDir, "preload.cjs")
	if err := os.WriteFile(preloadPath, preloadScript, 0o600); err != nil {
		set(domain.StatusProviderError, "preload_error", "cannot write cursor preload script")
		return
	}

	env := childEnv(preloadPath)
	cmd := p.command(ctx, bin, env, "status", "--format", "json")
	res.CleanupFailed = res.CleanupFailed || cmd.cleanupFailed
	if cmd.err != nil {
		set(commandErrorStatus(cmd.err), "status_query_failed", "cursor usage query failed")
		return
	}
	if cmd.exitCode != 0 {
		set(domain.StatusProviderError, "status_exit_error", "cursor CLI status command failed")
		return
	}

	outcome := parseStatus(cmd.output)
	if outcome.period != "" {
		res.Period = outcome.period
		res.QuotaID = QuotaIncluded
		res.Source = domain.SourceCursorCLIUsage
	}
	if outcome.status != domain.StatusOK {
		set(outcome.status, outcome.code, outcome.message)
		return
	}
	res.Status = domain.StatusOK
	res.UsedPercent = outcome.percent
	res.ResetAt = outcome.resetAt
	res.Period = domain.PeriodWeekly
	res.QuotaID = QuotaIncluded
	res.Source = domain.SourceCursorCLIUsage
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

type statusEnvelope struct {
	Status          *string `json:"status"`
	IsAuthenticated *bool   `json:"isAuthenticated"`
	HasAccessToken  *bool   `json:"hasAccessToken"`
	HasRefreshToken *bool   `json:"hasRefreshToken"`
	HasApiKey       *bool   `json:"hasApiKey"`
	PlanInfo        *struct {
		PlanName                *string `json:"planName"`
		IncludedUsagePeriod     *int    `json:"includedUsagePeriod"`
		IncludedUsageResetsAtMs *string `json:"includedUsageResetsAtMs"`
		BillingCycleEndMs       *string `json:"billingCycleEndMs"`
	} `json:"planInfo"`
	PlanError *struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"planError"`
	CurrentPeriodUsage *struct {
		BillingCycleStartMs *string `json:"billingCycleStartMs"`
		BillingCycleEndMs   *string `json:"billingCycleEndMs"`
		Enabled             *bool   `json:"enabled"`
		PlanUsage           *struct {
			TotalSpend       *float64 `json:"totalSpend"`
			IncludedSpend    *float64 `json:"includedSpend"`
			BonusSpend       *float64 `json:"bonusSpend"`
			Remaining        *float64 `json:"remaining"`
			Limit            *float64 `json:"limit"`
			AutoPercentUsed  *float64 `json:"autoPercentUsed"`
			ApiPercentUsed   *float64 `json:"apiPercentUsed"`
			TotalPercentUsed *float64 `json:"totalPercentUsed"`
		} `json:"planUsage"`
	} `json:"currentPeriodUsage"`
	UsageError *struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"usageError"`
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
			message: "cursor status output is not valid JSON",
		}
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return parsedOutcome{
			status:  domain.StatusProtocolError,
			code:    "protocol_error",
			message: "cursor status output contains trailing data",
		}
	}

	// Case 6: Unhooked plain status --format json lacks preload discriminator fields.
	if env.Status == nil || env.IsAuthenticated == nil ||
		env.HasAccessToken == nil || env.HasRefreshToken == nil || env.HasApiKey == nil {
		return parsedOutcome{
			status:  domain.StatusUnsupported,
			code:    "status_unsupported",
			message: "cursor status output is unsupported",
		}
	}

	// Case 4: API-key authentication has no subscription allowance.
	if *env.HasApiKey || *env.Status == "api_key" {
		return parsedOutcome{
			status:  domain.StatusNotApplicableAPIKey,
			code:    string(domain.StatusNotApplicableAPIKey),
			message: "API-key auth has no Cursor subscription quota",
		}
	}

	// Case 3: Unauthenticated or missing access/refresh token.
	if !*env.IsAuthenticated || !*env.HasAccessToken || !*env.HasRefreshToken || *env.Status == "unauthenticated" {
		return parsedOutcome{
			status:  domain.StatusAuthRequired,
			code:    "auth_required",
			message: "authenticate using the cursor-agent CLI outside gaebokchi",
		}
	}
	if *env.Status != "authenticated" {
		return parsedOutcome{
			status:  domain.StatusUnsupported,
			code:    "status_unsupported",
			message: "cursor status output is unsupported",
		}
	}

	// Case 5: ConnectRPC error on getPlanInfo or getCurrentPeriodUsage.
	if env.PlanError != nil || env.UsageError != nil {
		if (env.PlanError != nil && strings.EqualFold(env.PlanError.Code, "unauthenticated")) ||
			(env.UsageError != nil && strings.EqualFold(env.UsageError.Code, "unauthenticated")) {
			return parsedOutcome{
				status:  domain.StatusAuthRequired,
				code:    "auth_required",
				message: "authenticate using the cursor-agent CLI outside gaebokchi",
			}
		}
		return parsedOutcome{
			status:  domain.StatusProviderError,
			code:    "provider_error",
			message: "cursor usage query failed",
		}
	}

	period, ok := resolvePeriod(env)
	if !ok {
		return parsedOutcome{
			status:  domain.StatusUnavailablePeriod,
			code:    string(domain.StatusUnavailablePeriod),
			message: "cursor subscription period is unavailable",
		}
	}

	// Case 2: Monthly plan -> N/A (monthly).
	if period == domain.PeriodMonthly {
		return parsedOutcome{
			status:  domain.StatusNotApplicableMonthly,
			code:    string(domain.StatusNotApplicableMonthly),
			message: "cursor account reports a monthly period, not weekly",
			period:  domain.PeriodMonthly,
		}
	}

	// Weekly plan (Cases 1a, 1b, and 7).
	if env.CurrentPeriodUsage == nil || env.CurrentPeriodUsage.Enabled == nil ||
		!*env.CurrentPeriodUsage.Enabled || env.CurrentPeriodUsage.PlanUsage == nil {
		return parsedOutcome{
			status:  domain.StatusUnavailableNoData,
			code:    string(domain.StatusUnavailableNoData),
			message: "cursor weekly usage data is missing",
			period:  domain.PeriodWeekly,
		}
	}

	pu := env.CurrentPeriodUsage.PlanUsage
	if pu.Limit != nil {
		if *pu.Limit < 0 {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "cursor weekly usage limit is invalid",
				period:  domain.PeriodWeekly,
			}
		}
		if *pu.Limit > 0 && pu.IncludedSpend != nil && (*pu.IncludedSpend < 0 || *pu.IncludedSpend > *pu.Limit) {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "cursor weekly included spend is out of range",
				period:  domain.PeriodWeekly,
			}
		}
	}

	var used float64
	switch {
	case pu.TotalPercentUsed != nil:
		if !domain.ValidatePercent(*pu.TotalPercentUsed) {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "cursor weekly totalPercentUsed is out of range",
				period:  domain.PeriodWeekly,
			}
		}
		used = domain.NormalizePercentSignedZero(*pu.TotalPercentUsed)
	case pu.Limit != nil && *pu.Limit > 0:
		if pu.IncludedSpend == nil {
			return parsedOutcome{
				status:  domain.StatusUnavailableNoData,
				code:    string(domain.StatusUnavailableNoData),
				message: "cursor weekly usage percentage is missing",
				period:  domain.PeriodWeekly,
			}
		}
		pct := (*pu.IncludedSpend / *pu.Limit) * 100.0
		if !domain.ValidatePercent(pct) {
			return parsedOutcome{
				status:  domain.StatusProtocolError,
				code:    "protocol_error",
				message: "cursor weekly spend ratio is out of range",
				period:  domain.PeriodWeekly,
			}
		}
		used = domain.NormalizePercentSignedZero(pct)
	default:
		return parsedOutcome{
			status:  domain.StatusUnavailableNoData,
			code:    string(domain.StatusUnavailableNoData),
			message: "cursor weekly usage percentage is missing",
			period:  domain.PeriodWeekly,
		}
	}

	resetAt := resolveResetAt(
		env.PlanInfo.IncludedUsageResetsAtMs,
		env.CurrentPeriodUsage.BillingCycleEndMs,
	)
	return parsedOutcome{
		status:  domain.StatusOK,
		period:  domain.PeriodWeekly,
		percent: &used,
		resetAt: resetAt,
	}
}

func resolvePeriod(env statusEnvelope) (domain.PeriodKind, bool) {
	if env.PlanInfo == nil || env.PlanInfo.IncludedUsagePeriod == nil {
		return domain.PeriodUnknown, false
	}
	switch *env.PlanInfo.IncludedUsagePeriod {
	case periodWeekly:
		return domain.PeriodWeekly, true
	case periodMonthly:
		return domain.PeriodMonthly, true
	case periodUnspecified:
		if env.CurrentPeriodUsage == nil {
			return domain.PeriodUnknown, false
		}
		start, ok1 := parseEpochMs(env.CurrentPeriodUsage.BillingCycleStartMs)
		end, ok2 := parseEpochMs(env.CurrentPeriodUsage.BillingCycleEndMs)
		if !ok1 || !ok2 || !end.After(start) {
			return domain.PeriodUnknown, false
		}
		d := end.Sub(start)
		if d >= minWeeklyCycle && d <= maxWeeklyCycle {
			return domain.PeriodWeekly, true
		}
		if d >= minMonthlyCycle && d <= maxMonthlyCycle {
			return domain.PeriodMonthly, true
		}
		return domain.PeriodUnknown, false
	default:
		return domain.PeriodUnknown, false
	}
}

func resolveResetAt(primaryMs, fallbackMs *string) *time.Time {
	if t, ok := parseEpochMs(primaryMs); ok {
		return &t
	}
	if t, ok := parseEpochMs(fallbackMs); ok {
		return &t
	}
	return nil
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
