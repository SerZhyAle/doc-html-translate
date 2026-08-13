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
