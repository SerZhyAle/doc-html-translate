// Package md handles Markdown file conversion to HTML pages.
// Uses github.com/yuin/goldmark for Markdown → HTML rendering.
package md

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"

	"github.com/yuin/goldmark"
)

// sectionsPerPage controls how many H1/H2 sections go into one HTML page.
// If no headings found, the whole document becomes a single page.
const sectionsPerPage = 1

// Extract reads a Markdown file, converts it to HTML, generates per-page HTML
// files in outputDir, and returns an *epub.Book adapter for pipeline compatibility.
func Extract(mdPath, outputDir string) (*epub.Book, error) {
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("open markdown: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("no content found: %s", mdPath)
	}

	// Render full Markdown to HTML
	var htmlBuf bytes.Buffer
	if err := goldmark.Convert(data, &htmlBuf); err != nil {
		return nil, fmt.Errorf("render markdown: %w", err)
	}

	title := fileTitle(mdPath)
	body := htmlBuf.String()

	book := &epub.Book{
		Title:    title,
		BasePath: "",
	}

	// Split by <h1> or <h2> tags for pagination
	sections := splitBySections(body)
	if len(sections) == 0 {
		sections = []string{body}
	}

	totalPages := len(sections)
	for i, section := range sections {
		pageNum := i + 1
		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)

		pageHTML := wrapPageHTML(title, pageNum, totalPages, section)
		if err := os.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write page %d: %w", pageNum, err)
		}

		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID: id, Href: href, MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Sections: %d\n", totalPages)

	return book, nil
}

// splitBySections splits HTML content at <h1> or <h2> boundaries.
// Each section includes everything from one heading to the next.
// Content before the first heading becomes the first section.
func splitBySections(body string) []string {
	// Find split points at <h1 or <h2
	var sections []string
	lower := strings.ToLower(body)
	var indices []int
	for i := 0; i < len(lower); i++ {
		if i+3 < len(lower) && lower[i:i+3] == "<h1" || i+3 < len(lower) && lower[i:i+3] == "<h2" {
			indices = append(indices, i)
		}
	}

	if len(indices) == 0 {
		return []string{body}
	}

	// Content before first heading
	if indices[0] > 0 {
		pre := strings.TrimSpace(body[:indices[0]])
		if pre != "" {
			sections = append(sections, pre)
		}
	}

	for i, idx := range indices {
		end := len(body)
		if i+1 < len(indices) {
			end = indices[i+1]
		}
		sections = append(sections, body[idx:end])
	}

	return sections
}

// wrapPageHTML wraps rendered Markdown HTML into a full HTML page.
func wrapPageHTML(title string, pageNum, totalPages int, content string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(title), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: -apple-system, 'Segoe UI', Helvetica, Arial, sans-serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    .page-header { color: #666; font-size: 0.9em; border-bottom: 1px solid #eee; padding-bottom: 0.5em; margin-bottom: 1em; }\n")
	sb.WriteString("    pre { background: #f6f8fa; padding: 1em; overflow-x: auto; border-radius: 6px; }\n")
	sb.WriteString("    code { background: #f0f0f0; padding: 0.2em 0.4em; border-radius: 3px; font-size: 0.9em; }\n")
	sb.WriteString("    pre code { background: none; padding: 0; }\n")
	sb.WriteString("    blockquote { border-left: 4px solid #ddd; margin: 1em 0; padding: 0.5em 1em; color: #555; }\n")
	sb.WriteString("    table { border-collapse: collapse; width: 100%; }\n")
	sb.WriteString("    th, td { border: 1px solid #ddd; padding: 0.5em; text-align: left; }\n")
	sb.WriteString("    img { max-width: 100%; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(fmt.Sprintf("  <div class=\"page-header\">%s — %d / %d</div>\n",
		html.EscapeString(title), pageNum, totalPages))
	sb.WriteString(content)
	sb.WriteString("\n</body>\n</html>\n")
	return sb.String()
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
