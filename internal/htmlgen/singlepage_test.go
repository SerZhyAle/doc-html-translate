package htmlgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

// writePagedBook lays out an image-page book on disk - the shape internal/comic and
// internal/img produce - and returns it ready for GenerateSinglePage.
func writePagedBook(t *testing.T, dir string, pages int) *epub.Book {
	t.Helper()
	book := &epub.Book{Title: "Test Comic"}
	for i := 1; i <= pages; i++ {
		href := fmt.Sprintf("page_%03d.html", i)
		body := fmt.Sprintf(
			"<!DOCTYPE html><html lang=\"en\"><body>\n"+
				"  <section class=\"dht-page\" id=\"page_%03d\" aria-label=\"p%d\">\n"+
				"    <img src=\"page_%03d.jpg\" alt=\"Test Comic - page %d of %d\">\n"+
				"  </section>\n</body></html>\n", i, i, i, i, pages)
		if err := os.WriteFile(filepath.Join(dir, href), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", href, err)
		}
		id := fmt.Sprintf("page_%03d", i)
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "text/html"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}
	return book
}

// writeTextBook lays out an ordinary multi-chapter text book.
func writeTextBook(t *testing.T, dir string, chapters int) *epub.Book {
	t.Helper()
	book := &epub.Book{Title: "Test Book"}
	for i := 1; i <= chapters; i++ {
		href := fmt.Sprintf("ch_%03d.html", i)
		body := fmt.Sprintf("<!DOCTYPE html><html lang=\"en\"><body><h1>Chapter %d</h1><p>Text.</p></body></html>\n", i)
		if err := os.WriteFile(filepath.Join(dir, href), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", href, err)
		}
		id := fmt.Sprintf("ch_%03d", i)
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "text/html"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}
	return book
}

// The merged document must have exactly one <main>, however many pages were folded into
// it. Nesting one <main> per page inside the merged document's own is invalid HTML and
// breaks reader mode and screen-reader landmark navigation.
func TestGenerateSinglePageHasOneMain(t *testing.T) {
	dir := t.TempDir()
	book := writePagedBook(t, dir, 5)
	if _, err := GenerateSinglePage(book, dir, "test.cbz"); err != nil {
		t.Fatalf("GenerateSinglePage: %v", err)
	}
	merged := readFile(t, filepath.Join(dir, "index.html"))
	if got := strings.Count(merged, "<main"); got != 1 {
		t.Errorf("merged document has %d <main> elements, want exactly 1", got)
	}
	if strings.Count(merged, `class="dht-page"`) != 5 {
		t.Error("the merged document lost its per-page sections")
	}
}

// Every page keeps its anchor through the merge, and the header offers a jump box. The
// merged comic is one long scroll; without these a reader cannot reach page 30 except by
// dragging the scrollbar, and cannot link to it at all.
func TestGenerateSinglePagePageAnchorsAndSelector(t *testing.T) {
	dir := t.TempDir()
	book := writePagedBook(t, dir, 4)
	if _, err := GenerateSinglePage(book, dir, "test.cbz"); err != nil {
		t.Fatalf("GenerateSinglePage: %v", err)
	}
	merged := readFile(t, filepath.Join(dir, "index.html"))
	for i := 1; i <= 4; i++ {
		if !strings.Contains(merged, fmt.Sprintf(`id="page_%03d"`, i)) {
			t.Errorf("merged document has no anchor for page %d", i)
		}
		if !strings.Contains(merged, fmt.Sprintf(`<option value="#page_%03d">`, i)) {
			t.Errorf("page selector has no entry for page %d", i)
		}
	}
	if !strings.Contains(merged, `id="dht-page-sel"`) {
		t.Error("an image-page book has no page selector in its header")
	}
}

// The separator has to tell the truth about what it separates: a page break between comic
// pages, the chapter rule between chapters.
func TestGenerateSinglePageSeparatorMatchesBookKind(t *testing.T) {
	pagedDir := t.TempDir()
	paged := writePagedBook(t, pagedDir, 3)
	if _, err := GenerateSinglePage(paged, pagedDir, "test.cbz"); err != nil {
		t.Fatalf("GenerateSinglePage (paged): %v", err)
	}
	pagedHTML := readFile(t, filepath.Join(pagedDir, "index.html"))
	if !strings.Contains(pagedHTML, `<hr class="dht-page-sep"`) {
		t.Error("an image-page book does not use the page separator")
	}
	if strings.Contains(pagedHTML, `<hr class="dht-chapter-sep"`) {
		t.Error("an image-page book marks its page breaks as chapter breaks")
	}

	textDir := t.TempDir()
	text := writeTextBook(t, textDir, 3)
	if _, err := GenerateSinglePage(text, textDir, "test.epub"); err != nil {
		t.Fatalf("GenerateSinglePage (text): %v", err)
	}
	textHTML := readFile(t, filepath.Join(textDir, "index.html"))
	if !strings.Contains(textHTML, `<hr class="dht-chapter-sep"`) {
		t.Error("a text book lost its chapter separator")
	}
	if strings.Contains(textHTML, `id="dht-page-sel"`) {
		t.Error("a text book should not get the image-page jump box")
	}
}

// The merge absorbs every spine page, so the originals must not be left behind: nothing
// links to them, they carry no navbar, no theme and no OCR plates, so anyone who reaches
// one by guessing a filename gets a worse page than the book they asked for.
func TestGenerateSinglePageRemovesAbsorbedPages(t *testing.T) {
	dir := t.TempDir()
	book := writePagedBook(t, dir, 3)
	if _, err := GenerateSinglePage(book, dir, "test.cbz"); err != nil {
		t.Fatalf("GenerateSinglePage: %v", err)
	}
	for i := 1; i <= 3; i++ {
		orphan := filepath.Join(dir, fmt.Sprintf("page_%03d.html", i))
		if _, err := os.Stat(orphan); err == nil {
			t.Errorf("page_%03d.html survived the merge as an orphan", i)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		t.Fatalf("the merged index.html is missing: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
