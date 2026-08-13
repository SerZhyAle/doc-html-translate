package synth

import (
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// The scenes are the lab's only fixtures that exist without human licence work, so every metric
// test binds to them. If regeneration were not byte-stable, every run would rewrite the
// manifest and no diff would ever be readable.
func TestGenerateIsDeterministic(t *testing.T) {
	a, annsA, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, annsB, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || len(annsA) != len(annsB) {
		t.Fatalf("scene counts differ: %d/%d vs %d/%d", len(a), len(annsA), len(b), len(annsB))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("scene %d: %s vs %s", i, a[i].ID, b[i].ID)
		}
		if a[i].SHA256 != b[i].SHA256 {
			t.Errorf("%s: regenerating changed the bytes (%s vs %s)", a[i].ID, a[i].SHA256, b[i].SHA256)
		}
		if a[i].Bytes != b[i].Bytes {
			t.Errorf("%s: regenerating changed the size (%d vs %d)", a[i].ID, a[i].Bytes, b[i].Bytes)
		}
	}
}

func TestGenerateCoversTheDiagnosticClasses(t *testing.T) {
	scenes, _, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) < 8 {
		t.Fatalf("want at least 8 diagnostic scenes, got %d", len(scenes))
	}
	want := map[corpus.Category]bool{
		corpus.CatDocument: false,
		corpus.CatComic:    false,
		corpus.CatTexture:  false,
		corpus.CatPoster:   false,
		corpus.CatScript:   false,
	}
	for _, s := range scenes {
		for _, c := range s.Categories {
			if _, ok := want[c]; ok {
				want[c] = true
			}
		}
	}
	for c, got := range want {
		if !got {
			t.Errorf("no diagnostic scene exercises %s", c)
		}
	}
}

// Generated media is tunable by definition, so it must never end up gating the holdout.
func TestGeneratedScenesAreAlwaysDev(t *testing.T) {
	scenes, _, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scenes {
		if s.Split != corpus.SplitDev {
			t.Errorf("%s is %s; a generated scene can never be a holdout gate", s.ID, s.Split)
		}
		if s.Licence != corpus.LicenceSynthetic {
			t.Errorf("%s has licence %s, want SYNTHETIC", s.ID, s.Licence)
		}
	}
}

// The whole point of drawing the scenes is that their truth is exact rather than typed, so the
// annotations they emit must satisfy the validator with no problems at all.
func TestGeneratedAnnotationsValidate(t *testing.T) {
	root := t.TempDir()
	scenes, anns, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	m := &corpus.Manifest{SchemaVersion: corpus.SchemaVersion, Scenes: scenes}
	for _, a := range anns {
		s := m.Find(a.SceneID)
		if s == nil {
			t.Fatalf("%s: annotation with no manifest entry", a.SceneID)
		}
		if ps := truth.Validate(a, s); len(ps) > 0 {
			t.Errorf("%s: %d problem(s):", a.SceneID, len(ps))
			for _, p := range ps {
				t.Errorf("    %s", p)
			}
		}
		if !a.IsTruth() {
			t.Errorf("%s: a scene drawn by the lab is exact and must be scorable", a.SceneID)
		}
	}
}

// The comic scenes exist to test damage avoidance, which needs something to damage.
func TestBalloonScenesProtectTheirOutlines(t *testing.T) {
	_, anns, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range anns {
		switch a.SceneID {
		case "synth-balloon-on-panel", "synth-adjacent-balloons":
			if len(a.Protected) == 0 {
				t.Errorf("%s: a balloon scene with nothing protected cannot test damage", a.SceneID)
			}
			for _, g := range a.Groups {
				if g.Type != truth.GroupBalloon {
					continue
				}
				if g.ReplaceArea.Empty() {
					t.Errorf("%s: balloon %s has no explicit replace area", a.SceneID, g.ID)
				}
			}
		}
	}
}

// The adjacent-balloon scene is the merge trap, and a trap that does not spring is worse than no
// test at all. The recognizer joins lines while the step from one line's top to the next stays
// within internal/ocr's ocrClusterPitchFactor (1.2) x the page's reference line pitch, so the scene
// is measured in the same currency: the leading inside a balloon against the step across to the
// next one. Two bounds have to hold at once, and the scene is only interesting between them.
//
// Below the merge bound the scene could not carry two annotated groups at all - the shipped gate
// would join the balloons and the annotation would be unreachable. Far above it, distance alone
// separates them and the scene proves nothing about boundary detection: whatever Step 07.3 adds
// would never be exercised. Measured 2026-08-11 on the drawn geometry: 26 px of leading inside a
// balloon against a 42 px step across, so the balloons sit 1.35x outside a 31.2 px bound - close,
// on purpose.
func TestAdjacentBalloonsAreActuallyAdjacent(t *testing.T) {
	// Mirrors internal/ocr ocrClusterPitchFactor (docs/PARITY.md).
	const clusterPitchFactor = 1.2
	// Beyond this multiple of a balloon's own leading the balloons are simply far apart.
	const temptingWithin = 2.0

	_, anns, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range anns {
		if a.SceneID != "synth-adjacent-balloons" {
			continue
		}
		if len(a.Groups) != 2 {
			t.Fatalf("want 2 groups, got %d", len(a.Groups))
		}
		if len(a.Groups[0].Lines) < 2 || len(a.Groups[1].Lines) < 1 {
			t.Fatal("no line geometry to measure a pitch against")
		}
		top := func(r truth.Region) int { _, y0, _, _ := r.Bounds(); return y0 }
		first, second := a.Groups[0].Lines[0], a.Groups[0].Lines[1]
		last := a.Groups[0].Lines[len(a.Groups[0].Lines)-1]

		inner := top(second) - top(first)
		cross := top(a.Groups[1].Lines[0]) - top(last)
		if inner <= 0 {
			t.Fatalf("leading inside a balloon measured as %d px", inner)
		}
		if bound := float64(inner) * clusterPitchFactor; float64(cross) <= bound {
			t.Errorf("the step across to the next balloon is %d px, inside the clusterer's %.1f px "+
				"bound (%d px leading x %.1f) - the scene's two groups would merge into one plate",
				cross, bound, inner, clusterPitchFactor)
		}
		if far := float64(inner) * temptingWithin; float64(cross) > far {
			t.Errorf("the step across to the next balloon is %d px, beyond %.1f px (%d px leading x %.1f) "+
				"- distance separates the balloons and the scene does not tempt a merge",
				cross, far, inner, temptingWithin)
		}
		return
	}
	t.Fatal("synth-adjacent-balloons was not generated")
}
