package ocr

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

// diagFixture writes a small PNG and an HTML page referencing it, and returns the parsed page,
// its directory and the recognition results keyed by the image's absolute path - everything
// applyOverlays needs, with no tesseract involved.
func diagFixture(t *testing.T) (dir string, imgPath string, results map[string]recognition) {
	t.Helper()
	dir = t.TempDir()
	imgPath = filepath.Join(dir, "page.png")

	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{240, 240, 235, 255})
		}
	}
	for y := 20; y < 34; y++ { // a dark bar standing in for a line of text
		for x := 20; x < 160; x++ {
			img.SetRGBA(x, y, color.RGBA{20, 20, 24, 255})
		}
	}
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	results = map[string]recognition{
		imgPath: {
			ok: true,
			res: Result{Width: 200, Height: 100, Blocks: []Block{
				{Text: "Hello there reader", X0: 20, Y0: 20, X1: 160, Y1: 34, LineH: 14},
			}},
		},
	}
	return dir, imgPath, results
}

func overlayOnce(t *testing.T, dir string, results map[string]recognition) string {
	t.Helper()
	doc, err := gohtml.Parse(strings.NewReader(`<html><body><p><img src="page.png"></p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, changed := applyOverlays(doc, dir, results); !changed {
		t.Fatal("fixture produced no overlay")
	}
	var buf bytes.Buffer
	if err := gohtml.Render(&buf, doc); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// The promise the strategic spec's edition table makes: the runner gets diagnostics and the
// reader gets exactly the page they got before. If this test ever fails, the sidecar has become
// a feature rather than an observation, and the lab is measuring a different program than the
// one that ships.
func TestDiagnosticsDoNotChangeOutput(t *testing.T) {
	dir, _, results := diagFixture(t)

	t.Setenv(diagEnvVar, "")
	off := overlayOnce(t, dir, results)

	t.Setenv(diagEnvVar, filepath.Join(t.TempDir(), "diag.jsonl"))
	on := overlayOnce(t, dir, results)

	if off != on {
		t.Errorf("the rewritten page differs when diagnostics are on.\n--- off ---\n%s\n--- on ---\n%s", off, on)
	}
}

func TestDiagnosticsAreOffByDefault(t *testing.T) {
	t.Setenv(diagEnvVar, "")
	if got := DiagnosticsPath(); got != "" {
		t.Errorf("DiagnosticsPath() = %q with the variable unset, want empty", got)
	}

	dir, _, results := diagFixture(t)
	overlayOnce(t, dir, results)
	// Nothing to assert about a file that must not exist beyond the fact that no path was ever
	// opened; the real guarantee is the identity test above plus this empty path.
}

// What the runner actually reads: one JSON line per overlaid image, carrying the geometry the
// page was rendered with rather than a second implementation of it.
func TestDiagnosticsRecordGeometryAndColours(t *testing.T) {
	dir, imgPath, results := diagFixture(t)
	out := filepath.Join(t.TempDir(), "diag.jsonl")
	t.Setenv(diagEnvVar, out)

	page := overlayOnce(t, dir, results)

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("diagnostics file was not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("want one line per overlaid image, got %d", len(lines))
	}
	var rec diagImage
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatal(err)
	}

	if rec.File != imgPath {
		t.Errorf("File = %q, want %q", rec.File, imgPath)
	}
	if rec.Width != 200 || rec.Height != 100 {
		t.Errorf("geometry = %dx%d, want 200x100", rec.Width, rec.Height)
	}
	if len(rec.Blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(rec.Blocks))
	}
	b := rec.Blocks[0]
	if b.Text != "Hello there reader" {
		t.Errorf("Text = %q", b.Text)
	}
	if b.X0 != 20 || b.Y0 != 20 || b.X1 != 160 || b.Y1 != 34 || b.LineH != 14 {
		t.Errorf("box = (%d,%d)-(%d,%d) lineH %d, want the recognized values", b.X0, b.Y0, b.X1, b.Y1, b.LineH)
	}

	// The recorded style must be the style the page carries - that identity is the reason the
	// lab can trust the sidecar instead of recomputing plate geometry itself.
	if b.Style == "" {
		t.Fatal("no style recorded")
	}
	if !strings.Contains(page, b.Style) {
		t.Errorf("recorded style %q does not appear in the rendered page", b.Style)
	}
	if b.Background == "" || b.Ink == "" {
		t.Errorf("sampled colours missing: bg=%q ink=%q", b.Background, b.Ink)
	}
	if !strings.Contains(page, "background:"+b.Background) {
		t.Errorf("recorded background %q does not appear in the rendered page", b.Background)
	}
}

// readDiagLines applies the overlay to a one-image page and returns the diagnostics records it
// wrote, whatever applyOverlays decided about the DOM.
func readDiagLines(t *testing.T, dir string, results map[string]recognition) []diagImage {
	t.Helper()
	out := filepath.Join(t.TempDir(), "diag.jsonl")
	t.Setenv(diagEnvVar, out)
	doc, err := gohtml.Parse(strings.NewReader(`<html><body><p><img src="page.png"></p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	stats, changed := applyOverlays(doc, dir, results)
	if changed || stats.NoText != 1 || stats.Overlaid != 0 {
		t.Fatalf("fixture is not a no-plate image: changed=%v NoText=%d Overlaid=%d", changed, stats.NoText, stats.Overlaid)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("no diagnostics file for a no-plate image (OCR-OVERLAY rule 12): %v", err)
	}
	var recs []diagImage
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var rec diagImage
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatal(err)
		}
		recs = append(recs, rec)
		// Absent and empty must not be conflated: the raw line has to carry both arrays.
		if !strings.Contains(line, `"blocks":[`) || !strings.Contains(line, `"dropped":[`) {
			t.Errorf("record omits blocks or dropped: %s", line)
		}
	}
	return recs
}

