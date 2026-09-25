package htmlgen

import (
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
)

// anchorByClass returns the first <a ..> element whose class list holds class, up to its </a>.
func anchorByClass(t *testing.T, html, class string) string {
	t.Helper()
	m := regexp.MustCompile(`<a class="[^"]*\b` + regexp.QuoteMeta(class) + `\b[^"]*"[^>]*>.*?</a>`).FindString(html)
	if m == "" {
		t.Fatalf("no <a> with class %q in the bar", class)
	}
	return m
}

// The vocabulary was founded on this product's own defect: the page links read "Back" /
// "Forward" (nav.back and nav.forward's names) and drew ◀ / ▶ (no meaning, and media.play's
// shape). Paging is media.previous / media.next, in glyph and in name, in every language the
// chrome speaks (ICON-SET rules 1-3). This pins each pair so the breach cannot come back.
func TestPagingIsPreviousAndNextInEveryLanguage(t *testing.T) {
	defer i18n.SetLanguage(i18n.Language())

	// nav.back's and nav.forward's words in the languages that used them on these links.
	backWords := []string{"Back", "Назад", "Zurück", "Indietro", "Atrás", "Retour", "Voltar", "رجوع", "پیچھے"}
	forwardWords := []string{"Forward", "Вперёд", "Вперед", "Weiter", "Avanti", "Adelante", "Avançar", "آگے"}

	nav := NavInfo{PrevHref: "ch01.xhtml", NextHref: "ch03.xhtml", IndexHref: "index.html", Current: 2, Total: 3}
	for _, lang := range i18n.Codes {
		t.Run(lang, func(t *testing.T) {
			i18n.SetLanguage(lang)
			bar := buildNavBarHTML(nav)
			prev, next := anchorByClass(t, bar, "dht-prev"), anchorByClass(t, bar, "dht-next")

			for _, c := range []struct {
				link, id, name string
				banned         []string
			}{
				{prev, "media.previous", i18n.T(lang, "Previous page"), backWords},
				{next, "media.next", i18n.T(lang, "Next page"), forwardWords},
			} {
				if !strings.Contains(c.link, `d="`+Glyphs[c.id]+`"`) {
					t.Errorf("%s link does not draw %s: %s", c.id, c.id, c.link)
				}
				if !strings.Contains(c.link, c.name) {
					t.Errorf("%s link does not carry its name %q: %s", c.id, c.name, c.link)
				}
				for _, w := range c.banned {
					if c.name == w {
						t.Errorf("%s is named %q, another meaning's word", c.id, w)
					}
				}
			}
			for _, legacy := range []string{"&#9664;", "&#9654;", "&#9776;", "◀", "▶", "☰"} {
				if strings.Contains(bar, legacy) {
					t.Errorf("the bar still draws the pre-contract glyph %q", legacy)
				}
			}
			toc := regexp.MustCompile(`<a class="dht-nav-link" href="index.html">.*?</a>`).FindString(bar)
			if !strings.Contains(toc, Glyphs["nav.contents"]) || !strings.Contains(toc, i18n.T(lang, "Table of contents")) {
				t.Errorf("the contents link is not nav.contents with its name: %s", toc)
			}
		})
	}
}

// A disabled end of the book keeps the same glyph, so the pair reads the same on every page.
func TestDisabledPagingKeepsItsGlyph(t *testing.T) {
	bar := buildNavBarHTML(NavInfo{IndexHref: "index.html", Current: 1, Total: 1})
	for _, id := range []string{"media.previous", "media.next"} {
		if !strings.Contains(bar, Glyphs[id]) {
			t.Errorf("a one-page book's disabled link does not draw %s", id)
		}
	}
}

// Every glyph-only control carries a word as its accessible name (ICON-RENDER rule 8), and the
// theme options no longer borrow weather.clear, app.night-mode and camera.record-video.
func TestReaderControlsNameEveryGlyphOnlyControl(t *testing.T) {
	// Each glyph-only control draws its meaning and is named by that meaning's record, in every
	// language (ICON-RENDER rule 8, ICON-SET 0.15).
	defer i18n.SetLanguage(i18n.Language())
	controls := []struct{ id, glyph, name string }{
		{"dht-font-dec", "action.text-smaller", "Smaller text"},
		{"dht-font-inc", "action.text-larger", "Larger text"},
		{"dht-ocr-toggle", "view.text-layer", "Text layer"},
	}
	for _, lang := range i18n.Codes {
		i18n.SetLanguage(lang)
		html := readerControlsHTML()
		for _, c := range controls {
			m := regexp.MustCompile(`(?s)<button id="` + c.id + `"[^>]*>.*?</button>`).FindString(html)
			if !strings.Contains(m, `aria-label="`+i18n.T(lang, c.name)+`"`) || !strings.Contains(m, `d="`+Glyphs[c.glyph]+`"`) {
				t.Errorf("%s: %s should draw %s named %q: %s", lang, c.id, c.glyph, i18n.T(lang, c.name), m)
			}
		}
		if !strings.Contains(html, `d="`+Glyphs["app.theme"]+`"`) {
			t.Errorf("%s: the theme select does not show app.theme", lang)
		}
	}
	i18n.SetLanguage("en")
	html := readerControlsHTML()
	for _, legacy := range []string{"A&minus;", "&#9636;", "▤"} {
		if strings.Contains(html, legacy) {
			t.Errorf("a reader control still draws %q", legacy)
		}
	}
	for _, legacy := range []string{"&#9728;", "&#9681;", "&#9790;", "&#9679;", "☀", "◑", "☾", "●"} {
		if strings.Contains(html, legacy) {
			t.Errorf("a theme option still draws %q", legacy)
		}
	}
}

func TestGlyphSVGIsDecorativeAndThemed(t *testing.T) {
	for id := range Glyphs {
		svg := glyphSVG(id)
		for _, want := range []string{`viewBox="0 0 24 24"`, `aria-hidden="true"`, `fill="currentColor"`} {
			if !strings.Contains(svg, want) {
				t.Errorf("%s: %s lacks %s", id, svg, want)
			}
		}
	}
}
