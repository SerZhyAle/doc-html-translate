package report

import (
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
	"doc-html-translate/tools/ocrlab/truth"
)

func ptr(v float64) *float64 { return &v }

// The gate is driven to each of its three outcomes. The case this exists for is the middle one: a
// bound declared for a category with no scored scene used to count as a pass, so a thin corpus
// printed PASS over a dimension nobody measured (CHECK-VERDICT rule 2).
func TestGateOutcomes(t *testing.T) {
	cats := corpus.Categories()
	if len(cats) < 2 {
		t.Fatalf("need two categories, have %d", len(cats))
	}
	a, b := cats[0], cats[1]
	th := &Thresholds{Dimensions: map[string]Dimension{
		DimRecognition: {
			Overall:    Bound{Min: ptr(0.5)},
			ByCategory: map[string]Bound{string(a): {Min: ptr(0.5)}, string(b): {Min: ptr(0.5)}},
		},
	}}
	bucket := func(recall float64) *metrics.Bucket { return &metrics.Bucket{Scenes: 1, MeanRecall: recall} }

	cases := []struct {
		name    string
		byCat   map[corpus.Category]*metrics.Bucket
		verdict string
		code    int
	}{
		{"every bound judged and met", map[corpus.Category]*metrics.Bucket{a: bucket(0.9), b: bucket(0.9)}, VerdictPass, 0},
		{"a declared category has no scored scene", map[corpus.Category]*metrics.Bucket{a: bucket(0.9)}, VerdictUnverified, 2},
		{"a failure outranks an absence", map[corpus.Category]*metrics.Bucket{a: bucket(0.1)}, VerdictFail, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sum := &metrics.Summary{RunID: "r", Edition: "desktop", Overall: *bucket(0.9), ByCategory: c.byCat}
			res := Gate(sum, th, nil)
			if res.Verdict != c.verdict || res.ExitCode() != c.code {
				t.Fatalf("verdict %q exit %d, want %q exit %d", res.Verdict, res.ExitCode(), c.verdict, c.code)
			}
			if res.Pass != (c.code == 0) {
				t.Errorf("Pass = %v with verdict %q", res.Pass, res.Verdict)
			}
			lines := strings.Split(strings.TrimSpace(res.Render()), "\n")
			if last := lines[len(lines)-1]; !strings.HasPrefix(last, "ocrlab gate: "+c.verdict) {
				t.Errorf("last line %q does not carry the verdict %q", last, c.verdict)
			}
		})
	}
}

// TestGatePositionIoU verifies that DimPosition evaluates mean IoU against the min bound.
func TestGatePositionIoU(t *testing.T) {
	th := &Thresholds{Dimensions: map[string]Dimension{
		DimPosition: {
			Overall: Bound{Min: ptr(0.77)},
		},
	}}

	passingBucket := &metrics.Bucket{Scenes: 1, MeanIoU: 0.85, WorstDrift: 0.0}
	sumPass := &metrics.Summary{RunID: "r1", Edition: "desktop", Overall: *passingBucket}
	resPass := Gate(sumPass, th, nil)
	if !resPass.Pass || resPass.Verdict != VerdictPass {
		t.Errorf("mean IoU 0.85 >= 0.77 must pass, got verdict %q", resPass.Verdict)
	}

	failingBucket := &metrics.Bucket{Scenes: 1, MeanIoU: 0.45, WorstDrift: 0.0}
	sumFail := &metrics.Summary{RunID: "r2", Edition: "desktop", Overall: *failingBucket}
	resFail := Gate(sumFail, th, nil)
	if resFail.Pass || resFail.Verdict != VerdictFail {
		t.Errorf("mean IoU 0.45 < 0.77 must fail, got verdict %q", resFail.Verdict)
	}
}

