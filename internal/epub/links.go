package epub

import (
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	gohtml "golang.org/x/net/html"
)

// linkAttrs lists, per HTML element, the attributes that hold a URL the
// browser follows or loads. Only these are ever rewritten, so prose, inline
// scripts and unrelated attributes that merely contain a file name stay intact.
var linkAttrs = map[string][]string{
	"a":      {"href"},
	"area":   {"href"},
	"link":   {"href"},
	"img":    {"src", "srcset"},
	"source": {"src", "srcset"},
	"iframe": {"src"},
	"embed":  {"src"},
	"script": {"src"},
	"audio":  {"src"},
	"video":  {"src", "poster"},
	"track":  {"src"},
	"object": {"data"},
}

// svgLinkElements are the SVG elements whose href / xlink:href is a link.
var svgLinkElements = map[string]bool{"a": true, "image": true, "use": true}

// urlSchemeRe matches a URL scheme prefix ("https:", "mailto:", "data:", ..).
var urlSchemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// rewriteLinks offers every in-book link of doc to fn and writes back what fn
// returns. It is the single link-rewrite primitive of the package: renames
// (xhtml -> html, reserved names) use it today, and it is meant to be reused by
// href containment and the page splitter rather than re-implemented.
//
// fileHref is the OPF-relative href of the document doc was parsed from. Each
// link value is resolved against that file's directory the way a browser
// would, stripped of ?query and #fragment and percent-decoding, and fn receives
// the cleaned OPF-relative target path (it may start with ".." when a link
// escapes the OPF directory - fn decides what to do with it). When fn returns
// ok with a different path, the attribute is rewritten to a value relative to
// the same file, keeping the original query and fragment. Links fn declines
// are left byte-for-byte unchanged.
//
// External links (any scheme, protocol-relative "//", root-relative "/") and
// same-document links ("#id", "") are never offered. Attributes are visited
// in document order, so the result is deterministic. Returns whether any
// attribute changed.
func rewriteLinks(doc *gohtml.Node, fileHref string, fn func(target string) (string, bool)) bool {
	baseDir := path.Dir(fileHref)
	changed := false
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			for i := range n.Attr {
				a := &n.Attr[i]
				kind := linkAttrKind(n, a)
				if kind == "" {
					continue
				}
				var nv string
				var ok bool
				if kind == "srcset" {
					nv, ok = rewriteSrcset(baseDir, a.Val, fn)
				} else {
					nv, ok = rewriteLinkValue(baseDir, a.Val, fn)
				}
				if ok {
					a.Val = nv
					changed = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return changed
}

// linkAttrKind classifies a as "url", "srcset", or "" when it is not a link.
func linkAttrKind(n *gohtml.Node, a *gohtml.Attribute) string {
	switch n.Namespace {
	case "":
		if a.Namespace != "" {
			return ""
		}
		for _, k := range linkAttrs[n.Data] {
			if k == a.Key {
				if k == "srcset" {
					return "srcset"
				}
				return "url"
			}
		}
	case "svg":
		if svgLinkElements[n.Data] && a.Key == "href" && (a.Namespace == "" || a.Namespace == "xlink") {
			return "url"
		}
	}
	return ""
}

// rewriteLinkValue rewrites a single URL value found in a file under baseDir.
func rewriteLinkValue(baseDir, value string, fn func(string) (string, bool)) (string, bool) {
	v := strings.TrimSpace(value)
	if v == "" || strings.HasPrefix(v, "/") || strings.HasPrefix(v, `\`) || urlSchemeRe.MatchString(v) {
		return value, false
	}
	p, suffix := v, ""
	if i := strings.IndexAny(v, "?#"); i >= 0 {
		p, suffix = v[:i], v[i:]
	}
	if p == "" {
		return value, false
	}
	unescaped, err := url.PathUnescape(p)
	if err != nil {
		unescaped = p
	}
	target := path.Clean(path.Join(baseDir, unescaped))
	newTarget, ok := fn(target)
	if !ok || newTarget == target {
		return value, false
	}
	rel, err := filepath.Rel(filepath.FromSlash(baseDir), filepath.FromSlash(newTarget))
	if err != nil {
		return value, false
	}
	rel = filepath.ToSlash(rel)
	if unescaped != p {
		rel = (&url.URL{Path: rel}).EscapedPath()
	}
	return rel + suffix, true
}

// rewriteSrcset rewrites the URL part of each srcset candidate, copying
// separators and width/density descriptors verbatim. A candidate URL runs up to
// whitespace (so commas inside a data: URL stay part of it); trailing commas on
// it are separators, as in the HTML srcset parsing algorithm.
func rewriteSrcset(baseDir, value string, fn func(string) (string, bool)) (string, bool) {
	var b strings.Builder
	changed := false
	i := 0
	for i < len(value) {
		j := i
		for j < len(value) && (isHTMLSpace(value[j]) || value[j] == ',') {
			j++
		}
		b.WriteString(value[i:j])
		i = j
		if i >= len(value) {
			break
		}
		for j < len(value) && !isHTMLSpace(value[j]) {
			j++
		}
		u := value[i:j]
		trail := len(u)
		for trail > 0 && u[trail-1] == ',' {
			trail--
		}
		if nu, ok := rewriteLinkValue(baseDir, u[:trail], fn); ok {
			b.WriteString(nu)
			changed = true
		} else {
			b.WriteString(u[:trail])
		}
		b.WriteString(u[trail:])
		i = j
		if trail < len(u) {
			continue
		}
		for j < len(value) && value[j] != ',' {
			j++
		}
		b.WriteString(value[i:j])
		i = j
	}
	if !changed {
		return value, false
	}
	return b.String(), true
}

func isHTMLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}
