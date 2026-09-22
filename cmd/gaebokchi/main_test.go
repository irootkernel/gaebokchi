package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/options"
	"gaebokchi/internal/process"
	"gaebokchi/internal/provider/codex"
	"gaebokchi/internal/provider/grok"
	"gaebokchi/internal/testutil"
)

// fakeProv is an injectable provider returning scripted results.
type fakeProv struct {
	mu      sync.Mutex
	calls   []string
	results map[string]domain.Result
	delay   time.Duration
}

func (f *fakeProv) Query(ctx context.Context, tgt domain.Target) domain.Result {
	f.mu.Lock()
	f.calls = append(f.calls, tgt.ID)
	f.mu.Unlock()
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return domain.Result{TargetID: tgt.ID, Provider: tgt.Provider, Account: tgt.FallbackAccount,
				AccountSource: domain.AccountSourceConfiguredAlias, Status: domain.StatusCancelled}
		}
	}
	if res, ok := f.results[tgt.ID]; ok {
		res.TargetID = tgt.ID
		res.Provider = tgt.Provider
		return res
	}
	v := 10.0
	return domain.Result{TargetID: tgt.ID, Provider: tgt.Provider, Account: tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias, Status: domain.StatusOK, UsedPercent: &v}
}

func fakes() (codexQuerier, grokQuerier) {
	return &fakeProv{}, &fakeProv{results: map[string]domain.Result{
		"grok:0": {Account: "default", AccountSource: domain.AccountSourceConfiguredAlias, Status: domain.StatusOK, UsedPercent: ptr(28)},
	}}
}

func ptr(v float64) *float64 { return &v }

