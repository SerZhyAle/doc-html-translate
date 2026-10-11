package main

import (
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/internal/outputpath"
)

// -folder names the parent: the book and its record live one level down. A scene that looked for
// the record at the folder itself reported a state defect that was the stand's own.
func TestBookDirUnderFindsTheBookBelowTheFolder(t *testing.T) {
	out := t.TempDir()
	if got := bookDirUnder(out); got != "" {
		t.Fatalf("empty folder: %q", got)
	}
	book := filepath.Join(out, "epub-small")
	if err := os.MkdirAll(book, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := bookDirUnder(out); got != "" {
		t.Fatalf("a folder without a record is not a book: %q", got)
	}
	src := filepath.Join(out, "epub-small.epub")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := outputpath.MarkComplete(book, src, outputpath.Completion{Translation: "none"}); err != nil {
		t.Fatal(err)
	}
	if got := bookDirUnder(out); got != book {
		t.Fatalf("bookDirUnder = %q, want %q", got, book)
	}
	translation, complete, err := completionOf(out)
	if err != nil || !complete || translation != "none" {
		t.Fatalf("completionOf = %q, %v, %v", translation, complete, err)
	}
}

func TestCompletionOfReportsAnIncompleteRecord(t *testing.T) {
	out := t.TempDir()
	book := filepath.Join(out, "b")
	if err := os.MkdirAll(book, 0o755); err != nil {
		t.Fatal(err)
	}
	// An ownership record without its "complete" section: the run claimed the folder and never
	// finished.
	if err := os.WriteFile(filepath.Join(book, outputpath.MarkerName), []byte(`{"tool":"doc-html-translate","source":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, complete, err := completionOf(out)
	if err != nil || complete {
		t.Fatalf("complete=%v err=%v, want an incomplete record", complete, err)
	}
}
