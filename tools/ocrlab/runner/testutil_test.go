package runner

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makeGreyPNG draws a flat image with a dark bar, so a rendered plate has something to sit over
// and the fixture needs nothing from the corpus.
func makeGreyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{235, 235, 230, 255})
		}
	}
	for y := h / 4; y < h/4+10; y++ {
		for x := w / 8; x < w/2; x++ {
			img.SetRGBA(x, y, color.RGBA{30, 30, 34, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
