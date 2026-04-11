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

func TestParseArgsNoArgsImplicitRegister(t *testing.T) {
	cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.Register {
		t.Fatalf("expected Register=true for empty args")
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
