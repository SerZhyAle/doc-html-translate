package pipeline

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/internal/browser"
	"doc-html-translate/internal/comic"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fb2"
	"doc-html-translate/internal/htmlconv"
	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/htmlproc"
	"doc-html-translate/internal/htmlsplit"
	"doc-html-translate/internal/img"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/md"
	"doc-html-translate/internal/mobi"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/pdf"
	"doc-html-translate/internal/rtf"
	"doc-html-translate/internal/translator"
	"doc-html-translate/internal/txt"

	gohtml "golang.org/x/net/html"
)

// ExitCode constants for structured error handling.
const (
	ExitOK        = 0
	ExitArgsError = 1
	ExitIOError   = 2
	ExitEPUB      = 3
	ExitParse     = 3 // alias: same code for any parse error (EPUB or PDF)
	ExitAPI       = 4
)

type Runner struct {
	cfg config.Config
}

type contentPage struct {
	item      epub.ManifestItem
	filePath  string
	segments  []*htmlproc.TextSegment
	doc       *gohtml.Node
	err       error
	charCount int
}

func NewRunner(cfg config.Config) Runner {
	return Runner{cfg: cfg}
}

// Run executes the file-to-HTML pipeline (EPUB or PDF).
// Steps: [1] Check existing / Extract → [2] Build HTML → [3] Translate → [4] Open browser.
func (r Runner) Run() (int, error) {
	inputPath, err := filepath.Abs(r.cfg.InputFile)
	if err != nil {
		return ExitIOError, fmt.Errorf("resolve input path: %w", err)
	}

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return ExitArgsError, fmt.Errorf("file not found: %s", inputPath)
	}

	ext := strings.ToLower(filepath.Ext(inputPath))

	// Output directory: same location as file (or -folder path), named after the file (without extension)
	outputDir := outputpath.OutputDirFor(inputPath, r.cfg.OutputFolder)
	indexPath := filepath.Join(outputDir, "index.html")

	// R4: if output dir + index.html already exist → open browser immediately
	if _, err := os.Stat(indexPath); err == nil {
		if r.cfg.Force {
			logging.Printf("Book already extracted, forcing rebuild: %s\n", outputDir)
			if err := os.RemoveAll(outputDir); err != nil {
				return ExitIOError, fmt.Errorf("force cleanup output dir: %w", err)
			}
		} else {
			logging.Printf("Book already extracted: %s\n", outputDir)
			if r.cfg.NoOpen {
				logging.Println("[4/4] Browser open skipped (-noopen)")
				logging.Println("Done.")
				return ExitOK, nil
			}
			logging.Println("[4/4] Opening in browser..")
			if err := browser.Open(indexPath); err != nil {
				return ExitIOError, fmt.Errorf("open browser: %w", err)
			}
			logging.Println("Done.")
			return ExitOK, nil
		}
	}

	// Step 1: Extract (format-specific)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return ExitIOError, fmt.Errorf("create output dir: %w", err)
	}

	var book *epub.Book
	// A standalone image has no text to extract: wrap it in a one-page HTML doc and
	// force the OCR overlay below, so the browser shows the picture with translatable
	// text plates laid over it (mirrors the extension's image OCR overlay).
	forceOCR := false
	if img.IsImage(ext) {
		logging.Println("[1/4] Preparing image..")
		book, err = img.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("prepare image: %w", err)
		}
		forceOCR = true
	} else if comic.IsComic(ext) {
		// A comic archive is a container of page images with no text layer: wrap each
		// page in a one-page HTML doc and force the OCR overlay below, so the browser
		// shows every page with translatable text plates over the bubbles. Opening a
		// comic *is* the request to read its text, so OCR is forced here rather than
		// left to -ocr (same rationale as a standalone image).
		logging.Println("[1/4] Extracting comic archive..")
		book, err = comic.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract comic: %w", err)
		}
		forceOCR = true
	} else {
		switch ext {
		case ".epub":
			logging.Println("[1/4] Extracting EPUB..")
			book, err = epub.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitEPUB, fmt.Errorf("extract epub: %w", err)
			}
			logging.Printf("  Title: %s\n", book.Title)
			logging.Printf("  Chapters: %d\n", len(book.Spine))
		case ".pdf":
			logging.Println("[1/4] Extracting PDF..")
			book, err = pdf.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract pdf: %w", err)
			}
		case ".txt":
			logging.Println("[1/4] Extracting TXT..")
			book, err = txt.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract txt: %w", err)
			}
		case ".md":
			logging.Println("[1/4] Extracting Markdown..")
			book, err = md.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract markdown: %w", err)
			}
		case ".fb2":
			logging.Println("[1/4] Extracting FB2..")
			book, err = fb2.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract fb2: %w", err)
			}
		case ".rtf":
			logging.Println("[1/4] Extracting RTF..")
			book, err = rtf.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract rtf: %w", err)
			}
		case ".html", ".htm":
			logging.Println("[1/4] Extracting HTML..")
			book, err = htmlconv.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract html: %w", err)
			}
		case ".mobi", ".azw3":
			logging.Println("[1/4] Extracting MOBI..")
			book, err = mobi.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract mobi: %w", err)
			}
		default:
			// Unknown extension: treat it as plain text - but only if it is text. A binary
			// (a .docx, a .djvu, a comic archive) handed to the text extractor became a
			// multi-megabyte document of raw bytes rendered as prose, reported as success. The
			// browser extension routes on the byte signature and refuses these; this matches it.
			if head, herr := readHead(inputPath, 4096); herr == nil {
				if desc := txt.LooksBinary(head); desc != "" {
					_ = os.RemoveAll(outputDir)
					return ExitParse, fmt.Errorf("%s looks like %s, not a text document - refusing to convert it into garbage",
						filepath.Base(inputPath), desc)
				}
			}
			logging.Printf("[1/4] Unknown extension %q - reading as plain text..\n", ext)
			book, err = txt.Extract(inputPath, outputDir)
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return ExitParse, fmt.Errorf("extract as txt: %w", err)
			}
		}
	}

	// Optional: split oversized pages at paragraph boundaries so browser
	// translation extensions (Chrome GT: ~5000 chars) can handle each page.
	// Single-page mode merges everything into one file anyway, so splitting is moot.
	if r.cfg.SplitSize > 0 && !r.cfg.SinglePage {
		n, err := htmlsplit.SplitIfNeeded(book, outputDir, r.cfg.SplitSize)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitIOError, fmt.Errorf("split pages: %w", err)
		}
		if n > 0 {
			logging.Printf("  Split: %d additional pages created (%d total, max %d chars each)\n",
				n, len(book.Spine), r.cfg.SplitSize)
		}
	}

	// Step 2: Inject navigation bars (must happen before translation).
	logging.Println("[2/4] Building HTML structure..")
	var generatedIndex string
	switch {
	case r.cfg.SinglePage:
		// Merge the whole document into one HTML page; no TOC, no navigation bars.
		generatedIndex, err = htmlgen.GenerateSinglePage(book, outputDir, filepath.Base(inputPath))
		if err != nil {
			return ExitIOError, fmt.Errorf("generate single page: %w", err)
		}
		logging.Println("  Single-page mode - all content merged, TOC skipped.")
	case len(book.Spine) == 1:
		// Single page — no TOC, no navigation bars needed.
		generatedIndex, err = htmlgen.GenerateSinglePageIndex(book, outputDir)
		if err != nil {
			return ExitIOError, fmt.Errorf("generate single-page index: %w", err)
		}
		logging.Println("  Single page - TOC and navigation skipped.")
	default:
		if err := htmlgen.InjectNavBars(book, outputDir, filepath.Base(inputPath)); err != nil {
			return ExitIOError, fmt.Errorf("inject navbars: %w", err)
		}
		logging.Printf("  Navigation: %d pages\n", len(book.SpineHrefs()))
	}

	// Optional: OCR document images and overlay translatable text plates. Runs before
	// translation so the overlay text is translated too. Best-effort - never fatal.
	if r.cfg.OCR || forceOCR {
		r.overlayImages(book, outputDir)
	}

	// Step 3: Translation
	// Advisory: when a translation engine is on, warn once if a language code looks
	// malformed (e.g. "russian" instead of "ru"). Non-fatal - the engine still runs.
	if r.cfg.UseGoogle || r.cfg.UseOllama {
		if bad := config.SuspiciousLangCodes(r.cfg.SourceLang, r.cfg.TargetLang); len(bad) > 0 {
			logging.Errorf("WARNING: %s does not look like a language code (expected e.g. 'en', 'ru', 'zh-CN'); translation may be wrong.\n", strings.Join(bad, " and "))
		}
	}

	var tocSnippets map[string]string
	switch {
	case r.cfg.NoTranslate:
		logging.Println("[3/4] Translation skipped (-notranslate)")
	case r.cfg.UseGoogle:
		apiKey, keyErr := translator.LoadGoogleAPIKey()
		if keyErr != nil {
			logging.Printf("[3/4] Google Translate skipped - API key not available.\n")
			logging.Printf("       To enable: save your Google Cloud Translation API key as 'google_api.key' in either:\n")
			for _, p := range translator.GoogleAPIKeyPaths() {
				logging.Printf("         %s\n", p)
			}
			logging.Printf("       Details: %v\n", keyErr)
			break
		}
		pages := loadContentPages(book, outputDir)
		totalChars := countLoadedPageChars(pages)
		if totalChars > 1000 {
			estCost := float64(totalChars) / 1_000_000 * 20
			if r.cfg.MaxCost > 0 && estCost > r.cfg.MaxCost {
				logging.Printf("[3/4] Translation skipped - estimated cost $%.2f USD exceeds -max-cost $%.2f limit\n", estCost, r.cfg.MaxCost)
				break
			}
			msg := fmt.Sprintf(
				"Characters to send: %s\nEstimated cost: $%.2f USD\n\nProceed with Google Translate?",
				formatInt(totalChars), estCost,
			)
			if !dialog.ConfirmYesNo("Google Translate - Cost Warning", msg) {
				logging.Println("[3/4] Translation cancelled by user")
				break
			}
		}
		client := translator.NewCachingClient(translator.NewGoogleClient(apiKey))
		if exitCode, snippets, err := r.translateContent(book, client, pages); err != nil {
			return exitCode, err
		} else {
			tocSnippets = snippets
		}
	case r.cfg.UseOllama:
		ollamaWorker := translator.NewOllamaClient(r.cfg.OllamaModel)
		ollamaWorker.SetParallelism(r.cfg.OllamaParallel)
		ollamaWorker.SetNumCtx(r.cfg.OllamaNumCtx)
		// Unload model from VRAM on Ctrl-C.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		go func() {
			<-sigCh
			fmt.Println()
			logging.Println("Interrupted. Unloading Ollama model from VRAM..")
			ollamaWorker.Unload()
			os.Exit(130)
		}()
		defer signal.Stop(sigCh)
		client := translator.NewCachingClient(ollamaWorker)
		pages := loadContentPages(book, outputDir)
		if exitCode, snippets, err := r.translateContent(book, client, pages); err != nil {
			return exitCode, err
		} else {
			tocSnippets = snippets
		}
	default:
		logging.Println("[3/4] Translation skipped (use -google or -ollama to enable)")
	}

	// Generate TOC after translation so snippets reflect translated text.
	if len(book.Spine) > 1 {
		// No authored TOC (NCX/nav for EPUB, bookmarks for PDF)? Synthesize a
		// multi-level TOC from the headings on the now-final pages, injecting
		// stable anchors. Runs on the translated files so labels are translated.
		if len(book.TOC) == 0 {
			book.TOC = htmlgen.BuildFallbackTOC(book, outputDir, tocSnippets)
		}

		generatedIndex, err = htmlgen.GenerateIndexWithSnippetsDepth(book, outputDir, tocSnippets, r.cfg.TOCDepth)
		if err != nil {
			return ExitIOError, fmt.Errorf("generate index: %w", err)
		}
		logging.Printf("  TOC created: %s\n", generatedIndex)
	}

	// Step 4: Open in browser
	if r.cfg.NoOpen {
		logging.Println("[4/4] Browser open skipped (-noopen)")
	} else {
		logging.Println("[4/4] Opening in browser..")
		if err := browser.Open(generatedIndex); err != nil {
			return ExitIOError, fmt.Errorf("open browser: %w", err)
		}
	}

	logging.Println("Done.")
	return ExitOK, nil
}

