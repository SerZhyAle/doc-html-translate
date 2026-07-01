//go:build windows

package syslocale

import "syscall"

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getUserDefaultUILang = kernel32.NewProc("GetUserDefaultUILanguage")
)

// primaryLangID returns the Windows primary language identifier of the current
// user's UI language - the low 10 bits of the LANGID returned by
// GetUserDefaultUILanguage.
func primaryLangID() uint16 {
	ret, _, _ := getUserDefaultUILang.Call()
	return uint16(ret) & 0x3FF
}

// IsRussian returns true when the Windows UI language is Russian.
func IsRussian() bool {
	return primaryLangID() == 0x19 // LANG_RUSSIAN
}

// Lang returns the two-letter code of the Windows UI language, limited to the
// languages the app ships strings for: "ru", "uk", or "en" (the fallback).
func Lang() string {
	switch primaryLangID() {
	case 0x19: // LANG_RUSSIAN
		return "ru"
	case 0x22: // LANG_UKRAINIAN
		return "uk"
	default:
		return "en"
	}
}
