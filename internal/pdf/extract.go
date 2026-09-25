// Package pdf handles text extraction from PDF files and conversion to HTML pages.
// Uses github.com/ledongthuc/pdf for text extraction.
// Only supports text-based PDFs; OCR for scanned/image PDFs is a future TODO.
package pdf

import (
	"context"
	"fmt"
	"html"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/procrun"
	"doc-html-translate/internal/textutil"

	pdflib "github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// maxPageSize is the safety limit per page text (10 MB).
const maxPageSize = 10 * 1024 * 1024

// maxPDFToTextOutput caps what is kept of pdftotext's output. Text runs to a few megabytes
// even for a very long book, so reaching the cap means something is wrong with the file.
const maxPDFToTextOutput = 256 << 20

// Extract reads a PDF file, generates per-page HTML files in outputDir,
// and returns an *epub.Book adapter for pipeline compatibility.
// Tries pdftotext (Xpdf/Poppler) first for best quality; falls back to the pure-Go reader.
func Extract(pdfPath, outputDir string) (*epub.Book, error) {
	// pdftotext handles complex font encodings, ligatures, and text ordering
	// far better than the pure-Go reader.
	if pdftotext := findPDFToText(); pdftotext != "" {
		book, err := extractWithPDFToText(pdftotext, pdfPath, outputDir)
		if err == nil {
			return book, nil
		}
		logging.Printf("  WARNING: pdftotext failed, falling back to pdflib: %v\n", err)
		if isExecFileNotFound(err) {
			if book := retryBlockedPDFToText(pdfPath, outputDir); book != nil {
				return book, nil
			}
		}
	} else if advice := pdftotextMissingAdvice(); advice != "" {
		logging.Printf("  %s\n", advice)
	}

	book, err := extractWithPDFLib(pdfPath, outputDir)
	if err == nil {
		return book, nil
	}

	logging.Printf("  WARNING: PDF extract failed, trying repair fallback: %v\n", err)
	repairedPath, repErr := tryRepairPDF(pdfPath)
	if repErr != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(repairedPath) }()

	book, retryErr := extractWithPDFLib(repairedPath, outputDir)
	if retryErr != nil {
		if strings.Contains(retryErr.Error(), "no text content found in PDF") {
			return nil, fmt.Errorf("no text content found in PDF (likely scanned/image-only, OCR required): %s", pdfPath)
		}
		return nil, retryErr
	}

	logging.Printf("  PDF repair fallback succeeded.\n")
	return book, nil
}

// pageItem is a single rendered block on a PDF page with its HTML tag type.
type pageItem struct {
	text string
	tag  string // "p", "h2", "h3"
}

