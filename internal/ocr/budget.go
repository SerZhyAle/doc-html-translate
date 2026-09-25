package ocr

import (
	"image"
	"os"
	"path/filepath"
	"strconv"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/limits"
	"doc-html-translate/internal/logging"
)

// ocrFrame is the one decoded picture a recognition shares across its passes. The staging step
// already holds the prepared image in memory, and the grey ladder and the screen pass used to
// decode the staged PNG again from disk; now they derive from this frame instead.
type ocrFrame struct {
	path  string      // what tesseract reads
	img   image.Image // the prepared picture, decoded at most once
	tried bool
	gray  *image.Gray
}

func (f *ocrFrame) image() image.Image {
	if !f.tried {
		f.tried = true
		if f.img == nil {
			f.img = decodeImage(f.path)
		}
	}
	return f.img
}

// grey returns the frame's luminance rendition, built once. The colour picture is dropped as soon
// as the grey exists: every pass after this point reads the grey, and a full RGBA copy held
// through the remaining tesseract calls is four bytes a pixel the worker does not need.
func (f *ocrFrame) grey() *image.Gray {
	if f.gray == nil {
		f.gray = greyOf(f.image())
		f.img, f.tried = nil, true
	}
	return f.gray
}

// headerPixels reads only the image header and returns its declared pixel count, 0 when it cannot
// be read. Nothing is decoded.
func headerPixels(path string) (w, h int64) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return int64(cfg.Width), int64(cfg.Height)
}

// warnOverBudget says, once per image, that an image over the pixel budget is recognized in
// degraded form. It is not downscaled first: shrinking it would need the very full decode the
// budget forbids, and no standard decoder reads at reduced resolution. Tesseract, which runs in
// its own process, still reads the original; only the in-process passes (staging, the grey
// ladder, the screen pass, plate colours) go without, because decodeImage refuses it.
func warnOverBudget(path string) {
	if w, h := headerPixels(path); limits.CheckPixels(w, h) != nil {
		logging.Printf("  WARNING: %s\n", i18n.S("Image %s is above the %d-megapixel limit: Tesseract reads it on its own, without plate colours or rescue passes",
			filepath.Base(path), limits.MaxImagePixels/1_000_000))
	}
}

// Memory sizing for the worker pool. A worker holds up to ocrFrameCopies full-frame copies of
// its image at 4 bytes a pixel: the decoded source, the oriented or upscaled copy staging makes
// from it, and the grey pass's float32 blur buffer next to its input and output. On the 386 build
// the whole process has 2 GB of address space, so the pool may spend half of it; a 64-bit build
// gets 4 GB. Sixteen workers over 100-megapixel scans would otherwise ask for 25 GB.
const (
	ocrFrameCopies   = 4
	ocrBytesPerPixel = 4
)

func ocrMemoryBudget() int64 {
	if strconv.IntSize == 32 {
		return 1 << 30
	}
	return 4 << 30
}

// memoryWorkers is how many workers fit the budget when the largest image has largestPixels.
// At least one: a single image is always attempted, and one over the budget is never decoded.
func memoryWorkers(largestPixels, budget int64) int {
	largestPixels = min(largestPixels, limits.MaxImagePixels)
	if largestPixels <= 0 {
		return int(^uint(0) >> 1)
	}
	n := budget / (largestPixels * ocrBytesPerPixel * ocrFrameCopies)
	return int(max(1, min(n, 1<<16)))
}

// largestPixels is the biggest declared image among paths, from headers only.
func largestPixels(paths []string) int64 {
	var most int64
	for _, p := range paths {
		if w, h := headerPixels(p); w*h > most {
			most = w * h
		}
	}
	return most
}

// poolWorkers is the pool width for paths: the CPU-based count, capped by what memory allows for
// the largest image in the set. Deterministic for a given machine and book.
func poolWorkers(paths []string) int {
	return min(ocrWorkers(), memoryWorkers(largestPixels(paths), ocrMemoryBudget()))
}
