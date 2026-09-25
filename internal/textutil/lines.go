package textutil

import (
	"strings"
	"unicode/utf8"
)

// NormalizeLineSeparators converts common platform and Unicode line
// separators to '\n'. Form-feed is also treated as a line break.
func NormalizeLineSeparators(text string) string {
	return normalizeLineSeparators(text, false)
}

// NormalizeLineSeparatorsPreserveFormFeed converts common platform and
// Unicode line separators to '\n' but keeps form-feed intact for callers
// that use it as a page separator.
func NormalizeLineSeparatorsPreserveFormFeed(text string) string {
	return normalizeLineSeparators(text, true)
}

func normalizeLineSeparators(text string, preserveFormFeed bool) string {
	// Invalid bytes (pdftotext passes through whatever a broken font maps to) stay visible as
	// U+FFFD: dropping them silently glued the neighbouring letters together.
	if !utf8.ValidString(text) {
		text = DecodeUTF8([]byte(text))
	}

	pairs := []string{
		"\r\n", "\n",
		"\r", "\n",
		"\u0085", "\n",
		"\u2028", "\n",
		"\u2029", "\n",
		"\v", "\n",
	}
	if !preserveFormFeed {
		pairs = append(pairs, "\f", "\n")
	}

	return strings.NewReplacer(pairs...).Replace(text)
}