// extractWithPDFToText uses pdftotext -layout (Xpdf/Poppler) for extraction.
// The -layout flag preserves indentation so we can detect headings by centering.
func extractWithPDFToText(pdftotextBin, pdfPath, outputDir string) (*epub.Book, error) {
	pdfPathForTool := pdfPath
	cleanup := func() {}
	if needsPDFToTextPathStaging(pdfPath) {
		stagedPath, cleanupFn, err := stagePDFForPDFToText(pdfPath)
		if err != nil {
			return nil, fmt.Errorf("stage pdf for pdftotext: %w", err)
		}
		pdfPathForTool = stagedPath
		cleanup = cleanupFn
	}
	defer cleanup()

	res, err := procrun.Run(context.Background(), procrun.Cmd{
		Tool:      "pdftotext",
		Path:      pdftotextBin,
		Args:      []string{"-layout", "-enc", "UTF-8", pdfPathForTool, "-"},
		Timeout:   procrun.PDFToText.ForFile(pdfPath),
		MaxStdout: maxPDFToTextOutput,
	})
	if err != nil {
		return nil, err
	}
	// A cut-off text would silently lose the book's last pages; the pure-Go reader is slower
	// but reads them all.
	if res.StdoutTruncated {
		return nil, fmt.Errorf("pdftotext: output larger than %d MB", maxPDFToTextOutput>>20)
	}

	text := textutil.NormalizeLineSeparatorsPreserveFormFeed(string(res.Stdout))

	// Pages are separated by form-feed \f
	pageTexts := strings.Split(text, "\f")
	for len(pageTexts) > 0 && strings.TrimSpace(pageTexts[len(pageTexts)-1]) == "" {
		pageTexts = pageTexts[:len(pageTexts)-1]
	}
	if len(pageTexts) == 0 {
		return nil, fmt.Errorf("pdftotext extracted no content")
	}

	title := pdfTitle(pdfPath)
	book := &epub.Book{Title: title}

	images := extractImages(pdfPath, outputDir)
	pageImages := images.byPage
	// The trim above also drops trailing pages that have no text but do have a picture
	// (a back cover, scanned plates). The document's own count brings them back; it only
	// ever extends the list, so a count pdfcpu gets wrong on a malformed file cannot cost
	// pages the text extractor did see.
	for len(pageTexts) < images.pageCount {
		pageTexts = append(pageTexts, "")
	}
	totalPages := len(pageTexts)

	// Map each emitted page's source PDF page number to its generated href, so
	// PDF bookmarks (which reference 1-based PDF pages) can be linked even when
	// blank pages were skipped and output pages were renumbered.
	pdfPageToHref := make(map[int]string, totalPages)

	generated := 0
	withText := 0
	for i, rawPage := range pageTexts {
		pdfPageNum := i + 1
		imgs := pageImages[pdfPageNum]

		items := parsePDFLayoutPage(rawPage)
		if len(items) == 0 && len(imgs) == 0 {
			continue
		}

		generated++
		if len(items) > 0 {
			withText++
		}
		href := fmt.Sprintf("page_%03d.html", generated)
		id := fmt.Sprintf("page_%03d", generated)
		pdfPageToHref[pdfPageNum] = href

		pageHTML := buildPDFPageHTML(outputDir, title, pdfPageNum, totalPages, items, imgs)
		if err := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write page %d: %w", pdfPageNum, err)
		}
		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID: id, Href: href, MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}

	if generated == 0 {
		return nil, fmt.Errorf("pdftotext: no text content found (scanned/image-only PDF?)")
	}

	book.TOC = buildPDFTOC(pdfPath, pdfPageToHref)

	logging.Printf("  Title: %s\n", title)
	logPageCounts(totalPages, withText, generated)
	return book, nil
}

// logPageCounts reports what the conversion actually produced, keeping the kinds of page
// apart. One number cannot distinguish a book of prose from a book of scans, and that is
// exactly what a reader checks this line for: "with text" counts pages carrying real text,
// "image-only" counts pages that are a picture and nothing else - the shape of a scan, and
// the hint that -ocr is what makes them readable. Pages yielding neither are emitted nowhere,
// so they are named too rather than left as an unexplained gap below the total.
//
// The counts must come from the emit loop's own decisions. Reporting the number of *emitted*
// pages under a "with text" label is what made this line untrue for every scanned PDF.
func logPageCounts(totalPages, withText, generated int) {
	logging.Printf("  Pages: %d (%s)\n", totalPages, pageCountSummary(totalPages, withText, generated))
}

// pageCountSummary is the pure half of logPageCounts, split out so the wording can be tested
// without capturing stdout.
func pageCountSummary(totalPages, withText, generated int) string {
	summary := fmt.Sprintf("with text: %d", withText)
	if imageOnly := generated - withText; imageOnly > 0 {
		summary += fmt.Sprintf(", image-only: %d", imageOnly)
	}
	if empty := totalPages - generated; empty > 0 {
		summary += fmt.Sprintf(", empty: %d", empty)
	}
	return summary
}

// zeroWidthSpaceMarker (U+200B) is emitted literally by pdftotext for some PDFs
// as an invisible first-line indent on paragraph openings. We treat its presence
// in a line's leading whitespace as a paragraph-start signal and strip it from
// the rendered text.
const zeroWidthSpaceMarker = "\u200b"

// paragraphIndentMin is the minimum number of leading spaces (after stripping any
// zero-width-space marker) that marks the first line of a new paragraph. Wrapped
// continuation lines sit flush at the left margin (0–1 spaces).
const paragraphIndentMin = 2

// layoutBlock is one blank-line-delimited block of a pdftotext -layout page.
type layoutBlock struct {
	text          string // physical lines joined by single spaces, ZWSP stripped
	leadingSpaces int    // leading spaces of the first line (ZWSP already stripped)
	indented      bool   // first line opens a new paragraph (ZWSP marker present or leadingSpaces >= paragraphIndentMin)
	singleLine    bool   // block held exactly one non-empty physical line
}

