package ocr

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func writeTempPNG(t *testing.T, w, h int) string {
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

func TestUpscaleForOCRSmall(t *testing.T) {
	p, cleanup, ok := upscaleForOCR(writeTempPNG(t, 40, 30))
	if !ok {
		t.Fatal("small image should upscale")
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

func TestUpscaleForOCRLargeSkipped(t *testing.T) {
	// long side == threshold must NOT upscale (only strictly-below is enlarged).
	if _, _, ok := upscaleForOCR(writeTempPNG(t, ocrUpscaleBelow, 10)); ok {
		t.Error("image at/above threshold must not upscale")
	}
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
