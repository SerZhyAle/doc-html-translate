package htmlsplit

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// splitPart is one rendered page of a split file plus the anchors (element
// ids and legacy <a name>) it holds, in document order.
type splitPart struct {
	html    []byte
	anchors []string
}

// block is one unit of the split plan: either an original node moved as a
// whole, or a shallow copy of a wrapper element holding a subset of its
// children (so a sole <div class="calibre"> is repeated around every part).
type block struct {
	node  *gohtml.Node
	wrap  bool
	first bool // the first copy of a wrapper keeps its id; later copies drop it
	kids  []block
	size  int
}

// chunkHTMLFile reads srcPath and splits its body content into parts whose
// text length is at most maxChars characters. It returns nil when the file
// needs no split, so the caller keeps the original bytes untouched.
func chunkHTMLFile(srcPath string, maxChars int) ([]splitPart, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}

	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	head, body := findHeadBody(doc)
	if body == nil {
		return nil, nil
	}

	p := &planner{max: maxChars, sizes: make(map[*gohtml.Node]int)}
	if p.measure(body) <= maxChars {
		return nil, nil
	}

	groups := p.plan(body)
	if len(groups) <= 1 {
		return nil, nil
	}

	root := body.Parent
	parts := make([]splitPart, 0, len(groups))
	for _, group := range groups {
		page := buildPage(root, head, body, group)
		var buf bytes.Buffer
		if err := gohtml.Render(&buf, page); err != nil {
			return nil, fmt.Errorf("render part: %w", err)
		}
		parts = append(parts, splitPart{html: buf.Bytes(), anchors: collectAnchors(page)})
	}
	return parts, nil
}

// buildPage assembles a standalone document for one group, carrying over the
// source <html> and <body> attributes so language, direction and body styling
// survive on every part.
func buildPage(root, head, body *gohtml.Node, group []block) *gohtml.Node {
	doc := &gohtml.Node{Type: gohtml.DocumentNode}
	doc.AppendChild(&gohtml.Node{Type: gohtml.DoctypeNode, Data: "html"})

	var htmlEl *gohtml.Node
	if root != nil && root.Type == gohtml.ElementNode && root.DataAtom == atom.Html {
		htmlEl = shallowClone(root, true)
	} else {
		htmlEl = &gohtml.Node{Type: gohtml.ElementNode, DataAtom: atom.Html, Data: "html"}
	}
	doc.AppendChild(htmlEl)
	if head != nil {
		htmlEl.AppendChild(deepClone(head))
	}
	bodyEl := shallowClone(body, true)
	htmlEl.AppendChild(bodyEl)
	for _, b := range group {
		bodyEl.AppendChild(materialize(b))
	}
	return doc
}

// planner partitions a body into groups of blocks, descending into oversized
// wrapper elements. Text sizes are memoized so the whole plan stays linear.
type planner struct {
	max   int
	sizes map[*gohtml.Node]int
}

// measure returns the visible character count of n and records it for every
// descendant. Script and style content is excluded.
func (p *planner) measure(n *gohtml.Node) int {
	size := 0
	switch {
	case n.Type == gohtml.TextNode:
		size = utf8.RuneCountInString(strings.TrimSpace(n.Data))
	case n.Type == gohtml.ElementNode && (n.DataAtom == atom.Script || n.DataAtom == atom.Style):
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			size += p.measure(c)
		}
	}
	p.sizes[n] = size
	return size
}

// plan groups the significant children of container. An oversized child that
// is a pure block container is split recursively; each resulting subgroup
// becomes a wrapper copy that joins the greedy grouping like any other block.
func (p *planner) plan(container *gohtml.Node) [][]block {
	var items []block
	for c := container.FirstChild; c != nil; c = c.NextSibling {
		if !isSignificant(c) {
			continue
		}
		size := p.sizes[c]
		if size > p.max && isBlockContainer(c) {
			if sub := p.plan(c); len(sub) > 1 {
				for i, g := range sub {
					items = append(items, block{node: c, wrap: true, first: i == 0, kids: g, size: groupSize(g)})
				}
				continue
			}
		}
		items = append(items, block{node: c, size: size})
	}
	return p.group(items)
}