// translateContent translates all HTML content files in the book.
// R5: On error — report error, pause so user sees the message in shell.
func (r Runner) translateContent(book *epub.Book, client translator.Client, pages []contentPage) (int, map[string]string, error) {
	total := len(pages)
	if total == 0 {
		logging.Println("[3/4] No content files to translate")
		return ExitOK, nil, nil
	}

	logging.Printf("[3/4] Translating %d pages..\n", total)
	tocSnippets := make(map[string]string, total)

	for i, page := range pages {
		if page.err != nil {
			logging.Errorf("  WARNING: skip %s: %v\n", page.item.Href, page.err)
			continue
		}

		segments := page.segments
		if len(segments) == 0 {
			tocSnippets[page.item.Href] = htmlgen.ExtractSnippetFromDoc(page.doc)
			logging.Printf("  [%d/%d] %s (no text)\n", i+1, total, page.item.Href)
			continue
		}

		// Collect texts for translation.
		texts := make([]string, len(segments))
		for j, seg := range segments {
			texts[j] = seg.Text
		}

		// Set up per-page progress display (overwrites line with \r).
		pageStart := time.Now()
		nSegs := len(segments)
		pageIdx, pageTotal, pageName := i+1, total, page.item.Href
		if pr, ok := client.(translator.ProgressReporter); ok {
			pr.SetProgress(func(done, ttl int) {
				elapsed := time.Since(pageStart).Seconds()
				rate := float64(done) / elapsed
				etaStr := ""
				if rate > 0 {
					etaStr = " ETA " + formatDuration(float64(ttl-done)/rate)
				}
				logging.Progress("  [%d/%d] %s: %d/%d segs  %.1f/s%s     ",
					pageIdx, pageTotal, pageName, done, ttl, rate, etaStr)
			})
		} else {
			logging.Printf("  [%d/%d] %s", i+1, total, page.item.Href)
		}

		// Translate.
		translated, err := client.Translate(texts, r.cfg.SourceLang, r.cfg.TargetLang)

		// Clear progress callback so next page starts fresh.
		if pr, ok := client.(translator.ProgressReporter); ok {
			pr.SetProgress(nil)
		}

		if err != nil {
			// R5: Show error, pause for user to see
			logging.Errorf("\nTRANSLATION ERROR: %v\n", err)
			logging.Errorf("Translation failed at page %d/%d (%s)\n", i+1, total, page.item.Href)
			logging.Errorf("The book will be opened WITHOUT translation.\n")
			if logging.StdoutIsTerminal() {
				logging.Errorf("Press Enter to continue..\n")
				_, _ = fmt.Scanln()
			}
			return ExitOK, nil, nil
		}

		// Replace text nodes with translations
		htmlproc.ReplaceTexts(segments, translated)

		// Write back
		if err := htmlproc.RenderToFile(page.doc, page.filePath); err != nil {
			logging.Errorf("  WARNING: write failed %s: %v\n", page.item.Href, err)
			continue
		}
		tocSnippets[page.item.Href] = htmlgen.ExtractSnippetFromDoc(page.doc)

		elapsed := time.Since(pageStart).Seconds()
		rate := float64(nSegs) / elapsed
		logging.Progress("  [%d/%d] %s: %d segs in %s (%.1f/s)\n",
			i+1, total, page.item.Href, nSegs, formatDuration(elapsed), rate)
	}

	// Translate book title.
	if book.Title != "" {
		if titles, err := client.Translate([]string{book.Title}, r.cfg.SourceLang, r.cfg.TargetLang); err == nil && len(titles) > 0 {
			book.Title = titles[0]
		}
	}

	// Translate authored TOC labels (EPUB NCX/nav, PDF bookmarks). The
	// heading-scan fallback is built later from already-translated pages, so it
	// is not translated here. The shared CachingClient keeps labels consistent
	// with identical in-page headings.
	r.translateTOCTitles(book, client)

	logging.Println("  Translation complete.")
	return ExitOK, tocSnippets, nil
}

