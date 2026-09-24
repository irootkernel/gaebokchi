// Package process owns the lifecycle of provider child processes: start
// with explicit executable and arguments, pipe plumbing, exactly-once
// wait/reap, and bounded EOF/SIGTERM/SIGKILL cleanup. It never uses a
// shell and never touches processes it did not start.
package process

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Cleanup stage and budget constants from ARCH-05.
const (
	// DefaultCleanupBudget bounds the whole cleanup sequence.
	DefaultCleanupBudget = 2 * time.Second
	// eofStage and termStage bound the graceful stages before SIGKILL.
	eofStage  = 500 * time.Millisecond
	termStage = 500 * time.Millisecond
	// stderrRetainCap bounds in-memory stderr retention.
	stderrRetainCap = 64 << 10
)

// ErrNotStarted is returned by Close for children that never started.
var ErrNotStarted = errors.New("process: child not started")

// ErrCleanupIncomplete reports that cleanup could not finish within its
// single absolute budget (drain not joined or child not reaped). The
// child, pipes, and temporary directory were still handled best-effort.
var ErrCleanupIncomplete = errors.New("process: cleanup incomplete within budget")

// Config describes one owned child. Env is used verbatim: callers apply
// provider environment policy before calling Start.
type Config struct {
	Bin  string
	Args []string
	Env  []string
	// CleanupBudget defaults to DefaultCleanupBudget.
	CleanupBudget time.Duration
	// TempParent overrides the parent directory of the child's private
	// working directory; empty uses the system default. It exists as a
	// narrow, documented test seam so tests that own the parent can
	// deterministically reproduce cleanup failures (for example by
	// removing the parent's write permission after start). Production
	// callers leave it empty.
	TempParent string
}

// Factory creates owned children; production uses Start. Tests inject
// fakes through this narrow seam.
type Factory func(Config) (*Child, error)

// Child is one owned provider process with its pipes and private
// temporary working directory.
type Child struct {
	cfg Config

	cmd *exec.Cmd

	stdinW  *os.File
	stdoutR *os.File
	stderrR *os.File

	dir string

	mu         sync.Mutex
	stderrKept []byte

	waitErr error
	done    chan struct{}

	drainDone chan struct{}

	closeOnce sync.Once
	closeErr  error
	stdinOnce sync.Once
}

// Start launches the child in a private mode-0700 temporary directory
// outside the caller's project tree. The parent's copies of the child
// ends of the pipes are closed immediately so EOF is delivered when the
// child (and any fd holders it spawned) exit.
func Start(cfg Config) (*Child, error) {
	if cfg.CleanupBudget <= 0 {
		cfg.CleanupBudget = DefaultCleanupBudget
	}
	dir, err := os.MkdirTemp(cfg.TempParent, "gaebokchi-")
	if err != nil {
		return nil, err
	}

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, cleanupStartFailure(err, dir)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, cleanupStartFailure(err, dir, stdinR, stdinW)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return nil, cleanupStartFailure(err, dir, stdinR, stdinW, stdoutR, stdoutW)
	}

	cmd := exec.Command(cfg.Bin, cfg.Args...)
	cmd.Dir = dir
	cmd.Env = cfg.Env
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	prepareSysProcAttr(cmd)

	c := &Child{
		cfg:       cfg,
		cmd:       cmd,
		stdinW:    stdinW,
		stdoutR:   stdoutR,
		stderrR:   stderrR,
		dir:       dir,
		done:      make(chan struct{}),
		drainDone: make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		return nil, cleanupStartFailure(err, dir, stdinR, stdinW, stdoutR, stdoutW, stderrR, stderrW)
	}
	// Drop the parent's copies of the child-side fds: EOF on our read
	// ends must depend only on the child's lifetime.
	// These descriptors are already owned by the running child; closing
	// them here is best effort and cannot undo a successful start.
	_ = stdinR.Close()
	_ = stdoutW.Close()
	_ = stderrW.Close()

	// Exactly one owner ever calls Wait: this goroutine.
	go func() {
		c.waitErr = cmd.Wait()
		close(c.done)
	}()
	go c.drainStderr()
	return c, nil
}

func cleanupStartFailure(startErr error, dir string, pipes ...io.Closer) error {
	err := startErr
	for _, pipe := range pipes {
		err = errors.Join(err, pipe.Close())
	}
	return errors.Join(err, os.RemoveAll(dir))
}

// Stdin is the write end of the child's stdin.
func (c *Child) Stdin() io.Writer { return c.stdinW }

// Stdout is the read end of the child's stdout.
func (c *Child) Stdout() io.Reader { return c.stdoutR }

// Done closes when the child has been reaped. Err then returns the wait
// result (nil for exit code 0).
func (c *Child) Done() <-chan struct{} { return c.done }

