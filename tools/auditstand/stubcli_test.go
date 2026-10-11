package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"doc-html-translate/internal/outputpath"
)

// The test binary doubles as a stand-in for doc-html-translate.exe: when AUDITSTAND_STUB_CLI=1 it
// behaves like the CLI's translation-refusal paths (or deliberately misbehaves, per
// AUDITSTAND_STUB_MODE) instead of running tests. The stand's judgement - what it calls a
// pass, a leak, an English message under -ui-lang ru - is what these tests pin; the real CLI's
// wording is pinned separately against the pipeline source.
func TestMain(m *testing.M) {
	if os.Getenv("AUDITSTAND_STUB_CLI") == "1" {
		os.Exit(stubCLI(os.Args[1:]))
	}
	os.Exit(m.Run())
}

func stubCLI(args []string) int {
	var (
		src, folder, uiLang string
		google, ollama, ocr bool
		force               bool
		maxCost             float64
	)
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "-google":
			google = true
		case "-ollama":
			ollama = true
		case "-ocr":
			ocr = true
		case "-force":
			force = true
		case "-noopen":
		case "-max-cost":
			i++
			maxCost, _ = strconv.ParseFloat(args[i], 64)
		case "-folder":
			i++
			folder = args[i]
		case "-ui-lang":
			i++
			uiLang = args[i]
		case "-ollama-model":
			i++
		default:
			src = a
		}
	}
	mode := os.Getenv("AUDITSTAND_STUB_MODE")
	local := os.Getenv("LOCALAPPDATA")
	appDir := filepath.Join(local, "doc-html-translate")
	logDir := filepath.Join(appDir, "logs")
	_ = os.MkdirAll(logDir, 0o755)
	logFile, _ := os.Create(filepath.Join(logDir, "run-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".log"))
	say := func(format string, a ...interface{}) {
		line := fmt.Sprintf(format, a...)
		fmt.Println(line)
		if logFile != nil {
			fmt.Fprintln(logFile, line)
		}
	}

	if strings.HasSuffix(src, ".fd-sec") {
		// Locate runs first: an override that points at nothing is NotFound, whatever is
		// installed, and nothing is created.
		if p := os.Getenv("DOCHT_FILEDO"); p != "" {
			if _, err := os.Stat(p); err != nil {
				msg := "Error: audit-container.fd-sec looks like a FileDO secret file. Opening it needs FileDO, which was not found on this computer. No password was asked and nothing was changed."
				if mode == "filedo-asks-password" {
					msg = "Error: password dialog opened before looking for FileDO"
				}
				fmt.Fprintln(os.Stderr, msg)
				if logFile != nil {
					fmt.Fprintln(logFile, msg)
				}
				return 3
			}
		}
	}

	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	switch {
	case strings.Contains(name, "over-cap"):
		say("[1/4] Preparing image..")
		say("Error: prepare image: The image is 60000 x 60000 pixels, above the limit of 100 megapixels and 32768 pixels per side")
		return 3
	case name == "comic-big-2gb":
		say("[1/4] Extracting comic..")
		say("  WARNING: page0001.png unpacks to more than 200 MB, the limit for one file")
		say("Error: no page images found in comic-big-2gb.cbz")
		return 3
	}
	bookDir := filepath.Join(folder, name)
	// The completion record decides reuse; -force rebuilds and says so (the pipeline's lines).
	if m, err := outputpath.ReadMarker(bookDir); err == nil && m.Complete != nil && mode != "no-reuse" {
		if force {
			say("Book already extracted, forcing rebuild: %s", bookDir)
		} else {
			say("Book already extracted: %s", bookDir)
			say("Done.")
			return 0
		}
	}
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		say("Error: %v", err)
		return 2
	}
	_ = os.WriteFile(filepath.Join(bookDir, "index.html"), []byte("<html><body>book</body></html>"), 0o644)
	finish := func(translation string, code int) int {
		if mode == "claims-full" {
			translation = "full"
		}
		_ = outputpath.MarkComplete(bookDir, src, outputpath.Completion{Translation: translation})
		return code
	}

	say("[1/4] Extracting EPUB..")
	switch {
	case strings.HasSuffix(src, ".rtf"):
		say("  Paragraphs: 2, Pages: 1")
		say("Done.")
		return finish("none", 0)
	case name == "tiff-multiframe":
		say("  Image: tiff-multiframe.tif (3-page TIFF, one PNG page per frame)")
		say("Done.")
		return finish("none", 0)
	case name == "comic-2gb-pages":
		say("  Pages: 256")
		say("Done.")
		return finish("none", 0)
	case ocr && name == "epub-ocr-halfimage":
		say("  OCR overlay: 3 images in 0.8s (3.9/s)")
		if mode == "ocr-aborts" {
			say("  WARNING: OCR failed on page2.png: tesseract: exit status 1: libpng error: Read Error")
			return 3 // the regression this models: one failed image takes the book with it
		}
		if mode == "ocr-silent" {
			say("  OCR overlay: 2 image(s) overlaid")
			return finish("none", 0)
		}
		say("  OCR overlay: 2 image(s) overlaid, 1 failed")
		say("  WARNING: OCR failed on page2.png: tesseract: exit status 1: libpng error: Read Error")
		say("Done.")
		return finish("none", 0)
	case google:
		keyFile := filepath.Join(appDir, "google_api.key")
		raw, err := os.ReadFile(keyFile)
		if err != nil {
			say("[3/4] Google Translate skipped - API key not available.")
			if mode == "exit0-nokey" {
				return finish("none", 0)
			}
			return finish("none", 4)
		}
		if mode != "no-migrate" {
			_ = os.WriteFile(filepath.Join(appDir, "google_api.key.dpapi"), []byte("protected-blob"), 0o600)
			_ = os.Remove(keyFile)
			say("Migrated Google API key to Windows DPAPI storage; removed plaintext key file.")
		}
		if mode == "leak-key" {
			say("debug: using key %s", strings.TrimSpace(string(raw)))
		}
		const estimate = 0.05
		if maxCost > 0 && estimate > maxCost {
			if uiLang == "ru" && mode != "english-ru" {
				say("[3/4] Перевод пропущен - ориентировочная стоимость $%.2f USD превышает лимит -max-cost $%.2f", estimate, maxCost)
			} else {
				say("[3/4] Translation skipped - estimated cost $%.2f USD (2,464 characters) exceeds -max-cost $%.2f limit", estimate, maxCost)
			}
			return finish("none", 0)
		}
		say("[3/4] Translating..")
		return finish("full", 0)
	case ollama:
		say("TRANSLATION ERROR: ollama: http request: dial tcp: connection refused (is Ollama running?)")
		return finish("partial", 4)
	}
	say("Done.")
	return finish("none", 0)
}