// group partitions blocks so each group's text length is at most max.
// A single oversized block is kept as its own group (no mid-block splitting).
func (p *planner) group(items []block) [][]block {
	var groups [][]block
	var current []block
	currentLen := 0
	for _, b := range items {
		if len(current) > 0 && currentLen+b.size > p.max {
			groups = append(groups, current)
			current = nil
			currentLen = 0
		}
		current = append(current, b)
		currentLen += b.size
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

func groupSize(g []block) int {
	total := 0
	for _, b := range g {
		total += b.size
	}
	return total
}

// isSignificant reports whether n carries content worth placing in a part.
// Whitespace-only text and comments between blocks are dropped.
func isSignificant(n *gohtml.Node) bool {
	switch n.Type {
	case gohtml.ElementNode:
		return true
	case gohtml.TextNode:
		return strings.TrimSpace(n.Data) != ""
	}
	return false
}

var wrapperAtoms = map[atom.Atom]bool{
	atom.Div: true, atom.Section: true, atom.Article: true, atom.Main: true,
	atom.Aside: true, atom.Blockquote: true, atom.Header: true, atom.Footer: true,
	atom.Nav: true,
}

var phrasingAtoms = map[atom.Atom]bool{
	atom.A: true, atom.Abbr: true, atom.B: true, atom.Bdi: true, atom.Bdo: true,
	atom.Br: true, atom.Cite: true, atom.Code: true, atom.Dfn: true, atom.Em: true,
	atom.Font: true, atom.I: true, atom.Img: true, atom.Kbd: true, atom.Label: true,
	atom.Mark: true, atom.Q: true, atom.Ruby: true, atom.S: true, atom.Samp: true,
	atom.Small: true, atom.Span: true, atom.Strong: true, atom.Sub: true, atom.Sup: true,
	atom.Time: true, atom.U: true, atom.Var: true, atom.Wbr: true, atom.Tt: true,
	atom.Big: true, atom.Strike: true,
}

// isBlockContainer reports whether n is a wrapper whose children are all
// blocks. A wrapper with loose text or inline children is running prose, and
// splitting between those children would cut a sentence.
func isBlockContainer(n *gohtml.Node) bool {
	if n.Type != gohtml.ElementNode || !wrapperAtoms[n.DataAtom] {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case gohtml.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				return false
			}
		case gohtml.ElementNode:
			if phrasingAtoms[c.DataAtom] {
				return false
			}
		}
	}
	return true
}

// materialize turns a planned block into a node for the part being built,
// moving original nodes out of the source tree.
func materialize(b block) *gohtml.Node {
	if !b.wrap {
		if b.node.Parent != nil {
			b.node.Parent.RemoveChild(b.node)
		}
		return b.node
	}
	w := shallowClone(b.node, b.first)
	for _, k := range b.kids {
		w.AppendChild(materialize(k))
	}
	return w
}

// shallowClone copies an element without its children; keepID false drops
// the id so a wrapper repeated across parts does not claim one anchor twice.
func shallowClone(n *gohtml.Node, keepID bool) *gohtml.Node {
	c := &gohtml.Node{Type: n.Type, DataAtom: n.DataAtom, Data: n.Data, Namespace: n.Namespace}
	c.Attr = make([]gohtml.Attribute, 0, len(n.Attr))
	for _, a := range n.Attr {
		if !keepID && a.Namespace == "" && a.Key == "id" {
			continue
		}
		c.Attr = append(c.Attr, a)
	}
	return c
}

func deepClone(n *gohtml.Node) *gohtml.Node {
	c := shallowClone(n, true)
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		c.AppendChild(deepClone(k))
	}
	return c
}

// collectAnchors lists the link targets inside a page's body: every element
// id and every legacy <a name>.
func collectAnchors(page *gohtml.Node) []string {
	var out []string
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			if n.DataAtom == atom.Head {
				return
			}
			for _, a := range n.Attr {
				if a.Namespace != "" || a.Val == "" {
					continue
				}
				if a.Key == "id" || (a.Key == "name" && n.DataAtom == atom.A) {
					out = append(out, a.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(page)
	return out
}

// findHeadBody locates the <head> and <body> nodes in a parsed document.
func findHeadBody(doc *gohtml.Node) (head, body *gohtml.Node) {
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			switch n.DataAtom {
			case atom.Head:
				if head == nil {
					head = n
				}
			case atom.Body:
				if body == nil {
					body = n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return
}
