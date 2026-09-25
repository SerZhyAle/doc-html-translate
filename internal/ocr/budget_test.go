package ocr

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/internal/limits"
)

func TestMemoryWorkers(t *testing.T) {
	const gib = int64(1) << 30
	cases := []struct {
		name          string
		pixels, bytes int64
		want          int
	}{
		// 16 bytes a pixel per worker: a 100-megapixel scan needs 1.6 GB, so one worker on 386.
		{"budget-size scan, 386", limits.MaxImagePixels, gib, 1},
		{"budget-size scan, 64-bit", limits.MaxImagePixels, 4 * gib, 2},
		{"10-megapixel page, 386", 10_000_000, gib, 6},
		{"comic page, 64-bit", 4_000_000, 4 * gib, 67},
		// Above the budget the image is never decoded, so it costs no more than the budget.
		{"over-budget image", 10 * limits.MaxImagePixels, 4 * gib, 2},
		{"tiny image, tiny budget", 100, 1, 1},
	}
	for _, c := range cases {
		if got := memoryWorkers(c.pixels, c.bytes); got != c.want {
			t.Errorf("%s: memoryWorkers(%d, %d) = %d, want %d", c.name, c.pixels, c.bytes, got, c.want)
		}
	}
	if got := memoryWorkers(0, gib); got < ocrWorkers() {
		t.Errorf("unknown sizes must leave the CPU count in charge, got %d", got)
	}
}

// pngHeader is a PNG signature and IHDR declaring w x h, with nothing behind it: enough for
// DecodeConfig, and a bomb for Decode.
func pngHeader(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	_ = binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

// An image declaring more than the budget is never decoded in process: every pass that would
// decode it gets nil and keeps its default, and the pool is sized as if it were budget-sized.
func TestOverBudgetImageIsNotDecoded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.png")
	// 120 megapixels: over the budget, yet inside what image/png's header check accepts on the
	// 386 build (past about 268 megapixels it refuses the header there, and the image stays
	// undecoded all the same).
	if err := os.WriteFile(path, pngHeader(12000, 10000), 0o644); err != nil {
		t.Fatal(err)
	}
	if decodeImage(path) != nil {
		t.Fatal("decodeImage decoded an image over the pixel budget")
	}
	if greyRendition(path) != nil {
		t.Error("greyRendition built a rendition of an image over the pixel budget")
	}
	if got := largestPixels([]string{path}); got != 12000*10000 {
		t.Errorf("largestPixels = %d, want the declared size", got)
	}
}

// The grey rendition is derived from the frame already in memory, and the colour picture is
// released once it exists.
func TestFrameGreyReleasesColour(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	f := &ocrFrame{path: "not-read", img: src, tried: true}
	g := f.grey()
	if g == nil || g.Bounds().Dx() != 3 {
		t.Fatalf("grey = %v", g)
	}
	if f.img != nil {
		t.Error("the colour frame is still held after the grey rendition was built")
	}
	if f.grey() != g {
		t.Error("the grey rendition was built twice")
	}
}
