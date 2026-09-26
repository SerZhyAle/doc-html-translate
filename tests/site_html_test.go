package tests

// Parsing shared by the DOC-EXTERNAL-QUALITY site checks (docs/contracts/DOC-QUALITY.md). The pages
// are hand-authored static HTML, so the checks read the parsed tree rather than regex over markup:
// an attribute split across lines or an entity in a title must not hide a defect.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// siteBase is the address GitHub Pages serves the repository root at (robots.txt names it).
const siteBase = "https://serzhyale.github.io/doc-html-translate/"

// inPageTrioPages carry en, ru and ua in one file, each block marked data-l.
var inPageTrioPages = []string{"index.html", "extension.html", "privacy.html", "install-trust.html", "extension-privacy.html"}

func parseSitePage(t *testing.T, page string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(readRepoFile(t, strings.Split(page, "/")...)))
	if err != nil {
		t.Fatalf("%s: %v", page, err)
	}
	return doc
}

func attrOf(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// walkHTML visits n and its descendants in document order; visit returning false skips the subtree.
func walkHTML(n *html.Node, visit func(*html.Node) bool) {
	if !visit(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkHTML(c, visit)
	}
}

func findElement(root *html.Node, match func(*html.Node) bool) *html.Node {
	var found *html.Node
	walkHTML(root, func(n *html.Node) bool {
		if found != nil {
			return false
		}
		if n.Type == html.ElementNode && match(n) {
			found = n
			return false
		}
		return true
	})
	return found
}

func byAtom(a atom.Atom) func(*html.Node) bool {
	return func(n *html.Node) bool { return n.DataAtom == a }
}

func byID(id string) func(*html.Node) bool {
	return func(n *html.Node) bool { v, ok := attrOf(n, "id"); return ok && v == id }
}

// pageAnchors is every fragment a link may point at: element ids and legacy <a name>.
func pageAnchors(doc *html.Node) map[string]bool {
	ids := map[string]bool{}
	walkHTML(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			if v, ok := attrOf(n, "id"); ok {
				ids[v] = true
			}
			if v, ok := attrOf(n, "name"); ok && n.DataAtom == atom.A {
				ids[v] = true
			}
		}
		return true
	})
	return ids
}

var siteSpaceRun = regexp.MustCompile(`\s+`)

// visibleText is the reader-facing text under n. lang filters the in-page language blocks: "" keeps
// everything, otherwise only neutral text and blocks whose data-l equals lang. neutral=false drops
// the unmarked text too, leaving only what was written for that language.
func visibleText(n *html.Node, lang string, neutral bool) string {
	var b strings.Builder
	var rec func(*html.Node, bool)
	rec = func(n *html.Node, inLang bool) {
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.Script, atom.Style, atom.Template, atom.Svg:
				return
			}
			if l, ok := attrOf(n, "data-l"); ok && lang != "" {
				if l != lang {
					return
				}
				inLang = true
			}
		}
		if n.Type == html.TextNode && (inLang || neutral || lang == "") {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rec(c, inLang)
		}
	}
	rec(n, false)
	return strings.TrimSpace(siteSpaceRun.ReplaceAllString(b.String(), " "))
}

// headTitle and metaContent read the static head a crawler sees before any script runs.
func headTitle(doc *html.Node) string {
	if n := findElement(doc, byAtom(atom.Title)); n != nil {
		return strings.TrimSpace(visibleText(n, "", true))
	}
	return ""
}

func metaContent(doc *html.Node, key, name string) (string, bool) {
	n := findElement(doc, func(n *html.Node) bool {
		v, ok := attrOf(n, key)
		return n.DataAtom == atom.Meta && ok && v == name
	})
	if n == nil {
		return "", false
	}
	return attrOf(n, "content")
}

func repoPathExists(rel string) bool {
	_, err := os.Stat(filepath.Join("..", filepath.FromSlash(rel)))
	return err == nil
}
