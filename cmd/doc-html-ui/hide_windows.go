//go:build windows

package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hideWindow keeps a console child from opening a console window. HideWindow alone only
// hides a window Windows has already created for it; CREATE_NO_WINDOW stops the console
// being created, so the user has nothing to close by accident mid-run.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}
