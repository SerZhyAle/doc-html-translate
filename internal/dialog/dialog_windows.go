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

// ConfirmYesNo shows a Windows Yes/No dialog. Returns true if user clicked Yes.
func ConfirmYesNo(title, message string) bool {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	ret, _, _ := messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbYesNo|mbIconQuestion,
	)
	return ret == idYes
}

// ShowWarning displays a Windows warning message box with an OK button.
func ShowWarning(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbOK|mbIconWarning,
	)
}
