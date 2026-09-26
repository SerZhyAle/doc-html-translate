package mobi_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/mobi"
)

// TestExtract_NoCalibre verifies that a useful error is returned when
// Calibre is not installed. Skipped if ebook-convert is actually present.
func TestExtract_NoCalibre(t *testing.T) {
	if _, err := exec.LookPath("ebook-convert"); err == nil {
		t.Skip("ebook-convert found on PATH — skipping no-calibre error test")
	}

	dir := t.TempDir()
	mobiPath := filepath.Join(dir, "test.mobi")
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)
	_ = os.WriteFile(mobiPath, []byte("PalmDOC dummy"), 0o644)

	_, err := mobi.Extract(context.Background(), mobiPath, outDir)
	if err == nil {
		t.Fatal("expected error when Calibre is not installed, got nil")
	}
	if !strings.Contains(err.Error(), "calibre (ebook-convert) not found") {
		t.Errorf("expected 'calibre (ebook-convert) not found' in error, got: %v", err)
	}
}

// TestExtract_FileNotFound verifies that Extract returns an error for a
// non-existent input file (either from missing Calibre or from ebook-convert
// failing on the missing file).
func TestExtract_FileNotFound(t *testing.T) {
	_, err := mobi.Extract(context.Background(), "/nonexistent/path/file.mobi", t.TempDir())
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
