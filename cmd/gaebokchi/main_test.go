package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/options"
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
