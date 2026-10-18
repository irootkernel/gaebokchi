package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
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

func fakes() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
	return &fakeProv{}, &fakeProv{results: map[string]domain.Result{
		"grok:0": {Account: "default", AccountSource: domain.AccountSourceConfiguredAlias, Status: domain.StatusOK, UsedPercent: ptr(28)},
	}}, &fakeProv{}, &fakeProv{}, &fakeProv{}
}

func ptr(v float64) *float64 { return &v }

func parse(t testT, args ...string) options.Options {
	t.Helper()
	o, err := options.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

var _ = ginkgo.It("TestUsageErrorsExitTwo", func() {
	t := testT{ginkgo.GinkgoT()}
	var stdout, stderr bytes.Buffer
	if code := runWithIO([]string{"--bogus"}, &stdout, &stderr, fakes); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runWithIO([]string{"extra-arg"}, &stdout, &stderr, fakes); code != 2 {
		t.Fatalf("positional code = %d, want 2", code)
	}
})

var _ = ginkgo.It("missing configuration links to the dedicated guide before provider launch", func() {
	t := testT{ginkgo.GinkgoT()}
	const wantGuide = "https://github.com/irootkernel/gaebokchi/blob/main/configuration/README.md"
	if options.ConfigGuideURL != wantGuide || !strings.Contains(usage, wantGuide) {
		t.Fatalf("configuration guide URL is not used by errors and help")
	}
	t.Setenv("HOME", t.TempDir())
	bomb := &fakeProv{}
	providers := func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return bomb, bomb, bomb, bomb, bomb
	}
	var out, errBuf bytes.Buffer
	code := runWithIO(nil, &out, &errBuf, providers)
	if code != 2 || out.Len() != 0 || !strings.Contains(errBuf.String(), "config.yaml") ||
		!strings.Contains(errBuf.String(), wantGuide) || len(bomb.calls) != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q calls=%v", code, out.String(), errBuf.String(), bomb.calls)
	}
	out.Reset()
	errBuf.Reset()
	if code := runWithIO([]string{"--help"}, &out, &errBuf, providers); code != 0 || out.Len() == 0 {
		t.Fatalf("help without config: code=%d stderr=%q", code, errBuf.String())
	}
	out.Reset()
	if code := runWithIO([]string{"version"}, &out, &errBuf, providers); code != 0 || out.Len() == 0 {
		t.Fatalf("version without config: code=%d stderr=%q", code, errBuf.String())
	}
})

var _ = ginkgo.It("invalid configuration reports its path before provider launch", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".config", "gaebokchi", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("claude:\n  - name: same\n    path: /one\n  - name: same\n    path: /two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bomb := &fakeProv{}
	var out, errBuf bytes.Buffer
	code := runWithIO(nil, &out, &errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return bomb, bomb, bomb, bomb, bomb
	})
	if code != 2 || out.Len() != 0 || !strings.Contains(errBuf.String(), path) ||
		!strings.Contains(errBuf.String(), "claude account names must be unique") ||
		!strings.Contains(errBuf.String(), options.ConfigGuideURL) || len(bomb.calls) != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q calls=%v", code, out.String(), errBuf.String(), bomb.calls)
	}
})

var _ = ginkgo.It("TestVersionDispatchPrecedesOptionsAndProviders", func() {
	t := testT{ginkgo.GinkgoT()}
	bomb := &fakeProv{results: map[string]domain.Result{"codex:0": {Status: domain.StatusProtocolError}}}
	newProviders := func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return bomb, bomb, bomb, bomb, bomb
	}
	var stdout, stderr bytes.Buffer
	code := runWithIO([]string{"version", "--json"}, &stdout, &stderr, newProviders)
	if code != 0 || stdout.String() != "{\"name\":\"gaebokchi\",\"version\":\"(devel)\"}\n" || stderr.Len() != 0 {
		t.Fatalf("version dispatch: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if len(bomb.calls) != 0 {
		t.Fatal("version command launched a provider")
	}

	stdout.Reset()
	stderr.Reset()
	code = runWithIO([]string{"--version"}, &stdout, &stderr, newProviders)
	if code != 2 || !strings.Contains(stderr.String(), `unknown flag "--version"`) {
		t.Fatalf("removed flag: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if len(bomb.calls) != 0 {
		t.Fatal("removed version flag launched a provider")
	}
})

// executeWith runs execute with test fakes.
func executeWith(opts options.Options, stdout, stderr *bytes.Buffer) int {
	cx, gk, cl, ag, cu := fakes()
	return execute(opts, stdout, stderr, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return cx, gk, cl, ag, cu
	})
}

