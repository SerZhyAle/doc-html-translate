// Package img handles a standalone image file (PNG / JPG / WebP / ..) as input.
// The image itself carries no machine-readable text, so there is nothing to
// "extract" in the usual sense: instead the picture is wrapped in a one-page
// HTML document and the pipeline's OCR overlay step (internal/ocr) digitizes the
// text and lays translatable plates over it, positioned to match the source.
// Opened in Chrome, the page shows the original image with the built-in "Translate
// to.." working on the overlaid text - the same behaviour as the browser extension.
package img

import (
	"encoding/binary"
	"errors"
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
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/limits"
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

	pageHTML := buildPageHTML(title, imgName, 1, 1)
	if err := fsutil.WriteFile(filepath.Join(outputDir, "page_001.html"), []byte(pageHTML), 0o644); err != nil {
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
	f, err := os.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("read tiff: %w", err)
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("read tiff: %w", err)
	}
	offsets, order, err := tiffFrameOffsets(f, st.Size())
	if err != nil {
		return nil, fmt.Errorf("parse tiff: %w", err)
	}

	book := &epub.Book{Title: title}
	var budgetErr error
	for i, off := range offsets {
		frame, derr := decodeTIFFFrame(f, st.Size(), order, off)
		if derr != nil {
			// A frame we cannot decode is dropped with a named reason rather than
			// aborting the whole document - the same best-effort contract the rest of
			// the pipeline keeps.
			logging.Printf("  WARNING: TIFF frame %d could not be decoded, skipped: %v\n", i+1, derr)
			if errors.Is(derr, limits.ErrTooLarge) {
				budgetErr = derr
			}
			continue
		}
		pageNum := len(book.Spine) + 1
		pngName := fmt.Sprintf("page_%03d.png", pageNum)
		if werr := encodePNG(filepath.Join(outputDir, pngName), frame); werr != nil {
			return nil, fmt.Errorf("write tiff frame %d as png: %w", i+1, werr)
		}
		href := fmt.Sprintf("page_%03d.html", pageNum)
		id := fmt.Sprintf("page_%03d", pageNum)
		// len(offsets) is the frame count before any undecodable frame is dropped, so it
		// can overstate the total by the number skipped. That only affects the alt text,
		// and an over-count reads better than renumbering pages after the fact.
		if werr := fsutil.WriteFile(filepath.Join(outputDir, href), []byte(buildPageHTML(title, pngName, pageNum, len(offsets))), 0o644); werr != nil {
			return nil, fmt.Errorf("write tiff page %d: %w", pageNum, werr)
		}
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "text/html"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}
	if len(book.Spine) == 0 {
		if budgetErr != nil {
			// Name the limit rather than "no decodable frames": the file is fine, only too big.
			return nil, budgetErr
		}
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

// maxTIFFFrames is a sanity cap on the IFD walk; no real document has this many frames.
const maxTIFFFrames = 4096

// tiffFrameOffsets walks the IFD chain and returns the file offset of each frame's
// image file directory, plus the file's byte order. golang.org/x/image/tiff.Decode
// reads only the first IFD, so to reach later frames we need their offsets - and a
// TIFF's structure hands them over cheaply: an 8-byte header points at the first IFD,
// and every IFD ends with the offset of the next (0 = last).
//
// Offsets are uint32 in the file and are compared against the size in 64 bits: on the
// 386 build int(off) of an offset past 2 GB goes negative, passes a "< len" test and
// panics on the slice that follows.
func tiffFrameOffsets(r io.ReaderAt, size int64) ([]uint32, binary.ByteOrder, error) {
	var hdr [8]byte
	if size < int64(len(hdr)) {
		return nil, nil, fmt.Errorf("too short for a TIFF header")
	}
	if _, err := r.ReadAt(hdr[:], 0); err != nil {
		return nil, nil, err
	}
	var order binary.ByteOrder
	switch {
	case hdr[0] == 'I' && hdr[1] == 'I':
		order = binary.LittleEndian
	case hdr[0] == 'M' && hdr[1] == 'M':
		order = binary.BigEndian
	default:
		return nil, nil, fmt.Errorf("not a TIFF: bad byte-order mark")
	}
	if order.Uint16(hdr[2:4]) != 42 {
		return nil, nil, fmt.Errorf("not a TIFF: bad magic number")
	}

	var offsets []uint32
	seen := make(map[uint32]bool) // a malformed file could point an IFD at itself
	var buf [4]byte
	off := order.Uint32(hdr[4:8])
	for off != 0 && len(offsets) < maxTIFFFrames {
		if int64(off)+2 > size || seen[off] {
			break
		}
		seen[off] = true
		offsets = append(offsets, off)
		if _, err := r.ReadAt(buf[:2], int64(off)); err != nil {
			break
		}
		next := int64(off) + 2 + int64(order.Uint16(buf[:2]))*12
		if next+4 > size {
			break
		}
		if _, err := r.ReadAt(buf[:4], next); err != nil {
			break
		}
		off = order.Uint32(buf[:4])
	}
	if len(offsets) == 0 {
		return nil, nil, fmt.Errorf("no image frames found")
	}
	return offsets, order, nil
}

// frameView presents the file as if its header pointed at one chosen IFD, so Decode
// returns that frame. The strip/tile offsets inside the IFD are absolute file offsets and
// stay valid because the file itself is untouched; only the four bytes of the first-IFD
// pointer are substituted. Reading through the file instead of copying it per frame keeps
// a 4096-frame fax at one file's worth of memory, not 4096 copies of it.
type frameView struct {
	r   io.ReaderAt
	ifd [4]byte
}

func (v *frameView) ReadAt(p []byte, off int64) (int, error) {
	n, err := v.r.ReadAt(p, off)
	for i := 4; i < 8; i++ {
		if j := int64(i) - off; j >= 0 && j < int64(n) {
			p[j] = v.ifd[i-4]
		}
	}
	return n, err
}

// decodeTIFFFrame decodes the frame whose IFD sits at ifdOffset, after a header-only
// probe of its declared size: a 1 KB file can declare 60000 x 60000, and Decode would
// then allocate the whole raster before reading a single pixel.
func decodeTIFFFrame(r io.ReaderAt, size int64, order binary.ByteOrder, ifdOffset uint32) (image.Image, error) {
	if w, h, ok := limits.TIFFFrameSize(r, size, order, ifdOffset); ok {
		if err := limits.CheckPixels(w, h); err != nil {
			return nil, err
		}
	}
	view := &frameView{r: r}
	order.PutUint32(view.ifd[:], ifdOffset)
	return tiff.Decode(io.NewSectionReader(view, 0, size))
}

// encodePNG encodes img to a PNG file.
func encodePNG(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(out, img); err != nil {
		_ = out.Close()
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
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// buildPageHTML wraps the copied image in a minimal centred page. The OCR overlay
// step rewrites the <img> into a positioned container with text plates; if OCR is
// unavailable the page still shows the image unchanged.
//
// The wrapper is a <section id="page_NNN">, not a <main>, and the reason is the
// merge: a multi-frame TIFF's pages are concatenated into one index.html, where a
// <main> per page would nest inside the merged document's own <main>. Invalid HTML,
// and it breaks reader mode and landmark navigation. Kept identical to
// internal/comic's wrapper so both inputs overlay the same way.
func buildPageHTML(title, imgName string, pageNum, totalPages int) string {
	alt := html.EscapeString(title)
	if totalPages > 1 {
		alt = html.EscapeString(fmt.Sprintf("%s - page %d of %d", title, pageNum, totalPages))
	}
	var sb strings.Builder
	// No lang: the source declares none, and a guessed one can stop Chrome offering
	// "Translate page"; without it Chrome detects the language itself.
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { margin: 0; }\n")
	sb.WriteString("    section.dht-page { width: 95%; max-width: 1400px; margin: 1em auto; }\n")
	sb.WriteString("    section.dht-page img { display: block; width: 100%; height: auto; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString(fmt.Sprintf("  <section class=\"dht-page\" id=\"page_%03d\" aria-label=\"%s\">\n", pageNum, alt))
	// The source file name is the user's: "scan#1.png" or "50%.png" must be a path, not a
	// fragment or a broken escape.
	sb.WriteString(fmt.Sprintf("    <img src=\"%s\" alt=\"%s\">\n", html.EscapeString(epub.URLPath(imgName)), alt))
	sb.WriteString("  </section>\n</body>\n</html>\n")
	return sb.String()
}