// parsePDFLayoutPage parses one page from pdftotext -layout output into pageItems.
//
// In -layout mode:
//   - blank lines (\n\n) separate visual blocks (paragraphs / headings)
//   - leading spaces on the first line of a block indicate centering (more spaces = more centered)
//   - body paragraph first lines have ~1 space indent; centered headings have 8+ spaces
//   - lines within a block are word-wrapped and must be re-joined with spaces
//
// Some PDFs extract "double-spaced": a blank line follows almost every wrapped
// line, so each physical line lands in its own block and would become its own
// <p>. When that artifact is detected, non-indented continuation blocks are glued
// back onto the paragraph they belong to (see isDoubleSpacedLayout).
func parsePDFLayoutPage(text string) []pageItem {
	blocks := parseLayoutBlocks(text)
	if len(blocks) == 0 {
		return nil
	}

	merge := isDoubleSpacedLayout(blocks)

	var items []pageItem
	for _, b := range blocks {
		// A non-indented, single-line block in double-spaced mode is a wrapped
		// continuation of the paragraph above it. Only fold into body paragraphs
		// ("p"), never into headings, so a heading stays on its own line. Multi-line
		// blocks are paragraphs pdftotext already kept whole — never a continuation.
		if merge && b.singleLine && !b.indented && len(items) > 0 && items[len(items)-1].tag == "p" {
			items[len(items)-1].text += " " + b.text
			continue
		}
		items = append(items, pageItem{b.text, classifyBlock(b.text, b.leadingSpaces)})
	}

	return items
}

// parseLayoutBlocks splits a page into blank-line-delimited blocks and records,
// per block, the joined text, first-line indent, and whether it opens a paragraph.
func parseLayoutBlocks(text string) []layoutBlock {
	var blocks []layoutBlock

	for _, block := range strings.Split(text, "\n\n") {
		lines := strings.Split(block, "\n")

		// Collect non-empty lines and measure the leading indent of the first one.
		leadingSpaces := 0
		indented := false
		firstSeen := false
		nonEmpty := 0
		var parts []string
		for _, line := range lines {
			// Strip trailing whitespace only; keep leading for indent measurement.
			rline := strings.TrimRight(line, " \t\r")
			if strings.TrimSpace(rline) == "" {
				continue
			}
			nonEmpty++
			if !firstSeen {
				// Leading whitespace may mix spaces, tabs, and the ZWSP marker.
				trimmed := strings.TrimLeft(rline, " \t"+zeroWidthSpaceMarker)
				leadWS := rline[:len(rline)-len(trimmed)]
				leadingSpaces = strings.Count(leadWS, " ")
				indented = strings.Contains(leadWS, zeroWidthSpaceMarker) || leadingSpaces >= paragraphIndentMin
				firstSeen = true
			}
			parts = append(parts, strings.TrimSpace(strings.ReplaceAll(rline, zeroWidthSpaceMarker, "")))
		}

		if len(parts) == 0 {
			continue
		}

		joined := strings.Join(parts, " ")
		if isLigaturesArtifact(joined) {
			continue
		}

		blocks = append(blocks, layoutBlock{
			text:          joined,
			leadingSpaces: leadingSpaces,
			indented:      indented,
			singleLine:    nonEmpty == 1,
		})
	}

	return blocks
}

// isDoubleSpacedLayout reports whether a page exhibits the pdftotext artifact
// where a blank line follows almost every wrapped line, so each physical line
// lands in its own single-line block. Detected when single-line blocks dominate;
// normal PDFs keep whole paragraphs in multi-line blocks and return false, leaving
// their parsing unchanged.
func isDoubleSpacedLayout(blocks []layoutBlock) bool {
	if len(blocks) < 5 {
		return false
	}
	single := 0
	for _, b := range blocks {
		if b.singleLine {
			single++
		}
	}
	return float64(single)/float64(len(blocks)) >= 0.7
}

