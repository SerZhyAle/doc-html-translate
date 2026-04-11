package htmlconv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/htmlconv"
)

func TestExtract_BasicHTML(t *testing.T) {
	dir := t.TempDir()
	content := `<!DOCTYPE html>
<html>
<head><title>My Page</title></head>
<body>
<h1>Hello World</h1>
<p>Some content here.</p>
</body>
</html>`
	htmlPath := filepath.Join(dir, "test.html")
	if err := os.WriteFile(htmlPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := htmlconv.Extract(htmlPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if book.Title != "My Page" {
		t.Errorf("expected title 'My Page', got %q", book.Title)
	}
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if !strings.Contains(body, "Hello World") {
		t.Error("expected 'Hello World' in output")
	}
	if !strings.Contains(body, "Some content here.") {
		t.Error("expected content in output")
	}
}

func TestExtract_HTMLNoTitle(t *testing.T) {
	dir := t.TempDir()
	content := "<p>Just a paragraph.</p>"
	htmlPath := filepath.Join(dir, "notitle.htm")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := htmlconv.Extract(htmlPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	// Falls back to filename
	if book.Title != "notitle" {
		t.Errorf("expected fallback title 'notitle', got %q", book.Title)
	}
}

func TestExtract_HTMLEmpty(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "empty.html")
	_ = os.WriteFile(htmlPath, []byte("   \n\n  "), 0o644)
	_, err := htmlconv.Extract(htmlPath, dir)
	if err == nil {
		t.Error("expected error for empty HTML, got nil")
	}
}

func TestExtract_HTMLFileNotFound(t *testing.T) {
	_, err := htmlconv.Extract("/nonexistent/file.html", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExtract_HTMExtension(t *testing.T) {
	dir := t.TempDir()
	content := `<html><head><title>HTM Test</title></head><body><p>HTM content</p></body></html>`
	htmlPath := filepath.Join(dir, "test.htm")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := htmlconv.Extract(htmlPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if book.Title != "HTM Test" {
		t.Errorf("expected title 'HTM Test', got %q", book.Title)
	}
}