var _ = ginkgo.It("TestDefaultTableExactlyFiveLines", func() {
	t := testT{ginkgo.GinkgoT()}
	opts := parse(t, "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := executeWith(opts, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code = %d, stderr:\n%s", code, errBuf.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5 (header + 4 accounts):\n%s", len(lines), out.String())
	}
	if lines[0] != "AI PROVIDER  ACCOUNT  WEEKLY USAGE" && !strings.HasPrefix(lines[0], "AI PROVIDER") {
		t.Fatalf("header = %q", lines[0])
	}
	if !strings.Contains(out.String(), "ChatGPT") || !strings.Contains(out.String(), "Grok") || !strings.Contains(out.String(), "Claude") {
		t.Fatalf("provider names missing:\n%s", out.String())
	}
	if errBuf.Len() != 0 {
		t.Fatalf("clean run produced diagnostics:\n%s", errBuf.String())
	}
})

var _ = ginkgo.It("TestCustomHomesNPlusTwoLines", func() {
	t := testT{ginkgo.GinkgoT()}
	setTestConfig(t, "codex:\n  - name: a\n    path: ~/codex-a\n  - name: b\n    path: ~/codex-b\n  - name: c\n    path: ~/codex-c\ngrok:\n  - name: default\n    path: ~/grok\nclaude:\n  - name: default\n    path: ~/claude\n")
	opts := parse(t, "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := executeWith(opts, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 6 { // N=3 codex homes + header + grok + claude
		t.Fatalf("lines = %d, want 6:\n%s", len(lines), out.String())
	}
})

var _ = ginkgo.It("TestPartialFailureKeepsTableExitOne", func() {
	t := testT{ginkgo.GinkgoT()}
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
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return cx, gk, &fakeProv{}, &fakeProv{}, &fakeProv{}
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1 for one incomplete row", code)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("partial failure must keep all rows: %d lines", len(lines))
	}
	if !strings.Contains(out.String(), "AUTH REQUIRED") {
		t.Fatal("failed row missing from table")
	}
	if !strings.Contains(errBuf.String(), "gaebokchi: codex:1: AUTH REQUIRED:") {
		t.Fatalf("diagnostic missing:\n%s", errBuf.String())
	}
})

