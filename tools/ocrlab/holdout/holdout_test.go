package holdout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

func scene(id string, split corpus.Split) corpus.Scene {
	return corpus.Scene{
		ID: id, File: "x/" + id + ".png", Licence: corpus.LicencePD, Split: split,
		LicenceVerifiedBy: "owner", LicenceVerifiedOn: "2026-10-15",
		SHA256: "sha-" + id, Width: 100, Height: 100, Categories: []corpus.Category{corpus.CatPoster},
	}
}

func reviewedAnn(id string) *truth.Annotation {
	return &truth.Annotation{
		SchemaVersion: truth.SchemaVersion, SceneID: id, Origin: truth.OriginHuman,
		ImageWidth: 100, ImageHeight: 100, Ambiguity: truth.AmbiguityClear,
		Review: truth.Review{AnnotatedBy: "annotator", AnnotatedOn: "2026-10-15", CheckedBy: "checker", CheckedOn: "2026-10-15"},
	}
}

func writeRun(t *testing.T, root, runID, purpose string, sceneIDs []string, scored bool, startedAt string) string {
	t.Helper()
	dir := filepath.Join(root, runID)
	decl := map[string]any{"version": 1, "purpose": purpose, "sceneIds": sceneIDs}
	data, _ := json.Marshal(decl)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "declaration.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	ev, _ := json.Marshal(map[string]any{"startedAt": startedAt})
	if err := os.WriteFile(filepath.Join(dir, "evidence.json"), ev, 0o644); err != nil {
		t.Fatal(err)
	}
	if scored {
		if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func annsFor(m *corpus.Manifest) map[string]*truth.Annotation {
	out := map[string]*truth.Annotation{}
	for i := range m.Scenes {
		out[m.Scenes[i].ID] = reviewedAnn(m.Scenes[i].ID)
	}
	return out
}

func hasRule(rs []Refusal, rule string) bool {
	for _, r := range rs {
		if r.Rule == rule {
			return true
		}
	}
	return false
}

func hasProblem(ps []Problem, rule string) bool {
	for _, p := range ps {
		if p.Rule == rule {
			return true
		}
	}
	return false
}

func TestBuildRefusesAnEmptyHoldout(t *testing.T) {
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("dev-only", corpus.SplitDev)}}
	if _, err := Build(m, annsFor(m), nil, "operator", "2026-10-15"); err == nil ||
		!strings.Contains(err.Error(), "no holdout scenes") {
		t.Fatalf("expected empty-holdout refusal, got %v", err)
	}
}

func TestBuildRefusesMissingOperatorAndBadDate(t *testing.T) {
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("h", corpus.SplitHoldout)}}
	if _, err := Build(m, annsFor(m), nil, "", "2026-10-15"); err == nil || !strings.Contains(err.Error(), "-by") {
		t.Fatalf("expected operator refusal, got %v", err)
	}
	if _, err := Build(m, annsFor(m), nil, "op", "2026-13-99"); err == nil {
		t.Fatal("expected date refusal")
	}
}

func TestBuildRefusesScenePreviouslyJudgedInASelection(t *testing.T) {
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("tuned", corpus.SplitHoldout)}}
	runs := []RunUse{{RunID: "r1", Purpose: "selected-dev", SceneIDs: []string{"tuned"}, Scored: true}}
	_, err := Build(m, annsFor(m), runs, "operator", "2026-10-15")
	if err == nil || !strings.Contains(err.Error(), "holdout-previously-judged") {
		t.Fatalf("expected previously-judged refusal, got %v", err)
	}
	runs[0].Purpose = "full-benchmark"
	if _, err := Build(m, annsFor(m), runs, "operator", "2026-10-15"); err == nil {
		t.Fatal("full-benchmark prior use must also refuse")
	}
}

func TestBuildRecordsExploratoryPriorUseAndFreezesCleanScene(t *testing.T) {
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("clean", corpus.SplitHoldout)}}
	runs := []RunUse{{RunID: "explore-1", Purpose: "exploratory", SceneIDs: []string{"clean"}, Scored: true, StartedAt: "2026-10-01T10:00:00Z"}}
	f, err := Build(m, annsFor(m), runs, "operator", "2026-10-15")
	if err != nil {
		t.Fatal(err)
	}
	got := f.Scenes["clean"]
	if got.SHA256 != "sha-clean" || len(got.PriorUse) != 1 || got.PriorUse[0].Run != "explore-1" || got.PriorUse[0].On != "2026-10-01" {
		t.Fatalf("prior exploratory use not recorded: %+v", got)
	}
}

