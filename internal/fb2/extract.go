// Package fb2 handles FictionBook (FB2) XML conversion to HTML pages.
// Uses stdlib encoding/xml for parsing.
package fb2

import (
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
)

// paragraphsPerPage controls page splitting.
const paragraphsPerPage = 30

// ── FB2 XML structures ──────────────────────────────────────────────

type fb2File struct {
	Description fb2Desc   `xml:"description"`
	Bodies      []fb2Body `xml:"body"`
}

type fb2Desc struct {
	TitleInfo fb2TitleInfo `xml:"title-info"`
}

type fb2TitleInfo struct {
	BookTitle string   `xml:"book-title"`
	Authors   []author `xml:"author"`
}

type author struct {
	FirstName  string `xml:"first-name"`
	MiddleName string `xml:"middle-name"`
	LastName   string `xml:"last-name"`
}

type fb2Body struct {
	Sections []fb2Section `xml:"section"`
	Title    *fb2Title    `xml:"title"`
}

type fb2Section struct {
	Title      *fb2Title    `xml:"title"`
	Paragraphs []string     `xml:"p"`
	Epigraphs  []fb2Epigraph `xml:"epigraph"`
	Sections   []fb2Section `xml:"section"`
}

type fb2Title struct {
	Paragraphs []string `xml:"p"`
}

type fb2Epigraph struct {
	Paragraphs []string `xml:"p"`
}

// ── Public API ──────────────────────────────────────────────────────

// Extract reads an FB2 file, parses its XML, generates per-page HTML
// files in outputDir, and returns an *epub.Book adapter.
func Extract(fb2Path, outputDir string) (*epub.Book, error) {
	data, err := os.ReadFile(fb2Path)
	if err != nil {
		return nil, fmt.Errorf("open fb2: %w", err)
	}

	var doc fb2File
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse fb2 xml: %w", err)
	}

	title := strings.TrimSpace(doc.Description.TitleInfo.BookTitle)
	if title == "" {
		title = fileTitle(fb2Path)
	}

	paragraphs := collectParagraphs(&doc)
	if len(paragraphs) == 0 {
		return nil, fmt.Errorf("no content found: %s", fb2Path)
	}

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
		if err := os.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
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

// ── Internal helpers ────────────────────────────────────────────────

// collectParagraphs extracts all text paragraphs from all <body>/<section> elements.
func collectParagraphs(doc *fb2File) []string {
	var result []string
	for _, body := range doc.Bodies {
		if body.Title != nil {
			for _, p := range body.Title.Paragraphs {
				t := strings.TrimSpace(p)
				if t != "" {
					result = append(result, t)
				}
			}
		}
		for _, sec := range body.Sections {
			result = append(result, collectFromSection(&sec)...)
		}
	}
	return result
}

func collectFromSection(sec *fb2Section) []string {
	var result []string
	if sec.Title != nil {
		for _, p := range sec.Title.Paragraphs {
			t := strings.TrimSpace(p)
			if t != "" {
				result = append(result, t)
			}
		}
	}
	for _, ep := range sec.Epigraphs {
		for _, p := range ep.Paragraphs {
			t := strings.TrimSpace(p)
			if t != "" {
				result = append(result, t)
			}
		}
	}
	for _, p := range sec.Paragraphs {
		t := strings.TrimSpace(p)
		if t != "" {
			result = append(result, t)
		}
	}
	for _, sub := range sec.Sections {
		result = append(result, collectFromSection(&sub)...)
	}
	return result
}

func buildPageHTML(title string, pageNum, totalPages int, paragraphs []string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(title), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    .page-header { color: #666; font-size: 0.9em; border-bottom: 1px solid #eee; padding-bottom: 0.5em; margin-bottom: 1em; }\n")
	sb.WriteString("    p { text-indent: 1.5em; margin: 0.5em 0; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(fmt.Sprintf("  <div class=\"page-header\">%s — %d / %d</div>\n",
		html.EscapeString(title), pageNum, totalPages))

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
