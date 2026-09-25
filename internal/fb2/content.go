package fb2

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"doc-html-translate/internal/textutil"
)

// Paragraph classes for the prose elements that are not plain <p>. The extension's fb2.js
// renders the same classes (docs/PARITY.md, "FB2 prose elements").
const (
	classStanza     = "stanza"
	classSubtitle   = "subtitle"
	classTextAuthor = "text-author"
)

// fb2Item is one piece of body content in document order: a paragraph of text (with an
// optional class), a stanza of verse lines, or an image reference (an id pointing at a
// <binary>, later resolved to a written filename).
type fb2Item struct {
	text    string   // paragraph text
	lines   []string // verse lines of a stanza
	class   string
	imageID string // binary id (the <image l:href="#id"> target)
}

// fb2Binary is a decoded embedded image.
type fb2Binary struct {
	contentType string
	data        []byte
}

// fb2Doc is everything one pass over the file yields.
type fb2Doc struct {
	title    string
	items    []fb2Item
	binaries map[string]fb2Binary
}

// declPrescanBytes bounds the search for the XML declaration, which must open the document.
const declPrescanBytes = 1024

var xmlDeclEncodingRe = regexp.MustCompile(`^\s*<\?xml\s[^>]*?\bencoding\s*=\s*["']([A-Za-z0-9._:-]+)["']`)

// decodingReader returns r transcoded to UTF-8. A BOM wins, then the encoding the XML
// declaration names (resolved as a WHATWG label, the set the extension's TextDecoder knows),
// then UTF-8. A Russian FB2 declared windows-1251 used to fail outright, because encoding/xml
// only reads UTF-8 and nothing supplied a charset reader. An unknown label and UTF-8 alike go
// through a validating decoder, so a damaged byte shows as U+FFFD instead of failing the parse.
func decodingReader(r io.Reader) io.Reader {
	br := bufio.NewReaderSize(r, 64<<10)
	head, _ := br.Peek(declPrescanBytes) // a short file returns what there is
	switch {
	case bytes.HasPrefix(head, []byte{0xEF, 0xBB, 0xBF}):
		_, _ = br.Discard(3)
	case bytes.HasPrefix(head, []byte{0xFF, 0xFE}):
		return transform.NewReader(br, unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder())
	case bytes.HasPrefix(head, []byte{0xFE, 0xFF}):
		return transform.NewReader(br, unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder())
	default:
		if m := xmlDeclEncodingRe.FindSubmatch(head); m != nil {
			if codec, ok := textutil.LookupCodec(string(m[1])); ok {
				return codec.NewReader(br)
			}
		}
	}
	return transform.NewReader(br, unicode.UTF8.NewDecoder())
}

