// Package txt handles plain text file conversion to HTML pages.
// Paragraphs are detected by blank lines. Long files are split into pages.
package txt

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/textutil"
)

// Byte-order marks, in the order they must be tested. UTF-8's is three bytes and cannot be
// confused with the others; the two UTF-16 marks are each other's reverse, which is the whole
// point of them.
var (
	bomUTF8    = []byte{0xEF, 0xBB, 0xBF}
	bomUTF16LE = []byte{0xFF, 0xFE}
	bomUTF16BE = []byte{0xFE, 0xFF}
)

// decodeText turns a text file's raw bytes into a string, honouring the encoding those bytes
// declare instead of assuming every byte is a character.
//
// Reading the bytes as-is is what turned a Notepad "Unicode" save into mojibake: UTF-16 text
// is two bytes per character, so a Cyrillic file came out as its low bytes
// ("Это обычное" -> "-B> >1KG=>5"). ASCII-in-UTF-16 survived only by accident - it is exactly
// "every character followed by a NUL", and the single-page merge strips control characters on
// its way past - which is why the same file read correctly, or as spaced-out letters, or as
// mojibake, depending on the alphabet and on -multipage. The decode belongs here; the merge's
// NUL-stripping is a coincidence, not a fix.
//
// A BOM is authoritative when present. Without one, valid UTF-8 is taken at face value (the
// overwhelmingly common case, and free to check), and anything else is handed to
// decodeLegacy.
func decodeText(raw []byte) string {
	switch {
	case bytes.HasPrefix(raw, bomUTF8):
		// x/text would decode this too, but trimming says exactly what happens: the bytes
		// after the mark are already UTF-8. Left in place, the mark showed up as an
		// invisible character opening the first paragraph.
		return string(bytes.TrimPrefix(raw, bomUTF8))

	case bytes.HasPrefix(raw, bomUTF16LE), bytes.HasPrefix(raw, bomUTF16BE):
		// UseBOM reads the endianness off the mark and removes it, so one branch covers both
		// orders; the LittleEndian argument is only the fallback for BOM-less input, which
		// cannot reach here.
		out, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(raw)
		if err != nil {
			// Truncated or malformed UTF-16. Returning the raw bytes keeps the old
			// behaviour for a file we cannot honestly decode, rather than losing it.
			return string(raw)
		}
		return string(out)
	}

	if utf8.Valid(raw) {
		return string(raw)
	}
	return decodeLegacy(raw)
}

// decodeLegacy handles bytes that carry no BOM and are not valid UTF-8 - a text file saved in
// a pre-Unicode Cyrillic code page (Windows-1251, KOI8-R, CP866, ISO-8859-5). It commits to a
// code page only when the decode is confidently Russian; otherwise the bytes pass through
// unchanged, exactly as before, so a non-Cyrillic file is never forced into an alphabet.
// The detected encoding is logged, so a reader whose decades-old .txt suddenly reads correctly
// can see why.
func decodeLegacy(raw []byte) string {
	text, encName, ok := detectLegacy(raw)
	if !ok {
		return string(raw)
	}
	logging.Printf("  Decoded from %s\n", encName)
	return text
}

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
	normalized := textutil.NormalizeLineSeparators(decodeText(raw))

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
	sb.WriteString("    p { margin: 0.8em 0; text-indent: 1.5em; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	// Page number is shown in the injected navbar (top-right); no body header needed.
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
