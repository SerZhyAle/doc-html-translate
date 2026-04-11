package pipeline

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	"doc-html-translate/internal/pdf"
	"doc-html-translate/internal/rtf"
	"doc-html-translate/internal/translator"
	"doc-html-translate/internal/txt"
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
	outputDir := outputDirFor(inputPath, r.cfg.OutputFolder)
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
	if r.cfg.SplitSize > 0 {
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
	if len(book.Spine) == 1 {
		// Single page — no TOC, no navigation bars needed.
		generatedIndex, err = htmlgen.GenerateSinglePageIndex(book, outputDir)
		if err != nil {
			return ExitIOError, fmt.Errorf("generate single-page index: %w", err)
		}
		logging.Println("  Single page — TOC and navigation skipped.")
	} else {
		if err := htmlgen.InjectNavBars(book, outputDir); err != nil {
			return ExitIOError, fmt.Errorf("inject navbars: %w", err)
		}
		logging.Printf("  Navigation: %d pages\n", len(book.SpineHrefs()))
	}

	// Step 3: Translation
	switch {
	case r.cfg.NoTranslate:
		logging.Println("[3/4] Translation skipped (-notranslate)")
	case r.cfg.UseGoogle:
		apiKey, keyErr := translator.LoadGoogleAPIKey()
		if keyErr != nil {
			logging.Printf("[3/4] Google Translate skipped — API key not available.\n")
			logging.Printf("       To enable: place your Google Cloud Translation API key in a file\n")
			logging.Printf("       named 'google_api.key' next to the executable.\n")
			logging.Printf("       Details: %v\n", keyErr)
			break
		}
		totalChars := countBookChars(book, outputDir)
		if totalChars > 1000 {
			estCost := float64(totalChars) / 1_000_000 * 20
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
		if exitCode, err := r.translateContent(book, outputDir, client); err != nil {
			return exitCode, err
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
		if exitCode, err := r.translateContent(book, outputDir, client); err != nil {
			return exitCode, err
		}
	default:
		logging.Println("[3/4] Translation skipped (use -google or -ollama to enable)")
	}

	// Generate TOC after translation so snippets reflect translated text.
	if len(book.Spine) > 1 {
		generatedIndex, err = htmlgen.GenerateIndex(book, outputDir)
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
func (r Runner) translateContent(book *epub.Book, outputDir string, client translator.Client) (int, error) {
	contentFiles := book.ContentFiles()
	total := len(contentFiles)
	if total == 0 {
		logging.Println("[3/4] No content files to translate")
		return ExitOK, nil
	}

	logging.Printf("[3/4] Translating %d pages...\n", total)

	for i, item := range contentFiles {
		href := item.Href
		if book.BasePath != "" && book.BasePath != "." {
			href = book.BasePath + "/" + href
		}
		filePath := filepath.Join(outputDir, filepath.FromSlash(href))

		// Extract texts first so we know segment count for ETA.
		segments, doc, err := htmlproc.ExtractTexts(filePath)
		if err != nil {
			logging.Errorf("  WARNING: skip %s: %v\n", item.Href, err)
			continue
		}

		if len(segments) == 0 {
			logging.Printf("  [%d/%d] %s (no text)\n", i+1, total, item.Href)
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
		pageIdx, pageTotal, pageName := i+1, total, item.Href
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
			logging.Printf("  [%d/%d] %s", i+1, total, item.Href)
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
			logging.Errorf("Translation failed at page %d/%d (%s)\n", i+1, total, item.Href)
			logging.Errorf("The book will be opened WITHOUT translation.\n")
			logging.Errorf("Press Enter to continue...\n")
			fmt.Scanln()
			return ExitOK, nil
		}

		// Replace text nodes with translations
		htmlproc.ReplaceTexts(segments, translated)

		// Write back
		if err := htmlproc.RenderToFile(doc, filePath); err != nil {
			logging.Errorf("  WARNING: write failed %s: %v\n", item.Href, err)
			continue
		}

		elapsed := time.Since(pageStart).Seconds()
		rate := float64(nSegs) / elapsed
		logging.Progress("  [%d/%d] %s: %d segs in %s (%.1f/s)\n",
			i+1, total, item.Href, nSegs, formatDuration(elapsed), rate)
	}

	// Translate book title.
	if book.Title != "" {
		if titles, err := client.Translate([]string{book.Title}, r.cfg.SourceLang, r.cfg.TargetLang); err == nil && len(titles) > 0 {
			book.Title = titles[0]
		}
	}

	logging.Println("  Translation complete.")
	return ExitOK, nil
}

// outputDirForFile returns the output directory path for a given input file.
// outputDirFor returns the output directory for a given input file.
// If folder is non-empty, the result is placed inside that folder.
// Otherwise it falls back to the directory of the input file (original behaviour).
//
// Example (folder=""):        /path/to/My Book.epub → /path/to/My Book/
// Example (folder="C:/out"): /path/to/My Book.epub → C:/out/My Book/
func outputDirFor(filePath, folder string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := sanitizeOutputName(strings.TrimSuffix(base, ext))
	if folder != "" {
		return filepath.Join(folder, name)
	}
	return filepath.Join(filepath.Dir(filePath), name)
}

func sanitizeOutputName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, ". ")
	if name == "" || name == "." || name == ".." {
		return "document"
	}

	if isWindowsReservedName(name) {
		name += "_"
	}

	return name
}

func isWindowsReservedName(name string) bool {
	upper := strings.ToUpper(name)
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" {
		return true
	}

	if strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT") {
		if len(upper) == 4 {
			n, err := strconv.Atoi(upper[3:])
			if err == nil && n >= 1 && n <= 9 {
				return true
			}
		}
	}

	return false
}

// countBookChars returns total number of translatable characters across all content files.
func countBookChars(book *epub.Book, outputDir string) int {
	total := 0
	for _, item := range book.ContentFiles() {
		href := item.Href
		if book.BasePath != "" && book.BasePath != "." {
			href = book.BasePath + "/" + href
		}
		filePath := filepath.Join(outputDir, filepath.FromSlash(href))
		segments, _, err := htmlproc.ExtractTexts(filePath)
		if err != nil {
			continue
		}
		for _, seg := range segments {
			total += len(seg.Text)
		}
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
