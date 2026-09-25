package htmlsplit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// splitGroup records where the anchors of one split source file landed.
// hrefs[0] is the source href (part 1 overwrites it); ids maps an anchor to
// the index of the part holding it.
type splitGroup struct {
	hrefs []string
	ids   map[string]int
}

// anchorIndex resolves any href of a split file (the source or one of its
// parts), as a cleaned OPF-relative path, to its group.
type anchorIndex map[string]*splitGroup

func newSplitGroup(hrefs []string, parts []splitPart) *splitGroup {
	g := &splitGroup{hrefs: hrefs, ids: make(map[string]int)}
	for i, p := range parts {
		for _, id := range p.anchors {
			if _, dup := g.ids[id]; !dup {
				g.ids[id] = i
			}
		}
	}
	return g
}

func (ix anchorIndex) add(g *splitGroup) {
	for _, h := range g.hrefs {
		ix[path.Clean(h)] = g
	}
}

// relocate returns the href of the part now holding frag inside target,
// and whether that differs from target itself.
func (ix anchorIndex) relocate(target, frag, rawFrag string) (string, bool) {
	g := ix[target]
	if g == nil {
		return "", false
	}
	i, ok := g.ids[frag]
	if !ok {
		i, ok = g.ids[rawFrag]
	}
	if !ok {
		return "", false
	}
	moved := path.Clean(g.hrefs[i])
	return moved, moved != target
}

// rewriteTOC points TOC entries whose fragment moved to a later part at that
// part. A TOC href is an escaped URL over the decoded manifest paths the index
// is keyed by, so the file part is decoded to look up and re-escaped to write.
func rewriteTOC(entries []epub.TOCEntry, ix anchorIndex) {
	for i := range entries {
		e := &entries[i]
		if file, frag, ok := strings.Cut(e.Href, "#"); ok && file != "" && frag != "" {
			if u, err := url.PathUnescape(file); err == nil {
				file = u
			}
			if moved, changed := ix.relocate(path.Clean(file), frag, frag); changed {
				e.Href = epub.URLPath(moved) + "#" + frag
			}
		}
		rewriteTOC(e.Children, ix)
	}
}

// rewriteContentLinks re-points links in every HTML content file whose target
// anchor moved to another part. Files without a changed link are not
// re-rendered, so their bytes stay as they are.
func rewriteContentLinks(manifest []epub.ManifestItem, outputDir, basePath string, ix anchorIndex) error {
	seen := make(map[string]bool, len(manifest))
	for _, item := range manifest {
		href := path.Clean(item.Href)
		if !isHTMLMedia(item.MediaType) || seen[href] {
			continue
		}
		seen[href] = true

		p := resolveHref(outputDir, basePath, item.Href)
		data, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			// A manifest may list files the source never shipped; nothing links there to fix.
			continue
		}
		if err != nil {
			return err
		}
		if !bytes.ContainsRune(data, '#') {
			continue
		}
		doc, err := gohtml.Parse(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", item.Href, err)
		}
		if !rewriteLinks(doc, href, ix) {
			continue
		}
		var buf bytes.Buffer
		if err := gohtml.Render(&buf, doc); err != nil {
			return fmt.Errorf("render %s: %w", item.Href, err)
		}
		if err := fsutil.WriteFile(p, buf.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// rewriteLinks updates href attributes of <a> and <area> in doc, which lives
// at self (a cleaned OPF-relative path). It reports whether anything changed.
func rewriteLinks(doc *gohtml.Node, self string, ix anchorIndex) bool {
	dir := path.Dir(self)
	changed := false
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode && (n.DataAtom == atom.A || n.DataAtom == atom.Area) {
			for i, a := range n.Attr {
				if a.Namespace != "" || a.Key != "href" {
					continue
				}
				if v, ok := retarget(a.Val, dir, self, ix); ok {
					n.Attr[i].Val = v
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

// retarget rewrites one link found in the file at self. Only in-book
// relative links with a fragment are candidates; anything with a scheme,
// host, query or absolute path is left alone.
func retarget(ref, dir, self string, ix anchorIndex) (string, bool) {
	ref = strings.TrimSpace(ref)
	_, rawFrag, ok := strings.Cut(ref, "#")
	if !ok || rawFrag == "" {
		return "", false
	}
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "" || u.Host != "" || u.Opaque != "" || u.RawQuery != "" || strings.HasPrefix(u.Path, "/") {
		return "", false
	}
	target := self
	if u.Path != "" {
		target = path.Clean(path.Join(dir, u.Path))
	}
	moved, changed := ix.relocate(target, u.Fragment, rawFrag)
	if !changed {
		return "", false
	}
	if moved == self {
		return "#" + rawFrag, true
	}
	return (&url.URL{Path: relSlash(dir, moved)}).EscapedPath() + "#" + rawFrag, true
}

// relSlash returns the slash path from directory from to file to, both clean
// OPF-relative paths.
func relSlash(from, to string) string {
	var fromSegs []string
	if from != "." {
		fromSegs = strings.Split(from, "/")
	}
	toSegs := strings.Split(to, "/")
	common := 0
	for common < len(fromSegs) && common < len(toSegs)-1 && fromSegs[common] == toSegs[common] {
		common++
	}
	up := strings.Repeat("../", len(fromSegs)-common)
	return up + strings.Join(toSegs[common:], "/")
}
