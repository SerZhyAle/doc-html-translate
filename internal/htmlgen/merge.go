package htmlgen

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"doc-html-translate/internal/epub"

	gohtml "golang.org/x/net/html"
)

// mergeChapter is one parsed spine page on its way into the single-page merge.
type mergeChapter struct {
	href string       // cleaned OPF-relative path of the source page
	doc  *gohtml.Node // the parsed page; its <body> children become the merged content

	// ids maps each anchor as authored (id, or <a name>) to the id it carries in
	// the merged page. rootIDs are the ids of the page's <html> and <body>, which
	// do not survive the merge; links to them land on the chapter marker.
	ids     map[string]string
	rootIDs map[string]bool

	// anchor is the id of the marker opening the chapter in the merged page. It
	// is reserved for every chapter but only inserted when a link needs it, so a
	// book without cross-references merges exactly as before.
	anchor     string
	anchorUsed bool

	// scopedCSS holds the scoped CSS rules from this chapter's <style> blocks.
	scopedCSS []string
}

// chromeIDs are the ids the merged page's own chrome carries. A book id equal
// to one of them is renamed, or the reader controls would bind to book content.
var chromeIDs = []string{
	"dht-single-css", "dht-nav", "dht-reader-css", "dht-scoped-css", "dht-reader", "dht-progress", "dht-page-sel",
	"dht-font-dec", "dht-font-inc", "dht-ocr-toggle", "dht-family-sel", "dht-theme-sel",
	"dht-continue", "dht-zoom-sync",
}

// idRefAttrs hold space-separated id references that must follow a renamed id.
var idRefAttrs = map[string]bool{
	"for": true, "headers": true, "aria-labelledby": true, "aria-describedby": true,
	"aria-controls": true, "aria-owns": true,
}

// prepareMerge makes parsed spine pages safe to concatenate into one document
// that sits in the OPF base directory:
//   - an id or <a name> an earlier page already uses is renamed "cN-<id>"; the
//     first holder keeps it, so book CSS aimed at ids keeps working whenever
//     nothing collides;
//   - relative references in link attributes, style attributes and <style>
//     blocks are rebased from the page's folder to the base folder, because
//     the common Sigil layout keeps chapters in Text/ and images in Images/;
//   - a link to a spine page, with or without a fragment, becomes an in-page
//     anchor, since the chapter files are removed after the merge.
func prepareMerge(chapters []*mergeChapter) {
	used := make(map[string]bool, len(chromeIDs))
	for _, id := range chromeIDs {
		used[id] = true
	}
	counter := 0
	for i, ch := range chapters {
		ch.assignIDs(i, used, &counter)
	}
	index := make(map[string]int, len(chapters))
	for i, ch := range chapters {
		ch.anchor = uniqueID(fmt.Sprintf("dht-ch-%d", i+1), used, &counter)
		if _, dup := index[ch.href]; !dup {
			index[ch.href] = i
		}
	}
	for i, ch := range chapters {
		ch.rewriteRefs(chapters, index, i+1)
	}
	for _, ch := range chapters {
		if ch.anchorUsed {
			ch.insertAnchor()
		}
	}
}

// assignIDs records and, where they collide, renames the anchors of the page.
// A value repeated inside the same page maps to one merged id, so the page's
// own duplicate stays a duplicate rather than being split in two.
func (ch *mergeChapter) assignIDs(pos int, used map[string]bool, counter *int) {
	ch.ids = make(map[string]string)
	ch.rootIDs = make(map[string]bool)
	body := findBodyNode(ch.doc)
	for n := body; n != nil; n = n.Parent {
		if n.Type == gohtml.ElementNode {
			if id := nodeAttr(n, "id"); id != "" {
				ch.rootIDs[id] = true
			}
		}
	}
	if body == nil {
		return
	}
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			for i := range n.Attr {
				a := &n.Attr[i]
				isAnchor := a.Namespace == "" && (a.Key == "id" || (a.Key == "name" && n.Data == "a" && n.Namespace == ""))
				if !isAnchor || a.Val == "" {
					continue
				}
				if merged, seen := ch.ids[a.Val]; seen {
					a.Val = merged
					continue
				}
				merged := a.Val
				if used[merged] {
					merged = uniqueID(fmt.Sprintf("c%d-%s", pos+1, a.Val), used, counter)
				} else {
					used[merged] = true
				}
				ch.ids[a.Val] = merged
				a.Val = merged
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		walk(c)
	}
}

// lookupID resolves a fragment against the page's anchors, trying it as
// written and then percent-decoded, the order a browser uses.
func (ch *mergeChapter) lookupID(frag string) (string, bool) {
	if id, ok := ch.ids[frag]; ok {
		return id, true
	}
	if dec, err := url.PathUnescape(frag); err == nil && dec != frag {
		if id, ok := ch.ids[dec]; ok {
			return id, true
		}
	}
	return "", false
}

