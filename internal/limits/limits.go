// Package limits holds the published input budgets (ticket bugfix-resource-budgets, section
// 6.1). The app ships a 386 build with a 2 GB address space, so one hostile or merely huge
// input must be turned into a message before it is allocated, not after. Every reader probes
// first - an image header, an archive listing - and asks this package whether the full read
// fits. The same numbers are enforced by the extension and pinned in docs/PARITY.md.
package limits

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"doc-html-translate/internal/i18n"
)

// Image budget for any full decode. 100 megapixels is a 600 DPI scan of an A3 page with room
// to spare, and 400 MB as RGBA - already a fifth of the 386 address space.
const (
	MaxImagePixels = 100_000_000
	MaxImageSide   = 32768
)

// Archive budget, checked from the listing before anything is unpacked. The per-entry cap is
// format specific (EPUB and comic readers keep their own, stricter ones).
const (
	MaxArchiveEntries          = 20000
	MaxArchiveTotalBytes int64 = 4 << 30
)

// MaxTextInputBytes is the budget for a document read whole: TXT, Markdown, FB2, RTF and HTML.
// It is the EPUB per-file cap, because an EPUB chapter goes through the same whole-file path;
// parsing and rendering hold several copies, so a larger file is what ends the 386 build.
const MaxTextInputBytes int64 = 100 << 20

// ErrTooLarge marks every refusal from this package, so a caller can tell a budget refusal
// from a corrupt file with errors.Is.
var ErrTooLarge = errors.New("input exceeds a size limit")

type tooLarge struct{ msg string }

func (e *tooLarge) Error() string        { return e.msg }
func (e *tooLarge) Is(target error) bool { return target == ErrTooLarge }

// CheckPixels refuses an image whose declared size is over the budget. int64 because a TIFF
// header declares uint32 sides, and both a side and the product overflow int on the 386 build.
func CheckPixels(w, h int64) error {
	if w <= 0 || h <= 0 {
		return nil
	}
	if w > MaxImageSide || h > MaxImageSide || w*h > MaxImagePixels {
		return &tooLarge{i18n.S("The image is %d x %d pixels, above the limit of %d megapixels and %d pixels per side",
			w, h, MaxImagePixels/1_000_000, MaxImageSide)}
	}
	return nil
}

// CheckArchive refuses an archive listing whose entry count or total unpacked size is over the
// budget. total is the sum over the entries the caller will actually unpack.
func CheckArchive(entries int, total uint64) error {
	if entries > MaxArchiveEntries {
		return &tooLarge{i18n.S("The archive has %d entries, above the limit of %d", entries, MaxArchiveEntries)}
	}
	if total > uint64(MaxArchiveTotalBytes) {
		return &tooLarge{i18n.S("The archive unpacks to %s, above the limit of %s", FormatBytes(total), FormatBytes(uint64(MaxArchiveTotalBytes)))}
	}
	return nil
}

// EntryTooLarge is the named refusal for one archive member over its per-entry cap.
func EntryTooLarge(name string, limit int64) error {
	return &tooLarge{i18n.S("%s unpacks to more than %s, the limit for one file", name, FormatBytes(uint64(limit)))}
}

// UncheckableListing refuses an archive whose listing cannot be held against the budget: an
// entry with no stated size, or a listing too long to read in full. Unpacking it anyway would
// be exactly the blind extraction the budget exists to prevent.
func UncheckableListing(name string) error {
	if name == "" {
		return &tooLarge{i18n.S("The archive listing is too long to check against the size limits")}
	}
	return &tooLarge{i18n.S("7-Zip did not report the unpacked size of %s, so the archive cannot be checked against the size limits", name)}
}

// CopyCapped copies src to dst and fails with EntryTooLarge when src holds more than limit
// bytes. The byte past the cap is read, never written: truncating silently at the cap is the
// defect this replaces (a chapter cut mid-tag reads as a complete book).
func CopyCapped(dst io.Writer, src io.Reader, name string, limit int64) (int64, error) {
	n, err := io.CopyN(dst, src, limit)
	if err == io.EOF {
		return n, nil
	}
	if err != nil {
		return n, err
	}
	// The probe read is also where a reader reports a failure it only detects at end of
	// stream - archive/zip's checksum and size checks - so an entry of exactly limit bytes
	// must not have that error dropped.
	var probe [1]byte
	m, perr := io.ReadFull(src, probe[:])
	if m > 0 {
		return n, EntryTooLarge(name, limit)
	}
	if perr != nil && perr != io.EOF {
		return n, perr
	}
	return n, nil
}

// TextInputTooLarge is the refusal for a whole-file document over MaxTextInputBytes.
func TextInputTooLarge(size int64) error {
	return &tooLarge{i18n.S("The document is %s, above the limit of %s for a text document",
		FormatBytes(uint64(size)), FormatBytes(uint64(MaxTextInputBytes)))}
}

// ReadTextInput reads a whole-file document, refusing one over MaxTextInputBytes from its size on
// disk before the read allocates anything. The read itself is capped as well, so a file that
// grows after the check is refused instead of read without bound.
func ReadTextInput(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() > MaxTextInputBytes {
		return nil, TextInputTooLarge(fi.Size())
	}
	var buf bytes.Buffer
	buf.Grow(int(fi.Size()) + bytes.MinRead)
	if _, err := buf.ReadFrom(io.LimitReader(f, MaxTextInputBytes+1)); err != nil {
		return nil, err
	}
	if int64(buf.Len()) > MaxTextInputBytes {
		return nil, TextInputTooLarge(int64(buf.Len()))
	}
	return buf.Bytes(), nil
}

// FormatBytes renders a size in binary units, the way the limits are published.
func FormatBytes(n uint64) string {
	switch {
	case n >= 1<<30 && n%(1<<30) == 0:
		return fmt.Sprintf("%d GB", n>>30)
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%d MB", n>>20)
	default:
		return fmt.Sprintf("%d KB", (n+1023)>>10)
	}
}
