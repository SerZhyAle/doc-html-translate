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
	overlaySrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	clusterSrc := readRepoFile(t, "extension", "src", "ocr-cluster.js")
	pairs := []struct{ name, goRe, jsFile, jsRe string }{
		{"min line confidence", `ocrMinLineConf\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MIN_LINE_CONF\s*=\s*([\d.]+)`},
		{"cluster pitch factor", `ocrClusterPitchFactor\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_CLUSTER_PITCH_FACTOR\s*=\s*([\d.]+)`},
		{"max leading ratio", `ocrMaxLeadingRatio\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MAX_LEADING_RATIO\s*=\s*([\d.]+)`},
		{"upscale dpi floor", `ocrUpscaleDPIFloor\s*=\s*([\d.]+)`, "ocr-overlay.js", `OCR_UPSCALE_DPI_FLOOR\s*=\s*([\d.]+)`},
		{"assumed page inches", `ocrAssumedPageInches\s*=\s*([\d.]+)`, "ocr-overlay.js", `OCR_ASSUMED_PAGE_INCHES\s*=\s*([\d.]+)`},
		{"min declared dpi", `ocrMinDeclaredDPI\s*=\s*([\d.]+)`, "ocr-overlay.js", `OCR_MIN_DECLARED_DPI\s*=\s*([\d.]+)`},
		{"upscale factor", `ocrUpscaleFactor\s*=\s*([\d.]+)`, "ocr-overlay.js", `OCR_UPSCALE_FACTOR\s*=\s*([\d.]+)`},
		{"page-seg mode", `ocrPageSegMode\s*=\s*([\d.]+)`, "ocr-overlay.js", `OCR_PSM\s*=\s*"?([\d.]+)"?`},
	}
	for _, p := range pairs {
		jsSrc := overlaySrc
		if p.jsFile == "ocr-cluster.js" {
			jsSrc = clusterSrc
		}
		gv := num(t, p.name+" (tesseract.go)", p.goRe, goSrc)
		jv := num(t, p.name+" ("+p.jsFile+")", p.jsRe, jsSrc)
		if gv != jv {
			t.Errorf("%s drift: tesseract.go=%v %s=%v (must match - see docs/PARITY.md OCR)", p.name, gv, p.jsFile, jv)
		}
	}

	// The pitch factor's *meaning* is half the contract, and equal numbers do not carry it: a side
	// that multiplied 1.2 by the height of the recognized ink box instead of by the line pitch would
	// pass every check above and still group all-caps lettering into different plates - which is the
	// defect this pair of quantities was separated to fix (2026-08-11, synth-balloon-on-panel). So
	// pin the expression that computes the bound and the one that measures a pitch.
	meaning := []struct{ name, file, src, re string }{
		{"pitch bound", "tesseract.go", goSrc, `pitchMax\s*:=\s*float64\(refPitch\)\s*\*\s*ocrClusterPitchFactor`},
		{"pitch bound", "ocr-cluster.js", clusterSrc, `pitchMax\s*=\s*refPitch\s*\*\s*OCR_CLUSTER_PITCH_FACTOR`},
		{"pitch is measured top to top", "tesseract.go", goSrc, `dy\s*:=\s*l\.y0\s*-\s*prev\.y0`},
		{"pitch is measured top to top", "ocr-cluster.js", clusterSrc, `dy\s*=\s*cur\.y0\s*-\s*prev\.y0`},
	}
	for _, m := range meaning {
		if !regexp.MustCompile(m.re).MatchString(m.src) {
			t.Errorf("%s: %s no longer says %q - the clustering gate must compare line pitches on both sides (see docs/PARITY.md OCR)", m.file, m.name, m.re)
		}
	}
}

