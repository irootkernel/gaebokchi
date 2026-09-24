package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gaebokchi/internal/testutil"
)

// startWithFakes launches the built gaebokchi with the standard fake
// wiring but caller-provided stdout/stderr, for real-OS-pipe and signal
// scenarios that need control over the process's actual file
// descriptors.
func startWithFakes(t *testing.T, extraArgs []string, codexSteps, grokSteps []testutil.Step, stdout, stderr io.Writer) *exec.Cmd {
	t.Helper()
	bin := gaebokchiBin(t)
	fake := fakeBin(t)
	homeDir := t.TempDir()
	homeA := filepath.Join(homeDir, "e2e-a")
	homeB := filepath.Join(homeDir, "e2e-b")
	for _, h := range []string{homeA, homeB} {
		if err := os.MkdirAll(h, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	codexJSON, _ := json.Marshal(codexSteps)
	grokJSON, _ := json.Marshal(grokSteps)
	args := append([]string{
		"--timeout", "20s",
		"--codex-bin", fake, "--grok-bin", fake, "--claude-bin", fake,
		"--codex-home", homeA, "--codex-home", homeB,
	}, extraArgs...)
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(),
		"GAEBOKCHI_FAKE=1",
		"GAEBOKCHI_FAKE_CODEX_STEPS="+string(codexJSON),
		"GAEBOKCHI_FAKE_GROK_STEPS="+string(grokJSON),
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

// waitExitCode waits for the command and returns its exit code,
// requiring a normal exit (no signal death).
func waitExitCode(t *testing.T, cmd *exec.Cmd, waitErr error) int {
	t.Helper()
	if waitErr == nil {
		t.Fatal("expected a nonzero exit")
	}
	ee, ok := waitErr.(*exec.ExitError)
	if !ok {
		t.Fatal(waitErr)
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Exited() {
		t.Fatalf("process died of a signal instead of exiting: %v", ws)
	}
	return ws.ExitStatus()
}

// brokenStdoutPipe returns a pipe write end whose reader is already
// closed: every write raises EPIPE and (for fd 1) SIGPIPE.
func brokenStdoutPipe(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return w
}

// TestClosedStdoutPipeExitsOneNotSignalled pins the SIGPIPE policy
// against the audit finding: a closed stdout reader is the documented
// output failure (exit 1), never SIGPIPE signal death.
func TestClosedStdoutPipeExitsOneNotSignalled(t *testing.T) {
	w := brokenStdoutPipe(t)
	var errBuf bytes.Buffer
	cmd := startWithFakes(t, nil, codexOK("a@example.com", 40), grokWeekly(41), w, &errBuf)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	code := waitExitCode(t, cmd, cmd.Wait())
	if code != 1 {
		t.Fatalf("exit = %d, want 1 for output failure", code)
	}
	if !strings.Contains(errBuf.String(), "cannot write the usage table to stdout") {
		t.Fatalf("safe diagnostic missing:\n%s", errBuf.String())
	}
}

// TestSignalWithBrokenStdoutKeepsSignalCode pins the precedence the
// audit found inverted: the first handled cancellation signal (130 for
// SIGINT, 143 for SIGTERM) outranks the output failure that follows.
func TestSignalWithBrokenStdoutKeepsSignalCode(t *testing.T) {
	hang := []testutil.Step{{Wait: 1}, {SleepMs: 60000}}
	cases := []struct {
		name string
		sig  syscall.Signal
		want int
	}{
		{"SIGINT", syscall.SIGINT, 130},
		{"SIGTERM", syscall.SIGTERM, 143},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := brokenStdoutPipe(t)
			var errBuf bytes.Buffer
			cmd := startWithFakes(t, nil, hang, hang, w, &errBuf)
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			time.Sleep(700 * time.Millisecond) // children admitted and hanging
			if err := cmd.Process.Signal(tc.sig); err != nil {
				t.Fatal(err)
			}
			code := waitExitCode(t, cmd, cmd.Wait())
			if code != tc.want {
				t.Fatalf("exit = %d, want %d (signal outranks output failure)", code, tc.want)
			}
		})
	}
}

// TestSignalMixedCompletionKeepsNumericRow exercises the full signal
// contract end to end: a completed target stays numeric, in-flight
// targets render CANCELLED, the row count is unchanged, and the exit
// code is the signal's.
func TestSignalMixedCompletionKeepsNumericRow(t *testing.T) {
	// Codex children complete the handshake and account read, then hang
	// before the quota read; grok answers fully and quickly.
	codexHangBeforeQuota := []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: `{"id":2,"result":{"account":{"type":"chatgpt","email":"slow@example.com","planType":"pro"}}}` + "\n"},
		{Wait: 1},
		{SleepMs: 60000},
	}
	var out, errBuf bytes.Buffer
	cmd := startWithFakes(t, nil, codexHangBeforeQuota, grokWeekly(33), &out, &errBuf)
	time.Sleep(800 * time.Millisecond) // grok's numeric row is established
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	code := waitExitCode(t, cmd, cmd.Wait())
	if code != 130 {
		t.Fatalf("exit = %d, want 130", code)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "33.0%") {
		t.Fatalf("established grok row lost on cancellation:\n%s", stdout)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "ChatGPT") && !strings.Contains(line, "CANCELLED") {
			t.Fatalf("in-flight codex row must render CANCELLED:\n%s", stdout)
		}
	}
	if lines := strings.Count(strings.TrimSuffix(stdout, "\n"), "\n") + 1; lines != 5 {
		t.Fatalf("row count changed: %d lines:\n%s", lines, stdout)
	}
}

