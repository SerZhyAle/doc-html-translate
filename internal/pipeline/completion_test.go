package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/translator"
)

// stubEngine translates by prefixing "DE:" and can be told to fail, or to cancel the run, on
// its n-th page request.
type stubEngine struct {
	mu       sync.Mutex
	calls    int
	failOn   int
	cancelOn int
	cancel   context.CancelFunc
}

func (s *stubEngine) Translate(ctx context.Context, texts []string, _, _ string) ([]string, error) {
	s.mu.Lock()
	s.calls++
	n := s.calls
	s.mu.Unlock()
	if n == s.cancelOn {
		s.cancel()
		return nil, ctx.Err()
	}
	if n == s.failOn {
		return nil, errors.New("simulated engine failure")
	}
	out := make([]string, len(texts))
	for i, t := range texts {
		out[i] = "DE:" + t
	}
	return out, nil
}

// fivePageBook writes a plain-text book that the txt extractor splits into five pages.
func fivePageBook(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	for i := 1; i <= 150; i++ {
		fmt.Fprintf(&sb, "Paragraph number %d of the book.\n\n", i)
	}
	in := filepath.Join(t.TempDir(), "book.txt")
	writeFile(t, in, sb.String())
	return in
}

func translatingRunner(in string, eng translator.Client) Runner {
	r := NewRunner(config.Config{
		InputFile: in, NoOpen: true, UseOllama: true, OllamaModel: "gemma3:12b",
		SourceLang: "en", TargetLang: "de", SinglePage: false,
	})
	r.engines.ollama = func(config.Config) translator.Client { return eng }
	return r
}

func outputOf(in string) string {
	return strings.TrimSuffix(in, filepath.Ext(in))
}

func countTranslatedPages(t *testing.T, dir string) (translated, total int) {
	t.Helper()
	pages, _ := filepath.Glob(filepath.Join(dir, "page_*.html"))
	for _, p := range pages {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		total++
		if strings.Contains(string(data), "DE:Paragraph") {
			translated++
		}
	}
	return translated, total
}

// Done criterion 3: an engine failure on page 3 exits with ExitAPI and says how much was
// translated, and the output is recorded as partial so it is never reopened as a finished book.
func TestTranslationFailureOnPageThreeIsPartial(t *testing.T) {
	in := fivePageBook(t)
	code, err := translatingRunner(in, &stubEngine{failOn: 3}).Run()
	if code != ExitAPI {
		t.Fatalf("code = %d, err = %v; want ExitAPI", code, err)
	}
	if err == nil || !strings.Contains(err.Error(), "partially translated, 2 of 5 pages") {
		t.Fatalf("err = %v", err)
	}
	if got, total := countTranslatedPages(t, outputOf(in)); got != 2 || total != 5 {
		t.Fatalf("translated pages = %d of %d", got, total)
	}
	m, rerr := outputpath.ReadMarker(outputOf(in))
	if rerr != nil || m.Complete == nil || m.Complete.Translation != outputpath.TranslationPartial {
		t.Fatalf("record = %+v, err = %v", m.Complete, rerr)
	}

	// The next run with the same options rebuilds and finishes the job.
	code, err = translatingRunner(in, &stubEngine{}).Run()
	if code != ExitOK || err != nil {
		t.Fatalf("rerun: code = %d, err = %v", code, err)
	}
	if got, total := countTranslatedPages(t, outputOf(in)); got != total {
		t.Fatalf("rerun left %d of %d pages untranslated", total-got, total)
	}
}

