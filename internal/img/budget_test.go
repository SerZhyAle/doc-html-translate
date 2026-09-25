package img

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"doc-html-translate/internal/limits"
)

// tiffBomb returns a tiny little-endian TIFF whose one IFD declares a w x h 8-bit grey image.
// The file stays a few hundred bytes whatever the declared size, which is the whole attack.
func tiffBomb(w, h uint32) []byte {
	type entry struct {
		tag, typ uint16
		count    uint32
		value    uint32
	}
	const short, long = 3, 4
	entries := []entry{
		{256, long, 1, w},   // ImageWidth
		{257, long, 1, h},   // ImageLength
		{258, short, 1, 8},  // BitsPerSample
		{259, short, 1, 1},  // Compression: none
		{262, short, 1, 1},  // Photometric: BlackIsZero
		{273, long, 1, 8},   // StripOffsets
		{277, short, 1, 1},  // SamplesPerPixel
		{278, long, 1, h},   // RowsPerStrip
		{279, long, 1, 256}, // StripByteCounts
	}
	var b bytes.Buffer
	le := binary.LittleEndian
	b.Write([]byte{'I', 'I', 42, 0})
	_ = binary.Write(&b, le, uint32(8+256))
	b.Write(make([]byte, 256))
	_ = binary.Write(&b, le, uint16(len(entries)))
	for _, e := range entries {
		_ = binary.Write(&b, le, e)
	}
	_ = binary.Write(&b, le, uint32(0))
	return b.Bytes()
}

// Done criterion 1: a 1 KB TIFF declaring 60000 x 60000 is refused with a message naming the
// limit, before the raster is allocated. Run under GOARCH=386 as well: there the allocation
// alone would take the process down.
func TestExtractTIFFPixelBombRefused(t *testing.T) {
	data := tiffBomb(60000, 60000)
	if len(data) > 1024 {
		t.Fatalf("bomb fixture is %d bytes, want <= 1 KB", len(data))
	}
	tifPath := filepath.Join(t.TempDir(), "bomb.tif")
	if err := os.WriteFile(tifPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := Extract(tifPath, t.TempDir())
	runtime.ReadMemStats(&after)

	if !errors.Is(err, limits.ErrTooLarge) {
		t.Fatalf("Extract = %v, want a pixel-budget refusal", err)
	}
	if !strings.Contains(err.Error(), "60000 x 60000") || !strings.Contains(err.Error(), "megapixels") {
		t.Errorf("refusal does not name the size and the limit: %v", err)
	}
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 16<<20 {
		t.Errorf("refusal allocated %d bytes; the raster must never be allocated", grew)
	}
}

// A small frame from the same hand-built layout still decodes: the probe must not refuse
// legitimate input, and the view-based decode must read the strip through the real file.
func TestExtractTIFFUnderBudgetDecodes(t *testing.T) {
	tifPath := filepath.Join(t.TempDir(), "ok.tif")
	if err := os.WriteFile(tifPath, tiffBomb(16, 16), 0o644); err != nil {
		t.Fatal(err)
	}
	book, err := Extract(tifPath, t.TempDir())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != 1 {
		t.Fatalf("Spine length = %d, want 1", len(book.Spine))
	}
}

// An IFD offset near 4 GB used to be cast to int, which is negative on the 386 build, pass the
// bounds check and panic on the slice. It must end the walk instead.
func TestTIFFFrameOffsetsHugeIFDOffset(t *testing.T) {
	for _, off := range []uint32{0xFFFFFFF0, 0x80000000, 0x7FFFFFFF} {
		data := tiffBomb(8, 8)
		binary.LittleEndian.PutUint32(data[len(data)-4:], off) // the chain's "next IFD"
		offsets, _, err := tiffFrameOffsets(bytes.NewReader(data), int64(len(data)))
		if err != nil || len(offsets) != 1 {
			t.Errorf("next=%#x: offsets=%v err=%v, want the one real frame", off, offsets, err)
		}
		binary.LittleEndian.PutUint32(data[4:8], off) // and as the first IFD
		if _, _, err := tiffFrameOffsets(bytes.NewReader(data), int64(len(data))); err == nil {
			t.Errorf("first=%#x: want an error for a header pointing past the file", off)
		}
	}
}

// The per-frame view substitutes only the header's IFD pointer and never copies the file.
func TestFrameViewPatchesOnlyTheIFDPointer(t *testing.T) {
	src := []byte{'I', 'I', 42, 0, 1, 2, 3, 4, 9, 9}
	v := &frameView{r: bytes.NewReader(src), ifd: [4]byte{0xA, 0xB, 0xC, 0xD}}
	got := make([]byte, 6)
	if _, err := v.ReadAt(got, 3); err != nil {
		t.Fatal(err)
	}
	if want := []byte{0, 0xA, 0xB, 0xC, 0xD, 9}; !bytes.Equal(got, want) {
		t.Errorf("ReadAt(3) = %v, want %v", got, want)
	}
	if src[4] != 1 {
		t.Error("the underlying file bytes were modified")
	}
}
