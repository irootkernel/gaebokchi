//go:build unix

package process

import (
	"os/exec"
	"syscall"
)

// prepareSysProcAttr puts the child into its own process group. The group
// is created by gaebokchi for this child, so group-level SIGKILL during
// cleanup can only reach descendants of the child we started — never a
// provider-managed shared leader or any unrelated session.
func prepareSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup SIGKILLs the child's private process group and the child
// itself. Failures are ignored: the shutdown sequence is best-effort and
// bounded.
func (c *Child) killGroup() {
	if c.cmd.Process == nil {
		return
	}
	pid := c.cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
