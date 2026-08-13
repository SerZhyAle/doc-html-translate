package truth

import (
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
)

func devScene(id string) *corpus.Scene {
	return &corpus.Scene{ID: id, Width: 100, Height: 100, Split: corpus.SplitDev}
}

func holdoutScene(id string) *corpus.Scene {
	return &corpus.Scene{ID: id, Width: 100, Height: 100, Split: corpus.SplitHoldout}
}

// ann builds a minimal, valid annotation so a test can break exactly one thing.
func ann(id string) *Annotation {
	return &Annotation{
		SchemaVersion: SchemaVersion,
		SceneID:       id,
		Origin:        OriginHuman,
		ImageWidth:    100,
		ImageHeight:   100,
		Ambiguity:     AmbiguityClear,
		Groups: []Group{{
			ID:           "g1",
			Type:         GroupParagraph,
			Transcript:   "hello",
			ReadingOrder: 1,
			Lines:        []Region{Box("l1", 10, 10, 60, 30)},
			Bounds:       Box("b1", 10, 10, 60, 30),
			ReplaceArea:  Box("r1", 10, 10, 60, 30),
		}},
		Review: Review{AnnotatedBy: "alice", AnnotatedOn: "2026-08-11"},
	}
}

func hasRule(ps []Problem, r Rule) bool {
	for _, p := range ps {
		if p.Rule == r {
			return true
		}
	}
	return false
}

// The rule the whole lab rests on: an OCR draft is never truth.
func TestIsTruthRejectsSeededAndUnsigned(t *testing.T) {
	if (&Annotation{Origin: OriginOCRSeed, Review: Review{AnnotatedBy: "alice"}}).IsTruth() {
		t.Error("an OCR-seeded draft must never count as truth")
	}
	if (&Annotation{Origin: OriginHuman}).IsTruth() {
		t.Error("an unsigned annotation must never count as truth")
	}
	if (&Annotation{Origin: "", Review: Review{AnnotatedBy: "alice"}}).IsTruth() {
		t.Error("an unset origin must not default to trusted")
	}
	var nilAnn *Annotation
	if nilAnn.IsTruth() {
		t.Error("a missing annotation must never count as truth")
	}
	if !ann("s").IsTruth() {
		t.Error("a signed human annotation must count as truth")
	}
	if got := (&Annotation{Origin: OriginOCRSeed}).NotTruthReason(); got == "" {
		t.Error("a skipped scene must carry a reason, not just be absent")
	}
}

// A holdout scene needs a second pair of eyes; a dev scene does not.
func TestHoldoutNeedsIndependentCheck(t *testing.T) {
	a := ann("s")
	if ps := Validate(a, devScene("s")); hasRule(ps, RuleUnreviewed) {
		t.Errorf("a dev scene must not demand a second reviewer: %v", ps)
	}
	if ps := Validate(a, holdoutScene("s")); !hasRule(ps, RuleUnreviewed) {
		t.Error("a holdout scene with no checkedBy must be reported")
	}

	a.Review.CheckedBy = "alice" // the annotator checking their own work
	if ps := Validate(a, holdoutScene("s")); !hasRule(ps, RuleSelfChecked) {
		t.Error("the independent check must not be the annotator")
	}

	a.Review.CheckedBy = "bob"
	if ps := Validate(a, holdoutScene("s")); hasRule(ps, RuleUnreviewed) || hasRule(ps, RuleSelfChecked) {
		t.Errorf("a properly reviewed holdout annotation must pass: %v", ps)
	}

	draft := ann("s")
	draft.Origin = OriginOCRSeed
	draft.Review.CheckedBy = "bob"
	if ps := Validate(draft, holdoutScene("s")); !hasRule(ps, RuleDraftInHoldout) {
		t.Error("a draft must never gate a holdout scene")
	}
}

