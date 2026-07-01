package epub

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"strings"

	gohtml "golang.org/x/net/html"
)

// parseTOC populates book.TOC from the authored navigation document.
// EPUB3 nav.xhtml (manifest properties="nav") is preferred; the EPUB2
// toc.ncx navMap is the fallback. Hrefs are resolved to the final, normalized
// manifest hrefs; entries whose target no longer exists are dropped so the TOC
// never contains a broken link. A nil/empty result is not an error — callers
// fall back to a spine/heading-based TOC.
func parseTOC(book *Book, outputDir string) error {
	manifestHrefs := make(map[string]bool, len(book.Manifest))
	for _, it := range book.Manifest {
		manifestHrefs[it.Href] = true
	}

	// EPUB3 nav.xhtml — preferred.
	if navItem, ok := findNavItem(book); ok {
		navPath := bookPath(outputDir, book.BasePath, navItem.Href)
		entries, err := parseNavDoc(navPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: parse nav %s: %v\n", navItem.Href, err)
		} else if resolved := resolveTOC(entries, book, navItem.Href, manifestHrefs); len(resolved) > 0 {
			book.TOC = resolved
			return nil
		}
	}

	// EPUB2 toc.ncx — fallback.
	if ncxItem, ok := findNCXItem(book); ok {
		ncxPath := bookPath(outputDir, book.BasePath, ncxItem.Href)
		entries, err := parseNCXDoc(ncxPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: parse ncx %s: %v\n", ncxItem.Href, err)
		} else if resolved := resolveTOC(entries, book, ncxItem.Href, manifestHrefs); len(resolved) > 0 {
			book.TOC = resolved
		}
	}

	return nil
}

// findNavItem returns the manifest item declaring properties="nav".
func findNavItem(book *Book) (ManifestItem, bool) {
	for _, it := range book.Manifest {
		if hasProperty(it.Properties, "nav") {
			return it, true
		}
	}
	return ManifestItem{}, false
}

// findNCXItem returns the manifest item holding the NCX, located via the
// <spine toc="..."> idref when present, otherwise by media type.
func findNCXItem(book *Book) (ManifestItem, bool) {
	if book.spineTocID != "" {
		for _, it := range book.Manifest {
			if it.ID == book.spineTocID {
				return it, true
			}
		}
	}
	for _, it := range book.Manifest {
		if it.MediaType == "application/x-dtbncx+xml" {
			return it, true
		}
	}
	return ManifestItem{}, false
}

// hasProperty reports whether the whitespace-separated property list contains
// the exact token (so "nav cover" matches "nav" but "navigation" does not).
func hasProperty(properties, token string) bool {
	for _, p := range strings.Fields(properties) {
		if p == token {
			return true
		}
	}
	return false
}

// ── EPUB3 nav.xhtml ─────────────────────────────────────────────────

