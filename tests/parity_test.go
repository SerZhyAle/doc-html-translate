package tests

// Cross-edition parity guards. These tests parse the two independent codebases (the Go
// app and the JS extension) and assert the values that must stay identical actually do.
// A change on one side that is not mirrored on the other fails here. The pinned contracts
// live in docs/PARITY.md; keep this test, that doc, and the code in step.

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/internal/appearance"
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
	// Line endings are normalized because these tests match multi-line markers with "\n" in them,
	// and the same file is LF in the repository and CRLF in a Windows working copy. Without this a
	// parity check passes in CI and fails on a developer's machine - or, as happened here, starts
	// failing the moment a rebase re-checks-out the file, reporting a schema drift that is not one.
	return strings.ReplaceAll(string(b), "\r\n", "\n")
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
	boundarySrc := readRepoFile(t, "internal", "ocr", "boundary.go")
	overlaySrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	clusterSrc := readRepoFile(t, "extension", "src", "ocr-cluster.js")
	pairs := []struct{ name, goRe, jsFile, jsRe string }{
		{"min line confidence", `ocrMinLineConf\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MIN_LINE_CONF\s*=\s*([\d.]+)`},
		{"cluster pitch factor", `ocrClusterPitchFactor\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_CLUSTER_PITCH_FACTOR\s*=\s*([\d.]+)`},
		{"max leading ratio", `ocrMaxLeadingRatio\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MAX_LEADING_RATIO\s*=\s*([\d.]+)`},
		{"type size ratio", `ocrTypeSizeRatio\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_TYPE_SIZE_RATIO\s*=\s*([\d.]+)`},
		{"max plate coverage", `ocrMaxPlateCoverage\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MAX_PLATE_COVERAGE\s*=\s*([\d.]+)`},
		{"min plate line fill", `ocrMinPlateLineFill\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MIN_PLATE_LINE_FILL\s*=\s*([\d.]+)`},
		{"max word gap ratio", `ocrMaxWordGapRatio\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_MAX_WORD_GAP_RATIO\s*=\s*([\d.]+)`},
		{"boundary reach", `ocrBoundaryReach\s*=\s*([\d.]+)`, "ocr-cluster.js", `OCR_BOUNDARY_REACH\s*=\s*([\d.]+)`},
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
		// The type-size break is the same kind of contract and was given the same treatment after the
		// colour fix drifted in exactly this way (2026-08-13): equal constants said nothing about what
		// the constant was applied to. Two things have to hold on both sides - the candidate line is
		// weighed against its own *cluster's* median height (weighing it against the page's would make
		// every plate on a two-size page break), and the ratio multiplies the smaller of the two, so
		// the test is symmetric and a small line under a big cluster breaks like a big one under a
		// small cluster.
		{"size break asks the cluster, not the page", "tesseract.go", goSrc, `sameTypeSize\(l\.inkHeight\(\),\s*median\(cink,\s*0\)\)`},
		{"size break asks the cluster, not the page", "ocr-cluster.js", clusterSrc, `sameTypeSize\(lineInkHeight\(l\),\s*medianOf\(cur\.ink\)\)`},
		// And the quantity it compares is the median of the line's *words*, not the line box. The box
		// is their union, so one tall artefact - a balloon outline read as "|" - sets it for the whole
		// line and the ratio then sees two type sizes where a reader sees one (measured in the
		// extension edition on synth-adjacent-balloons, 2026-08-15). A side that reverted to the box
		// would pass every constant check above and split balloons again.
		{"type size is the words' median, not the line box", "tesseract.go", goSrc, `func \(l \*ocrLine\) inkHeight\(\) int \{[^}]*median\(l\.wordH, 0\)`},
		{"type size is the words' median, not the line box", "ocr-cluster.js", clusterSrc, `function lineInkHeight\(l\) \{\s*const h = medianOf\(\(l\.wordH \|\| \[\]\)`},
		// And the artefact is kept out of the line's own box, so the plate drawn from it cannot reach
		// past the lettering onto protected artwork. Both conditions are part of the invariant: only a
		// token with no letter or digit may go, and only when it is taller than the ratio allows -
		// either half alone would delete real words or real punctuation.
		{"a tall non-text token is trimmed from the line box", "tesseract.go", goSrc, `!hasLetterOrDigit\(w\.text\) && float64\(h\) > float64\(med\)\*ocrTypeSizeRatio`},
		{"a tall non-text token is trimmed from the line box", "ocr-cluster.js", clusterSrc, `if \(/\[\\p\{L\}\\p\{N\}\]/u\.test\(w\.text \|\| ""\)\) return true;[\s\S]{0,120}?med \* OCR_TYPE_SIZE_RATIO`},
		{"size break is symmetric", "tesseract.go", goSrc, `float64\(max\(h, clusterH\)\)\s*<=\s*float64\(min\(h, clusterH\)\)\s*\*\s*ocrTypeSizeRatio`},
		{"size break is symmetric", "ocr-cluster.js", clusterSrc, `Math\.max\(h, clusterH\)\s*<=\s*Math\.min\(h, clusterH\)\s*\*\s*OCR_TYPE_SIZE_RATIO`},
		// The coverage rule needs both of its conditions on both sides, and needs to release rather
		// than drop. Either half alone releases a scene the corpus says is one plate (size alone
		// takes samson-and-delilah-03-scroll, looseness alone takes synth-uniform-paper), and a side
		// that dropped the cluster instead of releasing its lines would lose text that reads today
		// while passing every constant check above.
		{"coverage is measured against the page", "tesseract.go", goSrc, `box <= float64\(imgW\)\*float64\(imgH\)\*ocrMaxPlateCoverage`},
		{"coverage is measured against the page", "ocr-cluster.js", clusterSrc, `box <= imgW \* imgH \* OCR_MAX_PLATE_COVERAGE`},
		{"looseness is the second condition", "tesseract.go", goSrc, `float64\(ink\) >= float64\(boxH\)\*ocrMinPlateLineFill`},
		{"looseness is the second condition", "ocr-cluster.js", clusterSrc, `ink >= boxH \* OCR_MIN_PLATE_LINE_FILL`},
		// The word-gap split runs before every rule above and repairs their input, so equal constants
		// are again only half the contract. Three things have to hold on both sides. The gap is
		// measured between the two word boxes and not left-to-right, or a right-to-left line never
		// cuts; it is weighed against the median of the line's own *word* heights, not the line box,
		// for the same reason inkHeight is; and each run is boxed to its own words, because handing
		// both halves the stitch's box back leaves the clustering exactly where it started.
		{"the gap is measured between the boxes", "tesseract.go", goSrc, `max\(w\.x0-prev\.x1, prev\.x0-w\.x1\)\) > maxGap`},
		{"the gap is measured between the boxes", "ocr-cluster.js", clusterSrc, `Math\.max\(at\(w\.bbox\.x0\) - at\(prev\.x1\), at\(prev\.x0\) - at\(w\.bbox\.x1\)\)`},
		{"the gap is weighed against the words' median", "tesseract.go", goSrc, `maxGap := float64\(med\) \* ocrMaxWordGapRatio`},
		{"the gap is weighed against the words' median", "ocr-cluster.js", clusterSrc, `const maxGap = med \* OCR_MAX_WORD_GAP_RATIO`},
		{"a cut run is boxed to its own words", "tesseract.go", goSrc, `func lineFromWords\(words \[\]ocrWord\) \*ocrLine`},
		{"a cut run is boxed to its own words", "ocr-overlay.js", overlaySrc, `const unionOf = \(words\) =>`},
		// And the cut runs are put back in reading order, column by column. clusterLines closes a
		// plate on the first line that does not belong to it, so without this the split trades one
		// oversized plate for three fragments (measured on test_doc/1.png, 2026-09-12). The columns
		// are formed from the lines that clear the confidence floor, on both sides: a full-page noise
		// "line" that never reaches a plate would otherwise chain every column into one, which sorts
		// the page straight back into the order this exists to undo.
		{"cut runs are regrouped into columns", "tesseract.go", goSrc, `func orderColumns\(runs \[\]\*ocrLine, minConf float64\) \[\]\*ocrLine`},
		{"cut runs are regrouped into columns", "ocr-cluster.js", clusterSrc, `export function orderColumns\(runs, minConf = OCR_MIN_LINE_CONF\)`},
		{"columns are formed from the lines that can reach a plate", "tesseract.go", goSrc, `if keepLine\(r, minConf\) && !r\.orphan \{\s*byX = append\(byX, r\)`},
		{"columns are formed from the lines that can reach a plate", "ocr-cluster.js", clusterSrc, `runs\.filter\(\(r\) => keepLine\(r, minConf\) && !r\.orphan\)\.sort`},
		// The stroke between two words is the other half of the same cut (Phase 07 Step 07.3 of the
		// lab ticket, 2026-09-25), and again the constant is the least of it. A gap is cut when it is
		// too wide *or* a stroke crosses it; the reach is the words' median height times the constant,
		// in the recognizer's own pixels; the path has to run past the band on both sides, which is
		// the whole difference between a balloon outline and a letter left out of its box; ink is the
		// plate colours' own contrast from a paper read outside the word boxes; and every pass reads
		// the plane of the picture itself. A side that dropped any one of these would pass the value
		// check above and cut real lines, or none.
		{"a stroke cuts a gap the ratio keeps", "tesseract.go", goSrc, `wide := float64\(max\(w\.x0-prev\.x1, prev\.x0-w\.x1\)\) > maxGap\s*if wide \|\| strokeBetween\(ink, prev, w, reach\)`},
		{"a stroke cuts a gap the ratio keeps", "ocr-cluster.js", clusterSrc, `const wide = gap > maxGap;\s*if \(wide \|\| strokeBetween\(ink, prev, w\.bbox, reach\)\)`},
		// And what a stroke cuts off is not always text: a run it separates that could not be a plate
		// on its own is the outline or the artwork beside it, and both sides park it rather than let it
		// sit in a column between two lines of one balloon (atomicwar0401, 2026-09-25).
		{"a stroke's untranslatable fragment is an orphan", "tesseract.go", goSrc, `byStroke\[i-1\] \|\| i < len\(byStroke\) && byStroke\[i\]\) && !isTranslatable\(`},
		{"a stroke's untranslatable fragment is an orphan", "ocr-cluster.js", clusterSrc, `byStroke\[i - 1\]\) \|\| \(i < byStroke\.length && byStroke\[i\]\)\) && !isTranslatable\(`},
		{"an orphan reaches the clustering as a line", "ocr-overlay.js", overlaySrc, `if \(words\.orphan === true\) line\.orphan = true;`},
		{"the reach is the words' median times the constant", "tesseract.go", goSrc, `reach := int\(float64\(med\) \* ocrBoundaryReach\)`},
		{"the reach is the words' median times the constant", "ocr-cluster.js", clusterSrc, `const reach = Math\.floor\(rawMed \* OCR_BOUNDARY_REACH\)`},
		{"the stroke runs past the band on both sides", "boundary.go", boundarySrc, `y0, y1 := min\(a\.y0, b\.y0\)-reach, max\(a\.y1, b\.y1\)\+reach`},
		{"the stroke runs past the band on both sides", "ocr-cluster.js", clusterSrc, `const y0 = Math\.min\(a\.y0, b\.y0\) - reach, y1 = Math\.max\(a\.y1, b\.y1\) \+ reach`},
		{"the path must reach the far row", "boundary.go", boundarySrc, `if py == h-1 \{\s*return true`},
		{"the path must reach the far row", "ocr-cluster.js", clusterSrc, `if \(py === h - 1\) return true;`},
		{"ink is the plate contrast from the paper", "boundary.go", boundarySrc, `d >= plateMinContrast \|\| -d >= plateMinContrast`},
		{"ink is the plate contrast from the paper", "ocr-cluster.js", clusterSrc, `- paper\) >= ink\.minContrast`},
		{"ink is the plate contrast from the paper", "ocr-overlay.js", overlaySrc, `minContrast: PLATE_MIN_CONTRAST`},
		{"paper is read outside the word boxes", "boundary.go", boundarySrc, `\{w\.y0 - 3, w\.y0 - 2, w\.y1 \+ 1, w\.y1 \+ 2\}`},
		{"paper is read outside the word boxes", "ocr-cluster.js", clusterSrc, `\[w\.y0 - 3, w\.y0 - 2, w\.y1 \+ 1, w\.y1 \+ 2\]`},
		{"every pass reads the picture's own plane", "tesseract.go", goSrc, `ocrMinLineConf, frame\.grey\)`},
		{"every pass reads the picture's own plane", "ocr-overlay.js", overlaySrc, `const ink = await strokePlane\(image\);`},
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
	// Both extension files, because the ladder is in ocr-overlay.js while the gate it applies at
	// each rung - keepLine and its three constants - is in ocr-cluster.js beside the other floors.
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js") +
		"\n" + readRepoFile(t, "extension", "src", "ocr-cluster.js")

	goLadder := between(goSrc, "var greyRescuePasses = []rescueRung{", "\n}")
	jsLadder := between(jsSrc, "const GREY_RESCUE_PASSES = [", "\n]")
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
		// The rescue floor lives in ocr-cluster.js on the JS side, beside keepLine which applies it,
		// so this pair reads it from the combined extension source above.
		{"rescue line confidence", `ocrRescueLineConf\s*=\s*([\d.]+)`, `OCR_RESCUE_LINE_CONF\s*=\s*([\d.]+)`},
		// The sparse rung's mode. It is the rung that recovers display lettering a poster's layout
		// analysis throws away, and a drift here means one edition reads the poster and the other
		// shows the reader a picture with nothing on it.
		{"sparse segmentation mode", `ocrSparsePageSegMode\s*=\s*(\d+)`, `OCR_SPARSE_PSM\s*=\s*"?(\d+)"?`},
	}
	for _, p := range pairs {
		gv := num(t, p.name+" (tesseract.go)", p.goRe, goSrc)
		jv := num(t, p.name+" (ocr-overlay.js)", p.jsRe, jsSrc)
		if gv != jv {
			t.Errorf("%s drift: tesseract.go=%v ocr-overlay.js=%v (must match - see docs/PARITY.md OCR)", p.name, gv, jv)
		}
	}
}

