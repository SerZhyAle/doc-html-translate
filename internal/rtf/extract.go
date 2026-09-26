// Package rtf handles RTF file conversion to HTML pages. The reader (parse.go) keeps RTF's
// group state, skips non-text destinations and decodes code-page bytes with the code page the
// document and its fonts declare.
package rtf

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/textutil"
)

// paragraphsPerPage controls page splitting.
const paragraphsPerPage = 30

// Extract reads an RTF file, strips control words, extracts plain-text
// paragraphs, generates per-page HTML files in outputDir, and returns
// an *epub.Book adapter.
func Extract(rtfPath, outputDir string) (*epub.Book, error) {
	data, err := os.ReadFile(rtfPath)
	if err != nil {
		return nil, fmt.Errorf("open rtf: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no content found: %s", rtfPath)
	}

	text := stripRTF(data)
	paragraphs := splitParagraphs(text)

	if len(paragraphs) == 0 {
		return nil, fmt.Errorf("no content found: %s", rtfPath)
	}

	title := fileTitle(rtfPath)

	book := &epub.Book{
		Title:    title,
		BasePath: "",
	}

	totalPages := (len(paragraphs) + paragraphsPerPage - 1) / paragraphsPerPage
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		start := (pageNum - 1) * paragraphsPerPage
		end := start + paragraphsPerPage
		if end > len(paragraphs) {
			end = len(paragraphs)
		}

		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)

		pageHTML := buildPageHTML(title, pageNum, totalPages, paragraphs[start:end])
		if err := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write page %d: %w", pageNum, err)
		}

		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID: id, Href: href, MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Paragraphs: %d, Pages: %d\n", len(paragraphs), totalPages)

	return book, nil
}

// splitParagraphs splits text by double newlines, trims, and filters empty lines.
func splitParagraphs(text string) []string {
	raw := strings.Split(textutil.NormalizeLineSeparators(text), "\n")
	var result []string
	var current strings.Builder

	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		} else {
			if current.Len() > 0 {
				current.WriteByte(' ')
			}
			current.WriteString(trimmed)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

// ── HTML builder ────────────────────────────────────────────────────

func buildPageHTML(title string, pageNum, totalPages int, paragraphs []string) string {
	var sb strings.Builder
	// No lang: the source declares none, and a guessed one can stop Chrome offering
	// "Translate page"; without it Chrome detects the language itself.
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(title), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    p { text-indent: 1.5em; margin: 0.5em 0; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	// Page number is shown in the injected navbar (top-right); no body header needed.

	for _, p := range paragraphs {
		sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(p)))
	}

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
