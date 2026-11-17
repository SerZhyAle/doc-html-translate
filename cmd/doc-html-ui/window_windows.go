//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32Win               = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows         = user32Win.NewProc("EnumWindows")
	procIsWindowVisible     = user32Win.NewProc("IsWindowVisible")
	procGetWindowTextW      = user32Win.NewProc("GetWindowTextW")
	procIsIconic            = user32Win.NewProc("IsIconic")
	procShowWindow          = user32Win.NewProc("ShowWindow")
	procBringWindowToTop    = user32Win.NewProc("BringWindowToTop")
	procSetForegroundWindow = user32Win.NewProc("SetForegroundWindow")
)

const (
	swRestore       = 9
	appWindowPrefix = "DOC-HTML-UI"
)

// raiseLauncherWindow finds the browser --app window titled "DOC-HTML-UI" via EnumWindows, restores it if
// minimized, and brings it to the foreground. Windows refusal to foreground is never a failure (APP-COMMANDS 0.2 N.2).
var raiseLauncherWindow = findAndRaiseLauncherWindow

func findAndRaiseLauncherWindow() (found bool, focused bool) {
	var target uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		var buf [256]uint16
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			return 1
		}
		title := windows.UTF16ToString(buf[:n])
		if title == appWindowPrefix || strings.HasPrefix(title, appWindowPrefix+" - ") {
			target = hwnd
			return 0
		}
		return 1
	})
	_, _, _ = procEnumWindows.Call(cb, 0)
	if target == 0 {
		return false, false
	}
	if iconic, _, _ := procIsIconic.Call(target); iconic != 0 {
		_, _, _ = procShowWindow.Call(target, swRestore)
	}
	_, _, _ = procBringWindowToTop.Call(target)
	r, _, _ := procSetForegroundWindow.Call(target)
	return true, r != 0
}
