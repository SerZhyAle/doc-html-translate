package pipeline

import (
	"bytes"
	"strings"
	"testing"

	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/ocr"
)

// capture collects everything the run log receives while fn runs. The report lines go through
// logging.Printf, which mirrors to the run log, so this reads exactly what the user is told.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	logging.StartRunLog(&buf)
	defer logging.StopRunLog()
	fn()
	return buf.String()
}

// "no text found" is a statement about the language data that was loaded, and it reads as a
// statement about the picture. When nothing at all was recognized, the language has to be named
// along with the way to change it - otherwise a Russian comic looks like a broken feature.
func TestReportOverlayNamesTheLanguageWhenNothingWasRecognized(t *testing.T) {
	out := capture(t, func() {
		reportOverlay(ocr.OverlayResult{NoText: 3}, "eng")
	})
	if !strings.Contains(out, "eng (English)") {
		t.Errorf("the hint must name the data that was used, got:\n%s", out)
	}
	if !strings.Contains(out, "-ocr-lang") {
		t.Errorf("the hint must say how to change the language, got:\n%s", out)
	}
}

// A comic whose art panels hold no dialogue is the ordinary case, not a language problem. The
// hint must stay quiet whenever anything at all was recognized.
func TestReportOverlayStaysQuietWhenSomethingWasRecognized(t *testing.T) {
	out := capture(t, func() {
		reportOverlay(ocr.OverlayResult{Overlaid: 7, NoText: 593}, "eng")
	})
	if strings.Contains(out, "-ocr-lang") {
		t.Errorf("a partially recognized book must not be told to change its language, got:\n%s", out)
	}
}

// No images and no text is a document without pictures, which the language cannot explain.
func TestReportOverlayStaysQuietWithoutImages(t *testing.T) {
	out := capture(t, func() {
		reportOverlay(ocr.OverlayResult{}, "eng")
	})
	if strings.Contains(out, "-ocr-lang") {
		t.Errorf("a document with no images must not get a language hint, got:\n%s", out)
	}
}
