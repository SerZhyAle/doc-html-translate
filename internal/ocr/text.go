package ocr

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
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

// ocrHangulJoinGapRatio is how close two Hangul words must stand, as a fraction of the line's median
// word height, to be one word the recognizer cut in two rather than two words the writer spaced.
//
// Japanese and Chinese are written without spaces, so two recognized tokens that meet on Han or kana
// are joined outright. Korean spaces its words (eojeol), never its syllables, and the recognizer cuts
// both: "코 모 나 시 장 으로서," for the lettering "코모나 시장으로서,", where exactly one of the
// four spaces is real. Joining every Hangul pair is as wrong as keeping every space, so the pair is
// judged on the gap between the two boxes.
//
// Measured on the 12 Korean Pepper&Carrot pages of the ticket 68 corpus against the author's own
// lettering, 480 Hangul word pairs labelled from that truth
// (DEV/research/RESEARCH_cjk-word-join_2026-09-29.md): the exact-substring rate of the lettering lines is 19/235 with every space kept and 13/235 with
// none, and 46-48/235 on a plateau from 0.30 to 0.36 of the median word height, falling to 43 at 0.28
// and 44 at 0.38. 0.33 is the geometric middle of the plateau's two ends.
//
// OCR-OVERLAY rule 13: derived - RESEARCH_cjk-word-join_2026-09-29 (OCR-PIPELINE amendment 1.8 A).
const ocrHangulJoinGapRatio = 0.33

// isCJKPunct reports the punctuation CJK text is set with, which Unicode files under the Common
// script and isCJK therefore does not see: the CJK Symbols and Punctuation block (、。「」), the
// katakana middle dot and prolonged sound mark (・ー, in the Katakana block but Common), and the
// halfwidth and fullwidth forms (，！？ and their halfwidth kana). Measured with the Hangul rule on
// the same corpus: joining on these as on Han and kana lifts the Japanese lettering lines matched
// exactly from 46 to 52 of 270 and the Chinese from 21 to 25 of 155, while also joining ASCII
// punctuation after a CJK character drops Japanese back to 47 - so ASCII punctuation keeps its space.
func isCJKPunct(r rune) bool {
	return (r >= 0x3000 && r <= 0x303F) || r == 0x30FB || r == 0x30FC || (r >= 0xFF00 && r <= 0xFFEF)
}

// cjkMeet reports whether two consecutive tokens meet on CJK text - the last character of prev and
// the first of next each a CJK letter (isCJK) or CJK punctuation (isCJKPunct) - and whether either of
// those two is Hangul, the one script of the set that spaces its words.
func cjkMeet(prev, next string) (cjk, hangul bool) {
	a, _ := utf8.DecodeLastRuneInString(prev)
	b, _ := utf8.DecodeRuneInString(next)
	if !(isCJK(a) || isCJKPunct(a)) || !(isCJK(b) || isCJKPunct(b)) {
		return false, false
	}
	return true, unicode.Is(unicode.Hangul, a) || unicode.Is(unicode.Hangul, b)
}

// joinLineWords builds one recognizer line's text from its words (OCR-PIPELINE amendment 1.8 A). Two
// consecutive words are joined with a space, except where they meet on CJK text (cjkMeet): on Han,
// kana and CJK punctuation with no space at all, and where Hangul is involved with no space only when
// the two boxes stand closer than ocrHangulJoinGapRatio times the median height of these words. A
// space between a CJK character and a Latin word, a digit or ASCII punctuation is kept. Words with no
// measurable height leave the Hangul space in place - there is nothing to measure the gap against.
// Mirrors the extension's ocr-text.js joinLineWords - keep the two in sync (docs/PARITY.md).
func joinLineWords(words []ocrWord) string {
	var hs []int
	for _, w := range words {
		if h := w.y1 - w.y0; h > 0 {
			hs = append(hs, h)
		}
	}
	med := median(hs, 0)
	var b strings.Builder
	for i, w := range words {
		if i > 0 && !joinsUnspaced(words[i-1], w, med) {
			b.WriteByte(' ')
		}
		b.WriteString(w.text)
	}
	return b.String()
}

func joinsUnspaced(prev, next ocrWord, med int) bool {
	cjk, hangul := cjkMeet(prev.text, next.text)
	if !cjk {
		return false
	}
	if !hangul {
		return true
	}
	gap := max(next.x0-prev.x1, prev.x0-next.x1)
	return med > 0 && float64(gap) < float64(med)*ocrHangulJoinGapRatio
}

// joinPlateLines builds a plate's text from its lines, in reading order (OCR-PIPELINE amendment 1.8
// B). Lines are joined with a space, except where they meet on Han, kana or CJK punctuation: a
// Japanese or Chinese phrase wraps with no space at the line break. A Hangul line break keeps its
// space - Korean lettering breaks its lines between words, and nothing is left to measure a gap on.
// Mirrors the extension's ocr-text.js joinPlateLines - keep the two in sync (docs/PARITY.md).
func joinPlateLines(lines []string) string {
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			if cjk, hangul := cjkMeet(lines[i-1], l); !cjk || hangul {
				b.WriteByte(' ')
			}
		}
		b.WriteString(l)
	}
	return b.String()
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
