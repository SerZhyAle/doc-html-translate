package runner

import (
	"slices"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// "zzz" is no catalogue language, so no machine has its data: the test does not depend on which
// packs the developer happens to have installed.
const absentPack = "zzz"

func TestMissingLangDataNamesOnlyTheAbsentPacks(t *testing.T) {
	if got := missingLangData(absentPack); !slices.Equal(got, []string{absentPack}) {
		t.Errorf("missingLangData(%q) = %v", absentPack, got)
	}
	if got := missingLangData("zzy+zzz"); !slices.Equal(got, []string{"zzy", "zzz"}) {
		t.Errorf("both absent packs are named, got %v", got)
	}
	if got := missingLangData(""); len(got) != 0 {
		t.Errorf("no language names no pack, got %v", got)
	}
}

// A scene whose language data is missing is unmeasured with the reason - not an error, not a scene
// with no plates - and the run never reaches the converter or the browser.
func TestRunSceneRecordsUnavailableLanguageDataAsUnmeasured(t *testing.T) {
	s := &corpus.Scene{ID: "poster", Width: 30, Height: 40}
	sc := runScene(nil, "", s, Options{OutDir: t.TempDir(), Lang: absentPack})
	if sc.Unmeasured != "language data unavailable: "+absentPack {
		t.Errorf("Unmeasured = %q", sc.Unmeasured)
	}
	if sc.Error != "" || len(sc.Plates) != 0 {
		t.Errorf("an unmeasured scene is neither a failure nor a scene with plates: %+v", sc)
	}
	if sc.Lang != absentPack || sc.LangSource != evidence.LangSourceFlag {
		t.Errorf("language record = %q from %q", sc.Lang, sc.LangSource)
	}
	if sc.ImageWidth != 30 || sc.ImageHeight != 40 {
		t.Errorf("image size = %dx%d, want the corpus size", sc.ImageWidth, sc.ImageHeight)
	}
}

// Cyrillic truth is read as rus, and the run says it chose per scene; an explicit -lang is the
// label and the packs verbatim.
func TestRunLangsLabelsTheRun(t *testing.T) {
	dir := t.TempDir()
	a := &truth.Annotation{
		SceneID: "poster", Origin: truth.OriginHuman, Review: truth.Review{AnnotatedBy: "alice"},
		Groups: []truth.Group{{ID: "g", Language: "ru"}},
	}
	if err := truth.Save(truth.FinalPath(dir, "poster"), a); err != nil {
		t.Fatal(err)
	}
	scenes := []*corpus.Scene{{ID: "poster"}}

	if got := evidence.SceneLang(dir, "", scenes[0]); got.Lang != "rus" {
		t.Fatalf("Cyrillic truth chose %q, want rus", got.Lang)
	}
	if label, _ := runLangs(scenes, Options{Annotations: dir}); label != evidence.LangPerScene {
		t.Errorf("label = %q, want %q", label, evidence.LangPerScene)
	}
	if label, packs := runLangs(scenes, Options{Annotations: dir, Lang: "eng"}); label != "eng" || packs != "eng" {
		t.Errorf("explicit -lang gave label %q, packs %q", label, packs)
	}
}