// TestParityOCRRungComparator: a rescue rung may replace what is already in hand only when it found
// strictly more, and a tie keeps the incumbent. Both editions have to apply the same rule or the
// same poster comes back with one word in one edition and six lines in the other. See docs/PARITY.md
// "OCR" (grey rescue ladder).
func TestParityOCRRungComparator(t *testing.T) {
	checks := []struct{ name, file, src, re string }{
		{"the ladder keeps the strongest rung", "tesseract.go",
			readRepoFile(t, "internal", "ocr", "tesseract.go"),
			`(?s)for _, rung := range greyRescuePasses.*?strictlyBetter\(res, best\)`},
		{"the ladder keeps the strongest rung", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`(?s)for \(const rung of GREY_RESCUE_PASSES.*?strictlyBetter\(blocks, best\)`},
		{"a tie keeps the incumbent", "strength.go",
			readRepoFile(t, "internal", "ocr", "strength.go"),
			`return resultStrength\(candidate\) > resultStrength\(current\)`},
		{"a tie keeps the incumbent", "ocr-cluster.js",
			readRepoFile(t, "extension", "src", "ocr-cluster.js"),
			`return resultStrength\(candidate\) > resultStrength\(current\)`},
	}
	for _, c := range checks {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: %s no longer holds (%q) - see docs/PARITY.md OCR (grey rescue ladder)", c.file, c.name, c.re)
		}
	}
}

