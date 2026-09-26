package comic

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"doc-html-translate/internal/limits"
	"doc-html-translate/internal/logging"
)

// zeroDeflate returns the raw DEFLATE stream and CRC of n zero bytes: a real bomb page
// that costs the test a fraction of a megabyte.
func zeroDeflate(t *testing.T, n int64) ([]byte, uint32) {
	t.Helper()
	var buf bytes.Buffer
	fw, err := flate.NewWriter(&buf, flate.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	crc := crc32.NewIEEE()
	chunk := make([]byte, 1<<20)
	for left := n; left > 0; left -= int64(len(chunk)) {
		c := chunk[:min(int64(len(chunk)), left)]
		_, _ = fw.Write(c)
		_, _ = crc.Write(c)
	}
	if err := fw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), crc.Sum32()
}

// writeZip builds a ZIP at path; add fills it.
func writeZip(t *testing.T, path string, add func(w *zip.Writer)) string {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	add(w)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func addRaw(t *testing.T, w *zip.Writer, name string, data []byte, crc uint32, size int64) {
	t.Helper()
	fw, err := w.CreateRaw(&zip.FileHeader{
		Name: name, Method: zip.Deflate, CRC32: crc,
		CompressedSize64: uint64(len(data)), UncompressedSize64: uint64(size),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatal(err)
	}
}

// 50 pages of 90 MB each: every page is under the per-page cap, the whole is 4.4 GB. The
// listing is refused before a single page file is written.
func TestExtractCBZRefusesTotalBomb(t *testing.T) {
	data, crc := zeroDeflate(t, 90<<20)
	path := writeZip(t, filepath.Join(t.TempDir(), "bomb.cbz"), func(w *zip.Writer) {
		for i := range 50 {
			addRaw(t, w, fmt.Sprintf("page%02d.jpg", i), data, crc, 90<<20)
		}
	})
	out := t.TempDir()
	_, err := Extract(context.Background(), path, out)
	if !errors.Is(err, limits.ErrTooLarge) || !strings.Contains(err.Error(), "4 GB") {
		t.Fatalf("Extract = %v, want the total-size refusal naming 4 GB", err)
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Errorf("refused archive still wrote %d files", len(entries))
	}
}

func TestExtractCBZRefusesEntryCountBomb(t *testing.T) {
	path := writeZip(t, filepath.Join(t.TempDir(), "many.cbz"), func(w *zip.Writer) {
		for i := range limits.MaxArchiveEntries + 1 {
			if _, err := w.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("p%d.jpg", i), Method: zip.Store}); err != nil {
				t.Fatal(err)
			}
		}
	})
	_, err := Extract(context.Background(), path, t.TempDir())
	if !errors.Is(err, limits.ErrTooLarge) || !strings.Contains(err.Error(), "entries") {
		t.Fatalf("Extract = %v, want the entry-count refusal", err)
	}
}

// A page over the per-page cap is skipped by name from its declared size, and the others
// still convert.
func TestExtractCBZSkipsOversizePage(t *testing.T) {
	data, crc := zeroDeflate(t, maxPageBytes+1)
	path := writeZip(t, filepath.Join(t.TempDir(), "one-huge.cbz"), func(w *zip.Writer) {
		addRaw(t, w, "page1.jpg", data, crc, maxPageBytes+1)
		fw, _ := w.Create("page2.jpg")
		_, _ = fw.Write([]byte("TWO"))
	})
	var logBuf bytes.Buffer
	logging.StartRunLog(&logBuf)
	defer logging.StopRunLog()

	out := t.TempDir()
	book, err := Extract(context.Background(), path, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != 1 {
		t.Fatalf("pages = %d, want 1", len(book.Spine))
	}
	if got, _ := os.ReadFile(filepath.Join(out, "page_001.jpg")); string(got) != "TWO" {
		t.Errorf("page_001.jpg = %q, want the surviving page", got)
	}
	if !strings.Contains(logBuf.String(), "page1.jpg") || !strings.Contains(logBuf.String(), "200 MB") {
		t.Errorf("no named warning for the skipped page; log:\n%s", logBuf.String())
	}
}

// Done criterion 3, at test scale: pages are streamed to disk one at a time, so what the
// conversion allocates does not grow with the archive. The old reader held every page in
// memory at once and allocated at least the whole archive.
func TestExtractCBZStreamsPages(t *testing.T) {
	const pages, pageSize = 64, 1 << 20
	page := bytes.Repeat([]byte{0xAB}, pageSize)
	path := writeZip(t, filepath.Join(t.TempDir(), "big.cbz"), func(w *zip.Writer) {
		for i := range pages {
			fw, err := w.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("p%03d.png", i), Method: zip.Store})
			if err != nil {
				t.Fatal(err)
			}
			_, _ = fw.Write(page)
		}
	})
	out := t.TempDir()

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	book, err := Extract(context.Background(), path, out)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != pages {
		t.Fatalf("pages = %d, want %d", len(book.Spine), pages)
	}
	total := uint64(pages * pageSize)
	if grew := after.TotalAlloc - before.TotalAlloc; grew > total/4 {
		t.Errorf("Extract allocated %d bytes for a %d-byte archive; pages are not being streamed", grew, total)
	}
	if got, _ := os.ReadFile(filepath.Join(out, "page_064.png")); !bytes.Equal(got, page) {
		t.Error("last page content differs from the archive")
	}
}

func TestSniffContainer(t *testing.T) {
	tarHead := make([]byte, 262)
	copy(tarHead[257:], "ustar")
	cases := map[string][]byte{
		containerZip: []byte("PK\x03\x04rest"),
		containerRar: []byte("Rar!\x1a\x07\x01\x00"),
		container7z:  []byte("7z\xbc\xaf\x27\x1c\x00\x04"),
		containerTar: tarHead,
		"":           []byte("PK\x05\x06"), // an empty ZIP: fall back to the extension
	}
	for want, head := range cases {
		if got := sniffContainer(head); got != want {
			t.Errorf("sniffContainer(%q) = %q, want %q", head[:min(len(head), 8)], got, want)
		}
	}
}

// A CBT is read through recorded offsets rather than sequentially; a page must still land
// in its own file with its own bytes when the archive order is not the page order.
func TestExtractCBTOffsets(t *testing.T) {
	path := makeCBT(t, map[string][]byte{
		"b/page2.jpg": bytes.Repeat([]byte("2"), 700), // spans two TAR blocks
		"a/page1.jpg": []byte("1"),
		"page3.jpg":   bytes.Repeat([]byte("3"), 512),
	})
	out := t.TempDir()
	if _, err := Extract(context.Background(), path, out); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for name, want := range map[string]string{
		"page_001.jpg": "1", "page_002.jpg": strings.Repeat("2", 700), "page_003.jpg": strings.Repeat("3", 512),
	} {
		if got, _ := os.ReadFile(filepath.Join(out, name)); string(got) != want {
			t.Errorf("%s holds %d bytes, want %d", name, len(got), len(want))
		}
	}
}
