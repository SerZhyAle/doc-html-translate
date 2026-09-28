package tests

// OCR-OVERLAY rule 13 (as amended in 1.2) asks every constant of the OCR pipeline to say what it is:
// derived with its report, inherited with its source, or policy with its reason. OCR-PIPELINE
// amendment 1.4 C places that answer where the constant is declared, in both editions, and this test
// is what keeps it there: every constant of the OCR-PIPELINE section 5 status table (amendment 1.2
// item M) must carry a marker comment of the form "OCR-OVERLAY rule 13: <status> - <source>." directly
// above its declaration, and the status must be the one the table gives. A constant that loses its
// marker, or a marker that changes its status without the table changing with it, fails here.
//
// The table below is that section 5 table, name by name. Adding a constant to the catalog's table
// means adding its rows here; changing a status means amending the catalog first
// (docs/contracts/OCR-PIPELINE.md).

import (
	"regexp"
	"strings"
	"testing"
)

var ocrConstantStatus = []struct{ file, name, status string }{
	{"internal/ocr/tesseract.go", "ocrPageSegMode", "policy"},
	{"internal/ocr/tesseract.go", "ocrSparsePageSegMode", "policy"},
	{"internal/ocr/tesseract.go", "ocrRescueLineConf", "derived"},
	{"internal/ocr/tesseract.go", "ocrRescueAnchorConf", "inherited"},
	{"internal/ocr/tesseract.go", "ocrRescueAnchorRun", "inherited"},
	{"internal/ocr/tesseract.go", "ocrRescueAnchorVotes", "derived"},
	{"internal/ocr/tesseract.go", "ocrUpscaleFactor", "policy"},
	{"internal/ocr/tesseract.go", "ocrAssumedPageInches", "policy"},
	{"internal/ocr/tesseract.go", "ocrUpscaleDPIFloor", "policy"},
	{"internal/ocr/tesseract.go", "ocrMinDeclaredDPI", "policy"},
	{"internal/ocr/tesseract.go", "ocrMinLineConf", "policy"},
	{"internal/ocr/tesseract.go", "ocrClusterPitchFactor", "policy"},
	{"internal/ocr/tesseract.go", "ocrMaxLeadingRatio", "policy"},
	{"internal/ocr/tesseract.go", "ocrTypeSizeRatio", "derived"},
	{"internal/ocr/tesseract.go", "ocrMaxPlateCoverage", "derived"},
	{"internal/ocr/tesseract.go", "ocrMinPlateLineFill", "derived"},
	{"internal/ocr/tesseract.go", "ocrMaxWordGapRatio", "derived"},
	{"internal/ocr/tesseract.go", "ocrBoundaryReach", "derived"},
	{"internal/ocr/tesseract.go", "clusterLinesRecording", "policy"},
	{"internal/ocr/script.go", "ocrScriptConfidenceFloor", "derived"},
	{"internal/ocr/screen.go", "ocrScreenSigmaDivisor", "derived"},
	{"internal/ocr/screen.go", "ocrScreenTile", "derived"},
	{"internal/ocr/screen.go", "ocrScreenMinPitch", "derived"},
	{"internal/ocr/screen.go", "ocrScreenMaxPitch", "derived"},
	{"internal/ocr/screen.go", "ocrScreenMaxTiles", "derived"},
	{"internal/ocr/screen.go", "ocrScreenMinEnergy", "derived"},
	{"internal/ocr/screen.go", "ocrScreenPeakFloor", "derived"},
	{"internal/ocr/screen.go", "ocrScreenTileFrac", "derived"},
	{"internal/ocr/screen.go", "ocrScreenTileCoverMax", "policy"},
	{"internal/ocr/screen.go", "ocrScreenMergeMaxOverlap", "policy"},
	{"internal/ocr/text.go", "isTranslatable", "policy"},
	{"internal/ocr/overlay.go", "ocrScript", "policy"},
	{"internal/ocr/overlay.go", "fontFitFactor", "policy"},
	{"internal/ocr/overlay.go", "ringMinSamples", "policy"},
	{"internal/ocr/overlay.go", "inkDeviationMin", "policy"},
	{"internal/ocr/overlay.go", "inkStripLines", "policy"},
	{"internal/ocr/overlay.go", "inkMinPerMille", "policy"},
	{"internal/ocr/overlay.go", "inkMinSamples", "policy"},
	{"internal/ocr/overlay.go", "plateMinContrast", "policy"},
	{"internal/ocr/overlay.go", "ringPadDivisor", "policy"},
	{"internal/ocr/overlay.go", "ringMinPad", "policy"},
	{"internal/ocr/overlay.go", "fallbackLumaSplit", "policy"},
	{"internal/ocr/overlay.go", "fallbackDarkInk", "policy"},
	{"internal/ocr/overlay.go", "fallbackLightInk", "policy"},
	{"internal/ocr/conceal.go", "modeBusyMax", "derived"},
	{"internal/ocr/conceal.go", "modeFlatSpread", "derived"},
	{"internal/ocr/conceal.go", "modeMaskPadDivisor", "policy"},
	{"extension/src/ocr-overlay.js", "OCR_PSM", "policy"},
	{"extension/src/ocr-overlay.js", "OCR_SPARSE_PSM", "policy"},
	{"extension/src/ocr-cluster.js", "OCR_RESCUE_LINE_CONF", "derived"},
	{"extension/src/ocr-cluster.js", "OCR_RESCUE_ANCHOR_CONF", "inherited"},
	{"extension/src/ocr-cluster.js", "OCR_RESCUE_ANCHOR_RUN", "inherited"},
	{"extension/src/ocr-cluster.js", "OCR_RESCUE_ANCHOR_VOTES", "derived"},
	{"extension/src/ocr-overlay.js", "OCR_UPSCALE_FACTOR", "policy"},
	{"extension/src/ocr-overlay.js", "OCR_ASSUMED_PAGE_INCHES", "policy"},
	{"extension/src/ocr-overlay.js", "OCR_UPSCALE_DPI_FLOOR", "policy"},
	{"extension/src/ocr-overlay.js", "OCR_MIN_DECLARED_DPI", "policy"},
	{"extension/src/ocr-cluster.js", "OCR_MIN_LINE_CONF", "policy"},
	{"extension/src/ocr-cluster.js", "OCR_CLUSTER_PITCH_FACTOR", "policy"},
	{"extension/src/ocr-cluster.js", "OCR_MAX_LEADING_RATIO", "policy"},
	{"extension/src/ocr-cluster.js", "OCR_TYPE_SIZE_RATIO", "derived"},
	{"extension/src/ocr-cluster.js", "OCR_MAX_PLATE_COVERAGE", "derived"},
	{"extension/src/ocr-cluster.js", "OCR_MIN_PLATE_LINE_FILL", "derived"},
	{"extension/src/ocr-cluster.js", "OCR_MAX_WORD_GAP_RATIO", "derived"},
	{"extension/src/ocr-cluster.js", "OCR_BOUNDARY_REACH", "derived"},
	{"extension/src/ocr-cluster.js", "clusterLines", "policy"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_SIGMA_DIVISOR", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_TILE", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_MIN_PITCH", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_MAX_PITCH", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_MAX_TILES", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_MIN_ENERGY", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_PEAK_FLOOR", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_TILE_FRAC", "derived"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_TILE_COVER_MAX", "policy"},
	{"extension/src/ocr-screen.js", "OCR_SCREEN_MERGE_MAX_OVERLAP", "policy"},
	{"extension/src/ocr-text.js", "isTranslatable", "policy"},
	{"extension/src/ocr-plates.js", "fitPlate", "policy"},
	{"extension/src/ocr-plates.js", "FONT_FIT", "policy"},
	{"extension/src/ocr-plates.js", "FONT_GROW_CAP", "policy"},
	{"extension/src/ocr-overlay.js", "RING_MIN_SAMPLES", "policy"},
	{"extension/src/ocr-overlay.js", "INK_DEVIATION_MIN", "policy"},
	{"extension/src/ocr-overlay.js", "INK_STRIP_LINES", "policy"},
	{"extension/src/ocr-overlay.js", "INK_MIN_PER_MILLE", "policy"},
	{"extension/src/ocr-overlay.js", "INK_MIN_SAMPLES", "policy"},
	{"extension/src/ocr-overlay.js", "PLATE_MIN_CONTRAST", "policy"},
	{"extension/src/ocr-overlay.js", "RING_PAD_DIVISOR", "policy"},
	{"extension/src/ocr-overlay.js", "RING_MIN_PAD", "policy"},
	{"extension/src/ocr-overlay.js", "FALLBACK_LUMA_SPLIT", "policy"},
	{"extension/src/ocr-overlay.js", "FALLBACK_DARK_INK", "policy"},
	{"extension/src/ocr-overlay.js", "FALLBACK_LIGHT_INK", "policy"},
	{"extension/src/ocr-conceal.js", "MODE_BUSY_MAX", "derived"},
	{"extension/src/ocr-conceal.js", "MODE_FLAT_SPREAD", "derived"},
	{"extension/src/ocr-conceal.js", "MODE_MASK_PAD_DIVISOR", "policy"},
	{"extension/src/page-agent.js", "MIN_PICTURE_PX", "policy"},
}

var ocrStatusMarker = regexp.MustCompile(`OCR-OVERLAY rule 13: (derived|inherited|policy) - \S`)

func TestOCRConstantsCarryTheirRule13Status(t *testing.T) {
	sources := map[string][]string{}
	for _, c := range ocrConstantStatus {
		lines, ok := sources[c.file]
		if !ok {
			lines = strings.Split(readRepoFile(t, strings.Split(c.file, "/")...), "\n")
			sources[c.file] = lines
		}
		decl := regexp.MustCompile(`^\s*(?:export\s+)?(?:const\s+|var\s+|func\s+|function\s+)?` + regexp.QuoteMeta(c.name) + `\b\s*[=(]`)
		at := -1
		for i, l := range lines {
			if decl.MatchString(l) {
				at = i
				break
			}
		}
		if at < 0 {
			t.Errorf("%s: no declaration of %s - the constant moved or was renamed; update this table and the catalog's section 5 table together", c.file, c.name)
			continue
		}
		// The comment block directly above the declaration, nearest line first.
		got := ""
		for i := at - 1; i >= 0; i-- {
			l := strings.TrimSpace(lines[i])
			if !strings.HasPrefix(l, "//") {
				break
			}
			if m := ocrStatusMarker.FindStringSubmatch(l); m != nil {
				got = m[1]
				break
			}
		}
		switch {
		case got == "":
			t.Errorf("%s: %s carries no OCR-OVERLAY rule 13 status marker directly above its declaration (OCR-PIPELINE amendment 1.4 C)", c.file, c.name)
		case got != c.status:
			t.Errorf("%s: %s is marked %q, the catalog's section 5 table says %q - amend the catalog first, then the marker", c.file, c.name, got, c.status)
		}
	}
}
