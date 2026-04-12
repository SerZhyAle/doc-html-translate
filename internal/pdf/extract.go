// Package pdf handles text extraction from PDF files and conversion to HTML pages.
// Uses github.com/ledongthuc/pdf for text extraction.
// Only supports text-based PDFs; OCR for scanned/image PDFs is a future TODO.
package pdf

import (
	"fmt"
	"html"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/textutil"

	pdflib "github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/image/tiff"
)

// maxPageSize is the safety limit per page text (10 MB).
const maxPageSize = 10 * 1024 * 1024

// findPDFToText locates the pdftotext binary via PATH or well-known install paths.
func findPDFToText() string {
	if p, err := exec.LookPath("pdftotext"); err == nil {
		return p
	}
	for _, p := range []string{
		`C:\Program Files\Git\mingw64\bin\pdftotext.exe`,
		`C:\Program Files (x86)\Git\mingw64\bin\pdftotext.exe`,
		`C:\Program Files\Xpdf\bin64\pdftotext.exe`,
		`C:\Program Files\poppler\bin\pdftotext.exe`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

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
	defer os.Remove(repairedPath)

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

	cmd := exec.Command(pdftotextBin, "-layout", "-enc", "UTF-8", pdfPathForTool, "-")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext: %w", err)
	}

	text := textutil.NormalizeLineSeparatorsPreserveFormFeed(string(out))

	// Pages are separated by form-feed \f
	pageTexts := strings.Split(text, "\f")
	for len(pageTexts) > 0 && strings.TrimSpace(pageTexts[len(pageTexts)-1]) == "" {
		pageTexts = pageTexts[:len(pageTexts)-1]
	}
	if len(pageTexts) == 0 {
		return nil, fmt.Errorf("pdftotext extracted no content")
	}

	title := pdfTitle(pdfPath)
	totalPages := len(pageTexts)
	book := &epub.Book{Title: title}

	pageImages := extractImages(pdfPath, outputDir, totalPages)

	generated := 0
	for i, rawPage := range pageTexts {
		pdfPageNum := i + 1
		imgs := pageImages[pdfPageNum]

		items := parsePDFLayoutPage(rawPage)
		if len(items) == 0 && len(imgs) == 0 {
			continue
		}

		generated++
		href := fmt.Sprintf("page_%03d.html", generated)
		id := fmt.Sprintf("page_%03d", generated)

		pageHTML := buildPDFPageHTML(title, pdfPageNum, totalPages, items, imgs)
		if err := os.WriteFile(filepath.Join(outputDir, href), []byte(pageHTML), 0o644); err != nil {
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

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Pages: %d (with text: %d)\n", totalPages, generated)
	return book, nil
}

// parsePDFLayoutPage parses one page from pdftotext -layout output into pageItems.
//
// In -layout mode:
//   - blank lines (\n\n) separate visual blocks (paragraphs / headings)
//   - leading spaces on the first line of a block indicate centering (more spaces = more centered)
//   - body paragraph first lines have ~1 space indent; centered headings have 8+ spaces
//   - lines within a block are word-wrapped and must be re-joined with spaces
func parsePDFLayoutPage(text string) []pageItem {
	var items []pageItem

	for _, block := range strings.Split(text, "\n\n") {
		lines := strings.Split(block, "\n")

		// Collect non-empty lines and measure leading indent of the first one.
		leadingSpaces := 0
		firstSeen := false
		var parts []string
		for _, line := range lines {
			// Strip trailing whitespace only; keep leading for indent measurement.
			rline := strings.TrimRight(line, " \t\r")
			if rline == "" {
				continue
			}
			if !firstSeen {
				leadingSpaces = len(rline) - len(strings.TrimLeft(rline, " \t"))
				firstSeen = true
			}
			parts = append(parts, strings.TrimSpace(rline))
		}

		if len(parts) == 0 {
			continue
		}

		joined := strings.Join(parts, " ")
		if isLigaturesArtifact(joined) {
			continue
		}

		tag := classifyBlock(joined, leadingSpaces)
		items = append(items, pageItem{joined, tag})
	}

	return items
}

// classifyBlock assigns an HTML tag based on text content and indentation.
//
// Centering heuristic: pdftotext -layout right-pads lines to the page width.
// A centered heading like "LITTLE TOKYO" gets ~20 spaces of left indent.
// Body paragraph first lines get ~1 space (first-line indent).
// leadingSpaces > 8 = centered = heading candidate.
func classifyBlock(text string, leadingSpaces int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "p"
	}

	upper := strings.ToUpper(text)
	isAllCaps := upper == text && strings.ContainsAny(text, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	isCentered := leadingSpaces > 8
	isShort := len(words) <= 8

	switch {
	case (isAllCaps || isCentered) && isShort:
		return "h2"
	case isCentered && len(words) <= 14:
		return "h3"
	default:
		return "p"
	}
}

// isLigaturesArtifact returns true for lines that are ligature-garbage rows
// (e.g. "if lf if if if if if") produced when pdftotext can't decode font maps.
func isLigaturesArtifact(s string) bool {
	words := strings.Fields(s)
	if len(words) < 4 {
		return false
	}
	total := 0
	for _, w := range words {
		total += len(w)
	}
	return float64(total)/float64(len(words)) < 3.0
}

// buildPDFPageHTML generates an HTML page from structured pageItems and images.
func buildPDFPageHTML(bookTitle string, pageNum, totalPages int, items []pageItem, images []string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s — Page %d</title>\n", html.EscapeString(bookTitle), pageNum))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; line-height: 1.6; }\n")
	sb.WriteString("    .pdf-page-header { color: #666; font-size: 0.9em; border-bottom: 1px solid #eee; padding-bottom: 0.5em; margin-bottom: 1.5em; }\n")
	sb.WriteString("    p { margin: 0.6em 0; text-indent: 1.5em; }\n")
	sb.WriteString("    p:first-of-type { text-indent: 0; }\n")
	sb.WriteString("    h2 { text-align: center; font-size: 1.5em; font-weight: bold; letter-spacing: 0.08em; margin: 1.8em 0 1.2em; text-transform: uppercase; }\n")
	sb.WriteString("    h3 { text-align: center; font-size: 1.15em; font-style: italic; margin: 1.4em 0 0.8em; }\n")
	sb.WriteString("    .pdf-images img { max-width: 100%; height: auto; display: block; margin: 0.5em 0; }\n")
	sb.WriteString("    .pdf-images img.pdf-flip-y { transform: scaleY(-1); transform-origin: center; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(fmt.Sprintf("  <div class=\"pdf-page-header\">Page %d / %d</div>\n", pageNum, totalPages))

	for _, item := range items {
		sb.WriteString(fmt.Sprintf("  <%s>%s</%s>\n", item.tag, html.EscapeString(item.text), item.tag))
	}

	if len(images) > 0 {
		sb.WriteString("  <div class=\"pdf-images\">\n")
		for _, imgPath := range images {
			sb.WriteString(fmt.Sprintf("    <img src=\"%s\" loading=\"lazy\"%s>\n", html.EscapeString(imgPath), imageHTMLClassAttr(imgPath)))
		}
		sb.WriteString("  </div>\n")
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
	defer f.Close()

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
	pageImages := extractImages(pdfPath, outputDir, totalPages)

	generated := 0
	for i := 1; i <= totalPages; i++ {
		pageContent, skip, pageErr := extractPage(reader, i)
		if pageErr != nil {
			fmt.Fprintf(os.Stderr, "WARNING: skip PDF page %d: %v\n", i, pageErr)
			continue
		}

		imgs := pageImages[i]

		if skip || strings.TrimSpace(pageContent) == "" {
			if len(imgs) == 0 {
				// Truly empty page — skip
				continue
			}
			// Image-only page — include it without text
			pageContent = ""
		}

		generated++
		href := fmt.Sprintf("page_%03d.html", generated)
		id := fmt.Sprintf("page_%03d", generated)

		pageHTML := buildPageHTML(title, i, totalPages, pageContent, imgs)
		pagePath := filepath.Join(outputDir, href)
		if err := os.WriteFile(pagePath, []byte(pageHTML), 0o644); err != nil {
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
		logging.Printf("  WARNING: No text layer detected in PDF. Creating a fallback HTML page without OCR.\n")

		pdfCopyName := "original.pdf"
		pdfCopyPath := filepath.Join(outputDir, pdfCopyName)
		if err := copyFile(pdfPath, pdfCopyPath); err != nil {
			return nil, fmt.Errorf("prepare fallback pdf copy: %w", err)
		}

		fallbackHTML := buildFallbackPDFHTML(title, pdfCopyName)
		href := "page_001.html"
		id := "page_001"
		if err := os.WriteFile(filepath.Join(outputDir, href), []byte(fallbackHTML), 0o644); err != nil {
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

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Pages: %d (with text: %d)\n", totalPages, generated)

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

	if err := api.OptimizeFile(inputPath, repairedPath, nil); err != nil {
		_ = os.Remove(repairedPath)
		return "", fmt.Errorf("repair pdf with pdfcpu: %w", err)
	}
	return repairedPath, nil
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
func buildPageHTML(bookTitle string, pageNum, totalPages int, text string, images []string) string {
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
	sb.WriteString("    .pdf-page-header { color: #666; font-size: 0.9em; border-bottom: 1px solid #eee; padding-bottom: 0.5em; margin-bottom: 1em; }\n")
	sb.WriteString("    p { margin: 0.4em 0; }\n")
	sb.WriteString("    .pdf-images img { max-width: 100%; height: auto; display: block; margin: 0.5em 0; }\n")
	sb.WriteString("    .pdf-images img.pdf-flip-y { transform: scaleY(-1); transform-origin: center; }\n")
	sb.WriteString("  </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString(fmt.Sprintf("  <div class=\"pdf-page-header\">Page %d / %d</div>\n", pageNum, totalPages))

	// Convert text lines to paragraphs
	if strings.TrimSpace(text) != "" {
		lines := strings.Split(strings.TrimSpace(text), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(trimmed)))
			}
		}
	}

	// Embed extracted images
	if len(images) > 0 {
		sb.WriteString("  <div class=\"pdf-images\">\n")
		for _, imgPath := range images {
			sb.WriteString(fmt.Sprintf("    <img src=\"%s\" loading=\"lazy\"%s>\n", html.EscapeString(imgPath), imageHTMLClassAttr(imgPath)))
		}
		sb.WriteString("  </div>\n")
	}

	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")
	return sb.String()
}

// extractImages extracts all embedded images from the PDF using pdfcpu into
// outputDir/pdf_images/ and returns a map of PDF page number -> relative image paths.
// Images are extracted one page at a time to limit peak memory usage.
// Returns nil map (non-fatal) if no images exist or extraction fails.
func extractImages(pdfPath, outputDir string, totalPages int) map[int][]string {
	imagesSubdir := "pdf_images"
	imagesDir := filepath.Join(outputDir, imagesSubdir)
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		logging.Printf("  WARNING: could not create images dir: %v\n", err)
		return nil
	}

	// Extract page by page to limit peak memory — decompressing all images at
	// once can exhaust memory for PDFs with large embedded images.
	anyExtracted := false
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		pageStr := strconv.Itoa(pageNum)
		if err := api.ExtractImagesFile(pdfPath, imagesDir, []string{pageStr}, nil); err != nil {
			// Per-page errors are expected for pages without images — skip silently.
			continue
		}
		anyExtracted = true
	}

	if !anyExtracted {
		logging.Printf("  NOTE: no images extracted from PDF\n")
		return nil
	}

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		logging.Printf("  WARNING: could not read images dir: %v\n", err)
		return nil
	}
	if err := normalizeExtractedPDFImages(imagesDir, entries); err != nil {
		logging.Printf("  WARNING: could not normalize extracted PDF images: %v\n", err)
	}

	pageImages := make(map[int][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		pageNum := parseImagePageNum(name, totalPages)
		if pageNum > 0 {
			pageImages[pageNum] = append(pageImages[pageNum], imagesSubdir+"/"+name)
		}
	}

	if len(pageImages) > 0 {
		total := 0
		for _, imgs := range pageImages {
			total += len(imgs)
		}
		logging.Printf("  Images: %d extracted across %d pages\n", total, len(pageImages))
	}

	return pageImages
}

