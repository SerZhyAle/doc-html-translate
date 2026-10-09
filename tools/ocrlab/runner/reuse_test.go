package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
)

var (
	testEngine  = evidence.Engine{Tesseract: "tesseract v5.4.0", TessdataVersion: "eng=sha256:aa", Lang: evidence.LangPerScene}
	testBrowser = evidence.Browser{Name: "edge", Version: "edge 154.0.1"}
)

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeCollect stands in for runScene: it writes the files a collection would and returns a complete
// scene, so what is under test is the reuse decision and the copy, not the browser.
func fakeCollect(t *testing.T, dir, id string, media []byte, fail bool) evidence.Scene {
	t.Helper()
	sc := evidence.Scene{SceneID: id, ImageWidth: 20, ImageHeight: 10, Lang: "eng", LangSource: "default", OcrMs: 1200,
		Screenshots: evidence.Screenshots{Source: "shots/" + id + "/source.png", Stress: map[string]string{}}}
	if fail {
		sc.Error = "convert: boom"
		return sc
	}
	writeTestFile(t, filepath.Join(dir, "pages", id, "page_001.html"), []byte("<html></html>"))
	writeTestFile(t, filepath.Join(dir, "shots", id, "source.png"), media)
	for _, v := range Viewports {
		for _, c := range StressNames() {
			o := evidence.Observation{Viewport: v.Name, StressCase: c,
				Rendered:  "shots/" + id + "/" + v.Name + "-" + c + ".png",
				Concealed: "shots/" + id + "/" + v.Name + "-" + c + "-hidden.png"}
			writeTestFile(t, filepath.Join(dir, o.Rendered), makeGreyPNG(t, 20, 10))
			writeTestFile(t, filepath.Join(dir, o.Concealed), makeGreyPNG(t, 20, 10))
			sc.Observations = append(sc.Observations, o)
			sc.Plates = append(sc.Plates, evidence.Plate{Text: "t", Viewport: v.Name, StressCase: c})
			if v.Name == Viewports[0].Name {
				sc.Screenshots.Stress[c] = o.Rendered
				if c == PrimaryStress {
					sc.Screenshots.Rendered = o.Rendered
				}
			}
		}
	}
	return sc
}

type fakeRunOptions struct {
	producer string
	fresh    bool
	from     string
	failing  string
}

// fakeRun performs one whole run of the scenes with a fake collector, finishing the way a real run
// does (evidence, execution record, file manifest), and reports which scenes were collected.
func fakeRun(t *testing.T, root, name string, ids []string, o fakeRunOptions) (collected []string, run *evidence.Run, log string) {
	t.Helper()
	media := makeGreyPNG(t, 20, 10)
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d := &evidence.Declaration{
		Version: 1, Purpose: "exploratory", Procedure: evidence.Procedure, SourceDigest: "src", ProducerDigest: o.producer,
		Environment: "windows/amd64; go1.26.1; CPUs=8; GOMAXPROCS=8", Viewports: Viewports, StressCases: StressNames(),
		Scenes: map[string]evidence.Input{},
	}
	for _, id := range ids {
		d.SceneIDs = append(d.SceneIDs, id)
		d.Scenes[id] = evidence.Input{MediaSHA256: evidence.Digest(media), SceneSHA256: "scene-" + id, Lang: "eng"}
	}
	data, _ := json.Marshal(d)
	writeTestFile(t, filepath.Join(dir, "declaration.json"), data)

	var buf bytes.Buffer
	opt := Options{OutDir: dir, Annotations: filepath.Join(root, "no-annotations"), Log: &buf, Fresh: o.fresh, ReuseFrom: o.from}
	if opt.ReuseFrom == "" {
		opt.ReuseFrom = root
	}
	reuse, err := newReuser(opt, d, testEngine, testBrowser)
	if err != nil {
		t.Fatal(err)
	}
	run = &evidence.Run{RunID: name, StartedAt: "2026-10-09T12:00:00Z", Edition: evidence.EditionDesktop,
		Engine: testEngine, Browser: testBrowser, Viewports: Viewports}
	for _, id := range ids {
		sc, _ := reuse.obtain(&corpus.Scene{ID: id}, opt, func() evidence.Scene {
			collected = append(collected, id)
			return fakeCollect(t, dir, id, media, id == o.failing)
		})
		run.Scenes = append(run.Scenes, sc)
	}
	if err := run.Save(filepath.Join(dir, EvidenceFile)); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "execution.json"), []byte(`{"sourceDigest":"src","unchanged":true}`))
	if err := evidence.WriteFilesManifest(dir, d.SceneIDs); err != nil {
		t.Fatal(err)
	}
	return collected, run, buf.String()
}

func withScorer(t *testing.T) {
	t.Helper()
	old := CurrentScorerDigest
	CurrentScorerDigest = func() (string, error) { return "scorer-1", nil }
	t.Cleanup(func() { CurrentScorerDigest = old })
}

