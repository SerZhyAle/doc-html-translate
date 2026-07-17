// Package img handles a standalone image file (PNG / JPG / WebP / ..) as input.
// The image itself carries no machine-readable text, so there is nothing to
// "extract" in the usual sense: instead the picture is wrapped in a one-page
// HTML document and the pipeline's OCR overlay step (internal/ocr) digitizes the
// text and lays translatable plates over it, positioned to match the source.
// Opened in Chrome, the page shows the original image with the built-in "Translate
// to.." working on the overlaid text - the same behaviour as the browser extension.
package img

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"html"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/tiff"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
)

// exts are the standalone image extensions (lower-case, leading dot) treated as
// image input. tesseract reads all of these directly; the overlay's colour
// sampling has Go decoders for every one except .bmp (which falls back to plain
// white plates - still correct, just not colour-matched).
//
// TIFF is a special case: Chrome cannot display it, so a .tif/.tiff is transcoded
// to PNG on the way in rather than copied (see extractTIFF). It stays in this set
// because it is still accepted input - it is just converted, not passed through.
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

// Extract wraps a standalone image into an *epub.Book the pipeline can run through
// its single-page flow; the caller forces the OCR overlay step, which finds each
// <img>, OCRs it, and appends translatable text plates.
//
// A browser-renderable image (PNG/JPEG/GIF/BMP/WebP) is copied through untouched.
// TIFF is transcoded to PNG because Chrome - the app's whole reason to exist is
// "open the result in Chrome" - cannot decode TIFF, so a copied .tif rendered as a
// broken-image icon with plates floating over white space. A multi-page TIFF becomes
// one PNG page per frame, so its pages are not piled onto a single unshowable image.
func Extract(imgPath, outputDir string) (*epub.Book, error) {
	title := strings.TrimSuffix(filepath.Base(imgPath), filepath.Ext(imgPath))
	if ext := strings.ToLower(filepath.Ext(imgPath)); ext == ".tif" || ext == ".tiff" {
		return extractTIFF(imgPath, outputDir, title)
	}

	imgName := filepath.Base(imgPath)
	if err := copyFile(imgPath, filepath.Join(outputDir, imgName)); err != nil {
		return nil, fmt.Errorf("copy image: %w", err)
	}

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

// extractTIFF decodes every frame of a TIFF, writes each as a PNG, and gives each its
// own page. One frame is one page; a multi-page TIFF (a scanned document, a fax)
// therefore reads page by page instead of stacking every frame's OCR plates onto a
// single image the browser will not draw.
func extractTIFF(imgPath, outputDir, title string) (*epub.Book, error) {
	data, err := os.ReadFile(imgPath)
	if err != nil {
		return nil, fmt.Errorf("read tiff: %w", err)
	}
	offsets, order, err := tiffFrameOffsets(data)
	if err != nil {
		return nil, fmt.Errorf("parse tiff: %w", err)
	}

	book := &epub.Book{Title: title}
	for i, off := range offsets {
		frame, derr := decodeTIFFFrame(data, order, off)
		if derr != nil {
			// A frame we cannot decode is dropped with a named reason rather than
			// aborting the whole document - the same best-effort contract the rest of
			// the pipeline keeps.
			logging.Printf("  WARNING: TIFF frame %d could not be decoded, skipped: %v\n", i+1, derr)
			continue
		}
		pageNum := len(book.Spine) + 1
		pngName := fmt.Sprintf("page_%03d.png", pageNum)
		if werr := encodePNG(filepath.Join(outputDir, pngName), frame); werr != nil {
			return nil, fmt.Errorf("write tiff frame %d as png: %w", i+1, werr)
		}
		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)
		if werr := os.WriteFile(filepath.Join(outputDir, href), []byte(buildPageHTML(title, pngName)), 0o644); werr != nil {
			return nil, fmt.Errorf("write tiff page %d: %w", pageNum, werr)
		}
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "text/html"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}
	if len(book.Spine) == 0 {
		return nil, fmt.Errorf("no decodable frames in TIFF")
	}

	logging.Printf("  Title: %s\n", title)
	if len(book.Spine) == 1 {
		logging.Printf("  Image: %s (TIFF transcoded to PNG so the browser can show it)\n", filepath.Base(imgPath))
	} else {
		logging.Printf("  Image: %s (%d-page TIFF, one PNG page per frame)\n", filepath.Base(imgPath), len(book.Spine))
	}
	return book, nil
}

// tiffFrameOffsets walks the IFD chain and returns the file offset of each frame's
// image file directory, plus the file's byte order. golang.org/x/image/tiff.Decode
// reads only the first IFD, so to reach later frames we need their offsets - and a
// TIFF's structure hands them over cheaply: an 8-byte header points at the first IFD,
// and every IFD ends with the offset of the next (0 = last).
func tiffFrameOffsets(data []byte) ([]uint32, binary.ByteOrder, error) {
	if len(data) < 8 {
		return nil, nil, fmt.Errorf("too short for a TIFF header")
	}
	var order binary.ByteOrder
	switch {
	case data[0] == 'I' && data[1] == 'I':
		order = binary.LittleEndian
	case data[0] == 'M' && data[1] == 'M':
		order = binary.BigEndian
	default:
		return nil, nil, fmt.Errorf("not a TIFF: bad byte-order mark")
	}
	if order.Uint16(data[2:4]) != 42 {
		return nil, nil, fmt.Errorf("not a TIFF: bad magic number")
	}

	var offsets []uint32
	seen := make(map[uint32]bool) // a malformed file could point an IFD at itself
	off := order.Uint32(data[4:8])
	for off != 0 {
		if int(off)+2 > len(data) || seen[off] {
			break
		}
		seen[off] = true
		offsets = append(offsets, off)
		entries := int(order.Uint16(data[off : off+2]))
		next := int(off) + 2 + entries*12
		if next+4 > len(data) {
			break
		}
		off = order.Uint32(data[next : next+4])
		if len(offsets) >= 4096 { // sanity cap; no real document has this many frames
			break
		}
	}
	if len(offsets) == 0 {
		return nil, nil, fmt.Errorf("no image frames found")
	}
	return offsets, order, nil
}

// decodeTIFFFrame decodes the frame whose IFD sits at ifdOffset. It copies the file
// and repoints the header's first-IFD offset at that frame, so Decode returns it: the
// strip/tile offsets inside the IFD are absolute file offsets and stay valid because
// the whole file is preserved. This reuses the standard decoder for every frame
// instead of reimplementing TIFF.
func decodeTIFFFrame(data []byte, order binary.ByteOrder, ifdOffset uint32) (image.Image, error) {
	buf := make([]byte, len(data))
	copy(buf, data)
	order.PutUint32(buf[4:8], ifdOffset)
	return tiff.Decode(bytes.NewReader(buf))
}

// encodePNG encodes img to a PNG file.
func encodePNG(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(out, img); err != nil {
		out.Close()
		return err
	}
	return out.Close()
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
