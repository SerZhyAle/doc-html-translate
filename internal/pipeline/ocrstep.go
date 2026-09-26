package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/ocr"
)

// overlayImages OCRs the images in every content page and rewrites each image into a
// positioned container with translatable text plates. Best-effort: a missing tesseract or
// a failed page is logged and skipped, never aborting the conversion.
func (r Runner) overlayImages(ctx context.Context, book *epub.Book, outputDir string) {
	bin, err := ocr.Locate()
	if err != nil {
		logging.Printf("  OCR skipped: %v\n", err)
		return
	}
	// Which language is read is decided here and nowhere else: -ocr-lang when given, else the
	// translation source language, which itself defaults to English. That default is easy not
	// to notice, so the line below names it rather than leaving the reader to infer it from a
	// page that came back empty.
	lang := r.cfg.OCRLang
	if lang == "" {
		lang = ocr.TessLang(r.cfg.SourceLang)
	}
	logging.Printf("  OCR overlay: engine %s, language %s\n", bin, ocr.LangLabel(lang))

	// Without its data file the engine fails on every image with the same sentence, so a book
	// of scans produced hundreds of identical errors that never named the fix. Ask once instead.
	if missing := ocr.MissingLangs(ctx, bin, lang); len(missing) > 0 {
		logging.Printf("  OCR skipped: no language data for %s. %s\n",
			strings.Join(missing, ", "), ocr.MissingAdvice(missing))
		return
	}
	// The same holds for data that exists but cannot reach the engine: a bundled pack that could not
	// be staged beside a downloaded one, or a folder whose path the engine cannot open.
	dataDir, err := ocr.DataDirFor(lang)
	if err == nil {
		dataDir, err = ocr.PrepareEngine(dataDir)
	}
	if err != nil {
		logging.Printf("  OCR skipped: %v\n", err)
		return
	}

	// Recognition batches across the whole book, not per content file: the Tesseract pool
	// spans every page's images at once, so it stays at full width even in -multipage mode,
	// where each page is one content file with one image and a per-file pool would recognize
	// serially. Progress is per image (not per file): in single-page mode the whole book is
	// one content file, so a per-file counter would sit at 0/1 for the entire run - the very
	// silence that reads as a hang on a scanned book of thousands of pages.
	filePaths := make([]string, 0, len(book.ContentFiles()))
	for _, item := range book.ContentFiles() {
		filePaths = append(filePaths, contentFilePath(book, outputDir, item))
	}
	tick := logging.NewTicker("OCR overlay", "images")
	stats := ocr.OverlayBook(ctx, bin, filePaths, lang, dataDir, r.cfg.OCRLang != "", tick.Report)
	if stats.Cancelled {
		return
	}
	reportOverlay(stats, lang)
}

// reportOverlay says what happened to every image, not just how many worked. The count alone
// forces the reader to subtract and then guess at the remainder: a 2304-page graphic novel
// reporting "1711 overlaid" is behaving perfectly (the other 593 are art panels with no
// dialogue), while an 8-of-9 on a scanned contract was a real failure - and the old line made
// those two look identical. A failure is named with its file and its reason; images that
// simply hold no text are counted, because they are ordinary and listing them would bury the
// failures.
func reportOverlay(stats ocr.OverlayResult, lang string) {
	// The document's own script may have corrected a language the reader never chose, and may have
	// stopped the pass outright when nothing installed can read it. Either way it is the first thing
	// to say - the counts below are about a language the log has not yet named.
	if stats.ScriptNote != "" {
		logging.Printf("  OCR overlay: %s\n", stats.ScriptNote)
		if stats.Lang != "" {
			lang = stats.Lang
		}
	}
	summary := fmt.Sprintf("%d image(s) overlaid", stats.Overlaid)
	if stats.NoText > 0 {
		summary += fmt.Sprintf(", %d with no text found", stats.NoText)
	}
	if len(stats.Failed) > 0 {
		summary += fmt.Sprintf(", %d failed", len(stats.Failed))
	}
	logging.Printf("  OCR overlay: %s\n", summary)

	// "no text found" is a true sentence about the data that was loaded, and it reads as a
	// sentence about the picture. When not one image on the whole book yielded a line, the
	// likeliest cause is not the artwork but the language: this app translates into Russian by
	// default, for readers whose documents are the least likely to be English, while OCR
	// defaults to English. Say which data was used and how to change it - only in the
	// all-empty case, so a comic whose art panels legitimately hold no dialogue stays quiet.
	if stats.Overlaid == 0 && stats.NoText > 0 {
		logging.Printf("  OCR overlay: nothing matched the %s data. If these pages are in another "+
			"language, pass -ocr-lang <code> (-ocr-langs lists what is installed)\n", ocr.LangLabel(lang))
	}

	// Named on stdout beside the count they explain, as WARNING like the other best-effort
	// problems in this pipeline - a reason that lands on a different stream from its summary
	// is a reason the reader never sees when the log is redirected to a file.
	//
	// Capped: when the cause is systemic (no language data, a broken engine) every image
	// fails with the same sentence, and a scanned book would bury the rest of the log under
	// thousands of copies. The remainder is counted, never silently dropped.
	const maxNamed = 5
	for i, f := range stats.Failed {
		if i == maxNamed {
			logging.Printf("  WARNING: ..and %d more image(s) failed the same way\n", len(stats.Failed)-maxNamed)
			break
		}
		logging.Printf("  WARNING: OCR failed on %s: %v\n", filepath.Base(f.File), f.Err)
	}
}