// targetID is the merged-page id a link to this page with fragment frag lands
// on. An unknown fragment, none, or one naming the page's root lands on the
// chapter marker: the start of the right chapter beats a dead link.
func (ch *mergeChapter) targetID(frag string) string {
	if frag != "" {
		if id, ok := ch.lookupID(frag); ok {
			return id
		}
	}
	ch.anchorUsed = true
	return ch.anchor
}

// rewriteRefs points the page's references at what they meant before the
// merge: rebased paths, renamed ids, in-page anchors for spine pages.
func (ch *mergeChapter) rewriteRefs(chapters []*mergeChapter, index map[string]int, pos int) {
	body := findBodyNode(ch.doc)
	if body == nil {
		return
	}
	dir := path.Dir(ch.href)
	epub.RewriteURLs(ch.doc, func(v string) (string, bool) {
		t := strings.TrimSpace(v)
		if strings.HasPrefix(t, "#") {
			return ch.sameDocLink(t[1:])
		}
		target, suffix, ok := resolveRelative(dir, t)
		if !ok {
			return v, false
		}
		if k, isSpine := index[target]; isSpine {
			_, frag, _ := strings.Cut(suffix, "#")
			return "#" + epub.URLPath(chapters[k].targetID(frag)), true
		}
		if dir == "." {
			return v, false
		}
		return epub.URLPath(target) + suffix, true
	})
	rebaseCSS := func(ref string) (string, bool) {
		target, suffix, ok := resolveRelative(dir, strings.TrimSpace(ref))
		if !ok || dir == "." {
			return ref, false
		}
		return epub.URLPath(target) + suffix, true
	}
	scopeClass := fmt.Sprintf(".dht-ch-%d", pos)
	var styleNodesToRemove []*gohtml.Node
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			for i := range n.Attr {
				a := &n.Attr[i]
				switch {
				case a.Namespace == "" && a.Key == "style":
					a.Val, _ = epub.RewriteCSSURLs(a.Val, rebaseCSS)
				case a.Namespace == "" && idRefAttrs[a.Key]:
					a.Val = ch.renameIDList(a.Val)
				}
			}
			if n.Data == "style" {
				var styleText strings.Builder
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == gohtml.TextNode {
						styleText.WriteString(c.Data)
					}
				}
				rawCSS := styleText.String()
				rebasedCSS, _ := epub.RewriteCSSURLs(rawCSS, rebaseCSS)
				scoped := ScopeCSS(rebasedCSS, scopeClass, ch.ids)
				if strings.TrimSpace(scoped) != "" {
					ch.scopedCSS = append(ch.scopedCSS, scoped)
				}
				styleNodesToRemove = append(styleNodesToRemove, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(ch.doc)
	for _, n := range styleNodesToRemove {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}
}

// sameDocLink rewrites "#frag" when the id it names was renamed or was a root id.
func (ch *mergeChapter) sameDocLink(frag string) (string, bool) {
	if frag == "" {
		return "", false
	}
	if id, ok := ch.lookupID(frag); ok {
		if id == frag {
			return "", false
		}
		return "#" + epub.URLPath(id), true
	}
	if ch.rootIDs[frag] {
		ch.anchorUsed = true
		return "#" + epub.URLPath(ch.anchor), true
	}
	return "", false
}

func (ch *mergeChapter) renameIDList(v string) string {
	fields := strings.Fields(v)
	changed := false
	for i, f := range fields {
		if id, ok := ch.ids[f]; ok && id != f {
			fields[i] = id
			changed = true
		}
	}
	if !changed {
		return v
	}
	return strings.Join(fields, " ")
}

// insertAnchor places the chapter marker as the first child of the page body.
func (ch *mergeChapter) insertAnchor() {
	body := findBodyNode(ch.doc)
	if body == nil {
		return
	}
	marker := &gohtml.Node{
		Type: gohtml.ElementNode,
		Data: "div",
		Attr: []gohtml.Attribute{{Key: "id", Val: ch.anchor}, {Key: "class", Val: "dht-chapter-anchor"}},
	}
	body.InsertBefore(marker, body.FirstChild)
}

// resolveRelative resolves a relative reference found in a page under dir to
// a cleaned OPF-relative path, split from its ?query / #fragment suffix. It
// declines what is not a relative path: empty, fragment-only, root-relative,
// network or any scheme.
func resolveRelative(dir, ref string) (target, suffix string, ok bool) {
	if ref == "" || ref[0] == '#' || ref[0] == '/' || ref[0] == '\\' {
		return "", "", false
	}
	if external, _ := epub.ExternalHref(ref); external {
		return "", "", false
	}
	p := ref
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		p, suffix = ref[:i], ref[i:]
	}
	if p == "" {
		return "", "", false
	}
	if dec, err := url.PathUnescape(p); err == nil {
		p = dec
	}
	return path.Clean(path.Join(dir, p)), suffix, true
}
