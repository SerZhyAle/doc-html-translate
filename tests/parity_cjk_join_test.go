package tests

import (
	"regexp"
	"testing"
)

// TestParityOCRCJKJoin: the CJK word join of OCR-PIPELINE amendment 1.8 has one constant and two call
// sites per edition, and the shared fixture (tests/testdata/ocr_cjk_join_cases.json) pins only the
// functions. So the value must match, and each edition must build a line's text and a plate's text
// through the rule - a side that went back to joining with a plain space would pass every fixture
// case and put the spaces back on the page. See docs/PARITY.md "CJK word join".
func TestParityOCRCJKJoin(t *testing.T) {
	textGo := readRepoFile(t, "internal", "ocr", "text.go")
	textJS := readRepoFile(t, "extension", "src", "ocr-text.js")
	goSrc := readRepoFile(t, "internal", "ocr", "tesseract.go")
	overlaySrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	clusterSrc := readRepoFile(t, "extension", "src", "ocr-cluster.js")
	strengthGo := readRepoFile(t, "internal", "ocr", "strength.go")

	gv := num(t, "hangul join gap ratio (text.go)", `ocrHangulJoinGapRatio\s*=\s*([\d.]+)`, textGo)
	jv := num(t, "hangul join gap ratio (ocr-text.js)", `OCR_HANGUL_JOIN_GAP_RATIO\s*=\s*([\d.]+)`, textJS)
	if gv != jv {
		t.Errorf("hangul join gap ratio drift: text.go=%v ocr-text.js=%v (must match - see docs/PARITY.md CJK word join)", gv, jv)
	}

	for _, c := range []struct{ name, file, src, re string }{
		{"a line's text is its words joined by the rule", "tesseract.go", goSrc, `cur\.text\.WriteString\(joinLineWords\(cur\.words\)\)`},
		{"a cut run's text is its own words joined by the rule", "tesseract.go", goSrc, `l\.text\.WriteString\(joinLineWords\(words\)\)`},
		{"a line's text is its words joined by the rule", "ocr-overlay.js", overlaySrc, `words\.length \? joinLineWords\(words, scale\)`},
		{"a plate's text is its lines joined by the rule", "tesseract.go", goSrc, `Text: joinPlateLines\(ctexts\)`},
		{"a plate's text is its lines joined by the rule", "ocr-cluster.js", clusterSrc, `text: joinPlateLines\(cur\.texts\)`},
		{"the Hangul gap is measured between the boxes", "text.go", textGo, `gap := max\(next\.x0-prev\.x1, prev\.x0-next\.x1\)`},
		{"the Hangul gap is measured between the boxes", "ocr-text.js", textJS, `Math\.max\(at\(next\.bbox\.x0\) - at\(prev\.bbox\.x1\), at\(prev\.bbox\.x0\) - at\(next\.bbox\.x1\)\)`},
		{"the CJK punctuation set", "text.go", textGo, `\(r >= 0x3000 && r <= 0x303F\) \|\| r == 0x30FB \|\| r == 0x30FC \|\| \(r >= 0xFF00 && r <= 0xFFEF\)`},
		{"the CJK punctuation set", "ocr-text.js", textJS, `\[\\u3000-\\u303F\\u30FB\\u30FC\\uFF00-\\uFFEF\]`},
		// The two readers of the old spaced text keep reading what they were measured on (1.8 C, F):
		// the rung comparator counts the recognizer's words, the rescue admission a word's own run.
		{"the comparator counts the recognizer's words", "strength.go", strengthGo, `n \+= b\.tokens`},
		{"the comparator counts the recognizer's words", "ocr-cluster.js", clusterSrc, `b\.tokens > 0 \? b\.tokens`},
		{"the rescue admission reads a word's own run", "tesseract.go", goSrc, `l\.letterRun\(\) >= ocrRescueAnchorRun`},
		{"the rescue admission reads a word's own run", "ocr-cluster.js", clusterSrc, `lineLetterRun\(l\) >= OCR_RESCUE_ANCHOR_RUN`},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: %s no longer holds (OCR-PIPELINE amendment 1.8, docs/PARITY.md CJK word join)", c.file, c.name)
		}
	}
}