func parse(t *testing.T, args ...string) options.Options {
	t.Helper()
	o, err := options.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestUsageErrorsExitTwo(t *testing.T) {
	if code := run([]string{"--bogus"}); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if code := run([]string{"extra-arg"}); code != 2 {
		t.Fatalf("positional code = %d, want 2", code)
	}
}

// executeWith runs execute with test fakes.
func executeWith(opts options.Options, stdout, stderr *bytes.Buffer) int {
	cx, gk := fakes()
	return execute(opts, stdout, stderr, func() (codexQuerier, grokQuerier) { return cx, gk })
}

func TestDefaultTableExactlyFourLines(t *testing.T) {
	opts := parse(t, "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := executeWith(opts, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code = %d, stderr:\n%s", code, errBuf.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4 (header + 3 accounts):\n%s", len(lines), out.String())
	}
	if lines[0] != "AI PROVIDER  ACCOUNT  WEEKLY USAGE" && !strings.HasPrefix(lines[0], "AI PROVIDER") {
		t.Fatalf("header = %q", lines[0])
	}
	if !strings.Contains(out.String(), "ChatGPT") || !strings.Contains(out.String(), "Grok") {
		t.Fatalf("provider names missing:\n%s", out.String())
	}
	if errBuf.Len() != 0 {
		t.Fatalf("clean run produced diagnostics:\n%s", errBuf.String())
	}
}

func TestCustomHomesNPlusTwoLines(t *testing.T) {
	home := t.TempDir()
	opts := parse(t, "--codex-home", home+"/a", "--codex-home", home+"/b", "--codex-home", home+"/c", "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := executeWith(opts, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 5 { // N=3 codex homes + header + grok
		t.Fatalf("lines = %d, want 5:\n%s", len(lines), out.String())
	}
}

func TestPartialFailureKeepsTableExitOne(t *testing.T) {
	cx := &fakeProv{results: map[string]domain.Result{
		"codex:0": {Account: "a@example.com", AccountSource: domain.AccountSourceProtocolEmail, Status: domain.StatusOK, UsedPercent: ptr(37)},
		"codex:1": {Account: ".codex-hsy", AccountSource: domain.AccountSourceConfiguredAlias, Status: domain.StatusAuthRequired,
			Diagnostic: domain.Diagnostic{Code: "auth_required", Message: "codex:1: authenticate using the provider CLI outside gaebokchi"}},
	}}
	gk := &fakeProv{results: map[string]domain.Result{
		"grok:0": {Account: "default", AccountSource: domain.AccountSourceConfiguredAlias, Status: domain.StatusNotApplicableMonthly,
			Diagnostic: domain.Diagnostic{Code: "not_applicable/monthly", Message: "grok:0: grok account reports a monthly period, not weekly"}},
	}}
	opts := parse(t, "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier) { return cx, gk })
	if code != 1 {
		t.Fatalf("code = %d, want 1 for one incomplete row", code)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("partial failure must keep all rows: %d lines", len(lines))
	}
	if !strings.Contains(out.String(), "AUTH REQUIRED") {
		t.Fatal("failed row missing from table")
	}
	if !strings.Contains(errBuf.String(), "gaebokchi: codex:1: AUTH REQUIRED:") {
		t.Fatalf("diagnostic missing:\n%s", errBuf.String())
	}
}

func TestMonthlyAndAPIKeyAreCleanExitZero(t *testing.T) {
	cx := &fakeProv{results: map[string]domain.Result{
		"codex:0": {Account: "a@example.com", Status: domain.StatusOK, UsedPercent: ptr(1)},
		"codex:1": {Account: ".x", Status: domain.StatusNotApplicableAPIKey},
	}}
	gk := &fakeProv{results: map[string]domain.Result{
		"grok:0": {Account: "default", Status: domain.StatusNotApplicableMonthly},
	}}
	opts := parse(t, "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier) { return cx, gk })
	if code != 0 {
		t.Fatalf("code = %d, want 0 for non-applicable outcomes", code)
	}
}

func TestHelpAndVersionLaunchNoProviders(t *testing.T) {
	// The fake providers fail the test if queried.
	bomb := &fakeProv{results: map[string]domain.Result{
		"codex:0": {Status: domain.StatusProtocolError},
	}}
	var out, errBuf bytes.Buffer
	code := execute(parse(t, "--help"), &out, &errBuf, func() (codexQuerier, grokQuerier) { return bomb, bomb })
	if code != 0 || !strings.HasPrefix(out.String(), "Usage: gaebokchi") {
		t.Fatalf("help: code=%d out=%q", code, out.String())
	}
	if len(bomb.calls) != 0 {
		t.Fatal("help must not launch providers")
	}
	out.Reset()
	code = execute(parse(t, "--version"), &out, &errBuf, func() (codexQuerier, grokQuerier) { return bomb, bomb })
	if code != 0 || !strings.Contains(out.String(), "gaebokchi") {
		t.Fatalf("version: code=%d out=%q", code, out.String())
	}
	if len(bomb.calls) != 0 {
		t.Fatal("version must not launch providers")
	}
}

func TestBrokenStdoutExitsOne(t *testing.T) {
	opts := parse(t, "--timeout", "5s")
	errBuf := &bytes.Buffer{}
	code := execute(opts, errWriter{}, errBuf, func() (codexQuerier, grokQuerier) { return fakes() })
	if code != 1 {
		t.Fatalf("code = %d, want 1 for stdout failure", code)
	}
	if !strings.Contains(errBuf.String(), "cannot write") {
		t.Fatalf("safe diagnostic missing: %q", errBuf.String())
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) { return 0, fmt.Errorf("broken pipe") }

// TestSignalCancelsAndExitsWithSignalCode re-executes the test binary
// as a child, sends SIGINT mid-query, and asserts exit code 130 with a
// CANCELLED table still rendered.
func TestSignalCancelsAndExitsWithSignalCode(t *testing.T) {
	if os.Getenv("GAEBOKCHI_SIG_TEST") == "1" {
		slow := &fakeProv{delay: 30 * time.Second}
		opts := parse(t, "--timeout", "20s")
		var out, errBuf bytes.Buffer
		code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier) { return slow, slow })
		fmt.Fprintf(os.Stderr, "SIGTEST-TABLE:\n%s", out.String())
		os.Exit(code)
	}
	proc, err := os.StartProcess(os.Args[0], os.Args, &os.ProcAttr{
		Env:   append(os.Environ(), "GAEBOKCHI_SIG_TEST=1"),
		Files: []*os.File{os.Stdin, os.Stderr, os.Stderr},
	})
	if err != nil {
		t.Skip("cannot re-exec")
	}
	time.Sleep(500 * time.Millisecond) // child installs handlers and admits all targets
	_ = proc.Signal(syscall.SIGINT)
	state, err := proc.Wait()
	if err != nil {
		t.Fatal(err)
	}
	ws, ok := state.Sys().(syscall.WaitStatus)
	if !ok || !ws.Exited() {
		t.Fatalf("child did not exit cleanly: %v", state)
	}
	if ws.ExitStatus() != 130 {
		t.Fatalf("child exit = %d, want 130 (SIGINT)", ws.ExitStatus())
	}
}

// TestFakeHelperProcess is not a real test: the re-executed child for
// the cleanup-propagation tests runs the scripted fake provider.
func TestFakeHelperProcess(t *testing.T) {
	if os.Getenv("GAEBOKCHI_FAKE") != "1" {
		return
	}
	if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// startFakeChild starts a scripted fake child whose private directory
// lives under the given parent (see process.Config.TempParent).
func startFakeChild(t *testing.T, parent string, steps []testutil.Step) *process.Child {
	t.Helper()
	script, err := json.Marshal(steps)
	if err != nil {
		t.Fatal(err)
	}
	c, err := process.Start(process.Config{
		Bin:        os.Args[0],
		Args:       []string{"-test.run=TestFakeHelperProcess", "--", "fake"},
		Env:        append(os.Environ(), "GAEBOKCHI_FAKE=1", "GAEBOKCHI_FAKE_STEPS="+string(script)),
		TempParent: parent,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// cmdCodexSteps scripts a complete successful codex exchange.
func cmdCodexSteps(email string, pct int) []testutil.Step {
	account := fmt.Sprintf(`{"id":2,"result":{"account":{"type":"chatgpt","email":%q,"planType":"pro"}}}`, email)
	quota := fmt.Sprintf(`{"id":3,"result":{"rateLimitsByLimitId":{"codex":{"limitId":"codex","primary":{"usedPercent":%d,"windowDurationMins":10080,"resetsAt":1790411774},"secondary":null}}}}`, pct)
	return []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: account + "\n"},
		{Wait: 1},
		{Emit: quota + "\n"},
	}
}

// cmdGrokSteps scripts a complete successful grok exchange.
func cmdGrokSteps(pct int) []testutil.Step {
	init := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"authMethods":[{"id":"cached_token"},{"id":"grok.com"}],"defaultAuthMethodId":"cached_token"}}`
	auth := `{"jsonrpc":"2.0","id":2,"result":{"_meta":{"auth_mode":"Oidc"}}}`
	bill := fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"result":{"config":{"creditUsagePercent":%d.0,"currentPeriod":{"type":"USAGE_PERIOD_TYPE_WEEKLY","start":"2026-09-19T15:52:30Z","end":"2026-09-26T15:52:30Z"}}}}`, pct)
	return []testutil.Step{
		{Wait: 1}, {Emit: init + "\n"},
		{Wait: 1}, {Emit: auth + "\n"},
		{Wait: 1}, {Emit: bill + "\n"},
	}
}

// TestCleanupFailureKeepsRowsAndExitsOne runs the full CLI path — real
// codex/grok providers over scripted children, one of whose temporary
// directories cannot be removed — and pins the contract: both quotas
// stay numeric, exactly one bounded safe diagnostic names the failing
// target, healthy siblings are untouched, and the command exits 1.
func TestCleanupFailureKeepsRowsAndExitsOne(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based removal failure does not apply as root")
	}
	homeDir := t.TempDir()
	homeA := filepath.Join(homeDir, "a")
	homeB := filepath.Join(homeDir, "b")
	for _, h := range []string{homeA, homeB} {
		if err := os.MkdirAll(h, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	badParent := t.TempDir()
	childA := startFakeChild(t, "", cmdCodexSteps("a@example.com", 37))
	childB := startFakeChild(t, badParent, cmdCodexSteps("b@example.com", 12))
	childG := startFakeChild(t, "", cmdGrokSteps(28))
	if err := os.Chmod(badParent, 0o500); err != nil {
		t.Skipf("cannot chmod temp parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(badParent, 0o700) })

	// The factory selects its child by the per-target CODEX_HOME, so
	// concurrent admissions stay deterministic and share no counter.
	cx := &codex.Provider{Start: func(cfg process.Config) (*process.Child, error) {
		for _, kv := range cfg.Env {
			if kv == "CODEX_HOME="+homeB {
				return childB, nil
			}
		}
		return childA, nil
	}}
	gk := &grok.Provider{Start: func(process.Config) (*process.Child, error) { return childG, nil }}

	opts := parse(t, "--timeout", "10s",
		"--codex-home", homeA, "--codex-home", homeB,
		"--codex-bin", os.Args[0], "--grok-bin", os.Args[0])
	var out, errBuf bytes.Buffer
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier) { return cx, gk })

	if code != 1 {
		t.Fatalf("code = %d, want 1 for incomplete cleanup; stderr:\n%s", code, errBuf.String())
	}
	for _, want := range []string{"37.0%", "12.0%", "28.0%"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("numeric row lost (%s):\n%s", want, out.String())
		}
	}
	lines := strings.Split(strings.TrimSuffix(errBuf.String(), "\n"), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "codex:1") || !strings.Contains(lines[0], "provider cleanup incomplete") {
		t.Fatalf("expected exactly one safe cleanup diagnostic for codex:1, got:\n%s", errBuf.String())
	}
}

// TestFinalExitPrecedenceWithCleanupFailure pins the centralized exit
// precedence: a handled signal outranks a cleanup failure, which
// outranks clean rows.
func TestFinalExitPrecedenceWithCleanupFailure(t *testing.T) {
	failing := []domain.Result{{TargetID: "codex:0", Status: domain.StatusOK, UsedPercent: ptr(5), CleanupFailed: true}}
	clean := []domain.Result{{TargetID: "codex:0", Status: domain.StatusOK, UsedPercent: ptr(5)}}
	if got := finalExit(int64(syscall.SIGINT), false, failing); got != 130 {
		t.Fatalf("SIGINT + cleanup failure = %d, want 130", got)
	}
	if got := finalExit(int64(syscall.SIGTERM), false, failing); got != 143 {
		t.Fatalf("SIGTERM + cleanup failure = %d, want 143", got)
	}
	if got := finalExit(0, false, failing); got != 1 {
		t.Fatalf("cleanup failure = %d, want 1", got)
	}
	if got := finalExit(0, false, clean); got != 0 {
		t.Fatalf("clean run = %d, want 0", got)
	}
}