// classifyBlock assigns an HTML tag based on text content and indentation.
//
// Centering heuristic: pdftotext -layout right-pads lines to the page width.
// A centered heading like "LITTLE TOKYO" gets ~20 spaces of left indent.
// Body paragraph first lines get ~1 space (first-line indent).
// leadingSpaces > 8 = centered = heading candidate.
// PDF reflow heuristic constants. These paragraph/heading thresholds are shared,
// value-for-value, with the browser extension's reflow.js (a hand port). A change here
// must be mirrored there and in docs/PARITY.md ("PDF reflow heuristics").
const (
	paraGapFactor         = 1.5  // paragraph break when a Y-gap exceeds this * median line spacing (reflow.js PARA_GAP_FACTOR)
	indentThreshold       = 8.0  // points; first-line indent past the left margin starts a paragraph (reflow.js INDENT_THRESHOLD)
	medianGapFallback     = 12.0 // fallback median line spacing when it can't be measured
	ligatureMaxAvgWordLen = 3.0  // avg word length below this (over >= ligatureMinWords words) = ligature garbage
	ligatureMinWords      = 4
	headingShortWords     = 8  // "short" line word cap for an h2 heading candidate
	headingMediumWords    = 14 // "medium" line word cap for an h3 heading candidate
)

func classifyBlock(text string, leadingSpaces int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "p"
	}

	// Language-agnostic ALL-CAPS test: the line is all upper-case (equals its own
	// upper-casing) AND contains at least one cased letter (differs from its
	// lower-casing). This catches Cyrillic headings like "ГЛАВА ПЕРВАЯ" - matching the
	// extension's `/[A-ZА-ЯЁ]/u` check - not just Latin A-Z, while still rejecting
	// digit/punctuation-only lines (which lower-case to themselves). See docs/PARITY.md.
	upper := strings.ToUpper(text)
	isAllCaps := upper == text && text != strings.ToLower(text)
	isCentered := leadingSpaces > 8
	isShort := len(words) <= headingShortWords

	switch {
	case (isAllCaps || isCentered) && isShort:
		return "h2"
	case isCentered && len(words) <= headingMediumWords:
		return "h3"
	default:
		return "p"
	}
}

// isLigaturesArtifact returns true for lines that are ligature-garbage rows
// (e.g. "if lf if if if if if") produced when pdftotext can't decode font maps.
func isLigaturesArtifact(s string) bool {
	words := strings.Fields(s)
	if len(words) < ligatureMinWords {
		return false
	}
	total := 0
	for _, w := range words {
		total += len(w)
	}
	return float64(total)/float64(len(words)) < ligatureMaxAvgWordLen
}

// buildPDFPageHTML generates an HTML page from structured pageItems and images.
// When both text and images are present the image floats left and text flows
// to its right. Text elements stay at body level so htmlsplit can still
// partition them across pages without creating image-less or text-less chunks.
func buildPDFPageHTML(outputDir, bookTitle string, pageNum, totalPages int, items []pageItem, images []string) string {
	hasText := len(items) > 0
	hasImages := len(images) > 0
	sideBySide := hasText && hasImages
	// One image and no text is a scanned page: it should fill the window rather than the
	// text measure. More than one image is a composed page, so leave it to the ordinary
	// column rules - guessing which of them is "the page" would be wrong as often as right.
	scanBox := ""
	if hasImages && !hasText && len(images) == 1 {
		scanBox = pageScanBox(outputDir, images[0])
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(bookTitle), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    p { margin: 0.6em 0; text-indent: 1.5em; }\n")
	sb.WriteString("    p:first-of-type { text-indent: 0; }\n")
	sb.WriteString("    h2 { text-align: center; font-size: 1.5em; font-weight: bold; letter-spacing: 0.08em; margin: 1.8em 0 1.2em; text-transform: uppercase; }\n")
	sb.WriteString("    h3 { text-align: center; font-size: 1.15em; font-style: italic; margin: 1.4em 0 0.8em; }\n")
	sb.WriteString("    .pdf-images img { max-width: 100%; height: auto; display: block; margin: 0.5em 0; }\n")
	sb.WriteString("    .pdf-images img.pdf-flip-y { transform: scaleY(-1); transform-origin: center; }\n")
	if sideBySide {
		sb.WriteString("    .pdf-images { float: left; width: 58%; margin-right: 1.5em; margin-bottom: 0.5em; }\n")
		sb.WriteString("    .pdf-float-clear { clear: both; }\n")
		sb.WriteString("    @media (max-width: 700px) { .pdf-images { float: none; width: 100%; margin-right: 0; } }\n")
	}
	sb.WriteString("  </style>\n</head>\n<body>\n")
	// Page number is shown in the injected navbar (top-right); no body header needed.

	if sideBySide {
		sb.WriteString("  <div class=\"pdf-images\">\n")
		for _, imgPath := range images {
			sb.WriteString(fmt.Sprintf("    <img src=\"%s\" loading=\"lazy\"%s>\n", html.EscapeString(imgPath), imageHTMLClassAttr(imgPath)))
		}
		sb.WriteString("  </div>\n")
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("  <%s>%s</%s>\n", item.tag, html.EscapeString(item.text), item.tag))
		}
		sb.WriteString("  <div class=\"pdf-float-clear\"></div>\n")
	} else {
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("  <%s>%s</%s>\n", item.tag, html.EscapeString(item.text), item.tag))
		}
		if hasImages {
			if scanBox != "" {
				sb.WriteString(fmt.Sprintf("  <div class=\"pdf-images pdf-page-scan\"%s>\n", scanBox))
			} else {
				sb.WriteString("  <div class=\"pdf-images\">\n")
			}
			for _, imgPath := range images {
				sb.WriteString(fmt.Sprintf("    <img src=\"%s\" loading=\"lazy\"%s>\n", html.EscapeString(imgPath), imageHTMLClassAttr(imgPath)))
			}
			sb.WriteString("  </div>\n")
		}
	}

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