// Done criterion 1: a run interrupted during translation leaves no completion record, so the
// next run rebuilds it instead of opening the half-translated output. Single-page mode is the
// case that used to bite: its index.html exists before translation starts.
func TestInterruptedRunIsRebuilt(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(fmt.Sprintf("single=%v", single), func(t *testing.T) {
			in := fivePageBook(t)
			run := func(eng translator.Client) Runner {
				r := translatingRunner(in, eng)
				r.cfg.SinglePage = single
				return r
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			code, err := run(&stubEngine{cancelOn: 1, cancel: cancel}).RunContext(ctx)
			if code != ExitInterrupted || err == nil {
				t.Fatalf("code = %d, err = %v; want ExitInterrupted", code, err)
			}
			out := outputOf(in)
			if r, _ := outputpath.CheckReuse(out, in, outputpath.OptionsFor(run(nil).cfg)); r != outputpath.ReuseNoRecord {
				t.Fatalf("interrupted output is reusable: reason %v", r)
			}
			assertPagesParse(t, out)

			eng := &stubEngine{}
			if code, err := run(eng).Run(); code != ExitOK || err != nil {
				t.Fatalf("rerun: code = %d, err = %v", code, err)
			}
			if eng.calls == 0 {
				t.Fatal("rerun reopened the interrupted output instead of rebuilding it")
			}
			pages, _ := filepath.Glob(filepath.Join(out, "*.html"))
			for _, p := range pages {
				if data, _ := os.ReadFile(p); strings.Contains(string(data), ">Paragraph number") {
					t.Fatalf("%s still holds untranslated text after the rebuild", filepath.Base(p))
				}
			}
		})
	}
}

// Done criterion 2: converted without translation, then asked for a translation - the second
// run must translate rather than open the untranslated result.
func TestReuseAfterOptionChangeRebuilds(t *testing.T) {
	in := fivePageBook(t)
	plain := NewRunner(config.Config{InputFile: in, NoOpen: true, NoTranslate: true, SourceLang: "en", TargetLang: "de"})
	if code, err := plain.Run(); code != ExitOK || err != nil {
		t.Fatalf("plain run: %d %v", code, err)
	}
	index := filepath.Join(outputOf(in), "index.html")
	before, err := os.Stat(index)
	if err != nil {
		t.Fatal(err)
	}

	// Same options: reused as-is.
	if code, err := plain.Run(); code != ExitOK || err != nil {
		t.Fatalf("reuse run: %d %v", code, err)
	}
	if after, _ := os.Stat(index); !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("identical options rebuilt the output")
	}

	eng := &stubEngine{}
	if code, err := translatingRunner(in, eng).Run(); code != ExitOK || err != nil {
		t.Fatalf("translate run: %d %v", code, err)
	}
	if got, total := countTranslatedPages(t, outputOf(in)); got == 0 || got != total {
		t.Fatalf("translated %d of %d pages after the option change", got, total)
	}
}

// Ollama keeps the batches that did come back: a page whose second batch failed is written with
// its first batch translated and counted as not fully translated.
func TestPartialPageKeepsTranslatedSegments(t *testing.T) {
	in := fivePageBook(t)
	eng := &partialEngine{}
	code, err := translatingRunner(in, eng).Run()
	if code != ExitAPI || err == nil || !strings.Contains(err.Error(), "partially translated, 0 of 5 pages") {
		t.Fatalf("code = %d, err = %v", code, err)
	}
	data, rerr := os.ReadFile(filepath.Join(outputOf(in), "page_001.html"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !strings.Contains(string(data), "DE:Paragraph number 1 of") || !strings.Contains(string(data), ">Paragraph number 30 of") {
		t.Fatal("the translated half of the page was discarded, or the failed half was not left in the source language")
	}
}

type partialEngine struct{}

func (partialEngine) Translate(_ context.Context, texts []string, _, _ string) ([]string, error) {
	out := make([]string, len(texts))
	var missing []int
	for i, t := range texts {
		if i >= len(texts)/2 {
			missing = append(missing, i)
			continue
		}
		out[i] = "DE:" + t
	}
	return out, &translator.PartialError{Missing: missing, Err: errors.New("batch 2 failed")}
}

func assertPagesParse(t *testing.T, dir string) {
	t.Helper()
	pages, _ := filepath.Glob(filepath.Join(dir, "*.html"))
	for _, p := range pages {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "</html>") {
			t.Errorf("%s is truncated", filepath.Base(p))
		}
	}
	if tmp, _ := filepath.Glob(filepath.Join(dir, ".*.tmp")); len(tmp) > 0 {
		t.Errorf("temp files left behind: %v", tmp)
	}
}
