package md_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/md"
)

func TestExtract_BasicMarkdown(t *testing.T) {
	dir := t.TempDir()
	content := "# Chapter One\n\nHello world paragraph.\n\n## Section Two\n\nSecond section content."
	mdPath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := md.Extract(mdPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if book.Title != "test" {
		t.Errorf("expected title 'test', got %q", book.Title)
	}

	// Two headings → 2 sections (each is one page)
	if len(book.Spine) != 2 {
		t.Errorf("expected 2 pages, got %d", len(book.Spine))
	}

	data, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatalf("page_001.html not found: %v", err)
	}
	if !strings.Contains(string(data), "Chapter One") {
		t.Error("expected 'Chapter One' in page_001")
	}
}

func TestExtract_NoHeadings(t *testing.T) {
	dir := t.TempDir()
	content := "Just plain text without headings.\n\nAnother paragraph."
	mdPath := filepath.Join(dir, "plain.md")
	_ = os.WriteFile(mdPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := md.Extract(mdPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page for headingless doc, got %d", len(book.Spine))
	}
}

func TestExtract_EmptyMarkdown(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "empty.md")
	_ = os.WriteFile(mdPath, []byte("   \n\n  "), 0o644)
	_, err := md.Extract(mdPath, dir)
	if err == nil {
		t.Error("expected error for empty markdown, got nil")
	}
}

func TestExtract_CodeBlock(t *testing.T) {
	dir := t.TempDir()
	content := "# Code Example\n\n```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n"
	mdPath := filepath.Join(dir, "code.md")
	_ = os.WriteFile(mdPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := md.Extract(mdPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) < 1 {
		t.Fatal("expected at least 1 page")
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if !strings.Contains(string(data), "<code") || !strings.Contains(string(data), "<pre") {
		t.Error("expected <pre><code> block in rendered output")
	}
}

func TestExtract_FileNotFound(t *testing.T) {
	_, err := md.Extract("/nonexistent/file.md", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// A local image the Markdown references is copied next to the pages and the reference
// rewritten; one outside the Markdown's folder is not copied.
func TestExtract_MarkdownLocalImagesCopied(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "out")
	for _, d := range []string{filepath.Join(srcDir, "img"), outDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{filepath.Join(srcDir, "img", "pic.png"), filepath.Join(root, "secret.png")} {
		if err := os.WriteFile(p, []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mdPath := filepath.Join(srcDir, "doc.md")
	content := "# Title\n\n![x](img/pic.png)\n\n![y](../secret.png)\n\n![z](https://example.com/r.png)\n"
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := md.Extract(mdPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	page, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(page)
	if !strings.Contains(body, `src="pic.png"`) {
		t.Errorf("image ref not rewritten: %s", body)
	}
	if _, err := os.Stat(filepath.Join(outDir, "pic.png")); err != nil {
		t.Error("pic.png not copied")
	}
	if _, err := os.Stat(filepath.Join(outDir, "secret.png")); err == nil || strings.Contains(body, "secret.png") {
		t.Error("an image outside the Markdown's folder was copied or left referenced")
	}
	if !strings.Contains(body, "https://example.com/r.png") {
		t.Error("remote image should be untouched")
	}
}

// Markdown carries no language, so the page must not claim one.
func TestExtract_MarkdownNoInventedLang(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(mdPath, []byte("Текст без заголовка.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := md.Extract(mdPath, dir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	page, _ := os.ReadFile(filepath.Join(dir, "page_001.html"))
	if strings.Contains(string(page), "lang=") {
		t.Errorf("page declares a language it does not know: %s", page)
	}
}
