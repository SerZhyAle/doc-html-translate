// Package htmlgen generates index.html from parsed EPUB metadata.
package htmlgen

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"doc-html-translate/internal/epub"

	gohtml "golang.org/x/net/html"
)

// GenerateIndex creates index.html in outputDir with a table of contents
// based on the book's spine order. CSS files from the manifest are linked.
func GenerateIndex(book *epub.Book, outputDir string) (string, error) {
	indexPath := filepath.Join(outputDir, "index.html")

	// Collect CSS files from manifest
	var cssLinks []string
	for _, item := range book.Manifest {
		if item.MediaType == "text/css" {
			href := item.Href
			if book.BasePath != "" && book.BasePath != "." {
				href = book.BasePath + "/" + href
			}
			cssLinks = append(cssLinks, fmt.Sprintf(`    <link rel="stylesheet" href="%s">`, html.EscapeString(href)))
		}
	}

	// Build TOC from spine, with first-sentence previews from each page.
	spineHrefs := book.SpineHrefs()
	var tocEntries []string
	for i, href := range spineHrefs {
		fullHref := href
		if book.BasePath != "" && book.BasePath != "." {
			fullHref = book.BasePath + "/" + href
		}
		label := chapterLabel(href, i+1)

		// Extract snippet from the page file (translated if available).
		pagePath := filepath.Join(outputDir, filepath.FromSlash(href))
		snippet := extractSnippet(pagePath)

		if snippet != "" {
			// Use snippet as the primary label — filename (part0001 etc.) is not informative.
			tocEntries = append(tocEntries, fmt.Sprintf(
				`      <li><a href="%s"><span class="toc-label">%d.</span><span class="toc-snippet">%s</span></a></li>`,
				html.EscapeString(fullHref), i+1, html.EscapeString(snippet)))
		} else {
			tocEntries = append(tocEntries, fmt.Sprintf(
				`      <li><a href="%s"><span class="toc-label">%d. %s</span></a></li>`,
				html.EscapeString(fullHref), i+1, html.EscapeString(label)))
		}
	}

	title := book.Title
	if title == "" {
		title = "EPUB Book"
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"en\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	if len(cssLinks) > 0 {
		sb.WriteString(strings.Join(cssLinks, "\n") + "\n")
	}
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; }\n")
	sb.WriteString("    h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.3em; }\n")
	sb.WriteString("    nav ul { list-style: none; padding: 0; }\n")
	sb.WriteString("    nav li { margin: 0.5em 0; }\n")
	sb.WriteString("    nav a { text-decoration: none; color: #1a0dab; display: block; padding: 0.3em 0; }\n")
	sb.WriteString("    nav a:hover { text-decoration: underline; }\n")
	sb.WriteString("    .toc-label { font-weight: bold; margin-right: 0.4em; }\n")
	sb.WriteString("    .toc-snippet { font-size: 0.9em; color: #333; font-style: italic; }\n")
	sb.WriteString("    .meta { color: #666; font-size: 0.9em; margin-bottom: 2em; }\n")
	sb.WriteString("  </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf("  <p class=\"meta\">Chapters: %d</p>\n", len(spineHrefs)))
	sb.WriteString("  <nav>\n")
	sb.WriteString("    <ul>\n")
	if len(tocEntries) > 0 {
		sb.WriteString(strings.Join(tocEntries, "\n") + "\n")
	}
	sb.WriteString("    </ul>\n")
	sb.WriteString("  </nav>\n")
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")

	if err := os.WriteFile(indexPath, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write index.html: %w", err)
	}

	return indexPath, nil
}

// extractSnippet reads an HTML page and returns the first meaningful sentence
// (up to ~150 chars) from its visible text content. Returns "" on any error.
func extractSnippet(htmlPath string) string {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return ""
	}
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	// Collect visible text from <body>, skipping nav/script/style/structural elements.
	var buf strings.Builder
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			switch n.Data {
			case "script", "style", "head", "nav", "button":
				return // skip non-content nodes
			case "div", "header", "footer":
				// Skip our structural divs (page-header, dht-navbar, etc.)
				if nodeHasClass(n, "page-header") || nodeHasClass(n, "dht-navbar") ||
					nodeAttr(n, "id") == "dht-nav" || nodeAttr(n, "id") == "dht-zoom-sync" {
					return
				}
			}
		}
		if n.Type == gohtml.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if buf.Len() > 0 {
					buf.WriteByte(' ')
				}
				buf.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if buf.Len() > 300 {
				return // collected enough
			}
			walk(c)
		}
	}
	// Start from body if available.
	body := findBodyNode(doc)
	if body != nil {
		walk(body)
	} else {
		walk(doc)
	}

	text := strings.TrimSpace(buf.String())
	if text == "" {
		return ""
	}

	const maxLen = 250
	runes := []rune(text)

	// Find up to 2 sentence boundaries (., !, ?) within maxLen.
	limit := len(runes)
	if limit > maxLen {
		limit = maxLen
	}
	sentences := 0
	for i, r := range runes[:limit] {
		if r == '.' || r == '!' || r == '?' {
			sentences++
			if sentences == 2 {
				return strings.TrimRightFunc(string(runes[:i+1]), unicode.IsSpace)
			}
		}
	}
	// Fewer than 2 sentences found within limit — return what we have.
	if len(runes) > maxLen {
		return strings.TrimRightFunc(string(runes[:maxLen]), unicode.IsSpace) + "…"
	}
	return text
}

// findBodyNode walks the parse tree to find the <body> element.
func findBodyNode(n *gohtml.Node) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findBodyNode(c); found != nil {
			return found
		}
	}
	return nil
}

// nodeAttr returns the value of the named attribute on an element node, or "".
func nodeAttr(n *gohtml.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// nodeHasClass reports whether an element node has the given CSS class.
func nodeHasClass(n *gohtml.Node, class string) bool {
	for _, word := range strings.Fields(nodeAttr(n, "class")) {
		if word == class {
			return true
		}
	}
	return false
}

// GenerateSinglePageIndex creates an index.html that instantly redirects
// to the single content page. Used when the book fits on one page —
// no TOC or navigation bars are needed.
func GenerateSinglePageIndex(book *epub.Book, outputDir string) (string, error) {
	indexPath := filepath.Join(outputDir, "index.html")

	spineHrefs := book.SpineHrefs()
	if len(spineHrefs) == 0 {
		return "", fmt.Errorf("book has no spine entries")
	}

	target := spineHrefs[0]
	if book.BasePath != "" && book.BasePath != "." {
		target = book.BasePath + "/" + target
	}

	// JS redirect (instant, preserves browser history correctly vs meta-refresh)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <script>location.replace(%q);</script>
</head>
<body></body>
</html>
`, target)

	if err := os.WriteFile(indexPath, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("write single-page index: %w", err)
	}
	return indexPath, nil
}

// chapterLabel creates a human-readable label for a spine entry.
func chapterLabel(href string, index int) string {
	base := filepath.Base(href)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Clean up common naming patterns
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	if name == "" {
		return fmt.Sprintf("Chapter %d", index)
	}
	return fmt.Sprintf("%d. %s", index, name)
}
