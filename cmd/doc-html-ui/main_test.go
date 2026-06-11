package main

import (
	"slices"
	"testing"
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
