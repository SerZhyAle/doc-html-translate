//go:build windows

package syslocale

import "syscall"

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getUserDefaultUILang = kernel32.NewProc("GetUserDefaultUILanguage")
)

// IsRussian returns true when the Windows UI language is Russian.
func IsRussian() bool {
	ret, _, _ := getUserDefaultUILang.Call()
	// Primary language ID occupies the low 10 bits; Russian = 0x19.
	return uint16(ret)&0x3FF == 0x19
}
