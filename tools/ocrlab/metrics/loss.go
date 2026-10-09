package metrics

import (
	"fmt"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/evidence"
)

// LossStage names where text that the annotation says is there stopped on its way to a plate.
type LossStage string

const (
	// LossEngineNothing: the recognizer returned no line at all, so no gate ever saw anything.
	LossEngineNothing LossStage = "engine-returned-nothing"
	// LossConfidenceGate: the engine read lines and every one of them failed the confidence floor.
	LossConfidenceGate LossStage = "dropped-at-confidence-gate"
	// LossGroupedThenLost: lines cleared the confidence floor and were grouped, then a later gate
	// (nothing to translate, a merge over lettering already plated) discarded the group.
	LossGroupedThenLost LossStage = "grouped-then-lost"
	// LossRenderedNothing: the pipeline kept blocks but the render recorded no plate for them.
	LossRenderedNothing LossStage = "rendered-nothing"
	// LossUnknown: no diagnostics record for the scene, so the stage cannot be told.
	LossUnknown LossStage = "unknown"
)

// gateConfidence is the gate name both editions write for the confidence floor.
const gateConfidence = "confidence"

// LossPoint is the stage a scene with annotated text and no plates lost its text at, derived from
// the diagnostics sidecar alone. It is a diagnosis for a reader, not a measurement: it adds no
// failure and no threshold, and the scene's failure text is unchanged.
type LossPoint struct {
	Stage  LossStage `json:"stage"`
	Detail string    `json:"detail"`
	// Blocks is how many blocks the pipeline kept; Dropped how many lines it discarded, by gate.
	Blocks  int            `json:"blocks"`
	Dropped int            `json:"dropped"`
	ByGate  map[string]int `json:"byGate,omitempty"`
}

// String is the one-line form used in console output and the reports.
func (l LossPoint) String() string { return fmt.Sprintf("%s - %s", l.Stage, l.Detail) }

// DiagnoseLoss classifies a scene that has annotated text and no plates. rec is the scene's
// sidecar record, or nil when there is none.
func DiagnoseLoss(rec *evidence.SidecarRecord) LossPoint {
	if rec == nil {
		return LossPoint{Stage: LossUnknown, Detail: "no diagnostics record for this scene (the sidecar is missing or does not list it)"}
	}
	out := LossPoint{Blocks: len(rec.Blocks), Dropped: len(rec.Dropped)}
	byGate := map[string]int{}
	var best *evidence.SidecarDropped
	for i := range rec.Dropped {
		d := &rec.Dropped[i]
		byGate[d.Gate]++
		if d.Gate == gateConfidence && (best == nil || d.Conf > best.Conf) {
			best = d
		}
	}
	if len(byGate) > 0 {
		out.ByGate = byGate
	}

	switch {
	case len(rec.Blocks) > 0:
		out.Stage = LossRenderedNothing
		out.Detail = fmt.Sprintf("%d block(s) kept by the pipeline, but the render recorded no plate", len(rec.Blocks))
	case len(rec.Dropped) == 0:
		out.Stage = LossEngineNothing
		out.Detail = "the recognizer returned no line"
	case len(byGate) == 1 && byGate[gateConfidence] > 0:
		out.Stage = LossConfidenceGate
		out.Detail = fmt.Sprintf("%d line(s) read and all dropped at the confidence gate; best confidence %.1f against floor %.1f",
			byGate[gateConfidence], best.Conf, best.Floor)
	default:
		out.Stage = LossGroupedThenLost
		out.Detail = fmt.Sprintf("%d line(s) dropped (%s); %d cleared the confidence floor and were lost after grouping",
			len(rec.Dropped), gateCounts(byGate), len(rec.Dropped)-byGate[gateConfidence])
	}
	return out
}

// gateCounts renders "confidence 3, translatable 2" in gate-name order, so the text is stable.
func gateCounts(byGate map[string]int) string {
	names := make([]string, 0, len(byGate))
	for g := range byGate {
		names = append(names, g)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, g := range names {
		parts[i] = fmt.Sprintf("%s %d", g, byGate[g])
	}
	return strings.Join(parts, ", ")
}

// noPlatesOverText is the hard failure's own condition: annotated text, nothing drawn over it.
func noPlatesOverText(s *SceneScore) bool {
	return s.Replacement.Plates == 0 && s.Detection.FN > 0
}
