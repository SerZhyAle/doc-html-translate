package config

import (
	"errors"
	"flag"
)

type Config struct {
	Register       bool
	NoTranslate    bool
	NoOpen         bool
	UseGoogle      bool
	UseOllama      bool
	OllamaModel    string
	OllamaParallel int
	OllamaNumCtx   int
	SplitSize      int
	TOCDepth       int
	SinglePage     bool // -single: merge the whole document into one HTML page (no TOC)
	OutputFolder   string
	Force          bool
	Verbose        bool
	SourceLang     string
	TargetLang     string
	MaxCost        float64
	OCR            bool   // -ocr: OCR document images and overlay translatable text
	OCRLang        string // -ocr-lang: tesseract language(s) for OCR (default: -src or eng)
	OCRList        bool   // -ocr-langs: list installed/available OCR languages and exit
	OCRDownload    string // -ocr-download <lang>: download an OCR language pack and exit
	InputFile      string
}

func ParseArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("doc-html-translate", flag.ContinueOnError)

	register := fs.Bool("register", false, "register app as document handler in HKCU")
	noTranslate := fs.Bool("notranslate", false, "convert only, skip translation")
	noOpen := fs.Bool("noopen", false, "do not open browser after conversion (batch mode)")
	useGoogle := fs.Bool("google", false, "translate using Google Translate API")
	useOllama := fs.Bool("ollama", false, "translate using local Ollama (default model: gemma3:12b)")
	useFree := fs.Bool("free", false, "alias for -ollama: translate using local Ollama")
	ollamaModel := fs.String("ollama-model", "gemma3:12b", "Ollama model name")
	ollamaParallel := fs.Int("ollama-parallel", 1, "concurrent batch requests (set OLLAMA_NUM_PARALLEL=N on Ollama side too)")
	ollamaNumCtx := fs.Int("ollama-ctx", 8192, "context window size in tokens sent to Ollama")
	splitSize := fs.Int("split", 5000, "split pages at N chars for browser GT extension (0 = disable)")
	tocDepth := fs.Int("toc-depth", 0, "table-of-contents nesting depth shown on index.html (0 = unlimited)")
	singlePage := fs.Bool("single", false, "combine the whole document into a single HTML page (no table of contents)")
	outputFolder := fs.String("folder", "", "output folder (default: same directory as input file)")
	force := fs.Bool("force", false, "re-extract and re-translate even if output already exists (yes, all over again)")
	verbose := fs.Bool("v", false, "verbose output - more than you ever wanted to know")
	src := fs.String("src", "en", "source language")
	dst := fs.String("dst", "ru", "target language")
	maxCost := fs.Float64("max-cost", 0, "abort paid translation before your wallet notices if the USD estimate exceeds N (0 = live dangerously, no limit)")
	ocr := fs.Bool("ocr", false, "OCR text inside document images and overlay it as translatable HTML (needs tesseract)")
	ocrLang := fs.String("ocr-lang", "", "OCR language(s) for -ocr, e.g. eng or eng+rus (default: -src, else eng)")
	ocrLangs := fs.Bool("ocr-langs", false, "list installed and available OCR languages, then exit")
	ocrDownload := fs.String("ocr-download", "", "download an OCR language pack (e.g. rus) into the app's tessdata, then exit")
	version := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if *version {
		return Config{}, errors.New("version")
	}

	cfg := Config{
		Register:       *register,
		NoTranslate:    *noTranslate,
		NoOpen:         *noOpen,
		UseGoogle:      *useGoogle,
		UseOllama:      *useOllama || *useFree,
		OllamaModel:    *ollamaModel,
		OllamaParallel: *ollamaParallel,
		OllamaNumCtx:   *ollamaNumCtx,
		SplitSize:      *splitSize,
		TOCDepth:       *tocDepth,
		SinglePage:     *singlePage,
		OutputFolder:   *outputFolder,
		Force:          *force,
		Verbose:        *verbose,
		SourceLang:     *src,
		TargetLang:     *dst,
		MaxCost:        *maxCost,
		OCR:            *ocr,
		OCRLang:        *ocrLang,
		OCRList:        *ocrLangs,
		OCRDownload:    *ocrDownload,
	}

	// First-click UX: running without any args behaves as registration mode.
	if len(args) == 0 {
		cfg.Register = true
	}

	// OCR language management commands need no input file.
	if !cfg.Register && !cfg.OCRList && cfg.OCRDownload == "" {
		rest := fs.Args()
		if len(rest) == 0 {
			return Config{}, errors.New("input file is required unless -register is used")
		}
		cfg.InputFile = rest[0]
	}

	return cfg, nil
}
