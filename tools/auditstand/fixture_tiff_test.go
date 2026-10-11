package main

import (
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/internal/img"
)

func TestGeneratedTIFFFramesAndBudgetRefusal(t *testing.T) {
	dir := t.TempDir()
	if err := genMultiFrameTIFF(dir); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.Mkdir(out, 0o700); err != nil {
		t.Fatal(err)
	}
	book, err := img.Extract(filepath.Join(dir, "tiff-multiframe.tif"), out)
	if err != nil {
		t.Fatal(err)
	}
	if len(book.Spine) != 3 {
		t.Fatalf("frames=%d, want 3", len(book.Spine))
	}
	if err := genOverCapTIFF(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := img.Extract(filepath.Join(dir, "tiff-over-cap.tif"), out); err == nil {
		t.Fatal("over-budget input accepted")
	}
}
