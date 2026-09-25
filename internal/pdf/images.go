package pdf

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/procrun"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpulib "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
	"golang.org/x/image/tiff"
)

// pdfImages is what the image pass learned about a document.
type pdfImages struct {
	// byPage maps a 1-based PDF page number to the images written for it, as paths
	// relative to the output directory.
	byPage map[int][]string
	// pageCount is the number of pages pdfcpu found in the document, or 0 when it
	// could not read it. It is independent of the text extractor, whose output drops
	// trailing pages that carry no text.
	pageCount int
}

// writePDFImages dumps every embedded image into imagesDir in a single pass over the
// PDF, reporting progress as it goes. It returns the written file names per page and
// the document's page count.
//
// The single pass is the point. api.ExtractImagesFile takes a file *path*, so every
// call re-opens, re-reads, re-validates and re-optimizes the whole document; asking it
// for one page at a time therefore bought one full parse per page. Measured on the
// 379 MB, 2304-page PDF that prompted this: 3.6s per call, about 2h20m for the loop,
// silent throughout - indistinguishable from a hang. The per-page form was there to cap
// peak memory, but it never could: each call already loaded the entire document, and
// images stream to disk one at a time either way. One pass over that same file takes
// 8.9s and peaks at 1.1 GB - the memory the old loop paid 2304 times in a row.
//
// Pages are still walked one by one (rather than handing the whole selection to
// api.ExtractImages) so that a single unreadable image stays a skipped page instead of
// aborting the run, and so page numbers can drive a progress line. The page number of
// each image is the loop's own, recorded as the file is written: reading it back out of
// the file name took the first number found, and a PDF named Volume_3.pdf put every
// image on page 3.
func writePDFImages(pdfPath, imagesDir string) (byPage map[int][]string, pageCount int) {
	f, err := os.Open(pdfPath)
	if err != nil {
		logging.Printf("  WARNING: could not open PDF for image extraction: %v\n", err)
		return nil, 0
	}
	defer func() { _ = f.Close() }()

	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.EXTRACTIMAGES
	ctx, err := api.ReadValidateAndOptimize(f, conf)
	if err != nil {
		logging.Printf("  WARNING: could not read PDF for image extraction: %v\n", err)
		return nil, 0
	}
	pageCount = ctx.PageCount

	// The name shape is pdfcpu's own ({source}_{page}_{resource}.{type}, sanitized the same
	// way), so output folders look as they always did; nothing parses it back any more.
	prefix := sanitize.PathOr(strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath)), "file")
	used := make(map[string]bool)
	byPage = make(map[int][]string)

	written := 0
	skippedThumbs := 0
	skippedDups := 0
	tick := logging.NewTicker("Extracting images", "pages")
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		tick.Report(pageNum-1, pageCount)
		kept, thumbs, dups, err := pageImagesSafe(ctx, pageNum)
		if err != nil {
			continue // expected for pages with no (or unreadable) images
		}
		skippedThumbs += thumbs
		skippedDups += dups
		for _, img := range kept {
			name := imageFileName(prefix, pageNum, img, used)
			if err := writeImageFile(filepath.Join(imagesDir, name), img); err != nil {
				logging.Printf("  WARNING: could not write image from page %d: %v\n", pageNum, err)
				continue
			}
			used[strings.ToLower(name)] = true
			byPage[pageNum] = append(byPage[pageNum], name)
			written++
		}
	}
	if written > 0 && !tick.Quiet() {
		tick.Report(pageCount, pageCount)
	}
	// Say what was dropped so an image count of 0 (or of "fewer than the PDF holds")
	// is explained rather than looking like a silent loss.
	switch {
	case skippedThumbs > 0 && skippedDups > 0:
		logging.Printf("  NOTE: skipped %d page thumbnail(s) and %d duplicate raster(s), not page content\n", skippedThumbs, skippedDups)
	case skippedThumbs > 0:
		logging.Printf("  NOTE: skipped %d page thumbnail(s), not page content\n", skippedThumbs)
	case skippedDups > 0:
		logging.Printf("  NOTE: skipped %d duplicate raster(s) - same page embedded at another resolution\n", skippedDups)
	}
	return byPage, pageCount
}