// parseNavDoc reads an EPUB3 navigation document and returns the raw entries of
// its "toc" <nav>. It uses the lenient HTML parser because nav files are not
// reliably well-formed XML (especially after .xhtml->.html link rewriting).
func parseNavDoc(navPath string) ([]TOCEntry, error) {
	f, err := os.Open(navPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := gohtml.Parse(f)
	if err != nil {
		return nil, err
	}

	nav := selectTocNav(doc)
	if nav == nil {
		return nil, nil
	}
	ol := firstChildElement(nav, "ol")
	if ol == nil {
		return nil, nil
	}
	return parseNavList(ol), nil
}

// selectTocNav picks the table-of-contents <nav>, preferring epub:type="toc",
// then role="doc-toc", then the first <nav> in the document.
func selectTocNav(doc *gohtml.Node) *gohtml.Node {
	var navs []*gohtml.Node
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode && n.Data == "nav" {
			navs = append(navs, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	for _, n := range navs {
		if hasToken(elementAttr(n, "epub:type"), "toc") {
			return n
		}
	}
	for _, n := range navs {
		if hasToken(elementAttr(n, "role"), "doc-toc") {
			return n
		}
	}
	if len(navs) > 0 {
		return navs[0]
	}
	return nil
}

// parseNavList parses an <ol> of EPUB3 nav <li> entries (each: a label link
// and an optional nested <ol>).
func parseNavList(ol *gohtml.Node) []TOCEntry {
	var entries []TOCEntry
	for li := ol.FirstChild; li != nil; li = li.NextSibling {
		if li.Type != gohtml.ElementNode || li.Data != "li" {
			continue
		}
		if e, ok := parseNavItem(li); ok {
			entries = append(entries, e)
		}
	}
	return entries
}

// parseNavItem turns one <li> into a TOCEntry: the first <a>/<span> before any
// nested list provides the label and href; a nested <ol> provides children.
func parseNavItem(li *gohtml.Node) (TOCEntry, bool) {
	var e TOCEntry
	if a := firstAnchorBeforeList(li); a != nil {
		e.Title = strings.TrimSpace(textContent(a))
		e.Href = elementAttr(a, "href")
	} else if span := firstChildElement(li, "span"); span != nil {
		e.Title = strings.TrimSpace(textContent(span))
	}

	if nested := firstChildElement(li, "ol"); nested != nil {
		e.Children = parseNavList(nested)
	}

	if e.Title == "" && e.Href == "" && len(e.Children) == 0 {
		return TOCEntry{}, false
	}
	return e, true
}

// firstAnchorBeforeList finds the first <a> under li that is not inside a
// nested <ol>/<ul> (which belongs to child entries, not this one's label).
func firstAnchorBeforeList(li *gohtml.Node) *gohtml.Node {
	var found *gohtml.Node
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if found != nil {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != gohtml.ElementNode {
				continue
			}
			if c.Data == "ol" || c.Data == "ul" {
				continue // skip the children list
			}
			if c.Data == "a" {
				found = c
				return
			}
			walk(c)
			if found != nil {
				return
			}
		}
	}
	walk(li)
	return found
}

// ── EPUB2 toc.ncx ───────────────────────────────────────────────────

type ncxDoc struct {
	NavMap ncxNavMap `xml:"navMap"`
}

type ncxNavMap struct {
	Points []ncxNavPoint `xml:"navPoint"`
}

type ncxNavPoint struct {
	NavLabel ncxNavLabel   `xml:"navLabel"`
	Content  ncxContent    `xml:"content"`
	Points   []ncxNavPoint `xml:"navPoint"`
}

type ncxNavLabel struct {
	Text string `xml:"text"`
}

type ncxContent struct {
	Src string `xml:"src,attr"`
}

// parseNCXDoc reads an EPUB2 NCX and returns its navMap as raw TOC entries.
func parseNCXDoc(ncxPath string) ([]TOCEntry, error) {
	data, err := os.ReadFile(ncxPath)
	if err != nil {
		return nil, err
	}
	var doc ncxDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return ncxPointsToEntries(doc.NavMap.Points), nil
}

func ncxPointsToEntries(points []ncxNavPoint) []TOCEntry {
	var entries []TOCEntry
	for _, p := range points {
		entries = append(entries, TOCEntry{
			// collapseWS (not just TrimSpace) so NCX titles with internal newlines/runs
			// match the nav.xhtml path (textContent) and the extension, which collapses
			// whitespace in both TOC sources - see docs/PARITY.md ("EPUB TOC parsing").
			Title:    collapseWS(p.NavLabel.Text),
			Href:     strings.TrimSpace(p.Content.Src),
			Children: ncxPointsToEntries(p.Points),
		})
	}
	return entries
}

// ── Shared resolution ───────────────────────────────────────────────

// resolveTOC resolves every entry's href against the final manifest, dropping
// entries whose target no longer exists (unless they have surviving children,
// in which case they remain as a label-only grouping).
func resolveTOC(raw []TOCEntry, book *Book, tocHref string, manifest map[string]bool) []TOCEntry {
	tocDir := path.Dir(tocHref)
	var out []TOCEntry
	for _, e := range raw {
		children := resolveTOC(e.Children, book, tocHref, manifest)
		href, ok := resolveTOCHref(e.Href, tocDir, book, manifest)
		if !ok && len(children) == 0 {
			continue
		}
		out = append(out, TOCEntry{
			Title:    e.Title,
			Href:     href, // "" when the node only groups children
			Children: children,
		})
	}
	return out
}

// resolveTOCHref maps a raw NCX/nav src to a final manifest href (+fragment).
// External links pass through; internal targets are resolved relative to the
// TOC document's directory and must exist in the manifest.
func resolveTOCHref(raw, tocDir string, book *Book, manifest map[string]bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if isExternalHref(raw) {
		return raw, true
	}

	file, frag := splitFragment(raw)
	if file == "" {
		return "", false // pure #fragment into the TOC doc itself — not useful
	}

	resolved := path.Clean(path.Join(tocDir, file))

	final := book.resolveHrefChain(resolved)
	if !manifest[final] {
		final = book.resolveHrefChain(toHTMLExt(resolved))
	}
	if !manifest[final] {
		return "", false
	}
	if frag != "" {
		return final + "#" + frag, true
	}
	return final, true
}

func splitFragment(href string) (file, frag string) {
	if i := strings.IndexByte(href, '#'); i >= 0 {
		return href[:i], href[i+1:]
	}
	return href, ""
}

func isExternalHref(href string) bool {
	lower := strings.ToLower(href)
	return strings.Contains(lower, "://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(lower, "data:")
}

// ── x/net/html helpers ──────────────────────────────────────────────

// firstChildElement returns the first direct child element with the given tag.
func firstChildElement(n *gohtml.Node, tag string) *gohtml.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == gohtml.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

// elementAttr returns the value of attr `key` on n, matching both "epub:type"
// and the namespaced (Namespace="epub", Key="type") form the HTML parser may
// produce.
func elementAttr(n *gohtml.Node, key string) string {
	ns, local := "", key
	if i := strings.IndexByte(key, ':'); i >= 0 {
		ns, local = key[:i], key[i+1:]
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
		if ns != "" && a.Namespace == ns && a.Key == local {
			return a.Val
		}
	}
	return ""
}

// collapseWS trims and collapses internal whitespace runs to a single space. Matches
// textContent (the nav.xhtml path) and the extension, which collapses whitespace for
// both TOC sources, so NCX titles stay identical across implementations.
func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// hasToken reports whether the whitespace-separated value contains token.
func hasToken(value, token string) bool {
	for _, f := range strings.Fields(value) {
		if f == token {
			return true
		}
	}
	return false
}

// textContent concatenates the visible text under n, collapsing runs of
// whitespace to single spaces.
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
