// Package outputpath computes the output directory name for a converted document.
// It has no dependency on any format extractor or the GUI's process model, so both
// internal/pipeline (the CLI) and cmd/doc-html-ui (the GUI, checking/clearing a
// previous result without shelling out) can share the exact same naming rules.
package outputpath

import (
	"path/filepath"
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
	// Characters Windows refuses in a file name arrive from non-Windows names and from
	// -folder values; control characters are invisible and break shell hand-offs.
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, ". ")
	if strings.Trim(name, ".") == "" {
		return "document"
	}

	// Windows maps a device name to the device whatever follows its first dot
	// ("CON.tar" is CON), so the check is on the stem and the suffix goes there too.
	stem, rest, _ := strings.Cut(name, ".")
	if isWindowsReservedName(strings.TrimRight(stem, " ")) {
		name = stem + "_"
		if rest != "" {
			name += "." + rest
		}
	}

	return name
}

func isWindowsReservedName(name string) bool {
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}

	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT") {
		// COM0-9 / LPT0-9 plus the superscript digits Windows also treats as ports.
		switch suffix := []rune(name)[3:]; {
		case len(suffix) != 1:
		case suffix[0] >= '0' && suffix[0] <= '9':
			return true
		case suffix[0] == '¹' || suffix[0] == '²' || suffix[0] == '³':
			return true
		}
	}

	return false
}
