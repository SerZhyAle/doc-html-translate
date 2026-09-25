//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// prepareTree puts the child in its own process group, so one signal reaches its children.
func prepareTree(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

type groupTree struct{ pgid int }

func attachTree(cmd *exec.Cmd) (procTree, error) {
	return &groupTree{pgid: cmd.Process.Pid}, nil
}

func (t *groupTree) Kill() error { return syscall.Kill(-t.pgid, syscall.SIGKILL) }

func (t *groupTree) Release() {}
