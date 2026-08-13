package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"doc-html-translate/internal/i18n"
)

// ErrHelp is returned when -h/-help was requested. Usage has already been written to stdout
// and the caller should exit 0 without reporting anything: help is a request, not a failure -
// the same treatment -version already gets.
var ErrHelp = flag.ErrHelp

type Config struct {
	Register         bool
	RegisterOpenWith bool
	Unregister       bool // -unregister: release the default-handler association
	FirstRun         bool // set when invoked with no args: non-destructive registration + opt-in prompt
	NoTranslate      bool
	NoOpen           bool
	UseGoogle        bool
	UseOllama        bool
	OllamaModel      string
	OllamaParallel   int
	OllamaNumCtx     int
	SplitSize        int
	TOCDepth         int
	SinglePage       bool // default true; -multipage disables it, producing per-chapter pages with a TOC
	OutputFolder     string
	Force            bool
	Verbose          bool
	SourceLang       string
	TargetLang       string
	MaxCost          float64
	OCR              bool   // -ocr: OCR document images and overlay translatable text
	OCRLang          string // -ocr-lang: tesseract language(s) for OCR (default: -src or eng)
	OCRList          bool   // -ocr-langs: list installed/available OCR languages and exit
	OCRDownload      string // -ocr-download <lang>: download an OCR language pack and exit
	UILang           string // -ui-lang: interface language of the console output and the page chrome ("" = from the OS)
	Report           bool   // -report: pack the recent run logs into an archive and exit
	InputFile        string
}

func ParseArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("doc-html-translate", flag.ContinueOnError)

	register := fs.Bool("register", false, "register app as document handler in HKCU")
	registerOpenWith := fs.Bool("register-openwith", false, `add app to the Windows "Open with" list without making it the default handler`)
	unregister := fs.Bool("unregister", false, `release the default-handler association (leaves the right-click "Convert to HTML" entry and "Open with")`)
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
	multiPage := fs.Bool("multipage", false, "produce multiple HTML pages with a table of contents instead of the default single page")
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
	uiLang := fs.String("ui-lang", "", "interface language for the console output and the converted page's navigation "+
		"(default: the Windows UI language); one of: "+strings.Join(i18n.Codes, " "))
	report := fs.Bool("report", false, "pack the recent run logs plus an environment summary into an archive for the author, then exit")
	version := fs.Bool("version", false, "print version and exit")

	// flag writes both the usage block and any parse error to a single writer, but the two
	// belong on different streams: -h is a request whose output a user pipes and reads, a bad
	// flag is a failure. Buffer flag's output and route it once Parse has told us which of the
	// two happened, rather than second-guessing it by scanning args for "-h" ourselves.
	var out bytes.Buffer
	fs.SetOutput(&out)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "Usage: doc-html-translate [flags] <file>\n\n"+
			"Flags may be written before or after the file; both orders mean the same thing.\n"+
			"A path that contains spaces needs quotes.\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(permuteArgs(fs, args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.Write(out.Bytes())
			return Config{}, ErrHelp
		}
		_, _ = os.Stderr.Write(out.Bytes())
		return Config{}, err
	}

	if *version {
		return Config{}, errors.New("version")
	}

	// An unknown -ui-lang is refused rather than silently ignored: the screenshot tooling and the
	// GUI both drive this flag, and a typo that quietly fell back to English would produce a set of
	// identical "translated" captures. The accepted list is i18n.Codes, so it cannot drift.
	if *uiLang != "" && i18n.Resolve(*uiLang, "", "") != *uiLang {
		return Config{}, fmt.Errorf("unknown -ui-lang %q: use one of %s", *uiLang, strings.Join(i18n.Codes, " "))
	}

	cfg := Config{
		Register:         *register,
		RegisterOpenWith: *registerOpenWith,
		Unregister:       *unregister,
		NoTranslate:      *noTranslate,
		NoOpen:           *noOpen,
		UseGoogle:        *useGoogle,
		UseOllama:        *useOllama || *useFree,
		OllamaModel:      *ollamaModel,
		OllamaParallel:   *ollamaParallel,
		OllamaNumCtx:     *ollamaNumCtx,
		SplitSize:        *splitSize,
		TOCDepth:         *tocDepth,
		SinglePage:       !*multiPage,
		OutputFolder:     *outputFolder,
		Force:            *force,
		Verbose:          *verbose,
		SourceLang:       *src,
		TargetLang:       *dst,
		MaxCost:          *maxCost,
		OCR:              *ocr,
		OCRLang:          *ocrLang,
		OCRList:          *ocrLangs,
		OCRDownload:      *ocrDownload,
		UILang:           *uiLang,
		Report:           *report,
	}

	// First-click UX: running without any args enters the first-run flow - a
	// non-destructive registration (right-click entry + "Open with") plus an
	// interactive opt-in prompt for the default handler. It no longer grabs defaults
	// silently (see app.Run / windowsreg).
	if len(args) == 0 {
		cfg.FirstRun = true
	}

	// Registration, OCR-management and report commands need no input file.
	if !cfg.Register && !cfg.RegisterOpenWith && !cfg.Unregister && !cfg.FirstRun && !cfg.OCRList && !cfg.Report && cfg.OCRDownload == "" {
		rest := fs.Args()
		if len(rest) == 0 {
			return Config{}, errors.New("input file is required unless -register is used")
		}
		// Everything past the first document used to be dropped without a word, which made the
		// commonest Windows mistake - an unquoted path with a space in it - look like a missing
		// file rather than a quoting error. One document per run is the real contract, so say so.
		if len(rest) > 1 {
			// Quoted with %s rather than %q: every path here is a Windows path, and %q doubles
			// the backslashes, so the message would show the user something they never typed.
			return Config{}, fmt.Errorf(`unexpected extra argument "%s" after "%s": this converts one document per run, `+
				`and a path containing spaces needs quotes ("C:\My Book.epub")`, rest[1], rest[0])
		}
		cfg.InputFile = rest[0]
	}

	return cfg, nil
}

// permuteArgs moves operands behind the flags so that `doc-html-translate book.epub -ocr
// -folder out` means what it says. Go's flag package stops at the first non-flag argument, so
// in that order every flag after the path was collected as a trailing operand and silently
// dropped: output landed beside the source, -force did nothing, and an explicit -max-cost
// vanished. Flags-first invocations are unaffected - they permute to themselves.
//
// Arity is read from the FlagSet rather than guessed, so `-split 5000 book.epub` keeps its
// value while `-ocr book.epub` does not swallow the path. An undefined flag consumes nothing
// and is passed through to Parse, whose error message for it is the one worth showing. A
// literal "--" ends flag parsing here exactly as it does in flag itself, which is how a file
// whose name begins with a dash is passed.
func permuteArgs(fs *flag.FlagSet, args []string) []string {
	flags := make([]string, 0, len(args))
	operands := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		// A lone "-" and anything not starting with one is a document, not a switch - the
		// same test flag's own parser applies before it stops.
		if len(arg) < 2 || arg[0] != '-' {
			operands = append(operands, arg)
			continue
		}
		flags = append(flags, arg)
		if strings.Contains(arg, "=") {
			continue // -name=value carries its own value
		}
		f := fs.Lookup(strings.TrimLeft(arg, "-"))
		if f == nil {
			continue
		}
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue // -ocr, -force, .. never take a following token
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if len(operands) == 0 {
		return flags
	}
	// The operands are fenced off with "--" so a document named like a flag survives the
	// second pass through Parse.
	out := make([]string, 0, len(flags)+len(operands)+1)
	out = append(out, flags...)
	out = append(out, "--")
	return append(out, operands...)
}
