// Package bundledtools embeds and extracts external binaries bundled with the app.
package bundledtools

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

//go:embed pdftotext
var pdftotextFS embed.FS

var (
	pdftotextOnce sync.Once
	pdftotextPath string
	pdftotextErr  error
)

// PDFToTextPath returns the path to the bundled pdftotext.exe, extracting it
// to the user cache directory on first call. Subsequent calls reuse the cache.
func PDFToTextPath() (string, error) {
	pdftotextOnce.Do(func() {
		pdftotextPath, pdftotextErr = extractPDFToText()
	})
	return pdftotextPath, pdftotextErr
}

func extractPDFToText() (string, error) {
	cacheBase, err := os.UserCacheDir()
	if err != nil {
		cacheBase = os.TempDir()
	}
	toolDir := filepath.Join(cacheBase, "doc-html-translate", "pdftotext")
	exePath := filepath.Join(toolDir, "pdftotext.exe")

	if _, err := os.Stat(exePath); err == nil {
		return exePath, nil
	}

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return "", fmt.Errorf("create pdftotext cache dir: %w", err)
	}

	entries, err := pdftotextFS.ReadDir("pdftotext")
	if err != nil {
		return "", fmt.Errorf("read embedded pdftotext: %w", err)
	}
	for _, e := range entries {
		data, err := pdftotextFS.ReadFile("pdftotext/" + e.Name())
		if err != nil {
			return "", fmt.Errorf("read embedded %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(toolDir, e.Name()), data, 0o755); err != nil {
			return "", fmt.Errorf("write %s: %w", e.Name(), err)
		}
	}

	return exePath, nil
}