// translateTOCTitles batch-translates every label in book.TOC in place,
// preserving the tree structure. Best-effort: a failed batch leaves labels as-is.
func (r Runner) translateTOCTitles(book *epub.Book, client translator.Client) {
	if len(book.TOC) == 0 {
		return
	}
	var titles []string
	collectTOCTitles(book.TOC, &titles)
	if len(titles) == 0 {
		return
	}
	translated, err := client.Translate(titles, r.cfg.SourceLang, r.cfg.TargetLang)
	if err != nil || len(translated) == 0 {
		return
	}
	idx := 0
	assignTOCTitles(book.TOC, translated, &idx)
}

func collectTOCTitles(entries []epub.TOCEntry, out *[]string) {
	for i := range entries {
		*out = append(*out, entries[i].Title)
		collectTOCTitles(entries[i].Children, out)
	}
}

func assignTOCTitles(entries []epub.TOCEntry, translated []string, idx *int) {
	for i := range entries {
		if *idx >= len(translated) {
			return
		}
		if t := translated[*idx]; t != "" {
			entries[i].Title = t
		}
		*idx++
		assignTOCTitles(entries[i].Children, translated, idx)
	}
}

// overlayImages OCRs the images in every content page and rewrites each image into a
// positioned container with translatable text plates. Best-effort: a missing tesseract or
// a failed page is logged and skipped, never aborting the conversion.
func (r Runner) overlayImages(book *epub.Book, outputDir string) {
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
	dataDir := ocr.DataDir()
	logging.Printf("  OCR overlay: engine %s, language %s\n", bin, ocr.LangLabel(lang))

	// Without its data file the engine fails on every image with the same sentence, so a book
	// of scans produced hundreds of identical errors that never named the fix. Ask once instead.
	if missing := ocr.MissingLangs(bin, lang); len(missing) > 0 {
		logging.Printf("  OCR skipped: no language data for %s. Install it with -ocr-download %s, "+
			"or choose another language with -ocr-lang (-ocr-langs lists them)\n",
			strings.Join(missing, ", "), missing[0])
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
		href := item.Href
		if book.BasePath != "" && book.BasePath != "." {
			href = book.BasePath + "/" + href
		}
		filePaths = append(filePaths, filepath.Join(outputDir, filepath.FromSlash(href)))
	}
	tick := logging.NewTicker("OCR overlay", "images")
	stats := ocr.OverlayBook(bin, filePaths, lang, dataDir, r.cfg.OCRLang != "", tick.Report)
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

// readHead returns up to n leading bytes of a file, for format sniffing. A short read (the
// file is smaller than n) is not an error - it returns what there was.
func readHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	got, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:got], nil
}

func loadContentPages(book *epub.Book, outputDir string) []contentPage {
	contentFiles := book.ContentFiles()
	pages := make([]contentPage, 0, len(contentFiles))
	for _, item := range contentFiles {
		href := item.Href
		if book.BasePath != "" && book.BasePath != "." {
			href = book.BasePath + "/" + href
		}
		filePath := filepath.Join(outputDir, filepath.FromSlash(href))
		segments, doc, err := htmlproc.ExtractTexts(filePath)
		page := contentPage{
			item:     item,
			filePath: filePath,
			segments: segments,
			doc:      doc,
			err:      err,
		}
		if err == nil {
			for _, seg := range segments {
				page.charCount += len(seg.Text)
			}
		}
		pages = append(pages, page)
	}
	return pages
}

// countLoadedPageChars returns total number of translatable characters across parsed content pages.
func countLoadedPageChars(pages []contentPage) int {
	total := 0
	for _, page := range pages {
		if page.err != nil {
			continue
		}
		total += page.charCount
	}
	return total
}

// formatDuration formats seconds as "4m5s" or "38s".
func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	m := int(seconds) / 60
	s := int(seconds) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

// formatInt formats an integer with thousands separators.
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		pos := len(s) - i
		if i > 0 && pos%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
