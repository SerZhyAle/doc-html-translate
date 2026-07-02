// Package outputpath computes the output directory name for a converted document.
// It has no dependency on any format extractor or the GUI's process model, so both
// internal/pipeline (the CLI) and cmd/doc-html-ui (the GUI, checking/clearing a
// previous result without shelling out) can share the exact same naming rules.
package outputpath

import (
	"path/filepath"
	"strconv"
	"strings"
)

// OutputDirFor returns the output directory for a given input file.
// If folder is non-empty, the result is placed inside that folder.
// Otherwise it falls back to the directory of the input file (original behaviour).
//
// Example (folder=""):        /path/to/My Book.epub → /path/to/My Book/
// Example (folder="C:/out"): /path/to/My Book.epub → C:/out/My Book/
func OutputDirFor(filePath, folder string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := sanitizeOutputName(strings.TrimSuffix(base, ext))
	if folder != "" {
		return filepath.Join(folder, name)
	}
	return filepath.Join(filepath.Dir(filePath), name)
}

func sanitizeOutputName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, ". ")
	if name == "" || name == "." || name == ".." {
		return "document"
	}

	if isWindowsReservedName(name) {
		name += "_"
	}

	return name
}

func isWindowsReservedName(name string) bool {
	upper := strings.ToUpper(name)
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" {
		return true
	}

	if strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT") {
		if len(upper) == 4 {
			n, err := strconv.Atoi(upper[3:])
			if err == nil && n >= 1 && n <= 9 {
				return true
			}
		}
	}

	return false
}
