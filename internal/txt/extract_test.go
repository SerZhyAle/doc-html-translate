package txt_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/txt"
)

func TestExtract_Basic(t *testing.T) {
	dir := t.TempDir()
	content := "First paragraph line one\nFirst paragraph line two\n\nSecond paragraph\n\nThird paragraph"
	txtPath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(txtPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if book.Title != "test" {
		t.Errorf("expected title 'test', got %q", book.Title)
	}
	// 3 paragraphs < paragraphsPerPage → 1 page
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}

	data, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatalf("page_001.html not found: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "First paragraph line one First paragraph line two") {
		t.Errorf("expected merged paragraph in output")
	}
	if !strings.Contains(body, "Third paragraph") {
		t.Errorf("expected third paragraph in output")
	}
}

func TestExtract_Empty(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(txtPath, []byte("   \n\n   \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := txt.Extract(txtPath, dir)
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

func TestExtract_Pagination(t *testing.T) {
	dir := t.TempDir()
	// 35 paragraphs separated by blank lines → 2 pages (>30)
	var lines []string
	for i := 1; i <= 35; i++ {
		lines = append(lines, fmt.Sprintf("Paragraph number %d with some text here.", i))
		lines = append(lines, "")
	}
	txtPath := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(txtPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) != 2 {
		t.Errorf("expected 2 pages, got %d", len(book.Spine))
	}
	if _, err := os.Stat(filepath.Join(outDir, "page_002.html")); err != nil {
		t.Error("page_002.html not created")
	}
}

func TestExtract_FileNotFound(t *testing.T) {
	_, err := txt.Extract("/nonexistent/path/file.txt", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExtract_HtmlEscaping(t *testing.T) {
	dir := t.TempDir()
	content := "Hello <World> & \"quotes\""
	txtPath := filepath.Join(dir, "escape.txt")
	if err := os.WriteFile(txtPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	_, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if strings.Contains(body, "<World>") {
		t.Error("HTML not escaped: raw <World> found in output")
	}
	if !strings.Contains(body, "&lt;World&gt;") {
		t.Error("expected &lt;World&gt; in HTML-escaped output")
	}
}

func TestExtract_LinuxNoBlankLines(t *testing.T) {
	// File with only single \n between lines and no blank lines at all.
	// Each line must become its own paragraph.
	dir := t.TempDir()
	content := "Line one\nLine two\nLine three\nLine four"
	txtPath := filepath.Join(dir, "linux.txt")
	if err := os.WriteFile(txtPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	// 4 lines → 4 paragraphs → 1 page
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	// Count <p> tags — must be 4 separate paragraphs
	count := strings.Count(body, "<p>")
	if count != 4 {
		t.Errorf("expected 4 <p> elements for 4 lines, got %d\n%s", count, body)
	}
	if strings.Contains(body, "Line one Line two") {
		t.Error("lines must not be merged into one paragraph")
	}
}

func TestExtract_CRLFLineEndings(t *testing.T) {
	// Windows \r\n with blank lines — must behave like the baseline test.
	dir := t.TempDir()
	content := "First paragraph\r\n\r\nSecond paragraph\r\n\r\nThird paragraph\r\n"
	txtPath := filepath.Join(dir, "windows.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 3 {
		t.Errorf("expected 3 paragraphs from CRLF file, got %d", count)
	}
}

func TestExtract_OldMacCRLineEndings(t *testing.T) {
	// Old Mac \r only — no blank lines, each \r-line becomes its own paragraph.
	dir := t.TempDir()
	content := "Alpha\rBeta\rGamma"
	txtPath := filepath.Join(dir, "mac.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	_ = book
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 3 {
		t.Errorf("expected 3 paragraphs from old-Mac CR file, got %d", count)
	}
}

func TestExtract_UnicodeParagraphSeparators(t *testing.T) {
	dir := t.TempDir()
	content := "First paragraph\u2028\u2028Second paragraph\u2029\u2029Third paragraph"
	txtPath := filepath.Join(dir, "unicode-separators.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	_, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 3 {
		t.Errorf("expected 3 paragraphs from Unicode separators, got %d", count)
	}
	if !strings.Contains(string(data), "First paragraph") || !strings.Contains(string(data), "Third paragraph") {
		t.Errorf("expected Unicode-separated paragraphs to survive conversion")
	}
}

func TestExtract_ControlCharacterLineSeparators(t *testing.T) {
	dir := t.TempDir()
	content := "Alpha\u0085Beta\vGamma\fDelta"
	txtPath := filepath.Join(dir, "control-separators.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	_, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 4 {
		t.Errorf("expected 4 paragraphs from control-character separators, got %d", count)
	}
}
