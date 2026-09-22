// Package claude reads the subscription's weekly all-models allowance through
// Claude Code's built-in /usage command. It never sends a model prompt.
package claude

import (
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
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/provider/envfilter"
)

const outputLimit = 256 << 10

var weeklyLine = regexp.MustCompile(`^Current week \(all models\): ([0-9]+(?:\.[0-9]+)?)% used(?:[ \t].*)?$`)

type Provider struct {
	Start process.Factory
}

func New() *Provider { return &Provider{Start: process.Start} }

func ResolveBin(bin string) (string, error) {
	if bin == "" {
		return "", errors.New("claude: empty executable")
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
	return envfilter.Apply(os.Environ(), envfilter.Claude())
}

type commandResult struct {
	output        []byte
	exitCode      int
	cleanupFailed bool
	err           error
}

var errOutputTooLarge = errors.New("claude output exceeds limit")

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

func (p *Provider) Query(ctx context.Context, tgt domain.Target) (res domain.Result) {
	res = domain.Result{
		TargetID: tgt.ID, Provider: tgt.Provider, Account: tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias, ObservedAt: time.Now(),
	}
	set := func(status domain.Status, code, message string) {
		res.Status = status
		res.Diagnostic = domain.Diagnostic{Code: code, Message: tgt.ID + ": " + message}
	}
	bin, err := ResolveBin(tgt.Bin)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			set(domain.StatusCLINotFound, "cli_not_found", "claude executable not found")
		} else {
			set(domain.StatusProviderError, "bin_error", "claude executable cannot be used")
		}
		return
	}
	env := childEnv()
	auth := p.command(ctx, bin, env, "--safe-mode", "auth", "status", "--json")
	res.CleanupFailed = auth.cleanupFailed
	if auth.err != nil {
		set(commandErrorStatus(auth.err), "auth_query_failed", "claude authentication check failed")
		return
	}
	var account struct {
		LoggedIn    *bool  `json:"loggedIn"`
		AuthMethod  string `json:"authMethod"`
		APIProvider string `json:"apiProvider"`
	}
	if err := json.Unmarshal(auth.output, &account); err != nil || account.LoggedIn == nil {
		set(domain.StatusUnsupported, "auth_unsupported", "claude authentication output is unsupported")
		return
	}
	if !*account.LoggedIn {
		set(domain.StatusAuthRequired, "auth_required", "authenticate using the claude CLI outside gaebokchi")
		return
	}
	if auth.exitCode != 0 {
		set(domain.StatusProviderError, "auth_error", "claude authentication check failed")
		return
	}
	if account.AuthMethod != "claude.ai" || account.APIProvider != "firstParty" {
		method := strings.ToLower(account.AuthMethod)
		if strings.Contains(method, "api") && strings.Contains(method, "key") {
			set(domain.StatusNotApplicableAPIKey, "api_key_auth", "API-key auth has no Claude subscription quota")
		} else {
			set(domain.StatusUnsupported, "auth_unsupported", "claude authentication mode is unsupported")
		}
		return
	}

	usage := p.command(ctx, bin, env, "--safe-mode", "--tools", "", "-p", "/usage",
		"--max-turns", "0", "--output-format", "json", "--no-session-persistence")
	res.CleanupFailed = res.CleanupFailed || usage.cleanupFailed
	if usage.err != nil {
		set(commandErrorStatus(usage.err), "usage_query_failed", "claude usage query failed")
		return
	}
	if usage.exitCode != 0 {
		set(domain.StatusUnsupported, "usage_unsupported", "claude CLI did not return a usage result")
		return
	}
	percent, status := parseUsage(usage.output)
	if status != domain.StatusOK {
		set(status, string(status), "claude weekly usage output is unsupported")
		return
	}
	res.Status = domain.StatusOK
	res.UsedPercent = &percent
	res.Period = domain.PeriodWeekly
	res.QuotaID = "claude_all_models"
	res.Source = domain.SourceClaudeCLIUsage
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

func parseUsage(raw []byte) (float64, domain.Status) {
	var envelope struct {
		LocalCommand string   `json:"local_command"`
		NumTurns     *int     `json:"num_turns"`
		TotalCostUSD *float64 `json:"total_cost_usd"`
		IsError      *bool    `json:"is_error"`
		Result       *string  `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.LocalCommand != "usage" ||
		envelope.NumTurns == nil || *envelope.NumTurns != 0 ||
		envelope.TotalCostUSD == nil || *envelope.TotalCostUSD != 0 ||
		envelope.IsError == nil || *envelope.IsError || envelope.Result == nil {
		return 0, domain.StatusUnsupported
	}
	var found *float64
	for _, line := range strings.Split(*envelope.Result, "\n") {
		match := weeklyLine.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if match == nil {
			continue
		}
		if found != nil {
			return 0, domain.StatusUnsupported
		}
		v, err := strconv.ParseFloat(match[1], 64)
		if err != nil || !domain.ValidatePercent(v) {
			return 0, domain.StatusProtocolError
		}
		v = domain.NormalizePercentSignedZero(v)
		found = &v
	}
	if found == nil {
		return 0, domain.StatusUnsupported
	}
	return *found, domain.StatusOK
}
