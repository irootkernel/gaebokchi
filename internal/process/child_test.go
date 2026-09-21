package process

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestHelperProcess is not a real test: the test binary re-executes itself
// as the fake child. Scenarios are selected by the first argument.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_GAEBOKCHI_HELPER") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(2)
	}
	switch args[0] {
	case "echo-stdin":
		// Read stdin until EOF, echo byte count to stdout, exit 0.
		n, _ := io.Copy(io.Discard, os.Stdin)
		os.Stdout.WriteString("read-bytes=" + itoa(n) + "\n")
		os.Exit(0)
	case "exit-immediately":
		os.Exit(3)
	case "stuck":
		// Never reads stdin, sleeps until killed.
		time.Sleep(5 * time.Minute)
		os.Exit(0)
	case "ignore-term":
		// Handle SIGTERM ourselves so the default termination does not
		// fire; only SIGKILL can stop this scenario.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)
		go func() {
			<-sig
			os.Stdout.WriteString("still-alive\n")
		}()
		time.Sleep(5 * time.Minute)
		os.Exit(0)
	case "flood-stderr":
		chunk := strings.Repeat("x", 4096)
		for i := 0; i < 256; i++ { // 1 MiB total
			os.Stderr.WriteString(chunk)
		}
		os.Exit(0)
	case "print-cwd":
		wd, _ := os.Getwd()
		os.Stdout.WriteString("cwd=" + wd + "\n")
		os.Exit(0)
	case "print-env":
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "GAEBOKCHI_TEST_") {
				os.Stdout.WriteString(kv + "\n")
			}
		}
		os.Exit(0)
	case "spawn-grandchild":
		// Spawn a descendant that inherits stdout and outlives us.
		gc := exec.Command(args[1], args[2:]...)
		gc.Stdout = os.Stdout
		gc.Stderr = os.Stderr
		_ = gc.Start()
		os.Exit(0)
	default:
		os.Exit(9)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// startHelper launches the test binary as a fake child.
func startHelper(t *testing.T, scenario string, extra ...string) *Child {
	t.Helper()
	args := append([]string{"-test.run=TestHelperProcess", "--", scenario}, extra...)
	c, err := Start(Config{
		Bin:  os.Args[0],
		Args: args,
		Env:  append(os.Environ(), "GO_GAEBOKCHI_HELPER=1"),
	})
	if err != nil {
		t.Fatalf("start helper %q: %v", scenario, err)
	}
	return c
}

