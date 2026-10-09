package metrics

import (
	"errors"
	"fmt"
	"image"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// PrimaryStressCase is the case scored as "the" result for a scene: the recognized text itself,
// unmodified. The other cases are reported alongside it rather than averaged into it, because a
// page that reads correctly and then breaks under a longer translation has two different facts
// worth knowing.
const PrimaryStressCase = "none"

// ErrNotTruth is returned when a scene's annotation may not be scored. Deliberately an error
// rather than a zero score: a zero would land in an aggregate and drag it, and the whole design
// rests on a skipped scene being visible.
var ErrNotTruth = errors.New("annotation is not truth")

// CostScore is what the run cost, per scene.
type CostScore struct {
	OcrMs        int64  `json:"ocrMs"`
	RenderMs     int64  `json:"renderMs"`
	PeakRSSBytes int64  `json:"peakRssBytes"`
	MemoryKind   string `json:"memoryKind,omitempty"`
	MemoryBytes  int64  `json:"memoryBytes,omitempty"`
}

// PixelDiagnostic is what one observation (a viewport and a stress case) shows at pixel level.
// The five measurements are independent: old lettering still showing (against the text-hidden
// capture), the plate rectangles' luma separation, the glyphs the plates drew, the rectangles'
// geometric intrusion on protected content, and the protected pixels actually painted.
type PixelDiagnostic struct {
	Concealment            ResidualScore    `json:"concealment"`
	BackgroundContrast     ContrastScore    `json:"backgroundContrast"`
	ReplacementReadability ReadabilityScore `json:"replacementReadability"`
	RectangleIntrusion     DamageScore      `json:"rectangleIntrusion"`
	PaintedDamage          DamageScore      `json:"paintedDamage"`
}

// Captures are the images one scene is scored against. Every part may be absent; what cannot be
// measured without it then reports unmeasured instead of zero.
type Captures struct {
	Source image.Image
	// Rendered is the legacy single normal render, used for background contrast only when no
	// observation carries a normal capture.
	Rendered image.Image
	// Open decodes a screenshot recorded in the evidence (a path relative to the run), or returns
	// nil when it is missing. Observations name their normal and text-hidden captures this way.
	Open func(rel string) image.Image
	// Lettering is the independently known mask of the source lettering, or nil. Without it the
	// residual is measured only where the background is a single tone.
	Lettering *truth.Mask
	// Diag is the scene's line of the diagnostics sidecar, or nil. Only the loss-point diagnosis
	// of a scene with no plates reads it.
	Diag *evidence.SidecarRecord
}

func (c Captures) open(rel string) image.Image {
	if c.Open == nil || rel == "" {
		return nil
	}
	return c.Open(rel)
}

// SceneScore is one scene measured across every dimension of the strategic table.
type SceneScore struct {
	SceneID    string            `json:"sceneId"`
	Edition    evidence.Edition  `json:"edition"`
	Categories []corpus.Category `json:"categories"`
	Split      corpus.Split      `json:"split"`
	Viewport   string            `json:"viewport"`

	Detection DetectionScore `json:"detection"`
	Text      TextScore      `json:"text"`
	Grouping  GroupingScore  `json:"grouping"`
	Placement PlacementScore `json:"placement"`
	Covered   float64        `json:"covered"`
	// Residual is measured on the primary observation's text-hidden capture.
	Residual           ResidualScore    `json:"residual"`
	BackgroundContrast ContrastScore    `json:"backgroundContrast"`
	Readability        ReadabilityScore `json:"replacementReadability"`
	// Damage is the protected pixels the overlay actually painted, the worst over every observation;
	// RectangleIntrusion is the plates' bounding rectangles over protected content, worst over the
	// matrix. Only Damage feeds hard failures and the gate.
	Damage             DamageScore                `json:"damage"`
	RectangleIntrusion DamageScore                `json:"rectangleIntrusion"`
	Replacement        ReplacementScore           `json:"replacement"`
	Stress             StressBreakdown            `json:"stress"`
	Cost               CostScore                  `json:"cost"`
	PixelDiagnostics   map[string]PixelDiagnostic `json:"pixelDiagnostics,omitempty"`

	// Failures names, in words, every dimension that failed in a way the strategic spec calls
	// hard. The report shows this string; an aggregate can never make it disappear.
	Failures []string `json:"failures,omitempty"`

	// Loss says where the text was lost when the scene has annotated text and no plates. It
	// explains the "no plates at all" failure and never replaces or rewords it.
	Loss *LossPoint `json:"loss,omitempty"`
}

// Skipped is a scene that could not be scored and why.
type Skipped struct {
	SceneID string `json:"sceneId"`
	Reason  string `json:"reason"`
}

// Score measures one scene at the primary viewport, with drift taken across all viewports.
//
// Any part of caps may be absent - the geometric dimensions still score, and the pixel-level
// ones report unmeasured. That is what lets `ocrlab score` re-run offline over an old evidence
// file whose screenshots have been cleaned up.
func Score(
	run *evidence.Run,
	sc *evidence.Scene,
	a *truth.Annotation,
	s *corpus.Scene,
	caps Captures,
) (*SceneScore, error) {
	if !a.IsTruth() {
		return nil, fmt.Errorf("%w: %s", ErrNotTruth, a.NotTruthReason())
	}
	if ps := truth.Validate(a, s); len(ps) > 0 {
		return nil, fmt.Errorf("invalid annotation: %s", strings.TrimSpace(ps[0].String()))
	}
	if sc.Error != "" {
		return nil, fmt.Errorf("scene errored during the run: %s", sc.Error)
	}
	if sc.Unmeasured != "" {
		return nil, errors.New(sc.Unmeasured)
	}
	w, h := a.ImageWidth, a.ImageHeight
	if w <= 0 || h <= 0 {
		return nil, errors.New("annotation declares no image size")
	}
	// A scene with no plates at all still scores, and must.
	//
	// The recognizer finding nothing in a speech balloon is the single most important result
	// this benchmark can produce, and refusing to score it would move it from "recall 0.00 on
	// the comic category" to a line in the skipped list - out of every aggregate, out of every
	// trend, and easy to read as a tooling problem rather than as the product failure it is.
	viewport := primaryViewport(run, sc)

	out := &SceneScore{
		SceneID:    sc.SceneID,
		Edition:    run.Edition,
		Categories: s.Categories,
		Split:      s.Split,
		Viewport:   viewport,
		Cost:       CostScore{OcrMs: sc.OcrMs, RenderMs: sc.RenderMs, PeakRSSBytes: sc.PeakRSSBytes, MemoryKind: sc.MemoryKind, MemoryBytes: sc.MemoryBytes},
	}

	plates := sc.PlatesFor(viewport, PrimaryStressCase)
	groups := a.Groups

	out.Detection = Detection(plates, a, groups, w, h, DefaultDetectionIoU)
	grouping, matches := Grouping(plates, groups, w, h)
	out.Grouping = grouping
	out.Text = Text(matches, a)
	out.Placement = Placement(matches)
	out.Placement.Drift, out.Placement.DriftGroup = Drift(
		MatchesByViewport(sc, groups, run.Viewports, PrimaryStressCase, w, h), w, h)

	out.RectangleIntrusion = RectangleIntrusion(plates, a, w, h)
	out.Replacement = Replacement(plates, groups, matches, w, h)
	out.Stress = StressBreakdown{}
	for _, v := range run.Viewports {
		for key, value := range ReplacementByStress(sc, groups, v.Name, w, h) {
			out.Stress[v.Name+"/"+key] = value
			intrusion := RectangleIntrusion(sc.PlatesFor(v.Name, key), a, w, h)
			if intrusion.ProtectedHit > out.RectangleIntrusion.ProtectedHit {
				out.RectangleIntrusion = intrusion
			}
		}
	}

	// Concealment over the whole scene: the mean over groups, plus the worst residual, because
	// one legible original word is a failure however good the average is.
	var coveredVals []float64
	for _, g := range groups {
		coveredVals = append(coveredVals, Covered(plates, g, w, h))
	}
	out.Covered = mean(coveredVals)

	out.PixelDiagnostics = map[string]PixelDiagnostic{}
	for _, o := range sc.Observations {
		out.PixelDiagnostics[o.Viewport+"/"+o.StressCase] = observationPixels(caps, o, sc.PlatesFor(o.Viewport, o.StressCase), a, w, h)
	}
	out.Damage = worstPainted(sc.Observations, out.PixelDiagnostics)

	// The scene-level pixel measures describe the primary observation, the unmodified text.
	primary, ok := out.PixelDiagnostics[viewport+"/"+PrimaryStressCase]
	if ok {
		out.Residual = primary.Concealment
		out.BackgroundContrast = primary.BackgroundContrast
		out.Readability = primary.ReplacementReadability
	} else {
		out.Residual = unmeasuredResidual("no observation recorded for the primary viewport")
		out.Readability = unmeasuredReadability("no observation recorded for the primary viewport")
		if caps.Rendered != nil {
			out.BackgroundContrast = BackgroundContrast(caps.Rendered, plates, w, h)
		}
	}

	out.Failures = hardFailures(out)
	if noPlatesOverText(out) {
		loss := DiagnoseLoss(caps.Diag)
		out.Loss = &loss
	}
	return out, nil
}

// observationPixels measures one observation. The normal capture shows what a reader sees; the
// text-hidden one keeps the plates and hides the replacement glyphs, which is what separates new
// text from old lettering and painted support from glyphs.
func observationPixels(caps Captures, o evidence.Observation, plates []evidence.Plate, a *truth.Annotation, w, h int) PixelDiagnostic {
	normal, hidden := caps.open(o.Rendered), caps.open(o.Concealed)
	d := PixelDiagnostic{
		Concealment:            ResidualAcross(caps.Source, hidden, caps.Lettering, a.Groups, w, h),
		ReplacementReadability: ReplacementReadability(normal, hidden, plates, w, h),
		RectangleIntrusion:     RectangleIntrusion(plates, a, w, h),
		PaintedDamage:          PaintedDamage(caps.Source, hidden, plates, a, w, h),
	}
	if normal != nil {
		d.BackgroundContrast = BackgroundContrast(normal, plates, w, h)
	}
	return d
}

// worstPainted is the largest painted protected-pixel count over the observed matrix. Pixels
// proven painted over protected content are a floor, so they stay in the result and still fail
// the scene when some other observation could not be measured; the scene is then marked
// unmeasured so that a run missing captures can never read as clean.
func worstPainted(observations []evidence.Observation, diags map[string]PixelDiagnostic) DamageScore {
	worst := DamageScore{State: StateMeasured}
	measured := 0
	var missing []string
	for _, o := range observations {
		key := o.Viewport + "/" + o.StressCase
		d := diags[key].PaintedDamage
		if !d.IsMeasured() {
			missing = append(missing, key+": "+d.Reason)
			continue
		}
		if measured == 0 || d.ProtectedHit > worst.ProtectedHit {
			worst = d
		}
		measured++
	}
	switch {
	case len(observations) == 0:
		return DamageScore{State: StateUnmeasured, Reason: "no observation recorded"}
	case measured == 0:
		return DamageScore{State: StateUnmeasured, Reason: missing[0]}
	case len(missing) > 0:
		worst.State = StateUnmeasured
		worst.Reason = fmt.Sprintf("%d of %d observation(s) unmeasured - %s", len(missing), len(observations), missing[0])
	}
	return worst
}

// hardFailures lists the strategic spec's zero-tolerance conditions that this scene hit. It
// applies no configurable bound - these four are failures by definition, everywhere, and
// Phase 06's thresholds add to the list rather than soften it.
func hardFailures(s *SceneScore) []string {
	var out []string
	// Not a bound, a fact: the scene has annotated text and the overlay drew nothing over any of
	// it, so a reader sees the original lettering untouched and untranslatable.
	if noPlatesOverText(s) {
		out = append(out, fmt.Sprintf("no plates at all over %d annotated group(s) - the original text is left as-is", s.Detection.FN))
	}
	if s.Damage.ProtectedHit > 0 {
		where := s.Damage.WorstProtectedRegion
		if where == "" {
			where = "protected content"
		}
		out = append(out, fmt.Sprintf("protected-area damage: %d px, worst in %s", s.Damage.ProtectedHit, where))
	}
	if s.Grouping.Merges > 0 {
		out = append(out, fmt.Sprintf("%d plate(s) merged separate reading groups", s.Grouping.Merges))
	}
	for _, name := range sortedKeys(s.Stress) {
		r := s.Stress[name]
		if r.OutOfBounds > 0 {
			out = append(out, fmt.Sprintf("%s: %d plate(s) outside image", name, r.OutOfBounds))
		}
		if r.Clipped > 0 {
			out = append(out, fmt.Sprintf("%s: %d plate(s) clipped", name, r.Clipped))
		}
		if r.CrossGroupOverlap > 0 {
			out = append(out, fmt.Sprintf("%s: %d plate(s) crossed another reading group", name, r.CrossGroupOverlap))
		}
	}
	return out
}

func sortedKeys(m StressBreakdown) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// primaryViewport is the first viewport the run declares that actually has plates, falling back
// to whatever a plate names and finally to the run's first declared viewport - so a scene the
// overlay skipped entirely still has a viewport to be scored at (with no plates, which is the
// point).
func primaryViewport(run *evidence.Run, sc *evidence.Scene) string {
	for _, v := range run.Viewports {
		if len(sc.PlatesFor(v.Name, PrimaryStressCase)) > 0 {
			return v.Name
		}
	}
	for _, p := range sc.Plates {
		if p.StressCase == PrimaryStressCase {
			return p.Viewport
		}
	}
	if len(run.Viewports) > 0 {
		return run.Viewports[0].Name
	}
	return ""
}

// Bucket is an aggregate over a set of scenes.
type Bucket struct {
	Scenes        int     `json:"scenes"`
	MeanRecall    float64 `json:"meanRecall"`
	MeanPrecision float64 `json:"meanPrecision"`
	MeanCER       float64 `json:"meanCer"`
	MeanIoU       float64 `json:"meanIou"`
	WorstIoU      float64 `json:"worstIou"`
	MeanCovered   float64 `json:"meanCovered"`
	WorstResidual float64 `json:"worstResidual"`
	WorstHalo     float64 `json:"worstHalo"`
	// Scenes whose concealment could not be measured, so WorstResidual and WorstHalo are silent
	// about them. A non-zero value here means the two worst-ofs above cover fewer scenes than
	// Scenes says, and must be reported rather than averaged away.
	UnmeasuredConcealment int `json:"unmeasuredConcealment"`
	// Scenes whose painted protected-pixel damage could not be measured for every observation,
	// so ProtectedHitPx is a floor rather than the whole count. Same rule: report, never average.
	UnmeasuredDamage      int     `json:"unmeasuredDamage"`
	MinBackgroundContrast float64 `json:"minBackgroundContrast"`
	Merges                int     `json:"merges"`
	Splits                int     `json:"splits"`
	ProtectedHitPx        int     `json:"protectedHitPx"`
	Clipped               int     `json:"clipped"`
	CrossGroup            int     `json:"crossGroup"`
	WorstDrift            float64 `json:"worstDrift"`
	TotalOcrMs            int64   `json:"totalOcrMs"`
	FailingScenes         int     `json:"failingScenes"`
}

// Summary is the whole run's result, sliced the ways a decision is actually made.
type Summary struct {
	Edition           evidence.Edition            `json:"edition"`
	RunID             string                      `json:"runId"`
	Overall           Bucket                      `json:"overall"`
	ByCategory        map[corpus.Category]*Bucket `json:"byCategory"`
	BySplit           map[corpus.Split]*Bucket    `json:"bySplit"`
	Procedure         string                      `json:"procedure,omitempty"`
	Purpose           string                      `json:"purpose,omitempty"`
	EvidenceIssues    []string                    `json:"evidenceIssues,omitempty"`
	EvidenceDigest    string                      `json:"evidenceDigest,omitempty"`
	ScoresDigest      string                      `json:"scoresDigest,omitempty"`
	InputDigest       string                      `json:"inputDigest,omitempty"`
	DeclarationDigest string                      `json:"declarationDigest,omitempty"`
	ScorerDigest      string                      `json:"scorerDigest,omitempty"`
	// CollectedScenes and ReusedScenes say how the run's evidence was obtained: a reused scene was
	// copied from an earlier complete run (Scene.ReusedFrom) and only scored again here.
	CollectedScenes int       `json:"collectedScenes,omitempty"`
	ReusedScenes    int       `json:"reusedScenes,omitempty"`
	Skipped         []Skipped `json:"skipped"`
}

// Aggregate folds per-scene scores into the summary. Skipped scenes are carried through
// untouched - they are part of the result, not missing from it.
func Aggregate(runID string, edition evidence.Edition, scores []*SceneScore, skipped []Skipped) *Summary {
	sum := &Summary{
		Edition:    edition,
		RunID:      runID,
		ByCategory: map[corpus.Category]*Bucket{},
		BySplit:    map[corpus.Split]*Bucket{},
		Skipped:    skipped,
	}
	if sum.Skipped == nil {
		sum.Skipped = []Skipped{}
	}
	buckets := func(s *SceneScore) []*Bucket {
		out := []*Bucket{&sum.Overall}
		for _, c := range s.Categories {
			if sum.ByCategory[c] == nil {
				sum.ByCategory[c] = &Bucket{}
			}
			out = append(out, sum.ByCategory[c])
		}
		if sum.BySplit[s.Split] == nil {
			sum.BySplit[s.Split] = &Bucket{}
		}
		return append(out, sum.BySplit[s.Split])
	}

	acc := map[*Bucket]*accum{}
	for _, s := range scores {
		for _, b := range buckets(s) {
			if acc[b] == nil {
				acc[b] = &accum{}
			}
			acc[b].add(s, b)
		}
	}
	for b, a := range acc {
		a.finish(b)
	}
	return sum
}

// accum carries the running means a Bucket cannot hold without exposing them.
type accum struct {
	recall, precision, cer, iou, covered []float64
}

func (a *accum) add(s *SceneScore, b *Bucket) {
	b.Scenes++
	if s.Detection.TP+s.Detection.FN > 0 {
		a.recall = append(a.recall, s.Detection.Recall)
	}
	a.precision = append(a.precision, s.Detection.Precision)
	if s.Text.Compared > 0 {
		a.cer = append(a.cer, s.Text.MeanCER)
	}
	if s.Placement.Compared > 0 {
		a.iou = append(a.iou, s.Placement.MeanIoU)
		if b.WorstIoU == 0 || s.Placement.WorstIoU < b.WorstIoU {
			b.WorstIoU = s.Placement.WorstIoU
		}
	}
	a.covered = append(a.covered, s.Covered)

	b.Merges += s.Grouping.Merges
	b.Splits += s.Grouping.Splits
	b.ProtectedHitPx += s.Damage.ProtectedHit
	b.TotalOcrMs += s.Cost.OcrMs
	for _, r := range s.Stress {
		b.Clipped += r.Clipped
		b.CrossGroup += r.CrossGroupOverlap
	}
	// Only a measured scene may move the worst-of. An unmeasured one carries a zero that reads as
	// flawless concealment, which is how a run with no stored render scored better than one that
	// had them - see DEV/research/ocrlab/2026-08-15__extension-parity-run.md.
	if s.Residual.IsMeasured() {
		if s.Residual.Residual > b.WorstResidual {
			b.WorstResidual = s.Residual.Residual
		}
		if s.Residual.Halo > b.WorstHalo {
			b.WorstHalo = s.Residual.Halo
		}
	} else {
		b.UnmeasuredConcealment++
	}
	if s.BackgroundContrast.Plates > 0 && (b.MinBackgroundContrast == 0 || s.BackgroundContrast.MinLuma < b.MinBackgroundContrast) {
		b.MinBackgroundContrast = s.BackgroundContrast.MinLuma
	}
	if !s.Damage.IsMeasured() {
		b.UnmeasuredDamage++
	}
	if s.Placement.Drift > b.WorstDrift {
		b.WorstDrift = s.Placement.Drift
	}
	if len(s.Failures) > 0 {
		b.FailingScenes++
	}
}

func (a *accum) finish(b *Bucket) {
	b.MeanRecall = mean(a.recall)
	b.MeanPrecision = mean(a.precision)
	b.MeanCER = mean(a.cer)
	b.MeanIoU = mean(a.iou)
	b.MeanCovered = mean(a.covered)
}
