// Package htmlsplit splits large HTML pages at paragraph boundaries so that
// each page's text content stays within a configurable character limit.
// This is useful for browser-extension translation tools (e.g. Chrome GT ext,
// which has a ~5000-char limit per translation request).
package htmlsplit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// SplitIfNeeded inspects every HTML spine item and splits any file whose text
// content exceeds maxChars. Splitting is always done at top-level block
// boundaries (paragraphs, headings, divs) so sentences are never cut.
//
// The book's Manifest and Spine slices are rebuilt in-place with the new
// file set. Non-HTML manifest items (CSS, images …) are kept unchanged.
// Returns the number of additional pages created (0 if nothing was split).
func SplitIfNeeded(book *epub.Book, outputDir string, maxChars int) (int, error) {
	// Index manifest items by ID for fast lookup.
	byID := make(map[string]epub.ManifestItem, len(book.Manifest))
	for _, item := range book.Manifest {
		byID[item.ID] = item
	}

	// Identify which IDs are referenced from the spine.
	spineSet := make(map[string]bool, len(book.Spine))
	for _, sp := range book.Spine {
		spineSet[sp.IDRef] = true
	}

	// Non-spine manifest items (CSS, images, fonts …) are preserved at the
	// front of the new manifest, in their original order.
	var newManifest []epub.ManifestItem
	for _, item := range book.Manifest {
		if !spineSet[item.ID] {
			newManifest = append(newManifest, item)
		}
	}

	var newSpine []epub.SpineItem
	added := 0

	for _, sp := range book.Spine {
		mItem, ok := byID[sp.IDRef]
		if !ok || !isHTMLMedia(mItem.MediaType) {
			// Defensive: keep unknown/non-HTML spine items as-is.
			if ok {
				newManifest = append(newManifest, mItem)
			}
			newSpine = append(newSpine, sp)
			continue
		}

		srcPath := resolveHref(outputDir, book.BasePath, mItem.Href)
		chunks, err := chunkHTMLFile(srcPath, maxChars)
		if err != nil {
			return added, fmt.Errorf("split %s: %w", mItem.Href, err)
		}

		if len(chunks) <= 1 {
			// File is within limit — keep untouched.
			newManifest = append(newManifest, mItem)
			newSpine = append(newSpine, sp)
			continue
		}

		// Write split chunks; the first chunk overwrites the original file.
		base := strings.TrimSuffix(mItem.Href, filepath.Ext(mItem.Href))
		for i, chunk := range chunks {
			var href, id string
			if i == 0 {
				href = mItem.Href
				id = mItem.ID
			} else {
				href = fmt.Sprintf("%s_s%d.html", base, i+1)
				id = fmt.Sprintf("%s_s%d", mItem.ID, i+1)
				added++
			}
			destPath := resolveHref(outputDir, book.BasePath, href)
			if err := os.WriteFile(destPath, []byte(chunk), 0o644); err != nil {
				return added, fmt.Errorf("write split chunk %d of %s: %w", i+1, mItem.Href, err)
			}
			newManifest = append(newManifest, epub.ManifestItem{
				ID:        id,
				Href:      href,
				MediaType: "text/html",
			})
			newSpine = append(newSpine, epub.SpineItem{IDRef: id})
		}
	}

	book.Manifest = newManifest
	book.Spine = newSpine
	return added, nil
}

// chunkHTMLFile reads srcPath and splits its body content into chunks whose
// text length is ≤ maxChars. Each chunk is returned as a complete HTML page.
// If the file is already within the limit, the slice has one element (original bytes).
func chunkHTMLFile(srcPath string, maxChars int) ([]string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}

	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	head, body := findHeadBody(doc)
	if body == nil {
		// Can't parse body; return original unchanged.
		return []string{string(data)}, nil
	}

	if textLen(body) <= maxChars {
		return []string{string(data)}, nil
	}

	// Collect direct body children that carry visible content.
	var blocks []*gohtml.Node
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case gohtml.ElementNode:
			blocks = append(blocks, c)
		case gohtml.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				blocks = append(blocks, c)
			}
		}
	}

	// Render <head> once; reuse in every chunk page.
	headHTML := ""
	if head != nil {
		var hb bytes.Buffer
		if err := gohtml.Render(&hb, head); err != nil {
			return nil, fmt.Errorf("render head: %w", err)
		}
		headHTML = hb.String()
	}

	groups := groupBlocks(blocks, maxChars)

	result := make([]string, 0, len(groups))
	for _, group := range groups {
		var bb bytes.Buffer
		for _, n := range group {
			if err := gohtml.Render(&bb, n); err != nil {
				return nil, fmt.Errorf("render block: %w", err)
			}
		}
		page := "<!DOCTYPE html>\n<html>\n" + headHTML + "\n<body>\n" + bb.String() + "\n</body>\n</html>\n"
		result = append(result, page)
	}
	return result, nil
}

// groupBlocks partitions blocks so each group's text length is ≤ maxChars.
// A single oversized block is kept as its own group (no mid-block splitting).
func groupBlocks(blocks []*gohtml.Node, maxChars int) [][]*gohtml.Node {
	var groups [][]*gohtml.Node
	var current []*gohtml.Node
	currentLen := 0

	for _, b := range blocks {
		bLen := textLen(b)
		if len(current) > 0 && currentLen+bLen > maxChars {
			groups = append(groups, current)
			current = nil
			currentLen = 0
		}
		current = append(current, b)
		currentLen += bLen
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

// textLen returns the total character count of visible text within node n.
// Script and style element content is excluded.
func textLen(n *gohtml.Node) int {
	if n == nil {
		return 0
	}
	if n.Type == gohtml.TextNode {
		return len(strings.TrimSpace(n.Data))
	}
	if n.Type == gohtml.ElementNode {
		switch n.DataAtom {
		case atom.Script, atom.Style:
			return 0
		}
	}
	total := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		total += textLen(c)
	}
	return total
}

// findHeadBody locates the <head> and <body> nodes in a parsed document.
func findHeadBody(doc *gohtml.Node) (head, body *gohtml.Node) {
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			switch n.DataAtom {
			case atom.Head:
				if head == nil {
					head = n
				}
			case atom.Body:
				if body == nil {
					body = n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return
}

// isHTMLMedia reports whether mt is an HTML content type.
func isHTMLMedia(mt string) bool {
	return mt == "text/html" || mt == "application/xhtml+xml" || mt == ""
}

// resolveHref joins outputDir with an OPF-relative href, accounting for the
// EPUB's basePath (the directory containing content.opf).
func resolveHref(outputDir, basePath, href string) string {
	if basePath != "" && basePath != "." {
		return filepath.Join(outputDir, filepath.FromSlash(basePath), filepath.FromSlash(href))
	}
	return filepath.Join(outputDir, filepath.FromSlash(href))
}
