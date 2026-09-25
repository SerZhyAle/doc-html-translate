// Package htmlsplit splits large HTML pages at paragraph boundaries so that
// each page's text content stays within a configurable character limit.
// This is useful for browser-extension translation tools (e.g. Chrome GT ext,
// which has a ~5000-char limit per translation request).
package htmlsplit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
)

// SplitIfNeeded inspects every HTML spine item and splits any file whose text
// content exceeds maxChars characters. Splitting is always done at block
// boundaries (paragraphs, headings, divs) so sentences are never cut; a sole
// wrapper element is descended into and repeated around every part.
//
// The book's Manifest and Spine slices are rebuilt in-place with the new
// file set. Non-HTML manifest items (CSS, images ..) are kept unchanged.
// TOC entries and in-book links whose target anchor moved to a later part
// are re-pointed at that part.
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

	// Non-spine manifest items (CSS, images, fonts ..) are preserved at the
	// front of the new manifest, in their original order.
	var newManifest []epub.ManifestItem
	for _, item := range book.Manifest {
		if !spineSet[item.ID] {
			newManifest = append(newManifest, item)
		}
	}

	var newSpine []epub.SpineItem
	added := 0
	index := anchorIndex{}

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
		parts, err := chunkHTMLFile(srcPath, maxChars)
		if err != nil {
			return added, fmt.Errorf("split %s: %w", mItem.Href, err)
		}

		if len(parts) <= 1 {
			newManifest = append(newManifest, mItem)
			newSpine = append(newSpine, sp)
			continue
		}

		// Write split parts; the first part overwrites the original file.
		base := strings.TrimSuffix(mItem.Href, filepath.Ext(mItem.Href))
		hrefs := make([]string, len(parts))
		for i, part := range parts {
			var href, id string
			if i == 0 {
				href = mItem.Href
				id = mItem.ID
			} else {
				href = fmt.Sprintf("%s_s%d.html", base, i+1)
				id = fmt.Sprintf("%s_s%d", mItem.ID, i+1)
				added++
			}
			hrefs[i] = href
			destPath := resolveHref(outputDir, book.BasePath, href)
			if err := os.WriteFile(destPath, part.html, 0o644); err != nil {
				return added, fmt.Errorf("write split part %d of %s: %w", i+1, mItem.Href, err)
			}
			newManifest = append(newManifest, epub.ManifestItem{
				ID:        id,
				Href:      href,
				MediaType: "text/html",
			})
			newSpine = append(newSpine, epub.SpineItem{IDRef: id})
		}
		index.add(newSplitGroup(hrefs, parts))
	}

	book.Manifest = newManifest
	book.Spine = newSpine

	if len(index) > 0 {
		rewriteTOC(book.TOC, index)
		if err := rewriteContentLinks(newManifest, outputDir, book.BasePath, index); err != nil {
			return added, fmt.Errorf("rewrite links after split: %w", err)
		}
	}
	return added, nil
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
