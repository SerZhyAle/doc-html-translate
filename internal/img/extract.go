// Package img handles a standalone image file (PNG / JPG / WebP / ..) as input.
// The image itself carries no machine-readable text, so there is nothing to
// "extract" in the usual sense: instead the picture is wrapped in a one-page
// HTML document and the pipeline's OCR overlay step (internal/ocr) digitizes the
// text and lays translatable plates over it, positioned to match the source.
// Opened in Chrome, the page shows the original image with the built-in "Translate
// to.." working on the overlaid text - the same behaviour as the browser extension.
package img

import (
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
)

// exts are the standalone image extensions (lower-case, leading dot) treated as
// image input. tesseract reads all of these directly; the overlay's colour
// sampling has Go decoders for every one except .bmp (which falls back to plain
// white plates - still correct, just not colour-matched).
var exts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".bmp":  true,
	".tif":  true,
	".tiff": true,
}

// IsImage reports whether ext (lower-case, with leading dot) is a supported
// standalone image input.
func IsImage(ext string) bool { return exts[ext] }

// Extract copies the source image into outputDir and writes a single-page
// page_001.html that references it. It returns a one-spine *epub.Book so the
// pipeline runs its single-page flow; the caller forces the OCR overlay step,
// which finds the <img>, OCRs it, and appends translatable text plates.
func Extract(imgPath, outputDir string) (*epub.Book, error) {
	imgName := filepath.Base(imgPath)
	if err := copyFile(imgPath, filepath.Join(outputDir, imgName)); err != nil {
		return nil, fmt.Errorf("copy image: %w", err)
	}

	title := strings.TrimSuffix(imgName, filepath.Ext(imgName))
	pageHTML := buildPageHTML(title, imgName)
	if err := os.WriteFile(filepath.Join(outputDir, "page_001.html"), []byte(pageHTML), 0o644); err != nil {
		return nil, fmt.Errorf("write page: %w", err)
	}

	book := &epub.Book{
		Title: title,
		Manifest: []epub.ManifestItem{
			{ID: "page_001", Href: "page_001.html", MediaType: "text/html"},
		},
		Spine: []epub.SpineItem{{IDRef: "page_001"}},
	}

	logging.Printf("  Title: %s\n", title)
	logging.Printf("  Image: %s\n", imgName)
	return book, nil
}

// copyFile copies src to dst byte-for-byte.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// buildPageHTML wraps the copied image in a minimal centred page. The OCR overlay
// step rewrites the <img> into a positioned container with text plates; if OCR is
// unavailable the page still shows the image unchanged.
func buildPageHTML(title, imgName string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { margin: 0; background: #f0f0f0; }\n")
	sb.WriteString("    main { width: 95%; max-width: 1400px; margin: 1em auto; }\n")
	sb.WriteString("    main img { display: block; width: 100%; height: auto; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString("  <main>\n")
	sb.WriteString(fmt.Sprintf("    <img src=\"%s\" alt=\"%s\">\n", html.EscapeString(imgName), html.EscapeString(title)))
	sb.WriteString("  </main>\n</body>\n</html>\n")
	return sb.String()
}
