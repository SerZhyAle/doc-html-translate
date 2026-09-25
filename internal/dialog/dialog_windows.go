//go:build windows

package dialog

import (
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	messageBoxW      = user32.NewProc("MessageBoxW")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
)

const (
	mbOK            = uintptr(0x00000000)
	mbOKCancel      = uintptr(0x00000001)
	mbIconWarning   = uintptr(0x00000030)
	mbDefButton2    = uintptr(0x00000100)
	mbSetForeground = uintptr(0x00010000)
	mbTopmost       = uintptr(0x00040000)
	idOK            = uintptr(1)
)

// messageBox is indirected so a test can check the owner, text and buttons a dialog asks for
// without showing one. It returns MessageBoxW's result: the button pressed, or 0 when the box
// could not be shown.
var messageBox = func(owner uintptr, title, message string, style uintptr) uintptr {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	ret, _, _ := messageBoxW.Call(
		owner,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		style,
	)
	return ret
}

// consoleOwner is the console window the converter runs in, so the box is owned by it and
// opens in front of it (APP-BEHAVIOUR rule 1). It is 0 when there is no console.
var consoleOwner = func() uintptr {
	h, _, _ := getConsoleWindow.Call()
	return h
}

// ownerStyle adds what a box needs to be seen: an owned box sits above its owner, and one with
// no owner is made topmost instead of opening behind whatever has the focus.
func ownerStyle(owner uintptr) uintptr {
	if owner == 0 {
		return mbSetForeground | mbTopmost
	}
	return mbSetForeground
}

// Confirm asks before an action that cannot be taken back, such as spending money. The safe
// answer is the default: Cancel has the focus, so Enter declines, and Escape and the close box
// are the same Cancel. Only OK proceeds. Under the GUI the question is asked in the GUI's window.
func Confirm(title, message string) bool {
	if hostedByGUI() {
		return askHost(title, message)
	}
	owner := consoleOwner()
	return messageBox(owner, title, message, mbOKCancel|mbIconWarning|mbDefButton2|ownerStyle(owner)) == idOK
}

// ShowWarning displays a warning with an OK button. Under the GUI it becomes a notice in the
// GUI's window.
func ShowWarning(title, message string) {
	if hostedByGUI() {
		noteHost(title, message)
		return
	}
	owner := consoleOwner()
	messageBox(owner, title, message, mbOK|mbIconWarning|ownerStyle(owner))
}