// Err returns the child's wait error; only meaningful after Done.
func (c *Child) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.waitErr
}

// StderrKept returns the retained head of the child's stderr, bounded to
// 64 KiB. Callers must sanitize before any diagnostic use; it is raw
// provider output.
func (c *Child) StderrKept() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]byte, len(c.stderrKept))
	copy(out, c.stderrKept)
	return out
}

// ExitCode returns the child's exit code, or -1 when unavailable.
func (c *Child) ExitCode() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.waitErr == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(c.waitErr, &ee) {
		return ee.ExitCode()
	}
	return -1
}

func (c *Child) drainStderr() {
	buf := make([]byte, 8192)
	for {
		n, err := c.stderrR.Read(buf)
		if n > 0 {
			c.mu.Lock()
			if room := stderrRetainCap - len(c.stderrKept); room > 0 {
				if n > room {
					n = room
				}
				c.stderrKept = append(c.stderrKept, buf[:n]...)
			}
			c.mu.Unlock()
		}
		if err != nil {
			close(c.drainDone)
			return
		}
	}
}

// Close terminates and reaps the child within one absolute cleanup
// budget: stdin EOF, then SIGTERM to the direct child, then SIGKILL to
// the direct child only — never to its process group, whose membership
// does not prove disposability. It unblocks pipe readers, joins the
// drain goroutine, and removes the temporary directory, all against
// that same deadline, returning an error when cleanup did not fully
// complete. Safe to call more than once (the first outcome is kept).
// Descendants that outlive the direct child are preserved; shared
// provider leaders outside this child are never signalled.
func (c *Child) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.shutdown()
	})
	return c.closeErr
}

func (c *Child) shutdown() error {
	// One absolute deadline bounds the whole sequence, including finish;
	// no stage grants a fresh budget.
	deadline := time.Now().Add(c.cfg.CleanupBudget)

	// Stage 1: close stdin and allow EOF-driven exit.
	c.closeStdin()
	if c.exitedBy(earliest(time.Now().Add(eofStage), deadline)) {
		// Descendants that survive the direct child's own exit are
		// preserved here on purpose: membership in this child's private
		// group does not prove a descendant is disposable (a
		// provider-managed shared leader can be a descendant). Our pipe
		// ends are closed in finish, which is what unblocks our readers;
		// the reaped child and removed temp dir complete our duties.
		return c.finish(deadline)
	}

	// Stage 2: SIGTERM the direct child only.
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Signal(syscall.SIGTERM)
	}
	if c.exitedBy(earliest(time.Now().Add(termStage), deadline)) {
		// Same descendant boundary as stage 1.
		return c.finish(deadline)
	}

	// Stage 3: SIGKILL the direct child only. The child refuses graceful
	// termination, but its unresponsiveness does not prove that every
	// process in its group is disposable: a provider-managed shared
	// leader can be a descendant with uncertain ownership. Terminating,
	// reaping, and unblocking pipes for the child we started stays
	// within our ownership; the group is never signalled (ARCH-05).
	if c.cmd.Process != nil {
		_ = c.killDirect()
	}
	reaped := false
	select {
	case <-c.done:
		reaped = true
	case <-time.After(time.Until(deadline)):
		// SIGKILL is not blockable; this is a scheduler artifact, but
		// report it rather than hiding an unreaped child.
	}
	err := c.finish(deadline)
	if !reaped {
		return errors.Join(ErrCleanupIncomplete, err)
	}
	return err
}

func earliest(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (c *Child) exitedBy(t time.Time) bool {
	select {
	case <-c.done:
		return true
	case <-time.After(time.Until(t)):
		return false
	}
}

// CloseStdin closes the write end of the child's stdin, signalling EOF to
// a well-behaved provider. It is part of Close's shutdown sequence and is
// safe to call separately.
func (c *Child) CloseStdin() {
	c.closeStdin()
}

func (c *Child) closeStdin() {
	c.stdinOnce.Do(func() {
		if c.stdinW != nil {
			_ = c.stdinW.Close()
		}
	})
}

// finish unblocks readers, joins the stderr drain, and removes the
// temporary directory, all against the shutdown's absolute deadline. A
// drain that cannot be joined in time surfaces ErrCleanupIncomplete
// instead of hiding incomplete cleanup behind a success result. It must
// not overwrite a collected result: the child's wait error belongs to
// the caller's interpretation layer.
func (c *Child) finish(deadline time.Time) error {
	if c.stdoutR != nil {
		_ = c.stdoutR.Close()
	}
	if c.stderrR != nil {
		_ = c.stderrR.Close()
	}
	select {
	case <-c.drainDone:
		return os.RemoveAll(c.dir)
	case <-time.After(time.Until(deadline)):
		return errors.Join(ErrCleanupIncomplete, os.RemoveAll(c.dir))
	}
}
