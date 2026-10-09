package evidence

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestObservationPresenceIndependentOfPlateCount(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "shot.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	v := Viewport{Name: "phone", Width: 390, Height: 844, DeviceScaleFactor: 2}
	d := &Declaration{Version: 1, Purpose: "selected-dev", Procedure: Procedure, SourceRevision: "r", SourceDigest: "s", ScorerDigest: "m", SceneIDs: []string{"negative"}, Scenes: map[string]Input{"negative": {}}, Viewports: []Viewport{v}, StressCases: []string{"none"}}
	r := &Run{Viewports: []Viewport{v}, Scenes: []Scene{{SceneID: "negative", ImageWidth: 10, ImageHeight: 10, Screenshots: Screenshots{Source: "shot.png"}, Observations: []Observation{{Viewport: "phone", StressCase: "none", Rendered: "shot.png", Concealed: "shot.png"}}}}}
	if issues := Issues(dir, r, d); len(issues) > 0 {
		t.Fatal(issues)
	}
	r.Scenes[0].Observations = nil
	if issues := Issues(dir, r, d); !strings.Contains(strings.Join(issues, " "), "missing observation") {
		t.Fatal(issues)
	}
	r.Scenes = nil
	if issues := Issues(dir, r, d); !strings.Contains(strings.Join(issues, " "), "required scene absent") {
		t.Fatal(issues)
	}
}

func TestShotDimensionsAndPathAreChecked(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.png"), []byte("broken"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"", "../escape.png", "bad.png", "missing.png"} {
		if err := CheckShot(dir, p, 10, 10); err == nil {
			t.Fatalf("accepted %q", p)
		}
	}
}
