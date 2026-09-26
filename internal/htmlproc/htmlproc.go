// Package htmlproc handles HTML text extraction and replacement for translation.
// It walks the DOM, extracts text nodes (skipping script/style/code/pre),
// and can replace those nodes with translated text.
package htmlproc

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"doc-html-translate/internal/fsutil"
)

// skipTags defines HTML elements whose text content should NOT be translated.
var skipTags = map[atom.Atom]bool{
	atom.Script: true,
	atom.Style:  true,
	atom.Code:   true,
	atom.Pre:    true,
}

// TextSegment represents a text node found in the HTML DOM.
type TextSegment struct {
	Node *html.Node
	Text string
}

// ExtractTexts reads an HTML file and returns all translatable text segments. skip, when not
// nil, names further elements whose whole subtree is left out - the caller's own markup, such
// as reader chrome, that is not the document's text.
func ExtractTexts(filePath string, skip func(*html.Node) bool) ([]*TextSegment, *html.Node, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}

	var segments []*TextSegment
	walkAndCollect(doc, &segments, skip, false)
	return segments, doc, nil
}

// walkAndCollect recursively traverses the DOM and collects text nodes.
func walkAndCollect(n *html.Node, segments *[]*TextSegment, skip func(*html.Node) bool, inSkip bool) {
	if n.Type == html.ElementNode && (skipTags[n.DataAtom] || (skip != nil && skip(n))) {
		inSkip = true
	}

	if n.Type == html.TextNode && !inSkip {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			*segments = append(*segments, &TextSegment{
				Node: n,
				Text: text,
			})
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkAndCollect(c, segments, skip, inSkip)
	}
}

// ReplaceTexts takes segments and their translated counterparts,
// updates the DOM nodes in-place.
func ReplaceTexts(segments []*TextSegment, translated []string) {
	for i, seg := range segments {
		if i >= len(translated) {
			break
		}
		t := translated[i]
		if t == "" {
			// Model returned nothing — keep original text intact.
			continue
		}
		// Preserve leading/trailing whitespace from the original node - the same Unicode
		// set ExtractTexts trimmed, so a no-break space before an inline element survives.
		origData := seg.Node.Data
		leading := origData[:len(origData)-len(strings.TrimLeftFunc(origData, unicode.IsSpace))]
		trailing := origData[len(strings.TrimRightFunc(origData, unicode.IsSpace)):]
		seg.Node.Data = leading + t + trailing
	}
}

// RenderToFile writes the modified DOM back to a file.
func RenderToFile(doc *html.Node, filePath string) error {
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	return fsutil.WriteFile(filePath, buf.Bytes(), 0o644)
}
