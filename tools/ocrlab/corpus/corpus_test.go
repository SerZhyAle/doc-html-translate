package corpus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scene builds a minimally valid scene, so each test can break exactly one thing.
func scene(id string, cats ...Category) Scene {
	if len(cats) == 0 {
		cats = []Category{CatDocument}
	}
	return Scene{
		ID:                id,
		File:              id + ".png",
		SourceURL:         "https://example.invalid/" + id,
		RetrievedOn:       "2026-08-11",
		Licence:           LicencePD,
		LicenceURL:        "https://example.invalid/licence",
		LicenceVerifiedBy: "tester",
		LicenceVerifiedOn: "2026-08-11",
		SHA256:            strings.Repeat("0", 64),
		Categories:        cats,
		Split:             SplitDev,
	}
}

// has reports whether the problem list contains a rule, optionally for a given scene.
func has(ps []Problem, r Rule, id string) bool {
	for _, p := range ps {
		if p.Rule == r && (id == "" || p.SceneID == id) {
			return true
		}
	}
	return false
}

func TestLicenceGate(t *testing.T) {
	for _, l := range allLicences {
		if !l.Valid() {
			t.Errorf("%s should be valid", l)
		}
	}
	for _, bad := range []Licence{"CC-BY-NC-4.0", "CC-BY-NoDerivatives-4.0", "unknown", "", "cc0"} {
		if bad.Valid() {
			t.Errorf("%q must not be a spellable licence", bad)
		}
	}
	if !LicenceCCBY30.NeedsAttribution() || !LicenceCCBY40.NeedsAttribution() {
		t.Error("CC BY requires attribution")
	}
	for _, l := range []Licence{LicencePD, LicencePDM, LicenceCC0, LicenceOwn, LicenceSynthetic} {
		if l.NeedsAttribution() {
			t.Errorf("%s must not require attribution", l)
		}
	}
	if LicenceOwn.NeedsDownload() || LicenceSynthetic.NeedsDownload() {
		t.Error("locally produced media is never downloaded")
	}
}

// The human gate is the rule most likely to be quietly skipped by tooling, so it gets its own
// test: an entry with everything else in order is still invalid without a named verifier.
func TestValidateRequiresHumanLicenceVerification(t *testing.T) {
	s := scene("a")
	s.LicenceVerifiedBy = ""
	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{s}}
	ps := Validate(m, "")
	if !has(ps, RuleNotVerified, "a") {
		t.Fatalf("expected %s, got %v", RuleNotVerified, ps)
	}
}

func TestValidatePerSceneRules(t *testing.T) {
	dup := scene("dup")
	ccby := scene("ccby")
	ccby.Licence = LicenceCCBY30
	noSource := scene("nosource")
	noSource.SourceURL = ""
	derived := scene("derived")
	derived.DerivedFrom = "nothing-like-this"
	badCat := scene("badcat", Category("mystery"))
	badSplit := scene("badsplit")
	badSplit.Split = Split("maybe")
	noCat := scene("nocat")
	noCat.Categories = nil

	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{
		dup, dup, ccby, noSource, derived, badCat, badSplit, noCat,
	}}
	ps := Validate(m, "")

	for _, want := range []struct {
		rule Rule
		id   string
	}{
		{RuleDuplicateID, "dup"},
		{RuleNoAttribution, "ccby"},
		{RuleNoSource, "nosource"},
		{RuleBadDerivedFrom, "derived"},
		{RuleBadCategory, "badcat"},
		{RuleBadSplit, "badsplit"},
		{RuleNoCategory, "nocat"},
	} {
		if !has(ps, want.rule, want.id) {
			t.Errorf("expected %s for %s; problems: %v", want.rule, want.id, ps)
		}
	}
}

// A SYNTHETIC scene has no URL to fetch, so demanding one would make every generated scene
// permanently invalid.
func TestValidateSyntheticNeedsNoSourceURL(t *testing.T) {
	s := scene("synth")
	s.Licence = LicenceSynthetic
	s.SourceURL = ""
	ps := Validate(&Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{s}}, "")
	if has(ps, RuleNoSource, "synth") {
		t.Fatalf("synthetic scene must not require a sourceUrl: %v", ps)
	}
}

