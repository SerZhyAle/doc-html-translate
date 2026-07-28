// Package i18n is the one place a user-visible Go string is chosen.
//
// The key is the English source string, so call sites keep reading like plain text and no
// resource-id indirection is needed; the value is one translation per shipped language. Adding
// a language means one column in the registration files instead of a branch in every caller -
// which is exactly why the old per-language `if` in each printer was replaced.
//
// The set is the thirteen languages the product ships in; see the ticket
// DEV/plan/2026-07-28_thirteen-ui-languages.md for why these thirteen.
package i18n

import "fmt"

// Codes are the shipped languages. Index 0 is the key language: a key *is* its English text,
// so English needs no translation table. Order is the menu order used by every surface.
var Codes = []string{"en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh"}

// Names are the endonyms parallel to Codes - what a speaker calls their own language, which is
// the only label useful to someone who cannot read the current interface language.
var Names = []string{
	"English", "Русский", "Українська", "Deutsch", "Italiano", "Español", "Français",
	"Português", "العربية", "हिन्दी", "বাংলা", "اردو", "中文",
}

// translations maps an English source string to its 12 non-English translations, in Codes order
// after index 0.
var translations = map[string][]string{}

// IndexOf returns the position of code in Codes, or 0 (English) for anything unknown.
func IndexOf(code string) int {
	for i, c := range Codes {
		if c == code {
			return i
		}
	}
	return 0
}

// Add registers one source string with its 12 translations, in Codes order after English.
// A wrong count is a programming error caught at init rather than a silently shifted column.
func Add(en string, tr ...string) {
	if len(tr) != len(Codes)-1 {
		panic(fmt.Sprintf("i18n: %q has %d translations, want %d", en, len(tr), len(Codes)-1))
	}
	translations[en] = tr
}

// All exposes the registry to the package test.
func All() map[string][]string { return translations }

// T translates an English source string. An unknown key returns the key itself, so a new string
// is visible but never crashes; an empty translation falls back to English rather than showing a
// German user Russian. Extra args go through fmt.Sprintf, so a key may carry verbs.
func T(lang, en string, args ...any) string {
	out := en
	if idx := IndexOf(lang); idx > 0 {
		if tr, ok := translations[en]; ok && tr[idx-1] != "" {
			out = tr[idx-1]
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(out, args...)
	}
	return out
}

// current is the process-wide interface language. A conversion run has exactly one, resolved
// once at start-up, which is why this is package state rather than a parameter threaded through
// every htmlgen signature.
var current = "en"

// SetLanguage sets the process-wide interface language. An unknown code selects English.
func SetLanguage(code string) { current = Codes[IndexOf(code)] }

// Language returns the process-wide interface language.
func Language() string { return current }

// S translates in the process language. This is the call the console output and the generated
// page chrome use.
func S(en string, args ...any) string { return T(current, en, args...) }

// IsRTL reports whether the language is written right to left.
func IsRTL(lang string) bool {
	switch Codes[IndexOf(lang)] {
	case "ar", "ur":
		return true
	}
	return false
}

// Dir returns the HTML dir attribute value for the language.
func Dir(lang string) string {
	if IsRTL(lang) {
		return "rtl"
	}
	return "ltr"
}

// FontFamily returns the Windows UI font for a script the default stack does not cover, or ""
// when the default is fine. Both fonts ship with Windows: the GUI runs offline, so a webfont is
// not an option.
func FontFamily(lang string) string {
	switch Codes[IndexOf(lang)] {
	case "hi", "bn":
		return "Nirmala UI"
	case "zh":
		return "Microsoft YaHei UI"
	}
	return ""
}

// Resolve picks the first shipped language among an explicit choice (a flag), a stored user
// choice, and the operating system's language, falling back to English. Anything not shipped is
// skipped rather than accepted, so "-ui-lang xx" cannot mute a saved preference.
func Resolve(explicit, saved, system string) string {
	for _, candidate := range []string{explicit, saved, system} {
		if candidate == "" {
			continue
		}
		for _, c := range Codes {
			if c == candidate {
				return c
			}
		}
	}
	return "en"
}
