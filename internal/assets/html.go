package assets

import (
	"strings"

	gohtml "golang.org/x/net/html"
)

// RewriteHTML copies the local assets referenced under root - <img src>, <img srcset>,
// <picture><source srcset>, style attributes, <style> blocks and <link rel=stylesheet> -
// and points each reference at its copy. References resolve from the source root. A
// reference refused for leaving the source tree is dropped (the tag stays, without a
// working source); a remote stylesheet link is removed.
func (c *Copier) RewriteHTML(root *gohtml.Node) {
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		for ch := n.FirstChild; ch != nil; {
			next := ch.NextSibling
			if ch.Type == gohtml.ElementNode && ch.Data == "link" && IsStylesheetLink(ch) {
				c.rewriteLink(ch)
			} else {
				walk(ch)
			}
			ch = next
		}
		if n.Type != gohtml.ElementNode {
			return
		}
		switch n.Data {
		case "img":
			c.rewriteSrc(n)
			c.rewriteSrcsetAttr(n)
		case "source":
			// <source> inside <video>/<audio> names media, not pictures.
			if n.Parent != nil && n.Parent.Type == gohtml.ElementNode && n.Parent.Data == "picture" {
				c.rewriteSrcsetAttr(n)
			}
		case "style":
			for t := n.FirstChild; t != nil; t = t.NextSibling {
				if t.Type == gohtml.TextNode {
					t.Data = c.RewriteCSS(t.Data, c.root)
				}
			}
		}
		if i := attrIndex(n, "style"); i >= 0 {
			n.Attr[i].Val = c.RewriteCSS(n.Attr[i].Val, c.root)
		}
	}
	walk(root)
}

// IsStylesheetLink reports a <link> that applies a stylesheet by default. An alternate
// sheet is not applied until the reader picks it, so it is not one.
func IsStylesheetLink(n *gohtml.Node) bool {
	i := attrIndex(n, "rel")
	if i < 0 {
		return false
	}
	isSheet := false
	for _, tok := range strings.Fields(strings.ToLower(n.Attr[i].Val)) {
		switch tok {
		case "stylesheet":
			isSheet = true
		case "alternate":
			return false
		}
	}
	return isSheet
}

// rewriteLink points a stylesheet link found in the body at its copy, or removes it.
func (c *Copier) rewriteLink(n *gohtml.Node) {
	i := attrIndex(n, "href")
	if i < 0 {
		n.Parent.RemoveChild(n)
		return
	}
	media := ""
	if m := attrIndex(n, "media"); m >= 0 {
		media = n.Attr[m].Val
	}
	name, ok := c.LinkedStylesheet(n.Attr[i].Val, media)
	if !ok {
		n.Parent.RemoveChild(n)
		return
	}
	n.Attr[i].Val = name
	if m := attrIndex(n, "media"); m >= 0 {
		n.Attr = append(n.Attr[:m], n.Attr[m+1:]...) // the media query now lives in the copy
	}
}

func (c *Copier) rewriteSrc(n *gohtml.Node) {
	i := attrIndex(n, "src")
	if i < 0 {
		return
	}
	name, out := c.place(n.Attr[i].Val, c.root, false)
	switch out {
	case copied:
		n.Attr[i].Val = name
	case refused:
		n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
	}
}

func (c *Copier) rewriteSrcsetAttr(n *gohtml.Node) {
	i := attrIndex(n, "srcset")
	if i < 0 {
		return
	}
	if v := c.rewriteSrcset(n.Attr[i].Val); v != "" {
		n.Attr[i].Val = v
	} else {
		n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
	}
}

// rewriteSrcset rewrites each candidate URL of a srcset, keeping its descriptor ("2x",
// "480w"). A refused candidate is dropped; "" means none is left.
func (c *Copier) rewriteSrcset(srcset string) string {
	var parts []string
	for _, cand := range parseSrcset(srcset) {
		u := cand.url
		name, out := c.place(u, c.root, false)
		switch out {
		case copied:
			u = name
		case refused:
			continue
		}
		if cand.desc != "" {
			u += " " + cand.desc
		}
		parts = append(parts, u)
	}
	return strings.Join(parts, ", ")
}

type srcsetCandidate struct {
	url, desc string
}

// parseSrcset splits a srcset the way the HTML spec does: a URL runs to the next
// whitespace (so the comma inside a data: URL is part of it), a trailing comma ends a
// descriptor-less candidate, and a descriptor runs to the next comma outside parentheses.
func parseSrcset(s string) []srcsetCandidate {
	var out []srcsetCandidate
	i := 0
	for i < len(s) {
		for i < len(s) && (isSpace(s[i]) || s[i] == ',') {
			i++
		}
		start := i
		for i < len(s) && !isSpace(s[i]) {
			i++
		}
		u := s[start:i]
		desc := ""
		if strings.HasSuffix(u, ",") {
			u = strings.TrimRight(u, ",")
		} else {
			dstart, depth := i, 0
			for ; i < len(s); i++ {
				switch s[i] {
				case '(':
					depth++
				case ')':
					if depth > 0 {
						depth--
					}
				}
				if s[i] == ',' && depth == 0 {
					break
				}
			}
			desc = strings.TrimSpace(s[dstart:i])
		}
		if u != "" {
			out = append(out, srcsetCandidate{url: u, desc: desc})
		}
	}
	return out
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func attrIndex(n *gohtml.Node, key string) int {
	for i, a := range n.Attr {
		if a.Key == key {
			return i
		}
	}
	return -1
}
