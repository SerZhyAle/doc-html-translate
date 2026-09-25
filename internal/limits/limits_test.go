package limits

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCheckPixels(t *testing.T) {
	cases := []struct {
		w, h int64
		ok   bool
	}{
		{10000, 10000, true},
		{MaxImageSide, 3000, true},
		{MaxImageSide + 1, 10, false},
		{10, MaxImageSide + 1, false},
		{20000, 20000, false}, // under the side limit, over the pixel budget
		{60000, 60000, false},
		{65535, 65535, false}, // w*h overflows a 32-bit int
		{0, 0, true},          // unknown size is the caller's decision
	}
	for _, c := range cases {
		err := CheckPixels(c.w, c.h)
		if (err == nil) != c.ok {
			t.Errorf("CheckPixels(%d, %d) = %v, want ok=%v", c.w, c.h, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrTooLarge) {
			t.Errorf("CheckPixels(%d, %d) error is not ErrTooLarge", c.w, c.h)
		}
	}
}

func TestCheckArchive(t *testing.T) {
	if err := CheckArchive(MaxArchiveEntries, uint64(MaxArchiveTotalBytes)); err != nil {
		t.Errorf("at the limits: %v", err)
	}
	if err := CheckArchive(MaxArchiveEntries+1, 0); !errors.Is(err, ErrTooLarge) {
		t.Errorf("entry count over the limit: %v", err)
	}
	if err := CheckArchive(1, uint64(MaxArchiveTotalBytes)+1); !errors.Is(err, ErrTooLarge) {
		t.Errorf("total over the limit: %v", err)
	}
}

// One byte past the cap is an error, never a silently shortened file.
func TestCopyCapped(t *testing.T) {
	var out bytes.Buffer
	if n, err := CopyCapped(&out, strings.NewReader("12345"), "a", 5); err != nil || n != 5 {
		t.Fatalf("exactly at the cap: n=%d err=%v", n, err)
	}
	out.Reset()
	_, err := CopyCapped(&out, strings.NewReader("123456"), "big.bin", 5)
	if !errors.Is(err, ErrTooLarge) || !strings.Contains(err.Error(), "big.bin") {
		t.Fatalf("one byte past the cap: err=%v", err)
	}
	if out.Len() != 5 {
		t.Errorf("wrote %d bytes, want the 5 under the cap", out.Len())
	}
}

func TestFormatBytes(t *testing.T) {
	for n, want := range map[uint64]string{100 << 20: "100 MB", 4 << 30: "4 GB", 5 << 29: "2.5 GB", 1: "1 KB"} {
		if got := FormatBytes(n); got != want {
			t.Errorf("FormatBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
