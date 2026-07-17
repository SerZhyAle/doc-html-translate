package img

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/tiff"
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

// Chrome cannot decode TIFF, so a .tif input is transcoded to PNG. The output must be a PNG
// the browser can show, referenced by the page - never the copied .tif.
func TestExtractTIFFTranscodesToPNG(t *testing.T) {
	dir := t.TempDir()
	tifPath := filepath.Join(dir, "scan.tif")
	writeTIFF(t, tifPath, 8, 8)

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	book, err := Extract(tifPath, outDir)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := len(book.Spine); got != 1 {
		t.Fatalf("Spine length = %d, want 1", got)
	}
	// A PNG was written and the page points at it; the .tif was NOT copied through.
	if _, err := os.Stat(filepath.Join(outDir, "page_001.png")); err != nil {
		t.Errorf("transcoded PNG not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "scan.tif")); err == nil {
		t.Errorf("the unshowable .tif was copied into the output")
	}
	pageHTML, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if !strings.Contains(string(pageHTML), `src="page_001.png"`) {
		t.Errorf("page does not reference the PNG:\n%s", pageHTML)
	}
}

// A multi-page TIFF becomes one PNG page per frame, instead of stacking every frame's OCR
// plates onto a single image. Verified against the real 3-frame corpus fixture (a hand-built
// multi-frame TIFF is fragile - x/image/tiff's decoder is strict about required tags - so this
// uses the genuine article, and skips if the generated fixtures are not present).
func TestExtractMultiFrameTIFFOnePagePerFrame(t *testing.T) {
	tifPath := `P:\WINDOWS\EPUB_2_HTML\test_doc\_generated\img-tiff-3page_Nyoka.tiff`
	if _, err := os.Stat(tifPath); err != nil {
		t.Skipf("multi-frame fixture not present: %v", err)
	}

	// Confirm the fixture really is multi-frame, so a pass means what we think it does.
	data, err := os.ReadFile(tifPath)
	if err != nil {
		t.Fatal(err)
	}
	offsets, _, err := tiffFrameOffsets(data)
	if err != nil {
		t.Fatalf("tiffFrameOffsets: %v", err)
	}
	if len(offsets) < 2 {
		t.Fatalf("fixture has %d frame(s), expected a multi-frame TIFF", len(offsets))
	}

	outDir := t.TempDir()
	book, err := Extract(tifPath, outDir)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := len(book.Spine); got != len(offsets) {
		t.Fatalf("Spine length = %d, want %d (one page per frame)", got, len(offsets))
	}
	for i := 1; i <= len(offsets); i++ {
		name := fmt.Sprintf("page_%03d", i)
		if _, err := os.Stat(filepath.Join(outDir, name+".png")); err != nil {
			t.Errorf("frame %d PNG missing: %v", i, err)
		}
		if _, err := os.Stat(filepath.Join(outDir, name+".html")); err != nil {
			t.Errorf("frame %d page missing: %v", i, err)
		}
	}
}

// solidImage returns a solid-colour w x h RGBA image.
func solidImage(w, h int, c color.Color) image.Image {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, c)
		}
	}
	return im
}

// writePNG writes a solid-colour PNG of the given size.
func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, solidImage(w, h, color.White)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeTIFF writes a single-frame solid-colour TIFF of the given size.
func writeTIFF(t *testing.T, path string, w, h int) {
	t.Helper()
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, solidImage(w, h, color.RGBA{200, 30, 30, 255}), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