func TestValidateHashAndMedia(t *testing.T) {
	root := t.TempDir()
	present := scene("present")
	if err := os.WriteFile(filepath.Join(root, "present.png"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	unmeasured := scene("unmeasured")
	unmeasured.SHA256 = ""
	if err := os.WriteFile(filepath.Join(root, "unmeasured.png"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	absent := scene("absent")

	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{present, unmeasured, absent}}
	ps := Validate(m, root)

	if !has(ps, RuleHashMismatch, "present") {
		t.Errorf("a wrong hash must be reported, not re-measured: %v", ps)
	}
	if !has(ps, RuleHashUnmeasured, "unmeasured") {
		t.Errorf("an empty hash must be visible: %v", ps)
	}
	if !has(ps, RuleMediaMissing, "absent") {
		t.Errorf("a missing file must be reported: %v", ps)
	}
}

// The real hash of a real file must validate clean, or every honest entry would be noise.
func TestValidateAcceptsMatchingHash(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ok.png")
	if err := os.WriteFile(path, []byte("some bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := scene("ok")
	s.File = "ok.png"
	s.SHA256 = sum
	ps := Validate(&Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{s}}, root)
	if has(ps, RuleHashMismatch, "ok") || has(ps, RuleMediaMissing, "ok") {
		t.Fatalf("a matching hash must not be reported: %v", ps)
	}
}

func TestValidateShapeMinimums(t *testing.T) {
	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{scene("a"), scene("b")}}
	ps := Validate(m, "")
	if !has(ps, RuleCorpusShort, "") {
		t.Error("a two-scene corpus must report the total minimum")
	}
	if !has(ps, RuleCategoryShort, "") {
		t.Error("unmet category minimums must be reported")
	}
	if !has(ps, RuleHoldoutShort, "") {
		t.Error("a corpus with no holdout must report it")
	}
}

// Holdout balance: 10 holdout scenes of which 6 are comics is 60%, over the 35% ceiling.
func TestValidateHoldoutSkew(t *testing.T) {
	var scenes []Scene
	for i := 0; i < 6; i++ {
		s := scene(string(rune('a'+i)), CatComic)
		s.Split = SplitHoldout
		scenes = append(scenes, s)
	}
	for i := 0; i < 4; i++ {
		s := scene(string(rune('m'+i)), CatDocument)
		s.Split = SplitHoldout
		scenes = append(scenes, s)
	}
	ps := Validate(&Manifest{SchemaVersion: SchemaVersion, Scenes: scenes}, "")
	if !has(ps, RuleHoldoutSkewed, "") {
		t.Fatalf("60%% comic holdout must be reported as skewed: %v", ps)
	}
}

func TestSaveIsStableAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.json")
	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{scene("z"), scene("a")}}
	if err := Save(path, m); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Scenes[0].ID != "a" {
		t.Errorf("scenes must be sorted by id, got %s first", loaded.Scenes[0].ID)
	}
	if err := Save(path, loaded); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("re-saving an unchanged manifest must be byte-identical")
	}
}

func TestLoadRejectsForeignSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":99,"scenes":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("a manifest from a future schema must not be read as if it were this one")
	}
}

func TestSelectReportsUnknownIDs(t *testing.T) {
	holdout := scene("h")
	holdout.Split = SplitHoldout
	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{scene("d"), holdout}}

	sel, missing := m.Select("holdout", nil)
	if len(sel) != 1 || sel[0].ID != "h" {
		t.Errorf("split filter returned %v", sel)
	}
	if len(missing) != 0 {
		t.Errorf("no ids requested, so nothing can be missing: %v", missing)
	}

	_, missing = m.Select("all", []string{"d", "typo"})
	if len(missing) != 1 || missing[0] != "typo" {
		t.Errorf("a typo'd scene id must be reported, not silently dropped: %v", missing)
	}
}

func TestFetchRefusesUnverifiedAndSkipsLocal(t *testing.T) {
	root := t.TempDir()
	unverified := scene("unverified")
	unverified.LicenceVerifiedBy = ""
	local := scene("local")
	local.Licence = LicenceSynthetic
	nohash := scene("nohash")
	nohash.SHA256 = ""

	m := &Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{unverified, local, nohash}}
	var sb strings.Builder
	results, err := Fetch(m, root, nil, &sb)
	if err == nil {
		t.Fatal("a refused scene must make the fetch fail")
	}
	byID := map[string]FetchResult{}
	for _, r := range results {
		byID[r.SceneID] = r
	}
	if byID["unverified"].Status != "fail" {
		t.Errorf("unverified licence must be refused, got %+v", byID["unverified"])
	}
	if byID["local"].Status != "skip" {
		t.Errorf("synthetic media must be skipped, got %+v", byID["local"])
	}
	if byID["nohash"].Status != "fail" {
		t.Errorf("an unverifiable download must be refused, got %+v", byID["nohash"])
	}
	if got := sb.String(); !strings.Contains(got, "unverified") {
		t.Errorf("fetch must report per scene, got %q", got)
	}
}

func TestFetchSkipsPresentAndCorrect(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ok.png")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := scene("ok")
	s.File = "ok.png"
	s.SHA256 = sum
	// An unreachable URL proves the skip happened before any network use.
	s.SourceURL = "https://127.0.0.1:1/never"

	var sb strings.Builder
	results, err := Fetch(&Manifest{SchemaVersion: SchemaVersion, Scenes: []Scene{s}}, root, nil, &sb)
	if err != nil {
		t.Fatalf("a present, correct file must not be re-fetched: %v", err)
	}
	if results[0].Status != "skip" {
		t.Errorf("expected skip, got %+v", results[0])
	}
}
