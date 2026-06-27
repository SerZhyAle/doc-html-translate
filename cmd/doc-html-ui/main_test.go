package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"doc-html-translate/internal/translator"
)

func TestAssembleArgsPassesExplicitSplitZero(t *testing.T) {
	args := assembleArgs(runRequest{
		Input:     `C:\books\story.epub`,
		SplitSize: "0",
		SrcLang:   "en",
		DstLang:   "ru",
	})

	if !slices.Contains(args, "-split") {
		t.Fatalf("expected -split to be passed when GUI explicitly sends 0, got %v", args)
	}

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-split" && args[i+1] == "0" {
			return
		}
	}
	t.Fatalf("expected -split 0 in args, got %v", args)
}

func assertFlagValue(t *testing.T, args []string, flag, val string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == val {
			return
		}
	}
	t.Fatalf("expected %s %s in args, got %v", flag, val, args)
}

func TestAssembleArgsForwardsTOCDepthAndMaxCost(t *testing.T) {
	args := assembleArgs(runRequest{
		Input:    `C:\books\story.epub`,
		Google:   true,
		TOCDepth: "1",
		MaxCost:  "2",
		SrcLang:  "en",
		DstLang:  "ru",
	})
	assertFlagValue(t, args, "-toc-depth", "1")
	assertFlagValue(t, args, "-max-cost", "2")
}

func TestAssembleArgsOmitsDefaultTOCDepthAndMaxCost(t *testing.T) {
	// 0 is the CLI default for both (unlimited TOC / no cost limit), so the GUI
	// should not clutter the command line with them.
	args := assembleArgs(runRequest{
		Input:    `C:\books\story.epub`,
		TOCDepth: "0",
		MaxCost:  "0",
		SrcLang:  "en",
		DstLang:  "ru",
	})
	if slices.Contains(args, "-toc-depth") {
		t.Fatalf("did not expect -toc-depth for default 0, got %v", args)
	}
	if slices.Contains(args, "-max-cost") {
		t.Fatalf("did not expect -max-cost for default 0, got %v", args)
	}
}

func TestSaveGoogleAPIKeyRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)

	want := filepath.Join(tmp, "doc-html-translate", "google_api.key")
	if got := writableGoogleKeyPath(); got != want {
		t.Fatalf("writableGoogleKeyPath() = %q, want %q", got, want)
	}

	if err := saveGoogleAPIKey("  AIzaSyTEST123  "); err != nil {
		t.Fatalf("saveGoogleAPIKey: %v", err)
	}

	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read saved key: %v", err)
	}
	if string(data) != "AIzaSyTEST123" {
		t.Fatalf("saved key = %q, want trimmed %q", string(data), "AIzaSyTEST123")
	}

	key, err := translator.LoadGoogleAPIKey()
	if err != nil {
		t.Fatalf("LoadGoogleAPIKey after save: %v", err)
	}
	if key != "AIzaSyTEST123" {
		t.Fatalf("LoadGoogleAPIKey = %q, want %q", key, "AIzaSyTEST123")
	}
}

func TestSaveGoogleAPIKeyRejectsEmpty(t *testing.T) {
	if err := saveGoogleAPIKey("   "); err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}
