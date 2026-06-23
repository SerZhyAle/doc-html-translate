package pdf

import (
	"os"
	"sort"
	"strings"

	"doc-html-translate/internal/epub"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// buildPDFTOC reads the PDF's bookmark (outline) hierarchy and maps it to a
// nested epub TOC. pageToHref maps 1-based PDF page numbers to the generated
// page hrefs. Returns nil when the PDF has no usable bookmarks; best-effort:
// any error or panic from the (re-parsing) pdfcpu reader yields no TOC.
func buildPDFTOC(pdfPath string, pageToHref map[int]string) (entries []epub.TOCEntry) {
	defer func() {
		if r := recover(); r != nil {
			entries = nil
		}
	}()

	if len(pageToHref) == 0 {
		return nil
	}

	f, err := os.Open(pdfPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	// pdfcpu returns (nil, nil) when a document simply has no bookmarks.
	bms, err := api.Bookmarks(f, nil)
	if err != nil || len(bms) == 0 {
		return nil
	}

	pages := make([]int, 0, len(pageToHref))
	for p := range pageToHref {
		pages = append(pages, p)
	}
	sort.Ints(pages)

	return bookmarksToEntries(bms, pageToHref, pages)
}

func bookmarksToEntries(bms []pdfcpu.Bookmark, pageToHref map[int]string, pages []int) []epub.TOCEntry {
	var out []epub.TOCEntry
	for _, bm := range bms {
		children := bookmarksToEntries(bm.Kids, pageToHref, pages)

		href := ""
		if bm.PageFrom >= 1 { // unresolved destinations come back as 0
			href = hrefForPDFPage(bm.PageFrom, pageToHref, pages)
		}

		title := strings.TrimSpace(bm.Title)
		if title == "" && href == "" && len(children) == 0 {
			continue
		}
		out = append(out, epub.TOCEntry{
			Title:    title,
			Href:     href,
			Children: children,
		})
	}
	return out
}

// hrefForPDFPage returns the href of the emitted page for a PDF page number,
// using the nearest emitted page at or before it (blank pages are skipped),
// clamped to the emitted range.
func hrefForPDFPage(page int, pageToHref map[int]string, pages []int) string {
	if h, ok := pageToHref[page]; ok {
		return h
	}
	if len(pages) == 0 {
		return ""
	}
	if page < pages[0] {
		return pageToHref[pages[0]]
	}
	best := pages[0]
	for _, p := range pages {
		if p <= page {
			best = p
		} else {
			break
		}
	}
	return pageToHref[best]
}