// TestParityOCRPlateColourOrientation: a plate must never come back as the negative of what it
// covers. Both editions decide which of the two sampled colours is the paper from the band around
// the block rather than from which colour fills more of it - heavy display capitals fill more of
// their own tight box than the paper between them does. See docs/PARITY.md "OCR" (plate colours).
func TestParityOCRPlateColourOrientation(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "overlay.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")

	if gv, jv := num(t, "ring sample floor (overlay.go)", `ringMinSamples\s*=\s*(\d+)`, goSrc),
		num(t, "ring sample floor (ocr-overlay.js)", `RING_MIN_SAMPLES\s*=\s*(\d+)`, jsSrc); gv != jv {
		t.Errorf("ring sample floor drift: overlay.go=%v ocr-overlay.js=%v (must match - see docs/PARITY.md OCR)", gv, jv)
	}
	// The band's own width, which the call shape below cannot see. It drifted once: the JS side
	// folded the line height and the 1.3-line ink strip into one variable and handed the strip to
	// the ring, so the extension sampled a band 30 % wider than the desktop app on the same block.
	// Both sides derive the pad from the raw line height, and this is what says so.
	// Its value is pinned with the other colour numbers in TestParityOCRPlateColourNumbers; what is
	// pinned here is the shape - derived from lh and floored on both sides.
	for _, c := range []struct{ file, src, re string }{
		{"overlay.go", goSrc, `pad := max\(lh/ringPadDivisor, ringMinPad\)`},
		{"ocr-overlay.js", jsSrc, `const pad = Math\.max\(RING_MIN_PAD, Math\.floor\(lh / RING_PAD_DIVISOR\)\)`},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: the ring band is no longer the floored line height over the divisor (%q) - see docs/PARITY.md OCR", c.file, c.re)
		}
	}
	for _, c := range []struct{ name, file, src, re string }{
		{"the ring decides which colour is paper", "overlay.go", goSrc, `if ringNearerInk\(img, x0, y0, x1, y1, lh, bgR, bgG, bgB, inkR, inkG, inkB\)`},
		{"the ring decides which colour is paper", "ocr-overlay.js", jsSrc, `ringNearerInk\(ctx, x0, y0, w, h, lh, bg, ink\)`},
		// The ring is derived from the line, the ink from the 1.3-line strip; one variable for both
		// is the drift above, so each edition has to keep them apart by name.
		{"the ink strip is not the ring's line height", "overlay.go", goSrc, `yFirst := y0 \+ int\(float64\(lh\)\*inkStripLines\)`},
		{"the ink strip is not the ring's line height", "ocr-overlay.js", jsSrc, `const firstBand = .*Math\.floor\(lh \* INK_STRIP_LINES\)`},
		// Ink is a median on both sides. A mean lands between the ink and the paper by construction.
		{"ink is a median", "overlay.go", goSrc, `inkR, inkG, inkB = medianOf\(ir\), medianOf\(ig\), medianOf\(ib\)`},
		{"ink is a median", "ocr-overlay.js", jsSrc, `\[medianOf\(ir\), medianOf\(ig\), medianOf\(ib\)\]`},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: %s no longer holds (%q) - see docs/PARITY.md OCR (plate colours)", c.file, c.name, c.re)
		}
	}
}