// TestParityOCRGreyRescue: an image the ordinary colour pass cannot read at all is retried through
// a greyscale rescue ladder, and both editions must climb the same rungs in the same order - a
// reader who sees a comic balloon recognized in the app must see it recognized in the browser too.
// See docs/PARITY.md "OCR" (grey rescue ladder).
func TestParityOCRGreyRescue(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "tesseract.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")

	goLadder := between(goSrc, "var greyRescuePasses = []int{", "}")
	jsLadder := between(jsSrc, "const GREY_RESCUE_PASSES = [", "]")
	rungs := regexp.MustCompile(`(?i)(EngineDefault|LeptonicaOtsu|ENGINE_DEFAULT|LEPTONICA_OTSU)`)
	goRungs := strings.ToUpper(strings.Join(flatten(rungs.FindAllStringSubmatch(goLadder, -1)), ","))
	jsRungs := strings.ToUpper(strings.Join(flatten(rungs.FindAllStringSubmatch(jsLadder, -1)), ","))
	goRungs, jsRungs = strings.ReplaceAll(goRungs, "_", ""), strings.ReplaceAll(jsRungs, "_", "")
	if goRungs == "" || jsRungs == "" {
		t.Fatalf("rescue ladder not parsed (tesseract.go=%q ocr-overlay.js=%q)", goLadder, jsLadder)
	}
	if goRungs != jsRungs {
		t.Errorf("grey rescue ladder drift: tesseract.go=[%s] ocr-overlay.js=[%s] (must match - see docs/PARITY.md OCR)", goRungs, jsRungs)
	}

	// The thresholding method numbers themselves are the shared contract with the engine.
	pairs := []struct{ name, goRe, jsRe string }{
		{"engine-default thresholder", `thresholdEngineDefault\s*=\s*(\d+)`, `THRESHOLD_ENGINE_DEFAULT\s*=\s*"?(\d+)"?`},
		{"leptonica tiled thresholder", `thresholdLeptonicaOtsu\s*=\s*(\d+)`, `THRESHOLD_LEPTONICA_OTSU\s*=\s*"?(\d+)"?`},
		{"rescue line confidence", `ocrRescueLineConf\s*=\s*([\d.]+)`, `OCR_RESCUE_LINE_CONF\s*=\s*([\d.]+)`},
	}
	for _, p := range pairs {
		gv := num(t, p.name+" (tesseract.go)", p.goRe, goSrc)
		jv := num(t, p.name+" (ocr-overlay.js)", p.jsRe, jsSrc)
		if gv != jv {
			t.Errorf("%s drift: tesseract.go=%v ocr-overlay.js=%v (must match - see docs/PARITY.md OCR)", p.name, gv, jv)
		}
	}
}

