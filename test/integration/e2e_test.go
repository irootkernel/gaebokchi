package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"gaebokchi/internal/testutil"
)

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// gaebokchiBin builds the real executable once for the whole suite.
func gaebokchiBin(t testT) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "gaebokchi-e2e-")
		if err != nil {
			buildErr = err
			return
		}
		registerOwnedDir(dir)
		binPath = filepath.Join(dir, "gaebokchi")
		out, err := exec.Command("go", "build", "-o", binPath, "gaebokchi/cmd/gaebokchi").CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build: %v: %s", err, out)
		}
	})
	if buildErr != nil {
		t.Fatalf("build gaebokchi: %v", buildErr)
	}
	return binPath
}

// fakeBin is this test binary acting as the fake provider CLI.
func fakeBin(t testT) string {
	t.Helper()
	p, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// run invokes the built gaebokchi with fake provider scripts.
type invocation struct {
	args              []string
	env               []string
	config            string
	codex             []testutil.Step
	grok              []testutil.Step
	grokUsageByHome   map[string]int
	claudeUsageByHome map[string]int
	cursorPayload     string
	zcodePayload      string
	argLog            string
	envLog            string
}

func setupConfigHome(t testT, config string) string {
	t.Helper()
	home := t.TempDir()
	for _, name := range []string{"e2e-a", "e2e-b", "grok", "claude", ".claude"} {
		if err := os.Mkdir(filepath.Join(home, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if config == "" {
		config = "codex:\n  - name: e2e-a\n    path: ~/e2e-a\n  - name: e2e-b\n    path: ~/e2e-b\ngrok:\n  - name: default\n    path: ~/grok\nclaude:\n  - name: default\n    path: ~/claude\n"
	}
	path := filepath.Join(home, ".config", "gaebokchi", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func runCLI(t testT, inv invocation) (stdout, stderr string, code int) {
	t.Helper()
	bin := gaebokchiBin(t)
	fake := fakeBin(t)
	codexJSON, _ := json.Marshal(inv.codex)
	grokJSON, _ := json.Marshal(inv.grok)

	args := make([]string, 0, len(inv.args)+12)
	if !hasFlag(inv.args, "--timeout") {
		args = append(args, "--timeout", "5s")
	}
	if !hasFlag(inv.args, "--codex-bin") {
		args = append(args, "--codex-bin", fake)
	}
	if !hasFlag(inv.args, "--grok-bin") {
		args = append(args, "--grok-bin", fake)
	}
	if !hasFlag(inv.args, "--claude-bin") {
		args = append(args, "--claude-bin", fake)
	}
	if !hasFlag(inv.args, "--agy-bin") {
		args = append(args, "--agy-bin", fake)
	}
	if !hasFlag(inv.args, "--cursor-bin") {
		args = append(args, "--cursor-bin", fake)
	}
	if !hasFlag(inv.args, "--zcode-bin") {
		args = append(args, "--zcode-bin", fake)
	}
	args = append(args, inv.args...)

	cmd := exec.Command(bin, args...)
	home := setupConfigHome(t, inv.config)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"GAEBOKCHI_FAKE=1",
		"GAEBOKCHI_FAKE_CODEX_STEPS="+string(codexJSON),
		"GAEBOKCHI_FAKE_GROK_STEPS="+string(grokJSON),
	)
	if inv.argLog != "" {
		cmd.Env = append(cmd.Env, "GAEBOKCHI_FAKE_ARGLOG="+inv.argLog, "GAEBOKCHI_FAKE_ENVLOG="+inv.envLog)
	}
	if inv.cursorPayload != "" {
		cmd.Env = append(cmd.Env, "GAEBOKCHI_FAKE_CURSOR_PAYLOAD="+inv.cursorPayload)
	}
	if inv.zcodePayload != "" {
		cmd.Env = append(cmd.Env, "GAEBOKCHI_FAKE_ZCODE_PAYLOAD="+inv.zcodePayload)
	}
	if len(inv.grokUsageByHome) > 0 {
		byHome, err := json.Marshal(inv.grokUsageByHome)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "GAEBOKCHI_FAKE_GROK_USAGE_BY_HOME="+string(byHome))
	}
	if len(inv.claudeUsageByHome) > 0 {
		byHome, err := json.Marshal(inv.claudeUsageByHome)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "GAEBOKCHI_FAKE_CLAUDE_USAGE_BY_HOME="+string(byHome))
	}
	cmd.Env = append(cmd.Env, inv.env...)

	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code = 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run gaebokchi: %v", err)
	}
	return out.String(), errBuf.String(), code
}

type usageJSONRow struct {
	TargetID      string   `json:"target_id"`
	Provider      string   `json:"provider"`
	Account       string   `json:"account"`
	Status        string   `json:"status"`
	UsedPercent   *float64 `json:"used_percent"`
	CleanupFailed bool     `json:"cleanup_failed"`
}

func decodeUsageJSON(t testT, stdout string) []usageJSONRow {
	t.Helper()
	if strings.Count(stdout, "\n") != 1 || !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("usage JSON must be one newline-terminated line: %q", stdout)
	}
	var snapshot struct {
		Results []usageJSONRow `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &snapshot); err != nil {
		t.Fatalf("usage stdout is not one JSON document: %v; output: %q", err, stdout)
	}
	return snapshot.Results
}

// codexOK scripts a full successful Codex exchange.
func codexOK(email string, pct int) []testutil.Step {
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

func grokWeekly(pct int) []testutil.Step {
	init := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"authMethods":[{"id":"cached_token","name":"cached_token"},{"id":"grok.com","name":"Grok"}],"defaultAuthMethodId":"cached_token"}}`
	auth := `{"jsonrpc":"2.0","id":2,"result":{"_meta":{"auth_mode":"Oidc","subscription_tier":"SuperGrok Heavy"}}}`
	bill := fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"result":{"config":{"creditUsagePercent":%d.0,"currentPeriod":{"type":"USAGE_PERIOD_TYPE_WEEKLY","start":"2026-09-19T15:52:30Z","end":"2026-09-26T15:52:30Z"}}}}`, pct)
	return []testutil.Step{
		{Wait: 1},
		{Emit: init + "\n"},
		{Wait: 1},
		{Emit: auth + "\n"},
		{Wait: 1},
		{Emit: bill + "\n"},
	}
}

var _ = ginkgo.It("TestDefaultHappyPath", func() {
	t := testT{ginkgo.GinkgoT()}
	stdout, stderr, code := runCLI(t, invocation{
		codex: codexOK("account1@example.com", 37),
		grok:  grokWeekly(28),
	})
	gomega.Expect(code).To(gomega.Equal(0), stderr)
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d:\n%s", len(lines), stdout)
	}
	if !strings.Contains(stdout, "e2e-a") || !strings.Contains(stdout, "37%") || !strings.Contains(stdout, "28%") {
		t.Fatalf("table content wrong:\n%s", stdout)
	}
	if strings.Contains(stdout, "FAKE") || strings.Contains(stdout, "jsonrpc") {
		t.Fatalf("raw provider data on stdout:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestJSONExecutableHappyPath", func() {
	t := testT{ginkgo.GinkgoT()}
	stdout, stderr, code := runCLI(t, invocation{
		args:  []string{"--json"},
		codex: codexOK("account1@example.com", 37),
		grok:  grokWeekly(28),
	})
	if code != 0 || stderr != "" {
		t.Fatalf("JSON success: code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 4 || rows[0].TargetID != "codex:0" || rows[1].TargetID != "codex:1" ||
		rows[2].TargetID != "grok:0" || rows[3].TargetID != "claude:0" ||
		rows[0].Account != "e2e-a" || rows[1].Account != "e2e-b" {
		t.Fatalf("JSON result order or identity: %+v", rows)
	}
	for i, want := range []float64{37, 37, 28, 43} {
		if rows[i].Status != "ok" || rows[i].UsedPercent == nil ||
			*rows[i].UsedPercent != want || rows[i].CleanupFailed {
			t.Fatalf("JSON row %d: %+v", i, rows[i])
		}
	}
})

var _ = ginkgo.It("TestJSONExecutablePartialFailure", func() {
	t := testT{ginkgo.GinkgoT()}
	authFail := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: `{"id":2,"result":{"account":null}}` + "\n"},
	}
	stdout, stderr, code := runCLI(t, invocation{
		args:  []string{"--json"},
		codex: authFail,
		grok:  grokWeekly(3),
	})
	if code != 1 {
		t.Fatalf("JSON partial failure: code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 4 || rows[0].Status != "auth_required" ||
		rows[1].Status != "auth_required" || rows[0].UsedPercent != nil ||
		rows[1].UsedPercent != nil || rows[2].Status != "ok" ||
		rows[2].UsedPercent == nil || *rows[2].UsedPercent != 3 ||
		rows[3].Status != "ok" || rows[3].UsedPercent == nil ||
		*rows[3].UsedPercent != 43 {
		t.Fatalf("JSON partial results: %+v", rows)
	}
	if strings.Count(stderr, "gaebokchi: codex:") != 2 ||
		!strings.Contains(stderr, "AUTH REQUIRED") || strings.Contains(stderr, `"account":null`) {
		t.Fatalf("JSON partial diagnostics: %q", stderr)
	}
})

var _ = ginkgo.It("TestMissingExecutableKeepsOtherRows", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	stdout, _, code := runCLI(t, invocation{
		args:  []string{"--grok-bin", filepath.Join(dir, "no-such-grok")},
		codex: codexOK("a@example.com", 11),
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "NOT INSTALLED") {
		t.Fatalf("grok row missing:\n%s", stdout)
	}
	if strings.Count(stdout, "\n") != 5 {
		t.Fatalf("row count changed:\n%s", stdout)
	}
	if !strings.Contains(stdout, "11%") {
		t.Fatalf("codex rows lost:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestMissingClaudeExecutableKeepsDefaultRow", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	stdout, stderr, code := runCLI(t, invocation{
		args:  []string{"--claude-bin", filepath.Join(dir, "no-such-claude")},
		codex: codexOK("a@example.com", 11),
		grok:  grokWeekly(28),
	})
	if code != 1 || strings.Count(stdout, "\n") != 5 ||
		!strings.Contains(stdout, "Claude") || !strings.Contains(stdout, "NOT INSTALLED") ||
		!strings.Contains(stderr, "claude:0: NOT INSTALLED") {
		t.Fatalf("missing Claude row: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
})

var _ = ginkgo.It("TestMissingHomeRow", func() {
	t := testT{ginkgo.GinkgoT()}
	stdout, _, code := runCLI(t, invocation{
		config: "codex:\n  - name: missing\n    path: /definitely/not/here\ngrok:\n  - name: default\n    path: ~/grok\n",
		grok:   grokWeekly(5),
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "HOME NOT FOUND") {
		t.Fatalf("missing home row:\n%s", stdout)
	}
	if !strings.Contains(stdout, "5%") {
		t.Fatalf("other rows lost:\n%s", stdout)
	}
})

var _ = ginkgo.It("missing Grok home keeps its row and a healthy Claude result", func() {
	t := testT{ginkgo.GinkgoT()}
	stdout, stderr, code := runCLI(t, invocation{
		config: "grok:\n  - name: missing\n    path: /definitely/not/here\nclaude:\n  - name: healthy\n    path: ~/claude\n",
	})
	if code != 1 || !strings.Contains(stdout, "missing") || !strings.Contains(stdout, "HOME NOT FOUND") ||
		!strings.Contains(stdout, "healthy") || !strings.Contains(stdout, "43%") ||
		!strings.Contains(stderr, "grok:0: HOME NOT FOUND") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
})

var _ = ginkgo.It("TestFailedAuthKeepsTable", func() {
	t := testT{ginkgo.GinkgoT()}
	authFail := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: `{"id":2,"result":{"account":null}}` + "\n"},
	}
	stdout, stderr, code := runCLI(t, invocation{
		codex: authFail,
		grok:  grokWeekly(3),
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "AUTH REQUIRED") {
		t.Fatalf("auth row missing:\n%s", stdout)
	}
	if !strings.Contains(stderr, "gaebokchi: codex:0: AUTH REQUIRED:") {
		t.Fatalf("diagnostic missing:\n%s", stderr)
	}
})

var _ = ginkgo.It("TestMonthlyGrokIsClean", func() {
	t := testT{ginkgo.GinkgoT()}
	monthly := grokWeekly(88)
	monthly[4] = testutil.Step{Wait: 1}
	monthly = append(monthly[:4], testutil.Step{Wait: 1},
		testutil.Step{Emit: `{"jsonrpc":"2.0","id":3,"result":{"config":{"creditUsagePercent":88.0,"currentPeriod":{"type":"USAGE_PERIOD_TYPE_MONTHLY"}}}}` + "\n"})
	stdout, _, code := runCLI(t, invocation{
		codex: codexOK("a@example.com", 10),
		grok:  monthly,
	})
	if code != 0 {
		t.Fatalf("monthly must be exit-clean, got %d", code)
	}
	if !strings.Contains(stdout, "N/A (monthly)") || strings.Contains(stdout, "88") {
		t.Fatalf("monthly handling wrong:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestMalformedGrokResponseIsProtocolError", func() {
	t := testT{ginkgo.GinkgoT()}
	bad := []testutil.Step{
		{Wait: 1},
		{Emit: "Grok CLI banner, not JSON\n"},
	}
	stdout, _, code := runCLI(t, invocation{
		codex: codexOK("a@example.com", 12),
		grok:  bad,
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "PROTOCOL ERROR") {
		t.Fatalf("protocol row missing:\n%s", stdout)
	}
	if strings.Contains(stdout, "banner") {
		t.Fatalf("raw banner leaked:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestWrongIDsNeverSatisfyRequest", func() {
	t := testT{ginkgo.GinkgoT()}
	// Keep the authenticate exchange, then answer only with a wrong
	// id and exit: the billing request must fail, not be satisfied.
	grok := append(grokWeekly(1)[:4],
		testutil.Step{Emit: `{"jsonrpc":"2.0","id":99,"result":{"config":{"creditUsagePercent":1.0,"currentPeriod":{"type":"USAGE_PERIOD_TYPE_WEEKLY"}}}}` + "\n"})
	stdout, _, code := runCLI(t, invocation{
		codex: codexOK("a@example.com", 9),
		grok:  grok,
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if strings.Contains(stdout, "  1%") {
		t.Fatalf("wrong-id response satisfied the grok row:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestLongStderrDoesNotLeak", func() {
	t := testT{ginkgo.GinkgoT()}
	grok := grokWeekly(7)
	grok = append([]testutil.Step{{EmitStderr: strings.Repeat("E2E-STDERR-NOISE ", 3800)}}, grok...)
	stdout, stderr, code := runCLI(t, invocation{
		codex: codexOK("a@example.com", 8),
		grok:  grok,
	})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if strings.Contains(stdout, "E2E-STDERR-NOISE") || strings.Contains(stderr, "E2E-STDERR-NOISE") {
		t.Fatal("child stderr leaked into gaebokchi output")
	}
	if !strings.Contains(stdout, "7%") {
		t.Fatalf("stderr flood broke the query:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestHungProviderTimesOut", func() {
	t := testT{ginkgo.GinkgoT()}
	hung := []testutil.Step{{Wait: 1}, {SleepMs: 60000}}
	stdout, _, code := runCLI(t, invocation{
		args:  []string{"--timeout", "1s"},
		codex: hung,
		grok:  grokWeekly(2),
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "TIMEOUT") {
		t.Fatalf("timeout row missing:\n%s", stdout)
	}
	if !strings.Contains(stdout, "2%") {
		t.Fatalf("healthy sibling lost:\n%s", stdout)
	}
})

var _ = ginkgo.It("TestUnsafeCallbackRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	grok := grokWeekly(4)
	grok[4] = testutil.Step{Emit: `{"jsonrpc":"2.0","id":77,"method":"session/prompt","params":{"prompt":"steal"}}` + "\n"}
	grok = append(grok, testutil.Step{Wait: 1}, testutil.Step{Wait: 1})
	stdout, stderr, code := runCLI(t, invocation{
		codex: codexOK("a@example.com", 6),
		grok:  grok,
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "PROTOCOL ERROR") {
		t.Fatalf("callback row:\n%s", stdout)
	}
	if strings.Contains(stdout, "steal") || strings.Contains(stderr, "steal") {
		t.Fatal("callback payload leaked")
	}
})

var _ = ginkgo.It("TestSecretMarkersNeverLeak", func() {
	t := testT{ginkgo.GinkgoT()}
	grok := grokWeekly(9)
	grok[2] = testutil.Step{Reply: []testutil.ReplySpec{{Slot: 1, ErrCode: intPtr(-32000), ErrMsg: "SECRET-TOKEN-XYZ leaked"}}}
	stdout, stderr, _ := runCLI(t, invocation{
		codex: codexOK("a@example.com", 5),
		grok:  grok,
		env:   []string{"XAI_API_KEY=ENV-SECRET-MARKER", "GROK_API_KEY=ENV-SECRET-MARKER-2"},
	})
	for _, out := range []string{stdout, stderr} {
		for _, marker := range []string{"SECRET-TOKEN-XYZ", "ENV-SECRET-MARKER", "ENV-SECRET-MARKER-2"} {
			if strings.Contains(out, marker) {
				t.Fatalf("secret marker %q leaked:\n%s", marker, out)
			}
		}
	}
})

var _ = ginkgo.It("TestEnvironmentIsolationAndArgv", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	argLog := filepath.Join(dir, "argv.log")
	envLog := filepath.Join(dir, "env.log")
	stdout, _, code := runCLI(t, invocation{
		codex:  codexOK("a@example.com", 13),
		grok:   grokWeekly(14),
		argLog: argLog,
		envLog: envLog,
		env:    []string{"OPENAI_API_KEY=CODEX-ENV-SECRET", "XAI_API_KEY=GROK-ENV-SECRET"},
	})
	if code != 0 {
		t.Fatalf("code = %d:\n%s", code, stdout)
	}
	log, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "app-server") {
		t.Fatalf("codex invoked without app-server: %q", log)
	}
	if !strings.Contains(string(log), "--no-auto-update") || !strings.Contains(string(log), "agent") || !strings.Contains(string(log), "stdio") {
		t.Fatalf("grok invocation wrong: %q", log)
	}
	envDump, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	seenCodex, seenGrok := false, false
	for _, block := range strings.Split(string(envDump), "=== child") {
		header, environment, ok := strings.Cut(block, "\n")
		if !ok {
			continue
		}
		if strings.Contains(header, "app-server") {
			seenCodex = true
			if strings.Contains("\n"+environment, "\nOPENAI_API_KEY=CODEX-ENV-SECRET\n") {
				t.Fatal("openai key override reached the codex child")
			}
			if !strings.Contains("\n"+environment, "\nCODEX_HOME=") {
				t.Fatal("CODEX_HOME not set for codex child")
			}
		}
		if strings.Contains(header, "agent") && strings.Contains(header, "stdio") {
			seenGrok = true
			if strings.Contains("\n"+environment, "\nXAI_API_KEY=GROK-ENV-SECRET\n") {
				t.Fatal("xai key override reached the grok child")
			}
		}
	}
	if !seenCodex || !seenGrok {
		t.Fatalf("missing child environment blocks: codex=%t grok=%t", seenCodex, seenGrok)
	}
})

var _ = ginkgo.It("TestSignalCancellationExits130", func() {
	t := testT{ginkgo.GinkgoT()}
	bin := gaebokchiBin(t)
	fake := fakeBin(t)
	codexJSON, _ := json.Marshal([]testutil.Step{{Wait: 1}, {SleepMs: 60000}})
	grokJSON, _ := json.Marshal([]testutil.Step{{Wait: 1}, {SleepMs: 60000}})
	cmd := exec.Command(bin,
		"--codex-bin", fake, "--grok-bin", fake, "--claude-bin", fake, "--timeout", "20s")
	cmd.Env = append(os.Environ(),
		"HOME="+setupConfigHome(t, ""),
		"GAEBOKCHI_FAKE=1",
		"GAEBOKCHI_FAKE_CODEX_STEPS="+string(codexJSON),
		"GAEBOKCHI_FAKE_GROK_STEPS="+string(grokJSON),
	)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(700 * time.Millisecond) // children admitted and hanging
	_ = cmd.Process.Signal(syscall.SIGINT)
	waitErr := cmd.Wait()
	if waitErr == nil {
		t.Fatal("SIGINT cancellation must exit nonzero")
	}
	ee, ok := waitErr.(*exec.ExitError)
	if !ok {
		t.Fatal(waitErr)
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Exited() {
		t.Fatalf("unexpected wait state: %v", waitErr)
	}
	if ws.ExitStatus() != 130 {
		t.Fatalf("exit = %d, want 130", ws.ExitStatus())
	}
	if !strings.Contains(out.String(), "CANCELLED") {
		t.Fatalf("cancelled table missing:\n%s", out.String())
	}
})

var _ = ginkgo.It("TestConcurrentAccountIsolation", func() {
	t := testT{ginkgo.GinkgoT()}
	// Two homes with distinct scripted identities must not mix; the
	// shared script serves both children identically, so verify via a
	// second run with distinct homes through arg log cwd/env.
	dir := t.TempDir()
	argLog := filepath.Join(dir, "argv.log")
	homeA := filepath.Join(dir, "work")
	homeB := filepath.Join(dir, "personal")
	if err := os.MkdirAll(homeA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeB, 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, _, code := runCLI(t, invocation{
		config: fmt.Sprintf("codex:\n  - name: work\n    path: %s\n  - name: personal\n    path: %s\n", homeA, homeB),
		codex:  codexOK("shared@example.com", 21),
		grok:   grokWeekly(22),
		argLog: argLog,
		envLog: filepath.Join(dir, "env.log"),
	})
	if code != 0 {
		t.Fatalf("code = %d:\n%s", code, stdout)
	}
	if !strings.Contains(stdout, "work") || !strings.Contains(stdout, "personal") {
		t.Fatalf("both homes must produce their own row:\n%s", stdout)
	}
	if strings.Contains(stdout, "shared@example.com") {
		t.Fatalf("configured aliases were replaced by provider email:\n%s", stdout)
	}
	log, _ := os.ReadFile(argLog)
	if strings.Count(string(log), "app-server") != 2 {
		t.Fatalf("expected two codex children, log: %q", log)
	}
})

var _ = ginkgo.It("queries multiple Grok and Claude homes and excludes omitted providers", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "grok-work"), filepath.Join(dir, "grok-personal"),
		filepath.Join(dir, "claude-work"), filepath.Join(dir, "claude-personal")}
	for _, path := range paths {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	config := fmt.Sprintf("grok:\n  - name: work\n    path: %s\n  - name: personal\n    path: %s\nclaude:\n  - name: work\n    path: %s\n  - name: personal\n    path: %s\n", paths[0], paths[1], paths[2], paths[3])
	argLog, envLog := filepath.Join(dir, "args.log"), filepath.Join(dir, "env.log")
	stdout, stderr, code := runCLI(t, invocation{config: config, args: []string{"--json"},
		grokUsageByHome:   map[string]int{paths[0]: 28, paths[1]: 62},
		claudeUsageByHome: map[string]int{paths[2]: 43, paths[3]: 75},
		argLog:            argLog, envLog: envLog,
		env: []string{"GROK_HOME=/wrong-grok", "CLAUDE_CONFIG_DIR=/wrong-claude", "CLAUDE_CODE_OAUTH_TOKEN=fake-ambient-token"}})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 4 || rows[0].TargetID != "grok:0" || rows[1].TargetID != "grok:1" ||
		rows[2].TargetID != "claude:0" || rows[3].TargetID != "claude:1" ||
		rows[0].Account != "work" || rows[1].Account != "personal" ||
		rows[2].Account != "work" || rows[3].Account != "personal" {
		t.Fatalf("configured rows = %+v", rows)
	}
	for i, want := range []float64{28, 62, 43, 75} {
		if rows[i].Status != "ok" || rows[i].UsedPercent == nil || *rows[i].UsedPercent != want {
			t.Fatalf("row %d usage = %+v, want %.0f%%", i, rows[i], want)
		}
	}
	log, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(log), "app-server") || strings.Count(string(log), "agent stdio") != 2 ||
		strings.Count(string(log), "auth status") != 2 {
		t.Fatalf("disabled or duplicate provider launch: %q", log)
	}
	env, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := "GROK_HOME="
		wantCount := 1
		if strings.Contains(path, "claude") {
			name = "CLAUDE_CONFIG_DIR="
			wantCount = 2 // auth status and usage use the same selected home
		}
		if got := strings.Count(string(env), "\n"+name+path+"\n"); got != wantCount {
			t.Fatalf("%s%s occurrences = %d, want %d", name, path, got, wantCount)
		}
	}
	for _, block := range strings.Split(string(env), "=== child") {
		header, _, _ := strings.Cut(block, "\n")
		if strings.Contains(header, "agent stdio") && strings.Contains(block, "GROK_HOME=/wrong-grok") {
			t.Fatal("ambient Grok home reached a Grok child")
		}
		if strings.Contains(header, "--safe-mode") &&
			(strings.Contains(block, "CLAUDE_CONFIG_DIR=/wrong-claude") ||
				strings.Contains(block, "CLAUDE_CODE_OAUTH_TOKEN=")) {
			t.Fatal("ambient Claude override reached a Claude child")
		}
	}
})

var _ = ginkgo.It("uses implicit Claude credentials for the default home", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	envLog := filepath.Join(dir, "env.log")
	stdout, stderr, code := runCLI(t, invocation{
		config: "claude:\n  - name: default\n    path: ~/.claude\n",
		args:   []string{"--json"},
		argLog: filepath.Join(dir, "args.log"), envLog: envLog,
		env: []string{"CLAUDE_CONFIG_DIR=/ambient/claude"},
	})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 1 || rows[0].TargetID != "claude:0" || rows[0].Status != "ok" ||
		rows[0].UsedPercent == nil || *rows[0].UsedPercent != 43 {
		t.Fatalf("default Claude row = %+v", rows)
	}
	logged, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(logged), "=== child") != 2 || strings.Contains(string(logged), "\nCLAUDE_CONFIG_DIR=") {
		t.Fatalf("default Claude child received a config override")
	}
})

var _ = ginkgo.It("queries agy once and emits gemini and claude/gpt weekly rows in table and JSON", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	argLog, envLog := filepath.Join(dir, "args.log"), filepath.Join(dir, "env.log")
	stdout, stderr, code := runCLI(t, invocation{
		config: "agy:\n  - name: default\n",
		args:   []string{"--json"},
		argLog: argLog, envLog: envLog,
		env: []string{
			"ANTIGRAVITY_LS_ADDRESS=127.0.0.1:9999",
			"GEMINI_API_KEY=secret-gemini-key",
		},
	})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 2 ||
		rows[0].TargetID != "agy:0" || rows[0].Provider != "agy" || rows[0].Account != "gemini" ||
		rows[0].Status != "ok" || rows[0].UsedPercent == nil || *rows[0].UsedPercent != 74.15718734264374 ||
		rows[1].TargetID != "agy:1" || rows[1].Provider != "agy" || rows[1].Account != "claude/gpt" ||
		rows[1].Status != "ok" || rows[1].UsedPercent == nil || *rows[1].UsedPercent != 12.76919841766357 {
		t.Fatalf("agy JSON rows = %+v", rows)
	}
	args, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(args)) != "argv: --version\nargv: --print /usage --output-format json --print-timeout 15s" {
		t.Fatalf("unexpected agy argv log: %q", args)
	}
	env, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(env), "ANTIGRAVITY_LS_ADDRESS=") ||
		strings.Contains(string(env), "GEMINI_API_KEY=") ||
		!strings.Contains(string(env), "\nAGY_CLI_DISABLE_AUTO_UPDATE=true\n") {
		t.Fatalf("agy child env was not filtered: %q", env)
	}

	tableOut, tableErr, tableCode := runCLI(t, invocation{
		config: "agy:\n  - name: default\n",
	})
	if tableCode != 0 || tableErr != "" {
		t.Fatalf("table code=%d stderr=%q", tableCode, tableErr)
	}
	lines := strings.Split(strings.TrimSuffix(tableOut, "\n"), "\n")
	if len(lines) != 3 ||
		!strings.Contains(lines[1], "agy") || !strings.Contains(lines[1], "gemini") || !strings.Contains(lines[1], "74.2%") ||
		!strings.Contains(lines[2], "agy") || !strings.Contains(lines[2], "claude/gpt") || !strings.Contains(lines[2], "12.8%") {
		t.Fatalf("unexpected agy table:\n%s", tableOut)
	}
})

var _ = ginkgo.It("queries Cursor once and emits its weekly or monthly row in table and JSON", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	argLog, envLog := filepath.Join(dir, "args.log"), filepath.Join(dir, "env.log")
	stdout, stderr, code := runCLI(t, invocation{
		config: "cursor:\n  - name: default\n",
		args:   []string{"--json"},
		argLog: argLog, envLog: envLog,
		env: []string{
			"CURSOR_API_KEY=secret-cursor-key",
			"NODE_OPTIONS=--inspect=0.0.0.0:9229",
		},
	})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 1 ||
		rows[0].TargetID != "cursor:0" || rows[0].Provider != "cursor" || rows[0].Account != "default" ||
		rows[0].Status != "ok" || rows[0].UsedPercent == nil || *rows[0].UsedPercent != 25 || rows[0].CleanupFailed {
		t.Fatalf("cursor JSON rows = %+v", rows)
	}
	args, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(args)) != "argv: status --format json" {
		t.Fatalf("unexpected cursor argv log: %q", args)
	}
	env, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(env), "CURSOR_API_KEY=") ||
		strings.Contains(string(env), "--inspect=") ||
		!strings.Contains(string(env), "\nNODE_OPTIONS=--require=") {
		t.Fatalf("cursor child env was not filtered: %q", env)
	}

	tableOut, tableErr, tableCode := runCLI(t, invocation{
		config: "cursor:\n  - name: default\n",
	})
	if tableCode != 0 || tableErr != "" {
		t.Fatalf("table code=%d stderr=%q", tableCode, tableErr)
	}
	lines := strings.Split(strings.TrimSuffix(tableOut, "\n"), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "Cursor") || !strings.Contains(lines[1], "default") || !strings.Contains(lines[1], "25%") {
		t.Fatalf("unexpected cursor table:\n%s", tableOut)
	}

	monthlyPayload := `{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"hasApiKey":false,"planInfo":{"planName":"Pro","includedUsagePeriod":1,"includedUsageResetsAtMs":"1794664361314","billingCycleEndMs":"1794664361314"},"planError":null,"currentPeriodUsage":{"billingCycleStartMs":"1791985961314","billingCycleEndMs":"1794664361314","enabled":true,"planUsage":{"totalSpend":1000,"includedSpend":1000,"bonusSpend":0,"remaining":1000,"limit":2000,"totalPercentUsed":50}},"usageError":null}`
	monthlyOut, monthlyErr, monthlyCode := runCLI(t, invocation{
		config:        "cursor:\n  - name: default\n",
		cursorPayload: monthlyPayload,
	})
	if monthlyCode != 0 || !strings.Contains(monthlyOut, "N/A (monthly)") || !strings.Contains(monthlyErr, "cursor:0: N/A (monthly):") {
		t.Fatalf("monthly cursor code=%d stdout=%q stderr=%q", monthlyCode, monthlyOut, monthlyErr)
	}
})

var _ = ginkgo.It("queries zcode 5h usage and renders weekly and 5h sections", func() {
	t := testT{ginkgo.GinkgoT()}
	argLog := filepath.Join(t.TempDir(), "args.log")
	envLog := filepath.Join(t.TempDir(), "env.log")
	stdout, stderr, code := runCLI(t, invocation{
		config: "zcode:\n  - name: default\n",
		args:   []string{"--json"},
		argLog: argLog, envLog: envLog,
		env: []string{
			"ZCODE_API_KEY=secret-zcode-key",
			"NODE_OPTIONS=--inspect=0.0.0.0:9229",
		},
	})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	rows := decodeUsageJSON(t, stdout)
	if len(rows) != 1 ||
		rows[0].TargetID != "zcode:0" || rows[0].Provider != "zcode" || rows[0].Account != "default" ||
		rows[0].Status != "ok" || rows[0].UsedPercent == nil || *rows[0].UsedPercent != 18 || rows[0].CleanupFailed {
		t.Fatalf("zcode JSON rows = %+v", rows)
	}
	args, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(args)) != "argv: doctor --json" {
		t.Fatalf("unexpected zcode argv log: %q", args)
	}
	env, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(env), "ZCODE_API_KEY=") ||
		strings.Contains(string(env), "--inspect=") ||
		!strings.Contains(string(env), "\nNODE_OPTIONS=--require=") {
		t.Fatalf("zcode child env was not filtered: %q", env)
	}

	tableOut, tableErr, tableCode := runCLI(t, invocation{
		config: "cursor:\n  - name: default\nzcode:\n  - name: default\n",
	})
	if tableCode != 0 || tableErr != "" {
		t.Fatalf("table code=%d stderr=%q", tableCode, tableErr)
	}
	if !strings.Contains(tableOut, "WEEKLY USAGE\n") || !strings.Contains(tableOut, "\n\nAI PROVIDER  ACCOUNT  5H USAGE\nZCode        default  18%\n") {
		t.Fatalf("unexpected two-section table:\n%s", tableOut)
	}
})

var _ = ginkgo.It("preserves configured top-level provider key order across all providers in table and JSON", func() {
	t := testT{ginkgo.GinkgoT()}
	config := `cursor:
  - name: cursor-main
agy:
  - name: default
claude:
  - name: claude-main
    path: ~/claude
grok:
  - name: grok-main
    path: ~/grok
codex:
  - name: e2e-a
    path: ~/e2e-a
  - name: e2e-b
    path: ~/e2e-b
`
	jsonOut, jsonErr, jsonCode := runCLI(t, invocation{
		config: config,
		args:   []string{"--json"},
		codex:  codexOK("account1@example.com", 37),
		grok:   grokWeekly(28),
	})
	if jsonCode != 0 || jsonErr != "" {
		t.Fatalf("json code=%d stderr=%q", jsonCode, jsonErr)
	}
	rows := decodeUsageJSON(t, jsonOut)
	wantRows := []struct{ id, provider, account string }{
		{"cursor:0", "cursor", "cursor-main"},
		{"agy:0", "agy", "gemini"},
		{"agy:1", "agy", "claude/gpt"},
		{"claude:0", "claude", "claude-main"},
		{"grok:0", "grok", "grok-main"},
		{"codex:0", "codex", "e2e-a"},
		{"codex:1", "codex", "e2e-b"},
	}
	if len(rows) != len(wantRows) {
		t.Fatalf("rows = %+v, want %d rows", rows, len(wantRows))
	}
	for i, want := range wantRows {
		if rows[i].TargetID != want.id || rows[i].Provider != want.provider || rows[i].Account != want.account || rows[i].Status != "ok" {
			t.Fatalf("row %d = %+v, want %+v", i, rows[i], want)
		}
	}

	tableOut, tableErr, tableCode := runCLI(t, invocation{
		config: config,
		codex:  codexOK("account1@example.com", 37),
		grok:   grokWeekly(28),
	})
	if tableCode != 0 || tableErr != "" {
		t.Fatalf("table code=%d stderr=%q", tableCode, tableErr)
	}
	lines := strings.Split(strings.TrimSuffix(tableOut, "\n"), "\n")
	wantLines := []struct{ provider, account, pct string }{
		{"Cursor", "cursor-main", "25%"},
		{"agy", "gemini", "74.2%"},
		{"agy", "claude/gpt", "12.8%"},
		{"Claude", "claude-main", "43%"},
		{"Grok", "grok-main", "28%"},
		{"ChatGPT", "e2e-a", "37%"},
		{"ChatGPT", "e2e-b", "37%"},
	}
	if len(lines) != len(wantLines)+1 {
		t.Fatalf("table lines = %d, want %d:\n%s", len(lines), len(wantLines)+1, tableOut)
	}
	for i, want := range wantLines {
		line := lines[i+1]
		if !strings.HasPrefix(line, want.provider+" ") || !strings.Contains(line, want.account) || !strings.HasSuffix(line, want.pct) {
			t.Fatalf("table line %d = %q, want %+v", i+1, line, want)
		}
	}
})

var _ = ginkgo.It("TestNoForbiddenProductionCapabilities", func() {
	t := testT{ginkgo.GinkgoT()}
	var failures []string
	fail := func(format string, args ...any) { failures = append(failures, fmt.Sprintf(format, args...)) }
	roots := []string{"../../internal", "../../cmd"}
	forbidden := []string{
		"net/http",    // no HTTP billing client or listener
		"net.Listen",  // no application service
		"auth.json",   // no credential file parsing
		"keychain",    // no keychain access
		"os.Setenv",   // no parent-environment mutation in production
		"pkill",       // no process spraying
		"sh -c",       // no shell wrappers
		"session/new", // no agent sessions (allowlists own these strings only in rpc codec)
	}
	for _, root := range roots {
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			src := string(b)
			for _, marker := range forbidden {
				if marker == "session/new" && strings.Contains(path, "internal/rpc/") {
					continue // the allowlist must name forbidden methods to reject them
				}
				if strings.Contains(src, marker) {
					fail("%s references forbidden %q", path, marker)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%s", strings.Join(failures, "\n"))
	}
})

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			return true
		}
	}
	return false
}

func intPtr(i int) *int { return &i }
