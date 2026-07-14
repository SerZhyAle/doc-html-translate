package img

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsImage(t *testing.T) {
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tif", ".tiff"} {
		if !IsImage(ext) {
			t.Errorf("IsImage(%q) = false, want true", ext)
		}
	}
	for _, ext := range []string{".pdf", ".epub", ".txt", "", ".PNG"} {
		if IsImage(ext) {
			t.Errorf("IsImage(%q) = true, want false", ext)
		}
	}
}

func TestExtract(t *testing.T) {
	dir := t.TempDir()

	// A tiny PNG to stand in for the input image.
	imgPath := filepath.Join(dir, "My Photo.png")
	writePNG(t, imgPath, 4, 3)

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := Extract(imgPath, outDir)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// One-spine book so the pipeline runs its single-page + OCR-overlay flow.
	if got := len(book.Spine); got != 1 {
		t.Fatalf("Spine length = %d, want 1", got)
	}
	if book.Title != "My Photo" {
		t.Errorf("Title = %q, want %q", book.Title, "My Photo")
	}
	if cf := book.ContentFiles(); len(cf) != 1 || cf[0].Href != "page_001.html" {
		t.Errorf("ContentFiles = %+v, want single page_001.html", cf)
	}

	// The image was copied next to the page under its original name.
	if _, err := os.Stat(filepath.Join(outDir, "My Photo.png")); err != nil {
		t.Errorf("image not copied into output dir: %v", err)
	}

	// The generated page references the copied image so the OCR overlay can find it.
	pageHTML, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if !strings.Contains(string(pageHTML), `src="My Photo.png"`) {
		t.Errorf("page does not reference the image:\n%s", pageHTML)
	}
}

// writePNG writes a solid-colour PNG of the given size.
func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, im); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
