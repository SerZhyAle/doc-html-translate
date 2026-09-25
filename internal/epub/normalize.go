package epub

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/logging"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// contentRename is the planned fate of one HTML manifest item: htmlHref is
// the href after the XHTML -> HTML rename, finalHref after the reserved-name
// rename as well. Both equal the original href when nothing applies.
type contentRename struct {
	htmlHref  string
	finalHref string
}

// normalizeContent prepares every HTML content file for the browser in one
// parse and one render per file, working on the DOM so no step can alter book
// text (ADR-1 of bugfix-epub-html-content-fidelity):
//   - XHTML is re-serialized as HTML and written under a .html name, because
//     browser translators (notably Chrome Translate) often fail on local XHTML;
//   - any declared non-UTF-8 encoding is transcoded to UTF-8;
//   - single-image SVG cover wrappers become <img> (see rewriteCoverSVGs);
//   - a file whose final path collides with the generated nav (index.html) is
//     renamed to _content_<name>;
//   - in-book links follow both renames, via rewriteLinks.
//
// Renames are planned for the whole book first, so every file's links can be
// rewritten in its single pass. Best-effort: a file that cannot be processed
// is logged and keeps its original href.
func normalizeContent(book *Book, outputDir string) {
	plan, linkMap := planContentRenames(book, outputDir)
	lookup := func(target string) (string, bool) {
		r, ok := linkMap[target]
		return r, ok
	}
	navPath := filepath.Clean(filepath.Join(outputDir, "index.html"))

	for i := range book.Manifest {
		item := &book.Manifest[i]
		if !isHTMLMediaType(item.MediaType) {
			continue
		}
		r := plan[i]
		srcPath := bookPath(outputDir, book.BasePath, item.Href)
		dstPath := bookPath(outputDir, book.BasePath, r.finalHref)
		xhtml := item.MediaType == "application/xhtml+xml" || r.htmlHref != item.Href
		if err := normalizeChapter(srcPath, dstPath, item.Href, xhtml, lookup); err != nil {
			logging.Errorf("WARNING: normalize %s: %v\n", item.Href, err)
			continue
		}
		// The reserved name is a move, not a copy: the generated nav takes that path.
		if samePathFold(srcPath, navPath) && r.finalHref != item.Href {
			if err := os.Remove(srcPath); err != nil {
				logging.Errorf("WARNING: remove renamed %s: %v\n", item.Href, err)
			}
		}
		book.recordHrefRewrite(item.Href, r.htmlHref)
		book.recordHrefRewrite(r.htmlHref, r.finalHref)
		if r.htmlHref != item.Href {
			item.MediaType = "text/html"
		}
		item.Href = r.finalHref
	}
}

// planContentRenames computes each HTML manifest item's renames (indexed like
// book.Manifest) and the link map rewriteLinks consults: cleaned, decoded
// original path -> decoded final path. Items whose file is missing are not
// renamed, so no link is redirected to a file that will never be written.
func planContentRenames(book *Book, outputDir string) ([]contentRename, map[string]string) {
	navPath := filepath.Clean(filepath.Join(outputDir, "index.html"))
	plan := make([]contentRename, len(book.Manifest))
	linkMap := make(map[string]string)
	for i, item := range book.Manifest {
		plan[i] = contentRename{htmlHref: item.Href, finalHref: item.Href}
		if !isHTMLMediaType(item.MediaType) {
			continue
		}
		if st, err := os.Stat(bookPath(outputDir, book.BasePath, item.Href)); err != nil || !st.Mode().IsRegular() {
			continue
		}
		htmlHref := toHTMLExt(item.Href)
		finalHref := htmlHref
		// Case-insensitive: on NTFS Index.html *is* index.html, and the nav would overwrite it.
		if samePathFold(bookPath(outputDir, book.BasePath, htmlHref), navPath) {
			finalHref = path.Join(path.Dir(htmlHref), "_content_"+path.Base(htmlHref))
		}
		plan[i] = contentRename{htmlHref: htmlHref, finalHref: finalHref}
		if finalHref != item.Href {
			linkMap[cleanHrefPath(item.Href)] = cleanHrefPath(finalHref)
		}
	}
	return plan, linkMap
}