func normalizeExtractedPDFImages(imagesDir string, entries []os.DirEntry) error {
	var firstErr error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(imagesDir, entry.Name())
		if err := flipImageFileVertically(path); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", entry.Name(), err)
		}
	}
	return firstErr
}

func imageHTMLClassAttr(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".tif" || ext == ".tiff" {
		return ` class="pdf-flip-y"`
	}
	return ""
}

// flipImageFileVertically vertically flips a TIFF image file in place.
// JPEG and PNG images extracted by pdfcpu are raw embedded streams and are
// already correctly oriented; only TIFFs (reconstructed from raw PDF pixel
// data, which uses a bottom-up Y axis) need a Y-flip.
func flipImageFileVertically(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".tif" && ext != ".tiff" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	var src image.Image
	src, err = tiff.Decode(file)
	if err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, bounds.Min.Y+bounds.Max.Y-1-y, src.At(x, y))
		}
	}

	tmpPath := path + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	success := false
	defer func() {
		out.Close()
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	err = tiff.Encode(out, dst, nil)
	if err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}

// parseImagePageNum extracts the PDF page number from a pdfcpu image filename.
// pdfcpu names files: {baseName}_{pageNr}_{identifier}.{ext}
// We find the first numeric-only segment (after position 0) that is in [1, totalPages].
func parseImagePageNum(filename string, totalPages int) int {
	base := filename
	if dot := strings.LastIndex(base, "."); dot >= 0 {
		base = base[:dot]
	}
	parts := strings.Split(base, "_")
	// Try segments from index 1, skip the last (it's the identifier, not a number)
	for i := 1; i < len(parts)-1; i++ {
		n, err := strconv.Atoi(parts[i])
		if err == nil && n >= 1 && n <= totalPages {
			return n
		}
	}
	// Also try last segment in case format differs
	if len(parts) >= 2 {
		n, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil && n >= 1 && n <= totalPages {
			return n
		}
	}
	return 0
}

// pdfTitle extracts a human-readable title from the PDF file path.
func pdfTitle(pdfPath string) string {
	base := filepath.Base(pdfPath)
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
	sb.WriteString("  <div class=\"note\">No extractable text layer was found in this PDF. OCR is disabled, so the original PDF is shown as-is.</div>\n")
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
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Sync()
}