// TestParityOCRPlateColourNumbers pins the colour sampling's numbers across editions: the ink
// deviation, the strip, the minimum ink share, the contrast floor, the ring band and the fallback
// colours. They are policy rather than measured values (OCR-OVERLAY rule 13 is ticket 21's open
// question 5), but a one-sided change is still a visible difference in plate colour, and until
// 2026-09-25 the ring band was rounded in the extension and floored on the desktop. The derived
// counts are floored on both sides, and luma is truncated on both, which is pinned too.
func TestParityOCRPlateColourNumbers(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "overlay.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	want := map[string]float64{
		"inkDeviationMin": 90, "inkStripLines": 1.3, "inkMinPerMille": 15, "inkMinSamples": 6,
		"plateMinContrast": 55, "ringPadDivisor": 3, "ringMinPad": 2,
		"fallbackLumaSplit": 140, "fallbackDarkInk": 17, "fallbackLightInk": 240,
	}
	for name, v := range want {
		jsName := strings.ToUpper(regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, "${1}_${2}"))
		gv := num(t, name+" (overlay.go)", `\b`+name+`\s*=\s*([\d.]+)`, goSrc)
		jv := num(t, jsName+" (ocr-overlay.js)", `const `+jsName+` = ([\d.]+);`, jsSrc)
		if gv != jv || gv != v {
			t.Errorf("%s drift: overlay.go=%v ocr-overlay.js=%v, pinned %v (must match - see docs/PARITY.md OCR, plate colours)", name, gv, jv, v)
		}
	}
	for _, c := range []struct{ file, src, re, what string }{
		{"overlay.go", goSrc, `minInk := max\(len\(fr\)\*inkMinPerMille/1000, inkMinSamples\)`, "the ink share is an integer count"},
		{"ocr-overlay.js", jsSrc, `Math\.max\(INK_MIN_SAMPLES, Math\.floor\(first\.rs\.length \* INK_MIN_PER_MILLE / 1000\)\)`, "the ink share is an integer count"},
		{"overlay.go", goSrc, `func luma\(r, g, b int\) int \{ return \(299\*r \+ 587\*g \+ 114\*b\) / 1000 \}`, "luma is truncated"},
		{"ocr-overlay.js", jsSrc, `const luma = \(r, g, b\) => Math\.floor\(\(299 \* r \+ 587 \* g \+ 114 \* b\) / 1000\);`, "luma is truncated"},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: %s no longer holds (%q) - see docs/PARITY.md OCR, plate colours", c.file, c.what, c.re)
		}
	}
}

