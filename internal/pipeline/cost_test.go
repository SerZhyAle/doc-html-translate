package pipeline

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/translator"
)

// cyrillicBook writes a one-page book of n Cyrillic letters: 2n bytes, n billable characters.
func cyrillicBook(t *testing.T, n int) string {
	t.Helper()
	in := filepath.Join(t.TempDir(), "k.txt")
	writeFile(t, in, strings.Repeat("я", n)+"\n")
	return in
}

type googleRun struct {
	runner   Runner
	engine   *stubEngine
	confirms int
}

func newGoogleRun(in string, maxCost float64, answer bool) *googleRun {
	g := &googleRun{engine: &stubEngine{}}
	g.runner = NewRunner(config.Config{
		InputFile: in, NoOpen: true, UseGoogle: true, SinglePage: true,
		SourceLang: "ru", TargetLang: "de", MaxCost: maxCost,
	})
	g.runner.engines.googleKey = func() (string, error) { return "key", nil }
	g.runner.engines.google = func(string) translator.Client { return g.engine }
	g.runner.engines.confirm = func(string, string) bool { g.confirms++; return answer }
	return g
}

// Done criterion 4: -max-cost 0.01 refuses a Cyrillic book only when its character cost is
// over the limit. Counted in bytes, 300 letters looked like $0.012; they cost $0.006.
func TestMaxCostCountsCharactersNotBytes(t *testing.T) {
	small := newGoogleRun(cyrillicBook(t, 300), 0.01, false)
	if code, err := small.runner.Run(); code != ExitOK || err != nil {
		t.Fatalf("small: %d %v", code, err)
	}
	if small.engine.calls == 0 {
		t.Fatal("a book within the limit was not translated")
	}
	if small.confirms != 0 {
		t.Fatal("a set -max-cost is a pre-approval, but the dialog was shown")
	}

	// 800 letters are $0.016: over the limit. Below the old 1000-byte threshold's reach too,
	// which is exactly where the guard used to be skipped.
	big := newGoogleRun(cyrillicBook(t, 800), 0.01, true)
	if code, err := big.runner.Run(); code != ExitOK || err != nil {
		t.Fatalf("big: %d %v", code, err)
	}
	if big.engine.calls != 0 {
		t.Fatal("a book over -max-cost was sent to the engine")
	}
}

// Without a limit the confirmation stays as it was: asked above the threshold, and a "no"
// leaves the book untranslated.
func TestNoLimitStillAsks(t *testing.T) {
	g := newGoogleRun(cyrillicBook(t, 1500), 0, false)
	if code, err := g.runner.Run(); code != ExitOK || err != nil {
		t.Fatalf("%d %v", code, err)
	}
	if g.confirms != 1 || g.engine.calls != 0 {
		t.Fatalf("confirms = %d, engine calls = %d", g.confirms, g.engine.calls)
	}
}

// The estimate covers everything that is sent: the pages, the title and every TOC label.
func TestBillableCharsIncludeTitleAndTOC(t *testing.T) {
	book := &epub.Book{
		Title: "Война",
		TOC: []epub.TOCEntry{
			{Title: "Глава", Children: []epub.TOCEntry{{Title: "Часть"}}},
		},
	}
	pages := []contentPage{{charCount: 10}, {charCount: 7}, {charCount: 99, err: errSkip}}
	if got := billableChars(book, pages); got != 10+7+5+5+5 {
		t.Fatalf("billableChars = %d", got)
	}
}

var errSkip = errors.New("unreadable page")