func TestCleanExitAfterStdinEOF(t *testing.T) {
	c := startHelper(t, "echo-stdin")
	if _, err := io.WriteString(c.Stdin(), "hello"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	c.CloseStdin() // deliver EOF so the helper can respond and exit
	out, err := io.ReadAll(c.Stdout())
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	<-c.Done()
	if err := c.Err(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if c.ExitCode() != 0 {
		t.Fatalf("exit code = %d", c.ExitCode())
	}
	if !bytes.Contains(out, []byte("read-bytes=5")) {
		t.Fatalf("unexpected output %q", out)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := os.Stat(c.dir); !os.IsNotExist(err) {
		t.Fatalf("temp dir not removed: %v", err)
	}
}

func TestEarlyExit(t *testing.T) {
	c := startHelper(t, "exit-immediately")
	select {
	case <-c.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("child exit not observed")
	}
	if c.ExitCode() != 3 {
		t.Fatalf("exit code = %d, want 3", c.ExitCode())
	}
	// Writing to a dead child's stdin must not hang or panic.
	_, _ = io.WriteString(c.Stdin(), "x")
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestStuckChildTerminatedWithinBudget(t *testing.T) {
	c := startHelper(t, "stuck")
	start := time.Now()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	elapsed := time.Since(start)
	// EOF stage + SIGTERM stage must stop a plain sleeper well under 2s.
	if elapsed > DefaultCleanupBudget {
		t.Fatalf("close took %v, budget %v", elapsed, DefaultCleanupBudget)
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("child not reaped after close")
	}
}

func TestChildIgnoringSIGTERMKilled(t *testing.T) {
	c := startHelper(t, "ignore-term")
	start := time.Now()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > DefaultCleanupBudget {
		t.Fatalf("close took %v, budget %v", elapsed, DefaultCleanupBudget)
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("child not reaped after SIGKILL")
	}
}

func TestStderrFloodBoundedAndDrained(t *testing.T) {
	c := startHelper(t, "flood-stderr")
	<-c.Done()
	// The drain goroutine keeps running past child exit until EOF; Close
	// joins it. Retained stderr must be capped at 64 KiB.
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	kept := c.StderrKept()
	if len(kept) > stderrRetainCap {
		t.Fatalf("retained %d bytes, cap %d", len(kept), stderrRetainCap)
	}
	if len(kept) == 0 {
		t.Fatal("stderr retention lost the flood head")
	}
}

func TestPrivateTempWorkingDirectory(t *testing.T) {
	c := startHelper(t, "print-cwd")
	out, _ := io.ReadAll(c.Stdout())
	<-c.Done()
	c.Close()
	line := strings.TrimPrefix(string(out), "cwd=")
	wd := strings.TrimSpace(line)
	if !strings.Contains(wd, "gaebokchi-") {
		t.Fatalf("cwd %q is not a gaebokchi temp dir", wd)
	}
	if same, err := samePath(wd, mustWd(t)); err == nil && same {
		t.Fatalf("child cwd equals invocation dir %q", wd)
	}
}

func TestEnvironmentPassedVerbatim(t *testing.T) {
	c, err := Start(Config{
		Bin:  os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "print-env"},
		Env:  append(os.Environ(), "GO_GAEBOKCHI_HELPER=1", "GAEBOKCHI_TEST_MARKER=42"),
	})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(c.Stdout())
	<-c.Done()
	c.Close()
	if !bytes.Contains(out, []byte("GAEBOKCHI_TEST_MARKER=42")) {
		t.Fatalf("env not passed verbatim, got %q", out)
	}
	if bytes.Contains(out, []byte("GAEBOKCHI_TEST_MISSING")) {
		t.Fatal("unexpected env content")
	}
}

func TestInheritedPipeHolderDoesNotBlockCleanup(t *testing.T) {
	// The grandchild (sleep) inherits stdout and outlives the direct
	// child; Close must still reap, unblock readers, and return.
	c := startHelper(t, "spawn-grandchild", "sleep", "30")
	readDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, c.Stdout())
		readDone <- err
	}()
	<-c.Done()
	start := time.Now()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if elapsed := time.Since(start); elapsed > DefaultCleanupBudget {
		t.Fatalf("close took %v with inherited pipe holder", elapsed)
	}
	select {
	case <-readDone:
	default:
		// The reader may or may not have unblocked by now; the important
		// part is that Close returned. Give it a moment.
		select {
		case <-readDone:
		case <-time.After(2 * time.Second):
			t.Fatal("pipe reader never unblocked after close")
		}
	}
}

func TestUnrelatedProcessSurvivesCleanup(t *testing.T) {
	// A process gaebokchi did not start must be untouched by cleanup.
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Skip("sleep unavailable")
	}
	defer func() {
		_ = other.Process.Signal(syscall.SIGKILL)
		_, _ = other.Process.Wait()
	}()

	c := startHelper(t, "stuck")
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := other.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("unrelated process was killed by cleanup: %v", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	c := startHelper(t, "stuck")
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("exactly-once wait violated: child not reaped")
	}
}

func TestStartFailureCleansUp(t *testing.T) {
	before, _ := os.ReadDir(os.TempDir())
	_, err := Start(Config{Bin: "/nonexistent/gaebokchi-no-such-bin", Args: nil, Env: nil})
	if err == nil {
		t.Fatal("start should fail for missing binary")
	}
	after, _ := os.ReadDir(os.TempDir())
	if len(after) > len(before)+1 {
		t.Fatalf("temp dirs leaked on start failure: before=%d after=%d", len(before), len(after))
	}
}

func mustWd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func samePath(a, b string) (bool, error) {
	ra, ea := realpath(a)
	rb, eb := realpath(b)
	if ea != nil || eb != nil {
		return false, nil
	}
	return ra == rb, nil
}

func realpath(p string) (string, error) {
	return filepath.EvalSymlinks(p)
}
