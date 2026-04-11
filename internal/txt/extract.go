// Package txt handles plain text file conversion to HTML pages.
// Paragraphs are detected by blank lines. Long files are split into pages.
package txt

import (
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/textutil"
)

// paragraphsPerPage controls how many paragraphs go into one HTML page.
const paragraphsPerPage = 30

// Extract reads a plain text file, generates per-page HTML files in outputDir,
// and returns an *epub.Book adapter for pipeline compatibility.
func Extract(txtPath, outputDir string) (*epub.Book, error) {
	f, err := os.Open(txtPath)
	if err != nil {
		return nil, fmt.Errorf("open txt: %w", err)
	}
	defer f.Close()

	paragraphs := parseParagraphs(f)
	if len(paragraphs) == 0 {
		return nil, fmt.Errorf("no text content found: %s", txtPath)
	}

	title := txtTitle(txtPath)
	book := &epub.Book{
		Title:    title,
		BasePath: "",
	}

	totalPages := (len(paragraphs) + paragraphsPerPage - 1) / paragraphsPerPage
	for pageIdx := 0; pageIdx < totalPages; pageIdx++ {
		start := pageIdx * paragraphsPerPage
		end := start + paragraphsPerPage
		if end > len(paragraphs) {
			end = len(paragraphs)
		}

		pageNum := pageIdx + 1
		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)

		pageHTML := buildPageHTML(title, pageNum, totalPages, paragraphs[start:end])
		if err := os.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write page %d: %w", pageNum, err)
		}

		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID:        id,
			Href:      href,
			MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Paragraphs: %d \u2192 Pages: %d\n", len(paragraphs), totalPages)

	return book, nil
}

// parseParagraphs splits an io.Reader into paragraphs.
// Strategy:
//   - Normalize line endings to \n (handles \r\n and \r).
//   - If the text contains blank lines, use them as paragraph separators
//     (consecutive non-blank lines are joined with a space).
//   - If there are NO blank lines (typical for Linux single-\n files),
//     treat each non-empty line as its own paragraph.
func parseParagraphs(r io.Reader) []string {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	normalized := textutil.NormalizeLineSeparators(string(raw))

	hasBlankLine := strings.Contains(normalized, "\n\n")

	lines := strings.Split(normalized, "\n")

	if hasBlankLine {
		return parseByBlankLines(lines)
	}
	return parseByLines(lines)
}

// parseByBlankLines groups consecutive non-blank lines into paragraphs,
// splitting on blank lines.
func parseByBlankLines(lines []string) []string {
	var paragraphs []string
	var current strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if current.Len() > 0 {
				paragraphs = append(paragraphs, current.String())
				current.Reset()
			}
		} else {
			if current.Len() > 0 {
				current.WriteByte(' ')
			}
			current.WriteString(strings.TrimSpace(line))
		}
	}
	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}
	return paragraphs
}

// parseByLines treats each non-empty line as its own paragraph.
// Used for files with single-\n line separators (no blank lines).
func parseByLines(lines []string) []string {
	var paragraphs []string
	for _, line := range lines {
		if t := strings.TrimSpace(line); t != "" {
			paragraphs = append(paragraphs, t)
		}
	}
	return paragraphs
}

// buildPageHTML generates an HTML page from a slice of paragraphs.
func buildPageHTML(title string, pageNum, totalPages int, paragraphs []string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(title), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    .page-header { color: #666; font-size: 0.9em; border-bottom: 1px solid #eee; padding-bottom: 0.5em; margin-bottom: 1em; }\n")
	sb.WriteString("    p { margin: 0.8em 0; text-indent: 1.5em; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(fmt.Sprintf("  <div class=\"page-header\">%s — %d / %d</div>\n",
		html.EscapeString(title), pageNum, totalPages))
	for _, para := range paragraphs {
		sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(para)))
	}
	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

// txtTitle extracts a human-readable title from the file path (filename without extension).
func txtTitle(txtPath string) string {
	base := filepath.Base(txtPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