// extractWithPDFLib performs text extraction with the ledongthuc/pdf reader.
func extractWithPDFLib(pdfPath, outputDir string) (book *epub.Book, err error) {
	// The underlying PDF library may panic on malformed files.
	defer func() {
		if r := recover(); r != nil {
			book = nil
			err = fmt.Errorf("pdf library panic: %v (file may be corrupted or unsupported)", r)
		}
	}()

	f, reader, err := pdflib.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer func() { _ = f.Close() }()

	totalPages := reader.NumPage()
	if totalPages == 0 {
		return nil, fmt.Errorf("pdf has no pages: %s", pdfPath)
	}

	title := pdfTitle(pdfPath)

	book = &epub.Book{
		Title:    title,
		BasePath: "", // pages at root of output dir
	}

	// Extract embedded images (non-fatal if PDF has none).
	pageImages := extractImages(pdfPath, outputDir).byPage

	// Source PDF page number -> generated href (blank pages are skipped).
	pdfPageToHref := make(map[int]string, totalPages)

	generated := 0
	withText := 0
	// This loop is silent for minutes on a long PDF: the library re-walks the page tree
	// from the root on every Page(i) call.
	textTick := logging.NewTicker("Reading text", "pages")
	for i := 1; i <= totalPages; i++ {
		textTick.Report(i-1, totalPages)
		pageContent, skip, pageErr := extractPage(reader, i)
		if pageErr != nil {
			fmt.Fprintf(os.Stderr, "WARNING: skip PDF page %d: %v\n", i, pageErr)
			continue
		}

		imgs := pageImages[i]

		hasText := true
		if skip || strings.TrimSpace(pageContent) == "" {
			if len(imgs) == 0 {
				// Truly empty page - skip
				continue
			}
			// Image-only page - include it without text
			pageContent = ""
			hasText = false
		}

		generated++
		if hasText {
			withText++
		}
		href := fmt.Sprintf("page_%03d.html", generated)
		id := fmt.Sprintf("page_%03d", generated)
		pdfPageToHref[i] = href

		pageHTML := buildPageHTML(outputDir, title, i, totalPages, pageContent, imgs)
		pagePath := filepath.Join(outputDir, href)
		if err := fsutil.WriteFile(pagePath, []byte(pageHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write page %d: %w", i, err)
		}

		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID:        id,
			Href:      href,
			MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{
			IDRef: id,
		})
	}

	if generated == 0 {
		logging.Printf("  WARNING: No text layer and no extractable page images in this PDF - there is nothing " +
			"for OCR to read. Creating a fallback page that embeds the original.\n")

		pdfCopyName := "original.pdf"
		pdfCopyPath := filepath.Join(outputDir, pdfCopyName)
		if err := copyFile(pdfPath, pdfCopyPath); err != nil {
			return nil, fmt.Errorf("prepare fallback pdf copy: %w", err)
		}

		fallbackHTML := buildFallbackPDFHTML(title, pdfCopyName)
		href := "page_001.html"
		id := "page_001"
		if err := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(fallbackHTML), 0o644); err != nil {
			return nil, fmt.Errorf("write fallback html: %w", err)
		}

		book.Manifest = append(book.Manifest, epub.ManifestItem{
			ID:        id,
			Href:      href,
			MediaType: "text/html",
		})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})

		logging.Printf("  Fallback page created: %s\n", href)
		logging.Printf("  Original PDF copied: %s\n", pdfCopyName)
		return book, nil
	}

	if !textTick.Quiet() {
		textTick.Report(totalPages, totalPages)
	}

	book.TOC = buildPDFTOC(pdfPath, pdfPageToHref)

	logging.Printf("  Title: %s\n", title)
	logPageCounts(totalPages, withText, generated)

	return book, nil
}

