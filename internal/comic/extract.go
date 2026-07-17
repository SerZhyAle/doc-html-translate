// Package comic handles comic-book archives (CBZ / CBR / CB7 / CBT) as input.
// A comic archive is a container of page images with no text layer at all - every
// word lives inside the artwork, in speech bubbles and captions. So, like a
// standalone image (internal/img), there is nothing to "extract" in the usual
// sense: each page image is wrapped in a one-page HTML document and the pipeline's
// OCR overlay step (internal/ocr) is forced on, digitizing the text and laying
// translatable plates over each page. Opened in Chrome, the book reads page by
// page with the built-in "Translate to.." working on the bubbles.
//
// The four extensions are three different containers: CBZ is ZIP and CBT is TAR
// (both pure stdlib); CBR is RAR and CB7 is 7z, which have no Go decoder and so
// shell out to 7-Zip (see readers_sevenzip.go), the same detect-or-notice
// precedent as MOBI/Calibre.
package comic

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
)

// exts are the comic-archive extensions (lower-case, leading dot) treated as
// comic input.
var exts = map[string]bool{
	".cbz": true,
	".cbr": true,
	".cb7": true,
	".cbt": true,
}

// IsComic reports whether ext (lower-case, with leading dot) is a supported comic
// archive input.
func IsComic(ext string) bool { return exts[ext] }

// pageExts are the image extensions that count as a comic page. TIFF is
// deliberately excluded: Chrome cannot display it and it is vanishingly rare in
// comic archives, so a .tif entry is treated as a non-page and ignored rather
// than rendered as a broken image with plates floating over white space. This set
// must match the extension's page-entry filter (see docs/PARITY.md).
var pageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
}

// Archive-hardening caps. Comic archives are large and untrusted by nature, so a
// single entry, the whole archive, and the page count are each bounded. These are
// generous enough for any real book and only exist to refuse a decompression bomb.
const (
	maxPageBytes  = 200 * 1024 * 1024        // 200 MB for one page image
	maxTotalBytes = int64(4 * 1024 * 1024 * 1024) // 4 GB inflated total
	maxPages      = 20000                     // page count ceiling
)

// page is one decoded archive entry: its original name (used only for ordering)
// and its bytes.
type page struct {
	name string
	data []byte
}

// Extract reads a comic archive, writes each page image and a one-page HTML
// wrapper into outputDir in natural filename order, and returns an *epub.Book the
// pipeline can run through its single-page flow. The caller forces the OCR overlay
// step (comics have no text layer), so opening the result shows every page with
// translatable text plates laid over the bubbles.
func Extract(comicPath, outputDir string) (*epub.Book, error) {
	ext := strings.ToLower(filepath.Ext(comicPath))
	title := strings.TrimSuffix(filepath.Base(comicPath), filepath.Ext(comicPath))

	var pages []page
	var err error
	switch ext {
	case ".cbz":
		pages, err = readCBZ(comicPath)
	case ".cbt":
		pages, err = readCBT(comicPath)
	case ".cbr", ".cb7":
		pages, err = readSevenZip(comicPath, ext)
	default:
		return nil, fmt.Errorf("not a comic archive: %s", ext)
	}
	if err != nil {
		return nil, err
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no page images found in %s (looked for %s)",
			filepath.Base(comicPath), strings.Join(sortedPageExts(), ", "))
	}

	// Natural, numeric-aware order: page order *is* archive entry order by
	// filename, so this sort is correctness, not cosmetics.
	sort.SliceStable(pages, func(i, j int) bool { return naturalLess(pages[i].name, pages[j].name) })

	book := &epub.Book{Title: title}
	for i, pg := range pages {
		pageNum := i + 1
		imgName := fmt.Sprintf("page_%03d%s", pageNum, strings.ToLower(filepath.Ext(pg.name)))
		if werr := os.WriteFile(filepath.Join(outputDir, imgName), pg.data, 0o644); werr != nil {
			return nil, fmt.Errorf("write page %d: %w", pageNum, werr)
		}
		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)
		if werr := os.WriteFile(filepath.Join(outputDir, href), []byte(buildPageHTML(title, imgName)), 0o644); werr != nil {
			return nil, fmt.Errorf("write page %d html: %w", pageNum, werr)
		}
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "text/html"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Pages: %d\n", len(book.Spine))
	return book, nil
}

// isPageEntry reports whether an archive entry name is a comic page: a regular
// file with a page image extension, not a directory, not archive metadata
// (ComicInfo.xml), not OS cruft (Thumbs.db, .DS_Store, anything under __MACOSX/),
// and not a hidden dotfile. This filter must match the extension's, or the two
// editions disagree on page count and numbering for the same file.
func isPageEntry(name string) bool {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.HasSuffix(name, "/") {
		return false // directory entry
	}
	if strings.HasPrefix(name, "__MACOSX/") || strings.Contains(name, "/__MACOSX/") {
		return false
	}
	base := name
	if i := strings.LastIndex(name, "/"); i >= 0 {
		base = name[i+1:]
	}
	if base == "" || strings.HasPrefix(base, ".") {
		return false // hidden dotfile (.DS_Store, ._resource forks)
	}
	if strings.EqualFold(base, "Thumbs.db") || strings.EqualFold(base, "ComicInfo.xml") {
		return false
	}
	return pageExts[strings.ToLower(filepath.Ext(base))]
}

// buildPageHTML wraps a page image in a minimal centred page, mirroring
// internal/img so the OCR overlay step (which finds each <img>, OCRs it, and
// appends translatable text plates) behaves identically for comics and images.
func buildPageHTML(title, imgName string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { margin: 0; background: #f0f0f0; }\n")
	sb.WriteString("    main { width: 95%; max-width: 1400px; margin: 1em auto; }\n")
	sb.WriteString("    main img { display: block; width: 100%; height: auto; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString("  <main>\n")
	sb.WriteString(fmt.Sprintf("    <img src=\"%s\" alt=\"%s\">\n", html.EscapeString(imgName), html.EscapeString(title)))
	sb.WriteString("  </main>\n</body>\n</html>\n")
	return sb.String()
}

func sortedPageExts() []string {
	out := make([]string, 0, len(pageExts))
	for e := range pageExts {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}
