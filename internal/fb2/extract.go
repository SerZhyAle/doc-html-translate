// Package fb2 handles FictionBook (FB2) XML conversion to HTML pages.
// Uses stdlib encoding/xml for parsing; content.go holds the single-pass reader.
package fb2

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"html"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/limits"
	"doc-html-translate/internal/logging"
)

// paragraphsPerPage controls page splitting.
const paragraphsPerPage = 30

// Extract reads an FB2 file, parses its XML, generates per-page HTML files in
// outputDir, and returns an *epub.Book adapter. Embedded images (<binary>) that
// the body references (<image>) are decoded to sibling files and shown in place;
// a reference with no matching binary degrades to a visible note, not a silent gap.
func Extract(fb2Path, outputDir string) (*epub.Book, error) {
	raw, err := limits.ReadTextInput(fb2Path)
	if err != nil {
		return nil, fmt.Errorf("open fb2: %w", err)
	}
	doc, err := parseFB2(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse fb2 xml: %w", err)
	}

	title := strings.TrimSpace(doc.title)
	if title == "" {
		title = fileTitle(fb2Path)
	}
	if len(doc.items) == 0 {
		return nil, fmt.Errorf("no content found: %s", fb2Path)
	}

	// Resolve image references to sibling files, writing each referenced binary
	// once. A reference with no binary keeps an empty resolved name so the page can
	// show a visible placeholder rather than pretend the picture was there.
	resolved := make([]resolvedItem, 0, len(doc.items))
	written := make(map[string]string) // binary id -> filename on disk
	names := newNameSet()
	paras, imgs, dangling := 0, 0, 0
	for _, it := range doc.items {
		if it.imageID == "" {
			resolved = append(resolved, resolvedItem{text: it.text, lines: it.lines, class: it.class})
			paras++
			continue
		}
		name, ok := written[it.imageID]
		if !ok {
			bin, exists := doc.binaries[it.imageID]
			if exists {
				name = names.imageFileName(it.imageID, bin.contentType)
				if err := os.WriteFile(filepath.Join(outputDir, name), bin.data, 0o644); err != nil {
					return nil, fmt.Errorf("write image %s: %w", it.imageID, err)
				}
			}
			// A miss is remembered too, so it is not retried.
			written[it.imageID] = name
		}
		if name == "" {
			resolved = append(resolved, resolvedItem{missingImage: it.imageID})
			dangling++
		} else {
			resolved = append(resolved, resolvedItem{image: name})
			imgs++
		}
	}

	book := &epub.Book{Title: title, BasePath: ""}

	totalPages := (len(resolved) + paragraphsPerPage - 1) / paragraphsPerPage
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		start := (pageNum - 1) * paragraphsPerPage
		end := start + paragraphsPerPage
		if end > len(resolved) {
			end = len(resolved)
		}

		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)

		pageHTML := buildPageHTML(title, pageNum, totalPages, resolved[start:end])
		if err := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write page %d: %w", pageNum, err)
		}

		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID: id, Href: href, MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}

	logging.Printf("  Title: %s\n", title)
	if imgs > 0 || dangling > 0 {
		logging.Printf("  Paragraphs: %d, Images: %d, Pages: %d\n", paras, imgs, totalPages)
	} else {
		logging.Printf("  Paragraphs: %d, Pages: %d\n", paras, totalPages)
	}
	if dangling > 0 {
		logging.Printf("  WARNING: %d image reference(s) had no matching <binary> and show a placeholder\n", dangling)
	}

	return book, nil
}

// resolvedItem is an fb2Item after image references have been turned into on-disk
// filenames (or flagged as missing).
type resolvedItem struct {
	text         string
	lines        []string
	class        string
	image        string // written image filename
	missingImage string // binary id that could not be resolved
}

// ── Rendering ───────────────────────────────────────────────────────

func buildPageHTML(title string, pageNum, totalPages int, items []resolvedItem) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(title), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    p { text-indent: 1.5em; margin: 0.5em 0; }\n")
	sb.WriteString("    p.stanza { text-indent: 0; margin: 0.8em 0 0.8em 2em; }\n")
	sb.WriteString("    p.subtitle { text-indent: 0; text-align: center; font-weight: bold; }\n")
	sb.WriteString("    p.text-author { text-indent: 0; text-align: right; font-style: italic; }\n")
	sb.WriteString("    img { display: block; max-width: 100%; height: auto; margin: 1em auto; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	// Page number is shown in the injected navbar (top-right); no body header needed.

	for _, it := range items {
		switch {
		case it.image != "":
			esc := html.EscapeString(it.image)
			sb.WriteString(fmt.Sprintf("  <img src=\"%s\" alt=\"%s\">\n", esc, esc))
		case it.missingImage != "":
			sb.WriteString(fmt.Sprintf("  <p><em>[image not found: %s]</em></p>\n", html.EscapeString(it.missingImage)))
		case len(it.lines) > 0:
			escaped := make([]string, len(it.lines))
			for i, l := range it.lines {
				escaped[i] = html.EscapeString(l)
			}
			sb.WriteString(fmt.Sprintf("  <p class=\"%s\">%s</p>\n", classStanza, strings.Join(escaped, "<br>")))
		case it.class != "":
			sb.WriteString(fmt.Sprintf("  <p class=\"%s\">%s</p>\n", it.class, html.EscapeString(it.text)))
		default:
			sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(it.text)))
		}
	}

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

// ── Image file names ────────────────────────────────────────────────

// maxImageStem keeps a generated name well inside path-length limits.
const maxImageStem = 48

// nameSet hands out image file names that are unique within one book. Keys are lower-cased
// because the output lands on case-insensitive Windows file systems too.
type nameSet map[string]bool

func newNameSet() nameSet { return make(nameSet) }

// imageFileName builds a safe sibling filename for an embedded image from its id and
// content-type. A plain ASCII id keeps its readable name ("cover.jpg", "fig1.png"). An id that
// had to be sanitized gains a hash of the original: two Cyrillic ids of equal length used to
// sanitize to the same underscores and overwrite each other. An id of ".." or one made only of
// dots can no longer name a directory, and a hash collision still gets a numbered name.
func (s nameSet) imageFileName(id, contentType string) string {
	stem := sanitizeName(id)
	ext := filepath.Ext(stem)
	if len(ext) < 2 || len(ext) > 6 {
		ext = extForContentType(contentType)
	} else {
		stem = strings.TrimSuffix(stem, ext)
	}
	clean := stem == strings.TrimSuffix(id, ext) && !strings.HasPrefix(stem, ".")
	stem = strings.TrimLeft(stem, ".")
	if len(stem) > maxImageStem {
		stem, clean = stem[:maxImageStem], false
	}
	if stem == "" {
		stem, clean = "image", false
	}
	if !clean {
		h := fnv.New32a()
		_, _ = h.Write([]byte(id))
		stem = fmt.Sprintf("%s-%08x", stem, h.Sum32())
	}
	name := stem + ext
	for n := 2; s[strings.ToLower(name)]; n++ {
		name = fmt.Sprintf("%s-%d%s", stem, n, ext)
	}
	s[strings.ToLower(name)] = true
	return name
}

func extForContentType(ct string) string {
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
}

// sanitizeName keeps an id safe as a filename: letters, digits and a few safe
// punctuation marks, everything else to '_'.
func sanitizeName(id string) string {
	var sb strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	return sb.String()
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
