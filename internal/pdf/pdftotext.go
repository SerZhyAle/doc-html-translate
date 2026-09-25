package pdf

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"doc-html-translate/internal/bundledtools"
)

// isExecFileNotFound reports whether err came from a failed fork/exec attempt
// (file missing or access denied - typical when antivirus quarantines the binary).
func isExecFileNotFound(err error) bool {
	var pathErr *os.PathError
	return errors.As(err, &pathErr) && pathErr.Op == "fork/exec"
}

// findPDFToText locates the pdftotext binary: the bundled copy first (Windows only),
// then PATH, then the platform's well-known install locations.
func findPDFToText() string {
	if p, err := bundledtools.PDFToTextPath(); err == nil {
		return p
	}
	return findSystemPDFToText()
}

// findSystemPDFToText is like findPDFToText but skips the bundled binary.
// Used after the bundled copy has been blocked by antivirus.
func findSystemPDFToText() string {
	// exec.LookPath appends the PATHEXT suffixes on Windows, so the bare name works everywhere.
	if p, err := exec.LookPath("pdftotext"); err == nil {
		return p
	}
	for _, p := range pdftotextKnownPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	for _, pattern := range pdftotextKnownGlobs {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}
