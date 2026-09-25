// Package htmlconv handles HTML/HTM file conversion for pipeline compatibility.
// Copies the source HTML, wrapping it with our standard CSS layout if needed.
package htmlconv

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"doc-html-translate/internal/assets"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/logging"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// Extract reads an HTML/HTM file, wraps it if necessary, writes the output
// to outputDir, and returns an *epub.Book adapter with a single page. Local assets
// the page references (relative paths next to the source, the "Save page as" shape) are
// copied into the output so they still display; a reference that does not resolve is left
// in place as a visibly broken image rather than silently dropped. The source's own
// stylesheets are kept as files in the book's manifest, so the single-page merge links
// them too.
func Extract(htmlPath, outputDir string) (*epub.Book, error) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("open html: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("no content found: %s", htmlPath)
	}

	doc, err := parseDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	copier, err := assets.NewCopier(filepath.Dir(htmlPath), outputDir)
	if err != nil {
		return nil, err
	}

	title := findTitle(doc)
	if title == "" {
		title = fileTitle(htmlPath)
	}
	sheets := keptStylesheets(doc, copier)
	body := renderBody(doc, copier)

	href := "page_001.html"
	id := "page_001"
	outputHTML := wrapHTML(title, rootAttrs(doc), sheets, body)
	if err := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(outputHTML), 0o644); err != nil {
		return nil, fmt.Errorf("write html page: %w", err)
	}

	book := &epub.Book{
		Title:    title,
		BasePath: "",
		Manifest: []epub.ManifestItem{
			{ID: id, Href: href, MediaType: "text/html"},
		},
		Spine: []epub.SpineItem{
			{IDRef: id},
		},
	}
	for i, name := range sheets {
		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID: fmt.Sprintf("css_%03d", i+1), Href: name, MediaType: "text/css",
		})
	}

	logging.Printf("  Title: %s\n", title)
	if n := copier.Files(); n > 0 {
		logging.Printf("  Pages: 1, Images: %d\n", n)
	} else {
		logging.Printf("  Pages: 1\n")
	}
	if n := copier.Stylesheets(); n > 0 {
		logging.Printf("  Stylesheets: %d\n", n)
	}

	return book, nil
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// parseDocument decodes the page to UTF-8 - by BOM, then the <meta> charset declaration,
// then detection - and parses it. The output page is always UTF-8.
func parseDocument(data []byte) (*gohtml.Node, error) {
	enc, name, certain := charset.DetermineEncoding(data, "")
	// windows-1252 is both the WHATWG fallback and a common mislabel; bytes that are valid
	// UTF-8 throughout were almost certainly written as UTF-8 (and pure ASCII reads the same
	// either way). The detector only looks at the first 1024 bytes, so this checks them all.
	if !certain && name == "windows-1252" && utf8.Valid(data) {
		enc, name = encoding.Nop, "utf-8"
	}
	if enc == encoding.Nop || name == "utf-8" {
		return gohtml.Parse(bytes.NewReader(bytes.TrimPrefix(data, utf8BOM)))
	}
	logging.Printf("  Encoding: %s\n", name)
	return gohtml.Parse(transform.NewReader(bytes.NewReader(data), enc.NewDecoder()))
}

// findTitle walks the HTML parse tree to find the <title> element text.
func findTitle(n *gohtml.Node) string {
	if n.Type == gohtml.ElementNode && n.Data == "title" {
		return textContent(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTitle(c); t != "" {
			return t
		}
	}
	return ""
}

// textContent returns the concatenated text content of a node.
func textContent(n *gohtml.Node) string {
	if n.Type == gohtml.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return strings.TrimSpace(sb.String())
}

// renderBody copies the local assets the body references and renders its inner HTML.
func renderBody(doc *gohtml.Node, copier *assets.Copier) string {
	bodyNode := findNode(doc, "body")
	if bodyNode == nil {
		return ""
	}
	copier.RewriteHTML(bodyNode)
	var sb strings.Builder
	for c := bodyNode.FirstChild; c != nil; c = c.NextSibling {
		_ = gohtml.Render(&sb, c)
	}
	return sb.String()
}

// keptStylesheets turns the head's <style> blocks and local stylesheet links into output
// stylesheet files, in document order, and returns their names. Remote stylesheets are
// dropped: the output is read offline and a conversion never fetches.
func keptStylesheets(doc *gohtml.Node, copier *assets.Copier) []string {
	head := findNode(doc, "head")
	if head == nil {
		return nil
	}
	var names []string
	for n := head.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != gohtml.ElementNode {
			continue
		}
		media := attr(n, "media")
		var name string
		var ok bool
		switch {
		case n.Data == "style":
			name, ok = copier.InlineStylesheet(textContent(n), media)
		case n.Data == "link" && assets.IsStylesheetLink(n):
			name, ok = copier.LinkedStylesheet(attr(n, "href"), media)
		}
		if ok {
			names = append(names, name)
		}
	}
	return names
}

// rootAttrs returns the language and direction the source declares, as attributes for the
// output <html>. <html lang> is what Chrome reads to decide whether to offer translation, so
// it carries only a declared language; with none, the page declares none and the browser
// detects it. A declaration on <body> is lifted too, because only the body's content is kept.
func rootAttrs(doc *gohtml.Node) string {
	root := findNode(doc, "html")
	body := findNode(doc, "body")
	pick := func(keys ...string) string {
		for _, n := range []*gohtml.Node{root, body} {
			if n == nil {
				continue
			}
			for _, k := range keys {
				if v := strings.TrimSpace(attr(n, k)); v != "" {
					return v
				}
			}
		}
		return ""
	}
	var sb strings.Builder
	if lang := pick("lang", "xml:lang"); lang != "" {
		sb.WriteString(fmt.Sprintf(" lang=\"%s\"", html.EscapeString(lang)))
	}
	if dir := strings.ToLower(pick("dir")); dir == "rtl" || dir == "ltr" || dir == "auto" {
		sb.WriteString(fmt.Sprintf(" dir=\"%s\"", dir))
	}
	return sb.String()
}

// attr returns the named attribute of n, or "".
func attr(n *gohtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// findNode walks the parse tree to find the first element with the given tag.
func findNode(n *gohtml.Node, tag string) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// wrapHTML wraps body content into a full HTML page with our standard layout. The
// source's stylesheets follow the baseline style so they win over it; the reader CSS the
// navbar step injects later lands after both.
func wrapHTML(title, rootAttrs string, sheets []string, bodyContent string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html" + rootAttrs + ">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: -apple-system, 'Segoe UI', Helvetica, Arial, sans-serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    img { max-width: 100%; }\n")
	sb.WriteString("    table { border-collapse: collapse; width: 100%; }\n")
	sb.WriteString("    th, td { border: 1px solid #ddd; padding: 0.5em; text-align: left; }\n")
	sb.WriteString("  </style>\n")
	for _, name := range sheets {
		sb.WriteString(fmt.Sprintf("  <link rel=\"stylesheet\" href=\"%s\">\n", html.EscapeString(name)))
	}
	sb.WriteString("</head>\n<body>\n")
	sb.WriteString(bodyContent)
	sb.WriteString("\n</body>\n</html>\n")
	return sb.String()
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
