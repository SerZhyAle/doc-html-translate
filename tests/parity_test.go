package tests

// Cross-edition parity guards. These tests parse the two independent codebases (the Go
// app and the JS extension) and assert the values that must stay identical actually do.
// A change on one side that is not mirrored on the other fails here. The pinned contracts
// live in docs/PARITY.md; keep this test, that doc, and the code in step.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// readRepoFile reads a file by repo-relative path (this package lives in tests/, so the
// repo root is one level up).
func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{".."}, parts...)...)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	s = s[i+len(start):]
	if j := strings.Index(s, end); j >= 0 {
		return s[:j]
	}
	return s
}

// normHex expands #rgb to #rrggbb and lowercases, so #222 and #222222 compare equal.
func normHex(h string) string {
	h = strings.ToLower(strings.TrimPrefix(h, "#"))
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	return h
}

var hexRe = regexp.MustCompile(`#[0-9a-fA-F]{3,6}\b`)

func hexesIn(block string) []string {
	var out []string
	for _, m := range hexRe.FindAllString(block, -1) {
		out = append(out, normHex(m))
	}
	return out
}

// cssBlock returns the text between `selector {` and the next `}`. When several blocks
// share a selector (e.g. :root), mustContain picks the one holding that substring.
func cssBlock(text, selector, mustContain string) string {
	re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(selector) + `\s*\{(.*?)\}`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		if mustContain == "" || strings.Contains(m[1], mustContain) {
			return m[1]
		}
	}
	return ""
}

