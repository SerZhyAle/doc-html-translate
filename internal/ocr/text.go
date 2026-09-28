package ocr

import (
	"regexp"
	"strings"
	"unicode"
)

// ocrVowel is the set of vowels (lower-cased, Latin + accented + Cyrillic) used to tell
// real words from consonant soup. CJK has no vowels and is handled separately.
const ocrVowel = "aeiouyàáâãäåæèéêëìíîïòóôõöøùúûüýÿаеёиоуыэюяєії"

// ocrAddress matches text that is, as a whole, an address rather than prose: a URL, an
// email, a bare domain (a.b, a.b.c/..), a Windows path (c:\..) or a POSIX path (/a/b).
var ocrAddress = regexp.MustCompile(`(?i)^(?:https?://|www\.)\S+$|^\S+@\S+\.\S+$|^[\w-]+(?:\.[\w-]+)+(?:[/?#]\S*)?$|^[a-z]:\\|^/[\w./-]+$`)

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

func hasVowel(s string) bool {
	for _, r := range strings.ToLower(s) {
		if strings.ContainsRune(ocrVowel, r) {
			return true
		}
	}
	return false
}

// isTranslatable reports whether a recognized OCR block is worth overlaying as a
// translatable plate. It rejects OCR noise that has nothing to translate: fewer than 5
// letters (also kills pure numbers / symbols), a run of letters with no vowels (consonant
// soup), text that is wholly an address (URL / email / domain / path), and low-quality
// "mishmash" where few whitespace tokens look like real words. Short CJK phrases are kept.
// Mirrors the extension's ocr-text.js isTranslatable - keep the two in sync (docs/PARITY.md).
// OCR-OVERLAY rule 13: policy - five letters, a vowel, half the lettered tokens word-like, a CJK
// bypass (OCR-PIPELINE 2.6).
func isTranslatable(raw string) bool {
	t := strings.Join(strings.Fields(raw), " ")
	if t == "" {
		return false
	}

	var cjk, letters int
	for _, r := range t {
		switch {
		case isCJK(r):
			cjk++
		case unicode.IsLetter(r):
			letters++
		}
	}
	if cjk >= 2 { // short CJK phrases are translatable (no spaces / vowels there)
		return true
	}
	if letters < 5 {
		return false
	}
	if !hasVowel(t) {
		return false
	}
	if ocrAddress.MatchString(t) {
		return false
	}

	// Mishmash: among tokens that carry letters, how many look like real words
	// (>= 2 letters and containing a vowel)?
	var lettered, wordlike int
	for _, w := range strings.Fields(t) {
		var l int
		for _, r := range w {
			if unicode.IsLetter(r) && !isCJK(r) {
				l++
			}
		}
		if l == 0 {
			continue
		}
		lettered++
		if l >= 2 && hasVowel(w) {
			wordlike++
		}
	}
	if lettered >= 3 && float64(wordlike)/float64(lettered) < 0.5 {
		return false
	}
	return true
}

// repairPipeMisreads rewrites a line's bare "|" tokens into "I", sparing the text-token indexes in
// protected, and returns the text unchanged when nothing was rewritten.
//
// A serif capital I is a bare vertical stroke, and the recognizer reads it as a pipe: on the
// 2026-09-28 field repro (a photographed school-text page) every one of the page's 15 standalone
// "I" tokens came back as "|" on an otherwise well recognized page, and the translation kept each
// bar and lost the subject with it - "I get up at seven o'clock" arrived as the imperative
// "Вставай в семь". This is the token mechanic; the guards that spare a bar - the outline the
// outlier trim handles, a table's column grid - live at the caller, in the cluster flush
// (OCR-PIPELINE amendment 1.5).
// Mirrors the extension's ocr-text.js repairPipeMisreads - keep the two in sync (docs/PARITY.md).
func repairPipeMisreads(line string, protected map[int]bool) string {
	toks := strings.Split(line, " ")
	changed := false
	for i, tok := range toks {
		if tok != "|" || protected[i] {
			continue
		}
		toks[i] = "I"
		changed = true
	}
	if !changed {
		return line
	}
	return strings.Join(toks, " ")
}
