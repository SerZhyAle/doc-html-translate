package tests

// SITE-REPRESENTATION rules 5 to 10 read on the manual (ticket 107). The site has no function pages: its guide pages
// are the task sections of the manual (docs.html and its ru and uk twins), so the fixed anatomy of rule 7 is held
// there. A task section is, in order: the title, a lead, the requirements (the editions it exists in, from the one
// list of docs/positioning.json, rule 9), the numbered steps (a step that happens in our own window carries one
// figure), one outcome callout and the related links. What no page can show - that a figure is the window of the
// build - is read by the browser run (tools/sitecheck, C4 and C5).

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var manualTaskSections = []string{"workflow", "quick-start", "logs", "secret-files"}

func hasClass(n *html.Node, class string) bool {
	v, _ := attrOf(n, "class")
	for _, c := range strings.Fields(v) {
		if c == class {
			return true
		}
	}
	return false
}

func elementChildren(n *html.Node) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			out = append(out, c)
		}
	}
	return out
}

func TestManualTaskSectionsFollowTheAnatomy(t *testing.T) {
	for lang, p := range docsPageFor {
		doc := parseSitePage(t, p)
		for _, id := range manualTaskSections {
			sec := findElement(doc, byID(id))
			if sec == nil {
				t.Errorf("%s: no #%s task section", p, id)
				continue
			}
			kids := elementChildren(sec)
			if len(kids) < 6 {
				t.Errorf("%s #%s: %d child elements, want title, lead, requirements, steps, outcome and related", p, id, len(kids))
				continue
			}
			title := kids[0]
			if title.DataAtom != atom.H2 && title.DataAtom != atom.Summary {
				t.Errorf("%s #%s: the first element is <%s>, want the title (<h2> or <summary>)", p, id, title.Data)
			}
			if visibleText(title, "", true) == "" {
				t.Errorf("%s #%s: empty title", p, id)
			}
			if kids[1].DataAtom != atom.P {
				t.Errorf("%s #%s: the lead (<p>) must follow the title, found <%s>", p, id, kids[1].Data)
			}
			reqs := kids[2]
			if reqs.DataAtom != atom.Dl || !hasClass(reqs, "req") {
				t.Errorf("%s #%s: the requirements (dl.req) must follow the lead", p, id)
			} else {
				dts, pills := 0, 0
				walkHTML(reqs, func(n *html.Node) bool {
					if n.DataAtom == atom.Dt {
						dts++
					}
					if hasClass(n, "pill") {
						pills++
					}
					return true
				})
				if dts != 2 || pills < 1 {
					t.Errorf("%s #%s: the requirements hold %d terms and %d edition marks, want 2 and at least 1", p, id, dts, pills)
				}
			}
			steps := kids[3]
			if steps.DataAtom != atom.Ol || !hasClass(steps, "steps") || len(elementChildren(steps)) < 1 {
				t.Errorf("%s #%s: the numbered steps (ol.steps) must follow the requirements", p, id)
			}
			notes := 0
			for _, k := range kids {
				if hasClass(k, "note") {
					notes++
				}
			}
			last, prev := kids[len(kids)-1], kids[len(kids)-2]
			if notes != 1 || !hasClass(prev, "note") {
				t.Errorf("%s #%s: want exactly one outcome callout (.note), placed right before the related links (found %d)", p, id, notes)
			}
			if last.DataAtom != atom.P || !hasClass(last, "related") {
				t.Errorf("%s #%s: the related links (p.related) must close the section", p, id)
			} else if findElement(last, byAtom(atom.A)) == nil {
				t.Errorf("%s #%s: the related functions hold no link", p, id)
			}
			// every figure of a step is a real capture with alt text and its size
			walkHTML(steps, func(n *html.Node) bool {
				if n.DataAtom != atom.Img {
					return true
				}
				src, _ := attrOf(n, "src")
				alt, _ := attrOf(n, "alt")
				_, w := attrOf(n, "width")
				_, h := attrOf(n, "height")
				if strings.TrimSpace(alt) == "" || !w || !h {
					t.Errorf("%s #%s: figure %s needs alt text, width and height", p, id, src)
				}
				if !repoPathExists(src) {
					t.Errorf("%s #%s: figure %s does not exist", p, id, src)
				}
				return true
			})
			_ = lang
		}
	}
}

// Rule 9: the availability marker uses the display names of the editions, from the one list.
func TestManualAvailabilityUsesTheEditionNames(t *testing.T) {
	facts := loadPositioning(t)
	for lang, p := range docsPageFor {
		names := map[string]bool{}
		for _, e := range facts.Editions {
			names[e.Names[lang]] = true
		}
		doc := parseSitePage(t, p)
		marks := 0
		walkHTML(doc, func(n *html.Node) bool {
			if n.DataAtom == atom.Dl && hasClass(n, "req") {
				walkHTML(n, func(m *html.Node) bool {
					if hasClass(m, "pill") {
						marks++
						if got := visibleText(m, "", true); !names[got] {
							t.Errorf("%s: availability mark %q is not an edition display name of docs/positioning.json", p, got)
						}
					}
					return true
				})
				return false
			}
			return true
		})
		if marks == 0 {
			t.Errorf("%s: no availability mark found", p)
		}
	}
}

// Rule 6: the plain convert command is given once, in the workflow that owns it; the quick start links to it.
func TestManualPlainConvertCommandIsGivenOnce(t *testing.T) {
	const plain = `doc-html-translate.exe "book.epub"`
	for _, p := range docsPageFor {
		n := 0
		walkHTML(parseSitePage(t, p), func(e *html.Node) bool {
			if v, ok := attrOf(e, "data-copy"); ok && v == plain {
				n++
			}
			return true
		})
		if n != 1 {
			t.Errorf("%s: the plain convert command is given %d times, want once", p, n)
		}
	}
}

// Rule 5: the two translation engines that need something beside the app say so in their own row.
func TestManualEngineRowsStateTheirRequirement(t *testing.T) {
	for _, p := range docsPageFor {
		doc := parseSitePage(t, p)
		for _, flag := range []string{"-google", "-ollama"} {
			found := false
			walkHTML(doc, func(n *html.Node) bool {
				if n.DataAtom != atom.Tr {
					return true
				}
				cells := elementChildren(n)
				if len(cells) == 3 && visibleText(cells[0], "", true) == flag {
					found = true
					if !strings.Contains(visibleText(cells[2], "", true), "(") {
						t.Errorf("%s: the %s row does not state what it needs", p, flag)
					}
				}
				return true
			})
			if !found {
				t.Errorf("%s: no %s row in the flags table", p, flag)
			}
		}
	}
}