// TestParityOCRFitLadder pins the runtime re-fit's numbers, which live in the desktop's inlined
// ocrScript and in ocr-plates.js fitPlate and were held by nothing but the grow cap: the shrink
// floor, step and iteration bound, the grow step and bound, and the overflow slack. A one-sided
// change is a one-sided change to how large translated text renders.
func TestParityOCRFitLadder(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "overlay.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-plates.js")
	for _, c := range []struct {
		name string
		want float64
		goRe string
		jsRe string
	}{
		{"shrink floor (share of base)", 0.5, `floor=base\*([\d.]+)`, `const floor = base \* ([\d.]+);`},
		{"shrink iterations", 40, `&&g<(\d+)\)`, `&& g < (\d+)\)`},
		{"shrink minimum step (cqw)", 0.3, `s-=Math\.max\(([\d.]+),s\*`, `s -= Math\.max\(([\d.]+), s \*`},
		{"shrink step (share)", 0.08, `s-=Math\.max\([\d.]+,s\*([\d.]+)\)`, `s -= Math\.max\([\d.]+, s \* ([\d.]+)\)`},
		{"grow iterations", 20, `&&gg<(\d+)\)`, `&& gg < (\d+)\)`},
		{"grow minimum step (cqw)", 0.3, `n\+Math\.max\(([\d.]+),n\*`, `n \+ Math\.max\(([\d.]+), n \*`},
		{"grow step (share)", 0.04, `n\+Math\.max\([\d.]+,n\*([\d.]+)\)`, `n \+ Math\.max\([\d.]+, n \* ([\d.]+)\)`},
		{"overflow slack (px)", 1, `scrollHeight>b\.clientHeight\+(\d+)`, `scrollHeight > b\.clientHeight \+ (\d+)`},
	} {
		gv := num(t, c.name+" (overlay.go ocrScript)", c.goRe, goSrc)
		jv := num(t, c.name+" (ocr-plates.js fitPlate)", c.jsRe, jsSrc)
		if gv != jv || gv != c.want {
			t.Errorf("fit ladder %s drift: overlay.go=%v ocr-plates.js=%v, pinned %v (see docs/PARITY.md OCR)", c.name, gv, jv, c.want)
		}
	}
}

// TestParityOCRColumnTest pins the column test both clustering stages share: two lines are one
// column when their horizontal overlap is at least a tenth of the narrower one. It is written as
// `overlap*10` in four places per edition (splitWideGaps' column check, orderColumns, clusterLines,
// medianLinePitch), so every occurrence is read rather than the first.
func TestParityOCRColumnTest(t *testing.T) {
	re := regexp.MustCompile(`overlap\s*\*\s*(\d+)\s*(>=|<)\s*narrower`)
	count := func(file, src string) {
		ms := re.FindAllStringSubmatch(src, -1)
		if len(ms) < 3 {
			t.Errorf("%s: found %d column tests, want the clustering's own at least - the parse is wrong", file, len(ms))
		}
		for _, m := range ms {
			if m[1] != "10" {
				t.Errorf("%s: a column test uses overlap*%s, want overlap*10 (0.1 of the narrower line) - see docs/PARITY.md OCR", file, m[1])
			}
		}
	}
	count("tesseract.go", readRepoFile(t, "internal", "ocr", "tesseract.go"))
	count("ocr-cluster.js", readRepoFile(t, "extension", "src", "ocr-cluster.js"))
}

// TestParityOCRDroppedLines: the confidence floor is the one place the overlay decides against
// words the recognizer did read, and both editions must record that decision rather than let it
// look like "nothing was recognized". The record is diagnostics only - it never reaches a page and
// never weighs a rung - but it is what any re-derivation of the floor is measured from, so an
// edition that stopped recording it would be measured on the other one's evidence. See
// docs/PARITY.md "OCR" (the confidence floor and its record).
func TestParityOCRDroppedLines(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "tesseract.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-cluster.js")

	for _, c := range []struct{ file, src, re, what string }{
		{"tesseract.go", goSrc, `func keepLine\(l \*ocrLine, minConf float64\) bool`,
			"the floor is one predicate"},
		{"tesseract.go", goSrc, `res\.Dropped = append\(res\.Dropped, lineDrop\(l, minConf, gateConfidence\)\)`,
			"the rejected lines are recorded"},
		{"ocr-cluster.js", jsSrc, `export function keepLine\(l, minConf`,
			"the floor is one predicate"},
		{"ocr-cluster.js", jsSrc, `export function droppedLines\(lines, minConf`,
			"the rejected lines are recorded"},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: %s no longer holds (%q) - see docs/PARITY.md OCR", c.file, c.what, c.re)
		}
	}

	// One predicate, applied by both the keep and the record. Two copies of `conf >= floor` would
	// pass every check above and still let the record describe a decision that is no longer taken.
	if !regexp.MustCompile(`if !keepLine\(l, minConf\) \{`).MatchString(goSrc) {
		t.Error("tesseract.go: clusterLines no longer asks keepLine - the record can drift from the decision")
	}
	if !regexp.MustCompile(`lines\.filter\(\(l\) => keepLine\(l, minConf\)\)`).MatchString(jsSrc) {
		t.Error("ocr-cluster.js: clusterLines no longer asks keepLine - the record can drift from the decision")
	}
}

