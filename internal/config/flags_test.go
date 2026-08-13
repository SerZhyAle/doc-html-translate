package config

import (
	"errors"
	"strings"
	"testing"
)

// -h is a request, not a failure. It must come back as ErrHelp so main can exit 0 with the
// usage on stdout, the same way -version already exits 0 - not as a generic error that gets
// an "Error:" prefix and exit 1.
func TestParseArgsHelpIsNotAnError(t *testing.T) {
	for _, arg := range []string{"-h", "-help"} {
		_, err := ParseArgs([]string{arg})
		if !errors.Is(err, ErrHelp) {
			t.Fatalf("ParseArgs(%q): expected ErrHelp, got %v", arg, err)
		}
	}
}

// A real flag error must stay an error, so it keeps its non-zero exit and stderr reporting.
func TestParseArgsBadFlagIsAnError(t *testing.T) {
	_, err := ParseArgs([]string{"-nosuchflag"})
	if err == nil {
		t.Fatal("expected an error for an undefined flag")
	}
	if errors.Is(err, ErrHelp) {
		t.Fatalf("an undefined flag must not be reported as a help request: %v", err)
	}
}

// -report packs the logs that already exist; asking for a document as well would make the flag
// unusable exactly when it is needed - after a run that failed.
func TestParseArgsReportNeedsNoInputFile(t *testing.T) {
	cfg, err := ParseArgs([]string{"-report"})
	if err != nil {
		t.Fatalf("ParseArgs(-report): %v", err)
	}
	if !cfg.Report {
		t.Error("expected Report=true")
	}
	if cfg.InputFile != "" {
		t.Errorf("InputFile = %q, want empty", cfg.InputFile)
	}
}

func TestParseArgsRegister(t *testing.T) {
	cfg, err := ParseArgs([]string{"-register"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.Register {
		t.Fatalf("expected Register=true")
	}
}

// No-arg invocation enters the first-run flow (non-destructive registration + an
// interactive opt-in prompt), not the become-default handler. It must NOT auto-register
// as the default anymore; Register stays false and FirstRun is set instead.
func TestParseArgsNoArgsFirstRun(t *testing.T) {
	cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.FirstRun {
		t.Fatalf("expected FirstRun=true for empty args")
	}
	if cfg.Register {
		t.Fatalf("no-arg first run must not auto-register as the default handler")
	}
}

func TestParseArgsUnregister(t *testing.T) {
	cfg, err := ParseArgs([]string{"-unregister"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.Unregister {
		t.Fatalf("expected Unregister=true")
	}
}

func TestParseArgsInputFileRequired(t *testing.T) {
	_, err := ParseArgs([]string{"-src", "en"})
	if err == nil {
		t.Fatalf("expected error when input file is missing")
	}
}

func TestParseArgsWithInputFile(t *testing.T) {
	cfg, err := ParseArgs([]string{"book.epub", "-src", "en", "-dst", "ru"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.InputFile != "book.epub" {
		t.Fatalf("unexpected input file: %s", cfg.InputFile)
	}
}

// Flags written after the document used to be dropped without a word: output landed beside the
// source instead of the named folder, switches did nothing, and an explicit -max-cost - the one
// guard a user types on purpose - disappeared. Both orders must now mean the same thing.
func TestParseArgsFlagsAfterInputFile(t *testing.T) {
	cfg, err := ParseArgs([]string{"book.epub", "-folder", "out", "-ocr", "-notranslate", "-force", "-max-cost", "5"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.InputFile != "book.epub" {
		t.Errorf("InputFile = %q, want book.epub", cfg.InputFile)
	}
	if cfg.OutputFolder != "out" {
		t.Errorf("OutputFolder = %q, want out - the flag was dropped", cfg.OutputFolder)
	}
	if cfg.MaxCost != 5 {
		t.Errorf("MaxCost = %v, want 5 - a spend cap must never be silently ignored", cfg.MaxCost)
	}
	if !cfg.OCR || !cfg.NoTranslate || !cfg.Force {
		t.Errorf("switches after the path were dropped: ocr=%v notranslate=%v force=%v", cfg.OCR, cfg.NoTranslate, cfg.Force)
	}
}

// A value-taking flag must not swallow the document, and a boolean must not swallow its
// neighbour. Both are decided from the FlagSet's own arity, so this pins that wiring.
func TestParseArgsPermutationRespectsFlagArity(t *testing.T) {
	cases := [][]string{
		{"-split", "1234", "book.epub"},
		{"book.epub", "-split", "1234"},
		{"-ocr", "book.epub", "-split", "1234"},
		{"-split=1234", "book.epub", "-ocr"},
	}
	for _, args := range cases {
		cfg, err := ParseArgs(args)
		if err != nil {
			t.Fatalf("ParseArgs(%v): %v", args, err)
		}
		if cfg.InputFile != "book.epub" {
			t.Errorf("ParseArgs(%v): InputFile = %q, want book.epub", args, cfg.InputFile)
		}
		if cfg.SplitSize != 1234 {
			t.Errorf("ParseArgs(%v): SplitSize = %d, want 1234", args, cfg.SplitSize)
		}
	}
}

// "--" ends flag parsing, which is how a document whose name starts with a dash is passed.
func TestParseArgsDoubleDashEndsFlags(t *testing.T) {
	cfg, err := ParseArgs([]string{"-ocr", "--", "-weird-name.epub"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.InputFile != "-weird-name.epub" {
		t.Errorf("InputFile = %q, want -weird-name.epub", cfg.InputFile)
	}
	if !cfg.OCR {
		t.Error("expected OCR=true")
	}
}

// Permuting must not hide a typo: an undefined flag after the document is still an error,
// not a silently ignored operand.
func TestParseArgsBadFlagAfterInputFileIsAnError(t *testing.T) {
	_, err := ParseArgs([]string{"book.epub", "-nosuchflag"})
	if err == nil {
		t.Fatal("expected an error for an undefined flag written after the input file")
	}
	if errors.Is(err, ErrHelp) {
		t.Fatalf("an undefined flag must not be reported as a help request: %v", err)
	}
}

// One document per run. The second operand is nearly always an unquoted path with a space in
// it, and dropping it silently made a quoting mistake look like a missing file.
func TestParseArgsExtraOperandIsRefused(t *testing.T) {
	_, err := ParseArgs([]string{`C:\My`, `Book.epub`})
	if err == nil {
		t.Fatal("expected an error for a second operand")
	}
	if !strings.Contains(err.Error(), "Book.epub") {
		t.Errorf("the error must name the argument it refused, got: %v", err)
	}
}

func TestParseArgsForce(t *testing.T) {
	cfg, err := ParseArgs([]string{"-force", "book.epub"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.Force {
		t.Fatalf("expected Force=true")
	}
}

func TestParseArgsTOCDepthDefault(t *testing.T) {
	cfg, err := ParseArgs([]string{"book.epub"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.TOCDepth != 0 {
		t.Fatalf("expected default TOCDepth=0 (unlimited), got %d", cfg.TOCDepth)
	}
}

func TestParseArgsTOCDepth(t *testing.T) {
	cfg, err := ParseArgs([]string{"-toc-depth", "2", "book.epub"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.TOCDepth != 2 {
		t.Fatalf("expected TOCDepth=2, got %d", cfg.TOCDepth)
	}
}
