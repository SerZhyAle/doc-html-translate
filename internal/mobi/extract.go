// Package mobi handles MOBI/AZW3 file conversion and extraction.
// It delegates to Calibre's ebook-convert binary to produce an EPUB,
// then reuses the existing EPUB pipeline for all further processing.
// DRM-protected files are not supported.
package mobi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
)

// findEbookConvert locates the Calibre ebook-convert binary via PATH or
// well-known installation paths.
func findEbookConvert() string {
	if p, err := exec.LookPath("ebook-convert"); err == nil {
		return p
	}
	for _, p := range []string{
		`C:\Program Files\Calibre2\ebook-convert.exe`,
		`C:\Program Files (x86)\Calibre2\ebook-convert.exe`,
		`C:\Program Files\Calibre\ebook-convert.exe`,
		`/usr/bin/ebook-convert`,
		`/usr/local/bin/ebook-convert`,
		`/Applications/calibre.app/Contents/MacOS/ebook-convert`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Extract converts a MOBI or AZW3 file to EPUB via Calibre's ebook-convert,
// then extracts the result using the EPUB pipeline.
// Returns an error if Calibre is not installed or if the file is DRM-protected.
func Extract(mobiPath, outputDir string) (*epub.Book, error) {
	bin := findEbookConvert()
	if bin == "" {
		return nil, fmt.Errorf(
			"Calibre not found — install Calibre from https://calibre-ebook.com to open .mobi/.azw3 files;\n" +
				"after installing, re-register this app with: doc-html-translate.exe -register",
		)
	}

	tmpDir, err := os.MkdirTemp("", "doc-html-translate-mobi-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	baseName := filepath.Base(mobiPath)
	baseName = baseName[:len(baseName)-len(filepath.Ext(baseName))]
	epubPath := filepath.Join(tmpDir, baseName+".epub")

	logging.Println("  Converting MOBI → EPUB via Calibre ebook-convert...")
	cmd := exec.Command(bin, mobiPath, epubPath)
	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		msg := string(out)
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		return nil, fmt.Errorf("ebook-convert failed (file may be DRM-protected): %w\n%s", cmdErr, msg)
	}

	if _, err := os.Stat(epubPath); err != nil {
		return nil, fmt.Errorf("ebook-convert produced no output EPUB at %s", epubPath)
	}

	logging.Println("  Extracting converted EPUB...")
	book, err := epub.Extract(epubPath, outputDir)
	if err != nil {
		return nil, fmt.Errorf("extract converted epub: %w", err)
	}

	return book, nil
}
