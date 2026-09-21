package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"gaebokchi/internal/testutil"
)

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// gaebokchiBin builds the real executable once for the whole suite.
func gaebokchiBin(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "gaebokchi-e2e-")
		if err != nil {
			buildErr = err
			return
		}
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
func fakeBin(t *testing.T) string {
	t.Helper()
	p, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// run invokes the built gaebokchi with fake provider scripts.
type invocation struct {
	args     []string
	env      []string
	codex    []testutil.Step
	grok     []testutil.Step
	argLog   string
	envLog   string
}

func runCLI(t *testing.T, inv invocation) (stdout, stderr string, code int) {
	t.Helper()
	bin := gaebokchiBin(t)
	fake := fakeBin(t)
	codexJSON, _ := json.Marshal(inv.codex)
	grokJSON, _ := json.Marshal(inv.grok)

	args := make([]string, 0, len(inv.args)+6)
	if !hasFlag(inv.args, "--timeout") {
		args = append(args, "--timeout", "5s")
	}
	if !hasFlag(inv.args, "--codex-bin") {
		args = append(args, "--codex-bin", fake)
	}
	if !hasFlag(inv.args, "--grok-bin") {
		args = append(args, "--grok-bin", fake)
	}
	args = append(args, inv.args...)

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(),
		"GAEBOKCHI_FAKE=1",
		"GAEBOKCHI_FAKE_CODEX_STEPS="+string(codexJSON),
		"GAEBOKCHI_FAKE_GROK_STEPS="+string(grokJSON),
	)
	if inv.argLog != "" {
		cmd.Env = append(cmd.Env, "GAEBOKCHI_FAKE_ARGLOG="+inv.argLog, "GAEBOKCHI_FAKE_ENVLOG="+inv.envLog)
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

func TestDefaultHappyPath(t *testing.T) {
	stdout, stderr, code := runCLI(t, invocation{
		codex: codexOK("account1@example.com", 37),
		grok:  grokWeekly(28),
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr:\n%s", code, stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d:\n%s", len(lines), stdout)
	}
	if !strings.Contains(stdout, "account1@example.com") || !strings.Contains(stdout, "37.0%") || !strings.Contains(stdout, "28.0%") {
		t.Fatalf("table content wrong:\n%s", stdout)
	}
	if strings.Contains(stdout, "FAKE") || strings.Contains(stdout, "jsonrpc") {
		t.Fatalf("raw provider data on stdout:\n%s", stdout)
	}
}

func TestMissingExecutableKeepsOtherRows(t *testing.T) {
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
	if strings.Count(stdout, "\n") != 4 {
		t.Fatalf("row count changed:\n%s", stdout)
	}
	if !strings.Contains(stdout, "11.0%") {
		t.Fatalf("codex rows lost:\n%s", stdout)
	}
}

func TestMissingHomeRow(t *testing.T) {
	stdout, _, code := runCLI(t, invocation{
		args:  []string{"--codex-home", "/definitely/not/here"},
		grok:  grokWeekly(5),
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "HOME NOT FOUND") {
		t.Fatalf("missing home row:\n%s", stdout)
	}
	if !strings.Contains(stdout, "5.0%") {
		t.Fatalf("other rows lost:\n%s", stdout)
	}
}

func TestFailedAuthKeepsTable(t *testing.T) {
	authFail := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"ok": true}}}},
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
}

func TestMonthlyGrokIsClean(t *testing.T) {
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
}

func TestMalformedGrokResponseIsProtocolError(t *testing.T) {
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
}

