package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/browser"
	"doc-html-translate/internal/comic"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fb2"
	"doc-html-translate/internal/htmlconv"
	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/htmlsplit"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/img"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/md"
	"doc-html-translate/internal/mobi"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/pdf"
	"doc-html-translate/internal/rtf"
	"doc-html-translate/internal/txt"
)

// ExitCode constants for structured error handling. The values are a published contract for the
// programs that call this CLI - OCR-INVOCATION.md section 1 - so a code never changes meaning and
// a new one is a contract amendment first.
const (
	ExitOK        = 0
	ExitArgsError = 1
	ExitIOError   = 2
	ExitEPUB      = 3
	ExitParse     = 3 // alias: same code for any parse error (EPUB or PDF)
	ExitAPI       = 4 // translation failed or stopped part-way; the book is still produced
	// ExitInterrupted is the shell's code for a run stopped by Ctrl+C, which the Ollama path
	// already returned before cancellation became cooperative.
	ExitInterrupted = 130
)

type Runner struct {
	cfg     config.Config
	engines engines
}

func NewRunner(cfg config.Config) Runner {
	return Runner{cfg: cfg, engines: defaultEngines()}
}

// run executes the file-to-HTML pipeline; RunContext wraps it in the panic guard. Cancelling
// ctx (Ctrl+C) stops it between steps, pages, batches and OCR images; the output then has no
// completion record, so the next run rebuilds it instead of opening it.
// Steps: [1] Check existing / Extract -> [2] Build HTML -> [3] Translate -> [4] Open browser.
func (r Runner) run(ctx context.Context) (int, error) {
	inputPath, err := filepath.Abs(r.cfg.InputFile)
	if err != nil {
		return ExitIOError, fmt.Errorf("resolve input path: %w", err)
	}

	info, err := os.Stat(inputPath)
	switch {
	case os.IsNotExist(err):
		return ExitArgsError, fmt.Errorf("file not found: %s", inputPath)
	case err != nil:
		return ExitIOError, fmt.Errorf("cannot read input: %w", err)
	case info.IsDir():
		// A folder maps its output onto itself (no extension to strip), and a failed
		// "plain text" read of it then deleted the folder with everything in it.
		return ExitArgsError, fmt.Errorf("%s is a folder, not a document - pass a file", inputPath)
	case !info.Mode().IsRegular():
		return ExitArgsError, fmt.Errorf("%s is not a regular file", inputPath)
	case info.Size() == 0:
		return ExitArgsError, fmt.Errorf("%s is empty (0 bytes)", inputPath)
	}
	if f, err := os.Open(inputPath); err != nil {
		return ExitIOError, fmt.Errorf("cannot read input: %w", err)
	} else {
		_ = f.Close()
	}

	// Output directory: same location as file (or -folder path), named after the file
	// (without extension) - unless that name is taken by another document or by a
	// folder that is not ours, in which case a suffixed sibling is used.
	target, err := outputpath.Resolve(inputPath, r.cfg.OutputFolder)
	if err != nil {
		return ExitIOError, err
	}
	outputDir := target.Dir
	if outputDir != target.Preferred {
		logging.Printf("  %s is taken by other content - using %s\n", target.Preferred, outputDir)
	}
	indexPath := filepath.Join(outputDir, "index.html")

	// R4: an output that finished, from this very document, with the same result-affecting
	// options is opened as it is. Anything else - interrupted, partially translated, built with
	// other settings - is rebuilt, and the log says why.
	if _, err := os.Stat(indexPath); err == nil && target.State.Ours() && !r.cfg.Force {
		if outputpath.Locked(outputDir) {
			return ExitIOError, fmt.Errorf("%s: %w", outputDir, outputpath.ErrLocked)
		}
		reason, changed := outputpath.CheckReuse(outputDir, inputPath, outputpath.OptionsFor(r.cfg))
		if reason != outputpath.ReuseOK {
			logging.Println(rebuildMessage(reason, changed))
			return r.build(ctx, inputPath, target)
		}
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
	if target.State.Ours() && r.cfg.Force {
		logging.Printf("Book already extracted, forcing rebuild: %s\n", outputDir)
	}
	return r.build(ctx, inputPath, target)
}

// build converts inputPath into target.Dir from scratch.
func (r Runner) build(ctx context.Context, inputPath string, target outputpath.Target) (int, error) {
	outputDir := target.Dir
	ext := strings.ToLower(filepath.Ext(inputPath))

	// Step 1: Extract (format-specific)
	claim, err := claimOutputDir(target, inputPath)
	if err != nil {
		return ExitIOError, err
	}
	defer claim.release()
	cleanup := claim.cleanup

	var book *epub.Book
	// A standalone image has no text to extract: wrap it in a one-page HTML doc and
	// force the OCR overlay below, so the browser shows the picture with translatable
	// text plates laid over it (mirrors the extension's image OCR overlay).
	forceOCR := false
	if img.IsImage(ext) {
		logging.Println("[1/4] Preparing image..")
		book, err = img.Extract(inputPath, outputDir)
		if err != nil {
			cleanup()
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
		book, err = comic.Extract(ctx, inputPath, outputDir)
		if err != nil {
			cleanup()
			return extractFailed(ctx, fmt.Errorf("extract comic: %w", err))
		}
		forceOCR = true
	} else {
		switch ext {
		case ".epub":
			logging.Println("[1/4] Extracting EPUB..")
			book, err = epub.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitEPUB, fmt.Errorf("extract epub: %w", err)
			}
			logging.Printf("  Title: %s\n", book.Title)
			logging.Printf("  Chapters: %d\n", len(book.Spine))
		case ".pdf":
			logging.Println("[1/4] Extracting PDF..")
			book, err = pdf.Extract(ctx, inputPath, outputDir)
			if err != nil {
				cleanup()
				return extractFailed(ctx, fmt.Errorf("extract pdf: %w", err))
			}
		case ".txt":
			logging.Println("[1/4] Extracting TXT..")
			book, err = txt.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitParse, fmt.Errorf("extract txt: %w", err)
			}
		case ".md":
			logging.Println("[1/4] Extracting Markdown..")
			book, err = md.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitParse, fmt.Errorf("extract markdown: %w", err)
			}
		case ".fb2":
			logging.Println("[1/4] Extracting FB2..")
			book, err = fb2.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitParse, fmt.Errorf("extract fb2: %w", err)
			}
		case ".rtf":
			logging.Println("[1/4] Extracting RTF..")
			book, err = rtf.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitParse, fmt.Errorf("extract rtf: %w", err)
			}
		case ".html", ".htm":
			logging.Println("[1/4] Extracting HTML..")
			book, err = htmlconv.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitParse, fmt.Errorf("extract html: %w", err)
			}
		case ".mobi", ".azw3":
			logging.Println("[1/4] Extracting MOBI..")
			book, err = mobi.Extract(ctx, inputPath, outputDir)
			if err != nil {
				cleanup()
				return extractFailed(ctx, fmt.Errorf("extract mobi: %w", err))
			}
		default:
			// Unknown extension: treat it as plain text - but only if it is text. A binary
			// (a .docx, a .djvu, a comic archive) handed to the text extractor became a
			// multi-megabyte document of raw bytes rendered as prose, reported as success. The
			// browser extension routes on the byte signature and refuses these; this matches it.
			if head, herr := readHead(inputPath, 4096); herr == nil {
				if desc := txt.LooksBinary(head); desc != "" {
					cleanup()
					return ExitParse, fmt.Errorf("%s looks like %s, not a text document - refusing to convert it into garbage",
						filepath.Base(inputPath), desc)
				}
			}
			logging.Printf("[1/4] Unknown extension %q - reading as plain text..\n", ext)
			book, err = txt.Extract(inputPath, outputDir)
			if err != nil {
				cleanup()
				return ExitParse, fmt.Errorf("extract as txt: %w", err)
			}
		}
	}
	if ctx.Err() != nil {
		return interrupted()
	}

	// Optional: split oversized pages at paragraph boundaries so browser
	// translation extensions (Chrome GT: ~5000 chars) can handle each page.
	// Single-page mode merges everything into one file anyway, so splitting is moot.
	if r.cfg.SplitSize > 0 && !r.cfg.SinglePage {
		n, err := htmlsplit.SplitIfNeeded(book, outputDir, r.cfg.SplitSize)
		if err != nil {
			cleanup()
			return ExitIOError, fmt.Errorf("split pages: %w", err)
		}
		if n > 0 {
			logging.Printf("  Split: %d additional pages created (%d total, max %d chars each)\n",
				n, len(book.Spine), r.cfg.SplitSize)
		}
	}

	// Fixed here, before translation rewrites the title: every page and index.html must
	// namespace the saved reading position under this one key. The page count is taken
	// after the split, so a layout change does not offer a position in a page that is gone.
	var srcSize int64
	if fi, statErr := os.Stat(inputPath); statErr == nil {
		srcSize = fi.Size()
	}
	book.ReaderKey = htmlgen.ReaderKey(filepath.Base(inputPath), srcSize, book.Title, len(book.Spine))

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
		// Single page - no TOC, no navigation bars needed.
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
	if (r.cfg.OCR || forceOCR) && ctx.Err() == nil {
		r.overlayImagesSafe(ctx, book, outputDir)
	}
	if ctx.Err() != nil {
		return interrupted()
	}

	// Step 3: Translation
	outcome := r.translate(ctx, book, outputDir)
	if ctx.Err() != nil {
		return interrupted()
	}

	// Generate TOC after translation so snippets reflect translated text.
	if len(book.Spine) > 1 {
		// No authored TOC (NCX/nav for EPUB, bookmarks for PDF)? Synthesize a
		// multi-level TOC from the headings on the now-final pages, injecting
		// stable anchors. Runs on the translated files so labels are translated.
		if len(book.TOC) == 0 {
			book.TOC = htmlgen.BuildFallbackTOC(book, outputDir, outcome.snippets)
		}

		generatedIndex, err = htmlgen.GenerateIndexWithSnippetsDepth(book, outputDir, outcome.snippets, r.cfg.TOCDepth)
		if err != nil {
			return ExitIOError, fmt.Errorf("generate index: %w", err)
		}
		logging.Printf("  TOC created: %s\n", generatedIndex)
	}

	// The very last write: from here on the output counts as finished and may be reopened.
	// Without it the next run rebuilds, so a failure to write it costs time, not correctness.
	if err := outputpath.MarkComplete(outputDir, inputPath, outputpath.Completion{
		ToolVersion: logging.AppVersion,
		Options:     outputpath.OptionsFor(r.cfg),
		OCRForced:   forceOCR,
		Translation: outcome.state,
	}); err != nil {
		logging.Errorf("  WARNING: could not record the finished output: %v\n", err)
	}

	// Step 4: Open in browser - a partial translation is still opened, it is the book the
	// reader asked for with some pages in the source language.
	if r.cfg.NoOpen {
		logging.Println("[4/4] Browser open skipped (-noopen)")
	} else {
		logging.Println("[4/4] Opening in browser..")
		if err := browser.Open(generatedIndex); err != nil {
			return ExitIOError, fmt.Errorf("open browser: %w", err)
		}
	}

	if outcome.state == outputpath.TranslationPartial {
		return ExitAPI, outcome.partialError()
	}
	if outcome.err != nil {
		return ExitAPI, outcome.err
	}
	logging.Println("Done.")
	return ExitOK, nil
}

// extractFailed classifies an extraction that returned an error. A helper killed because the
// run was cancelled fails like a broken file, but the reader pressed Ctrl+C: that is an
// interrupted run, not a parse failure.
func extractFailed(ctx context.Context, err error) (int, error) {
	if ctx.Err() != nil {
		return interrupted()
	}
	return ExitParse, err
}

func interrupted() (int, error) {
	return ExitInterrupted, fmt.Errorf("%s", i18n.S("interrupted - the output is incomplete and will be rebuilt on the next run"))
}

// readHead returns up to n leading bytes of a file, for format sniffing. A short read (the
// file is smaller than n) is not an error - it returns what there was.
func readHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, n)
	got, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:got], nil
}
