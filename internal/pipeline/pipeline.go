package pipeline

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/internal/browser"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fb2"
	"doc-html-translate/internal/htmlconv"
	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/htmlproc"
	"doc-html-translate/internal/htmlsplit"
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
			logging.Println("[4/4] Opening in browser...")
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
	switch ext {
	case ".epub":
		logging.Println("[1/4] Extracting EPUB...")
		book, err = epub.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitEPUB, fmt.Errorf("extract epub: %w", err)
		}
		logging.Printf("  Title: %s\n", book.Title)
		logging.Printf("  Chapters: %d\n", len(book.Spine))
	case ".pdf":
		logging.Println("[1/4] Extracting PDF...")
		book, err = pdf.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract pdf: %w", err)
		}
	case ".txt":
		logging.Println("[1/4] Extracting TXT...")
		book, err = txt.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract txt: %w", err)
		}
	case ".md":
		logging.Println("[1/4] Extracting Markdown...")
		book, err = md.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract markdown: %w", err)
		}
	case ".fb2":
		logging.Println("[1/4] Extracting FB2...")
		book, err = fb2.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract fb2: %w", err)
		}
	case ".rtf":
		logging.Println("[1/4] Extracting RTF...")
		book, err = rtf.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract rtf: %w", err)
		}
	case ".html", ".htm":
		logging.Println("[1/4] Extracting HTML...")
		book, err = htmlconv.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract html: %w", err)
		}
	case ".mobi", ".azw3":
		logging.Println("[1/4] Extracting MOBI...")
		book, err = mobi.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract mobi: %w", err)
		}
	default:
		// Unknown extension — treat as plain text.
		logging.Printf("[1/4] Unknown format %q — treating as plain text...\n", ext)
		book, err = txt.Extract(inputPath, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return ExitParse, fmt.Errorf("extract as txt: %w", err)
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
	logging.Println("[2/4] Building HTML structure...")
	var generatedIndex string
	switch {
	case r.cfg.SinglePage:
		// Merge the whole document into one HTML page; no TOC, no navigation bars.
		generatedIndex, err = htmlgen.GenerateSinglePage(book, outputDir, filepath.Base(inputPath))
		if err != nil {
			return ExitIOError, fmt.Errorf("generate single page: %w", err)
		}
		logging.Println("  Single-page mode — all content merged, TOC skipped.")
	case len(book.Spine) == 1:
		// Single page — no TOC, no navigation bars needed.
		generatedIndex, err = htmlgen.GenerateSinglePageIndex(book, outputDir)
		if err != nil {
			return ExitIOError, fmt.Errorf("generate single-page index: %w", err)
		}
		logging.Println("  Single page — TOC and navigation skipped.")
	default:
		if err := htmlgen.InjectNavBars(book, outputDir, filepath.Base(inputPath)); err != nil {
			return ExitIOError, fmt.Errorf("inject navbars: %w", err)
		}
		logging.Printf("  Navigation: %d pages\n", len(book.SpineHrefs()))
	}

	// Optional: OCR document images and overlay translatable text plates. Runs before
	// translation so the overlay text is translated too. Best-effort - never fatal.
	if r.cfg.OCR {
		r.overlayImages(book, outputDir)
	}

	// Step 3: Translation
	var tocSnippets map[string]string
	switch {
	case r.cfg.NoTranslate:
		logging.Println("[3/4] Translation skipped (-notranslate)")
	case r.cfg.UseGoogle:
		apiKey, keyErr := translator.LoadGoogleAPIKey()
		if keyErr != nil {
			logging.Printf("[3/4] Google Translate skipped — API key not available.\n")
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
			if !dialog.ConfirmYesNo("Google Translate — Cost Warning", msg) {
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
			logging.Println("Interrupted. Unloading Ollama model from VRAM...")
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
		logging.Println("[4/4] Opening in browser...")
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

	logging.Printf("[3/4] Translating %d pages...\n", total)
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
			logging.Errorf("Press Enter to continue...\n")
			_, _ = fmt.Scanln()
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
	lang := r.cfg.OCRLang
	if lang == "" {
		lang = ocr.TessLang(r.cfg.SourceLang)
	}
	dataDir := ocr.DataDir()
	logging.Printf("  OCR overlay: engine %s, language %s\n", bin, lang)

	total := 0
	for _, item := range book.ContentFiles() {
		href := item.Href
		if book.BasePath != "" && book.BasePath != "." {
			href = book.BasePath + "/" + href
		}
		filePath := filepath.Join(outputDir, filepath.FromSlash(href))
		n, err := ocr.OverlayFile(bin, filePath, lang, dataDir)
		if err != nil {
			logging.Printf("  OCR overlay %s: %v\n", filepath.Base(filePath), err)
			continue
		}
		total += n
	}
	logging.Printf("  OCR overlay: %d image(s) overlaid\n", total)
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
