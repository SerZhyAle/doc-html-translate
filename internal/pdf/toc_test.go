package pdf

import (
	"path/filepath"
	"testing"

	"doc-html-translate/internal/epub"

	"github.com/go-pdf/fpdf"
)

func TestHrefForPDFPage_Nearest(t *testing.T) {
	pageToHref := map[int]string{2: "page_001.html", 4: "page_002.html", 7: "page_003.html"}
	pages := []int{2, 4, 7}

	cases := []struct {
		page int
		want string
	}{
		{2, "page_001.html"}, // exact
		{4, "page_002.html"}, // exact
		{5, "page_002.html"}, // nearest emitted page <= 5
		{6, "page_002.html"}, // nearest emitted page <= 6
		{9, "page_003.html"}, // beyond last -> clamp to last
		{1, "page_001.html"}, // before first -> clamp to first
	}
	for _, c := range cases {
		if got := hrefForPDFPage(c.page, pageToHref, pages); got != c.want {
			t.Errorf("hrefForPDFPage(%d) = %q, want %q", c.page, got, c.want)
		}
	}
}

func TestBuildPDFTOC_FromBookmarks(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "bm.pdf")

	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetFont("Helvetica", "", 12)
	doc.AddPage()
	doc.Bookmark("Chapter 1", 0, 0)
	doc.CellFormat(0, 6, "one", "", 1, "", false, 0, "")
	doc.AddPage()
	doc.Bookmark("Section 1.1", 1, 0)
	doc.CellFormat(0, 6, "two", "", 1, "", false, 0, "")
	doc.AddPage()
	doc.Bookmark("Chapter 2", 0, 0)
	doc.CellFormat(0, 6, "three", "", 1, "", false, 0, "")
	if err := doc.OutputFileAndClose(pdfPath); err != nil {
		t.Fatalf("write pdf: %v", err)
	}

	pageToHref := map[int]string{1: "page_001.html", 2: "page_002.html", 3: "page_003.html"}
	toc := buildPDFTOC(pdfPath, pageToHref)
	if len(toc) != 2 {
		t.Fatalf("expected 2 top-level bookmarks, got %d: %s", len(toc), tocDump(toc))
	}
	if toc[0].Title != "Chapter 1" || toc[0].Href != "page_001.html" {
		t.Errorf("entry 0 = {%q %q}, want {Chapter 1 page_001.html}", toc[0].Title, toc[0].Href)
	}
	if len(toc[0].Children) != 1 {
		t.Fatalf("expected 1 child under Chapter 1, got %d: %s", len(toc[0].Children), tocDump(toc))
	}
	if c := toc[0].Children[0]; c.Title != "Section 1.1" || c.Href != "page_002.html" {
		t.Errorf("child = {%q %q}, want {Section 1.1 page_002.html}", c.Title, c.Href)
	}
	if toc[1].Title != "Chapter 2" || toc[1].Href != "page_003.html" {
		t.Errorf("entry 1 = {%q %q}, want {Chapter 2 page_003.html}", toc[1].Title, toc[1].Href)
	}
}

func TestBuildPDFTOC_NoBookmarks(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "plain.pdf")
	createTestPDF(t, pdfPath, []string{"page one", "page two"})

	if toc := buildPDFTOC(pdfPath, map[int]string{1: "page_001.html", 2: "page_002.html"}); toc != nil {
		t.Errorf("expected nil TOC for a PDF without bookmarks, got %s", tocDump(toc))
	}
}

func tocDump(entries []epub.TOCEntry) string {
	var s string
	var walk func([]epub.TOCEntry, int)
	walk = func(es []epub.TOCEntry, d int) {
		for _, e := range es {
			for i := 0; i < d; i++ {
				s += "  "
			}
			s += e.Title + "|" + e.Href + "\n"
			walk(e.Children, d+1)
		}
	}
	walk(entries, 0)
	return s
}