// TestParityOCRDiscardGates: OCR-OVERLAY rule 12 asks every gate that drops a line to say which one
// it was, so both editions name the same gates with the same words, record each where its decision
// is taken, and merge the passes' records the same way - the ordinary pass's drops followed by the
// ladder's or the sweep's. The record is in display space on both: the desktop scales it back with
// the plates (rule 2). See docs/PARITY.md "OCR" (the confidence floor and its record).
func TestParityOCRDiscardGates(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "tesseract.go")
	goScreen := readRepoFile(t, "internal", "ocr", "screen.go")
	jsCluster := readRepoFile(t, "extension", "src", "ocr-cluster.js")
	jsOverlay := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	jsScreen := readRepoFile(t, "extension", "src", "ocr-screen.js")

	goGates := map[string]string{}
	for _, m := range regexp.MustCompile(`(gate\w+)\s*= "([a-z-]+)"`).FindAllStringSubmatch(goSrc, -1) {
		goGates[strings.ToLower(m[1])] = m[2]
	}
	jsGates := map[string]string{}
	for _, m := range regexp.MustCompile(`export const (GATE_\w+) = "([a-z-]+)";`).FindAllStringSubmatch(jsCluster, -1) {
		jsGates[strings.ToLower(strings.ReplaceAll(m[1], "_", ""))] = m[2]
	}
	if len(goGates) != 3 || !reflect.DeepEqual(goGates, jsGates) {
		t.Errorf("the editions name the discard gates differently:\n go %v\n js %v", goGates, jsGates)
	}

	for _, c := range []struct{ file, src, re, what string }{
		{"tesseract.go", goSrc, `lineDrop\(m, minConf, gateTranslatable\)`, "a refused cluster's lines are recorded"},
		{"ocr-cluster.js", jsCluster, `lineDrop\(m, minConf, GATE_TRANSLATABLE\)`, "a refused cluster's lines are recorded"},
		{"screen.go", goScreen, `rejected = append\(rejected, b\)`, "the merge hands back what it refused"},
		{"ocr-screen.js", jsScreen, `if \(rejected\) rejected\.push\(b\);`, "the merge hands back what it refused"},
		{"tesseract.go", goSrc, `blockDrops\(rejected, ocrRescueLineConf, gateScreenMerge\)`, "the sweep records the merge's refusals"},
		{"ocr-overlay.js", jsOverlay, `gate: GATE_SCREEN_MERGE`, "the sweep records the merge's refusals"},
		{"tesseract.go", goSrc, `res\.Dropped = append\(primary, alt\.Dropped\.\.\.\)`, "primary and ladder drops are merged"},
		{"ocr-overlay.js", jsOverlay, `dropped\.push\(\.\.\.rescued\.dropped\)`, "primary and ladder drops are merged"},
		{"tesseract.go", goSrc, `(?s)func scaleDown\(.*?for i := range res\.Dropped \{`, "the record is scaled back with the plates"},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: %s no longer holds (%q) - see docs/PARITY.md OCR", c.file, c.what, c.re)
		}
	}
}

// TestParityOCRDiscardRecord: both editions write the discard record of OCR-OVERLAY rule 12 as the
// same JSON line, also for an image with no plates. Each edition pins its own output to a literal
// (internal/ocr/diag_test.go goDiagLine, extension/test/ocrlab-evidence.test.mjs GO_DIAG_LINE); this
// holds the two literals equal, so a field renamed on one side fails here rather than in a reader.
// See docs/PARITY.md "OCR" (the confidence floor and its record).
func TestParityOCRDiscardRecord(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "diag_test.go")
	jsSrc := readRepoFile(t, "extension", "test", "ocrlab-evidence.test.mjs")
	goM := regexp.MustCompile("const goDiagLine = `([^`]+)`").FindStringSubmatch(goSrc)
	jsM := regexp.MustCompile(`const GO_DIAG_LINE = '([^']+)';`).FindStringSubmatch(jsSrc)
	if goM == nil || jsM == nil {
		t.Fatalf("discard-record literal not found (go=%v js=%v) - see docs/PARITY.md OCR", goM != nil, jsM != nil)
	}
	if goM[1] != jsM[1] {
		t.Errorf("the editions write different discard records:\n go %s\n js %s", goM[1], jsM[1])
	}
	if !regexp.MustCompile(`recordDiagnostics\(job\.file, r\.res, nil\)`).MatchString(readRepoFile(t, "internal", "ocr", "overlay.go")) {
		t.Error("overlay.go: the no-plate arm no longer writes the discard record (OCR-OVERLAY rule 12)")
	}
	if !regexp.MustCompile(`container\.ocrRecord = \{ width, height, blocks, dropped \}`).MatchString(readRepoFile(t, "extension", "src", "ocr-overlay.js")) {
		t.Error("ocr-overlay.js: overlayImage no longer leaves the discard record for the lab harness")
	}
}

// TestParityOCRPrintPlate: an overlaid page has to survive being printed. A browser drops
// background colours from a printed page by default, and a plate is an opaque background carrying
// text - so without `print-color-adjust:exact` the translation prints on top of the source
// lettering that is still there, and the sheet is unreadable. Print is the one output where the
// reader cannot toggle the overlay off. Both editions derive the plate from internal/appearance and
// TestAppearanceRolesMatchSource holds them to it; this pins that the source keeps the
// declaration. See docs/PARITY.md "OCR" (plate shape).
func TestParityOCRPrintPlate(t *testing.T) {
	// The -webkit- prefix is what Chromium reads, and Chromium is what both editions are printed
	// from. Dropping it is a silent revert on the only browser that matters here.
	for _, p := range []string{"print-color-adjust", "-webkit-print-color-adjust"} {
		if v := plateDecl(t, p); v != "exact" {
			t.Errorf("internal/appearance plate: %s is %q, want exact - the plate no longer prints its paper (docs/PARITY.md OCR, plate shape)", p, v)
		}
	}
}

// plateDecl returns the value the canonical source gives the plate for property, or "".
func plateDecl(t *testing.T, property string) string {
	t.Helper()
	for _, d := range loadAppearance(t).Roles[appearance.RolePlate] {
		if d.Property == property {
			return d.Value
		}
	}
	return ""
}

