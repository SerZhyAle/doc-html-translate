package evidence

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	fixViewports = []Viewport{{Name: "desktop", Width: 1280, Height: 800, DeviceScaleFactor: 1}}
	fixStress    = []string{"none", "short"}
	fixEngine    = Engine{Tesseract: "tesseract v5.4.0", TessdataVersion: "eng=sha256:aa;rus=sha256:bb", Lang: LangPerScene}
	fixBrowser   = Browser{Name: "edge", Version: "edge 154.0.1"}
)

const (
	fixID    = "scene-a"
	fixScore = "scorer-1"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, data)
}

// fixtureDecl is the declaration every fixture bundle and every lookup in these tests derive
// their key from; a test that wants a different run edits a copy.
func fixtureDecl(media []byte) *Declaration {
	return &Declaration{
		Version: 1, Purpose: "exploratory", Procedure: Procedure,
		SourceDigest: "src", ProducerDigest: "prod", ScorerDigest: "m",
		Environment: "windows/amd64; go1.26.1; CPUs=20; GOMAXPROCS=20",
		Viewports:   fixViewports, StressCases: fixStress, SceneIDs: []string{fixID},
		Scenes: map[string]Input{fixID: {
			MediaSHA256: Digest(media), AnnotationSHA256: "ann", SceneSHA256: "scn", LetteringSHA256: "let", Lang: "eng",
		}},
	}
}