// parseFB2 walks the FB2 token stream once, streaming from r, and returns the title, body
// content in document order and every decoded <binary>. It is token-based rather than
// struct-unmarshalled so a picture between two paragraphs stays between them, and single-pass
// so a large illustrated book is never held in memory twice.
func parseFB2(r io.Reader) (*fb2Doc, error) {
	dec := xml.NewDecoder(decodingReader(r))
	// The bytes are UTF-8 by now, whatever the declaration still says.
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }

	doc := &fb2Doc{binaries: make(map[string]fb2Binary)}
	bodyDepth := 0
	inCover, inTitleInfo, inStanza := false, false, false
	coverID := ""
	var stanza, stanzaImages []string

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			switch name {
			case "body":
				bodyDepth++
			case "coverpage":
				inCover = true
			case "title-info":
				inTitleInfo = true
			case "book-title":
				if inTitleInfo && doc.title == "" {
					doc.title, _ = readInline(dec)
				}
			case "binary":
				readBinary(dec, t, doc.binaries)
			case "image":
				href := strings.TrimPrefix(attrValue(t, "href"), "#")
				switch {
				case inCover:
					if coverID == "" {
						coverID = href
					}
				case bodyDepth > 0:
					doc.items = append(doc.items, fb2Item{imageID: href})
				}
			case "stanza":
				if bodyDepth > 0 {
					inStanza, stanza = true, nil
				}
			case "v":
				if bodyDepth > 0 {
					line, images := readInline(dec)
					if inStanza {
						if line != "" {
							stanza = append(stanza, line)
						}
						// A picture inside a verse line follows the whole stanza.
						stanzaImages = append(stanzaImages, images...)
						break
					}
					if line != "" {
						doc.items = append(doc.items, fb2Item{lines: []string{line}, class: classStanza})
					}
					doc.addImages(images)
				}
			case "p", "subtitle", "text-author", "td", "th":
				if bodyDepth > 0 {
					text, images := readInline(dec)
					if text != "" {
						doc.items = append(doc.items, fb2Item{text: text, class: proseClass(name)})
					}
					// FB2 commonly wraps an illustration in its own <p>
					// (<p><image l:href="#id"/></p>), so images live inside paragraphs,
					// not only between them - emit those too or the picture is lost.
					doc.addImages(images)
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "body":
				bodyDepth--
			case "coverpage":
				inCover = false
			case "title-info":
				inTitleInfo = false
			case "stanza":
				if inStanza && len(stanza) > 0 {
					doc.items = append(doc.items, fb2Item{lines: stanza, class: classStanza})
				}
				doc.addImages(stanzaImages)
				inStanza, stanza, stanzaImages = false, nil, nil
			}
		}
	}

	// The cover lives in <description>, ahead of the body, so it opens the book.
	if coverID != "" {
		doc.items = append([]fb2Item{{imageID: coverID}}, doc.items...)
	}
	return doc, nil
}

func (d *fb2Doc) addImages(hrefs []string) {
	for _, id := range hrefs {
		d.items = append(d.items, fb2Item{imageID: strings.TrimPrefix(id, "#")})
	}
}

// proseClass is the paragraph class for a prose element; table cells and plain paragraphs
// need none.
func proseClass(element string) string {
	switch element {
	case "subtitle":
		return classSubtitle
	case "text-author":
		return classTextAuthor
	}
	return ""
}

// b64Chunk is how much base64 text is gathered before decoding it; a multiple of 4, so every
// chunk but the last is whole quanta.
const b64Chunk = 16 << 10

// readBinary consumes a <binary> element and stores the decoded bytes. FB2 wraps base64 across
// lines; the whitespace is dropped and the text decoded chunk by chunk straight into the
// output, so a picture costs its decoded size plus one small buffer rather than several copies
// of its base64 text.
func readBinary(dec *xml.Decoder, start xml.StartElement, into map[string]fb2Binary) {
	id := attrValue(start, "id")
	contentType := attrValue(start, "content-type")
	var out []byte
	var chunk [b64Chunk]byte
	n := 0
	bad := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.CharData:
			if out == nil {
				out = make([]byte, 0, base64.StdEncoding.DecodedLen(len(t)))
			}
			for _, c := range t {
				if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
					continue
				}
				chunk[n] = c
				n++
				if n == len(chunk) {
					if out, err = base64.StdEncoding.AppendDecode(out, chunk[:n]); err != nil {
						bad = true
					}
					n = 0
				}
			}
		case xml.EndElement:
			if t.Name.Local != "binary" {
				continue
			}
			if n > 0 {
				if out, err = base64.StdEncoding.AppendDecode(out, chunk[:n]); err != nil {
					bad = true
				}
			}
			// A malformed binary is left unresolved -> visible placeholder.
			if id != "" && !bad {
				into[id] = fb2Binary{contentType: contentType, data: out}
			}
			return
		}
	}
}

// readInline consumes the current element (already read as a StartElement), returning its
// flattened text and the hrefs of any inline <image> elements. Inline formatting
// (<emphasis>, <strong>, ..) is flattened to text; images are pulled out so an illustration
// wrapped in a <p> is not lost.
func readInline(dec *xml.Decoder) (text string, images []string) {
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
	// Whitespace collapses as the extension's textContent-based reader collapses it, so both
	// editions produce the same paragraph text.
	return strings.Join(strings.Fields(sb.String()), " "), images
}

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
