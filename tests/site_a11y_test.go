package tests

// SITE-EXPERIENCE rules 15-17 (docs/contracts/SITE-EXPERIENCE.md), the part a parse can hold: a skip link
// first in <body> that lands on <main>, a labelled <nav> around every navigation, and the 44 px / contrast
// rules that live in assets/site.css. Contrast ratios themselves are measured in a browser (ticket 100).

import (
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func a11yPages() []string { return append(sitePages(), "404.html") }

func TestSiteSkipLinkLeadsToMain(t *testing.T) {
	for _, p := range a11yPages() {
		doc := parseSitePage(t, p)
		body := findElement(doc, byAtom(atom.Body))
		first := body.FirstChild
		for first != nil && (first.Type != html.ElementNode) {
			first = first.NextSibling
		}
		if first == nil || first.DataAtom != atom.A {
			t.Errorf("%s: the first element of <body> is not the skip link", p)
			continue
		}
		if cls, _ := attrOf(first, "class"); cls != "skip-link" {
			t.Errorf("%s: the first element of <body> is %q, want the skip link", p, cls)
		}
		href, _ := attrOf(first, "href")
		if !strings.HasPrefix(href, "#") || len(href) < 2 {
			t.Errorf("%s: skip link href %q is not an in-page anchor", p, href)
			continue
		}
		main := findElement(doc, byAtom(atom.Main))
		if main == nil {
			t.Errorf("%s: no <main>", p)
			continue
		}
		if id, _ := attrOf(main, "id"); id != href[1:] {
			t.Errorf("%s: skip link targets %s but <main> has id %q", p, href, id)
		}
		if v, ok := attrOf(main, "tabindex"); !ok || v != "-1" {
			t.Errorf("%s: <main> needs tabindex=\"-1\" so the skip link moves focus into it", p)
		}
		if strings.TrimSpace(visibleText(first, "", true)) == "" {
			t.Errorf("%s: the skip link has no text", p)
		}
	}
}

func TestSiteNavLandmarksAreLabelled(t *testing.T) {
	for _, p := range a11yPages() {
		doc := parseSitePage(t, p)
		navs := 0
		walkHTML(doc, func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.DataAtom == atom.Nav {
				navs++
				if v, _ := attrOf(n, "aria-label"); strings.TrimSpace(v) == "" {
					t.Errorf("%s: a <nav> without an aria-label", p)
				}
			}
			return true
		})
		// the header link row sits in a nav whenever the header carries a link past the brand
		header := findElement(doc, byAtom(atom.Header))
		walkHTML(header, func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.DataAtom == atom.A {
				if cls, _ := attrOf(n, "class"); strings.Contains(cls, "btn") {
					inNav := false
					for q := n.Parent; q != nil; q = q.Parent {
						if q.DataAtom == atom.Nav {
							inNav = true
						}
					}
					if !inNav {
						t.Errorf("%s: the header link row is outside a <nav>", p)
					}
				}
			}
			return true
		})
		if navs == 0 {
			t.Errorf("%s: no <nav> landmark", p)
		}
	}
}

// Trilingual pages relabel their data-nav landmarks from window.SITE strings, so each language declares it.
func TestSiteNavLabelStringsOnTrilingualPages(t *testing.T) {
	for _, p := range inPageTrioPages {
		src := readRepoFile(t, p)
		if !strings.Contains(src, "data-nav") {
			continue
		}
		for _, l := range []string{"ru", "en", "ua"} {
			if !regexp.MustCompile(`(?m)^\s*` + l + `:\{[^\n]*,nav:'[^']+'`).MatchString(src) {
				t.Errorf("%s: window.SITE strings for %s carry no nav label", p, l)
			}
		}
	}
}

func TestSiteStylesheetKeepsKeyboardAndTargetRules(t *testing.T) {
	css := readRepoFile(t, "assets", "site.css")
	for _, want := range []string{
		".skip-link{", ".skip-link:focus", "main:focus{outline:none}",
		".seg button{min-height:44px;min-width:44px}", ".theme-btn{width:44px;height:44px}", ".btn{min-height:44px}",
		".brand{display:flex;align-items:center;gap:.65rem;min-height:44px;min-width:44px}", ".copybox .cmt-end{display:block}",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("assets/site.css lost %q (SITE-EXPERIENCE 15, 17)", want)
		}
	}
	// the focus ring is never removed: only <main>, a programmatic focus target, may drop it
	for _, m := range regexp.MustCompile(`([^{}]+)\{[^}]*outline:\s*(?:none|0)[^}]*\}`).FindAllStringSubmatch(css, -1) {
		if strings.TrimSpace(m[1]) != "main:focus" {
			t.Errorf("assets/site.css removes the focus outline for %q", strings.TrimSpace(m[1]))
		}
	}
}
