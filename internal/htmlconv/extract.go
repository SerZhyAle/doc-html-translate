// Package htmlconv handles HTML/HTM file conversion for pipeline compatibility.
// Copies the source HTML, wrapping it with our standard CSS layout if needed.
package htmlconv

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"

	gohtml "golang.org/x/net/html"
)

// Extract reads an HTML/HTM file, wraps it if necessary, writes the output
// to outputDir, and returns an *epub.Book adapter with a single page. Local images
// the page references (relative paths next to the source, the "Save page as" shape)
// are copied into the output so they still display; a reference that does not resolve
// is left in place as a visibly broken image rather than silently dropped.
func Extract(htmlPath, outputDir string) (*epub.Book, error) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("open html: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("no content found: %s", htmlPath)
	}

	title := extractTitle(data, htmlPath)
	body, imgCount := extractBody(data, filepath.Dir(htmlPath), outputDir)

	outputHTML := wrapHTML(title, body)
	href := "page_001.html"
	id := "page_001"

	if err := os.WriteFile(filepath.Join(outputDir, href), []byte(outputHTML), 0o644); err != nil {
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

	logging.Printf("  Title: %s\n", title)
	if imgCount > 0 {
		logging.Printf("  Pages: 1, Images: %d\n", imgCount)
	} else {
		logging.Printf("  Pages: 1\n")
	}

	return book, nil
}

// extractTitle tries to find <title> in the HTML document.
// Falls back to filename.
func extractTitle(data []byte, path string) string {
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err == nil {
		if t := findTitle(doc); t != "" {
			return t
		}
	}
	return fileTitle(path)
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

// extractBody extracts the <body> inner HTML, copying every local image the page
// references from srcDir into outputDir and rewriting its src to the copied name, so
// the pictures still display from the output folder. Returns the rendered body and the
// number of images copied. If no <body> is found, returns the whole document as-is.
func extractBody(data []byte, srcDir, outputDir string) (string, int) {
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return string(data), 0
	}
	bodyNode := findNode(doc, "body")
	if bodyNode == nil {
		return string(data), 0
	}

	n := copyLocalImages(bodyNode, srcDir, outputDir)

	var sb strings.Builder
	for c := bodyNode.FirstChild; c != nil; c = c.NextSibling {
		_ = gohtml.Render(&sb, c)
	}
	return sb.String(), n
}

// copyLocalImages walks the tree, copies each local image referenced by an <img src>
// into outputDir, and rewrites the src to the copied filename. Returns the number of
// distinct files copied.
//
// Only images that resolve to a real file inside the source's own directory subtree are
// copied: remote (http/https/protocol-relative) and inline (data:) srcs are left alone, an
// absolute or "../" path that escapes the source directory is refused (so a page cannot copy
// arbitrary files off the disk into the output), and a reference that does not resolve is left
// untouched to show as a visibly broken image rather than vanish. The same source file
// referenced twice is copied once (deduped by resolved path); distinct files whose basenames
// collide get a numeric suffix.
func copyLocalImages(root *gohtml.Node, srcDir, outputDir string) int {
	absSrcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return 0
	}
	copied := make(map[string]string) // resolved source path -> output filename
	used := make(map[string]bool)     // output filenames already taken

	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode && n.Data == "img" {
			if i := attrIndex(n, "src"); i >= 0 {
				if name, ok := resolveAndCopy(n.Attr[i].Val, absSrcDir, outputDir, copied, used); ok {
					n.Attr[i].Val = name
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return len(copied)
}

// resolveAndCopy resolves one src to a file inside absSrcDir's subtree and copies it into
// outputDir, returning the output filename. ok=false means "leave the src unchanged": a
// remote/inline reference, an absolute or escaping path, or a file that does not exist.
func resolveAndCopy(src, absSrcDir, outputDir string, copied map[string]string, used map[string]bool) (string, bool) {
	s := strings.TrimSpace(src)
	if s == "" {
		return "", false
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "data:") || strings.HasPrefix(low, "http://") ||
		strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "//") ||
		strings.HasPrefix(low, "file:") {
		return "", false // remote or inline: not ours to copy
	}
	// Drop any query string; percent-decode the path (a saved page writes "img%20a.png").
	if q := strings.IndexAny(s, "?#"); q >= 0 {
		s = s[:q]
	}
	if dec, err := url.PathUnescape(s); err == nil {
		s = dec
	}
	if filepath.IsAbs(s) {
		return "", false // absolute local path: outside the source subtree by policy
	}

	candidate := filepath.Join(absSrcDir, filepath.FromSlash(s))
	rel, err := filepath.Rel(absSrcDir, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false // "../.." escaping the source directory subtree
	}
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", false // dangling reference: leave it visibly broken
	}

	if name, ok := copied[candidate]; ok {
		return name, true // same file already copied
	}
	name := uniqueName(filepath.Base(candidate), used)
	if err := copyFileTo(candidate, filepath.Join(outputDir, name)); err != nil {
		return "", false
	}
	copied[candidate] = name
	used[name] = true
	return name, true
}

// uniqueName returns base, or base with a numeric suffix, so distinct source files whose
// basenames collide do not overwrite each other in the flat output directory.
func uniqueName(base string, used map[string]bool) string {
	base = sanitizeName(base)
	if !used[base] {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s_%d%s", stem, i, ext)
		if !used[cand] {
			return cand
		}
	}
}

// sanitizeName keeps a basename safe for the flat output directory.
func sanitizeName(base string) string {
	var sb strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	if sb.Len() == 0 {
		return "image"
	}
	return sb.String()
}

// attrIndex returns the index of the named attribute in n.Attr, or -1.
func attrIndex(n *gohtml.Node, key string) int {
	for i, a := range n.Attr {
		if a.Key == key {
			return i
		}
	}
	return -1
}

// copyFileTo copies src to dst byte-for-byte.
func copyFileTo(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
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

// wrapHTML wraps body content into a full HTML page with our standard layout.
func wrapHTML(title, bodyContent string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: -apple-system, 'Segoe UI', Helvetica, Arial, sans-serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    img { max-width: 100%; }\n")
	sb.WriteString("    table { border-collapse: collapse; width: 100%; }\n")
	sb.WriteString("    th, td { border: 1px solid #ddd; padding: 0.5em; text-align: left; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(bodyContent)
	sb.WriteString("\n</body>\n</html>\n")
	return sb.String()
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