// pageImagesSafe extracts and filters one page's images. A panic inside pdfcpu on one
// malformed image stream costs that page its pictures, not the conversion.
func pageImagesSafe(ctx *model.Context, pageNum int) (kept []model.Image, thumbs, dups int, err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Printf("  WARNING: PDF image extraction panicked on page %d: %v\n", pageNum, r)
			logging.RunLogf("%s\n", debug.Stack())
			kept, thumbs, dups, err = nil, 0, 0, fmt.Errorf("panic on page %d: %v", pageNum, r)
		}
	}()
	imgs, err := pdfcpulib.ExtractPageImages(ctx, pageNum, false)
	if err != nil {
		return nil, 0, 0, err
	}
	// The real (non-stub) extraction leaves Width/Height zero and never looks at the
	// dictionary's mask entries; a stub pass fills both from the image dict without
	// decoding any pixels. selectPageImages needs the dimensions to spot a page
	// embedded twice at two resolutions, and the /Mask flag to tell an MRC scan's
	// foreground layer from the page it is painted over.
	if stubs, serr := pdfcpulib.ExtractPageImages(ctx, pageNum, true); serr == nil {
		for objNr, img := range imgs {
			if s, ok := stubs[objNr]; ok {
				img.Width = s.Width
				img.Height = s.Height
				img.HasImgMask = s.HasImgMask
				img.HasSMask = s.HasSMask
				imgs[objNr] = img
			}
		}
	}
	kept, thumbs, dups = selectPageImages(imgs)
	return kept, thumbs, dups, nil
}

// imageFileName names one extracted image. used holds the lower-cased names already
// taken in this pass: two resources can share a name on one page (a form XObject
// reusing "Im0"), and on a case-insensitive file system the second write would
// silently replace the first.
func imageFileName(prefix string, pageNum int, img model.Image, used map[string]bool) string {
	qual := sanitize.PathOr(img.Name, "image")
	fileType := sanitize.PathOr(img.FileType, "img")
	name := fmt.Sprintf("%s_%d_%s.%s", prefix, pageNum, qual, fileType)
	if used[strings.ToLower(name)] {
		name = fmt.Sprintf("%s_%d_%s_%d.%s", prefix, pageNum, qual, img.ObjNr, fileType)
	}
	return name
}

func writeImageFile(path string, img model.Image) error {
	if err := pdfcpulib.WriteReader(path, img.Reader); err != nil {
		if errors.Is(err, pdfcpulib.ErrMissingReader) {
			return fmt.Errorf("image obj#%d has no data", img.ObjNr)
		}
		return err
	}
	return nil
}