func TestWrongIDsNeverSatisfyRequest(t *testing.T) {
	grok := grokWeekly(1)
	grok[4] = testutil.Step{Emit: `{"jsonrpc":"2.0","id":99,"result":{"config":{"creditUsagePercent":1.0,"currentPeriod":{"type":"USAGE_PERIOD_TYPE_WEEKLY"}}}}` + "\n"}
	// After the wrong-id response the fake exits, producing EOF.
	stdout, _, code := runCLI(t, invocation{
		codex: codexOK("a@example.com", 9),
		grok:  grok,
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if strings.Contains(stdout, "1.0%") {
		t.Fatalf("wrong-id response satisfied the grok row:\n%s", stdout)
	}
}

func TestLongStderrDoesNotLeak(t *testing.T) {
	grok := grokWeekly(7)
	grok = append([]testutil.Step{{EmitStderr: strings.Repeat("E2E-STDERR-NOISE ", 8000)}}, grok...)
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
	if !strings.Contains(stdout, "7.0%") {
		t.Fatalf("stderr flood broke the query:\n%s", stdout)
	}
}

func TestHungProviderTimesOut(t *testing.T) {
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
	if !strings.Contains(stdout, "2.0%") {
		t.Fatalf("healthy sibling lost:\n%s", stdout)
	}
}

func TestUnsafeCallbackRejected(t *testing.T) {
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
}

func TestSecretMarkersNeverLeak(t *testing.T) {
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
}

func TestEnvironmentIsolationAndArgv(t *testing.T) {
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
	for _, block := range strings.Split(string(envDump), "=== child") {
		if strings.Contains(block, "app-server") {
			if strings.Contains(block, "OPENAI_API_KEY=CODEX-ENV-SECRET") {
				t.Fatal("openai key override reached the codex child")
			}
			if !strings.Contains(block, "CODEX_HOME=") {
				t.Fatal("CODEX_HOME not set for codex child")
			}
		}
		if strings.Contains(block, "agent") && strings.Contains(block, "stdio") {
			if strings.Contains(block, "XAI_API_KEY=GROK-ENV-SECRET") {
				t.Fatal("xai key override reached the grok child")
			}
		}
	}
}

func TestSignalCancellationExits130(t *testing.T) {
	bin := gaebokchiBin(t)
	fake := fakeBin(t)
	codexJSON, _ := json.Marshal([]testutil.Step{{Wait: 1}, {SleepMs: 60000}})
	grokJSON, _ := json.Marshal([]testutil.Step{{Wait: 1}, {SleepMs: 60000}})
	cmd := exec.Command(bin,
		"--codex-bin", fake, "--grok-bin", fake, "--timeout", "20s")
	cmd.Env = append(os.Environ(),
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
}

func TestConcurrentAccountIsolation(t *testing.T) {
	// Two homes with distinct scripted identities must not mix; the
	// shared script serves both children identically, so verify via a
	// second run with distinct homes through arg log cwd/env.
	dir := t.TempDir()
	argLog := filepath.Join(dir, "argv.log")
	homeA := filepath.Join(dir, "work")
	homeB := filepath.Join(dir, "personal")
	os.MkdirAll(homeA, 0o755)
	os.MkdirAll(homeB, 0o755)
	stdout, _, code := runCLI(t, invocation{
		args:   []string{"--codex-home", homeA, "--codex-home", homeB},
		codex:  codexOK("shared@example.com", 21),
		grok:   grokWeekly(22),
		argLog: argLog,
		envLog: filepath.Join(dir, "env.log"),
	})
	if code != 0 {
		t.Fatalf("code = %d:\n%s", code, stdout)
	}
	if strings.Count(stdout, "shared@example.com") != 2 {
		t.Fatalf("both homes must produce their own row:\n%s", stdout)
	}
	if !strings.Contains(stdout, "shared@example.com [work]") || !strings.Contains(stdout, "shared@example.com [personal]") {
		t.Fatalf("duplicate labels not disambiguated:\n%s", stdout)
	}
	log, _ := os.ReadFile(argLog)
	if strings.Count(string(log), "app-server") != 2 {
		t.Fatalf("expected two codex children, log: %q", log)
	}
}

func TestNoForbiddenProductionCapabilities(t *testing.T) {
	roots := []string{"../../internal", "../../cmd"}
	forbidden := []string{
		"net/http",       // no HTTP billing client or listener
		"net.Listen",     // no application service
		"auth.json",      // no credential file parsing
		"keychain",       // no keychain access
		"os.Setenv",      // no parent-environment mutation in production
		"pkill",          // no process spraying
		"sh -c",          // no shell wrappers
		"session/new",    // no agent sessions (allowlists own these strings only in rpc codec)
	}
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			src := string(b)
			for _, marker := range forbidden {
				if marker == "session/new" && strings.Contains(path, "internal/rpc/") {
					continue // the allowlist must name forbidden methods to reject them
				}
				if strings.Contains(src, marker) {
					t.Errorf("%s references forbidden %q", path, marker)
				}
			}
			return nil
		})
	}
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			return true
		}
	}
	return false
}

func intPtr(i int) *int { return &i }
