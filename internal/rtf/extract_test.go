package rtf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/rtf"
)

func TestExtract_BasicRTF(t *testing.T) {
	dir := t.TempDir()
	// Simple RTF with two paragraphs
	content := `{\rtf1\ansi
Hello first paragraph.\par
Second paragraph here.\par
}`
	rtfPath := filepath.Join(dir, "test.rtf")
	if err := os.WriteFile(rtfPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := rtf.Extract(rtfPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if book.Title != "test" {
		t.Errorf("expected title 'test', got %q", book.Title)
	}
	if len(book.Spine) < 1 {
		t.Fatal("expected at least 1 page")
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if !strings.Contains(body, "Hello first paragraph") {
		t.Error("expected first paragraph in output")
	}
	if !strings.Contains(body, "Second paragraph") {
		t.Error("expected second paragraph in output")
	}
}

func TestExtract_RTFHexEscape(t *testing.T) {
	dir := t.TempDir()
	// \'cf\'f0\'e8\'e2\'e5\'f2 — "Привет" in CP1251
	content := `{\rtf1\ansi\ansicpg1251
\'cf\'f0\'e8\'e2\'e5\'f2\par
}`
	rtfPath := filepath.Join(dir, "cyrillic.rtf")
	_ = os.WriteFile(rtfPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := rtf.Extract(rtfPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) < 1 {
		t.Fatal("expected at least 1 page")
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if !strings.Contains(body, "Привет") {
		t.Errorf("expected decoded Cyrillic 'Привет', got: %s", body)
	}
}

func TestExtract_RTFEmpty(t *testing.T) {
	dir := t.TempDir()
	rtfPath := filepath.Join(dir, "empty.rtf")
	_ = os.WriteFile(rtfPath, []byte(""), 0o644)
	_, err := rtf.Extract(rtfPath, dir)
	if err == nil {
		t.Error("expected error for empty RTF, got nil")
	}
}

func TestExtract_RTFSpecialChars(t *testing.T) {
	dir := t.TempDir()
	content := `{\rtf1\ansi
Escaped \{ brace \} and \\ backslash.\par
}`
	rtfPath := filepath.Join(dir, "special.rtf")
	_ = os.WriteFile(rtfPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := rtf.Extract(rtfPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) < 1 {
		t.Fatal("expected at least 1 page")
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if !strings.Contains(body, "brace") {
		t.Error("expected 'brace' in output")
	}
}

func TestExtract_RTFFileNotFound(t *testing.T) {
	_, err := rtf.Extract("/nonexistent/file.rtf", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}