func TestSecondRunOverTheSameScenesCollectsNothing(t *testing.T) {
	withScorer(t)
	root, ids := t.TempDir(), []string{"a", "b", "c"}
	first, _, _ := fakeRun(t, root, "run-1", ids, fakeRunOptions{producer: "prod-1"})
	if !slices.Equal(first, ids) {
		t.Fatalf("nothing earlier exists, so every scene is collected: %v", first)
	}
	second, run, log := fakeRun(t, root, "run-2", ids, fakeRunOptions{producer: "prod-1"})
	if len(second) != 0 {
		t.Fatalf("unchanged inputs must collect nothing, collected %v\n%s", second, log)
	}
	for _, sc := range run.Scenes {
		if sc.ReusedFrom == nil || !strings.HasSuffix(sc.ReusedFrom.Bundle, "run-1") {
			t.Fatalf("%s: a reused scene must name its source, got %+v", sc.SceneID, sc.ReusedFrom)
		}
		if _, err := os.Stat(filepath.Join(root, "run-2", "shots", sc.SceneID, "phone-cjk-hidden.png")); err != nil {
			t.Fatalf("%s: the captures were not copied into the new run", sc.SceneID)
		}
	}
	if !strings.Contains(log, "reused a from ") || !strings.Contains(log, "run-1") {
		t.Fatalf("the log must say what was reused and from where:\n%s", log)
	}
}

func TestProductEditCollectsEverythingAgain(t *testing.T) {
	withScorer(t)
	root, ids := t.TempDir(), []string{"a", "b"}
	fakeRun(t, root, "run-1", ids, fakeRunOptions{producer: "prod-1"})
	collected, run, log := fakeRun(t, root, "run-2", ids, fakeRunOptions{producer: "prod-2"})
	if !slices.Equal(collected, ids) {
		t.Fatalf("a changed producer digest must collect every scene, collected %v", collected)
	}
	for _, sc := range run.Scenes {
		if sc.ReusedFrom != nil {
			t.Fatalf("%s: a collected scene claims to be reused", sc.SceneID)
		}
	}
	if !strings.Contains(log, "producer digest differs") {
		t.Fatalf("the reason must be logged:\n%s", log)
	}
}

func TestFreshCollectsEverythingWithoutLookingBack(t *testing.T) {
	withScorer(t)
	root, ids := t.TempDir(), []string{"a", "b"}
	fakeRun(t, root, "run-1", ids, fakeRunOptions{producer: "prod-1"})
	collected, _, log := fakeRun(t, root, "run-2", ids, fakeRunOptions{producer: "prod-1", fresh: true})
	if !slices.Equal(collected, ids) || strings.Contains(log, "reused") {
		t.Fatalf("-fresh must collect every scene, collected %v\n%s", collected, log)
	}
}

func TestOnlyTheFailedSceneIsCollectedAgain(t *testing.T) {
	withScorer(t)
	root, ids := t.TempDir(), []string{"a", "b", "c"}
	_, first, _ := fakeRun(t, root, "run-1", ids, fakeRunOptions{producer: "prod-1", failing: "b"})
	if first.Find("b").Error == "" {
		t.Fatal("fixture: scene b should have failed")
	}
	collected, run, _ := fakeRun(t, root, "run-2", ids, fakeRunOptions{producer: "prod-1"})
	if !slices.Equal(collected, []string{"b"}) {
		t.Fatalf("a failed scene gets a fresh attempt and the rest are reused, collected %v", collected)
	}
	if run.Find("b").Error != "" || run.Find("b").ReusedFrom != nil || run.Find("a").ReusedFrom == nil {
		t.Fatalf("b should be fresh and a reused: %+v / %+v", run.Find("b"), run.Find("a"))
	}
}

func TestReuseFromLimitsTheSearchToOneBundle(t *testing.T) {
	withScorer(t)
	root, ids := t.TempDir(), []string{"a"}
	fakeRun(t, root, "run-1", ids, fakeRunOptions{producer: "prod-1"})
	fakeRun(t, root, "run-2", ids, fakeRunOptions{producer: "prod-2"})
	collected, _, log := fakeRun(t, root, "run-3", ids, fakeRunOptions{producer: "prod-2", from: filepath.Join(root, "run-1")})
	if !slices.Equal(collected, ids) {
		t.Fatalf("run-2 would match, but the search was limited to run-1 (producer differs): %v\n%s", collected, log)
	}
	collected, run, _ := fakeRun(t, root, "run-4", ids, fakeRunOptions{producer: "prod-2", from: filepath.Join(root, "run-2")})
	if len(collected) != 0 || !strings.HasSuffix(run.Find("a").ReusedFrom.Bundle, "run-2") {
		t.Fatalf("limited to run-2 it must reuse from run-2: %v %+v", collected, run.Find("a").ReusedFrom)
	}
}
