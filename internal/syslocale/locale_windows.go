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

// Lang returns the two-letter code of the Windows UI language, limited to the languages the app
// ships strings for. The list mirrors i18n.Codes; anything else falls back to "en". Regional
// variants resolve to their language, because the primary language id is all this reads - so
// de-AT, pt-BR and zh-Hans-CN each find their translation.
func Lang() string {
	switch primaryLangID() {
	case 0x19: // LANG_RUSSIAN
		return "ru"
	case 0x22: // LANG_UKRAINIAN
		return "uk"
	case 0x07: // LANG_GERMAN
		return "de"
	case 0x10: // LANG_ITALIAN
		return "it"
	case 0x0A: // LANG_SPANISH
		return "es"
	case 0x0C: // LANG_FRENCH
		return "fr"
	case 0x16: // LANG_PORTUGUESE
		return "pt"
	case 0x01: // LANG_ARABIC
		return "ar"
	case 0x39: // LANG_HINDI
		return "hi"
	case 0x45: // LANG_BENGALI
		return "bn"
	case 0x20: // LANG_URDU
		return "ur"
	case 0x04: // LANG_CHINESE
		return "zh"
	default:
		return "en"
	}
}
