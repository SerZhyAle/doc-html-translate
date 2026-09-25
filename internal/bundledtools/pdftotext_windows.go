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

// pdftotextSet is what PDFToTextPath unpacks; a test swaps in its own set, since a source
// checkout may not carry the vendored pdftotext.exe.
var pdftotextSet fs.FS = pdftotextFS

const pdftotextExe = "pdftotext.exe"

var (
	pdftotextMu   sync.Mutex
	pdftotextPath string
)

// PDFToTextPath returns the path to the bundled pdftotext.exe, unpacking the bundled set into
// a folder named after its content hash on first use (see extractSet). It re-extracts when the
// cached file has gone, e.g. quarantined by antivirus after the first run. A build whose
// embedded set carries no pdftotext.exe (a source checkout without the vendored binary)
// reports ErrNotBundled.
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
