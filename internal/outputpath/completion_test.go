package outputpath

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"doc-html-translate/internal/config"
)

func completedOutput(t *testing.T, c Completion) (dir, src string) {
	t.Helper()
	root := t.TempDir()
	src = filepath.Join(root, "book.epub")
	write(t, src, "book")
	dir = filepath.Join(root, "book")
	mkdir(t, dir)
	if err := WriteMarker(dir, src); err != nil {
		t.Fatal(err)
	}
	if err := MarkComplete(dir, src, c); err != nil {
		t.Fatal(err)
	}
	return dir, src
}

func plain() Options {
	return OptionsFor(config.Config{NoTranslate: true, SinglePage: true, SourceLang: "en", TargetLang: "ru", SplitSize: 5000})
}

func TestCheckReuseMatchingOutput(t *testing.T) {
	dir, src := completedOutput(t, Completion{ToolVersion: "old", Options: plain(), Translation: TranslationNone})
	if r, _ := CheckReuse(dir, src, plain()); r != ReuseOK {
		t.Fatalf("reason = %v, want reuse (a tool upgrade alone is no reason to rebuild)", r)
	}
}

// The marker is written when a run claims the folder; only the completion makes it reusable.
func TestCheckReuseRejectsIncompleteAndLegacy(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "book.epub")
	write(t, src, "book")
	dir := filepath.Join(root, "book")
	mkdir(t, dir)
	if r, _ := CheckReuse(dir, src, plain()); r != ReuseNoRecord {
		t.Fatalf("legacy output without a record: %v", r)
	}
	if err := WriteMarker(dir, src); err != nil {
		t.Fatal(err)
	}
	if r, _ := CheckReuse(dir, src, plain()); r != ReuseNoRecord {
		t.Fatalf("interrupted output: %v", r)
	}
}

func TestCheckReuseSourceChanged(t *testing.T) {
	dir, src := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(src, later, later); err != nil {
		t.Fatal(err)
	}
	if r, _ := CheckReuse(dir, src, plain()); r != ReuseSourceChanged {
		t.Fatalf("reason = %v", r)
	}
}

// Converted without translation, then asked for an Ollama translation into German: the old
// untranslated output must not be opened as if it answered the new request.
func TestCheckReuseOptionsChanged(t *testing.T) {
	dir, src := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	want := OptionsFor(config.Config{UseOllama: true, OllamaModel: "gemma3:12b", SinglePage: true, SourceLang: "en", TargetLang: "de", SplitSize: 5000})
	r, diff := CheckReuse(dir, src, want)
	if r != ReuseOptionsChanged {
		t.Fatalf("reason = %v", r)
	}
	if !reflect.DeepEqual(diff, []string{"-google/-ollama"}) {
		t.Fatalf("diff = %v", diff)
	}
}

// Settings that cannot change this output are not a reason to throw it away.
func TestCheckReuseIgnoresInertOptions(t *testing.T) {
	dir, src := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	inert := OptionsFor(config.Config{NoTranslate: true, SinglePage: true, SourceLang: "fr", TargetLang: "de", SplitSize: 3000, TOCDepth: 2, OllamaModel: "x"})
	if r, diff := CheckReuse(dir, src, inert); r != ReuseOK {
		t.Fatalf("reason = %v %v", r, diff)
	}
}

// An image or a comic is OCRed whatever -ocr says, so -ocr on or off is the same output.
func TestCheckReuseForcedOCR(t *testing.T) {
	opts := plain()
	dir, src := completedOutput(t, Completion{Options: opts, OCRForced: true, Translation: TranslationNone})
	withOCR := opts
	withOCR.OCR = true
	if r, diff := CheckReuse(dir, src, withOCR); r != ReuseOK {
		t.Fatalf("reason = %v %v", r, diff)
	}
	withOCR.OCRLang = "rus"
	if r, _ := CheckReuse(dir, src, withOCR); r != ReuseOptionsChanged {
		t.Fatalf("a different OCR language must rebuild: %v", r)
	}
}

func TestCheckReuseTranslationState(t *testing.T) {
	google := OptionsFor(config.Config{UseGoogle: true, SinglePage: true, SourceLang: "en", TargetLang: "de"})
	for _, tc := range []struct {
		state string
		want  Reason
	}{
		{TranslationFull, ReuseOK},
		{TranslationPartial, ReusePartial},
		{TranslationNone, ReuseUntranslated},
	} {
		dir, src := completedOutput(t, Completion{Options: google, Translation: tc.state})
		if r, _ := CheckReuse(dir, src, google); r != tc.want {
			t.Errorf("%s: reason = %v, want %v", tc.state, r, tc.want)
		}
	}
}

func TestMarkCompleteKeepsOwnership(t *testing.T) {
	dir, src := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	if st := Inspect(dir, src); st != StateOwned {
		t.Fatalf("state = %v", st)
	}
	m, err := ReadMarker(dir)
	if err != nil || m.Complete == nil || m.Complete.Completed.IsZero() {
		t.Fatalf("marker = %+v, err = %v", m, err)
	}
}
