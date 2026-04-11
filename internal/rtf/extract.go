// Package rtf handles RTF file conversion to HTML pages.
// Uses a lightweight parser that strips RTF control words and decodes
// \'XX hex escapes as Windows-1251 (common for Russian RTF documents).
package rtf

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/textutil"

	"golang.org/x/text/encoding/charmap"
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

	text := stripRTF(string(data))
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

// ── RTF stripper ────────────────────────────────────────────────────

// stripRTF removes RTF control words, braces, and decodes hex escapes.
func stripRTF(input string) string {
	var sb strings.Builder
	i := 0
	depth := 0

	for i < len(input) {
		ch := input[i]

		switch {
		case ch == '{':
			depth++
			i++
		case ch == '}':
			depth--
			if depth < 0 {
				depth = 0
			}
			i++
		case ch == '\\':
			i++
			if i >= len(input) {
				break
			}
			next := input[i]
			switch {
			case next == '\\':
				sb.WriteByte('\\')
				i++
			case next == '{':
				sb.WriteByte('{')
				i++
			case next == '}':
				sb.WriteByte('}')
				i++
			case next == '\'':
				// Hex escape: \'XX
				if i+2 < len(input) {
					hexStr := input[i+1 : i+3]
					b := hexToByte(hexStr)
					decoded := decodeCP1251(b)
					sb.WriteString(decoded)
					i += 3
				} else {
					i++
				}
			case next == '\n' || next == '\r':
				sb.WriteString("\n\n") // \<newline> is a paragraph break
				i++
			default:
				// Control word: \word followed by optional numeric parameter
				word, newI := readControlWord(input, i)
				i = newI
				switch word {
				case "par", "line":
					sb.WriteString("\n\n")
				case "tab":
					sb.WriteByte('\t')
				case "u":
					// \uN — Unicode escape: read decimal N then skip one char
					num, nextI := readNumber(input, i)
					i = nextI
					if num >= 0 && num <= 0x10FFFF {
						sb.WriteRune(rune(num))
					}
					// Skip the replacement character (if present)
					if i < len(input) && input[i] != '\\' && input[i] != '{' && input[i] != '}' {
						i++
					}
				}
			}
		default:
			sb.WriteByte(ch)
			i++
		}
	}

	return sb.String()
}

// readControlWord reads an RTF control word starting at position i,
// returns the word (without \) and the new position.
func readControlWord(input string, i int) (string, int) {
	start := i
	// Read alphabetic characters
	for i < len(input) && ((input[i] >= 'a' && input[i] <= 'z') || (input[i] >= 'A' && input[i] <= 'Z')) {
		i++
	}
	word := input[start:i]
	// Skip optional numeric parameter
	for i < len(input) && ((input[i] >= '0' && input[i] <= '9') || input[i] == '-') {
		i++
	}
	// Skip one trailing space (delimiter)
	if i < len(input) && input[i] == ' ' {
		i++
	}
	return word, i
}

// readNumber reads a decimal number (possibly negative) from position i.
func readNumber(input string, i int) (int, int) {
	neg := false
	if i < len(input) && input[i] == '-' {
		neg = true
		i++
	}
	num := 0
	found := false
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		num = num*10 + int(input[i]-'0')
		found = true
		i++
	}
	if !found {
		return 0, i
	}
	if neg {
		num = -num
	}
	// Skip optional space delimiter
	if i < len(input) && input[i] == ' ' {
		i++
	}
	return num, i
}

func hexToByte(s string) byte {
	var b byte
	for _, c := range s {
		b <<= 4
		switch {
		case c >= '0' && c <= '9':
			b |= byte(c - '0')
		case c >= 'a' && c <= 'f':
			b |= byte(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			b |= byte(c - 'A' + 10)
		}
	}
	return b
}

// decodeCP1251 attempts to decode a byte as Windows-1251.
// If it's valid UTF-8, returns as-is.
func decodeCP1251(b byte) string {
	if b < 0x80 {
		return string(rune(b))
	}
	decoded, _ := charmap.Windows1251.NewDecoder().Bytes([]byte{b})
	if utf8.Valid(decoded) {
		return string(decoded)
	}
	return string(rune(b))
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
