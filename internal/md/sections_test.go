package md_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/md"
)

func writeMarkdown(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Text before the first heading is a page of its own, only h1 and h2 start a page, and each
// page is listed in manifest and spine under the name it was written to.
func TestExtract_PreambleAndDeepHeadings(t *testing.T) {
	dir := t.TempDir()
	mdPath := writeMarkdown(t, dir, "guide.md",
		"Preface text.\n\n# Part A\n\nA body.\n\n### A detail\n\nDetail body.\n\n## Part B\n\nB body.\n")
	outDir := filepath.Join(dir, "out")
	if err := os.Mkdir(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := md.Extract(mdPath, outDir)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	want := []struct{ href, has, hasNot string }{
		{"page_001.html", "Preface text.", "Part A"},
		{"page_002.html", "A detail", "Part B"},
		{"page_003.html", "B body.", "Preface"},
	}
	if len(book.Spine) != len(want) || len(book.Manifest) != len(want) {
		t.Fatalf("spine %d, manifest %d; want %d pages", len(book.Spine), len(book.Manifest), len(want))
	}
	for i, w := range want {
		if book.Manifest[i].Href != w.href || book.Spine[i].IDRef != book.Manifest[i].ID {
			t.Errorf("page %d: manifest %+v, spine %+v", i+1, book.Manifest[i], book.Spine[i])
		}
		data, err := os.ReadFile(filepath.Join(outDir, w.href))
		if err != nil {
			t.Fatalf("%s: %v", w.href, err)
		}
		if !strings.Contains(string(data), w.has) || strings.Contains(string(data), w.hasNot) {
			t.Errorf("%s: want %q and not %q in:\n%s", w.href, w.has, w.hasNot, data)
		}
	}
}

// The title comes from the file name, which may hold characters that are markup in HTML. The
// name avoids < and >, which Windows does not allow in a file name.
func TestExtract_TitleIsEscaped(t *testing.T) {
	dir := t.TempDir()
	mdPath := writeMarkdown(t, dir, "Tom & Jerry's.md", "Text.\n")
	book, err := md.Extract(mdPath, dir)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if book.Title != "Tom & Jerry's" {
		t.Errorf("title = %q", book.Title)
	}
	page, err := os.ReadFile(filepath.Join(dir, "page_001.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "<title>Tom &amp; Jerry&#39;s") {
		t.Errorf("title not escaped:\n%s", page)
	}
}

// A directory handed in as the Markdown file is a read error, not an empty document.
func TestExtract_DirectoryInputFails(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.md")
	if err := os.Mkdir(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := md.Extract(notes, t.TempDir()); err == nil || !strings.Contains(err.Error(), "open markdown") {
		t.Errorf("err = %v, want an open markdown error", err)
	}
}

// A page that cannot be written fails the extraction instead of returning a book whose spine
// points at files that are not there.
func TestExtract_UnwritableOutputFails(t *testing.T) {
	dir := t.TempDir()
	mdPath := writeMarkdown(t, dir, "doc.md", "# Title\n\nBody.\n")
	book, err := md.Extract(mdPath, filepath.Join(dir, "missing", "out"))
	if err == nil || !strings.Contains(err.Error(), "write page 1") {
		t.Fatalf("err = %v, want a write page error", err)
	}
	if book != nil {
		t.Errorf("a failed extraction returned a book: %+v", book)
	}
}