// tryRepairPDF attempts to normalize/rewrite malformed PDF structure so the
// text extractor can parse the document without panicking.
func tryRepairPDF(inputPath string) (string, error) {
	tmp, err := os.CreateTemp("", "doc-html-translate-repair-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create repair temp file: %w", err)
	}
	repairedPath := tmp.Name()
	_ = tmp.Close()

	if err := optimizeSafe(inputPath, repairedPath); err != nil {
		_ = os.Remove(repairedPath)
		return "", fmt.Errorf("repair pdf with pdfcpu: %w", err)
	}
	return repairedPath, nil
}

// optimizeSafe is the repair attempt behind a panic guard. The file reaching it has already
// defeated one PDF library, which is exactly the input most likely to panic the next one; a
// failed repair must end as the original extraction error, not as a crash.
func optimizeSafe(inputPath, outputPath string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Printf("  WARNING: PDF repair panicked: %v\n", r)
			logging.RunLogf("%s\n", debug.Stack())
			err = fmt.Errorf("pdfcpu panic: %v", r)
		}
	}()
	return api.OptimizeFile(inputPath, outputPath, nil)
}

// extractPage safely extracts text from a single PDF page.
// Returns (text, shouldSkip, error). Recovers from library panics.
func extractPage(reader *pdflib.Reader, pageNum int) (text string, skip bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			text = ""
			skip = false
			err = fmt.Errorf("panic on page %d: %v", pageNum, r)
		}
	}()

	page := reader.Page(pageNum)
	if page.V.IsNull() {
		return "", true, nil
	}

	rows, err := page.GetTextByRow()
	if err != nil {
		return "", false, err
	}

	t := rowsToText(rows)
	if strings.TrimSpace(t) == "" {
		return "", true, nil
	}
	return t, false, nil
}