// fixtureBundle writes a complete finished desktop run of one scene and returns its directory.
func fixtureBundle(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	media := pngBytes(t, 20, 10)
	d := fixtureDecl(media)
	writeJSONFile(t, filepath.Join(dir, "declaration.json"), d)

	shot := func(file string) string {
		rel := "shots/" + fixID + "/" + file
		writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), pngBytes(t, 20, 10))
		return rel
	}
	writeFile(t, filepath.Join(dir, "shots", fixID, "source.png"), media)
	sc := Scene{
		SceneID: fixID, ImageWidth: 20, ImageHeight: 10, Lang: "eng", LangSource: "default",
		Plates:      []Plate{{Text: "hi", Viewport: "desktop", StressCase: "none"}},
		Screenshots: Screenshots{Source: "shots/" + fixID + "/source.png", Stress: map[string]string{}},
		OcrMs:       1500, RenderMs: 900,
	}
	for _, c := range fixStress {
		o := Observation{Viewport: "desktop", StressCase: c, Rendered: shot("desktop-" + c + ".png"), Concealed: shot("desktop-" + c + "-hidden.png")}
		sc.Observations = append(sc.Observations, o)
		sc.Screenshots.Stress[c] = o.Rendered
		if c == "none" {
			sc.Screenshots.Rendered = o.Rendered
		}
	}
	writeFile(t, filepath.Join(dir, "pages", fixID, "page_001.html"), []byte("<html></html>"))
	line := `{"file":` + jsonString(filepath.Join(root, name, "pages", fixID, fixID+".jpg")) + `,"width":20,"height":10,"blocks":[],"dropped":[]}` + "\n"
	writeFile(t, filepath.Join(dir, DiagFile), []byte(line))

	run := &Run{
		RunID: name, StartedAt: "2026-10-09T10:00:00Z", Edition: EditionDesktop,
		Engine: fixEngine, Browser: fixBrowser, Viewports: fixViewports, Scenes: []Scene{sc},
	}
	if err := run.Save(filepath.Join(dir, "evidence.json")); err != nil {
		t.Fatal(err)
	}
	writeJSONFile(t, filepath.Join(dir, "execution.json"), map[string]any{"sourceDigest": "src", "unchanged": true})
	if err := WriteFilesManifest(dir, d.SceneIDs); err != nil {
		t.Fatal(err)
	}
	return dir
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func fixtureKey(t *testing.T) ReuseKey {
	t.Helper()
	k, err := NewKey(EditionDesktop, fixtureDecl(pngBytes(t, 20, 10)), fixID, fixEngine, fixBrowser)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// mutateBundle rewrites the bundle's evidence after loading it, for tests that damage a record.
func mutateBundle(t *testing.T, dir string, edit func(*Run)) {
	t.Helper()
	run, err := LoadRun(filepath.Join(dir, "evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	edit(run)
	if err := run.Save(filepath.Join(dir, "evidence.json")); err != nil {
		t.Fatal(err)
	}
}

func find(t *testing.T, root, exclude string, key ReuseKey) (*Reusable, string) {
	t.Helper()
	r, why, err := FindReusable(root, exclude, fixScore, key, fixID)
	if err != nil {
		t.Fatal(err)
	}
	return r, why
}

func TestIdenticalKeyReusesAndKeepsProvenance(t *testing.T) {
	root := t.TempDir()
	fixtureBundle(t, root, "earlier")
	r, why := find(t, root, "", fixtureKey(t))
	if r == nil {
		t.Fatalf("an identical key was refused: %s", why)
	}
	if r.From.Integrity != "sizes" || r.From.CollectedAt != "2026-10-09T10:00:00Z" || r.From.ProducerDigest != "prod" || r.From.DeclarationSHA256 == "" {
		t.Fatalf("provenance incomplete: %+v", r.From)
	}

	dst := filepath.Join(root, "later")
	sc, err := Apply(r, dst)
	if err != nil {
		t.Fatal(err)
	}
	if sc.ReusedFrom == nil || sc.ReusedFrom.ReusedAt == "" || !strings.HasSuffix(sc.ReusedFrom.Bundle, "earlier") {
		t.Fatalf("a reused scene must say where it came from: %+v", sc.ReusedFrom)
	}
	if sc.OcrMs != 1500 || len(sc.Observations) != 2 {
		t.Fatalf("the earlier record was not carried over: %+v", sc)
	}
	for _, rel := range []string{"shots/scene-a/source.png", "shots/scene-a/desktop-short-hidden.png", "pages/scene-a/page_001.html"} {
		if _, err := os.Stat(filepath.Join(dst, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("not copied: %s", rel)
		}
	}
	diag, err := os.ReadFile(filepath.Join(dst, DiagFile))
	if err != nil {
		t.Fatal(err)
	}
	want := jsonString(filepath.Join(dst, "pages", fixID, fixID+".jpg"))
	if !strings.Contains(string(diag), `"file":`+want) || !strings.Contains(string(diag), `"width":20`) {
		t.Fatalf("the sidecar line must follow the pages and keep its other bytes:\n%s", diag)
	}
}

func TestEachKeyComponentForcesCollection(t *testing.T) {
	root := t.TempDir()
	fixtureBundle(t, root, "earlier")
	cases := map[string]func(*ReuseKey){
		"edition":         func(k *ReuseKey) { k.Edition = EditionExtension },
		"producer digest": func(k *ReuseKey) { k.Producer = "prod-edited" },
		"procedure":       func(k *ReuseKey) { k.Procedure = "ocrlab-999" },
		"environment":     func(k *ReuseKey) { k.Environment = "linux/amd64; go1.26.1" },
		"scene media":     func(k *ReuseKey) { k.Input.MediaSHA256 = "other" },
		"annotation":      func(k *ReuseKey) { k.Input.AnnotationSHA256 = "other" },
		"lettering mask":  func(k *ReuseKey) { k.Input.LetteringSHA256 = "other" },
		"corpus record":   func(k *ReuseKey) { k.Input.SceneSHA256 = "other" },
		"language":        func(k *ReuseKey) { k.Input.Lang = "rus" },
		"viewports": func(k *ReuseKey) {
			k.Viewports = []Viewport{{Name: "desktop", Width: 1000, Height: 800, DeviceScaleFactor: 1}}
		},
		"stress cases":       func(k *ReuseKey) { k.StressCases = []string{"none"} },
		"OCR engine version": func(k *ReuseKey) { k.Tesseract = "tesseract v5.5.0" },
		"language data":      func(k *ReuseKey) { k.Tessdata = "eng=sha256:changed" },
		"browser":            func(k *ReuseKey) { k.Browser = "edge|edge 155.0.0" },
	}
	for component, change := range cases {
		t.Run(component, func(t *testing.T) {
			k := fixtureKey(t)
			change(&k)
			r, why := find(t, root, "", k)
			if r != nil {
				t.Fatalf("a changed %s still reused the earlier scene", component)
			}
			// producer-level changes are caught by the cheap prefilter with its own words.
			if component != "producer digest" && component != "edition" && !strings.Contains(why, component) {
				t.Fatalf("refused for the wrong reason: %q", why)
			}
		})
	}
}

func TestTessdataOfAnUnreadLanguageDoesNotMatter(t *testing.T) {
	root := t.TempDir()
	fixtureBundle(t, root, "earlier")
	engine := fixEngine
	engine.TessdataVersion = "eng=sha256:aa;rus=sha256:a-different-russian-pack"
	k, err := NewKey(EditionDesktop, fixtureDecl(pngBytes(t, 20, 10)), fixID, engine, fixBrowser)
	if err != nil {
		t.Fatal(err)
	}
	if r, why := find(t, root, "", k); r == nil {
		t.Fatalf("a scene read with English was refused over the Russian pack: %s", why)
	}
}

func TestNewKeyRefusesUnknownIdentities(t *testing.T) {
	media := pngBytes(t, 20, 10)
	cases := map[string]func(*Declaration, *Engine, *Browser){
		"no producer digest": func(d *Declaration, _ *Engine, _ *Browser) { d.ProducerDigest = "" },
		"unknown engine":     func(_ *Declaration, e *Engine, _ *Browser) { e.Tesseract = "unknown" },
		"no browser":         func(_ *Declaration, _ *Engine, b *Browser) { b.Version = "" },
		"no pack identity":   func(_ *Declaration, e *Engine, _ *Browser) { e.TessdataVersion = "rus=sha256:bb" },
		"no language":        func(d *Declaration, _ *Engine, _ *Browser) { in := d.Scenes[fixID]; in.Lang = ""; d.Scenes[fixID] = in },
	}
	for name, change := range cases {
		d, e, b := fixtureDecl(media), fixEngine, fixBrowser
		change(d, &e, &b)
		if _, err := NewKey(EditionDesktop, d, fixID, e, b); err == nil {
			t.Errorf("%s: two unknown identities must not produce a key", name)
		}
	}
}

func TestIncompleteOrDamagedScenesAreNeverReused(t *testing.T) {
	damage := map[string]func(t *testing.T, dir string){
		"error": func(t *testing.T, dir string) {
			mutateBundle(t, dir, func(r *Run) { r.Scenes[0].Error = "convert: boom" })
		},
		"unmeasured": func(t *testing.T, dir string) {
			mutateBundle(t, dir, func(r *Run) { r.Scenes[0].Unmeasured = "language data unavailable: rus" })
		},
		"partial observations": func(t *testing.T, dir string) {
			mutateBundle(t, dir, func(r *Run) { r.Scenes[0].Observations = r.Scenes[0].Observations[:1] })
		},
		"duplicate observation": func(t *testing.T, dir string) {
			mutateBundle(t, dir, func(r *Run) {
				r.Scenes[0].Observations[1] = r.Scenes[0].Observations[0]
			})
		},
		"missing screenshot": func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "shots", fixID, "desktop-short-hidden.png")); err != nil {
				t.Fatal(err)
			}
		},
		"wrong dimensions": func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, "shots", fixID, "desktop-none.png"), pngBytes(t, 5, 5))
		},
		"corrupt screenshot": func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, "shots", fixID, "desktop-none.png"), []byte("not a png"))
		},
		"source is not the media": func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, "shots", fixID, "source.png"), pngBytes(t, 20, 11))
		},
		"missing pages": func(t *testing.T, dir string) {
			if err := os.RemoveAll(filepath.Join(dir, "pages", fixID)); err != nil {
				t.Fatal(err)
			}
		},
		"file grew": func(t *testing.T, dir string) {
			p := filepath.Join(dir, "pages", fixID, "page_001.html")
			writeFile(t, p, []byte("<html>edited later</html>"))
		},
		"file added": func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, "shots", fixID, "stray.png"), []byte("x"))
		},
		"source changed during that run": func(t *testing.T, dir string) {
			writeJSONFile(t, filepath.Join(dir, "execution.json"), map[string]any{"sourceDigest": "src", "unchanged": false})
		},
		"no execution record": func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "execution.json")); err != nil {
				t.Fatal(err)
			}
		},
		"corrupt manifest": func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, FilesManifest), []byte("{"))
		},
		"scene missing from the manifest": func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, FilesManifest), []byte("{}"))
		},
	}
	for name, hurt := range damage {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dir := fixtureBundle(t, root, "earlier")
			hurt(t, dir)
			if r, _ := find(t, root, "", fixtureKey(t)); r != nil {
				t.Fatalf("%s: the scene was reused", name)
			}
		})
	}
}