var _ = ginkgo.It("TestJSONPartialFailureKeepsOrderedResults", func() {
	t := testT{ginkgo.GinkgoT()}
	cx := &fakeProv{results: map[string]domain.Result{
		"codex:0": {Account: "same@example.com", Status: domain.StatusOK, UsedPercent: ptr(37.125), CleanupFailed: true,
			Diagnostic: domain.Diagnostic{Code: "cleanup_failed", Message: "codex:0: cleanup failed"}},
		"codex:1": {Account: "same@example.com", Status: domain.StatusAuthRequired,
			Diagnostic: domain.Diagnostic{Code: "auth_required", Message: "codex:1: authentication required"}},
	}}
	var out, errBuf bytes.Buffer
	code := runWithIO([]string{"--json"}, &out, &errBuf,
		func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
			return cx, &fakeProv{}, &fakeProv{}, &fakeProv{}, &fakeProv{}
		})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	var snapshot struct {
		Results []struct {
			TargetID      string   `json:"target_id"`
			Account       string   `json:"account"`
			Status        string   `json:"status"`
			UsedPercent   *float64 `json:"used_percent"`
			CleanupFailed bool     `json:"cleanup_failed"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &snapshot); err != nil {
		t.Fatalf("stdout is not JSON: %v; output: %q", err, out.String())
	}
	if len(snapshot.Results) != 4 || snapshot.Results[0].TargetID != "codex:0" ||
		snapshot.Results[1].TargetID != "codex:1" || snapshot.Results[2].TargetID != "grok:0" ||
		snapshot.Results[3].TargetID != "claude:0" || snapshot.Results[0].UsedPercent == nil ||
		*snapshot.Results[0].UsedPercent != 37.125 || !snapshot.Results[0].CleanupFailed ||
		snapshot.Results[0].Account != "first" ||
		snapshot.Results[1].Account != "second" ||
		snapshot.Results[1].Status != "auth_required" || snapshot.Results[1].UsedPercent != nil {
		t.Fatalf("JSON results = %+v", snapshot.Results)
	}
	if !strings.Contains(errBuf.String(), "AUTH REQUIRED") {
		t.Fatalf("stderr diagnostics missing: %q", errBuf.String())
	}
})

var _ = ginkgo.It("TestJSONOutputFailureExitsOne", func() {
	t := testT{ginkgo.GinkgoT()}
	var errBuf bytes.Buffer
	code := runWithIO([]string{"--json"}, errWriter{}, &errBuf, fakes)
	if code != 1 || !strings.Contains(errBuf.String(), "cannot write usage JSON") {
		t.Fatalf("code=%d stderr=%q", code, errBuf.String())
	}
})

var _ = ginkgo.It("treats short stdout writes as output failures", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, tc := range []struct {
		args    []string
		message string
	}{
		{nil, "cannot write the usage table to stdout"},
		{[]string{"--json"}, "cannot write usage JSON to stdout"},
		{[]string{"--help"}, "cannot write help to stdout"},
	} {
		var errBuf bytes.Buffer
		code := runWithIO(tc.args, shortWriter{}, &errBuf, fakes)
		if code != 1 || !strings.Contains(errBuf.String(), tc.message) {
			t.Fatalf("args=%v code=%d stderr=%q", tc.args, code, errBuf.String())
		}
	}
})

var _ = ginkgo.It("TestJSONCleanHelpAndInvalidUsage", func() {
	t := testT{ginkgo.GinkgoT()}
	var out, errBuf bytes.Buffer
	if code := runWithIO([]string{"--json"}, &out, &errBuf, fakes); code != 0 || errBuf.Len() != 0 {
		t.Fatalf("clean JSON: code=%d stderr=%q", code, errBuf.String())
	}
	if !json.Valid(out.Bytes()) || !strings.HasSuffix(out.String(), "}\n") {
		t.Fatalf("clean JSON output = %q", out.String())
	}
	out.Reset()
	if code := runWithIO([]string{"--json", "--help"}, &out, &errBuf, fakes); code != 0 ||
		!strings.HasPrefix(out.String(), "Usage:\n") || !strings.Contains(out.String(), "--json") ||
		!strings.Contains(out.String(), "Interactive terminals show a live usage view.") ||
		!strings.Contains(out.String(), "Pipes and redirects keep the") {
		t.Fatalf("JSON help: code=%d stdout=%q", code, out.String())
	}
	out.Reset()
	if code := runWithIO([]string{"--json", "--bogus"}, &out, &errBuf, fakes); code != 2 || out.Len() != 0 {
		t.Fatalf("JSON usage error: code=%d stdout=%q", code, out.String())
	}
})

var _ = ginkgo.It("TestMonthlyAndAPIKeyAreCleanExitZero", func() {
	t := testT{ginkgo.GinkgoT()}
	cx := &fakeProv{results: map[string]domain.Result{
		"codex:0": {Account: "a@example.com", Status: domain.StatusOK, UsedPercent: ptr(1)},
		"codex:1": {Account: ".x", Status: domain.StatusNotApplicableAPIKey},
	}}
	gk := &fakeProv{results: map[string]domain.Result{
		"grok:0": {Account: "default", Status: domain.StatusNotApplicableMonthly},
	}}
	opts := parse(t, "--timeout", "5s")
	var out, errBuf bytes.Buffer
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return cx, gk, &fakeProv{}, &fakeProv{}, &fakeProv{}
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0 for non-applicable outcomes", code)
	}
})

var _ = ginkgo.It("TestHelpLaunchesNoProviders", func() {
	t := testT{ginkgo.GinkgoT()}
	// The fake providers fail the test if queried.
	bomb := &fakeProv{results: map[string]domain.Result{
		"codex:0": {Status: domain.StatusProtocolError},
	}}
	var out, errBuf bytes.Buffer
	code := execute(parse(t, "--help"), &out, &errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return bomb, bomb, bomb, bomb, bomb
	})
	if code != 0 || !strings.HasPrefix(out.String(), "Usage:\n  gaebokchi") {
		t.Fatalf("help: code=%d out=%q", code, out.String())
	}
	if len(bomb.calls) != 0 {
		t.Fatal("help must not launch providers")
	}
})

var _ = ginkgo.It("TestBrokenStdoutExitsOne", func() {
	t := testT{ginkgo.GinkgoT()}
	opts := parse(t, "--timeout", "5s")
	errBuf := &bytes.Buffer{}
	code := execute(opts, errWriter{}, errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) { return fakes() })
	if code != 1 {
		t.Fatalf("code = %d, want 1 for stdout failure", code)
	}
	if !strings.Contains(errBuf.String(), "cannot write") {
		t.Fatalf("safe diagnostic missing: %q", errBuf.String())
	}
})

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) { return 0, fmt.Errorf("broken pipe") }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return max(0, len(p)-1), nil }

// TestSignalCancelsAndExitsWithSignalCode re-executes the test binary
// as a child, sends SIGINT mid-query, and asserts exit code 130 with a
// CANCELLED table still rendered.
var _ = ginkgo.It("TestSignalCancelsAndExitsWithSignalCode", func() {
	t := testT{ginkgo.GinkgoT()}

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
})

// startFakeChild starts a scripted fake child whose private directory
// lives under the given parent (see process.Config.TempParent).
func startFakeChild(t testT, parent string, steps []testutil.Step) *process.Child {
	t.Helper()
	script, err := json.Marshal(steps)
	if err != nil {
		t.Fatal(err)
	}
	c, err := process.Start(process.Config{
		Bin:        os.Args[0],
		Args:       []string{"--", "fake"},
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
var _ = ginkgo.It("TestCleanupFailureKeepsRowsAndExitsOne", func() {
	t := testT{ginkgo.GinkgoT()}
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

	setTestConfig(t, fmt.Sprintf("codex:\n  - name: a\n    path: %s\n  - name: b\n    path: %s\ngrok:\n  - name: default\n    path: ~/grok\n", homeA, homeB))
	opts := parse(t, "--timeout", "10s", "--codex-bin", os.Args[0], "--grok-bin", os.Args[0])
	var out, errBuf bytes.Buffer
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return cx, gk, &fakeProv{}, &fakeProv{}, &fakeProv{}
	})

	if code != 1 {
		t.Fatalf("code = %d, want 1 for incomplete cleanup; stderr:\n%s", code, errBuf.String())
	}
	for _, want := range []string{"37%", "12%", "28%"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("numeric row lost (%s):\n%s", want, out.String())
		}
	}
	lines := strings.Split(strings.TrimSuffix(errBuf.String(), "\n"), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "codex:1") || !strings.Contains(lines[0], "provider cleanup incomplete") {
		t.Fatalf("expected exactly one safe cleanup diagnostic for codex:1, got:\n%s", errBuf.String())
	}
})

// TestFinalExitPrecedenceWithCleanupFailure pins the centralized exit
// precedence: a handled signal outranks a cleanup failure, which
// outranks clean rows.
var _ = ginkgo.It("TestFinalExitPrecedenceWithCleanupFailure", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

// TestUnicodeWhitespaceAliasExitsTwoBeforeProviders pins the
// confirmation review's finding through the actual CLI entry: an alias
// built only from Unicode spaces is a usage error (exit 2) before any
// provider can launch.
var _ = ginkgo.It("TestUnicodeWhitespaceAliasExitsTwoBeforeProviders", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, alias := range []string{"\u3000", "\u205f", "\u1680", " \u3000\t\u205f"} {
		if code := run([]string{"--grok-account", alias}); code != 2 {
			t.Fatalf("alias %q: code = %d, want 2", alias, code)
		}
	}
})

var _ = ginkgo.It("TestAgyTwoWeeklyRowsInTableAndJSON", func() {
	t := testT{ginkgo.GinkgoT()}
	setTestConfig(t, "agy:\n  - name: default\n")
	ag := &fakeProv{results: map[string]domain.Result{
		"agy:0": {Status: domain.StatusOK, UsedPercent: ptr(74.15718734264374), Period: domain.PeriodWeekly},
		"agy:1": {Status: domain.StatusOK, UsedPercent: ptr(12.76919841766357), Period: domain.PeriodWeekly},
	}}
	prov := func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return &fakeProv{}, &fakeProv{}, &fakeProv{}, ag, &fakeProv{}
	}
	var out, errBuf bytes.Buffer
	code := runWithIO(nil, &out, &errBuf, prov)
	if code != 0 || errBuf.Len() != 0 {
		t.Fatalf("table run failed: code=%d stderr=%q", code, errBuf.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 3 || !strings.Contains(lines[1], "agy") || !strings.Contains(lines[1], "gemini") || !strings.Contains(lines[1], "74.2%") ||
		!strings.Contains(lines[2], "agy") || !strings.Contains(lines[2], "claude/gpt") || !strings.Contains(lines[2], "12.8%") {
		t.Fatalf("unexpected agy table:\n%s", out.String())
	}

	out.Reset()
	code = runWithIO([]string{"--json"}, &out, &errBuf, prov)
	if code != 0 || errBuf.Len() != 0 {
		t.Fatalf("json run failed: code=%d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"target_id":"agy:0"`) || !strings.Contains(out.String(), `"account":"gemini"`) ||
		!strings.Contains(out.String(), `"target_id":"agy:1"`) || !strings.Contains(out.String(), `"account":"claude/gpt"`) {
		t.Fatalf("unexpected agy JSON:\n%s", out.String())
	}
})

