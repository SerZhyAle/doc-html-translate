package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/runner"
)

const reuseTestScene = "scene-x"

func writeTestJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	writeTestBytes(t, path, data)
}

func writeTestBytes(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func reuseTestDecl(media []byte) *evidence.Declaration {
	return &evidence.Declaration{
		Version: 1, Purpose: "exploratory", Procedure: evidence.Procedure, SourceDigest: "src", ProducerDigest: "prod",
		Environment: "windows/amd64; go1.26.1; CPUs=8; GOMAXPROCS=8",
		Viewports:   runner.Viewports, StressCases: runner.StressNames(), SceneIDs: []string{reuseTestScene},
		Scenes: map[string]evidence.Input{reuseTestScene: {MediaSHA256: evidence.Digest(media), SceneSHA256: "scn", Lang: "eng"}},
	}
}

// extensionBundle writes a finished extension run of one scene: no pages, a diagnostics line that
// names the corpus file, every observation capture present.
func extensionBundle(t *testing.T, root string) string {
	t.Helper()
	var shot bytes.Buffer
	if err := png.Encode(&shot, image.NewRGBA(image.Rect(0, 0, 20, 10))); err != nil {
		t.Fatal(err)
	}
	media := shot.Bytes()
	dir := filepath.Join(root, "earlier")
	writeTestJSON(t, filepath.Join(dir, "declaration.json"), reuseTestDecl(media))
	writeTestBytes(t, filepath.Join(dir, "shots", reuseTestScene, "source.png"), media)
	sc := evidence.Scene{SceneID: reuseTestScene, ImageWidth: 20, ImageHeight: 10, Lang: "eng", LangSource: "default", OcrMs: 4000,
		Screenshots: evidence.Screenshots{Source: "shots/" + reuseTestScene + "/source.png"}}
	for _, v := range runner.Viewports {
		for _, c := range runner.StressNames() {
			o := evidence.Observation{Viewport: v.Name, StressCase: c,
				Rendered:  "shots/" + reuseTestScene + "/" + v.Name + "-" + c + ".png",
				Concealed: "shots/" + reuseTestScene + "/" + v.Name + "-" + c + "-hidden.png"}
			writeTestBytes(t, filepath.Join(dir, o.Rendered), media)
			writeTestBytes(t, filepath.Join(dir, o.Concealed), media)
			sc.Observations = append(sc.Observations, o)
		}
	}
	sc.Screenshots.Rendered = sc.Observations[0].Rendered
	run := &evidence.Run{
		RunID: "earlier", StartedAt: "2026-10-09T10:00:00Z", Edition: evidence.EditionExtension,
		Engine:  evidence.Engine{Tesseract: "tesseract.js 7.0.0", TessdataVersion: "eng=sha256:aa", Lang: evidence.LangPerScene},
		Browser: evidence.Browser{Name: "chrome", Version: "Chrome/151.0.1"}, Viewports: runner.Viewports, Scenes: []evidence.Scene{sc},
	}
	if err := run.Save(filepath.Join(dir, "evidence.json")); err != nil {
		t.Fatal(err)
	}
	writeTestBytes(t, filepath.Join(dir, "execution.json"), []byte(`{"sourceDigest":"src","unchanged":true}`))
	writeTestBytes(t, filepath.Join(dir, evidence.DiagFile), []byte(`{"file":"test_doc/ocrlab/scene-x.png","width":20,"height":10,"blocks":[],"dropped":[]}`+"\n"))
	if err := evidence.WriteFilesManifest(dir, []string{reuseTestScene}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func reuseArgs(root string, extra ...string) []string {
	return append([]string{
		"-run", filepath.Join(root, "later"), "-edition", "extension", "-scene", reuseTestScene,
		"-tesseract", "tesseract.js 7.0.0", "-tessdata", "eng=sha256:aa",
		"-browser-name", "chrome", "-browser-version", "Chrome/151.0.1", "-reuse-from", root,
	}, extra...)
}

func newReuseRun(t *testing.T, root string) {
	t.Helper()
	writeTestJSON(t, filepath.Join(root, "later", "declaration.json"), reuseTestDecl(func() []byte {
		d, _ := os.ReadFile(filepath.Join(root, "earlier", "shots", reuseTestScene, "source.png"))
		return d
	}()))
	old := runner.CurrentScorerDigest
	runner.CurrentScorerDigest = func() (string, error) { return "scorer-1", nil }
	t.Cleanup(func() { runner.CurrentScorerDigest = old })
}

func TestReusePrintsTheSceneRecordAndCopiesItsFiles(t *testing.T) {
	root := t.TempDir()
	extensionBundle(t, root)
	newReuseRun(t, root)

	var out, errOut bytes.Buffer
	if err := runReuse(reuseArgs(root), &out, &errOut); err != nil {
		t.Fatal(err)
	}
	var sc evidence.Scene
	if err := json.Unmarshal(out.Bytes(), &sc); err != nil {
		t.Fatalf("stdout must be exactly the scene record: %v\n%s", err, out.String())
	}
	if sc.SceneID != reuseTestScene || sc.OcrMs != 4000 || sc.ReusedFrom == nil || sc.ReusedFrom.ProducerDigest != "prod" {
		t.Fatalf("unexpected record: %+v", sc)
	}
	if !strings.Contains(errOut.String(), "reused scene-x from ") {
		t.Fatalf("the hit must be logged: %q", errOut.String())
	}
	if _, err := os.Stat(filepath.Join(root, "later", "shots", reuseTestScene, "phone-cjk-hidden.png")); err != nil {
		t.Fatal("the captures were not copied")
	}
	diag, _ := os.ReadFile(filepath.Join(root, "later", evidence.DiagFile))
	if !strings.Contains(string(diag), `"file":"test_doc/ocrlab/scene-x.png"`) {
		t.Fatalf("an extension line names the corpus file and is copied as it is: %s", diag)
	}
}

func TestReuseMissPrintsNullAndTheReason(t *testing.T) {
	root := t.TempDir()
	extensionBundle(t, root)
	newReuseRun(t, root)

	var out, errOut bytes.Buffer
	args := reuseArgs(root)
	for i, a := range args {
		if a == "-tessdata" {
			args[i+1] = "eng=sha256:other"
		}
	}
	if err := runReuse(args, &out, &errOut); err != nil {
		t.Fatalf("a miss is an ordinary outcome, not an error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "null" || !strings.Contains(errOut.String(), "language data differs") {
		t.Fatalf("stdout %q stderr %q", out.String(), errOut.String())
	}
	if _, err := os.Stat(filepath.Join(root, "later", "shots")); err == nil {
		t.Fatal("a miss must copy nothing")
	}
}

func TestReuseRefusesBadArguments(t *testing.T) {
	root := t.TempDir()
	var out, errOut bytes.Buffer
	if err := runReuse([]string{"-run", root, "-scene", "x", "-edition", "tablet"}, &out, &errOut); err == nil {
		t.Fatal("an unknown edition must be rejected")
	}
	if err := runReuse([]string{"-edition", "extension"}, &out, &errOut); err == nil {
		t.Fatal("-run and -scene are required")
	}
	if err := runReuse([]string{"-run", root, "-scene", "x", "-edition", "extension"}, &out, &errOut); err == nil {
		t.Fatal("an undeclared run cannot be reused into")
	}
}