// TestParityOCRScreenRung: the ladder's last rung measures the halftone screen a picture is
// printed with and low-passes it away with a kernel derived from that measurement. Both editions
// must measure the same way and filter with the same kernel, or a screened comic page recognized
// in the app comes back blank in the browser. See docs/PARITY.md "OCR" (halftone screen rung).
func TestParityOCRScreenRung(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "screen.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-screen.js")

	// Every number the detector and the kernel depend on. The sigma divisor is the parameter the
	// research cycle actually derived; the rest decide whether the rung fires at all, and a drift
	// in any of them changes which images get the pass.
	pairs := []struct{ name, goRe, jsRe string }{
		{"sigma divisor", `ocrScreenSigmaDivisor\s*=\s*([\d.]+)`, `OCR_SCREEN_SIGMA_DIVISOR\s*=\s*([\d.]+)`},
		{"tile size", `ocrScreenTile\s*=\s*([\d.]+)`, `OCR_SCREEN_TILE\s*=\s*([\d.]+)`},
		{"minimum pitch", `ocrScreenMinPitch\s*=\s*([\d.]+)`, `OCR_SCREEN_MIN_PITCH\s*=\s*([\d.]+)`},
		{"maximum pitch", `ocrScreenMaxPitch\s*=\s*([\d.]+)`, `OCR_SCREEN_MAX_PITCH\s*=\s*([\d.]+)`},
		{"tile cap", `ocrScreenMaxTiles\s*=\s*([\d.]+)`, `OCR_SCREEN_MAX_TILES\s*=\s*([\d.]+)`},
		{"minimum tile energy", `ocrScreenMinEnergy\s*=\s*([\d.]+)`, `OCR_SCREEN_MIN_ENERGY\s*=\s*([\d.]+)`},
		{"autocorrelation peak floor", `ocrScreenPeakFloor\s*=\s*([\d.]+)`, `OCR_SCREEN_PEAK_FLOOR\s*=\s*([\d.]+)`},
		{"agreeing-tile share", `ocrScreenTileFrac\s*=\s*([\d.]+)`, `OCR_SCREEN_TILE_FRAC\s*=\s*([\d.]+)`},
		// The additive sweep's two numbers. The first decides which tiles count as screened area the
		// reader is not served on - that is the whole trigger, and a drift changes which pages pay
		// for a second recognition. The second decides which of the sweep's plates may join a page
		// that already read, and a drift there is a duplicate plate painted over lettering that has
		// one, which is visible damage.
		{"served-tile bound", `ocrScreenTileCoverMax\s*=\s*([\d.]+)`, `OCR_SCREEN_TILE_COVER_MAX\s*=\s*([\d.]+)`},
		{"merge overlap bound", `ocrScreenMergeMaxOverlap\s*=\s*([\d.]+)`, `OCR_SCREEN_MERGE_MAX_OVERLAP\s*=\s*([\d.]+)`},
	}
	for _, p := range pairs {
		gv := num(t, p.name+" (screen.go)", p.goRe, goSrc)
		jv := num(t, p.name+" (ocr-screen.js)", p.jsRe, jsSrc)
		if gv != jv {
			t.Errorf("%s drift: screen.go=%v ocr-screen.js=%v (must match - see docs/PARITY.md OCR)", p.name, gv, jv)
		}
	}

	// The rung's *position* is as much the invariant as its numbers: it costs accuracy on lettering
	// the cheaper rungs already read, so it must stay after the grey ladder rather than in front of
	// it. Both editions state that by calling it only once the ladder has returned nothing.
	position := []struct{ name, file, src, re string }{
		{"screen rung follows the grey ladder", "tesseract.go",
			readRepoFile(t, "internal", "ocr", "tesseract.go"),
			`(?s)for _, thresholding := range greyRescuePasses.*?return screenRescue\(`},
		{"screen rung follows the grey ladder", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`(?s)for \(const method of GREY_RESCUE_PASSES.*?return screenRescue\(`},
		{"sigma is the measured pitch over the divisor", "tesseract.go",
			readRepoFile(t, "internal", "ocr", "tesseract.go"),
			`float64\(pitch\)\s*/\s*ocrScreenSigmaDivisor`},
		{"sigma is the measured pitch over the divisor", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`blur\(\$\{pitch\s*/\s*OCR_SCREEN_SIGMA_DIVISOR\}px\)`},

		// The additive sweep's position is the same kind of invariant, and a stronger one. The screen
		// pass wins on screened material and loses badly where there is no screen (measured: 16
		// confident words down to 0 on one cover), so a sweep moved in front of the ordinary pass, or
		// applied unconditionally, is exactly the regression the strategic spec rejected. Both
		// editions must run it on the branch where the ordinary pass *found* plates.
		{"sweep runs on the branch that already read", "tesseract.go",
			readRepoFile(t, "internal", "ocr", "tesseract.go"),
			`(?s)if len\(res\.Blocks\) == 0 \{.*?\} else \{\s*res\.Blocks = screenSweep\(`},
		{"sweep runs on the branch that already read", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`if \(!blocks\.length\) blocks = await greyRescue\(.*\n\s*else blocks = await screenSweep\(`},

		// And its trigger: the sweep must ask the detector the narrower question - is there screened
		// area no plate covers - or it would spend a whole recognition on a page whose screened part
		// is already plated.
		{"sweep measures only outside the plates", "tesseract.go",
			readRepoFile(t, "internal", "ocr", "tesseract.go"),
			`screenPitchOutside\(grey, blockRects\(kept\)\)`},
		{"sweep measures only outside the plates", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`(?s)measureScreenPitch\(image, covered\).*?screenPitch\(grey, width, height, covered\)`},

		// And that what it finds is merged rather than substituted, which is what keeps every plate
		// the ordinary pass produced.
		{"sweep merges rather than replaces", "tesseract.go",
			readRepoFile(t, "internal", "ocr", "tesseract.go"),
			`return mergeScreenBlocks\(kept, res\.Blocks\)`},
		{"sweep merges rather than replaces", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`return mergeScreenBlocks\(kept, clusterLines\(`},
	}
	for _, p := range position {
		if !regexp.MustCompile(p.re).MatchString(p.src) {
			t.Errorf("%s: %s no longer holds (%q) - see docs/PARITY.md OCR (halftone screen rung)", p.file, p.name, p.re)
		}
	}
}

