package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/internal/limits"
)

// Every concrete type the TIFF decoder returns is flipped by a row swap on its own buffer, and
// a type without one takes the generic path; all must end up mirrored.
func TestFlipRowsInPlace(t *testing.T) {
	gray := image.NewGray(image.Rect(0, 0, 2, 3))
	gray.Pix = []byte{1, 1, 2, 2, 3, 3}
	pal := image.NewPaletted(image.Rect(0, 0, 1, 3), color.Palette{color.Black, color.White, color.Gray{Y: 128}})
	pal.Pix = []byte{0, 1, 2}
	ycc := image.NewYCbCr(image.Rect(0, 0, 1, 2), image.YCbCrSubsampleRatio444) // no single Pix: the generic path
	ycc.Y[0], ycc.Y[1] = 0, 255
	for i := range ycc.Cb {
		ycc.Cb[i], ycc.Cr[i] = 128, 128 // neutral chroma, so Y alone sets the grey
	}
	sub := image.NewGray(image.Rect(0, 0, 4, 4)).SubImage(image.Rect(1, 1, 3, 3)).(*image.Gray)
	sub.Pix[0], sub.Pix[sub.Stride] = 7, 9

	if got := flipRowsInPlace(gray).(*image.Gray).Pix; !bytes.Equal(got, []byte{3, 3, 2, 2, 1, 1}) {
		t.Errorf("gray = %v", got)
	}
	if got := flipRowsInPlace(pal).(*image.Paletted).Pix; !bytes.Equal(got, []byte{2, 1, 0}) {
		t.Errorf("paletted = %v", got)
	}
	if _, ok := flipRowsInPlace(image.NewRGBA(image.Rect(0, 0, 1, 1))).(*image.RGBA); !ok {
		t.Error("a single-row image must come back unchanged")
	}
	generic := flipRowsInPlace(ycc)
	if top, _, _, _ := generic.At(0, 0).RGBA(); top < 0xF000 {
		t.Errorf("generic path: top row is %#x, want the former bottom (white)", top)
	}
	flipped := flipRowsInPlace(sub).(*image.Gray)
	if flipped.Pix[0] != 9 || flipped.Pix[flipped.Stride] != 7 {
		t.Errorf("sub-image rows not swapped: %v", flipped.Pix)
	}
}

// The flip needs the whole raster, and its size comes from the PDF. A frame over the pixel
// budget is refused from the header, before any allocation.
func TestFlipImageFileRefusesOverBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bomb.tif")
	if err := os.WriteFile(path, grayTIFFHeader(40000, 40000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := flipImageFileVertically(path); !errors.Is(err, limits.ErrTooLarge) {
		t.Fatalf("flipImageFileVertically = %v, want a pixel-budget refusal", err)
	}
}

// grayTIFFHeader is a minimal uncompressed 8-bit grey TIFF header declaring w x h, with no
// pixel data behind it.
func grayTIFFHeader(w, h uint32) []byte {
	type entry struct {
		Tag, Typ uint16
		Count    uint32
		Value    uint32
	}
	entries := []entry{
		{256, 4, 1, w}, {257, 4, 1, h}, {258, 3, 1, 8}, {259, 3, 1, 1}, {262, 3, 1, 1},
		{273, 4, 1, 8}, {277, 3, 1, 1}, {278, 4, 1, h}, {279, 4, 1, 16},
	}
	var b bytes.Buffer
	le := binary.LittleEndian
	b.Write([]byte{'I', 'I', 42, 0})
	_ = binary.Write(&b, le, uint32(8+16))
	b.Write(make([]byte, 16))
	_ = binary.Write(&b, le, uint16(len(entries)))
	for _, e := range entries {
		_ = binary.Write(&b, le, e)
	}
	_ = binary.Write(&b, le, uint32(0))
	return b.Bytes()
}
