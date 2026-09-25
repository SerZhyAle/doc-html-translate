//go:build windows

package bundledtools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// usePDFToTextSet points PDFToTextPath at set and at a private cache, and forgets any path an
// earlier call remembered.
func usePDFToTextSet(t *testing.T, set fstest.MapFS) string {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("LocalAppData", cache)
	origSet, origPath := pdftotextSet, pdftotextPath
	t.Cleanup(func() { pdftotextSet, pdftotextPath = origSet, origPath })
	pdftotextSet, pdftotextPath = set, ""
	return cache
}

func TestPDFToTextPathExtractsAndRecoversFromQuarantine(t *testing.T) {
	cache := usePDFToTextSet(t, fstest.MapFS{
		"pdftotext/pdftotext.exe":      {Data: []byte("MZ pdftotext")},
		"pdftotext/libgcc_s_seh-1.dll": {Data: []byte("runtime")},
	})

	p, err := PDFToTextPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p, CacheRoot()) || !strings.HasPrefix(CacheRoot(), cache) || filepath.Base(p) != pdftotextExe {
		t.Fatalf("PDFToTextPath = %q, want pdftotext.exe under %s", p, cache)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), "libgcc_s_seh-1.dll")); err != nil {
		t.Errorf("runtime DLL not unpacked next to the exe: %v", err)
	}

	// Antivirus quarantine removes the cached exe after the first run; the next call must
	// unpack it again rather than hand back a path that no longer exists.
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	again, err := PDFToTextPath()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(again); err != nil || string(got) != "MZ pdftotext" {
		t.Errorf("after quarantine %s = %q (err %v), want it re-extracted", again, got, err)
	}
}

// A source checkout without the vendored exe carries only the DLLs; that is "not bundled",
// so the caller looks on PATH instead of running nothing.
func TestPDFToTextPathWithoutExe(t *testing.T) {
	usePDFToTextSet(t, fstest.MapFS{"pdftotext/libgcc_s_seh-1.dll": {Data: []byte("runtime")}})
	if p, err := PDFToTextPath(); !errors.Is(err, ErrNotBundled) || p != "" {
		t.Errorf("PDFToTextPath = %q, %v; want ErrNotBundled", p, err)
	}
}

func TestPDFToTextPathReportsExtractionFailure(t *testing.T) {
	cache := usePDFToTextSet(t, fstest.MapFS{"pdftotext/pdftotext.exe": {Data: []byte("MZ")}})
	// The cache root exists as a file, so no folder can be made under it.
	if err := os.WriteFile(filepath.Join(cache, "doc-html-translate"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := PDFToTextPath()
	if err == nil || errors.Is(err, ErrNotBundled) || p != "" {
		t.Errorf("PDFToTextPath = %q, %v; want an extraction error", p, err)
	}
}