func TestEligibilityRefusalRules(t *testing.T) {
	synthetic := scene("syn", corpus.SplitHoldout)
	synthetic.Licence = corpus.LicenceSynthetic
	unreviewed := scene("unreviewed", corpus.SplitHoldout)
	unreviewed.LicenceVerifiedBy = "commons-api"
	selfChecked := scene("self", corpus.SplitHoldout)
	crop := scene("crop", corpus.SplitHoldout)
	crop.DerivedFrom = "parent-dev"
	m := &corpus.Manifest{Scenes: []corpus.Scene{synthetic, unreviewed, selfChecked, crop, scene("parent-dev", corpus.SplitDev)}}
	anns := annsFor(m)
	anns["self"].Review.CheckedBy = "annotator" // the annotator checked their own work
	anns["unreviewed"].Review.CheckedBy = ""
	_, refusals := Eligibility(m, anns, nil)
	if !hasRule(refusals, "holdout-not-independent") {
		t.Error("synthetic holdout must be refused as not independent")
	}
	if !hasRule(refusals, "holdout-rights-pending") {
		t.Error("machine-stamped rights must be refused")
	}
	if !hasRule(refusals, "holdout-annotation-invalid") {
		t.Error("self-checked annotation must be refused for the holdout")
	}
	// A holdout annotation without an independent check is refused for exactly that reason.
	unreviewedNamed := false
	for _, r := range refusals {
		if r.SceneID == "unreviewed" && strings.Contains(r.Detail, "independent check") {
			unreviewedNamed = true
		}
	}
	if !unreviewedNamed {
		t.Errorf("unreviewed holdout annotation must be refused for the missing independent check, got %v", refusals)
	}
	// A crop of a dev scene cannot be holdout, and the pair crossing splits must be named.
	found := false
	for _, r := range refusals {
		if r.SceneID == "crop" && r.Rule == "holdout-family-conflict" {
			found = true
		}
	}
	if !found {
		t.Errorf("family conflict across splits must be reported against the holdout candidate, got %v", refusals)
	}
}

func TestScanRunsReadsHistory(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", "selected-dev", []string{"s1"}, true, "2026-10-01T00:00:00Z")
	writeRun(t, filepath.Join(root, "nested"), "b", "exploratory", []string{"s2"}, false, "")
	if _, err := ScanRuns(filepath.Join(root, "missing")); err == nil {
		t.Fatal("a missing history must be an error, never an empty one")
	}
	runs, err := ScanRuns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("want 2 runs, got %d", len(runs))
	}
	byID := map[string]RunUse{}
	for _, r := range runs {
		byID[r.RunID] = r
	}
	if byID["a"].Purpose != "selected-dev" || !byID["a"].Scored || byID["a"].RunDate() != "2026-10-01" {
		t.Fatalf("run a misread: %+v", byID["a"])
	}
	if byID["b"].Scored || byID["b"].RunDate() != "" {
		t.Fatalf("run b misread: %+v", byID["b"])
	}
}

func frozen(t *testing.T) *Freeze {
	t.Helper()
	return &Freeze{SchemaVersion: 1, FrozenOn: "2026-10-01", FrozenBy: "operator",
		Scenes: map[string]FrozenScene{"h1": {SHA256: "sha-h1"}}}
}

func TestAuditReportsDrift(t *testing.T) {
	f := frozen(t)
	// Clean state.
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	if ps := Audit(m, f, nil); len(ps) != 0 {
		t.Fatalf("clean audit must pass, got %v", ps)
	}
	// Unrecorded addition.
	m2 := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout), scene("h2", corpus.SplitHoldout)}}
	if ps := Audit(m2, f, nil); !hasProblem(ps, "holdout-addition-unrecorded") {
		t.Fatalf("addition must be flagged, got %v", ps)
	}
	// Unrecorded removal.
	m3 := &corpus.Manifest{}
	if ps := Audit(m3, f, nil); !hasProblem(ps, "holdout-removal-unrecorded") {
		t.Fatalf("removal must be flagged, got %v", ps)
	}
	// Changed media.
	m4 := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	m4.Scenes[0].SHA256 = "sha-other"
	if ps := Audit(m4, f, nil); !hasProblem(ps, "holdout-media-changed") {
		t.Fatalf("media change must be flagged, got %v", ps)
	}
}

