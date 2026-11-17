package outputpath

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"doc-html-translate/internal/config"
)

// finishedOutput is completedOutput with the entry page the converter always writes, which PreviousResult looks for.
func finishedOutput(t *testing.T, c Completion) (dir, src string) {
	t.Helper()
	dir, src = completedOutput(t, c)
	write(t, filepath.Join(dir, "index.html"), "<html></html>")
	return dir, src
}

func ollamaOptions(dst string) Options {
	return OptionsFor(config.Config{UseOllama: true, OllamaModel: "gemma3:12b", SinglePage: true, SourceLang: "en", TargetLang: dst, SplitSize: 5000})
}

// The window and the verb share one definition of a previous result: only an output the converter owns, with its entry
// page, for this very input. A saved website with an index.html is not one.
func TestPreviousResultOnlyForTheConvertersOwnOutput(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "book.epub")
	write(t, src, "book")

	if _, err := PreviousResult(src, ""); !errors.Is(err, ErrNoPreviousResult) {
		t.Fatalf("nothing there: %v", err)
	}

	// A folder of the user's own that happens to hold an index.html, under the name the output would take.
	foreign := filepath.Join(root, "book")
	write(t, filepath.Join(foreign, "index.html"), "my website")
	if got, err := PreviousResult(src, ""); !errors.Is(err, ErrNoPreviousResult) {
		t.Fatalf("a foreign folder was reported as a result: %q, %v", got, err)
	}
	if err := os.RemoveAll(foreign); err != nil {
		t.Fatal(err)
	}

	// An output the converter owns but that has no entry page is not a result to open.
	mkdir(t, foreign)
	if err := WriteMarker(foreign, src); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviousResult(src, ""); !errors.Is(err, ErrNoPreviousResult) {
		t.Fatalf("an output without index.html was reported: %v", err)
	}

	write(t, filepath.Join(foreign, "index.html"), "<html></html>")
	got, err := PreviousResult(src, "")
	if err != nil || got != foreign {
		t.Fatalf("PreviousResult = %q, %v; want %q", got, err, foreign)
	}

	// A relative input resolves the same way the pipeline resolves it: against the working directory.
	t.Chdir(root)
	if got, err := PreviousResult("book.epub", ""); err != nil || !samePath(got, foreign) {
		t.Fatalf("relative input: %q, %v", got, err)
	}
	// With -folder the output is looked for under that folder, not beside the input.
	if _, err := PreviousResult(src, filepath.Join(root, "elsewhere")); !errors.Is(err, ErrNoPreviousResult) {
		t.Fatalf("another folder: %v", err)
	}
}

// The window offers "delete and rebuild" only for a changed option: an unfinished output, a changed source and a
// translation that stopped part-way are rebuilt on their own and are not "changed".
func TestOptionsChangedIsOnlyTheOptions(t *testing.T) {
	dir, src := finishedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	same := config.Config{NoTranslate: true, SinglePage: true, SourceLang: "en", TargetLang: "ru", SplitSize: 5000, InputFile: src}
	if OptionsChanged(same, dir) {
		t.Error("the very options the output was built with count as changed")
	}
	withOCR := same
	withOCR.OCR = true
	if !OptionsChanged(withOCR, dir) {
		t.Error("-ocr on an output built without it is not a change")
	}
	// The execution-only switches are not result-affecting.
	quiet := same
	quiet.Verbose, quiet.Force, quiet.NoOpen, quiet.MaxCost = true, true, true, 2
	if OptionsChanged(quiet, dir) {
		t.Error("an execution-only switch counts as a change")
	}

	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(src, later, later); err != nil {
		t.Fatal(err)
	}
	if OptionsChanged(withOCR, dir) {
		t.Error("a changed source counted as a changed option")
	}

	bad := withOCR
	bad.InputFile = filepath.Join(t.TempDir(), "other.epub")
	if OptionsChanged(bad, dir) {
		t.Error("an output of another input counted as a changed option")
	}
}

func TestStatusForAFreshInput(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "book.epub")
	write(t, src, "book")
	st, err := StatusFor(src, "", plain())
	if err != nil || st != (Status{}) {
		t.Fatalf("StatusFor of a fresh input = %+v, %v; want exists:false and nothing else", st, err)
	}
}

