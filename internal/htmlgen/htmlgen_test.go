package htmlgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

func TestGenerateIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create OEBPS subdir to mimic real structure
	oebpsDir := filepath.Join(tmpDir, "OEBPS")
	if err := os.MkdirAll(oebpsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book := &epub.Book{
		Title:    "My Test Book",
		BasePath: "OEBPS",
		Manifest: []epub.ManifestItem{
			{ID: "ch1", Href: "chapter1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "ch2", Href: "chapter2.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "css1", Href: "style.css", MediaType: "text/css"},
		},
		Spine: []epub.SpineItem{
			{IDRef: "ch1"},
			{IDRef: "ch2"},
		},
	}

	indexPath, err := GenerateIndex(book, tmpDir)
	if err != nil {
		t.Fatalf("GenerateIndex failed: %v", err)
	}

	if filepath.Base(indexPath) != "index.html" {
		t.Errorf("expected index.html, got %s", filepath.Base(indexPath))
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	content := string(data)

	// Check title
	if !strings.Contains(content, "My Test Book") {
		t.Error("index.html should contain book title")
	}

	// Check CSS link
	if !strings.Contains(content, "OEBPS/style.css") {
		t.Error("index.html should link to CSS")
	}

	// Check chapter links
	if !strings.Contains(content, "OEBPS/chapter1.xhtml") {
		t.Error("index.html should link to chapter1")
	}
	if !strings.Contains(content, "OEBPS/chapter2.xhtml") {
		t.Error("index.html should link to chapter2")
	}

	// Check chapters count
	if !strings.Contains(content, "Chapters: 2") {
		t.Error("index.html should show chapter count")
	}
}

func TestChapterLabel(t *testing.T) {
	tests := []struct {
		href  string
		index int
		want  string
	}{
		{"chapter1.xhtml", 1, "1. chapter1"},
		{"ch_02_intro.xhtml", 2, "2. ch 02 intro"},
		{"", 3, "Chapter 3"},
	}

	for _, tt := range tests {
		got := chapterLabel(tt.href, tt.index)
		if got != tt.want {
			t.Errorf("chapterLabel(%q, %d) = %q, want %q", tt.href, tt.index, got, tt.want)
		}
	}
}

func TestExtractSnippet_Basic(t *testing.T) {
	dir := t.TempDir()
	page := `<!DOCTYPE html><html><head><title>T</title></head>
<body>
  <div class="page-header">My Book — 1 / 3</div>
  <p>The quick brown fox jumps over the lazy dog. Second sentence follows.</p>
  <p>Another paragraph here.</p>
</body></html>`
	path := filepath.Join(dir, "page_001.html")
	if err := os.WriteFile(path, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	got := extractSnippet(path)
	if !strings.Contains(got, "quick brown fox") {
		t.Errorf("expected first sentence in snippet, got: %q", got)
	}
	// With 2-sentence limit, second sentence should also be included
	if !strings.Contains(got, "Second sentence follows") {
		t.Errorf("expected second sentence in snippet, got: %q", got)
	}
	// Should NOT include the structural page-header text
	if strings.Contains(got, "My Book") {
		t.Errorf("snippet should skip page-header div, got: %q", got)
	}
}

func TestExtractSnippet_LongNoSentence(t *testing.T) {
	dir := t.TempDir()
	// Long text without sentence terminators → truncated with ellipsis
	long := strings.Repeat("word ", 70) // 350 chars, no punctuation → must truncate with ellipsis
	page := "<html><body><p>" + long + "</p></body></html>"
	path := filepath.Join(dir, "page.html")
	_ = os.WriteFile(path, []byte(page), 0o644)
	got := extractSnippet(path)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis for long unbreakable text, got: %q", got)
	}
}

func TestExtractSnippet_MissingFile(t *testing.T) {
	got := extractSnippet("/nonexistent/path/page.html")
	if got != "" {
		t.Errorf("expected empty string for missing file, got: %q", got)
	}
}

func TestGenerateIndex_WithSnippets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create actual page files so extractSnippet can read them
	pages := []struct {
		file    string
		content string
	}{
		{"page_001.html", `<html><body><p>First page content here. This is the opening sentence.</p></body></html>`},
		{"page_002.html", `<html><body><p>Second page begins with this text.</p></body></html>`},
	}
	for _, p := range pages {
		if err := os.WriteFile(filepath.Join(tmpDir, p.file), []byte(p.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	book := &epub.Book{
		Title:    "Snippet Test Book",
		BasePath: "",
		Manifest: []epub.ManifestItem{
			{ID: "p1", Href: "page_001.html", MediaType: "text/html"},
			{ID: "p2", Href: "page_002.html", MediaType: "text/html"},
		},
		Spine: []epub.SpineItem{
			{IDRef: "p1"},
			{IDRef: "p2"},
		},
	}

	indexPath, err := GenerateIndex(book, tmpDir)
	if err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}
	data, _ := os.ReadFile(indexPath)
	content := string(data)
	if !strings.Contains(content, "toc-snippet") {
		t.Error("expected toc-snippet class in TOC with page files present")
	}
	if !strings.Contains(content, "First page content here") {
		t.Errorf("expected first-page snippet in TOC, got content without it")
	}
	if !strings.Contains(content, "Second page begins") {
		t.Errorf("expected second-page snippet in TOC")
	}
	// Filename labels should NOT appear as link text when snippet is available.
	// (They still appear in href="page_001.html" attribute, so check label span specifically)
	if strings.Contains(content, ">page_001<") || strings.Contains(content, ">page_002<") {
		t.Error("filename labels should be replaced by snippet text in TOC")
	}
}