// selectPageImages decides which of a page's extracted rasters are real page
// content, dropping two kinds that otherwise reach the reader as bugs:
//
//   - Thumbnails. pdfcpu returns a page's /Thumb preview as one of its images;
//     taken as content it renders a ~128px postage stamp where the page should be.
//   - Proportional-scale duplicates. A scanned page is often embedded twice - the
//     same picture at two resolutions (e.g. 1455x2065 and 4363x6193) - and emitting
//     both shows the page twice. Only the largest of a same-shape group is kept,
//     unless the largest is painted through a stencil /Mask (see below).
//
// Images of different shapes are left alone: a composed page (an illustration beside
// a figure) keeps all of them, because there guessing "the page" would be wrong as
// often as right. The returned slice is ordered by object number so output is stable.
//
// Size decides a duplicate group only among rasters that are whole pictures. A mixed
// raster content (MRC) scan - what library and archive scanners produce - embeds a page
// as a low-resolution *background* layer plus a high-resolution *foreground* layer that
// is painted through a stencil /Mask, and the foreground layer is undefined wherever the
// mask does not select it. Extracted whole it is not a lower-quality page, it is not a
// page at all: measured on pdf-1page-blackletter_Plague-Proclamation-1625, the 4363x6193
// foreground layer comes out as a pink-and-brown smear with the lettering trailed into
// vertical streaks, while the 1455x2065 background layer beside it is the readable page.
// Both are JPXDecode and the decoder reports no error, so nothing downstream can tell
// them apart - the /Mask on the dictionary is the only signal, and it is decisive here
// exactly because the two rasters are already known to be the same page.
//
// Only /Mask counts, not /SMask: soft-masked transparency leaves the base image a
// complete picture, and it is the ordinary shape of a PNG-with-alpha illustration. And
// the preference applies only *inside* a duplicate group, so a lone masked illustration -
// which has no unmasked twin to fall back to - is still kept and still reaches the reader.
func selectPageImages(imgs map[int]model.Image) (kept []model.Image, thumbs, dups int) {
	list := make([]model.Image, 0, len(imgs))
	for _, img := range imgs {
		if img.Thumb {
			thumbs++
			continue
		}
		list = append(list, img)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ObjNr < list[j].ObjNr })

	for _, img := range list {
		merged := false
		for k := range kept {
			if sameShapeRaster(kept[k], img) {
				dups++
				if betterPageRaster(img, kept[k]) {
					kept[k] = img
				}
				merged = true
				break
			}
		}
		if !merged {
			kept = append(kept, img)
		}
	}
	return kept, thumbs, dups
}

func imagePixels(img model.Image) int64 {
	return int64(img.Width) * int64(img.Height)
}

// betterPageRaster reports whether candidate should replace current as the one raster kept
// for a page shape. A raster painted through a stencil /Mask loses to one without a mask
// however big it is - that is the MRC foreground layer, undefined outside its mask - and
// otherwise the larger picture wins, as it always did.
func betterPageRaster(candidate, current model.Image) bool {
	if candidate.HasImgMask != current.HasImgMask {
		return !candidate.HasImgMask
	}
	return imagePixels(candidate) > imagePixels(current)
}

// aspectRatioTolerance is how close two rasters' aspect ratios must be to count as
// the same picture at a different resolution: 1% absorbs rounding in stored pixel
// dimensions without matching genuinely different shapes.
const aspectRatioTolerance = 0.01

// sameShapeRaster reports whether two rasters are the same picture at a different
// scale: both dimensions positive and their aspect ratios equal within tolerance.
// A uniform scale preserves the aspect ratio, so equal ratios is exactly the signal
// for "one is a resized copy of the other".
func sameShapeRaster(a, b model.Image) bool {
	if a.Width <= 0 || a.Height <= 0 || b.Width <= 0 || b.Height <= 0 {
		return false
	}
	ra := float64(a.Width) / float64(a.Height)
	rb := float64(b.Width) / float64(b.Height)
	return math.Abs(ra-rb) <= aspectRatioTolerance*math.Max(ra, rb)
}

// extractImages extracts all embedded images from the PDF using pdfcpu into
// outputDir/pdf_images/. The page map is empty (non-fatal) if no images exist or
// extraction fails; pageCount is 0 when pdfcpu could not read the document at all.
func extractImages(pdfPath, outputDir string) pdfImages {
	imagesSubdir := "pdf_images"
	imagesDir := filepath.Join(outputDir, imagesSubdir)
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		logging.Printf("  WARNING: could not create images dir: %v\n", err)
		return pdfImages{}
	}

	byName, pageCount := writePDFImagesSafe(pdfPath, imagesDir)
	if len(byName) == 0 {
		logging.Printf("  NOTE: no images extracted from PDF\n")
		return pdfImages{pageCount: pageCount}
	}

	if err := normalizeExtractedPDFImages(imagesDir, byName); err != nil {
		logging.Printf("  WARNING: could not normalize extracted PDF images: %v\n", err)
		if strings.Contains(err.Error(), "no JPX converter found") {
			dialog.ShowWarning(
				"PDF Images Not Displayed",
				"This PDF contains images in JPEG2000 format (.jpx).\n"+
					"Browsers cannot display JPEG2000 - images will be missing in the result.\n\n"+
					"To fix, install ffmpeg:\n"+
					"  winget install ffmpeg\n\n"+
					"Or download from: https://www.gyan.dev/ffmpeg/builds/\n"+
					`(choose "release essentials" build)`,
			)
		}
	}

	pageImages := make(map[int][]string, len(byName))
	total := 0
	for pageNum, names := range byName {
		for _, name := range names {
			pageImages[pageNum] = append(pageImages[pageNum], imagesSubdir+"/"+name)
		}
		total += len(names)
	}
	logging.Printf("  Images: %d extracted across %d pages\n", total, len(pageImages))
	return pdfImages{byPage: pageImages, pageCount: pageCount}
}

