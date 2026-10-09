package evidence

import (
	"encoding/json"
	"strings"
	"testing"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

func humanAnnotation(sceneID string, langs ...string) *truth.Annotation {
	a := &truth.Annotation{
		SceneID: sceneID, Origin: truth.OriginHuman,
		Review: truth.Review{AnnotatedBy: "alice", AnnotatedOn: "2026-10-09"},
	}
	for _, l := range langs {
		a.Groups = append(a.Groups, truth.Group{ID: "g" + l, Language: l})
	}
	return a
}

func saveAnnotation(t *testing.T, dir string, a *truth.Annotation) {
	t.Helper()
	if err := truth.Save(truth.FinalPath(dir, a.SceneID), a); err != nil {
		t.Fatal(err)
	}
}

func TestSceneLangFollowsTheDeclaredLanguage(t *testing.T) {
	dir := t.TempDir()
	saveAnnotation(t, dir, humanAnnotation("poster", "ru"))
	saveAnnotation(t, dir, humanAnnotation("bilingual", "ru", "en", "ru"))
	saveAnnotation(t, dir, humanAnnotation("undeclared"))
	seed := humanAnnotation("seeded", "ru")
	seed.Origin = truth.OriginOCRSeed
	saveAnnotation(t, dir, seed)

	for _, c := range []struct {
		name     string
		explicit string
		scene    corpus.Scene
		want     LangChoice
	}{
		{"Cyrillic truth reads as rus", "", corpus.Scene{ID: "poster"}, LangChoice{"rus", LangSourceTruth}},
		{"truth beats the corpus field", "", corpus.Scene{ID: "poster", Languages: []string{"en"}}, LangChoice{"rus", LangSourceTruth}},
		{"several languages keep first-appearance order", "", corpus.Scene{ID: "bilingual"}, LangChoice{"rus+eng", LangSourceTruth}},
		{"an explicit -lang wins over the truth", "eng", corpus.Scene{ID: "poster"}, LangChoice{"eng", LangSourceFlag}},
		{"an explicit multi-language value is kept verbatim", "rus+eng", corpus.Scene{ID: "poster"}, LangChoice{"rus+eng", LangSourceFlag}},
		{"no group language falls to the corpus field", "", corpus.Scene{ID: "undeclared", Languages: []string{"fr"}}, LangChoice{"fra", LangSourceCorpus}},
		{"no annotation falls to the corpus field", "", corpus.Scene{ID: "bare", Languages: []string{"uk", "pt-BR"}}, LangChoice{"ukr+por", LangSourceCorpus}},
		{"an OCR-seeded draft declares nothing", "", corpus.Scene{ID: "seeded"}, LangChoice{DefaultLang, LangSourceDefault}},
		{"nothing declared reads as the default", "", corpus.Scene{ID: "bare"}, LangChoice{DefaultLang, LangSourceDefault}},
		{"a language the catalogue cannot read reads as the default", "", corpus.Scene{ID: "bare", Languages: []string{"ar"}}, LangChoice{DefaultLang, LangSourceDefault}},
		{"the readable part of a mixed declaration is kept", "", corpus.Scene{ID: "bare", Languages: []string{"ar", "ru"}}, LangChoice{"rus", LangSourceCorpus}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := SceneLang(dir, c.explicit, &c.scene); got != c.want {
				t.Errorf("SceneLang = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestTessCodeHandlesRegionAndScriptSubtags(t *testing.T) {
	for in, want := range map[string]string{
		"en": "eng", "EN": "eng", " ru ": "rus", "pt-BR": "por", "pt_BR": "por", "zh": "chi_sim",
		"zh-CN": "chi_sim", "zh-Hans": "chi_sim", "zh-TW": "chi_tra", "zh-Hant": "chi_tra", "ja": "jpn",
	} {
		if got, ok := TessCode(in); !ok || got != want {
			t.Errorf("TessCode(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "ar", "hi", "xx", "nl"} {
		if got, ok := TessCode(in); ok {
			t.Errorf("TessCode(%q) = %q; the catalogue has no pack for it", in, got)
		}
	}
}

// The lab table is a subset of the app's own mapping, never a rival: every entry must give what
// internal/ocr derives from the same -src code, and every target must be a catalogue pack.
func TestLangTableAgreesWithTheApp(t *testing.T) {
	iso, region := LangTables()
	catalogue := map[string]bool{}
	for _, l := range ocr.Available {
		catalogue[l.Code] = true
	}
	for _, table := range []map[string]string{iso, region} {
		for declared, pack := range table {
			if !catalogue[pack] {
				t.Errorf("%s -> %s: not a catalogue pack", declared, pack)
			}
			if got := ocr.TessLang(declared); got != pack {
				t.Errorf("%s: the lab maps to %s, the app to %s", declared, pack, got)
			}
		}
	}
	for _, l := range ocr.Available {
		if code := ocr.ISOFor(l.Code); code != "" {
			if _, ok := iso[code]; !ok {
				t.Errorf("the app maps %s -> %s but the lab table does not", code, l.Code)
			}
		}
	}
}

func TestSceneRecordsLanguageAdditively(t *testing.T) {
	plain, err := json.Marshal(Scene{SceneID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"lang", "langSource", "unmeasured"} {
		if strings.Contains(string(plain), `"`+k+`"`) {
			t.Errorf("a scene without the field serializes %s: %s", k, plain)
		}
	}
	with, _ := json.Marshal(Scene{SceneID: "a", Lang: "rus", LangSource: LangSourceTruth, Unmeasured: "language data unavailable: rus"})
	for _, want := range []string{`"lang":"rus"`, `"langSource":"truth"`, `"unmeasured":"language data unavailable: rus"`} {
		if !strings.Contains(string(with), want) {
			t.Errorf("%s lacks %s", with, want)
		}
	}
}

// An unmeasured scene is one explicit issue, and a scene read with another language than declared
// is caught instead of trusted.
func TestIssuesNamesUnmeasuredAndWrongLanguage(t *testing.T) {
	d := &Declaration{
		Version: 1, Purpose: "selected-dev", Procedure: Procedure,
		SceneIDs: []string{"a", "b"}, StressCases: []string{"none"},
		Viewports: []Viewport{{Name: "v", Width: 1, Height: 1, DeviceScaleFactor: 1}},
		Scenes:    map[string]Input{"a": {Lang: "rus"}, "b": {Lang: "rus"}},
	}
	r := &Run{
		Viewports: d.Viewports,
		Scenes: []Scene{
			{SceneID: "a", Unmeasured: "language data unavailable: rus", Lang: "rus"},
			{SceneID: "b", Lang: "eng", ImageWidth: 1, ImageHeight: 1},
		},
	}
	issues := strings.Join(Issues(t.TempDir(), r, d), "\n")
	if !strings.Contains(issues, "a: unmeasured: language data unavailable: rus") {
		t.Errorf("no unmeasured issue for a:\n%s", issues)
	}
	if strings.Contains(issues, "a: missing observation") {
		t.Errorf("the unmeasured scene also reports a cascade of missing captures:\n%s", issues)
	}
	if !strings.Contains(issues, "b: read with language eng, declared rus") {
		t.Errorf("no wrong-language issue for b:\n%s", issues)
	}
}
