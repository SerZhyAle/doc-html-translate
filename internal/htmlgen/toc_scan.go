package htmlgen

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"doc-html-translate/internal/epub"

	gohtml "golang.org/x/net/html"
)

// BuildFallbackTOC builds a synthetic multi-level TOC for books that ship no
// authored navigation (PDF without bookmarks, TXT/MD/FB2/RTF/HTML/MOBI). Each
// spine page becomes a top-level entry (label = translated snippet or filename),
// and the page's own h1–h6 headings become nested children, each anchored to a
// stable injected id. Page files are rewritten in place when an id is injected.
//
// It should run after translation so heading labels are already translated and
// the anchors land in the files the reader opens. Returns nil if no spine.
func BuildFallbackTOC(book *epub.Book, outputDir string, snippets map[string]string) []epub.TOCEntry {
	spineHrefs := book.SpineHrefs()
	if len(spineHrefs) == 0 {
		return nil
	}

	var top []epub.TOCEntry
	for i, href := range spineHrefs {
		pagePath := bookPath(outputDir, book.BasePath, href)

		label := ""
		if snippets != nil {
			label = snippets[href]
		}
		if label == "" {
			label = extractSnippet(pagePath)
		}
		if label == "" {
			label = chapterLabel(href, i+1)
		}

		headings := scanAndAnchorHeadings(pagePath)
		top = append(top, epub.TOCEntry{
			Title:    label,
			Href:     href,
			Children: nestHeadings(headings, href),
		})
	}
	return top
}

// flatHeading is one heading found on a page, with its level (1–6 for h1–h6),
// translated text, and (existing or injected) anchor id.
type flatHeading struct {
	level int
	title string
	id    string
}

// scanAndAnchorHeadings parses an HTML page, collects its content headings
// (skipping injected navbar/header chrome), ensures each has a stable id
// (injecting one when missing), rewrites the file if anything changed, and
// returns the headings in document order. Best-effort: returns nil on any I/O
// or parse error.
func scanAndAnchorHeadings(pagePath string) []flatHeading {
	data, err := os.ReadFile(pagePath)
	if err != nil {
		return nil
	}
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	body := findBodyNode(doc)
	if body == nil {
		return nil
	}

	// Pre-collect every existing id so injected ids never collide.
	used := make(map[string]bool)
	collectIDs(doc, used)

	var headings []flatHeading
	counter := 0
	injected := false

	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			if isStructuralSkip(n) {
				return // do not descend into navbar / page-header / image chrome
			}
			if lvl := headingLevel(n.Data); lvl > 0 {
				title := strings.TrimSpace(textContent(n))
				if title != "" {
					id := nodeAttr(n, "id")
					if id == "" {
						id = uniqueID(slugify(title), used, &counter)
						setAttr(n, "id", id)
						injected = true
					}
					headings = append(headings, flatHeading{level: lvl, title: title, id: id})
				}
				return // headings don't nest other headings
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(body)

	if injected {
		var buf bytes.Buffer
		if err := gohtml.Render(&buf, doc); err == nil {
			_ = os.WriteFile(pagePath, buf.Bytes(), 0o644)
		}
	}

	return headings
}

// nestHeadings turns a flat, in-order heading list into a tree by heading level,
// linking each entry to href#id. A deeper heading nests under the nearest
// shallower one (skipped levels are tolerated).
func nestHeadings(flat []flatHeading, href string) []epub.TOCEntry {
	type node struct {
		entry    epub.TOCEntry
		level    int
		children []*node
	}
	var roots []*node
	var stack []*node

	for _, h := range flat {
		n := &node{
			entry: epub.TOCEntry{Title: h.title, Href: href + "#" + h.id},
			level: h.level,
		}
		for len(stack) > 0 && stack[len(stack)-1].level >= h.level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			parent := stack[len(stack)-1]
			parent.children = append(parent.children, n)
		}
		stack = append(stack, n)
	}

	var convert func([]*node) []epub.TOCEntry
	convert = func(nodes []*node) []epub.TOCEntry {
		var out []epub.TOCEntry
		for _, n := range nodes {
			e := n.entry
			e.Children = convert(n.children)
			out = append(out, e)
		}
		return out
	}
	return convert(roots)
}

// headingLevel returns 1–6 for "h1".."h6", else 0.
func headingLevel(tag string) int {
	if len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6' {
		return int(tag[1] - '0')
	}
	return 0
}

// isStructuralSkip reports whether an element is injected chrome (navbar,
// page header, image wrapper) or a non-content container that should not be
// descended into when scanning for headings. Mirrors ExtractSnippetFromDoc.
func isStructuralSkip(n *gohtml.Node) bool {
	switch n.Data {
	case "script", "style", "head", "nav", "button":
		return true
	case "div", "header", "footer":
		if nodeHasClass(n, "page-header") || nodeHasClass(n, "pdf-page-header") ||
			nodeHasClass(n, "dht-navbar") || nodeHasClass(n, "pdf-images") ||
			nodeAttr(n, "id") == "dht-nav" || nodeAttr(n, "id") == "dht-zoom-sync" {
			return true
		}
	}
	return false
}

// collectIDs records every existing id attribute in the document.
func collectIDs(n *gohtml.Node, used map[string]bool) {
	if n.Type == gohtml.ElementNode {
		if id := nodeAttr(n, "id"); id != "" {
			used[id] = true
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectIDs(c, used)
	}
}

// setAttr sets (or replaces) an attribute on an element node.
func setAttr(n *gohtml.Node, key, val string) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, gohtml.Attribute{Key: key, Val: val})
}

// uniqueID returns an id not already in `used`, derived from base when possible
// (base, base-2, …) and otherwise from a "dht-toc-N" counter. The chosen id is
// added to `used`.
func uniqueID(base string, used map[string]bool, counter *int) string {
	if base != "" && !used[base] {
		used[base] = true
		return base
	}
	if base != "" {
		for i := 2; i < 1000; i++ {
			c := fmt.Sprintf("%s-%d", base, i)
			if !used[c] {
				used[c] = true
				return c
			}
		}
	}
	for {
		*counter++
		c := fmt.Sprintf("dht-toc-%d", *counter)
		if !used[c] {
			used[c] = true
			return c
		}
	}
}

// slugify produces a short ascii-lowercase id fragment from heading text.
// Non-ascii text (e.g. translated Cyrillic) yields "" so the caller falls back
// to a counter-based id.
func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = strings.Trim(out[:40], "-")
	}
	return out
}

// textContent concatenates the visible text under n, collapsing whitespace.
func textContent(n *gohtml.Node) string {
	var sb strings.Builder
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.TextNode {
			for _, f := range strings.Fields(n.Data) {
				if sb.Len() > 0 {
					sb.WriteByte(' ')
				}
				sb.WriteString(f)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}
