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

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"

	gohtml "golang.org/x/net/html"
)

// Extract reads an HTML/HTM file, wraps it if necessary, writes the output
// to outputDir, and returns an *epub.Book adapter with a single page.
func Extract(htmlPath, outputDir string) (*epub.Book, error) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("open html: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("no content found: %s", htmlPath)
	}

	title := extractTitle(data, htmlPath)
	body := extractBody(data)

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
	logging.Printf("  Pages: 1\n")

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

// extractBody tries to extract the <body> inner HTML.
// If no <body> found, returns the whole document as-is.
func extractBody(data []byte) string {
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return string(data)
	}
	bodyNode := findNode(doc, "body")
	if bodyNode == nil {
		return string(data)
	}
	var sb strings.Builder
	for c := bodyNode.FirstChild; c != nil; c = c.NextSibling {
		gohtml.Render(&sb, c)
	}
	return sb.String()
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
