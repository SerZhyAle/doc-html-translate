package tests

import (
	"regexp"
	"strings"
	"testing"
)

// TestParityOCRColumnOrderFloor: each recognition pass orders its columns by its own confidence
// floor. The desktop app hands tsvLines the pass floor and tsvLines hands it to orderColumns; the
// extension's collectLines always used the ordinary floor, so on a rescue pass a line the pass then
// dropped could still chain two columns into one, in the extension only (audit finding B64). The
// shared fixture tests/testdata/ocr_column_order_cases.json shows what the floor changes; this pins
// that every pass passes the floor it filters with. See docs/PARITY.md, OCR line integrity.
func TestParityOCRColumnOrderFloor(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "tesseract.go")
	if !strings.Contains(goSrc, "orderColumns(lines, minConf)") {
		t.Error("tesseract.go: tsvLines no longer hands orderColumns the pass floor (orderColumns(lines, minConf))")
	}

	js := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	if !regexp.MustCompile(`function collectLines\([^)]*\bminConf\b[^)]*\)`).MatchString(js) ||
		!strings.Contains(js, "orderColumns(out, minConf)") {
		t.Error("ocr-overlay.js: collectLines no longer takes the pass floor and hands it to orderColumns")
	}
	// Every pass reads its lines with collectLines and filters them with droppedLines right after;
	// the floor of the two must be the same.
	call := regexp.MustCompile(`collectLines\(data, scale, ink(?:, (\w+))?\);\s*\n\s*const dropped = droppedLines\(lines, (\w+)\)`)
	calls := call.FindAllStringSubmatch(js, -1)
	if len(calls) < 4 {
		t.Fatalf("ocr-overlay.js: found %d collectLines/droppedLines passes, want the ordinary, rescue, screen rescue and screen sweep passes", len(calls))
	}
	for _, m := range calls {
		ordered := m[1]
		if ordered == "" {
			ordered = "OCR_MIN_LINE_CONF"
		}
		if ordered != m[2] {
			t.Errorf("ocr-overlay.js: a pass orders its columns by %s but drops lines by %s", ordered, m[2])
		}
	}
}