// TestGateFailsPreFixNavbarDefect demonstrates that the gate fails a run whose plates are
// systematically off their annotated groups due to the pre-fix navbar aspect guard defect,
// where plates drifted 0 px across viewports but IoU collapsed.
func TestGateFailsPreFixNavbarDefect(t *testing.T) {
	th, err := LoadThresholds("../../DEV/ocrlab/thresholds.json")
	if err != nil {
		// Try root-relative path if running from package directory
		th, err = LoadThresholds("../../../DEV/ocrlab/thresholds.json")
		if err != nil {
			t.Fatalf("failed to load thresholds.json: %v", err)
		}
	}

	// synth-uniform-paper is a 640x320 image with a ground truth box at [48, 63, 453, 167].
	// In the pre-fix navbar defect, the image in a 1216 px column rendered at 640 px width
	// while the container stayed 1216 px wide. The plate rendered at x0=91, width=770 px.
	// In 640x320 image space, the plate rect was [91, 63, 861, 167] instead of [48, 63, 453, 167].
	truthGroup := truth.Group{
		ID:         "p1",
		Bounds:     truth.Box("p1", 48, 63, 453, 167),
		Transcript: "Sample text line",
	}
	ann := &truth.Annotation{
		SchemaVersion: truth.SchemaVersion,
		SceneID:       "synth-uniform-paper",
		Origin:        truth.OriginHuman,
		ImageWidth:    640,
		ImageHeight:   320,
		Groups:        []truth.Group{truthGroup},
		Review:        truth.Review{AnnotatedBy: "synth"},
	}
	sceneMeta := &corpus.Scene{
		ID:         "synth-uniform-paper",
		Categories: []corpus.Category{corpus.CatDocument},
		Split:      corpus.SplitDev,
	}

	// Pre-fix plates at all 3 viewports: identical wrong coordinates (drift = 0.000 px).
	var preFixPlates []evidence.Plate
	for _, vp := range []string{"desktop", "tablet", "phone"} {
		preFixPlates = append(preFixPlates, evidence.Plate{
			Text:         "Sample text line",
			Rect:         evidence.Rect{X0: 91, Y0: 63, X1: 861, Y1: 167},
			Viewport:     vp,
			StressCase:   metrics.PrimaryStressCase,
			Mode:         evidence.ModeFill,
			ScrollHeight: 104,
			ClientHeight: 104,
		})
	}

	evSc := evidence.Scene{
		SceneID:     "synth-uniform-paper",
		ImageWidth:  640,
		ImageHeight: 320,
		Plates:      preFixPlates,
	}
	evRun := &evidence.Run{
		SchemaVersion: evidence.SchemaVersion,
		RunID:         "pre-fix-navbar-defect",
		Edition:       evidence.EditionDesktop,
		Viewports: []evidence.Viewport{
			{Name: "desktop", Width: 1280, Height: 800, DeviceScaleFactor: 1},
			{Name: "tablet", Width: 768, Height: 1024, DeviceScaleFactor: 1},
			{Name: "phone", Width: 390, Height: 844, DeviceScaleFactor: 2},
		},
		Scenes: []evidence.Scene{evSc},
	}

	score, err := metrics.Score(evRun, &evSc, ann, sceneMeta, nil, nil)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// Stability metric shows 0 drift because it was equally wrong across all viewports.
	if score.Placement.Drift != 0 {
		t.Errorf("drift = %v, want 0 (equally displaced across viewports)", score.Placement.Drift)
	}

	// But IoU is far below 1.0 (measured ~0.61 vs min bound 0.77).
	if score.Placement.MeanIoU >= 0.77 {
		t.Errorf("pre-fix navbar defect mean IoU = %v, expected < 0.77", score.Placement.MeanIoU)
	}

	summary := metrics.Aggregate("pre-fix-navbar-defect", evidence.EditionDesktop, []*metrics.SceneScore{score}, nil)
	res := Gate(summary, th, nil)

	if res.Pass || res.Verdict != VerdictFail {
		t.Fatalf("Gate must FAIL on pre-fix navbar defect, got verdict %q (pass=%v)", res.Verdict, res.Pass)
	}

	// Verify that the failure was specifically caught on position (mean IoU).
	var foundPosFail bool
	for _, chk := range res.Checks {
		if chk.Dimension == DimPosition && chk.Scope == "overall" && chk.Measure == "mean IoU" && !chk.Pass && !chk.Absent {
			foundPosFail = true
			if chk.Value >= chk.Limit {
				t.Errorf("failing check has value %v >= limit %v", chk.Value, chk.Limit)
			}
		}
	}
	if !foundPosFail {
		t.Error("expected overall check failure for DimPosition ('mean IoU') in GateResult")
	}
}