// TestHelpVersionBrokenPipeExitsOne covers output failures on the
// help/version paths, which never launch providers.
func TestHelpVersionBrokenPipeExitsOne(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"version"}} {
		w := brokenStdoutPipe(t)
		var errBuf bytes.Buffer
		cmd := exec.Command(gaebokchiBin(t), args...)
		cmd.Stdout = w
		cmd.Stderr = &errBuf
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		code := waitExitCode(t, cmd, cmd.Wait())
		if code != 1 {
			t.Fatalf("%v: exit = %d, want 1 for output failure", args, code)
		}
	}
}

// TestRelativeBinaryOverrideFromInvocationCwd pins the audit finding
// that an explicit relative --codex-bin used to become NOT INSTALLED
// because the child runs from its private directory: the resolver must
// absolutize against the invocation cwd, including names with spaces.
func TestRelativeBinaryOverrideFromInvocationCwd(t *testing.T) {
	dir := t.TempDir()
	fake := fakeBin(t)
	script := filepath.Join(dir, "my codex") // includes a space
	body := "#!/bin/sh\nexec \"$GAEBOKCHI_E2E_FAKE_BIN\" app-server\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	homeDir := t.TempDir()
	homeA := filepath.Join(homeDir, "e2e-a")
	homeB := filepath.Join(homeDir, "e2e-b")
	for _, h := range []string{homeA, homeB} {
		if err := os.MkdirAll(h, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	codexJSON, _ := json.Marshal(codexOK("rel@example.com", 44))
	grokJSON, _ := json.Marshal(grokWeekly(45))
	cmd := exec.Command(gaebokchiBin(t),
		"--timeout", "5s",
		"--codex-bin", "./my codex",
		"--grok-bin", fake,
		"--claude-bin", fake,
		"--codex-home", homeA, "--codex-home", homeB,
	)
	cmd.Dir = dir // the invocation cwd contains the relative script
	cmd.Env = append(os.Environ(),
		"GAEBOKCHI_FAKE=1",
		"GAEBOKCHI_FAKE_CODEX_STEPS="+string(codexJSON),
		"GAEBOKCHI_FAKE_GROK_STEPS="+string(grokJSON),
		"GAEBOKCHI_E2E_FAKE_BIN="+fake,
	)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("code = %d, stderr:\n%s", code, errBuf.String())
	}
	if strings.Contains(out.String(), "NOT INSTALLED") {
		t.Fatalf("relative binary override failed to launch:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "44.0%") {
		t.Fatalf("relative binary did not complete its query:\n%s", out.String())
	}
}
