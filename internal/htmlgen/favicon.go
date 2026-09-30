package htmlgen

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/fsutil"
)

// faviconICO is the app icon. Kept in step with assets/doc-html-translate.ico by
// scripts/build.ps1, the same way cmd/doc-html-ui/favicon.ico is kept in step by
// scripts/build-ui.ps1.
//
//go:embed favicon.ico
var faviconICO []byte

// FaviconName is the file written beside index.html and linked from every generated page.
// Exported so extractors that copy source assets into the output can keep clear of it.
const FaviconName = "favicon.ico"

// WriteFavicon drops the app icon into the output folder so a converted book carries its
// own tab icon instead of the browser's blank-page glyph - the reader usually has several
// of these tabs open at once, and they are otherwise indistinguishable.
//
// A sibling file, not a data: URI inlined into each page: the icon is ~21 KB, so a
// multi-page book would carry a copy of it in every one of hundreds of files. The output
// is already a folder of related files (pdf_images/ and friends), so one more is free.
// Best-effort - a result without a tab icon is still a result.
func WriteFavicon(outputDir string) {
	if len(faviconICO) == 0 {
		return
	}
	_ = os.WriteFile(filepath.Join(outputDir, FaviconName), faviconICO, 0o644)
}

// faviconLink is the <link> tag for a page sitting in fromDir, relative to the output
// root ("" or "." for a page at the root itself).
func faviconLink(fromDir string) string {
	return `<link rel="icon" href="` + relativePath(fromDir, FaviconName) + "\">\n"
}

// indexASCIIFold finds an ASCII HTML tag without changing offsets in the original bytes.
// Unicode case folding can change byte lengths (for example, U+0130 in Turkish titles).
func indexASCIIFold(content, tag string) int {
	for i := 0; i+len(tag) <= len(content); i++ {
		matched := true
		for j := 0; j < len(tag); j++ {
			b := content[i+j]
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if b != tag[j] {
				matched = false
				break
			}
		}
		if matched {
			return i
		}
	}
	return -1
}

// injectFavicon adds the tab-icon link to a page that was already written. The
// single-spine index path uses it before adding reader controls to the content page.
// Best-effort, like the rest of the icon handling.
func injectFavicon(filePath, fromDir string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	content := string(data)
	idx := indexASCIIFold(content, "</head>")
	if idx < 0 {
		return
	}
	if containsGeneratedFavicon(content, fromDir) {
		return
	}
	_ = fsutil.WriteFile(filePath, []byte(content[:idx]+faviconLink(fromDir)+content[idx:]), 0o644)
}

func containsGeneratedFavicon(content, fromDir string) bool {
	return strings.Contains(content, faviconLink(fromDir))
}
