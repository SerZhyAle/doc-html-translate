package htmlsplit

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

// flatBook mirrors what the TXT and Markdown extractors hand the splitter: pages written
// straight into outputDir (no OPF base path), a stylesheet outside the spine, and a spine item
// that is not HTML.
func flatBook(t *testing.T, dir string, big string) *epub.Book {
	t.Helper()
	for name, content := range map[string]string{
		"page_001.html": big,
		"page_002.html": "<html><body><p>Short closing page.</p></body></html>",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &epub.Book{
		Manifest: []epub.ManifestItem{
			{ID: "page_001", Href: "page_001.html", MediaType: "text/html"},
			{ID: "css", Href: "style.css", MediaType: "text/css"},
			// Never written to disk: a non-HTML spine item must be passed through unread.
			{ID: "cover", Href: "cover.svg", MediaType: "image/svg+xml"},
			{ID: "page_002", Href: "page_002.html", MediaType: "text/html"},
		},
		Spine: []epub.SpineItem{{IDRef: "cover"}, {IDRef: "page_001"}, {IDRef: "page_002"}},
	}
}

func numberedParagraphs(n int) string {
	var b strings.Builder
	b.WriteString("<html><head><title>Big</title></head><body>")
	for i := 1; i <= n; i++ {
		b.WriteString("<p>Paragraph " + strconv.Itoa(i) + " " + strings.Repeat("filler text ", 8) + "</p>\n")
	}
	b.WriteString("</body></html>")
	return b.String()
}

var paragraphNumber = regexp.MustCompile(`Paragraph (\d+) `)

// The main path for a flat book: the oversized page is cut into parts that keep every
// paragraph exactly once and in order, the parts take the original's place in the spine, and
// everything that is not a splittable page keeps its position.
func TestSplitFlatBookKeepsEveryParagraphInOrder(t *testing.T) {
	dir := t.TempDir()
	const paragraphs, limit = 60, 1000
	book := flatBook(t, dir, numberedParagraphs(paragraphs))

	added, err := SplitIfNeeded(book, dir, limit)
	if err != nil {
		t.Fatal(err)
	}
	if added < 2 {
		t.Fatalf("added = %d, want the big page split into several parts", added)
	}

	hrefs := spineHrefs(book)
	if len(hrefs) != 3+added {
		t.Fatalf("spine = %v, want %d entries", hrefs, 3+added)
	}
	if hrefs[0] != "cover.svg" || hrefs[1] != "page_001.html" || hrefs[len(hrefs)-1] != "page_002.html" {
		t.Fatalf("spine order changed: %v", hrefs)
	}
	for i, h := range hrefs[2 : len(hrefs)-1] {
		if want := "page_001_s" + strconv.Itoa(i+2) + ".html"; h != want {
			t.Errorf("part %d href = %q, want %q", i+2, h, want)
		}
	}
	if book.Manifest[0].ID != "css" {
		t.Errorf("stylesheet moved from the front of the manifest: %+v", book.Manifest)
	}

	next := 1
	for _, h := range hrefs[1 : len(hrefs)-1] {
		data, err := os.ReadFile(filepath.Join(dir, h))
		if err != nil {
			t.Fatalf("part %s: %v", h, err)
		}
		for _, m := range paragraphNumber.FindAllStringSubmatch(string(data), -1) {
			if got, _ := strconv.Atoi(m[1]); got != next {
				t.Fatalf("%s: paragraph %d where %d was expected - text lost, duplicated or reordered", h, got, next)
			}
			next++
		}
	}
	if next-1 != paragraphs {
		t.Fatalf("parts hold %d paragraphs, want %d", next-1, paragraphs)
	}
}

// A spine page that cannot be read fails the split with its name, and the book is left as it
// was: the pipeline then removes the output, so a half-rewritten spine must never escape.
func TestSplitMissingPageFailsAndLeavesBookUnchanged(t *testing.T) {
	dir := t.TempDir()
	book := flatBook(t, dir, numberedParagraphs(60))
	book.Manifest = append(book.Manifest, epub.ManifestItem{ID: "gone", Href: "gone.html", MediaType: "text/html"})
	book.Spine = append(book.Spine, epub.SpineItem{IDRef: "gone"})
	manifest := append([]epub.ManifestItem(nil), book.Manifest...)
	spine := append([]epub.SpineItem(nil), book.Spine...)

	_, err := SplitIfNeeded(book, dir, 1000)
	if err == nil {
		t.Fatal("a missing spine page was split without an error")
	}
	if !strings.Contains(err.Error(), "gone.html") {
		t.Errorf("error does not name the page: %v", err)
	}
	if !reflect.DeepEqual(book.Manifest, manifest) || !reflect.DeepEqual(book.Spine, spine) {
		t.Errorf("a failed split rewrote the book:\nmanifest %+v\nspine %+v", book.Manifest, book.Spine)
	}
}