// cleanHrefPath returns a manifest href as the cleaned path that rewriteLinks
// hands to its callback. The href is already decoded (resolveBookPath), so it
// must not be decoded again: a book file literally named "a%20b" stays that.
func cleanHrefPath(href string) string {
	return path.Clean(href)
}

// samePathFold compares two file paths the way Windows does, ignoring case.
func samePathFold(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// normalizeChapter reads srcPath, applies the DOM normalizations and writes
// the result to dstPath. fileHref is the chapter's current OPF-relative href,
// against which its links resolve (renames never change the directory). A
// file that needs no change is left untouched on disk.
func normalizeChapter(srcPath, dstPath, fileHref string, xhtml bool, lookup func(string) (string, bool)) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	data, recoded, err := decodeToUTF8(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if xhtml {
		data = xhtmlToHTMLSyntax(data)
	}
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	changed := xhtml || recoded || filepath.Clean(srcPath) != filepath.Clean(dstPath)
	if rewriteCoverSVGs(doc) {
		changed = true
	}
	if rewriteLinks(doc, fileHref, lookup) {
		changed = true
	}
	if !changed {
		return nil
	}
	mapXMLLang(doc)
	ensureUTF8Meta(doc)

	var buf bytes.Buffer
	buf.Grow(len(data) + len(data)/8)
	if err := gohtml.Render(&buf, doc); err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFile(dstPath, buf.Bytes(), 0o644)
}

// htmlVoidElements are the HTML elements that never have an end tag, so an
// XML self-closing form of them already means the same thing in HTML.
var htmlVoidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "keygen": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// xhtmlToHTMLSyntax rewrites XHTML markup so the HTML parser reads it as an
// XML parser would. An HTML parser ignores the "/" of <a id="x"/> or
// <script src=".."/>, leaving the element open so it swallows the rest of the
// chapter; such non-void elements are expanded to a start and an end tag.
// Inside <svg> and <math> the HTML parser honours self-closing tags, so
// foreign content passes through, as does every other token byte-for-byte.
// The XML declaration is dropped: HTML would keep it as a bogus comment.
func xhtmlToHTMLSyntax(data []byte) []byte {
	z := gohtml.NewTokenizer(bytes.NewReader(data))
	var out bytes.Buffer
	out.Grow(len(data) + len(data)/16)
	foreign := 0
	for {
		tt := z.Next()
		switch tt {
		case gohtml.ErrorToken:
			return out.Bytes()
		case gohtml.CommentToken:
			if isXMLDeclaration(z.Raw()) {
				continue
			}
		case gohtml.StartTagToken:
			if name, _ := z.TagName(); isForeignRoot(name) {
				foreign++
			}
		case gohtml.EndTagToken:
			if name, _ := z.TagName(); isForeignRoot(name) && foreign > 0 {
				foreign--
			}
		case gohtml.SelfClosingTagToken:
			if foreign > 0 {
				break
			}
			raw := z.Raw()
			name, _ := z.TagName()
			if htmlVoidElements[string(name)] {
				break
			}
			// The tokenizer switched to raw-text mode for <script/>, <title/>,
			// <style/> and the like; the element is empty, so undo that.
			z.NextIsNotRawText()
			start := bytes.TrimRight(bytes.TrimSuffix(raw, []byte("/>")), " \t\r\n\f")
			out.Write(start)
			out.WriteString("></")
			out.Write(name)
			out.WriteByte('>')
			continue
		}
		out.Write(z.Raw())
	}
}

func isForeignRoot(name []byte) bool {
	return string(name) == "svg" || string(name) == "math"
}

func isXMLDeclaration(raw []byte) bool {
	if !bytes.HasPrefix(raw, []byte("<?xml")) || len(raw) < 6 {
		return false
	}
	return isHTMLSpace(raw[5])
}

// mapXMLLang copies xml:lang to lang on <html> when lang is absent: HTML
// ignores xml:lang, and Chrome's translate offer relies on lang.
func mapXMLLang(doc *gohtml.Node) {
	root := findElement(doc, atom.Html)
	if root == nil {
		return
	}
	xmlLang := ""
	for _, a := range root.Attr {
		if a.Namespace == "" && a.Key == "lang" {
			return
		}
		if (a.Namespace == "" && a.Key == "xml:lang") || (a.Namespace == "xml" && a.Key == "lang") {
			xmlLang = a.Val
		}
	}
	if xmlLang != "" {
		root.Attr = append(root.Attr, gohtml.Attribute{Key: "lang", Val: xmlLang})
	}
}

// ensureUTF8Meta replaces any charset declaration in <head> with
// <meta charset="utf-8">, matching the bytes Render writes.
func ensureUTF8Meta(doc *gohtml.Node) {
	head := findElement(doc, atom.Head)
	if head == nil {
		return
	}
	for c := head.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == gohtml.ElementNode && c.DataAtom == atom.Meta && declaresCharset(c) {
			head.RemoveChild(c)
		}
		c = next
	}
	meta := &gohtml.Node{
		Type:     gohtml.ElementNode,
		Data:     "meta",
		DataAtom: atom.Meta,
		Attr:     []gohtml.Attribute{{Key: "charset", Val: "utf-8"}},
	}
	head.InsertBefore(meta, head.FirstChild)
}