func TestAuditReportsUseLedgerProblems(t *testing.T) {
	f := frozen(t)
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	// A scored full-benchmark run exists but is not recorded.
	runs := []RunUse{{RunID: "bench-1", Purpose: "full-benchmark", SceneIDs: []string{"h1"}, Scored: true, StartedAt: "2026-10-02T00:00:00Z"}}
	if ps := Audit(m, f, runs); !hasProblem(ps, "holdout-use-unrecorded") {
		t.Fatalf("unrecorded acceptance use must be flagged, got %v", ps)
	}
	// Recording it fixes that but leaves a second problem class to check separately.
	f.Use = append(f.Use, Use{Run: "bench-1", On: "2026-10-02", Purpose: "full-benchmark"})
	if ps := Audit(m, f, runs); len(ps) != 0 {
		t.Fatalf("recorded clean use must audit clean, got %v", ps)
	}
	// A use before the freeze, an unknown run, a duplicate and a wrong purpose are all caught.
	f.Use = append(f.Use,
		Use{Run: "ghost", On: "2026-10-02", Purpose: "full-benchmark"},
		Use{Run: "bench-1", On: "2026-10-03", Purpose: "full-benchmark"},
		Use{Run: "early", On: "2026-09-01", Purpose: "full-benchmark"},
		Use{Run: "tuning", On: "2026-10-02", Purpose: "selected-dev"})
	ps := Audit(m, f, runs)
	for _, rule := range []string{"holdout-use-unknown-run", "holdout-use-duplicate", "holdout-use-before-freeze", "holdout-use-purpose"} {
		if !hasProblem(ps, rule) {
			t.Errorf("expected %s in %v", rule, ps)
		}
	}
}

func TestAuditCatchesHoldoutUsedForTuningAfterFreeze(t *testing.T) {
	f := frozen(t)
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	post := []RunUse{{RunID: "dev-2", Purpose: "selected-dev", SceneIDs: []string{"h1"}, Scored: true, StartedAt: "2026-10-05T00:00:00Z"}}
	if ps := Audit(m, f, post); !hasProblem(ps, "holdout-tuned-after-freeze") {
		t.Fatalf("post-freeze dev selection over a holdout scene must be flagged, got %v", ps)
	}
	pre := []RunUse{{RunID: "dev-0", Purpose: "selected-dev", SceneIDs: []string{"h1"}, Scored: true, StartedAt: "2026-09-01T00:00:00Z"}}
	if ps := Audit(m, f, pre); hasProblem(ps, "holdout-tuned-after-freeze") {
		t.Fatalf("pre-freeze history belongs to the freeze-time check, not the audit, got %v", ps)
	}
}

func TestAuditReportsReplacementBookkeeping(t *testing.T) {
	f := frozen(t)
	f.Replacements = []Replacement{
		{SceneID: "h1", On: "2026-10-02", By: "", Reason: "", Added: false},
		{SceneID: "never-added", On: "2026-10-02", By: "op", Reason: "r", Added: true},
	}
	m := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	ps := Audit(m, f, nil)
	for _, rule := range []string{"holdout-replacement-reason-missing", "holdout-replacement-operator-missing", "holdout-replacement-orphan"} {
		if !hasProblem(ps, rule) {
			t.Errorf("expected %s in %v", rule, ps)
		}
	}
}

func TestMembershipIssues(t *testing.T) {
	f := frozen(t)
	matching := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	if issues := MembershipIssues(f, matching); len(issues) != 0 {
		t.Fatalf("matching membership must be clean, got %v", issues)
	}
	extra := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout), scene("h9", corpus.SplitHoldout)}}
	if issues := MembershipIssues(f, extra); len(issues) != 1 || !strings.Contains(issues[0], "not in the frozen holdout") {
		t.Fatalf("unexpected issues: %v", issues)
	}
	drained := &corpus.Manifest{}
	if issues := MembershipIssues(f, drained); len(issues) != 1 || !strings.Contains(issues[0], "not in the run's") {
		t.Fatalf("unexpected issues: %v", issues)
	}
	changed := &corpus.Manifest{Scenes: []corpus.Scene{scene("h1", corpus.SplitHoldout)}}
	changed.Scenes[0].SHA256 = "sha-other"
	if issues := MembershipIssues(f, changed); len(issues) != 1 || !strings.Contains(issues[0], "media hash") {
		t.Fatalf("unexpected issues: %v", issues)
	}
}

func TestSaveLoadRoundTripAndMissingRecord(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("a missing freeze record must be an error, never an empty freeze")
	}
	path := filepath.Join(t.TempDir(), "holdout.json")
	f := frozen(t)
	f.Use = []Use{{Run: "r", On: "2026-10-02", Purpose: "full-benchmark"}}
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.FrozenOn != f.FrozenOn || got.Scenes["h1"].SHA256 != "sha-h1" || len(got.Use) != 1 {
		t.Fatalf("round trip lost data: %+v", got)
	}
	// A record without operator or date is refused on load.
	if err := Save(path, &Freeze{Scenes: map[string]FrozenScene{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "label somebody moved") {
		t.Fatalf("operatorless record must be refused, got %v", err)
	}
}