func TestRunWithoutFileManifestIsReusableOnDimensionsOnly(t *testing.T) {
	root := t.TempDir()
	dir := fixtureBundle(t, root, "earlier")
	if err := os.Remove(filepath.Join(dir, FilesManifest)); err != nil {
		t.Fatal(err)
	}
	r, why := find(t, root, "", fixtureKey(t))
	if r == nil {
		t.Fatalf("an older run with intact captures was refused: %s", why)
	}
	if r.From.Integrity != "dimensions" {
		t.Fatalf("the weaker check must be named, got %q", r.From.Integrity)
	}
}

func TestHardFailureRuleNeedsTheSameScorer(t *testing.T) {
	score := func(t *testing.T, dir, scorer string, failures []string) {
		writeJSONFile(t, filepath.Join(dir, "summary.json"), map[string]any{"scorerDigest": scorer})
		writeJSONFile(t, filepath.Join(dir, "scores.json"), []map[string]any{{"sceneId": fixID, "failures": failures}})
	}
	cases := []struct {
		name     string
		scorer   string
		failures []string
		reused   bool
	}{
		{"passed under the same scorer", fixScore, nil, true},
		{"hard failure under the same scorer", fixScore, []string{"painted over protected content"}, false},
		{"failure recorded by another scorer", "scorer-0", []string{"painted over protected content"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			score(t, fixtureBundle(t, root, "earlier"), c.scorer, c.failures)
			if r, why := find(t, root, "", fixtureKey(t)); (r != nil) != c.reused {
				t.Fatalf("reused=%v want %v (%s)", r != nil, c.reused, why)
			}
		})
	}
	root := t.TempDir()
	fixtureBundle(t, root, "unscored")
	if r, why := find(t, root, "", fixtureKey(t)); r == nil {
		t.Fatalf("a run that was never scored has no failure to hold against it: %s", why)
	}
}

