package htmlgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

func TestSinglePageAuthoredContentsKeepsHierarchyAndResolvesCollisions(t *testing.T) {
	dir := t.TempDir()
	book := &epub.Book{
		Title: "Book", Language: "en",
		Manifest: []epub.ManifestItem{{ID: "a", Href: "a.html"}, {ID: "b", Href: "b.html"}},
		Spine:    []epub.SpineItem{{IDRef: "a"}, {IDRef: "b"}},
		TOC: []epub.TOCEntry{{Title: "Part", Href: "a.html", Children: []epub.TOCEntry{
			{Title: "First", Href: "a.html#same"},
			{Title: "Second", Href: "b.html#same"},
		}}},
	}
	for name, content := range map[string]string{
		"a.html": `<html lang="en"><body><h1 id="same">First</h1><a href="b.html#same">Next</a></body></html>`,
		"b.html": `<html lang="en"><body><h1 id="same">Second</h1></body></html>`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GenerateSinglePage(book, dir, "book.epub"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "index.html"))
	for _, want := range []string{
		`id="dht-contents-button"`, `aria-controls="dht-contents"`, `<li><details open><summary><a href="#dht-ch-1">Part</a>`,
		`href="#same">First</a>`, `href="#c2-same">Second</a>`, `href="#c2-same">Next</a>`,
		`id="dht-ch-1"`, `id="c2-same"`, `if (!location.hash && saved`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestSinglePageHeadingFallbackGetsUniqueAnchors(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 2)
	for _, name := range []string{"ch_001.html", "ch_002.html"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(`<html lang="en"><body><h1>Repeated</h1><h2>Part</h2><p>Text.</p></body></html>`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GenerateSinglePage(book, dir, "book.txt"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "index.html"))
	for _, want := range []string{`href="#repeated"`, `href="#repeated-2"`, `id="repeated"`, `id="repeated-2"`, `href="#part"`, `href="#part-2"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(got, `<main`) != 1 {
		t.Error("contents created another main landmark")
	}
}

func TestSinglePageUntitledImagesHaveNoContents(t *testing.T) {
	dir := t.TempDir()
	book := writePagedBook(t, dir, 2)
	if _, err := GenerateSinglePage(book, dir, "comic.cbz"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "index.html"))
	if strings.Contains(got, `id="dht-contents-button"`) || !strings.Contains(got, `id="dht-page-sel"`) {
		t.Error("untitled image pages should keep only the page selector")
	}
}

func TestSinglePagePDFBookmarksNavigateImagePages(t *testing.T) {
	dir := t.TempDir()
	book := writePagedBook(t, dir, 2)
	book.TOC = []epub.TOCEntry{{Title: "Introduction", Href: "page_002.html"}}
	if _, err := GenerateSinglePage(book, dir, "scanned.pdf"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "index.html"))
	for _, want := range []string{`id="dht-page-sel"`, `id="dht-contents-button"`, `href="#dht-ch-2">Introduction</a>`, `id="dht-ch-2"`} {
		if !strings.Contains(got, want) {
			t.Errorf("PDF bookmark output missing %q", want)
		}
	}
}

func TestSinglePageContentsHonorsDepth(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 1)
	book.TOC = []epub.TOCEntry{{Title: "Chapter", Href: "ch_001.html", Children: []epub.TOCEntry{{Title: "Section", Href: "ch_001.html#section"}}}}
	if _, err := GenerateSinglePageWithDepth(book, dir, "book.epub", 1); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "index.html"))
	if !strings.Contains(got, `>Chapter</a>`) || strings.Contains(got, `>Section</a>`) {
		t.Error("depth 1 should show only top-level contents entries")
	}
}