// flatten takes the first capture group of every regexp match.
func flatten(matches [][]string) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
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

// TestParityOCRLangReport: a pass that recognized nothing must name the language data it used,
// on both editions. "No text found" is true about the data that was loaded and reads as a verdict
// on the picture - the wrong lesson when an English recognizer was pointed at a Russian page, which
// is the default on both sides. The label is the shared piece, so a rename or a one-sided removal
// fails here; the trigger differs on purpose (see docs/PARITY.md "OCR").
func TestParityOCRLangReport(t *testing.T) {
	goLabel := readRepoFile(t, "internal", "ocr", "tessdata.go")
	if !strings.Contains(goLabel, "func LangLabel(") {
		t.Error("tessdata.go: LangLabel is gone - the desktop can no longer name its language data")
	}
	jsLabel := readRepoFile(t, "extension", "src", "ocr-lang.js")
	if !strings.Contains(jsLabel, "export function langLabel(") {
		t.Error("ocr-lang.js: langLabel is gone - the extension can no longer name its language data")
	}
	// Same shape on both sides: the code first (it is what the user types or picks back), the
	// catalog name in brackets. A one-sided reshuffle to "English (eng)" fails here.
	if !strings.Contains(goLabel, `c += " (" + name + ")"`) {
		t.Error(`tessdata.go: LangLabel no longer renders "code (Name)" (see docs/PARITY.md OCR)`)
	}
	if !strings.Contains(jsLabel, "`${c} (${known.name})`") {
		t.Error("ocr-lang.js: langLabel no longer renders \"code (Name)\" (see docs/PARITY.md OCR)")
	}
	// And it has to reach the reader on both sides, not just exist.
	if !strings.Contains(readRepoFile(t, "internal", "pipeline", "pipeline.go"), "ocr.LangLabel(") {
		t.Error("pipeline.go: the overlay report no longer names the language")
	}
	for _, f := range []string{"viewer.js", "ocr.js"} {
		if !strings.Contains(readRepoFile(t, "extension", "src", f), "ocrNoTextLang") {
			t.Errorf("%s: an empty OCR result no longer names the language", f)
		}
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

// TestParityReportFields: the labels the desktop archive's environment.txt and the extension's
// clipboard summary have in common must be spelled identically, or the author ends up reading
// two report formats. See docs/PARITY.md "Report field labels".
func TestParityReportFields(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "report", "environment.go")
	jsSrc := readRepoFile(t, "extension", "src", "diagnostics.js")
	// The label is matched with its opening delimiter, so "ocr" does not pass on the strength
	// of "ocr languages".
	for _, label := range []string{"edition", "version", "platform", "interface language", "ocr"} {
		if !strings.Contains(goSrc, `{"`+label+`",`) {
			t.Errorf("environment.go: shared report label %q is gone (see docs/PARITY.md)", label)
		}
		if !strings.Contains(jsSrc, `["`+label+`",`) {
			t.Errorf("diagnostics.js: shared report label %q is gone (see docs/PARITY.md)", label)
		}
	}
}

// TestParityOCRLabEvidenceSchema: the OCR lab grades both editions with one metrics package, so
// the desktop runner and the extension runner must describe a run in the same terms. This reads
// the contract out of both implementations - the schema version, the Plate field names and the
// translation-stress table - and fails when one side moves alone. See docs/PARITY.md "OCR lab
// evidence schema".
func TestParityOCRLabEvidenceSchema(t *testing.T) {
	goSrc := readRepoFile(t, "tools", "ocrlab", "evidence", "evidence.go")
	jsSrc := readRepoFile(t, "extension", "scripts", "_ocrlab-evidence.mjs")

	goVersion := regexp.MustCompile(`SchemaVersion = (\d+)`).FindStringSubmatch(goSrc)
	jsVersion := regexp.MustCompile(`export const SCHEMA_VERSION = (\d+);`).FindStringSubmatch(jsSrc)
	if goVersion == nil || jsVersion == nil {
		t.Fatalf("schema version not found (evidence.go=%v _ocrlab-evidence.mjs=%v)", goVersion != nil, jsVersion != nil)
	}
	if goVersion[1] != jsVersion[1] {
		t.Errorf("evidence schema version drift: evidence.go=%s _ocrlab-evidence.mjs=%s - bump both or neither",
			goVersion[1], jsVersion[1])
	}

	// The Go Plate's JSON tags in declaration order, against the object makePlate returns. Order
	// matters as well as membership: the two editions' evidence files are meant to diff cleanly.
	goPlate := jsonTags(between(goSrc, "type Plate struct {", "\n}"))
	jsPlate := objectKeys(between(jsSrc, "export function makePlate(p = {}) {\n  return {", "\n  };"))
	if strings.Join(goPlate, ",") != strings.Join(jsPlate, ",") {
		t.Errorf("Plate field drift:\n  evidence.go          : %v\n  _ocrlab-evidence.mjs : %v", goPlate, jsPlate)
	}

	// The stress table is a shared constant in the same sense: a case that exists on one side only
	// silently drops a column out of half the report.
	goStress := quoted(t, `Name: "([^"]+)"`, readRepoFile(t, "tools", "ocrlab", "runner", "stress.go"))
	runner := readRepoFile(t, "extension", "scripts", "ocrlab.mjs")
	jsStress := quoted(t, `name: "([^"]+)"`, between(runner, "const STRESS_CASES = [", "\n];"))
	if strings.Join(goStress, ",") != strings.Join(jsStress, ",") {
		t.Errorf("stress-case drift:\n  stress.go  : %v\n  ocrlab.mjs : %v", goStress, jsStress)
	}
	if len(goStress) != 6 {
		t.Errorf("stress.go declares %d cases, the plan pins six", len(goStress))
	}

	// The clip slack decides what counts as a hidden translation, which is one of the strategic
	// spec's hard gates. Read from both sides rather than compared with a literal here, so the
	// test cannot pass on a stale copy of the number.
	goSlack := regexp.MustCompile(`ClipSlackPx = (\d+)`).FindStringSubmatch(goSrc)
	jsSlack := regexp.MustCompile(`export const CLIP_SLACK_PX = (\d+);`).FindStringSubmatch(jsSrc)
	if goSlack == nil || jsSlack == nil {
		t.Fatalf("clip slack not found (evidence.go=%v _ocrlab-evidence.mjs=%v)", goSlack != nil, jsSlack != nil)
	}
	if goSlack[1] != jsSlack[1] {
		t.Errorf("clip slack drift: evidence.go=%s _ocrlab-evidence.mjs=%s", goSlack[1], jsSlack[1])
	}
}

// jsonTags returns the `json:"name"` tags of a struct body, in declaration order, dropping any
// option suffix so `error,omitempty` compares as `error`.
func jsonTags(body string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`json:"([^"]+)"`).FindAllStringSubmatch(body, -1) {
		out = append(out, strings.Split(m[1], ",")[0])
	}
	return out
}

// objectKeys returns the keys of a JS object literal body, in source order.
func objectKeys(body string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`(?m)^\s{4}([A-Za-z][A-Za-z0-9]*):`).FindAllStringSubmatch(body, -1) {
		out = append(out, m[1])
	}
	return out
}

// quoted returns the first capture of every match, in source order.
func quoted(t *testing.T, pattern, s string) []string {
	t.Helper()
	ms := regexp.MustCompile(pattern).FindAllStringSubmatch(s, -1)
	if len(ms) == 0 {
		t.Fatalf("no match for %q - the source shape changed", pattern)
	}
	var out []string
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
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