func TestValidateGeometry(t *testing.T) {
	t.Run("region outside the image", func(t *testing.T) {
		a := ann("s")
		a.Groups[0].Bounds = Box("b1", 10, 10, 400, 30)
		if ps := Validate(a, devScene("s")); !hasRule(ps, RuleRegionOutside) {
			t.Errorf("a box past the image edge must be reported: %v", ps)
		}
	})

	t.Run("size disagrees with the manifest", func(t *testing.T) {
		a := ann("s")
		a.ImageWidth = 200
		if ps := Validate(a, devScene("s")); !hasRule(ps, RuleSizeMismatch) {
			t.Error("an annotation for a differently-sized image must be reported")
		}
	})

	t.Run("replace area misses its own text", func(t *testing.T) {
		a := ann("s")
		a.Groups[0].ReplaceArea = Box("r1", 70, 70, 95, 95) // nowhere near the line
		if ps := Validate(a, devScene("s")); !hasRule(ps, RuleReplaceMissesText) {
			t.Errorf("an area that does not cover its text must be reported: %v", ps)
		}
	})

	// A balloon's permitted area is legitimately larger than the text inside it - that is what
	// gives a longer translation room to reflow, so it must not be reported.
	t.Run("replace area larger than the text is fine", func(t *testing.T) {
		a := ann("s")
		a.Groups[0].ReplaceArea = Box("r1", 5, 5, 90, 60)
		if ps := Validate(a, devScene("s")); hasRule(ps, RuleReplaceMissesText) {
			t.Errorf("a roomy balloon interior must be allowed: %v", ps)
		}
	})

	t.Run("replace area hits protected content", func(t *testing.T) {
		a := ann("s")
		a.Protected = []Region{Box("border", 0, 0, 100, 20)}
		if ps := Validate(a, devScene("s")); !hasRule(ps, RuleReplaceProtect) {
			t.Error("an area overlapping a protected region must be reported")
		}
	})

	t.Run("reading order must be a permutation", func(t *testing.T) {
		a := ann("s")
		a.Groups = append(a.Groups, Group{
			ID: "g2", Type: GroupCaption, ReadingOrder: 5,
			Bounds: Box("b2", 10, 40, 60, 60), ReplaceArea: Box("r2", 10, 40, 60, 60),
		})
		if ps := Validate(a, devScene("s")); !hasRule(ps, RuleBadReadingOrder) {
			t.Error("reading orders with a gap must be reported")
		}
	})

	t.Run("transcript without geometry", func(t *testing.T) {
		a := ann("s")
		a.Groups[0].Lines = nil
		if ps := Validate(a, devScene("s")); !hasRule(ps, RuleNoLines) {
			t.Error("a transcript with no line geometry must be reported")
		}
	})
}

// A polygon replace-area whose bounding box overlaps a protected region, but whose actual
// filled shape does not, must pass. Approximating by bounding boxes would reject exactly the
// careful annotations the spec asks for.
func TestProtectedOverlapUsesTheRealShape(t *testing.T) {
	a := ann("s")
	// An L-shaped area hugging the left and bottom, and a protected square top-right. The two
	// bounding boxes overlap; the shapes do not.
	a.Groups[0].Lines = []Region{Box("l1", 5, 60, 40, 90)}
	a.Groups[0].Bounds = Box("b1", 5, 5, 40, 90)
	a.Groups[0].ReplaceArea = Region{ID: "r1", Kind: RegionPolygon, Points: [][2]int{
		{5, 5}, {40, 5}, {40, 40}, {90, 40}, {90, 95}, {5, 95},
	}}
	a.Protected = []Region{Box("art", 50, 5, 90, 35)}
	if ps := Validate(a, devScene("s")); hasRule(ps, RuleReplaceProtect) {
		t.Errorf("bounding boxes overlap but the shapes do not - must not be reported: %v", ps)
	}

	a.Protected = []Region{Box("art", 60, 50, 85, 90)} // now genuinely inside the L
	if ps := Validate(a, devScene("s")); !hasRule(ps, RuleReplaceProtect) {
		t.Error("a protected region genuinely inside the polygon must be reported")
	}
}