// OCR-OVERLAY rule 12: the discard record is written also for an image that produced no plates,
// which is the case it exists for. This is the 2026-09-22 probe that found it missing, kept.
func TestDiagnosticsRecordDiscardsForNoPlateImage(t *testing.T) {
	dir, imgPath, _ := diagFixture(t)
	results := map[string]recognition{
		imgPath: {res: Result{Width: 200, Height: 100, Dropped: []DroppedLine{
			{Text: "Hello there reader", Conf: ocrMinLineConf - 5, Floor: ocrMinLineConf, X0: 20, Y0: 20, X1: 160, Y1: 34},
		}}},
	}
	recs := readDiagLines(t, dir, results)
	if len(recs) != 1 {
		t.Fatalf("want one line for the no-plate image, got %d", len(recs))
	}
	rec := recs[0]
	if rec.File != imgPath || rec.Width != 200 || rec.Height != 100 {
		t.Errorf("record = %q %dx%d, want %q 200x100", rec.File, rec.Width, rec.Height, imgPath)
	}
	if len(rec.Blocks) != 0 {
		t.Errorf("no-plate image recorded %d blocks", len(rec.Blocks))
	}
	if len(rec.Dropped) != 1 {
		t.Fatalf("want the one discarded line, got %d", len(rec.Dropped))
	}
	d := rec.Dropped[0]
	if d.Text != "Hello there reader" || d.Conf != ocrMinLineConf-5 || d.Floor != ocrMinLineConf || d.X1 != 160 {
		t.Errorf("discard record = %+v, want the dropped line as recognized", d)
	}

	// The extension's lab harness writes the same line; both pin it to one literal.
	rec.File = "page.png"
	line, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != goDiagLine {
		t.Errorf("no-plate line drifted from the shape both editions write:\n got %s\nwant %s", line, goDiagLine)
	}
}

// goDiagLine is the line written for TestDiagnosticsRecordDiscardsForNoPlateImage's image, with the
// path shortened. extension/test/ocrlab-evidence.test.mjs holds GO_DIAG_LINE to the same bytes and
// TestParityOCRDiscardRecord holds the two literals equal.
const goDiagLine = `{"file":"page.png","width":200,"height":100,"blocks":[],"dropped":[{"text":"Hello there reader","conf":45,"floor":50,"x0":20,"y0":20,"x1":160,"y1":34}]}`

// "Read fine, found no text" still writes its line, with an empty dropped array, so it stays
// distinguishable from "everything was thrown away".
func TestDiagnosticsRecordEmptyDiscardsForBlankImage(t *testing.T) {
	dir, imgPath, _ := diagFixture(t)
	results := map[string]recognition{imgPath: {res: Result{Width: 200, Height: 100}}}
	recs := readDiagLines(t, dir, results)
	if len(recs) != 1 {
		t.Fatalf("want one line for the blank image, got %d", len(recs))
	}
	if len(recs[0].Blocks) != 0 || len(recs[0].Dropped) != 0 {
		t.Errorf("blank image recorded blocks=%d dropped=%d, want 0 and 0", len(recs[0].Blocks), len(recs[0].Dropped))
	}
}
