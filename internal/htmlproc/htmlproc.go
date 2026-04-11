// Package htmlproc handles HTML text extraction and replacement for translation.
// It walks the DOM, extracts text nodes (skipping script/style/code/pre),
// and can replace those nodes with translated text.
package htmlproc

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

// ExtractTexts reads an HTML file and returns all translatable text segments.
func ExtractTexts(filePath string) ([]*TextSegment, *html.Node, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}

	var segments []*TextSegment
	walkAndCollect(doc, &segments, false)
	return segments, doc, nil
}

// walkAndCollect recursively traverses the DOM and collects text nodes.
func walkAndCollect(n *html.Node, segments *[]*TextSegment, inSkip bool) {
	if n.Type == html.ElementNode && skipTags[n.DataAtom] {
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
		walkAndCollect(c, segments, inSkip)
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
		// Preserve leading/trailing whitespace from the original node.
		origData := seg.Node.Data
		leading := ""
		trailing := ""
		if len(origData) > 0 && (origData[0] == ' ' || origData[0] == '\n' || origData[0] == '\t') {
			leading = origData[:len(origData)-len(strings.TrimLeft(origData, " \t\n\r"))]
		}
		if len(origData) > 0 {
			trimmed := strings.TrimRight(origData, " \t\n\r")
			trailing = origData[len(trimmed):]
		}
		seg.Node.Data = leading + t + trailing
	}
}

// RenderToFile writes the modified DOM back to a file.
func RenderToFile(doc *html.Node, filePath string) error {
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	return os.WriteFile(filePath, buf.Bytes(), 0o644)
}
