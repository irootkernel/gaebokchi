package process

import (
	"bytes"
	"errors"
	ginkgo "github.com/onsi/ginkgo/v2"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// runHelperProcess handles test-binary re-execution as a fake child.
// Scenarios are selected by the first argument after --.
func runHelperProcess() {
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
		writeHelper(os.Stdout, "read-bytes="+itoa(n)+"\n")
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
			writeHelper(os.Stdout, "still-alive\n")
		}()
		time.Sleep(5 * time.Minute)
		os.Exit(0)
	case "flood-stderr":
		chunk := strings.Repeat("x", 4096)
		for i := 0; i < 256; i++ { // 1 MiB total
			writeHelper(os.Stderr, chunk)
		}
		os.Exit(0)
	case "print-cwd":
		wd, _ := os.Getwd()
		writeHelper(os.Stdout, "cwd="+wd+"\n")
		os.Exit(0)
	case "print-env":
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "GAEBOKCHI_TEST_") {
				writeHelper(os.Stdout, kv+"\n")
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
	case "spawn-grandchild-pid":
		// Like spawn-grandchild, but report the descendant's pid first.
		gc := exec.Command(args[1], args[2:]...)
		gc.Stdout = os.Stdout
		gc.Stderr = os.Stderr
		_ = gc.Start()
		writeHelper(os.Stdout, "grandchild-pid="+itoa(int64(gc.Process.Pid))+"\n")
		os.Exit(0)
	case "ignore-term-grandchild":
		// Ignores SIGTERM and keeps a descendant alive in the private
		// group: only the stage-3 group kill can clean both up.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)
		go func() {
			<-sig
			writeHelper(os.Stdout, "still-alive\n")
		}()
		gc := exec.Command(args[1], args[2:]...)
		gc.Stdout = os.Stdout
		gc.Stderr = os.Stderr
		_ = gc.Start()
		writeHelper(os.Stdout, "grandchild-pid="+itoa(int64(gc.Process.Pid))+"\n")
		time.Sleep(5 * time.Minute)
		os.Exit(0)
	default:
		os.Exit(9)
	}
}

func writeHelper(out *os.File, s string) {
	if _, err := out.WriteString(s); err != nil {
		os.Exit(1)
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
func startHelper(t testT, scenario string, extra ...string) *Child {
	t.Helper()
	args := append([]string{"--", scenario}, extra...)
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

var _ = ginkgo.It("TestCleanExitAfterStdinEOF", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestEarlyExit", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestStuckChildTerminatedWithinBudget", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestChildIgnoringSIGTERMKilled", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestStderrFloodBoundedAndDrained", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestPrivateTempWorkingDirectory", func() {
	t := testT{ginkgo.GinkgoT()}
	c := startHelper(t, "print-cwd")
	out, _ := io.ReadAll(c.Stdout())
	<-c.Done()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	line := strings.TrimPrefix(string(out), "cwd=")
	wd := strings.TrimSpace(line)
	if !strings.Contains(wd, "gaebokchi-") {
		t.Fatalf("cwd %q is not a gaebokchi temp dir", wd)
	}
	if same, err := samePath(wd, mustWd(t)); err == nil && same {
		t.Fatalf("child cwd equals invocation dir %q", wd)
	}
})

var _ = ginkgo.It("TestEnvironmentPassedVerbatim", func() {
	t := testT{ginkgo.GinkgoT()}
	c, err := Start(Config{
		Bin:  os.Args[0],
		Args: []string{"--", "print-env"},
		Env:  append(os.Environ(), "GO_GAEBOKCHI_HELPER=1", "GAEBOKCHI_TEST_MARKER=42"),
	})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(c.Stdout())
	<-c.Done()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("GAEBOKCHI_TEST_MARKER=42")) {
		t.Fatalf("env not passed verbatim, got %q", out)
	}
	if bytes.Contains(out, []byte("GAEBOKCHI_TEST_MISSING")) {
		t.Fatal("unexpected env content")
	}
})

var _ = ginkgo.It("TestInheritedPipeHolderDoesNotBlockCleanup", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestUnrelatedProcessSurvivesCleanup", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestCloseIsIdempotent", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

// TestCloseHonorsSingleAbsoluteBudgetEndToEnd measures the whole Close
// against the one absolute cleanup budget (EOF + SIGTERM + SIGKILL +
// drain + temp-dir removal), not a sum of stage timers: the audit found
// finish granting itself a fresh full budget after shutdown.
var _ = ginkgo.It("TestCloseHonorsSingleAbsoluteBudgetEndToEnd", func() {
	t := testT{ginkgo.GinkgoT()}
	c := startHelper(t, "ignore-term")
	start := time.Now()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > DefaultCleanupBudget+750*time.Millisecond {
		t.Fatalf("close took %v end-to-end; single budget is %v", elapsed, DefaultCleanupBudget)
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("child not reaped after close")
	}
})

var _ = ginkgo.It("TestSlowDirectoryRemovalReportsExpiredCleanupBudget", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := filepath.Join(t.TempDir(), "delayed-removal")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	finished := make(chan error, 1)
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		<-finished
	}()
	start := time.Now()
	err := removeDirBy(start.Add(100*time.Millisecond), dir, func(path string) error {
		<-release
		removeErr := os.RemoveAll(path)
		finished <- removeErr
		return removeErr
	})
	if !errors.Is(err, ErrCleanupIncomplete) {
		t.Fatalf("slow removal reported %v, want incomplete cleanup", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("slow removal blocked Close for %v", elapsed)
	}
})

// readGrandchildPID reads the helper's reported descendant pid without
// waiting for EOF (the descendant inherits stdout, so EOF only arrives
// when it exits).
func readGrandchildPID(t testT, c *Child) int {
	t.Helper()
	pidLine := make(chan string, 1)
	go func() {
		var buf []byte
		one := make([]byte, 1)
		for {
			n, err := c.Stdout().Read(one)
			if n > 0 {
				if one[0] == '\n' {
					pidLine <- string(buf)
					return
				}
				buf = append(buf, one[0])
			}
			if err != nil {
				pidLine <- string(buf)
				return
			}
		}
	}()
	select {
	case line := <-pidLine:
		p := strings.TrimSpace(strings.TrimPrefix(line, "grandchild-pid="))
		v, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("grandchild pid line %q: %v", line, err)
		}
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("no grandchild pid reported")
		return 0
	}
}