func codeSet(ms [][]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range ms {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

// TestParityThemePalette: the four reader themes carry identical colour values in the
// desktop app's navbar.go readerCSS and the extension's viewer.css (var names differ,
// values must not). See docs/PARITY.md "Reader theme palette".
func TestParityThemePalette(t *testing.T) {
	goReader := regexp.MustCompile("(?s)readerCSS = \x60(.*?)\x60").
		FindStringSubmatch(readRepoFile(t, "internal", "htmlgen", "navbar.go"))
	if goReader == nil {
		t.Fatal("could not locate readerCSS block in navbar.go")
	}
	goCSS := goReader[1]
	jsCSS := readRepoFile(t, "extension", "src", "viewer.css")

	themes := []struct{ name, goSel, jsSel, goHas, jsHas string }{
		{"light", ":root", ":root", "--dht-bg", "--bg"},
		{"sepia", `html[data-dht-theme="sepia"]`, `html[data-theme="sepia"]`, "", ""},
		{"dark", `html[data-dht-theme="dark"]`, `html[data-theme="dark"]`, "", ""},
		{"night", `html[data-dht-theme="night"]`, `html[data-theme="night"]`, "", ""},
	}
	for _, th := range themes {
		goHex := hexesIn(cssBlock(goCSS, th.goSel, th.goHas))
		jsHex := hexesIn(cssBlock(jsCSS, th.jsSel, th.jsHas))
		if len(goHex) == 0 || len(jsHex) == 0 {
			t.Fatalf("theme %q: palette not found (navbar.go=%d viewer.css=%d hexes) - selectors may have changed", th.name, len(goHex), len(jsHex))
		}
		if strings.Join(goHex, ",") != strings.Join(jsHex, ",") {
			t.Errorf("theme %q palette drift:\n  navbar.go : %v\n  viewer.css: %v\n  values must match - see docs/PARITY.md", th.name, goHex, jsHex)
		}
	}
}

// TestParityOCRVersion: the desktop app and the extension must download the same
// tessdata_fast version, or OCR recognition differs. See docs/PARITY.md "OCR".
func TestParityOCRVersion(t *testing.T) {
	goV := regexp.MustCompile(`tessdata_fast/raw/(\d+\.\d+\.\d+)`).
		FindStringSubmatch(readRepoFile(t, "internal", "ocr", "tessdata.go"))
	verRe := regexp.MustCompile(`projectnaptha\.com/(\d+\.\d+\.\d+)_fast`)
	jsLangV := verRe.FindStringSubmatch(readRepoFile(t, "extension", "src", "ocr-lang.js"))
	jsBuildV := verRe.FindStringSubmatch(readRepoFile(t, "extension", "build.mjs"))
	if goV == nil || jsLangV == nil || jsBuildV == nil {
		t.Fatalf("tessdata version not found (tessdata.go=%v ocr-lang.js=%v build.mjs=%v)", goV, jsLangV, jsBuildV)
	}
	if goV[1] != jsLangV[1] || goV[1] != jsBuildV[1] {
		t.Errorf("tessdata version drift: tessdata.go=%s ocr-lang.js=%s build.mjs=%s (must match - see docs/PARITY.md OCR)", goV[1], jsLangV[1], jsBuildV[1])
	}
}

// TestParityOCRCatalog: the OCR language catalog is the same on both sides. See
// docs/PARITY.md "OCR".
func TestParityOCRCatalog(t *testing.T) {
	// The `{"code", "Name"}` tuple is unique to LangInfo entries in tessdata.go (Bundled
	// uses {"eng"} with no comma), so scan the whole file rather than a brace-delimited
	// block (entries themselves contain `}`).
	goSrc := readRepoFile(t, "internal", "ocr", "tessdata.go")
	jsBlock := between(readRepoFile(t, "extension", "src", "ocr-lang.js"), "export const LANGS = [", "]")
	goCodes := codeSet(regexp.MustCompile(`\{"([a-z_]+)",\s*"`).FindAllStringSubmatch(goSrc, -1))
	jsCodes := codeSet(regexp.MustCompile(`code:\s*"([a-z_]+)"`).FindAllStringSubmatch(jsBlock, -1))
	if len(goCodes) == 0 || len(jsCodes) == 0 {
		t.Fatalf("catalog not parsed (tessdata.go=%d ocr-lang.js=%d entries)", len(goCodes), len(jsCodes))
	}
	if strings.Join(goCodes, ",") != strings.Join(jsCodes, ",") {
		t.Errorf("OCR language catalog drift:\n  tessdata.go Available: %v\n  ocr-lang.js LANGS    : %v\n  must match - see docs/PARITY.md", goCodes, jsCodes)
	}
}

// TestParityOCRClustering: the overlay's line-clustering, pre-OCR upscale, and page-segmentation
// constants must match, or the two editions recognize/group text differently. See docs/PARITY.md "OCR".
func TestParityOCRClustering(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "tesseract.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	pairs := []struct{ name, goRe, jsRe string }{
		{"min line confidence", `ocrMinLineConf\s*=\s*([\d.]+)`, `OCR_MIN_LINE_CONF\s*=\s*([\d.]+)`},
		{"cluster gap factor", `ocrClusterGapFactor\s*=\s*([\d.]+)`, `OCR_CLUSTER_GAP_FACTOR\s*=\s*([\d.]+)`},
		{"upscale dpi floor", `ocrUpscaleDPIFloor\s*=\s*([\d.]+)`, `OCR_UPSCALE_DPI_FLOOR\s*=\s*([\d.]+)`},
		{"assumed page inches", `ocrAssumedPageInches\s*=\s*([\d.]+)`, `OCR_ASSUMED_PAGE_INCHES\s*=\s*([\d.]+)`},
		{"min declared dpi", `ocrMinDeclaredDPI\s*=\s*([\d.]+)`, `OCR_MIN_DECLARED_DPI\s*=\s*([\d.]+)`},
		{"upscale factor", `ocrUpscaleFactor\s*=\s*([\d.]+)`, `OCR_UPSCALE_FACTOR\s*=\s*([\d.]+)`},
		{"page-seg mode", `ocrPageSegMode\s*=\s*([\d.]+)`, `OCR_PSM\s*=\s*"?([\d.]+)"?`},
	}
	for _, p := range pairs {
		gv := num(t, p.name+" (tesseract.go)", p.goRe, goSrc)
		jv := num(t, p.name+" (ocr-overlay.js)", p.jsRe, jsSrc)
		if gv != jv {
			t.Errorf("%s drift: tesseract.go=%v ocr-overlay.js=%v (must match - see docs/PARITY.md OCR)", p.name, gv, jv)
		}
	}
}

// TestParityOCRFontFit: the plate font-fit factor (font-size = median line height x factor) must
// match across editions - overlay.go fontFitFactor vs ocr-overlay.js FONT_FIT. See docs/PARITY.md
// "OCR" (plate geometry).
func TestParityOCRFontFit(t *testing.T) {
	gv := num(t, "fontFitFactor (overlay.go)", `fontFitFactor\s*=\s*([\d.]+)`, readRepoFile(t, "internal", "ocr", "overlay.go"))
	jv := num(t, "FONT_FIT (ocr-overlay.js)", `FONT_FIT\s*=\s*([\d.]+)`, readRepoFile(t, "extension", "src", "ocr-overlay.js"))
	if gv != jv {
		t.Errorf("font fit factor drift: overlay.go=%v ocr-overlay.js=%v (must match - see docs/PARITY.md OCR)", gv, jv)
	}
}

// TestParityReflowConstants: the PDF reflow heuristic thresholds shared by extract.go
// and reflow.js must hold the same values. See docs/PARITY.md "PDF reflow heuristics".
func TestParityReflowConstants(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "pdf", "extract.go")
	jsSrc := readRepoFile(t, "extension", "src", "reflow.js")
	pairs := []struct{ name, goRe, jsRe string }{
		{"paragraph gap factor", `paraGapFactor\s+=\s+([\d.]+)`, `PARA_GAP_FACTOR\s*=\s*([\d.]+)`},
		{"indent threshold", `indentThreshold\s+=\s+([\d.]+)`, `INDENT_THRESHOLD\s*=\s*([\d.]+)`},
		{"ligature max avg word length", `ligatureMaxAvgWordLen\s+=\s+([\d.]+)`, `total\s*/\s*words\.length\s*<\s*([\d.]+)`},
	}
	for _, p := range pairs {
		gv := num(t, p.name+" (extract.go)", p.goRe, goSrc)
		jv := num(t, p.name+" (reflow.js)", p.jsRe, jsSrc)
		if gv != jv {
			t.Errorf("%s drift: extract.go=%v reflow.js=%v (must match - see docs/PARITY.md)", p.name, gv, jv)
		}
	}
}

