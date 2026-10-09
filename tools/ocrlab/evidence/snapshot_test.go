package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

func TestPortableSnapshotMutationsCannotPass(t *testing.T) {
	for _, change := range []string{"none", "media", "truth", "scene", "rights", "family"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			media := []byte("same media bytes")
			if err := os.WriteFile(filepath.Join(dir, "source"), media, 0644); err != nil {
				t.Fatal(err)
			}
			s := corpus.Scene{ID: "s", File: "source", Width: 10, Height: 10, SHA256: Digest(media), Licence: corpus.LicenceSynthetic, Split: corpus.SplitDev}
			m := &corpus.Manifest{SchemaVersion: corpus.SchemaVersion, Scenes: []corpus.Scene{s}}
			if err := corpus.Save(filepath.Join(dir, "corpus.json"), m); err != nil {
				t.Fatal(err)
			}
			if err := corpus.Save(filepath.Join(dir, "family-corpus.json"), m); err != nil {
				t.Fatal(err)
			}
			a := &truth.Annotation{SceneID: "s", Origin: truth.OriginHuman, ImageWidth: 10, ImageHeight: 10, Ambiguity: truth.AmbiguityClear, Review: truth.Review{AnnotatedBy: "Human"}}
			if err := truth.Save(truth.FinalPath(filepath.Join(dir, "truth"), "s"), a); err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(&s)
			ann, _ := os.ReadFile(truth.FinalPath(filepath.Join(dir, "truth"), "s"))
			family, _ := os.ReadFile(filepath.Join(dir, "family-corpus.json"))
			d := &Declaration{Purpose: "selected-dev", SceneIDs: []string{"s"}, Scenes: map[string]Input{"s": {MediaSHA256: Digest(media), AnnotationSHA256: Digest(ann), SceneSHA256: Digest(data)}}, FamilyDigest: Digest(family)}
			d.SourceDigest = "source-tree"
			if err := os.WriteFile(filepath.Join(dir, "execution.json"), []byte(`{"sourceDigest":"source-tree","unchanged":true}`), 0644); err != nil {
				t.Fatal(err)
			}
			r := &Run{Scenes: []Scene{{SceneID: "s", ImageWidth: 10, ImageHeight: 10, Screenshots: Screenshots{Source: "source"}}}}
			switch change {
			case "media":
				mustWrite(t, os.WriteFile(filepath.Join(dir, "source"), []byte("changed"), 0644))
			case "truth":
				a.Review.AnnotatedBy = ""
				mustWrite(t, truth.Save(truth.FinalPath(filepath.Join(dir, "truth"), "s"), a))
			case "scene":
				m.Scenes[0].Width = 11
				mustWrite(t, corpus.Save(filepath.Join(dir, "corpus.json"), m))
			case "rights":
				m.Scenes[0].Licence = corpus.LicencePD
				m.Scenes[0].LicenceVerifiedBy = "commons-api"
				mustWrite(t, corpus.Save(filepath.Join(dir, "corpus.json"), m))
			case "family":
				mustWrite(t, os.WriteFile(filepath.Join(dir, "family-corpus.json"), []byte("{}"), 0644))
			}
			issues := SnapshotIssues(dir, r, d)
			if (len(issues) == 0) != (change == "none") {
				t.Fatalf("%s: %v", change, issues)
			}
		})
	}
}

func TestInvalidDeclarationsAreUnavailable(t *testing.T) {
	r := &Run{}
	d := &Declaration{Version: 99, Purpose: "made-up", SceneIDs: []string{"s", "s"}, Scenes: map[string]Input{}, Viewports: []Viewport{{Name: "phone"}, {Name: "phone"}}, StressCases: []string{"none", "none"}}
	issues := strings.Join(Issues(t.TempDir(), r, d), " ")
	for _, want := range []string{"unsupported declaration", "duplicate declared scene", "input identity absent", "duplicate declared viewport", "duplicate declared stress"} {
		if !strings.Contains(issues, want) {
			t.Fatalf("missing %s in %s", want, issues)
		}
	}
}

func mustWrite(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