// TestParityOCRExifOrientation: the two editions must recognize the same picture a reader sees, on
// a file whose EXIF says it is rotated. The desktop app turns the staged copy itself
// (internal/ocr/exif.go); the extension relies on createImageBitmap, whose imageOrientation default
// moved from "none" to "from-image" while the spec settled - so an unnamed option makes the
// agreement hold only for as long as the browser default does. Name it, and keep it named. See
// docs/PARITY.md "OCR" (EXIF orientation).
func TestParityOCRExifOrientation(t *testing.T) {
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	if !regexp.MustCompile(`BITMAP_OPTS\s*=\s*\{\s*imageOrientation:\s*"from-image"\s*\}`).MatchString(jsSrc) {
		t.Error(`ocr-overlay.js: BITMAP_OPTS no longer names imageOrientation: "from-image" - see docs/PARITY.md OCR (EXIF orientation)`)
	}
	// Every decode, not one of them: the recognizer's bitmap, the colour sample and the grey rungs
	// all have to be in the same space as the plates, which are positioned in percent of the
	// displayed picture. A bare call is the drift this test exists for.
	for _, m := range regexp.MustCompile(`createImageBitmap\([^)]*\)`).FindAllString(jsSrc, -1) {
		if !strings.Contains(m, "BITMAP_OPTS") {
			t.Errorf("ocr-overlay.js: %s decodes without BITMAP_OPTS - see docs/PARITY.md OCR (EXIF orientation)", m)
		}
	}
	// The desktop half of the same contract: the staged copy handed to tesseract is turned, so
	// recognition reports coordinates in the space the plates use.
	goSrc := readRepoFile(t, "internal", "ocr", "exif.go")
	if !strings.Contains(goSrc, "func orientImage(") || !strings.Contains(goSrc, "func exifOrientation(") {
		t.Error("exif.go: the desktop edition no longer turns an EXIF-rotated image into display space - see docs/PARITY.md OCR (EXIF orientation)")
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
			`(?s)for _, rung := range greyRescuePasses.*?screenRescue\(`},
		{"screen rung follows the grey ladder", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`(?s)for \(const rung of GREY_RESCUE_PASSES.*?screenRescue\(`},
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
			`(?s)if len\(res\.Blocks\) == 0 \{.*?\} else \{\s*var swept \[\]DroppedLine\s*res\.Blocks, swept = screenSweep\(`},
		{"sweep runs on the branch that already read", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`(?s)if \(!blocks\.length\) \{\s*const rescued = await greyRescue\(.*?\} else \{\s*const swept = await screenSweep\(`},

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
			`merged, rejected := mergeScreenBlocks\(kept, res\.Blocks\)`},
		{"sweep merges rather than replaces", "ocr-overlay.js",
			readRepoFile(t, "extension", "src", "ocr-overlay.js"),
			`const blocks = mergeScreenBlocks\(kept, clusterLines\(`},
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
// match across editions - overlay.go fontFitFactor vs ocr-plates.js FONT_FIT. See docs/PARITY.md
// "OCR" (plate geometry).
//
// The extension's plate half moved to ocr-plates.js so the page-OCR agent can draw plates inside a
// third-party document without carrying the recognition engine in with it; ocr-overlay.js re-exports
// it. The guard follows the code - a constant is pinned where it is defined, not where it used to be.
func TestParityOCRFontFit(t *testing.T) {
	goSrc := readRepoFile(t, "internal", "ocr", "overlay.go")
	jsSrc := readRepoFile(t, "extension", "src", "ocr-overlay.js")
	plateSrc := readRepoFile(t, "extension", "src", "ocr-plates.js")

	gv := num(t, "fontFitFactor (overlay.go)", `fontFitFactor\s*=\s*([\d.]+)`, goSrc)
	jv := num(t, "FONT_FIT (ocr-plates.js)", `FONT_FIT\s*=\s*([\d.]+)`, plateSrc)
	if gv != jv {
		t.Errorf("font fit factor drift: overlay.go=%v ocr-plates.js=%v (must match - see docs/PARITY.md OCR)", gv, jv)
	}

	// The runtime fit runs in both directions, and the ceiling on the grow half is the number that
	// keeps a plate from printing larger than the lettering it covers: a block's box includes the
	// leading between its lines, so "grow until the box is full" is not "match the source". A
	// one-sided change here is a one-sided change to how big translated text renders.
	gc := num(t, "grow cap (overlay.go)", `cap=base\*([\d.]+)`, goSrc)
	jc := num(t, "FONT_GROW_CAP (ocr-plates.js)", `FONT_GROW_CAP\s*=\s*([\d.]+)`, plateSrc)
	if gc != jc {
		t.Errorf("font grow cap drift: overlay.go=%v ocr-plates.js=%v (must match - see docs/PARITY.md OCR)", gc, jc)
	}

	// The plate's ink colour must be sampled the same way on both sides. It is a median rather than
	// a mean because the deviation test that selects ink pixels also admits a glyph's antialiased
	// edge, and averaging that ramp lands between the ink and the paper - measured rgb(61,61,61)
	// for source lettering of rgb(17,17,17). One side reverting to a mean is a visible drift in
	// plate text colour that no constant check would catch.
	if !regexp.MustCompile(`inkR, inkG, inkB = medianOf\(ir\), medianOf\(ig\), medianOf\(ib\)`).MatchString(goSrc) {
		t.Error("overlay.go: the plate ink is no longer a median of the selected pixels - see docs/PARITY.md OCR")
	}
	if !regexp.MustCompile(`ink = measured \? \[medianOf\(ir\), medianOf\(ig\), medianOf\(ib\)\]`).MatchString(jsSrc) {
		t.Error("ocr-overlay.js: the plate ink is no longer a median of the selected pixels - see docs/PARITY.md OCR")
	}

	// The opaque paper is carried by the plate box on both sides, because the box is what covers the
	// source region. Carrying it on an inline span around the string instead gives the paper the
	// shape of the rendered words and leaves the source lettering showing wherever the string is
	// shorter than the region - measured over 46 lab scenes at a mean 93% residual against 17% for
	// the box. A side that moved the background back onto the string would still pass every constant
	// check here, so the carrier is pinned by name on both sides.
	for _, c := range []struct{ what, src, re string }{
		{"overlay.go writes the sampled paper onto the box", goSrc, `style \+= ";background:" \+ paper`},
		{"ocr-plates.js writes the sampled paper onto the plate", plateSrc, `plate\.style\.background = s\.bg`},
	} {
		if !regexp.MustCompile(c.re).MatchString(c.src) {
			t.Errorf("%s: no longer true (%q) - see docs/PARITY.md OCR (plate shape)", c.what, c.re)
		}
	}
	// And the string carrier is gone from both, not merely unused: a leftover .ocr-ink rule would
	// still paint a second, tighter background inside every plate.
	for _, c := range []struct{ what, src string }{
		{"overlay.go still carries a paper span", goSrc},
		{"ocr-overlay.js still carries a paper span", jsSrc},
		{"ocr-plates.js still carries a paper span", plateSrc},
		{"ocr-overlay.css still carries a paper span", readRepoFile(t, "extension", "src", "ocr-overlay.css")},
	} {
		if strings.Contains(c.src, "ocr-ink") {
			t.Errorf("%s: the paper belongs on the plate box - see docs/PARITY.md OCR (plate shape)", c.what)
		}
	}

	// The plate's own CSS - the paper default on the box, and its padding and corner radius, both
	// measured values - is a declaration of the plate role in internal/appearance, which both
	// editions derive from and TestAppearanceRolesMatchSource compares declaration by declaration.
	if v := plateDecl(t, "background"); v != "#fff" {
		t.Errorf("internal/appearance plate: background is %q - the paper belongs on the plate box (docs/PARITY.md OCR, plate shape)", v)
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
	if !strings.Contains(readRepoFile(t, "internal", "pipeline", "ocrstep.go"), "ocr.LangLabel(") {
		t.Error("ocrstep.go: the overlay report no longer names the language")
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

// TestPlateRulesHaveOneImplementation: the whole-page OCR feature draws plates inside a page the
// extension does not own, which is the obvious place for a second, "just for this surface" copy of
// the plate rules to appear - and a copy would drift away from overlay.go without any of the
// constant checks above noticing, because they only compare the two files they know about.
//
// So the page agent is pinned to the shipped unit: it loads ocr-plates.js and calls into it, and it
// does not carry plate arithmetic or plate constants of its own. See docs/PARITY.md "OCR" and
// DEV/plan/done/2026-09-19_page-ocr-overlay.md (done criterion 8).
func TestPlateRulesHaveOneImplementation(t *testing.T) {
	agent := readRepoFile(t, "extension", "src", "page-agent.js")

	for _, want := range []string{"ocr-plates.js", "renderPlates", "scheduleFit"} {
		if !strings.Contains(agent, want) {
			t.Errorf("page-agent.js no longer uses the shipped plate unit (%q missing) - see docs/PARITY.md OCR", want)
		}
	}
	// A constant or a percent-of-the-picture calculation in the agent is the copy this test exists
	// to catch: plate geometry is computed once, by plateSpecs in ocr-plates.js, and travels to the
	// page as finished values.
	for _, banned := range []string{"FONT_FIT", "FONT_GROW_CAP", "lineHeight", "cqw"} {
		if strings.Contains(agent, banned) {
			t.Errorf("page-agent.js computes plate geometry of its own (%q) - the plate rules live in ocr-plates.js (docs/PARITY.md OCR)", banned)
		}
	}
	// And the engine stays out of the reader's document: the agent must not reach for the module
	// that owns the Tesseract worker. See DEV/plan/done/2026-09-19_page-ocr-overlay.md, ADR-1.
	if strings.Contains(agent, "ocr-overlay.js") {
		t.Error("page-agent.js imports ocr-overlay.js, which carries the recognition engine into the reader's page - see DEV/plan/done/2026-09-19_page-ocr-overlay.md ADR-1")
	}
}

// TestPageOcrKeepsTheDeclaredMinimumBrowser: the extension's declared minimum Chrome version is a
// reach commitment - raising it to buy an API drops installed readers, which this project does not
// do. The whole-page OCR feature wants an offscreen document, which is newer than that floor, so
// the broker has to detect it and fall back rather than assume it. A future edit that drops the
// detection would work on the developer's browser and silently do nothing on the oldest supported
// one, which is exactly the failure no manual test catches.
func TestPageOcrKeepsTheDeclaredMinimumBrowser(t *testing.T) {
	manifest := readRepoFile(t, "extension", "manifest.json")
	if !regexp.MustCompile(`"minimum_chrome_version":\s*"105"`).MatchString(manifest) {
		t.Error("manifest.json: minimum_chrome_version moved - a release must not shrink reach (AGENTS.md, SZA canon)")
	}
	broker := readRepoFile(t, "extension", "src", "page-ocr.js")
	if !strings.Contains(broker, "function offscreenAvailable()") {
		t.Error("page-ocr.js: the offscreen host is no longer capability-detected - below the declared minimum browser the feature would silently do nothing")
	}
	if !strings.Contains(broker, "host-frame") {
		t.Error("page-ocr.js: the in-page fallback host is gone - there is nothing left for a browser older than the offscreen API")
	}
}