// TestParityComicPageFilter: the comic page-image extension set must be identical on both
// editions, or the two disagree on which archive entries count as pages. See docs/PARITY.md
// "Comic archive page order and entry filter".
func TestParityComicPageFilter(t *testing.T) {
	goBlock := between(readRepoFile(t, "internal", "comic", "extract.go"), "var pageExts = map[string]bool{", "}")
	jsBlock := between(readRepoFile(t, "extension", "src", "comic.js"), "export const PAGE_EXTS = [", "]")
	goExts := codeSet(regexp.MustCompile(`"\.([a-z0-9]+)"`).FindAllStringSubmatch(goBlock, -1))
	jsExts := codeSet(regexp.MustCompile(`"([a-z0-9]+)"`).FindAllStringSubmatch(jsBlock, -1))
	if len(goExts) == 0 || len(jsExts) == 0 {
		t.Fatalf("comic page-ext set not parsed (extract.go=%d comic.js=%d)", len(goExts), len(jsExts))
	}
	if strings.Join(goExts, ",") != strings.Join(jsExts, ",") {
		t.Errorf("comic page-ext set drift:\n  internal/comic pageExts: %v\n  comic.js PAGE_EXTS     : %v\n  must match - see docs/PARITY.md", goExts, jsExts)
	}
}

func num(t *testing.T, what, pattern, s string) float64 {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("%s: value not found (pattern %q)", what, pattern)
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("%s: parse %q: %v", what, m[1], err)
	}
	return f
}
