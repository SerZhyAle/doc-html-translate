package limits

import (
	"archive/zip"
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"strings"
	"testing"
)

// errAtEOF yields its payload, then a non-EOF error where a clean stream would say io.EOF -
// the way archive/zip reports a checksum mismatch.
type errAtEOF struct {
	r   io.Reader
	err error
}

func (e *errAtEOF) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF {
		return n, e.err
	}
	return n, err
}

// An entry of exactly limit bytes reaches end of stream on the probe read, so an error the
// reader only reports there must come back, not be read as a clean finish.
func TestCopyCappedReturnsErrorAtExactLimit(t *testing.T) {
	want := errors.New("checksum error")
	var dst bytes.Buffer
	n, err := CopyCapped(&dst, &errAtEOF{r: strings.NewReader("abcd"), err: want}, "x", 4)
	if n != 4 || !errors.Is(err, want) {
		t.Fatalf("CopyCapped = %d, %v; want 4, %v", n, err, want)
	}
}

func TestCopyCappedExactLimitCleanEOF(t *testing.T) {
	var dst bytes.Buffer
	n, err := CopyCapped(&dst, strings.NewReader("abcd"), "x", 4)
	if n != 4 || err != nil || dst.String() != "abcd" {
		t.Fatalf("CopyCapped = %d, %v, %q; want 4, nil, abcd", n, err, dst.String())
	}
}

// The real case: a zip entry of exactly limit bytes whose stored CRC is wrong.
func TestCopyCappedZipChecksumAtExactLimit(t *testing.T) {
	payload := []byte("hello world")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.CreateRaw(&zip.FileHeader{
		Name:               "a.txt",
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE(payload) ^ 1,
		CompressedSize64:   uint64(len(payload)),
		UncompressedSize64: uint64(len(payload)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	_, err = CopyCapped(io.Discard, rc, "a.txt", int64(len(payload)))
	if !errors.Is(err, zip.ErrChecksum) {
		t.Fatalf("CopyCapped err = %v; want zip.ErrChecksum", err)
	}
}