func TestLegacyDeclarationWithoutProducerDigestIsNeverReused(t *testing.T) {
	root := t.TempDir()
	dir := fixtureBundle(t, root, "legacy")
	d := fixtureDecl(pngBytes(t, 20, 10))
	d.ProducerDigest = ""
	writeJSONFile(t, filepath.Join(dir, "declaration.json"), d)
	r, why := find(t, root, "", fixtureKey(t))
	if r != nil || !strings.Contains(why, "predates scene reuse") {
		t.Fatalf("reused=%v why=%q", r != nil, why)
	}
}

func TestNewestBundleWinsAndTheRunBeingWrittenIsExcluded(t *testing.T) {
	root := t.TempDir()
	old := fixtureBundle(t, root, "old")
	newest := fixtureBundle(t, root, "newest")
	self := fixtureBundle(t, root, "self")
	now := time.Now()
	for dir, at := range map[string]time.Time{old: now.Add(-3 * time.Hour), newest: now.Add(-1 * time.Hour), self: now} {
		if err := os.Chtimes(filepath.Join(dir, "declaration.json"), at, at); err != nil {
			t.Fatal(err)
		}
	}
	r, why := find(t, root, self, fixtureKey(t))
	if r == nil || filepath.Base(r.Bundle) != "newest" {
		t.Fatalf("want newest, got %v (%s)", r, why)
	}
	if r, _ := find(t, filepath.Join(root, "old"), "", fixtureKey(t)); r == nil || filepath.Base(r.Bundle) != "old" {
		t.Fatalf("a single bundle directory as the search root must be searched directly")
	}
	if r, why := find(t, filepath.Join(root, "absent"), "", fixtureKey(t)); r != nil || why == "" {
		t.Fatalf("a missing search root has nothing to reuse and must say so: %q", why)
	}
}