// writePDFImagesSafe runs the image pass with a panic guard around the parts that are not
// per page (reading and validating the whole document). The pass is best-effort: a pdfcpu
// panic there costs the book its pictures, never its text.
func writePDFImagesSafe(pdfPath, imagesDir string) (byPage map[int][]string, pageCount int) {
	defer func() {
		if r := recover(); r != nil {
			logging.Printf("  WARNING: PDF image extraction panicked: %v\n", r)
			logging.RunLogf("%s\n", debug.Stack())
			byPage, pageCount = nil, 0
		}
	}()
	return writePDFImages(pdfPath, imagesDir)
}

// normalizeExtractedPDFImages makes the written images displayable in a browser, updating
// byPage in place where a conversion renames a file (.jpx becomes .jpg). A file that could
// not be converted keeps its entry, as it did before the page map was recorded at write time.
func normalizeExtractedPDFImages(imagesDir string, byPage map[int][]string) error {
	var firstErr error
	for _, names := range byPage {
		for i, name := range names {
			path := filepath.Join(imagesDir, name)
			if strings.EqualFold(filepath.Ext(name), ".jpx") {
				jpgPath, err := convertJPXFile(path)
				if err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("%s: %w", name, err)
					}
					continue
				}
				names[i] = filepath.Base(jpgPath)
				continue
			}
			if err := flipImageFileVertically(path); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return firstErr
}

// findJPXConverter returns the path to a binary that can convert JPEG2000 (.jpx)
// images to JPEG, along with a tag identifying the tool ("magick" or "ffmpeg").
// Browsers do not support JPEG2000; pdfcpu extracts JPXDecode PDF streams as .jpx.
func findJPXConverter() (bin, kind string) {
	if p, err := exec.LookPath("magick"); err == nil {
		return p, "magick"
	}
	for _, pattern := range []string{
		`C:\Program Files\ImageMagick-*\magick.exe`,
		`C:\Program Files (x86)\ImageMagick-*\magick.exe`,
	} {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[0], "magick"
		}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, "ffmpeg"
	}
	return "", ""
}

// convertJPXFile converts a JPEG 2000 (.jpx) file to JPEG using ImageMagick or
// ffmpeg, removes the original, and returns the new .jpg path.
func convertJPXFile(jpxPath string) (string, error) {
	bin, kind := findJPXConverter()
	if bin == "" {
		return "", fmt.Errorf("no JPX converter found (install ImageMagick or ffmpeg)")
	}
	jpgPath := strings.TrimSuffix(jpxPath, filepath.Ext(jpxPath)) + ".jpg"
	args := []string{jpxPath, jpgPath}
	if kind == "ffmpeg" {
		args = []string{"-y", "-i", jpxPath, "-update", "1", jpgPath}
	}
	if _, err := procrun.Run(context.Background(), procrun.Cmd{
		Tool: kind, Path: bin, Args: args, Timeout: procrun.ImageConvert.ForFile(jpxPath),
	}); err != nil {
		return "", err
	}
	_ = os.Remove(jpxPath)
	return jpgPath, nil
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
		_ = out.Close()
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
