package config

import "testing"

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