// rowsToText converts PDF text rows into paragraphs.
//
// Paragraph detection uses two independent signals:
//  1. Y-gap > 1.5× median line spacing  — catches blank-line / chapter breaks.
//  2. First-word X > typical left margin + 8pt — catches first-line-indent style
//     books where paragraphs have no extra vertical space between them.
//
// Words within a row are space-joined to fix the "ofNate"-style merge artifact.
// Each detected paragraph is emitted as one line; buildPageHTML wraps it in <p>.
//
// NOTE: ligature characters (fi, fl, ff, …) encoded with non-standard font maps
// may appear garbled — this is a limitation of the underlying PDF text extractor.
func rowsToText(rows pdflib.Rows) string {
	type rowData struct {
		text   string
		y      float64
		firstX float64
	}

	var rd []rowData
	for _, row := range rows {
		var b strings.Builder
		for _, word := range row.Content {
			trimmedWord := strings.TrimSpace(word.S)
			if trimmedWord == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(trimmedWord)
		}
		if t := strings.TrimSpace(b.String()); t != "" {
			y, firstX := 0.0, 0.0
			if len(row.Content) > 0 {
				y = row.Content[0].Y
				firstX = row.Content[0].X
			}
			rd = append(rd, rowData{t, y, firstX})
		}
	}

	if len(rd) == 0 {
		return ""
	}

	// Median Y-gap = typical single line spacing.
	var gaps []float64
	for i := 1; i < len(rd); i++ {
		if g := math.Abs(rd[i].y - rd[i-1].y); g > 0 {
			gaps = append(gaps, g)
		}
	}
	medianGap := 12.0
	if len(gaps) > 0 {
		sort.Float64s(gaps)
		medianGap = gaps[len(gaps)/2]
	}

	// 25th-percentile of first-word X = typical left margin.
	// Using 25th percentile (not median) so that even if ~40% of rows are
	// indented paragraph openers, the baseline stays at the true left margin.
	xPos := make([]float64, len(rd))
	for i, r := range rd {
		xPos[i] = r.firstX
	}
	sort.Float64s(xPos)
	leftMargin := xPos[len(xPos)/4]
	const indentThreshold = 8.0 // points; typical indent is 18–36 pt

	// Group rows into paragraphs.
	var paragraphs [][]string
	var cur []string
	for i, r := range rd {
		if i > 0 {
			yBreak := math.Abs(r.y-rd[i-1].y) > medianGap*1.5
			xBreak := r.firstX > leftMargin+indentThreshold
			if yBreak || xBreak {
				if len(cur) > 0 {
					paragraphs = append(paragraphs, cur)
				}
				cur = nil
			}
		}
		cur = append(cur, r.text)
	}
	if len(cur) > 0 {
		paragraphs = append(paragraphs, cur)
	}

	var sb strings.Builder
	for _, para := range paragraphs {
		sb.WriteString(strings.Join(para, " "))
		sb.WriteByte('\n')
		if sb.Len() > maxPageSize {
			break
		}
	}
	return sb.String()
}

// buildPageHTML generates an HTML page from extracted PDF text and images.
// When both text and images are present the image floats left and text flows
// to its right. Text elements stay at body level so htmlsplit can still
// partition them across pages without creating image-less or text-less chunks.
func buildPageHTML(outputDir, bookTitle string, pageNum, totalPages int, text string, images []string) string {
	hasText := strings.TrimSpace(text) != ""
	hasImages := len(images) > 0
	sideBySide := hasText && hasImages
	// See buildPDFPageHTML: one image and no text is a scanned page, and it should fill
	// the window rather than the text measure.
	scanBox := ""
	if hasImages && !hasText && len(images) == 1 {
		scanBox = pageScanBox(outputDir, images[0])
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"en\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n",
		html.EscapeString(bookTitle), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    p { margin: 0.4em 0; }\n")
	sb.WriteString("    .pdf-images img { max-width: 100%; height: auto; display: block; margin: 0.5em 0; }\n")
	sb.WriteString("    .pdf-images img.pdf-flip-y { transform: scaleY(-1); transform-origin: center; }\n")
	if sideBySide {
		sb.WriteString("    .pdf-images { float: left; width: 58%; margin-right: 1.5em; margin-bottom: 0.5em; }\n")
		sb.WriteString("    .pdf-float-clear { clear: both; }\n")
		sb.WriteString("    @media (max-width: 700px) { .pdf-images { float: none; width: 100%; margin-right: 0; } }\n")
	}
	sb.WriteString("  </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	// Page number is shown in the injected navbar (top-right); no body header needed.

	if sideBySide {
		sb.WriteString("  <div class=\"pdf-images\">\n")
		for _, imgPath := range images {
			sb.WriteString(fmt.Sprintf("    <img src=\"%s\" loading=\"lazy\"%s>\n", html.EscapeString(imgPath), imageHTMLClassAttr(imgPath)))
		}
		sb.WriteString("  </div>\n")
		lines := strings.Split(strings.TrimSpace(text), "\n")
		for _, line := range lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(trimmed)))
			}
		}
		sb.WriteString("  <div class=\"pdf-float-clear\"></div>\n")
	} else {
		if hasText {
			lines := strings.Split(strings.TrimSpace(text), "\n")
			for _, line := range lines {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(trimmed)))
				}
			}
		}
		if hasImages {
			if scanBox != "" {
				sb.WriteString(fmt.Sprintf("  <div class=\"pdf-images pdf-page-scan\"%s>\n", scanBox))
			} else {
				sb.WriteString("  <div class=\"pdf-images\">\n")
			}
			for _, imgPath := range images {
				sb.WriteString(fmt.Sprintf("    <img src=\"%s\" loading=\"lazy\"%s>\n", html.EscapeString(imgPath), imageHTMLClassAttr(imgPath)))
			}
			sb.WriteString("  </div>\n")
		}
	}

	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")
	return sb.String()
}

