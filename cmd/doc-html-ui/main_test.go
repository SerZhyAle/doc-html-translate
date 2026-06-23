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
