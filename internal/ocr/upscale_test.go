package ocr

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func tempImageFile(t *testing.T, w, h int) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "img-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestUpscaleForOCR(t *testing.T) {
	p, cleanup, ok := stageForOCR(tempImageFile(t, 40, 30), orientNormal, true)
	if !ok {
		t.Fatal("image should upscale")
	}
	defer cleanup()
	im := decodeImage(p)
	if im == nil {
		t.Fatal("upscaled temp not decodable")
	}
	if w := im.Bounds().Dx(); w != 40*ocrUpscaleFactor {
		t.Errorf("width = %d, want %d", w, 40*ocrUpscaleFactor)
	}
	if h := im.Bounds().Dy(); h != 30*ocrUpscaleFactor {
		t.Errorf("height = %d, want %d", h, 30*ocrUpscaleFactor)
	}
}

func TestEstimateDPI(t *testing.T) {
	cases := []struct{ long, want int }{
		{0, 0},
		{1001, 91},  // a tiny newsprint page scan: below the floor -> upscaled
		{1664, 151}, // a mid-res comic scan: above the floor -> DPI declared only
	}
	for _, c := range cases {
		if got := estimateDPI(c.long); got != c.want {
			t.Errorf("estimateDPI(%d) = %d, want %d", c.long, got, c.want)
		}
	}
}

// TestPrepareForOCRGate: a low-DPI-estimate image is upscaled (scale 2, DPI declared at the
// post-upscale value); a mid-DPI-estimate image is recognized as-is (scale 1) with only its DPI
// declared. The gate keys on estimated DPI, not raw pixel count.
func TestPrepareForOCRGate(t *testing.T) {
	// long side 660 -> estDPI 60 (< floor 120) -> upscale
	path, scale, dpi, cleanup := prepareForOCR(tempImageFile(t, 660, 400))
	defer cleanup()
	if scale != ocrUpscaleFactor {
		t.Errorf("low-DPI image: scale = %d, want %d", scale, ocrUpscaleFactor)
	}
	if want := clampDeclaredDPI(estimateDPI(660) * ocrUpscaleFactor); dpi != want {
		t.Errorf("low-DPI image: declared dpi = %d, want %d", dpi, want)
	}
	if im := decodeImage(path); im == nil || im.Bounds().Dx() != 660*ocrUpscaleFactor {
		t.Errorf("low-DPI image was not upscaled")
	}

	// long side 2200 -> estDPI 200 (>= floor 120) -> no upscale, DPI declared as-is
	src := tempImageFile(t, 2200, 1600)
	path2, scale2, dpi2, cleanup2 := prepareForOCR(src)
	defer cleanup2()
	if scale2 != 1 {
		t.Errorf("mid-DPI image: scale = %d, want 1", scale2)
	}
	if dpi2 != estimateDPI(2200) {
		t.Errorf("mid-DPI image: declared dpi = %d, want %d", dpi2, estimateDPI(2200))
	}
	if path2 != src {
		t.Errorf("ASCII path should be handed to tesseract unchanged, got %q", path2)
	}
}

func TestClampDeclaredDPI(t *testing.T) {
	if got := clampDeclaredDPI(0); got != 0 {
		t.Errorf("clampDeclaredDPI(0) = %d, want 0 (no declaration)", got)
	}
	if got := clampDeclaredDPI(40); got != ocrMinDeclaredDPI {
		t.Errorf("clampDeclaredDPI(40) = %d, want %d", got, ocrMinDeclaredDPI)
	}
	if got := clampDeclaredDPI(150); got != 150 {
		t.Errorf("clampDeclaredDPI(150) = %d, want 150", got)
	}
}

// TestStageASCIIPath: an all-ASCII path is handed to tesseract unchanged; a path with non-ASCII
// bytes (a book under a Cyrillic name) is copied to an ASCII temp file, because tesseract/leptonica
// mangle non-ANSI-codepage bytes and would fail recognition silently.
func TestStageASCIIPath(t *testing.T) {
	ascii := tempImageFile(t, 10, 10)
	if got, cleanup := stageASCIIPath(ascii); got != ascii {
		cleanup()
		t.Errorf("ASCII path changed: got %q, want %q", got, ascii)
	}

	dir := t.TempDir()
	cyr := filepath.Join(dir, "комикс.png")
	if err := os.WriteFile(cyr, mustPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	got, cleanup := stageASCIIPath(cyr)
	defer cleanup()
	if got == cyr {
		t.Fatal("non-ASCII path should be staged to an ASCII copy")
	}
	if !isASCIIPath(got) {
		t.Errorf("staged path is not ASCII: %q", got)
	}
	if decodeImage(got) == nil {
		t.Errorf("staged copy is not a readable image")
	}
}

func mustPNG(t *testing.T) []byte {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "src-*.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestScaleDown(t *testing.T) {
	res := Result{Width: 200, Height: 100, Blocks: []Block{{Text: "x", X0: 10, Y0: 20, X1: 110, Y1: 60, LineH: 18}}}
	scaleDown(&res, 2)
	if res.Width != 100 || res.Height != 50 {
		t.Errorf("dims = %dx%d, want 100x50", res.Width, res.Height)
	}
	b := res.Blocks[0]
	if b.X0 != 5 || b.Y0 != 10 || b.X1 != 55 || b.Y1 != 30 || b.LineH != 9 {
		t.Errorf("block = (%d,%d,%d,%d) lineH=%d, want (5,10,55,30) lineH=9", b.X0, b.Y0, b.X1, b.Y1, b.LineH)
	}
}