var _ = ginkgo.It("TestCursorRowInTableAndJSON", func() {
	t := testT{ginkgo.GinkgoT()}
	setTestConfig(t, "cursor:\n  - name: default\n")
	cu := &fakeProv{results: map[string]domain.Result{
		"cursor:0": {
			Status:      domain.StatusOK,
			UsedPercent: ptr(25),
			Period:      domain.PeriodWeekly,
			Source:      domain.SourceCursorCLIUsage,
		},
	}}
	prov := func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return &fakeProv{}, &fakeProv{}, &fakeProv{}, &fakeProv{}, cu
	}
	var out, errBuf bytes.Buffer
	code := runWithIO(nil, &out, &errBuf, prov)
	if code != 0 || errBuf.Len() != 0 {
		t.Fatalf("table run failed: code=%d stderr=%q", code, errBuf.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "Cursor") || !strings.Contains(lines[1], "default") || !strings.Contains(lines[1], "25%") {
		t.Fatalf("unexpected cursor table:\n%s", out.String())
	}

	out.Reset()
	code = runWithIO([]string{"--json"}, &out, &errBuf, prov)
	if code != 0 || errBuf.Len() != 0 {
		t.Fatalf("json run failed: code=%d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"target_id":"cursor:0"`) || !strings.Contains(out.String(), `"provider":"cursor"`) ||
		!strings.Contains(out.String(), `"account":"default"`) || !strings.Contains(out.String(), `"used_percent":25`) {
		t.Fatalf("unexpected cursor JSON:\n%s", out.String())
	}

	cuMonthly := &fakeProv{results: map[string]domain.Result{
		"cursor:0": {
			Status: domain.StatusNotApplicableMonthly,
			Period: domain.PeriodMonthly,
			Source: domain.SourceCursorCLIUsage,
			Diagnostic: domain.Diagnostic{
				Code:    string(domain.StatusNotApplicableMonthly),
				Message: "cursor:0: cursor account reports a monthly period, not weekly",
			},
		},
	}}
	provMonthly := func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return &fakeProv{}, &fakeProv{}, &fakeProv{}, &fakeProv{}, cuMonthly
	}
	out.Reset()
	errBuf.Reset()
	code = runWithIO(nil, &out, &errBuf, provMonthly)
	if code != 0 || !strings.Contains(out.String(), "N/A (monthly)") || !strings.Contains(errBuf.String(), "cursor:0: N/A (monthly):") {
		t.Fatalf("monthly cursor table failed: code=%d stdout=%q stderr=%q", code, out.String(), errBuf.String())
	}
})

func runSignalTestChild() {
	slow := &fakeProv{delay: 30 * time.Second}
	opts, err := options.Parse([]string{"--timeout", "20s"})
	if err != nil {
		os.Exit(2)
	}
	var out, errBuf bytes.Buffer
	code := execute(opts, &out, &errBuf, func() (codexQuerier, grokQuerier, claudeQuerier, agyQuerier, cursorQuerier) {
		return slow, slow, slow, slow, slow
	})
	fmt.Fprintf(os.Stderr, "SIGTEST-TABLE:\n%s", out.String())
	os.Exit(code)
}
