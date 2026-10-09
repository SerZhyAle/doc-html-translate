package metrics

import (
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/evidence"
)

func dropped(gate string, conf, floor float64) evidence.SidecarDropped {
	return evidence.SidecarDropped{Text: "x", Conf: conf, Floor: floor, Gate: gate}
}

func TestDiagnoseLossNamesEachStage(t *testing.T) {
	for _, c := range []struct {
		name   string
		rec    *evidence.SidecarRecord
		stage  LossStage
		detail []string
	}{
		{"no record", nil, LossUnknown, []string{"no diagnostics record"}},
		{"engine returned nothing", &evidence.SidecarRecord{}, LossEngineNothing, []string{"returned no line"}},
		{"dropped at the confidence gate",
			&evidence.SidecarRecord{Dropped: []evidence.SidecarDropped{
				dropped("confidence", 20, 60), dropped("confidence", 31.5, 60), dropped("confidence", 28, 60),
			}},
			LossConfidenceGate, []string{"3 line(s)", "best confidence 31.5 against floor 60.0"}},
		{"grouped then lost",
			&evidence.SidecarRecord{Dropped: []evidence.SidecarDropped{
				dropped("confidence", 20, 60), dropped("translatable", 88, 60), dropped("screen-merge", 75, 45),
			}},
			LossGroupedThenLost, []string{"3 line(s) dropped", "confidence 1, screen-merge 1, translatable 1", "2 cleared the confidence floor"}},
		{"rendered nothing",
			&evidence.SidecarRecord{Blocks: []evidence.SidecarBlock{{Text: "kept", Conf: 90}}},
			LossRenderedNothing, []string{"1 block(s) kept", "no plate"}},
		{"blocks win over later drops",
			&evidence.SidecarRecord{
				Blocks:  []evidence.SidecarBlock{{Text: "kept", Conf: 90}},
				Dropped: []evidence.SidecarDropped{dropped("confidence", 10, 60)},
			},
			LossRenderedNothing, []string{"1 block(s) kept"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := DiagnoseLoss(c.rec)
			if got.Stage != c.stage {
				t.Fatalf("stage = %q, want %q (%s)", got.Stage, c.stage, got.Detail)
			}
			for _, want := range c.detail {
				if !strings.Contains(got.Detail, want) {
					t.Errorf("detail %q lacks %q", got.Detail, want)
				}
			}
		})
	}
}

// The loss point explains the "no plates" failure and leaves its wording alone: compare.go diffs
// failure strings exactly, so rewording them would turn every existing run into a regression.
func TestScoreAttachesLossToNoPlatesWithoutChangingTheFailure(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]
	sc := evScene(a, nil)

	rec := &evidence.SidecarRecord{Dropped: []evidence.SidecarDropped{dropped("confidence", 25, 60)}}
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], Captures{Diag: rec})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Failures) == 0 || !strings.HasPrefix(got.Failures[0], "no plates at all over ") ||
		!strings.HasSuffix(got.Failures[0], " annotated group(s) - the original text is left as-is") {
		t.Fatalf("failure wording changed: %v", got.Failures)
	}
	if got.Loss == nil || got.Loss.Stage != LossConfidenceGate {
		t.Fatalf("loss = %+v, want the confidence gate", got.Loss)
	}

	withoutSidecar, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], Captures{})
	if err != nil {
		t.Fatal(err)
	}
	if withoutSidecar.Loss == nil || withoutSidecar.Loss.Stage != LossUnknown {
		t.Errorf("without a sidecar the stage is %+v, want unknown", withoutSidecar.Loss)
	}
	if strings.Join(withoutSidecar.Failures, "|") != strings.Join(got.Failures, "|") {
		t.Errorf("the sidecar changed the failures: %v vs %v", withoutSidecar.Failures, got.Failures)
	}
}

// A scene that has plates has no loss point to explain.
func TestScoreLeavesLossEmptyWhenPlatesExist(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]
	sc := evScene(a, perfectPlates(a, PrimaryStressCase))
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], Captures{Diag: &evidence.SidecarRecord{}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Loss != nil {
		t.Errorf("loss = %+v for a scene with plates", got.Loss)
	}
}

// An unmeasured scene is skipped with its reason; it is not scored as a scene with no plates.
func TestScoreRefusesAnUnmeasuredScene(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]
	sc := evScene(a, nil)
	sc.Unmeasured = "language data unavailable: rus"
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], Captures{})
	if got != nil || err == nil || err.Error() != "language data unavailable: rus" {
		t.Errorf("Score = %+v, %v; want no score and the reason verbatim", got, err)
	}
}
