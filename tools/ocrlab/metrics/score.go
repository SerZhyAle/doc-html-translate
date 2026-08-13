package metrics

import (
	"errors"
	"fmt"
	"image"
	"sort"

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
	OcrMs        int64 `json:"ocrMs"`
	RenderMs     int64 `json:"renderMs"`
	PeakRSSBytes int64 `json:"peakRssBytes"`
}

// SceneScore is one scene measured across every dimension of the strategic table.
type SceneScore struct {
	SceneID    string            `json:"sceneId"`
	Edition    evidence.Edition  `json:"edition"`
	Categories []corpus.Category `json:"categories"`
	Split      corpus.Split      `json:"split"`
	Viewport   string            `json:"viewport"`

	Detection   DetectionScore   `json:"detection"`
	Text        TextScore        `json:"text"`
	Grouping    GroupingScore    `json:"grouping"`
	Placement   PlacementScore   `json:"placement"`
	Covered     float64          `json:"covered"`
	Residual    ResidualScore    `json:"residual"`
	Contrast    ContrastScore    `json:"contrast"`
	Damage      DamageScore      `json:"damage"`
	Replacement ReplacementScore `json:"replacement"`
	Stress      StressBreakdown  `json:"stress"`
	Cost        CostScore        `json:"cost"`

	// Failures names, in words, every dimension that failed in a way the strategic spec calls
	// hard. The report shows this string; an aggregate can never make it disappear.
	Failures []string `json:"failures,omitempty"`
}

// Skipped is a scene that could not be scored and why.
type Skipped struct {
	SceneID string `json:"sceneId"`
	Reason  string `json:"reason"`
}

// Score measures one scene at the primary viewport, with drift taken across all viewports.
//
// source and rendered may be nil - the geometric dimensions still score, and the pixel-level
// ones report zero with the sample size that says so. That is what lets `ocrlab score` re-run
// offline over an old evidence file whose screenshots have been cleaned up.
func Score(
	run *evidence.Run,
	sc *evidence.Scene,
	a *truth.Annotation,
	s *corpus.Scene,
	source, rendered image.Image,
) (*SceneScore, error) {
	if !a.IsTruth() {
		return nil, fmt.Errorf("%w: %s", ErrNotTruth, a.NotTruthReason())
	}
	if sc.Error != "" {
		return nil, fmt.Errorf("scene errored during the run: %s", sc.Error)
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
		Cost:       CostScore{OcrMs: sc.OcrMs, RenderMs: sc.RenderMs, PeakRSSBytes: sc.PeakRSSBytes},
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

	out.Damage = Damage(plates, a, w, h)
	out.Replacement = Replacement(plates, groups, matches, w, h)
	out.Stress = ReplacementByStress(sc, groups, viewport, w, h)

	// Concealment over the whole scene: the mean over groups, plus the worst residual, because
	// one legible original word is a failure however good the average is.
	var coveredVals []float64
	for _, g := range groups {
		coveredVals = append(coveredVals, Covered(plates, g, w, h))
	}
	out.Covered = mean(coveredVals)

	if source != nil && rendered != nil {
		for _, g := range groups {
			r := ResidualInk(source, rendered, g, w, h)
			out.Residual.InkPx += r.InkPx
			if r.Residual > out.Residual.Residual {
				out.Residual.Residual = r.Residual
			}
			if r.Halo > out.Residual.Halo {
				out.Residual.Halo = r.Halo
			}
		}
		out.Contrast = RenderedContrast(rendered, plates, w, h)
	}

	out.Failures = hardFailures(out)
	return out, nil
}

// hardFailures lists the strategic spec's zero-tolerance conditions that this scene hit. It
// applies no configurable bound - these four are failures by definition, everywhere, and
// Phase 06's thresholds add to the list rather than soften it.
func hardFailures(s *SceneScore) []string {
	var out []string
	// Not a bound, a fact: the scene has annotated text and the overlay drew nothing over any of
	// it, so a reader sees the original lettering untouched and untranslatable.
	if s.Replacement.Plates == 0 && s.Detection.FN > 0 {
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
	Scenes         int     `json:"scenes"`
	MeanRecall     float64 `json:"meanRecall"`
	MeanPrecision  float64 `json:"meanPrecision"`
	MeanCER        float64 `json:"meanCer"`
	MeanIoU        float64 `json:"meanIou"`
	WorstIoU       float64 `json:"worstIou"`
	MeanCovered    float64 `json:"meanCovered"`
	WorstResidual  float64 `json:"worstResidual"`
	WorstHalo      float64 `json:"worstHalo"`
	MinContrast    float64 `json:"minContrast"`
	Merges         int     `json:"merges"`
	Splits         int     `json:"splits"`
	ProtectedHitPx int     `json:"protectedHitPx"`
	Clipped        int     `json:"clipped"`
	CrossGroup     int     `json:"crossGroup"`
	WorstDrift     float64 `json:"worstDrift"`
	TotalOcrMs     int64   `json:"totalOcrMs"`
	FailingScenes  int     `json:"failingScenes"`
}

// Summary is the whole run's result, sliced the ways a decision is actually made.
type Summary struct {
	Edition    evidence.Edition            `json:"edition"`
	RunID      string                      `json:"runId"`
	Overall    Bucket                      `json:"overall"`
	ByCategory map[corpus.Category]*Bucket `json:"byCategory"`
	BySplit    map[corpus.Split]*Bucket    `json:"bySplit"`
	Skipped    []Skipped                   `json:"skipped"`
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
	a.recall = append(a.recall, s.Detection.Recall)
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
	if s.Residual.Residual > b.WorstResidual {
		b.WorstResidual = s.Residual.Residual
	}
	if s.Residual.Halo > b.WorstHalo {
		b.WorstHalo = s.Residual.Halo
	}
	if s.Contrast.Plates > 0 && (b.MinContrast == 0 || s.Contrast.MinLuma < b.MinContrast) {
		b.MinContrast = s.Contrast.MinLuma
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
