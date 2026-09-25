//go:build !windows

package bundledtools

import (
	"errors"
	"testing"
)

// The embedded pdftotext is a Windows executable; elsewhere the caller must fall back to PATH.
func TestPDFToTextPathNotBundledOffWindows(t *testing.T) {
	if p, err := PDFToTextPath(); !errors.Is(err, ErrNotBundled) || p != "" {
		t.Errorf("PDFToTextPath = %q, %v; want ErrNotBundled", p, err)
	}
}
