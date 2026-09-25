package pipeline

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/htmlproc"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/translator"

	gohtml "golang.org/x/net/html"
)

type contentPage struct {
	item      epub.ManifestItem
	filePath  string
	segments  []*htmlproc.TextSegment
	doc       *gohtml.Node
	err       error
	charCount int
}

// engines builds the translation clients. It is a field so the tests can put a stub engine
// behind the real page loop instead of a network service.
type engines struct {
	googleKey func() (string, error)
	google    func(apiKey string) translator.Client
	ollama    func(cfg config.Config) translator.Client
	confirm   func(title, message string) bool
}

func defaultEngines() engines {
	return engines{
		googleKey: translator.LoadGoogleAPIKey,
		google:    func(key string) translator.Client { return translator.NewGoogleClient(key) },
		ollama: func(cfg config.Config) translator.Client {
			c := translator.NewOllamaClient(cfg.OllamaModel)
			c.SetParallelism(cfg.OllamaParallel)
			c.SetNumCtx(cfg.OllamaNumCtx)
			return c
		},
		confirm: dialog.ConfirmYesNo,
	}
}

// translationOutcome is what the translation step did, in the terms the completion record and
// the exit code need: how many pages with text there were, how many came back fully translated,
// and the engine error that stopped it, if any.
type translationOutcome struct {
	state        string // an outputpath.Translation* value
	pages, done  int
	labelsFailed bool
	err          error
	snippets     map[string]string
}

// partialError is the run's error for a translation that did not finish. The book is still
// produced and opened; the exit code says it is not what was asked for.
func (o translationOutcome) partialError() error {
	msg := i18n.S("partially translated, %d of %d pages", o.done, o.pages)
	if o.done == o.pages && o.labelsFailed {
		msg = i18n.S("translated, but the title or table of contents labels were not")
	}
	if o.err != nil {
		return fmt.Errorf("%s: %w", msg, o.err)
	}
	return errors.New(msg)
}

// translate runs the configured engine over the book. It never fails the conversion: an engine
// problem is reported in the outcome, and a cancelled ctx stops it between pages.
func (r Runner) translate(ctx context.Context, book *epub.Book, outputDir string) translationOutcome {
	none := translationOutcome{state: outputpath.TranslationNone}

	// Advisory: when a translation engine is on, warn once if a language code looks
	// malformed (e.g. "russian" instead of "ru"). Non-fatal - the engine still runs.
	if (r.cfg.UseGoogle || r.cfg.UseOllama) && !r.cfg.NoTranslate {
		if bad := config.SuspiciousLangCodes(r.cfg.SourceLang, r.cfg.TargetLang); len(bad) > 0 {
			logging.Errorf("WARNING: %s does not look like a language code (expected e.g. 'en', 'ru', 'zh-CN'); translation may be wrong.\n", strings.Join(bad, " and "))
		}
	}

	switch {
	case r.cfg.NoTranslate:
		logging.Println("[3/4] Translation skipped (-notranslate)")
		return none
	case r.cfg.UseGoogle:
		apiKey, keyErr := r.engines.googleKey()
		if keyErr != nil {
			logging.Printf("[3/4] Google Translate skipped - API key not available.\n")
			logging.Printf("       To enable: save your Google Cloud Translation API key as 'google_api.key' in either:\n")
			for _, p := range translator.GoogleAPIKeyPaths() {
				logging.Printf("         %s\n", p)
			}
			logging.Printf("       Details: %v\n", keyErr)
			return none
		}
		pages := loadContentPages(book, outputDir)
		if !r.approveGoogleCost(pages) {
			return none
		}
		client := translator.NewCachingClient(r.engines.google(apiKey))
		return r.translateContent(ctx, book, client, pages)
	case r.cfg.UseOllama:
		worker := r.engines.ollama(r.cfg)
		// Ctrl+C cancels ctx; the model is released from VRAM on the way out rather than from a
		// signal goroutine that exited the process in the middle of a page write.
		if u, ok := worker.(interface{ Unload() }); ok {
			defer func() {
				if ctx.Err() != nil {
					logging.Println("Interrupted. Unloading Ollama model from VRAM..")
					u.Unload()
				}
			}()
		}
		client := translator.NewCachingClient(worker)
		return r.translateContent(ctx, book, client, loadContentPages(book, outputDir))
	default:
		logging.Println("[3/4] Translation skipped (use -google or -ollama to enable)")
		return none
	}
}

