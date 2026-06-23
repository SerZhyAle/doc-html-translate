package htmlgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

func nestedTOCBook() *epub.Book {
	return &epub.Book{
		Title:    "Depth Book",
		BasePath: "",
		Manifest: []epub.ManifestItem{
			{ID: "p1", Href: "page_001.html", MediaType: "text/html"},
			{ID: "p2", Href: "page_002.html", MediaType: "text/html"},
		},
		Spine: []epub.SpineItem{{IDRef: "p1"}, {IDRef: "p2"}},
		TOC: []epub.TOCEntry{
			{Title: "Chapter 1", Href: "page_001.html", Children: []epub.TOCEntry{
				{Title: "Section 1.1", Href: "page_001.html#a", Children: []epub.TOCEntry{
					{Title: "Sub 1.1.1", Href: "page_001.html#b"},
				}},
			}},
			{Title: "Chapter 2", Href: "page_002.html"},
		},
	}
}

func TestGenerateIndex_MultiLevelDepth(t *testing.T) {
	tests := []struct {
		depth       int
		mustContain []string
		mustNotHave []string
		wantDetails bool
	}{
		{depth: 0, mustContain: []string{"Chapter 1", "Section 1.1", "Sub 1.1.1", `href="page_001.html#b"`}, wantDetails: true},
		{depth: 1, mustContain: []string{"Chapter 1", "Chapter 2"}, mustNotHave: []string{"Section 1.1", "Sub 1.1.1"}, wantDetails: false},
		{depth: 2, mustContain: []string{"Chapter 1", "Section 1.1"}, mustNotHave: []string{"Sub 1.1.1"}, wantDetails: true},
	}

	for _, tc := range tests {
		dir := t.TempDir()
		idxPath, err := GenerateIndexWithSnippetsDepth(nestedTOCBook(), dir, nil, tc.depth)
		if err != nil {
			t.Fatalf("depth %d: %v", tc.depth, err)
		}
		data, _ := os.ReadFile(idxPath)
		content := string(data)

		for _, s := range tc.mustContain {
			if !strings.Contains(content, s) {
				t.Errorf("depth %d: expected %q in index.html", tc.depth, s)
			}
		}
		for _, s := range tc.mustNotHave {
			if strings.Contains(content, s) {
				t.Errorf("depth %d: did not expect %q in index.html", tc.depth, s)
			}
		}
		if got := strings.Contains(content, "<details"); got != tc.wantDetails {
			t.Errorf("depth %d: <details> present=%v, want %v", tc.depth, got, tc.wantDetails)
		}
	}
}

func TestRenderTOC_BasePathPrefix(t *testing.T) {
	book := &epub.Book{
		Title:    "B",
		BasePath: "OEBPS",
		Manifest: []epub.ManifestItem{{ID: "c", Href: "ch.html", MediaType: "text/html"}},
		Spine:    []epub.SpineItem{{IDRef: "c"}},
		TOC:      []epub.TOCEntry{{Title: "Ch", Href: "ch.html#x"}},
	}
	dir := t.TempDir()
	idxPath, err := GenerateIndex(book, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(idxPath)
	if !strings.Contains(string(data), `href="OEBPS/ch.html#x"`) {
		t.Errorf("expected base-prefixed href OEBPS/ch.html#x, got:\n%s", data)
	}
}

func TestBuildFallbackTOC_HeadingScanAndAnchors(t *testing.T) {
	dir := t.TempDir()
	page := `<!DOCTYPE html><html><head><title>P</title></head><body>
  <div class="page-header">Page 1 / 1</div>
  <h1>Alpha</h1><p>text</p>
  <h2 id="keep">Beta</h2><p>text</p>
  <h2>Gamma</h2>
  <div class="dht-navbar"><h3>NAVSKIP</h3></div>
</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "page_001.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	book := &epub.Book{
		Title:    "Scan",
		BasePath: "",
		Manifest: []epub.ManifestItem{{ID: "p1", Href: "page_001.html", MediaType: "text/html"}},
		Spine:    []epub.SpineItem{{IDRef: "p1"}},
	}

	toc := BuildFallbackTOC(book, dir, nil)

	if len(toc) != 1 {
		t.Fatalf("expected 1 top-level page entry, got %d", len(toc))
	}
	page1 := toc[0]
	if len(page1.Children) != 1 {
		t.Fatalf("expected 1 heading child (Alpha) under page, got %d: %+v", len(page1.Children), page1.Children)
	}
	alpha := page1.Children[0]
	if alpha.Title != "Alpha" {
		t.Errorf("first heading = %q, want Alpha", alpha.Title)
	}
	if len(alpha.Children) != 2 {
		t.Fatalf("expected Beta+Gamma under Alpha, got %d", len(alpha.Children))
	}
	if beta := alpha.Children[0]; beta.Href != "page_001.html#keep" {
		t.Errorf("Beta href = %q, want page_001.html#keep (existing id preserved)", beta.Href)
	}
	if gamma := alpha.Children[1]; gamma.Href != "page_001.html#gamma" {
		t.Errorf("Gamma href = %q, want page_001.html#gamma (slug injected)", gamma.Href)
	}

	// The navbar heading must never appear in the TOC.
	if strings.Contains(tocString(toc), "NAVSKIP") {
		t.Error("structural (navbar) heading leaked into TOC")
	}

	// The injected id must have been written back to the file.
	data, _ := os.ReadFile(filepath.Join(dir, "page_001.html"))
	if !strings.Contains(string(data), `id="gamma"`) {
		t.Error("expected injected id=\"gamma\" persisted to page file")
	}

	// Idempotent: a second scan yields the same hrefs (no churn, no duplicate ids).
	toc2 := BuildFallbackTOC(book, dir, nil)
	if tocString(toc) != tocString(toc2) {
		t.Errorf("fallback TOC not idempotent:\n first: %s\nsecond: %s", tocString(toc), tocString(toc2))
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":  "hello-world",
		"  Trim--Me  ": "trim-me",
		"Заголовок":    "", // non-ascii -> empty (caller uses counter fallback)
		"A & B / C":    "a-b-c",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// tocString renders a TOC tree to a compact string for comparison/inspection.
func tocString(entries []epub.TOCEntry) string {
	var sb strings.Builder
	var walk func([]epub.TOCEntry, int)
	walk = func(es []epub.TOCEntry, depth int) {
		for _, e := range es {
			sb.WriteString(strings.Repeat(" ", depth))
			sb.WriteString(e.Title + "|" + e.Href + "\n")
			walk(e.Children, depth+1)
		}
	}
	walk(entries, 0)
	return sb.String()
}
