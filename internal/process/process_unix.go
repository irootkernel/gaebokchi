//go:build unix

package process

import (
	"os/exec"
	"syscall"
)

// prepareSysProcAttr puts the child into its own process group for
// signal isolation: the child and anything it starts cannot deliver
// terminal-control signals to gaebokchi's own group, and gaebokchi's
// signals to itself do not reach the child indirectly. The private
// group is NOT an ownership claim — cleanup never signals the group,
// because group membership alone does not prove a descendant (possibly
// a provider-managed shared leader) is disposable (ARCH-05).
func prepareSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killDirect SIGKILLs only the direct child gaebokchi started. The
// failure is ignored: the shutdown sequence is best-effort, bounded,
// and reports incompleteness through its returned error.
func (c *Child) killDirect() error {
	if c.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(c.cmd.Process.Pid, syscall.SIGKILL)
}