func TestApplyLeavesNothingBehindWhenTheTargetExists(t *testing.T) {
	root := t.TempDir()
	fixtureBundle(t, root, "earlier")
	r, why := find(t, root, "", fixtureKey(t))
	if r == nil {
		t.Fatal(why)
	}
	dst := filepath.Join(root, "later")
	writeFile(t, filepath.Join(dst, "shots", fixID, "already.png"), []byte("x"))
	if _, err := Apply(r, dst); err == nil {
		t.Fatal("an existing scene folder must not be overlaid")
	}
	if _, err := os.Stat(filepath.Join(dst, "pages", fixID)); err == nil {
		t.Fatal("a refused copy left pages behind")
	}
}

func TestReuseChainKeepsTheOriginalCollectionTime(t *testing.T) {
	root := t.TempDir()
	dir := fixtureBundle(t, root, "second")
	mutateBundle(t, dir, func(r *Run) {
		r.StartedAt = "2026-10-10T10:00:00Z"
		r.Scenes[0].ReusedFrom = &ReusedFrom{Bundle: "first", CollectedAt: "2026-10-01T08:00:00Z"}
	})
	r, why := find(t, root, "", fixtureKey(t))
	if r == nil {
		t.Fatal(why)
	}
	if r.From.CollectedAt != "2026-10-01T08:00:00Z" {
		t.Fatalf("a copy of a copy must still name when the files were collected, got %s", r.From.CollectedAt)
	}
}

func TestEvidenceWithoutReusedFromStillLoads(t *testing.T) {
	root := t.TempDir()
	dir := fixtureBundle(t, root, "earlier")
	run, err := LoadRun(filepath.Join(dir, "evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	if run.Scenes[0].ReusedFrom != nil {
		t.Fatal("a collected scene must not carry provenance")
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "evidence.json"))
	if bytes.Contains(raw, []byte("reusedFrom")) {
		t.Fatal("the field is omitempty and must not appear on a collected scene")
	}
}

func TestProducerDigestIgnoresTestsAndScorersButNotProductFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) { writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), []byte(content)) }
	write("internal/ocr/ocr.go", "package ocr")
	write("internal/ocr/ocr_test.go", "package ocr")
	write("internal/ocr/testdata/fixture.bin", "one")
	write("internal/ocr/embedded.txt", "asset")
	write("extension/src/ocr.js", "export {}")
	write("extension/src/ocr.test.mjs", "test")
	write("tools/ocrlab/runner/run.go", "package runner")
	write("tools/ocrlab/metrics/score.go", "package metrics")
	write("go.mod", "module x")
	entries := []string{"internal", "extension/src", "tools/ocrlab/runner", "go.mod"}
	digest := func() string {
		d, err := ProducerDigestOf(root, entries)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	base := digest()

	write("internal/ocr/ocr_test.go", "package ocr // edited")
	write("internal/ocr/testdata/fixture.bin", "two")
	write("extension/src/ocr.test.mjs", "edited test")
	write("tools/ocrlab/metrics/score.go", "package metrics // a scorer edit")
	if digest() != base {
		t.Fatal("editing a test file, testdata or a scorer must not discard a collection")
	}
	for _, rel := range []string{"internal/ocr/ocr.go", "internal/ocr/embedded.txt", "extension/src/ocr.js", "tools/ocrlab/runner/run.go", "go.mod"} {
		before, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		write(rel, string(before)+" ")
		if digest() == base {
			t.Fatalf("editing %s did not change the producer digest", rel)
		}
		write(rel, string(before))
	}
	write("internal/ocr/new.go", "package ocr")
	if digest() == base {
		t.Fatal("a new product file did not change the producer digest")
	}
	if _, err := ProducerDigestOf(root, []string{"internal", "absent"}); err == nil {
		t.Fatal("a missing entry must be an error, not a smaller digest")
	}
}

func TestFilesManifestListsSceneFiles(t *testing.T) {
	root := t.TempDir()
	dir := fixtureBundle(t, root, "earlier")
	m, present, err := loadFilesManifest(dir)
	if err != nil || !present || len(m[fixID]) != 6 {
		t.Fatalf("manifest: present=%v err=%v files=%v", present, err, m[fixID])
	}
}
