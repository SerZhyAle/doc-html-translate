//go:build !windows

package procrun

import (
	"os/exec"
	"syscall"
)

// processTree is the helper's process group. The helper leads its own group, so one signal
// to the negative pid reaches every process it started that did not leave the group.
type processTree struct {
	pgid int
}

func prepareTree(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func attachTree(cmd *exec.Cmd) *processTree {
	return &processTree{pgid: cmd.Process.Pid}
}

func killTree(cmd *exec.Cmd, t *processTree) error {
	if t != nil {
		_ = syscall.Kill(-t.pgid, syscall.SIGKILL)
	}
	return cmd.Process.Kill()
}

// releaseTree kills whatever is still in the group once the helper has been waited for. The
// group id cannot be handed to a new process while any member lives, so this reaches only the
// helper's own leftovers.
func releaseTree(t *processTree) {
	if t != nil {
		_ = syscall.Kill(-t.pgid, syscall.SIGKILL)
	}
}
