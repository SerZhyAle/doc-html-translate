package htmlgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

const (
	figureOpen   = `<img src="figures/gone.png" alt="A missing figure"`
	figureMarkup = figureOpen + `>`
)

// writeFigureBook lays out a one-chapter book whose only picture is not on disk.
func writeFigureBook(t *testing.T, dir string) *epub.Book {
	t.Helper()
	body := `<!DOCTYPE html><html lang="en"><head><title>T</title></head><body><h1>One</h1><p>` + figureMarkup + `</p></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "ch_001.html"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return &epub.Book{
		Title:    "Test Book",
		Manifest: []epub.ManifestItem{{ID: "ch_001", Href: "ch_001.html", MediaType: "text/html"}},
		Spine:    []epub.SpineItem{{IDRef: "ch_001"}},
	}
}

// ICON-EXTERNAL section 13 F: a picture of the book that cannot be shown draws content.image with its alt
// text. The handler must be in the head, ahead of the first <img> - the single page's reader script sits
// after the whole book - and must listen in the capture phase, since an error event does not bubble.
func assertMissingImageHandler(t *testing.T, page string) {
	t.Helper()
	head := page[:strings.Index(page, "</head>")]
	if strings.Count(page, `<script id="dht-missing-image">`) != 1 {
		t.Fatalf("want exactly one missing-image script, got %d", strings.Count(page, `<script id="dht-missing-image">`))
	}
	if !strings.Contains(head, `<script id="dht-missing-image">`) {
		t.Error("the missing-image handler is not in the head, so a picture can fail before it listens")
	}
	if strings.Index(page, `<script id="dht-missing-image">`) > strings.Index(page, "<img") {
		t.Error("the missing-image handler follows the first <img>")
	}
	if !strings.Contains(page, `.dht-missing-img {`) {
		t.Error("the stand-in has no styles on the page")
	}
	script := page[strings.Index(page, `<script id="dht-missing-image">`):]
	script = script[:strings.Index(script, "</script>")]
	for _, want := range []string{
		`document.addEventListener("error"`,
		"}, true);",
		`"` + Glyphs["content.image"] + `"`,
		`className = "dht-missing-img"`,
		`getAttribute("alt")`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("the handler lacks %q", want)
		}
	}
	if !strings.Contains(page, figureOpen) {
		t.Errorf("the picture's own markup changed: %s", page)
	}
}

func TestChapterPageCatchesAFailedPicture(t *testing.T) {
	dir := t.TempDir()
	book := writeFigureBook(t, dir)
	if err := InjectNavBars(book, dir, "t.epub"); err != nil {
		t.Fatal(err)
	}
	assertMissingImageHandler(t, readFile(t, filepath.Join(dir, "ch_001.html")))
}

func TestSinglePageCatchesAFailedPicture(t *testing.T) {
	dir := t.TempDir()
	book := writeFigureBook(t, dir)
	if _, err := GenerateSinglePage(book, dir, "t.epub"); err != nil {
		t.Fatal(err)
	}
	assertMissingImageHandler(t, readFile(t, filepath.Join(dir, "index.html")))
}

// The handler never asks for anything: it swaps the node and leaves the failed picture's src alone.
// A stand-in uses the reading theme's tokens, not a palette value of its own (internal/appearance
// is the one source of colour).
func TestMissingImageStandInMakesNoRequestAndHasNoPaletteValue(t *testing.T) {
	for _, banned := range []string{".src", "srcset", "fetch(", "XMLHttpRequest", "new Image", "sendBeacon"} {
		if strings.Contains(missingImageScript, banned) {
			t.Errorf("the missing-image script mentions %q: it must not request anything", banned)
		}
	}
	if m := regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgba?\(|hsla?\(`).FindString(missingImageCSS); m != "" {
		t.Errorf("the stand-in's styles declare a colour literal %q, not a --dht-* token", m)
	}
	if !strings.Contains(missingImageCSS, "var(--dht-fg)") || !strings.Contains(missingImageCSS, "var(--dht-border)") {
		t.Error("the stand-in does not take the reading theme's ink and border")
	}
}
