package epub

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/limits"
	"doc-html-translate/internal/logging"
)

// zeroDeflate returns the raw DEFLATE stream and CRC of n zero bytes. Zeros compress about a
// thousandfold, so a real multi-gigabyte bomb costs the test a few megabytes on disk.
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

// addRaw adds a pre-compressed entry whose header states size, so the listing declares what
// the bomb will inflate to without the test ever inflating it.
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

// bombEPUB writes a minimal valid EPUB plus whatever extra entries add writes.
func bombEPUB(t *testing.T, add func(w *zip.Writer)) string {
	t.Helper()
	src := createMinimalEPUB(t, t.TempDir())
	r, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	path := filepath.Join(t.TempDir(), "bomb.epub")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, e := range r.File {
		if err := w.Copy(e); err != nil {
			t.Fatal(err)
		}
	}
	add(w)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// The total budget is checked from the listing: 50 entries of 90 MB each (4.4 GB unpacked,
// every one under the per-file cap) are refused before anything reaches the disk.
func TestExtractRefusesTotalBomb(t *testing.T) {
	data, crc := zeroDeflate(t, 90<<20)
	path := bombEPUB(t, func(w *zip.Writer) {
		for i := range 50 {
			addRaw(t, w, fmt.Sprintf("OEBPS/fill%02d.bin", i), data, crc, 90<<20)
		}
	})
	out := t.TempDir()
	_, err := Extract(path, out)
	if !errors.Is(err, limits.ErrTooLarge) || !strings.Contains(err.Error(), "4 GB") {
		t.Fatalf("Extract = %v, want the total-size refusal naming 4 GB", err)
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Errorf("refused archive still wrote %d entries", len(entries))
	}
}

func TestExtractRefusesEntryCountBomb(t *testing.T) {
	path := bombEPUB(t, func(w *zip.Writer) {
		for i := range limits.MaxArchiveEntries {
			if _, err := w.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("x/%d", i), Method: zip.Store}); err != nil {
				t.Fatal(err)
			}
		}
	})
	_, err := Extract(path, t.TempDir())
	if !errors.Is(err, limits.ErrTooLarge) || !strings.Contains(err.Error(), "entries") {
		t.Fatalf("Extract = %v, want the entry-count refusal", err)
	}
}

// One entry over the per-file cap is skipped with a warning that names it, and the rest of the
// book still converts. It used to be cut at 100 MB without a word.
func TestExtractSkipsOversizeEntryByName(t *testing.T) {
	data, crc := zeroDeflate(t, maxEntryBytes+1)
	path := bombEPUB(t, func(w *zip.Writer) {
		addRaw(t, w, "OEBPS/huge.bin", data, crc, maxEntryBytes+1)
	})
	var logBuf bytes.Buffer
	logging.StartRunLog(&logBuf)
	defer logging.StopRunLog()

	out := t.TempDir()
	book, err := Extract(path, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != 2 {
		t.Errorf("spine = %d, want the book's 2 chapters", len(book.Spine))
	}
	if _, err := os.Stat(filepath.Join(out, "OEBPS", "huge.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("oversize entry was written (stat err %v)", err)
	}
	if !strings.Contains(logBuf.String(), "huge.bin") || !strings.Contains(logBuf.String(), "100 MB") {
		t.Errorf("no named warning for the skipped entry; log:\n%s", logBuf.String())
	}
}

// An entry whose header understates its size is caught while unpacking, and the partial file
// is removed rather than left behind as if it were whole.
func TestExtractFileRemovesOverflowingEntry(t *testing.T) {
	data, crc := zeroDeflate(t, 4096)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	addRaw(t, w, "a.bin", data, crc, 1024)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := extractFile(r.File[0], dir); err == nil {
		t.Fatal("an entry inflating past its declared size must fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "a.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("partial file left behind (stat err %v)", err)
	}
}
