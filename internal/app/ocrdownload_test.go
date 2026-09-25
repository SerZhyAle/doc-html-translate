package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/ocr"
)

// -ocr-download is an entry point of its own: a code outside the catalogue ends the run with the
// bad-arguments exit code before anything reaches the network or the disk (ticket 13).
func TestOCRDownloadRefusesATraversalCode(t *testing.T) {
	root := t.TempDir()
	// os.UserCacheDir reads these; the per-user tessdata folder would land under root.
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LocalAppData", root)
	t.Setenv("HOME", root)

	code, err := New(config.Config{OCRDownload: "../../x", UILang: "en"}).Run()
	if code != 1 || !errors.Is(err, ocr.ErrUnknownLang) {
		t.Errorf("Run = %d, %v; want 1 and ErrUnknownLang", code, err)
	}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && p != root {
			t.Errorf("a refused code created %s", p)
		}
		return nil
	})
}
