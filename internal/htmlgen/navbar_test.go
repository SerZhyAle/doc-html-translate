package htmlgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

func TestRelativePath(t *testing.T) {
	tests := []struct {
		from, target, want string
	}{
		{"OEBPS", "OEBPS/ch01.xhtml", "ch01.xhtml"},
		{"OEBPS", "OEBPS/ch02.xhtml", "ch02.xhtml"},
		{"OEBPS", "index.html", "../index.html"},
		{"OEBPS/sub", "OEBPS/ch01.xhtml", "../ch01.xhtml"},
		{".", "index.html", "index.html"},
		{"", "index.html", "index.html"},
	}

	for _, tt := range tests {
		got := relativePath(tt.from, tt.target)
		if got != tt.want {
			t.Errorf("relativePath(%q, %q) = %q, want %q", tt.from, tt.target, got, tt.want)
		}
	}
}

func TestBuildNavBarHTML(t *testing.T) {
	nav := NavInfo{
		PrevHref:  "ch01.xhtml",
		NextHref:  "ch03.xhtml",
		IndexHref: "../index.html",
		Title:     "Test Book",
		Current:   2,
		Total:     5,
	}

	html := buildNavBarHTML(nav)

	// Must contain prev/next links and TOC link
	if !strings.Contains(html, "ch01.xhtml") {
		t.Error("expected prev link to ch01.xhtml")
	}
	if !strings.Contains(html, "ch03.xhtml") {
		t.Error("expected next link to ch03.xhtml")
	}
	if !strings.Contains(html, "../index.html") {
		t.Error("expected TOC link to ../index.html")
	}
	if !strings.Contains(html, "2 / 5") {
		t.Error("expected page counter 2 / 5")
	}
	if !strings.Contains(html, "dht-zoom-sync") {
		t.Error("expected zoom sync script marker")
	}
}

func TestBuildNavBarHTML_FirstPage(t *testing.T) {
	nav := NavInfo{
		PrevHref:  "",
		NextHref:  "ch02.xhtml",
		IndexHref: "../index.html",
		Current:   1,
		Total:     3,
	}

	html := buildNavBarHTML(nav)

	// Prev should be disabled
	if !strings.Contains(html, `class="disabled"`) {
		t.Error("expected disabled class for first page prev link")
	}
	if !strings.Contains(html, "ch02.xhtml") {
		t.Error("expected next link")
	}
}

func TestInjectNavBars(t *testing.T) {
	tmpDir := t.TempDir()

	// Create OEBPS directory
	oebpsDir := filepath.Join(tmpDir, "OEBPS")
	if err := os.MkdirAll(oebpsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create 3 simple HTML files
	pages := []string{"ch01.xhtml", "ch02.xhtml", "ch03.xhtml"}
	for _, p := range pages {
		content := `<!DOCTYPE html>
<html><head><title>Test</title></head>
<body><p>Hello World</p></body></html>`
		if err := os.WriteFile(filepath.Join(oebpsDir, p), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	book := &epub.Book{
		Title:    "Test Book",
		BasePath: "OEBPS",
		Manifest: []epub.ManifestItem{
			{ID: "ch01", Href: "ch01.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "ch02", Href: "ch02.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "ch03", Href: "ch03.xhtml", MediaType: "application/xhtml+xml"},
		},
		Spine: []epub.SpineItem{
			{IDRef: "ch01"},
			{IDRef: "ch02"},
			{IDRef: "ch03"},
		},
	}

	if err := InjectNavBars(book, tmpDir); err != nil {
		t.Fatal(err)
	}

	// Check first page: no prev, has next
	data1, _ := os.ReadFile(filepath.Join(oebpsDir, "ch01.xhtml"))
	s1 := string(data1)
	if !strings.Contains(s1, "dht-navbar") {
		t.Error("ch01: expected navbar class")
	}
	if !strings.Contains(s1, "dht-nav") {
		t.Error("ch01: expected navbar CSS")
	}
	if !strings.Contains(s1, "dht-zoom-sync") {
		t.Error("ch01: expected zoom sync script")
	}
	if !strings.Contains(s1, "ch02.xhtml") {
		t.Error("ch01: expected next link to ch02")
	}
	// Prev should be disabled on first page
	if !strings.Contains(s1, `class="disabled"`) {
		t.Error("ch01: expected disabled prev link")
	}

	// Check middle page: has prev and next
	data2, _ := os.ReadFile(filepath.Join(oebpsDir, "ch02.xhtml"))
	s2 := string(data2)
	if !strings.Contains(s2, "ch01.xhtml") {
		t.Error("ch02: expected prev link to ch01")
	}
	if !strings.Contains(s2, "ch03.xhtml") {
		t.Error("ch02: expected next link to ch03")
	}
	if !strings.Contains(s2, "../index.html") {
		t.Error("ch02: expected TOC link to ../index.html")
	}

	// Check last page: has prev, no next
	data3, _ := os.ReadFile(filepath.Join(oebpsDir, "ch03.xhtml"))
	s3 := string(data3)
	if !strings.Contains(s3, "ch02.xhtml") {
		t.Error("ch03: expected prev link to ch02")
	}
	// Count disabled links - last page should have disabled "next"
	// The string "disabled" should appear for the next link
	if strings.Count(s3, `class="disabled"`) < 1 {
		t.Error("ch03: expected at least one disabled link (next)")
	}
}