func TestValidateAllReportsMissingAnnotation(t *testing.T) {
	m := &corpus.Manifest{SchemaVersion: corpus.SchemaVersion, Scenes: []corpus.Scene{
		{ID: "annotated", Width: 100, Height: 100, Split: corpus.SplitDev},
		{ID: "bare", Width: 100, Height: 100, Split: corpus.SplitDev},
		{ID: "context", Width: 100, Height: 100, Split: corpus.SplitDev, Note: corpus.NoteUnscored},
	}}
	anns := map[string]*Annotation{"annotated": ann("annotated")}
	ps := ValidateAll(anns, m)

	var missing []string
	for _, p := range ps {
		if p.Rule == Rule(corpus.RuleAnnotationMissing) {
			missing = append(missing, p.SceneID)
		}
	}
	if len(missing) != 1 || missing[0] != "bare" {
		t.Fatalf("exactly the scored, unannotated scene must be reported, got %v", missing)
	}
}

func TestLoadDirPrefersTheReviewedFileOverTheDraft(t *testing.T) {
	dir := t.TempDir()
	draft := ann("s")
	draft.Origin = OriginOCRSeed
	if err := Save(DraftPath(dir, "s"), draft); err != nil {
		t.Fatal(err)
	}
	if err := Save(FinalPath(dir, "s"), ann("s")); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got["s"].IsTruth() {
		t.Error("a reviewed annotation must win over a leftover draft")
	}
}

func TestLoadDirTolerAtesAMissingDirectory(t *testing.T) {
	got, err := LoadDir(filepath.Join(t.TempDir(), "not-there"))
	if err != nil {
		t.Fatalf("an empty corpus is a legitimate early state: %v", err)
	}
	if len(got) != 0 {
		t.Error("expected no annotations")
	}
}

func TestLoadRejectsForeignSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":99,"sceneId":"s"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("an annotation from a future schema must not be read as if it were this one")
	}
}

func TestRegionRasterize(t *testing.T) {
	box := Box("b", 2, 2, 6, 5)
	if got := box.Rasterize(10, 10).Area(); got != 12 {
		t.Errorf("box area = %d, want 12", got)
	}

	// The same rectangle as a polygon must rasterize to the same pixels.
	poly := Region{Kind: RegionPolygon, Points: [][2]int{{2, 2}, {6, 2}, {6, 5}, {2, 5}}}
	if got := poly.Rasterize(10, 10).Area(); got != 12 {
		t.Errorf("polygon area = %d, want 12", got)
	}

	if got := box.Rasterize(10, 10).IntersectArea(poly.Rasterize(10, 10)); got != 12 {
		t.Errorf("identical shapes must intersect fully, got %d", got)
	}
	far := Box("f", 8, 8, 10, 10)
	if got := box.Rasterize(10, 10).IntersectArea(far.Rasterize(10, 10)); got != 0 {
		t.Errorf("disjoint shapes must not intersect, got %d", got)
	}
}

func TestRegionValid(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    Region
		ok   bool
	}{
		{"good box", Box("", 0, 0, 5, 5), true},
		{"inverted box", Box("", 5, 5, 0, 0), false},
		{"box with three points", Region{Kind: RegionBox, Points: [][2]int{{0, 0}, {1, 1}, {2, 2}}}, false},
		{"triangle", Region{Kind: RegionPolygon, Points: [][2]int{{0, 0}, {4, 0}, {2, 4}}}, true},
		{"two-point polygon", Region{Kind: RegionPolygon, Points: [][2]int{{0, 0}, {4, 0}}}, false},
		{"unknown kind", Region{Kind: "blob", Points: [][2]int{{0, 0}, {1, 1}}}, false},
	} {
		if err := tc.r.Valid(); (err == nil) != tc.ok {
			t.Errorf("%s: Valid() = %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}
