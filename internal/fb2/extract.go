// Package fb2 handles FictionBook (FB2) XML conversion to HTML pages.
// Uses stdlib encoding/xml for parsing.
package fb2

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
)

// paragraphsPerPage controls page splitting.
const paragraphsPerPage = 30

// ── FB2 XML structures (title only) ─────────────────────────────────
// Body content and images are walked token by token (collectContent) so text
// and pictures keep their document order; only the title metadata is small and
// positional-order-free enough to unmarshal into a struct.

type fb2File struct {
	Description fb2Desc `xml:"description"`
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

// ── Content model ───────────────────────────────────────────────────

// fb2Item is one piece of body content in document order: either a paragraph of
// text or an image reference (an id pointing at a <binary>, later resolved to a
// written filename).
type fb2Item struct {
	text    string // paragraph text; empty for an image
	imageID string // binary id (the <image l:href="#id"> target); empty for text
}

// fb2Binary is a decoded embedded image.
type fb2Binary struct {
	contentType string
	data        []byte
}

// ── Public API ──────────────────────────────────────────────────────

// Extract reads an FB2 file, parses its XML, generates per-page HTML files in
// outputDir, and returns an *epub.Book adapter. Embedded images (<binary>) that
// the body references (<image>) are decoded to sibling files and shown in place;
// a reference with no matching binary degrades to a visible note, not a silent gap.
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

	items, binaries, err := collectContent(data)
	if err != nil {
		return nil, fmt.Errorf("parse fb2 content: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no content found: %s", fb2Path)
	}

	// Resolve image references to sibling files, writing each referenced binary
	// once. A reference with no binary keeps an empty resolved name so the page can
	// show a visible placeholder rather than pretend the picture was there.
	resolved := make([]resolvedItem, 0, len(items))
	written := make(map[string]string) // binary id -> filename on disk
	paras, imgs, dangling := 0, 0, 0
	for _, it := range items {
		if it.imageID == "" {
			resolved = append(resolved, resolvedItem{text: it.text})
			paras++
			continue
		}
		name, ok := written[it.imageID]
		if !ok {
			bin, exists := binaries[it.imageID]
			if exists {
				name = imageFileName(it.imageID, bin.contentType)
				if err := os.WriteFile(filepath.Join(outputDir, name), bin.data, 0o644); err != nil {
					return nil, fmt.Errorf("write image %s: %w", it.imageID, err)
				}
				written[it.imageID] = name
			} else {
				written[it.imageID] = "" // remember the miss so we do not retry
			}
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
		if err := os.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
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
	image        string // written image filename
	missingImage string // binary id that could not be resolved
}

// ── Content walk ────────────────────────────────────────────────────

// collectContent walks the FB2 token stream once, returning body content in
// document order plus every decoded <binary>. It is token-based rather than
// struct-unmarshalled so a picture between two paragraphs stays between them
// instead of being dropped or floated to the end.
func collectContent(data []byte) ([]fb2Item, map[string]fb2Binary, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	binaries := make(map[string]fb2Binary)
	var items []fb2Item
	bodyDepth := 0
	inCover := false
	coverID := ""

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "body":
				bodyDepth++
			case "coverpage":
				inCover = true
			case "binary":
				readBinary(dec, t, binaries)
			case "image":
				href := strings.TrimPrefix(attrValue(t, "href"), "#")
				switch {
				case inCover:
					if coverID == "" {
						coverID = href
					}
				case bodyDepth > 0:
					items = append(items, fb2Item{imageID: href})
				}
			case "p":
				if bodyDepth > 0 {
					text, inlineImages := readParagraph(dec)
					if text != "" {
						items = append(items, fb2Item{text: text})
					}
					// FB2 commonly wraps an illustration in its own <p>
					// (<p><image l:href="#id"/></p>), so images live inside paragraphs,
					// not only between them - emit those too or the picture is lost.
					for _, id := range inlineImages {
						items = append(items, fb2Item{imageID: strings.TrimPrefix(id, "#")})
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "body":
				bodyDepth--
			case "coverpage":
				inCover = false
			}
		}
	}

	// The cover lives in <description>, ahead of the body, so it opens the book.
	if coverID != "" {
		items = append([]fb2Item{{imageID: coverID}}, items...)
	}
	return items, binaries, nil
}

// readBinary consumes a <binary> element's base64 body and stores the decoded
// bytes. FB2 wraps base64 across lines, so whitespace is stripped before decoding.
func readBinary(dec *xml.Decoder, start xml.StartElement, into map[string]fb2Binary) {
	id := attrValue(start, "id")
	contentType := attrValue(start, "content-type")
	var b64 strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.CharData:
			b64.Write(t)
		case xml.EndElement:
			if t.Name.Local == "binary" {
				if id == "" {
					return
				}
				raw := strings.Map(dropSpace, b64.String())
				decoded, err := base64.StdEncoding.DecodeString(raw)
				if err != nil {
					return // a malformed binary is left unresolved -> visible placeholder
				}
				into[id] = fb2Binary{contentType: contentType, data: decoded}
				return
			}
		}
	}
}

// readParagraph consumes the current <p> (already read as a StartElement), returning
// its flattened text and the hrefs of any inline <image> elements. Inline formatting
// (<emphasis>, <strong>, ..) is flattened to text, matching the previous behaviour;
// images are pulled out so an illustration wrapped in a <p> is not lost.
func readParagraph(dec *xml.Decoder) (text string, images []string) {
	var sb strings.Builder
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.StartElement:
			depth++
			if t.Name.Local == "image" {
				images = append(images, attrValue(t, "href"))
			}
		case xml.EndElement:
			depth--
		}
	}
	return strings.TrimSpace(sb.String()), images
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
		default:
			sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(it.text)))
		}
	}

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

// ── Small helpers ───────────────────────────────────────────────────

// attrValue returns the value of the attribute with the given local name,
// ignoring its namespace (FB2's image href is the XLink-namespaced l:href).
func attrValue(e xml.StartElement, local string) string {
	for _, a := range e.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// imageFileName builds a safe sibling filename for an embedded image from its id
// and content-type.
func imageFileName(id, contentType string) string {
	base := sanitizeName(id)
	if filepath.Ext(base) != "" {
		return base // the id already carries an extension (e.g. "cover.jpg")
	}
	return base + extForContentType(contentType)
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
// punctuation marks, everything else to '_'. Never empty.
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
	if sb.Len() == 0 {
		return "image"
	}
	return sb.String()
}

func dropSpace(r rune) rune {
	if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
		return -1
	}
	return r
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