func TestStatusForAFinishedCurrentResult(t *testing.T) {
	dir, src := finishedOutput(t, Completion{ToolVersion: "26.1114.0001", Options: plain(), Translation: TranslationNone})
	st, err := StatusFor(src, "", plain())
	if err != nil {
		t.Fatal(err)
	}
	want := Status{
		Exists: true, Complete: true, OutputDir: dir, EntryFile: filepath.Join(dir, "index.html"),
		Translation: TranslationNone, ToolVersion: "26.1114.0001",
	}
	if st != want {
		t.Fatalf("StatusFor = %+v, want %+v", st, want)
	}
}

// "stale after the same input is asked for with another target language" holds when an engine is on: without one the
// target language does not shape the output (normalized drops it), and the app already ships that rule.
func TestStatusForOptionsChangedNeedsAnEngineForTheTargetLanguage(t *testing.T) {
	_, src := finishedOutput(t, Completion{Options: ollamaOptions("ru"), Translation: TranslationFull})

	st, err := StatusFor(src, "", ollamaOptions("ru"))
	if err != nil || !st.Exists || !st.Complete || st.Stale || st.Reason != "" {
		t.Fatalf("same options: %+v, %v", st, err)
	}
	st, err = StatusFor(src, "", ollamaOptions("de"))
	if err != nil || !st.Exists || !st.Complete || !st.Stale || st.Reason != "options-changed" {
		t.Fatalf("another --dst with an engine: %+v, %v", st, err)
	}

	_, plainSrc := finishedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	other := plain()
	other.TargetLang = "de"
	if st, _ := StatusFor(plainSrc, "", other); st.Stale {
		t.Errorf("another --dst without an engine counts as stale: %+v", st)
	}
}

func TestStatusForTellsEveryReasonApart(t *testing.T) {
	t.Run("an unfinished run", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "book.epub")
		write(t, src, "book")
		dir := filepath.Join(root, "book")
		mkdir(t, dir)
		if err := WriteMarker(dir, src); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, "index.html"), "<html></html>")
		st, err := StatusFor(src, "", plain())
		if err != nil || !st.Exists || st.Complete || !st.Stale || st.Reason != "no-record" {
			t.Fatalf("%+v, %v", st, err)
		}
	})
	t.Run("a changed source", func(t *testing.T) {
		_, src := finishedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
		later := time.Now().Add(time.Hour)
		if err := os.Chtimes(src, later, later); err != nil {
			t.Fatal(err)
		}
		st, err := StatusFor(src, "", plain())
		if err != nil || !st.Exists || !st.Complete || !st.Stale || st.Reason != "source-changed" {
			t.Fatalf("%+v, %v", st, err)
		}
	})
	t.Run("a translation that stopped part-way", func(t *testing.T) {
		_, src := finishedOutput(t, Completion{Options: ollamaOptions("ru"), Translation: TranslationPartial})
		st, err := StatusFor(src, "", ollamaOptions("ru"))
		if err != nil || !st.Complete || !st.Stale || st.Reason != "partial" || st.Translation != TranslationPartial {
			t.Fatalf("%+v, %v", st, err)
		}
	})
	t.Run("a translation that never ran", func(t *testing.T) {
		_, src := finishedOutput(t, Completion{Options: ollamaOptions("ru"), Translation: TranslationNone})
		st, err := StatusFor(src, "", ollamaOptions("ru"))
		if err != nil || !st.Complete || !st.Stale || st.Reason != "untranslated" {
			t.Fatalf("%+v, %v", st, err)
		}
	})
	t.Run("a folder that is not the converter's", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "book.epub")
		write(t, src, "book")
		write(t, filepath.Join(root, "book", "index.html"), "a saved website")
		st, err := StatusFor(src, "", plain())
		if err != nil || st != (Status{}) {
			t.Fatalf("a foreign index.html was reported: %+v, %v", st, err)
		}
	})
}

func TestReasonTokensAreOneWordEach(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range []Reason{ReuseNoRecord, ReuseSourceChanged, ReuseOptionsChanged, ReusePartial, ReuseUntranslated} {
		tok := r.Token()
		if tok == "" || seen[tok] {
			t.Errorf("reason %d has token %q (empty or repeated)", r, tok)
		}
		seen[tok] = true
	}
	if ReuseOK.Token() != "" {
		t.Error("a current result has a reason token")
	}
}
