package outputpath

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fsutil"
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

// editRecord rewrites dir's record as the generic JSON object it is on disk, the way another
// build of the converter, or an older one, would have left it. It goes through fsutil.WriteFile
// because the record is hidden on Windows and a plain truncating write to it is refused.
func editRecord(t *testing.T, dir string, edit func(map[string]any)) {
	t.Helper()
	p := filepath.Join(dir, MarkerName)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	edit(rec)
	out, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func recordSchema(t *testing.T, dir string) (float64, bool) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, MarkerName))
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	v, ok := rec["schemaVersion"].(float64)
	return v, ok
}

// OCR-INVOCATION 1.2 item C: the record names its schema, both while the run is claimed and
// once it is complete.
func TestRecordCarriesSchemaVersion(t *testing.T) {
	dir, _ := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	if v, ok := recordSchema(t, dir); !ok || v != MarkerSchema || MarkerSchema != 1 {
		t.Fatalf("schemaVersion = %v (present %v), want %d, and the contract's schema 1", v, ok, MarkerSchema)
	}
}

// A record made before the member existed has none: it reads as schema 1, so a finished output
// of an older build is still reused, and the next completion stamps it.
func TestRecordWithoutSchemaVersionReadsAsSchema1(t *testing.T) {
	dir, src := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	editRecord(t, dir, func(rec map[string]any) { delete(rec, "schemaVersion") })
	if _, ok := recordSchema(t, dir); ok {
		t.Fatal("the test record still carries schemaVersion")
	}
	if r, _ := CheckReuse(dir, src, plain()); r != ReuseOK {
		t.Fatalf("an older record without schemaVersion: reason = %v, want reuse", r)
	}
	if err := MarkComplete(dir, src, Completion{Options: plain(), Translation: TranslationNone}); err != nil {
		t.Fatal(err)
	}
	if v, ok := recordSchema(t, dir); !ok || v != MarkerSchema {
		t.Fatalf("after MarkComplete schemaVersion = %v (present %v), want %d", v, ok, MarkerSchema)
	}
}

// Item C3: a record of a schema above the one this build knows is not completed to it, but the
// folder is still the converter's own, so the run rebuilds it instead of dodging to a sibling.
func TestRecordOfNewerSchemaIsNotCompleted(t *testing.T) {
	dir, src := completedOutput(t, Completion{Options: plain(), Translation: TranslationNone})
	editRecord(t, dir, func(rec map[string]any) { rec["schemaVersion"] = MarkerSchema + 1 })
	if r, _ := CheckReuse(dir, src, plain()); r != ReuseNoRecord {
		t.Fatalf("a newer schema: reason = %v, want not completed", r)
	}
	if st := Inspect(dir, src); st != StateOwned {
		t.Fatalf("a newer schema: state = %v, want owned", st)
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