// TestDescendantsPreservedWhenDirectChildExits pins the qualified
// ownership boundary: a descendant surviving the direct child's clean
// exit is preserved by cleanup. Group membership alone does not prove a
// descendant is disposable — a provider-managed shared leader can be a
// descendant — so stages 1 and 2 must not signal it.
var _ = ginkgo.It("TestDescendantsPreservedWhenDirectChildExits", func() {
	t := testT{ginkgo.GinkgoT()}
	c := startHelper(t, "spawn-grandchild-pid", "sleep", "30")
	pid := readGrandchildPID(t, c)

	<-c.Done()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("descendant killed by cleanup despite preserved boundary: %v", err)
	}
	// The descendant is the test's own helper process: end it explicitly
	// rather than leaving it to its 30s sleep.
	_ = syscall.Kill(pid, syscall.SIGKILL)
})

// TestStuckDescendantSurvivesForcedDirectKill covers the forced side of
// the ownership boundary: when the direct child we started refuses
// graceful termination, escalation kills and reaps only that direct
// child. The uncertain-ownership descendant survives even forced
// cleanup — direct-child unresponsiveness does not prove the group is
// disposable — while bounded cleanup still completes.
var _ = ginkgo.It("TestStuckDescendantSurvivesForcedDirectKill", func() {
	t := testT{ginkgo.GinkgoT()}
	c := startHelper(t, "ignore-term-grandchild", "sleep", "30")
	pid := readGrandchildPID(t, c)

	start := time.Now()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if elapsed := time.Since(start); elapsed > DefaultCleanupBudget+750*time.Millisecond {
		t.Fatalf("close took %v end-to-end; single budget is %v", elapsed, DefaultCleanupBudget)
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("direct child not reaped after forced kill")
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("uncertain-ownership descendant killed by forced cleanup: %v", err)
	}
	// The descendant is the test's own helper process: end it explicitly
	// rather than leaving it to its 30s sleep.
	_ = syscall.Kill(pid, syscall.SIGKILL)
})

var _ = ginkgo.It("TestStartFailureCleansUp", func() {
	t := testT{ginkgo.GinkgoT()}
	before, _ := os.ReadDir(os.TempDir())
	_, err := Start(Config{Bin: "/nonexistent/gaebokchi-no-such-bin", Args: nil, Env: nil})
	if err == nil {
		t.Fatal("start should fail for missing binary")
	}
	after, _ := os.ReadDir(os.TempDir())
	if len(after) > len(before)+1 {
		t.Fatalf("temp dirs leaked on start failure: before=%d after=%d", len(before), len(after))
	}
})

func mustWd(t testT) string {
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

// TestCloseReportsRemovalFailure reproduces a real cleanup failure with
// permissions: after the test-owned parent directory loses its write
// bit, the private directory cannot be unlinked, so Close must report
// the failure, keep reporting the same outcome on repeated calls, and
// still reap the child.
var _ = ginkgo.It("TestCloseReportsRemovalFailure", func() {
	t := testT{ginkgo.GinkgoT()}
	if os.Getuid() == 0 {
		t.Skip("permission-based removal failure does not apply as root")
	}
	parent := t.TempDir()
	c, err := Start(Config{
		Bin:        os.Args[0],
		Args:       []string{"--", "stuck"},
		Env:        append(os.Environ(), "GO_GAEBOKCHI_HELPER=1"),
		TempParent: parent,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only the test's own directory is touched; the permission is
	// restored before the temp-dir cleanup runs.
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Skipf("cannot chmod temp parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	first := c.Close()
	if first == nil {
		t.Fatal("cleanup failure not reported")
	}
	second := c.Close()
	if second != first {
		t.Fatalf("repeated close changed the outcome: %v vs %v", second, first)
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("child not reaped despite incomplete removal")
	}
})
