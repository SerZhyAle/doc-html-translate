//go:build windows

package bundledtools

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

//go:embed pdftotext
var pdftotextFS embed.FS

// pdftotextSet is what PDFToTextPath unpacks; a test swaps in its own set so it does not
// depend on the vendored binaries.
var pdftotextSet fs.FS = pdftotextFS

const pdftotextExe = "pdftotext.exe"

var (
	pdftotextMu   sync.Mutex
	pdftotextPath string
)

// PDFToTextBundled reports whether this build carries a pdftotext.exe. It only looks into the embedded set: nothing is
// unpacked and nothing is written, which the capabilities report relies on.
func PDFToTextBundled() bool {
	_, err := fs.Stat(pdftotextSet, "pdftotext/"+pdftotextExe)
	return err == nil
}

// PDFToTextPath returns the path to the bundled pdftotext.exe, unpacking the bundled set into
// a folder named after its content hash on first use (see extractSet). It re-extracts when the
// cached file has gone, e.g. quarantined by antivirus after the first run. A build whose
// embedded set carries no pdftotext.exe reports ErrNotBundled; the release workflow refuses
// to build such a set.
func PDFToTextPath() (string, error) {
	pdftotextMu.Lock()
	defer pdftotextMu.Unlock()
	if pdftotextPath != "" {
		if _, err := os.Stat(pdftotextPath); err == nil {
			return pdftotextPath, nil
		}
	}
	if _, err := fs.Stat(pdftotextSet, "pdftotext/"+pdftotextExe); err != nil {
		return "", ErrNotBundled
	}
	dir, err := extractSet(pdftotextSet, "pdftotext", CacheRoot(), "pdftotext")
	if err != nil {
		return "", err
	}
	pdftotextPath = filepath.Join(dir, pdftotextExe)
	return pdftotextPath, nil
}
