//go:build windows

package dialog

import (
	"syscall"
	"unsafe"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32.NewProc("MessageBoxW")
)

const (
	mbOK           = uintptr(0x00000000)
	mbYesNo        = uintptr(0x00000004)
	mbIconWarning  = uintptr(0x00000030)
	mbIconQuestion = uintptr(0x00000020)
	idYes          = uintptr(6)
)

// messageBox is indirected so a test can check the text and buttons a dialog asks
// for without showing one. It returns MessageBoxW's result: the button pressed, or 0
// when the box could not be shown.
var messageBox = func(title, message string, style uintptr) uintptr {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	ret, _, _ := messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		style,
	)
	return ret
}

// ConfirmYesNo shows a Windows Yes/No dialog. Returns true if user clicked Yes.
func ConfirmYesNo(title, message string) bool {
	return messageBox(title, message, mbYesNo|mbIconQuestion) == idYes
}

// ShowWarning displays a Windows warning message box with an OK button.
func ShowWarning(title, message string) {
	messageBox(title, message, mbOK|mbIconWarning)
}
