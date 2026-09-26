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
	"context"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/limits"
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

// maxPageBytes caps one page image. Stricter than the general archive budget
// (internal/limits) on purpose: no real comic page comes near it. The entry count
// and the unpacked total are the shared archive limits, and since every page is
// an entry, the entry cap is also the page cap.
const maxPageBytes = 200 << 20

// entry is one regular file in an archive listing. open streams its bytes; the
// size is the one the listing states, checked against the budget before any
// entry is unpacked.
type entry struct {
	name string
	size int64
	open func() (io.ReadCloser, error)
}

// archive is a listed comic container. Pages are never held in memory: each one
// is streamed from here to its page file, so the working set does not grow with
// the page count (a 2 GB CBZ used to be read whole into RAM on a 386 build).
type archive struct {
	entries []entry // regular files only
	count   int     // every entry in the listing, directories and links included
	// fetch, when set, makes the chosen pages readable - 7-Zip unpacks only those.
	// Without it every entry opens straight from the archive.
	fetch func(pages []entry) ([]entry, error)
	close func()
}

// Extract reads a comic archive, writes each page image and a one-page HTML
// wrapper into outputDir in natural filename order, and returns an *epub.Book the
// pipeline can run through its single-page flow. The caller forces the OCR overlay
// step (comics have no text layer), so opening the result shows every page with
// translatable text plates laid over the bubbles.
//
// The container is identified by its signature and the extension is only the
// fallback: a RAR renamed to .cbz is common in the wild and used to fail as a
// "corrupt ZIP".
func Extract(ctx context.Context, comicPath, outputDir string) (*epub.Book, error) {
	ext := strings.ToLower(filepath.Ext(comicPath))
	if !exts[ext] {
		return nil, fmt.Errorf("not a comic archive: %s", ext)
	}
	title := strings.TrimSuffix(filepath.Base(comicPath), filepath.Ext(comicPath))
	kind := containerKind(comicPath, ext)

	var arc *archive
	var err error
	switch kind {
	case containerZip:
		arc, err = openCBZ(comicPath)
	case containerTar:
		arc, err = openCBT(comicPath)
	default:
		arc, err = openSevenZip(ctx, comicPath, ext, kind)
	}
	if err != nil {
		return nil, err
	}
	defer func() { arc.close() }() // fetch may replace close with the temp-dir cleanup

	pages, err := selectPages(arc)
	if err != nil {
		return nil, err
	}
	if arc.fetch != nil && len(pages) > 0 {
		if pages, err = arc.fetch(pages); err != nil {
			return nil, err
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no page images found in %s (looked for %s)",
			filepath.Base(comicPath), strings.Join(sortedPageExts(), ", "))
	}

	book := &epub.Book{Title: title}
	for _, pg := range pages {
		pageNum := len(book.Spine) + 1
		imgName := fmt.Sprintf("page_%03d%s", pageNum, strings.ToLower(filepath.Ext(pg.name)))
		if werr := writePage(filepath.Join(outputDir, imgName), pg); werr != nil {
			// One unreadable page costs that page, not the book - the contract the
			// in-memory readers kept when a read failed.
			logging.Printf("  WARNING: comic page %s could not be read, skipped: %v\n", pg.name, werr)
			continue
		}
		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)
		// len(pages) can overstate the total by the pages skipped above; that only
		// touches the alt text, the same trade internal/img makes for TIFF frames.
		if werr := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(buildPageHTML(title, imgName, pageNum, len(pages))), 0o644); werr != nil {
			return nil, fmt.Errorf("write page %d html: %w", pageNum, werr)
		}
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "text/html"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}
	if len(book.Spine) == 0 {
		return nil, fmt.Errorf("no readable page images in %s", filepath.Base(comicPath))
	}

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Pages: %d\n", len(book.Spine))
	return book, nil
}

// selectPages keeps the page entries of a listing in natural order, skipping (by
// name) a page over maxPageBytes, and refuses the whole archive when the listing
// is over the entry-count or unpacked-total budget. Nothing has been unpacked yet
// when it runs.
func selectPages(arc *archive) ([]entry, error) {
	var pages []entry
	var total uint64
	for _, e := range arc.entries {
		if !isPageEntry(e.name) {
			continue
		}
		if e.size > maxPageBytes {
			logging.Printf("  WARNING: %v\n", limits.EntryTooLarge(e.name, maxPageBytes))
			continue
		}
		pages = append(pages, e)
		total += uint64(e.size)
	}
	if err := limits.CheckArchive(arc.count, total); err != nil {
		return nil, err
	}
	// Natural, numeric-aware order: page order *is* archive entry order by
	// filename, so this sort is correctness, not cosmetics.
	sort.SliceStable(pages, func(i, j int) bool { return naturalLess(pages[i].name, pages[j].name) })
	return pages, nil
}

// writePage streams one page into path, failing (never truncating) when the entry
// holds more than its listing promised the cap allows.
func writePage(path string, pg entry) error {
	rc, err := pg.open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return fsutil.Write(path, 0o644, func(w io.Writer) error {
		_, cerr := limits.CopyCapped(w, rc, pg.name, maxPageBytes)
		return cerr
	})
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
//
// The page is a <section id="page_NNN">, not a <main>: these wrappers are merged
// verbatim into the single-page index.html, where a <main> per page would nest
// dozens of them inside the merged document's own <main> - invalid HTML that
// breaks reader mode and screen-reader landmark navigation. The id survives the
// merge, so every page in the merged book is linkable and reachable by anchor.
func buildPageHTML(title, imgName string, pageNum, totalPages int) string {
	alt := fmt.Sprintf("%s - page %d of %d", title, pageNum, totalPages)
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { margin: 0; }\n")
	sb.WriteString("    section.dht-page { width: 95%; max-width: 1400px; margin: 1em auto; }\n")
	sb.WriteString("    section.dht-page img { display: block; width: 100%; height: auto; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(fmt.Sprintf("  <section class=\"dht-page\" id=\"page_%03d\" aria-label=\"%s\">\n",
		pageNum, html.EscapeString(alt)))
	sb.WriteString(fmt.Sprintf("    <img src=\"%s\" alt=\"%s\">\n", html.EscapeString(imgName), html.EscapeString(alt)))
	sb.WriteString("  </section>\n</body>\n</html>\n")
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
