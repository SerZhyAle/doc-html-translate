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
	OutputFolder   string
	Force          bool
	Verbose        bool
	SourceLang     string
	TargetLang     string
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
	outputFolder := fs.String("folder", "", "output folder (default: same directory as input file)")
	force := fs.Bool("force", false, "force re-extract and re-translate even if output already exists")
	verbose := fs.Bool("v", false, "verbose output (debug)")
	src := fs.String("src", "en", "source language")
	dst := fs.String("dst", "ru", "target language")
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
		OutputFolder:   *outputFolder,
		Force:          *force,
		Verbose:        *verbose,
		SourceLang:     *src,
		TargetLang:     *dst,
	}

	// First-click UX: running without any args behaves as registration mode.
	if len(args) == 0 {
		cfg.Register = true
	}

	if !cfg.Register {
		rest := fs.Args()
		if len(rest) == 0 {
			return Config{}, errors.New("input file is required unless -register is used")
		}
		cfg.InputFile = rest[0]
	}

	return cfg, nil
}