// approveGoogleCost applies the -max-cost guard and the confirmation dialog.
func (r Runner) approveGoogleCost(pages []contentPage) bool {
	totalChars := countLoadedPageChars(pages)
	if totalChars <= 1000 {
		return true
	}
	estCost := float64(totalChars) / 1_000_000 * 20
	if r.cfg.MaxCost > 0 && estCost > r.cfg.MaxCost {
		logging.Printf("[3/4] Translation skipped - estimated cost $%.2f USD exceeds -max-cost $%.2f limit\n", estCost, r.cfg.MaxCost)
		return false
	}
	msg := fmt.Sprintf(
		"Characters to send: %s\nEstimated cost: $%.2f USD\n\nProceed with Google Translate?",
		formatInt(totalChars), estCost,
	)
	if !r.engines.confirm("Google Translate - Cost Warning", msg) {
		logging.Println("[3/4] Translation cancelled by user")
		return false
	}
	return true
}

// translateContent translates all HTML content files in the book. The first engine failure
// stops it: the page in hand keeps whatever part of it did come back, every later page stays in
// the source language, and the outcome says how many pages are fully translated.
func (r Runner) translateContent(ctx context.Context, book *epub.Book, client translator.Client, pages []contentPage) translationOutcome {
	out := translationOutcome{state: outputpath.TranslationFull}
	total := len(pages)
	if total == 0 {
		logging.Println("[3/4] No content files to translate")
		return out
	}

	logging.Printf("[3/4] Translating %d pages..\n", total)
	out.snippets = make(map[string]string, total)

	for i, page := range pages {
		if ctx.Err() != nil {
			return out
		}
		if page.err != nil {
			logging.Errorf("  WARNING: skip %s: %v\n", page.item.Href, page.err)
			out.pages++
			continue
		}
		if len(page.segments) == 0 {
			out.snippets[page.item.Href] = htmlgen.ExtractSnippetFromDoc(page.doc)
			logging.Printf("  [%d/%d] %s (no text)\n", i+1, total, page.item.Href)
			continue
		}
		out.pages++
		if out.err != nil {
			continue // counted, left in the source language
		}
		if r.translatePage(ctx, &out, client, page, i+1, total) {
			out.done++
		}
	}
	if ctx.Err() != nil {
		return out
	}

	if out.err == nil {
		r.translateLabels(ctx, book, client, &out)
	}
	if out.done < out.pages || out.labelsFailed {
		out.state = outputpath.TranslationPartial
	} else {
		logging.Println("  Translation complete.")
	}
	return out
}

