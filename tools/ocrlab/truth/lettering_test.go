package truth

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func writeMask(t *testing.T, dir, id string, w, h int, lit image.Rectangle) {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := lit.Min.Y; y < lit.Max.Y; y++ {
		for x := lit.Min.X; x < lit.Max.X; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	f, err := os.Create(LetteringPath(dir, id))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLettering(t *testing.T) {
	dir := t.TempDir()
	if m, err := LoadLettering(dir, "none", 10, 10); m != nil || err != nil {
		t.Fatalf("a scene without a sidecar has unknown lettering, not empty: %v %v", m, err)
	}
	writeMask(t, dir, "s", 20, 10, image.Rect(2, 3, 6, 5))
	m, err := LoadLettering(dir, "s", 20, 10)
	if err != nil {
		t.Fatal(err)
	}
	if m.Area() != 8 || !m.At(2, 3) || !m.At(5, 4) || m.At(6, 4) {
		t.Errorf("mask area %d", m.Area())
	}
	if _, err := LoadLettering(dir, "s", 21, 10); err == nil {
		t.Error("a mask of another size must be refused, not stretched")
	}
}
