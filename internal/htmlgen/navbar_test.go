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

// The navbar's aspect guard writes an inline width on every image it watches, and an inline width
// beats the overlay's own .ocr-fig>img{width:100%}. When that happened the picture fell back to
// its natural width while the plate container kept the column's, so every plate sat off the text
// by the ratio between them - measured on a 640 px scene in a 1216 px column, a plate moved from
// x=48 to x=91 and grew from 405 px wide to 770. The guard must therefore leave overlaid images
// alone, at both entry points: the per-image call and the MutationObserver it installs.
func TestImageAspectGuardSkipsOCROverlay(t *testing.T) {
	if !strings.Contains(navBarScript, `img.closest(".ocr-fig")`) {
		t.Error("the aspect guard has no .ocr-fig exemption - overlay plates will drift off their text")
	}
	guardCalls := strings.Count(navBarScript, "hasOCROverlay(img)")
	if guardCalls < 3 {
		t.Errorf("hasOCROverlay is consulted %d time(s); expected the definition plus both entry points "+
			"(preserveImageProportion and installImageAspectGuards)", guardCalls)
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

	if err := InjectNavBars(book, tmpDir, "Test Book.epub"); err != nil {
		t.Fatal(err)
	}

	// The source file name must appear on the left of the bar.
	if dataF, _ := os.ReadFile(filepath.Join(oebpsDir, "ch01.xhtml")); !strings.Contains(string(dataF), "Test Book.epub") {
		t.Error("ch01: expected source file name in navbar")
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
	// Reading themes [12] + reading-position controller [14].
	if !strings.Contains(s1, "dht-theme-sel") {
		t.Error("ch01: expected theme dropdown")
	}
	if !strings.Contains(s1, "dht-font-inc") {
		t.Error("ch01: expected text-size controls")
	}
	if !strings.Contains(s1, "dht-progress") {
		t.Error("ch01: expected reading-progress bar")
	}
	if !strings.Contains(s1, "dht-reader") {
		t.Error("ch01: expected reader controller script")
	}
	if !strings.Contains(s1, "dht-next") {
		t.Error("ch01: expected next link to carry dht-next class for auto-nav")
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

// TestReaderChromeAccessibilityFloor pins the injected chrome to the floor the 2026-09-29
// accessibility pass (ticket 57) established; docs/PARITY.md "Reader chrome accessibility
// floor" holds the extension edition to the same list. A control under 24px, a chrome without
// a theme-accent focus ring, or white ink on an accent button regresses here first.
func TestReaderChromeAccessibilityFloor(t *testing.T) {
	css := navBarCSS + readerCSS + searchCSS
	for _, want := range []string{
		"flex-wrap: wrap",            // the bar reflows at 200% zoom instead of clipping
		".dht-navbar :focus-visible", // the chrome carries a theme-accent focus ring
		"outline: 2px solid var(--dht-accent)",
		"box-sizing:border-box; min-height:28px", // buttons and selects, fine-pointer floor included
		"min-width:28px",                         // glyph target width, independent of its drawing
		"@media (pointer: coarse)",               // touch raises the target floor
		"min-height:44px; min-width:44px",
		`.dht-navbar[dir="rtl"] .nav-turn .dht-glyph { transform:scaleX(-1); }`,
		"margin-inline-start: auto", // RTL mirrors through logical properties
		"padding-inline-end: 12px",
		"color:var(--dht-bg)",       // accent buttons take theme ink, never white
		"#dht-search-close{padding", // the close word clears the floor too
	} {
		if !strings.Contains(css, want) {
			t.Errorf("injected reader chrome lost %q - the accessibility floor regressed", want)
		}
	}
}

// TestReaderComfortControls pins the reading-comfort additions of ticket 59 to the chrome of
// both output modes: the chapter navbar and the single-page header carry the same controls,
// the progress bar is readable as a number as well as a bar, and the keyboard handler owns
// page turning with the focus guard the ticket's boundary demands.
func TestReaderComfortControls(t *testing.T) {
	navHTML := buildNavBarHTML(NavInfo{PrevHref: "ch01.xhtml", NextHref: "ch03.xhtml", IndexHref: "../index.html", Current: 2, Total: 5})
	singleHTML := buildSinglePageHeader("book.epub", "Test", 3, true)
	for name, html := range map[string]string{"navbar": navHTML, "single page": singleHTML} {
		for _, want := range []string{
			`id="dht-size-reset"`,                     // text-size reset
			`id="dht-night-toggle"`,                   // day/night quick toggle
			`aria-pressed="false" title="Night mode"`, // pressed state carried in markup
			`id="dht-leading-sel"`,                    // line spacing select
			`id="dht-width-sel"`,                      // column width select
			`value="36em"`, `value="46em"`, `value="64em"`, `value="none"`,
			`id="dht-fitw-toggle"`,  // fit width, hidden until image pages
			`role="progressbar"`,    // progress readable as a number
			`id="dht-progress-tip"`, // the hover/focus readout
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s header lost %q", name, want)
			}
		}
	}

	for _, want := range []string{
		// Keyboard page turning: the arrows and the space bar at the edges, next to the
		// PageUp/PageDown behavior, with the focused-control guard.
		`e.key === "ArrowDown"`, `e.key === "ArrowRight"`,
		`e.key === "ArrowUp"`, `e.key === "ArrowLeft"`,
		`e.key === " "`, `e.shiftKey`,
		`t !== document.body`, // never fight a focused control or an in-book link
		// Fit width: per-session like the zoom, applied as a class + zoom-compensated width.
		`"dht_fitw"`, `classList.toggle("dht-fitw"`, `--dht-fitw`,
	} {
		if !strings.Contains(navBarScript, want) {
			t.Errorf("zoom/edge-nav script lost %q", want)
		}
	}
	for _, want := range []string{
		`"dht_leading"`, `"dht_width"`, // appearance scope: localStorage, like the theme
		`dht-night-toggle`, `lastFamilyTheme`, // the toggle remembers each family's last theme
		`applySize(DEFSZ, true)`, // the reset returns the shipped size, persisted
		`aria-valuenow`,          // the number form of the bar
	} {
		if !strings.Contains(readerScript("k", "c01.xhtml", 1, 3), want) {
			t.Errorf("reader script lost %q", want)
		}
	}

	// The comfort CSS gates on the reader's own choices - nothing applied before one is made.
	for _, want := range []string{
		"html[data-dht-leading] body", "html[data-dht-width] body",
		"html.dht-fitw .pdf-page-scan", "#dht-progress:hover .dht-progress-tip",
	} {
		if !strings.Contains(readerCSS, want) {
			t.Errorf("reader CSS lost %q", want)
		}
	}
}
