package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/limits"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// cmykFixtureW and cmykFixtureH size the two-band picture: the top half red, the bottom
// half blue, so an upside-down copy is told apart from an upright one by one pixel.
const (
	cmykFixtureW = 64
	cmykFixtureH = 96
)

// cmykRed and cmykBlue are the two bands in DeviceCMYK.
var (
	cmykRed  = [4]byte{0, 255, 255, 0}
	cmykBlue = [4]byte{255, 255, 0, 0}
)

// flate compresses b with zlib, the /FlateDecode stream format.
func flate(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// cmykBands returns the image samples, first row first, as a PDF stores them: the rows are
// written top to bottom, so the red band is the first half of the stream.
func cmykBands(indexed bool) []byte {
	var b []byte
	for y := 0; y < cmykFixtureH; y++ {
		for x := 0; x < cmykFixtureW; x++ {
			top := y < cmykFixtureH/2
			switch {
			case indexed && top:
				b = append(b, 0)
			case indexed:
				b = append(b, 1)
			case top:
				b = append(b, cmykRed[:]...)
			default:
				b = append(b, cmykBlue[:]...)
			}
		}
	}
	return b
}

// writeCMYKFixturePDF writes a one-page PDF painting one Flate-compressed CMYK raster, either
// DeviceCMYK or an Indexed palette over DeviceCMYK. The placement matrix has positive scales,
// so the page shows the raster as stored: red on top.
func writeCMYKFixturePDF(t *testing.T, path string, indexed bool) {
	t.Helper()
	data := flate(t, cmykBands(indexed))
	colorSpace := "/DeviceCMYK"
	if indexed {
		lookup := fmt.Sprintf("%X%X", cmykRed[:], cmykBlue[:])
		colorSpace = "[/Indexed /DeviceCMYK 1 <" + lookup + ">]"
	}
	content := "q 256 0 0 384 100 200 cm /Im0 Do Q\n"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>",
		fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace %s /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream",
			cmykFixtureW, cmykFixtureH, colorSpace, len(data), data),
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A Flate CMYK raster used to land on disk as an upside-down .tif, which Chrome does not show
// at all, with a CSS flip on the <img> to turn it back. It must now be a PNG the right way up,
// and the page must not flip it again.
func TestExtractImages_CMYKRasterIsUprightPNG(t *testing.T) {
	for _, indexed := range []bool{false, true} {
		name := "DeviceCMYK"
		if indexed {
			name = "IndexedCMYK"
		}
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			pdfPath := filepath.Join(tmp, name+".pdf")
			writeCMYKFixturePDF(t, pdfPath, indexed)
			out := filepath.Join(tmp, "out")

			got := extractImages(pdfPath, out)
			imgs := got.byPage[1]
			if len(imgs) != 1 {
				t.Fatalf("page 1 images = %v, want one", imgs)
			}
			rel := imgs[0]
			if !strings.EqualFold(filepath.Ext(rel), ".png") {
				t.Fatalf("image written as %q, want a .png Chrome can show", rel)
			}
			assertRedOverBlue(t, filepath.Join(out, filepath.FromSlash(rel)))
			if leftovers, _ := filepath.Glob(filepath.Join(out, "pdf_images", "*.tif")); len(leftovers) > 0 {
				t.Errorf("the source .tif was left beside the PNG: %v", leftovers)
			}

			page := buildPDFPageHTML(out, "Book", 1, 1, nil, imgs)
			if strings.Contains(page, "scaleY(-1)") || strings.Contains(page, "flip") {
				t.Errorf("page still flips the image in CSS:\n%s", page)
			}
		})
	}
}

// assertRedOverBlue checks that the image file at path is the fixture the right way up.
func assertRedOverBlue(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if b := img.Bounds(); b.Dx() != cmykFixtureW || b.Dy() != cmykFixtureH {
		t.Fatalf("size = %v, want %dx%d", b, cmykFixtureW, cmykFixtureH)
	}
	isRed := func(p image.Point) bool {
		r, g, b, _ := img.At(p.X, p.Y).RGBA()
		return r > 0xC000 && g < 0x4000 && b < 0x4000
	}
	isBlue := func(p image.Point) bool {
		r, g, b, _ := img.At(p.X, p.Y).RGBA()
		return r < 0x4000 && g < 0x4000 && b > 0xC000
	}
	top, bottom := image.Pt(cmykFixtureW/2, 2), image.Pt(cmykFixtureW/2, cmykFixtureH-3)
	if !isRed(top) || !isBlue(bottom) {
		t.Errorf("image is not upright: top %v, bottom %v, want red over blue", img.At(top.X, top.Y), img.At(bottom.X, bottom.Y))
	}
}

// The transcode needs the whole raster, and its size comes from the PDF. A frame over the pixel
// budget is refused from the header, before any allocation, and the original bytes are kept so
// the caller can still write the .tif.
func TestTIFFAsPNGRefusesOverBudget(t *testing.T) {
	hdr := grayTIFFHeader(40000, 40000)
	img, err := tiffAsPNG(model.Image{FileType: "tif", Reader: bytes.NewReader(hdr)})
	if !errors.Is(err, limits.ErrTooLarge) {
		t.Fatalf("tiffAsPNG = %v, want a pixel-budget refusal", err)
	}
	if img.FileType != "tif" {
		t.Errorf("FileType = %q after a refusal, want tif", img.FileType)
	}
	if kept, _ := io.ReadAll(img.Reader); !bytes.Equal(kept, hdr) {
		t.Error("the original TIFF bytes were not put back")
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
