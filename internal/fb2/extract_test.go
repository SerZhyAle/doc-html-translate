package fb2_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/fb2"
)

const simpleFB2 = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Test Book</book-title>
      <author><first-name>John</first-name><last-name>Doe</last-name></author>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Chapter One</p></title>
      <p>First paragraph of the book.</p>
      <p>Second paragraph of the book.</p>
    </section>
    <section>
      <title><p>Chapter Two</p></title>
      <p>Third paragraph here.</p>
    </section>
  </body>
</FictionBook>`

func TestExtract_BasicFB2(t *testing.T) {
	dir := t.TempDir()
	fb2Path := filepath.Join(dir, "test.fb2")
	if err := os.WriteFile(fb2Path, []byte(simpleFB2), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := fb2.Extract(fb2Path, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if book.Title != "Test Book" {
		t.Errorf("expected title 'Test Book', got %q", book.Title)
	}
	// 4 paragraphs (2 titles + 3 content) < 30 → 1 page
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if !strings.Contains(string(data), "First paragraph of the book.") {
		t.Error("expected content in page_001.html")
	}
}

func TestExtract_FB2Pagination(t *testing.T) {
	dir := t.TempDir()
	var paragraphs string
	for i := 1; i <= 35; i++ {
		paragraphs += fmt.Sprintf("      <p>Paragraph number %d</p>\n", i)
	}
	fb2Content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title>Big Book</book-title></title-info></description>
  <body>
    <section>
%s
    </section>
  </body>
</FictionBook>`, paragraphs)

	fb2Path := filepath.Join(dir, "big.fb2")
	_ = os.WriteFile(fb2Path, []byte(fb2Content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := fb2.Extract(fb2Path, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) != 2 {
		t.Errorf("expected 2 pages for 35 paragraphs, got %d", len(book.Spine))
	}
}

func TestExtract_FB2Empty(t *testing.T) {
	dir := t.TempDir()
	content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title></book-title></title-info></description>
  <body></body>
</FictionBook>`
	fb2Path := filepath.Join(dir, "empty.fb2")
	_ = os.WriteFile(fb2Path, []byte(content), 0o644)
	_, err := fb2.Extract(fb2Path, dir)
	if err == nil {
		t.Error("expected error for empty FB2, got nil")
	}
}

func TestExtract_FB2NestedSections(t *testing.T) {
	dir := t.TempDir()
	content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title>Nested</book-title></title-info></description>
  <body>
    <section>
      <section>
        <p>Deep paragraph</p>
      </section>
    </section>
  </body>
</FictionBook>`
	fb2Path := filepath.Join(dir, "nested.fb2")
	_ = os.WriteFile(fb2Path, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := fb2.Extract(fb2Path, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) < 1 {
		t.Fatal("expected at least 1 page")
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if !strings.Contains(string(data), "Deep paragraph") {
		t.Error("expected nested paragraph in output")
	}
}

func TestExtract_FB2FileNotFound(t *testing.T) {
	_, err := fb2.Extract("/nonexistent/file.fb2", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}