func declaresCharset(meta *gohtml.Node) bool {
	for _, a := range meta.Attr {
		if a.Key == "charset" {
			return true
		}
		if a.Key == "http-equiv" && strings.EqualFold(strings.TrimSpace(a.Val), "content-type") {
			return true
		}
	}
	return false
}

// findElement returns the first HTML element with the given atom, depth-first.
func findElement(n *gohtml.Node, a atom.Atom) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.DataAtom == a && n.Namespace == "" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findElement(c, a); f != nil {
			return f
		}
	}
	return nil
}

// rewriteCoverSVGs replaces single-image SVG wrappers with a plain <img>.
//
// EPUB tools commonly wrap the cover in <svg><image ../></svg>. That breaks two
// things in the generated output: the injected `img { object-fit: contain }` CSS
// and aspect-ratio JS guard never touch SVG <image>, and Calibre often emits
// preserveAspectRatio="none" - together this stretches the cover to fill the
// viewport. Browsers also do not apply EXIF orientation to SVG <image>, so a
// cover tagged Orientation=3 shows upside down. Converting to <img> fixes both:
// sizing falls under the injected CSS and the browser honours EXIF orientation.
//
// Only an outermost <svg> whose sole drawing content is exactly one <image> is
// replaced, and the replacement is that one element, so text or other markup
// around or between drawings can never be lost. SVGs with text, several
// images or any other shape are genuine illustrations and stay.
func rewriteCoverSVGs(doc *gohtml.Node) bool {
	var svgs []*gohtml.Node
	var collect func(*gohtml.Node)
	collect = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode && n.Namespace == "svg" && n.Data == "svg" {
			svgs = append(svgs, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(doc)

	changed := false
	for _, svg := range svgs {
		href, ok := singleImageHref(svg)
		if !ok || svg.Parent == nil {
			continue
		}
		img := &gohtml.Node{
			Type:     gohtml.ElementNode,
			Data:     "img",
			DataAtom: atom.Img,
			Attr:     []gohtml.Attribute{{Key: "src", Val: href}, {Key: "alt", Val: ""}},
		}
		svg.Parent.InsertBefore(img, svg)
		svg.Parent.RemoveChild(svg)
		changed = true
	}
	return changed
}

// singleImageHref returns the href of the one child <image> of svg when that
// is all the SVG draws. Whitespace, comments and the non-rendered title, desc,
// defs and metadata elements are ignored; anything else disqualifies it.
func singleImageHref(svg *gohtml.Node) (string, bool) {
	href, images := "", 0
	for c := svg.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case gohtml.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				return "", false
			}
		case gohtml.ElementNode:
			switch c.Data {
			case "title", "desc", "defs", "metadata":
			case "image":
				images++
				href = svgImageHref(c)
			default:
				return "", false
			}
		}
	}
	if images != 1 || href == "" {
		return "", false
	}
	return href, true
}

// svgImageHref prefers xlink:href, the form EPUB 2 tools emit, over SVG 2 href.
func svgImageHref(image *gohtml.Node) string {
	plain := ""
	for _, a := range image.Attr {
		if a.Key != "href" {
			continue
		}
		if a.Namespace == "xlink" {
			return strings.TrimSpace(a.Val)
		}
		if a.Namespace == "" {
			plain = strings.TrimSpace(a.Val)
		}
	}
	return plain
}
