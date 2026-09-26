package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/outputpath"
)

// recordingEngine translates like stubEngine and keeps every text it was sent.
type recordingEngine struct {
	mu   sync.Mutex
	sent []string
}

func (e *recordingEngine) Translate(_ context.Context, texts []string, _, _ string) ([]string, error) {
	e.mu.Lock()
	e.sent = append(e.sent, texts...)
	e.mu.Unlock()
	out := make([]string, len(texts))
	for i, t := range texts {
		out[i] = "DE:" + t
	}
	return out, nil
}

// chromeWords are texts only the injected reader chrome carries: the paging and contents links,
// the font and theme choices, the source file name and the "N / M" page counter.
var chromeWords = []string{"Previous page", "Next page", "Table of contents", "Serif", "Sepia", "Night", "book.txt", " / 5"}

// T13: the reader chrome is the app's, not the book's - it is never sent to the engine, in the
// paged book or in the merged single page, and it stays as it was written.
func TestReaderChromeIsNotSentForTranslation(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(fmt.Sprintf("single=%v", single), func(t *testing.T) {
			in := fivePageBook(t)
			eng := &recordingEngine{}
			r := translatingRunner(in, eng)
			r.cfg.SinglePage = single
			if code, err := r.Run(); code != ExitOK || err != nil {
				t.Fatalf("code = %d, err = %v", code, err)
			}
			if len(eng.sent) == 0 {
				t.Fatal("nothing was sent: the book text is missing too")
			}
			for _, s := range eng.sent {
				for _, w := range chromeWords {
					if strings.Contains(s, w) {
						t.Errorf("reader chrome sent for translation: %q", s)
					}
				}
			}
			pages, _ := filepath.Glob(filepath.Join(outputOf(in), "*.html"))
			for _, p := range pages {
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				for _, w := range chromeWords {
					if strings.Contains(string(data), "DE:"+w) {
						t.Errorf("%s: the chrome came back translated (%q)", filepath.Base(p), w)
					}
				}
			}
		})
	}
}

// T13: the cost estimate counts what is sent, so a page's chrome costs nothing.
func TestChromeIsNotBilled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "p.html"), `<html><body><div class="dht-navbar" lang="en"><span class="nav-file">book.epub</span>`+
		`<a class="dht-nav-link">Next page</a><span class="nav-info">1 / 9</span></div><p>Hello</p></body></html>`)
	book := &epub.Book{Manifest: []epub.ManifestItem{{ID: "p", Href: "p.html", MediaType: "text/html"}}}
	pages := loadContentPages(book, dir)
	if len(pages) != 1 || pages[0].err != nil {
		t.Fatalf("pages = %+v", pages)
	}
	if got := billableChars(book, pages); got != len("Hello") {
		t.Fatalf("billable characters = %d, want %d (the chrome is not sent)", got, len("Hello"))
	}
}

// emptySlotEngine translates everything except one text on the second page, which it returns
// empty and unflagged - the shape of an Ollama reply that skipped a number.
type emptySlotEngine struct {
	mu    sync.Mutex
	calls int
}

func (e *emptySlotEngine) Translate(_ context.Context, texts []string, _, _ string) ([]string, error) {
	e.mu.Lock()
	e.calls++
	n := e.calls
	e.mu.Unlock()
	out := make([]string, len(texts))
	for i, t := range texts {
		out[i] = "DE:" + t
	}
	if n == 2 {
		out[0] = ""
	}
	return out, nil
}

// T15: an empty slot is a partial result. The book goes on - the engine is working - but the
// run is not reported complete, and the output is recorded partial so it is rebuilt.
func TestEmptySlotIsPartialNotComplete(t *testing.T) {
	in := fivePageBook(t)
	code, err := translatingRunner(in, &emptySlotEngine{}).Run()
	if code != ExitAPI || err == nil || !strings.Contains(err.Error(), "partially translated, 4 of 5 pages") {
		t.Fatalf("code = %d, err = %v", code, err)
	}
	if got, total := countTranslatedPages(t, outputOf(in)); got != 5 || total != 5 {
		t.Fatalf("translated pages = %d of %d; every page but one segment should be translated", got, total)
	}
	m, rerr := outputpath.ReadMarker(outputOf(in))
	if rerr != nil || m.Complete == nil || m.Complete.Translation != outputpath.TranslationPartial {
		t.Fatalf("record = %+v, err = %v", m.Complete, rerr)
	}
}

// T10: -google with no key is the requested engine being unavailable, the same exit code as
// an Ollama that is not running. The book is still produced, recorded untranslated.
func TestGoogleWithoutKeyExitsLikeOllamaDown(t *testing.T) {
	s := newPipelineSandbox(t)
	in := s.input("book.txt", "Some text.\n")
	for _, engine := range []string{"google", "ollama"} {
		t.Run(engine, func(t *testing.T) {
			cfg := s.config(in)
			cfg.NoTranslate = false
			cfg.Force = true
			cfg.UseGoogle, cfg.UseOllama = engine == "google", engine == "ollama"
			code, err := s.run(cfg)
			if code != ExitAPI || err == nil {
				t.Fatalf("code = %d, err = %v; want ExitAPI", code, err)
			}
			want := map[string]string{"google": outputpath.TranslationNone, "ollama": outputpath.TranslationPartial}[engine]
			if rec := s.record(in); rec == nil || rec.Translation != want {
				t.Fatalf("record = %+v; want the book produced and recorded %q", rec, want)
			}
		})
	}
}
