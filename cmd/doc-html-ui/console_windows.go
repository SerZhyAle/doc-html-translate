//go:build windows

package main

import (
	"io"
	"os"

	"golang.org/x/sys/windows"
)

var procAttachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

// attachParentProcess is ATTACH_PARENT_PROCESS, (DWORD)-1.
const attachParentProcess = ^uintptr(0)

// stdHandleUsable reports whether the process was started with that standard handle - a redirect or a pipe from the
// caller. A window-subsystem program started from a console without a redirect has none.
func stdHandleUsable(which uint32) bool {
	h, err := windows.GetStdHandle(which)
	return err == nil && h != 0 && h != windows.InvalidHandle
}

// openConsole is where the commands front writes (APP-COMMANDS rule 9): this executable has no console of its own, so
// for a channel the caller did not redirect it attaches to the parent's console and writes there. A redirected
// channel is left as it is, so `doc-html-ui.exe commands --json > manifest.json` and a pipe work. With no parent
// console (started from Explorer or a service) the writes go nowhere and the exit code is the answer.
func openConsole() (io.Writer, io.Writer) {
	needOut := !stdHandleUsable(windows.STD_OUTPUT_HANDLE)
	needErr := !stdHandleUsable(windows.STD_ERROR_HANDLE)
	if needOut || needErr {
		_, _, _ = procAttachConsole.Call(attachParentProcess)
		if needOut {
			if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
				os.Stdout = f
			}
		}
		if needErr {
			if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
				os.Stderr = f
			}
		}
	}
	return os.Stdout, os.Stderr
}