// translatePage translates and rewrites one page, reporting whether every segment on it was
// translated. An engine failure is recorded in out and stops the book.
func (r Runner) translatePage(ctx context.Context, out *translationOutcome, client translator.Client, page contentPage, idx, total int) bool {
	texts := make([]string, len(page.segments))
	for j, seg := range page.segments {
		texts[j] = seg.Text
	}

	// Set up per-page progress display (overwrites line with \r).
	pageStart := time.Now()
	nSegs := len(texts)
	if pr, ok := client.(translator.ProgressReporter); ok {
		pr.SetProgress(func(done, ttl int) {
			elapsed := time.Since(pageStart).Seconds()
			rate := float64(done) / elapsed
			etaStr := ""
			if rate > 0 {
				etaStr = " ETA " + formatDuration(float64(ttl-done)/rate)
			}
			logging.Progress("  [%d/%d] %s: %d/%d segs  %.1f/s%s     ",
				idx, total, page.item.Href, done, ttl, rate, etaStr)
		})
		defer pr.SetProgress(nil)
	} else {
		logging.Printf("  [%d/%d] %s", idx, total, page.item.Href)
	}

	translated, err := client.Translate(ctx, texts, r.cfg.SourceLang, r.cfg.TargetLang)
	var partial *translator.PartialError
	switch {
	case ctx.Err() != nil:
		return false
	case errors.As(err, &partial):
		logging.Errorf("\nTRANSLATION ERROR: %v\n", err)
		logging.Errorf("Translation stopped at page %d/%d (%s): %d of %d segments kept\n",
			idx, total, page.item.Href, nSegs-len(partial.Missing), nSegs)
		out.err = partial.Err
	case err != nil:
		logging.Errorf("\nTRANSLATION ERROR: %v\n", err)
		logging.Errorf("Translation stopped at page %d/%d (%s)\n", idx, total, page.item.Href)
		out.err = err
		return false
	}

	// ReplaceTexts keeps the source text in every empty slot, which is where a partial reply
	// left the segments that did not come back.
	htmlproc.ReplaceTexts(page.segments, translated)
	if err := htmlproc.RenderToFile(page.doc, page.filePath); err != nil {
		logging.Errorf("  WARNING: write failed %s: %v\n", page.item.Href, err)
		return false
	}
	out.snippets[page.item.Href] = htmlgen.ExtractSnippetFromDoc(page.doc)
	if partial != nil {
		return false
	}

	elapsed := time.Since(pageStart).Seconds()
	rate := float64(nSegs) / elapsed
	logging.Progress("  [%d/%d] %s: %d segs in %s (%.1f/s)\n",
		idx, total, page.item.Href, nSegs, formatDuration(elapsed), rate)
	return true
}

// translateLabels translates the book title and the authored TOC labels (EPUB NCX/nav, PDF
// bookmarks). The heading-scan fallback is built later from already-translated pages, so it is
// not translated here. The shared CachingClient keeps labels consistent with identical in-page
// headings. A failure leaves the labels in the source language and marks the outcome partial.
func (r Runner) translateLabels(ctx context.Context, book *epub.Book, client translator.Client, out *translationOutcome) {
	if book.Title != "" {
		titles, err := client.Translate(ctx, []string{book.Title}, r.cfg.SourceLang, r.cfg.TargetLang)
		switch {
		case err == nil && len(titles) == 1:
			if titles[0] != "" {
				book.Title = titles[0]
			}
		case ctx.Err() != nil:
			return
		default:
			logging.Errorf("  WARNING: title not translated: %v\n", err)
			out.labelsFailed = true
		}
	}

	if len(book.TOC) == 0 {
		return
	}
	var titles []string
	collectTOCTitles(book.TOC, &titles)
	if len(titles) == 0 {
		return
	}
	translated, err := client.Translate(ctx, titles, r.cfg.SourceLang, r.cfg.TargetLang)
	if err != nil || len(translated) != len(titles) {
		if ctx.Err() == nil {
			logging.Errorf("  WARNING: table of contents not translated: %v\n", err)
			out.labelsFailed = true
		}
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

func contentFilePath(book *epub.Book, outputDir string, item epub.ManifestItem) string {
	href := item.Href
	if book.BasePath != "" && book.BasePath != "." {
		href = book.BasePath + "/" + href
	}
	return filepath.Join(outputDir, filepath.FromSlash(href))
}

func loadContentPages(book *epub.Book, outputDir string) []contentPage {
	contentFiles := book.ContentFiles()
	pages := make([]contentPage, 0, len(contentFiles))
	for _, item := range contentFiles {
		filePath := contentFilePath(book, outputDir, item)
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