// pageScanBox sizes the box holding a page that is nothing but a scan.
//
// Such a page is a page, not an illustration sitting in a column of text: held to the
// reading measure it renders at roughly half the window, which is why it ends up being
// read at browser zoom instead. This gives it the window, bounded three ways - the
// image's own pixel width (never upscaled into mush), the window's width, and the
// window's height via the image's aspect ratio - so one scanned page lands on about one
// screen and scrolling moves page by page.
//
// It sizes the *box*, never the image: the OCR overlay positions its plates as
// percentages of a container carrying the image's aspect ratio, so bounding the image's
// height directly would clamp the container and drift every plate off the text it covers.
// Returns "" when the size can't be read, leaving the page to the ordinary rules.
func pageScanBox(outputDir, imgHref string) string {
	w, h := imageSize(filepath.Join(outputDir, filepath.FromSlash(imgHref)))
	if w <= 0 || h <= 0 {
		return ""
	}
	// height = width / (w/h), so height <= scanMaxVH  =>  width <= scanMaxVH * (w/h).
	const scanMaxVH = 92.0 // leaves room for the navigation bar
	byHeight := scanMaxVH * float64(w) / float64(h)
	return fmt.Sprintf(` style="width:min(%dpx,96vw,%.1fvh)"`, w, byHeight)
}

// imageSize reads just the header of an image file for its pixel dimensions.
func imageSize(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func imageHTMLClassAttr(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".tif" || ext == ".tiff" {
		return ` class="pdf-flip-y"`
	}
	return ""
}

// pdfTitle is the file name without its extension. Both separators count, as they
// do on Windows, so the result does not depend on the OS the code runs on
// (filepath.Base on Linux keeps `C:\Books\` as part of the name).
func pdfTitle(pdfPath string) string {
	base := pdfPath[strings.LastIndexAny(pdfPath, `/\`)+1:]
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func buildFallbackPDFHTML(title, pdfFileName string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"en\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Segoe UI, Arial, sans-serif; margin: 1rem; }\n")
	sb.WriteString("    .note { background: #fff8e1; border: 1px solid #f0d98c; padding: 0.8rem; margin-bottom: 1rem; border-radius: 6px; }\n")
	sb.WriteString("    .viewer { width: 100%; height: 90vh; border: 1px solid #ddd; border-radius: 6px; }\n")
	sb.WriteString("  </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	// The sentence says what this page knows and nothing else. It used to read "OCR is disabled, so
	// the original PDF is shown as-is", which is a literal rather than a reading of the flag: a run
	// with -ocr printed it too, telling the reader the feature they had just switched on was off
	// (DEV/research/ocr_sweep_2026-08-13.md defect 4). The fact that actually decides the outcome is
	// on this page's own branch - it is reached only when neither a text layer nor a single page
	// image could be extracted - and it holds whichever way the flag is set, so the page states that
	// instead of guessing at a setting it was never given.
	sb.WriteString("  <div class=\"note\">This PDF has no text layer, and no page images could be extracted from it - " +
		"so there is nothing to convert, and nothing for OCR to read. The original PDF is shown below as-is.</div>\n")
	sb.WriteString(fmt.Sprintf("  <embed class=\"viewer\" src=\"%s\" type=\"application/pdf\">\n", html.EscapeString(pdfFileName)))
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")
	return sb.String()
}

func needsPDFToTextPathStaging(path string) bool {
	for _, r := range path {
		if r > 127 {
			return true
		}
	}
	return false
}

func stagePDFForPDFToText(srcPath string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "doc-html-translate-pdftotext-")
	if err != nil {
		return "", nil, err
	}

	stagedPath := filepath.Join(tempDir, "input.pdf")
	if err := copyFile(srcPath, stagedPath); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}
	return stagedPath, cleanup, nil
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Sync()
}
