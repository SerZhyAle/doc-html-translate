package report

import (
	"doc-html-translate/tools/ocrlab/metrics"
	"testing"
)

func TestIncompleteEvidenceNeverPasses(t *testing.T) {
	th := &Thresholds{Dimensions: map[string]Dimension{DimRecognition: {Overall: Bound{Min: ptr(0)}, Tolerance: ptr(0)}}}
	for _, name := range []string{"missing scene", "render undecodable", "missing stress observation", "annotation review pending", "missing reference", "empty selection", "unmeasured concealment"} {
		t.Run(name, func(t *testing.T) {
			s := &metrics.Summary{Overall: metrics.Bucket{Scenes: 1, MeanRecall: 1}}
			switch name {
			case "empty selection":
				s.Overall.Scenes = 0
			case "unmeasured concealment":
				s.Overall.UnmeasuredConcealment = 1
			case "missing scene":
				s.Skipped = []metrics.Skipped{{SceneID: "required", Reason: "missing"}}
			case "missing reference":
			default:
				s.EvidenceIssues = []string{name}
			}
			got := Gate(s, th, nil)
			if got.ExitCode() != 2 {
				t.Fatalf("%s: %s", name, got.Render())
			}
			s.Overall.Clipped = 1
			got = Gate(s, th, nil)
			if got.ExitCode() != 1 {
				t.Fatalf("failure must outrank absence: %s", got.Render())
			}
		})
	}
}

func TestObservedDefectOutranksIncompatibleThresholds(t *testing.T) {
	th := &Thresholds{Procedure: "old", Dimensions: map[string]Dimension{DimRecognition: {Overall: Bound{Min: ptr(.9)}}}}
	s := &metrics.Summary{Procedure: "new", Overall: metrics.Bucket{Scenes: 1, FailingScenes: 1}}
	if got := Gate(s, th, nil); got.ExitCode() != 1 {
		t.Fatalf("named defects must survive unavailable baselines: %s", got.Render())
	}
	s.Overall.FailingScenes = 0
	if got := Gate(s, th, nil); got.ExitCode() != 2 {
		t.Fatalf("old numerical bounds must not judge changed measurements: %s", got.Render())
	}
}
