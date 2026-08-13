package report

// The acceptance gate: the one place in the lab where a number decides something.
//
// The measurement packages stay threshold-free on purpose. `metrics` answers "what happened",
// this file answers "is that acceptable", and keeping the two apart is what lets a scoring change
// be re-applied to yesterday's evidence and compared - the only way to tell "the app got better"
// from "the bar moved". Every bound arrives from DEV/ocrlab/thresholds.json and carries the
// baseline value it was derived from, so a bound that was quietly retrofitted to make a preferred
// change pass is visible in the diff rather than only in the verdict.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/metrics"
)

// ThresholdsSchemaVersion guards a file written against different meanings from being read as if
// it agreed with this build.
const ThresholdsSchemaVersion = 1

// Bound is one acceptance limit and the evidence it came from.
//
// Baseline is not decoration. A bound with no recorded baseline is a bound somebody chose, and the
// strategic spec forbids exactly that; Validate refuses the file when a derived bound lacks one.
type Bound struct {
	Max      *float64 `json:"max,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Baseline *float64 `json:"baseline,omitempty"`
	Note     string   `json:"note,omitempty"`
}

// HardGates are the strategic spec's zero-tolerance conditions. They are counts, they are fixed at
// zero, and no baseline can raise them - they are in the file so a reader sees the whole contract
// in one place, not so they can be tuned.
type HardGates struct {
	ProtectedDamagePx  int `json:"protectedDamagePx"`
	Merges             int `json:"merges"`
	ClippedPlates      int `json:"clippedPlates"`
	CrossGroupOverlaps int `json:"crossGroupOverlaps"`
}

// Dimension is the per-category acceptance for one of the strategic table's eight dimensions.
type Dimension struct {
	Overall    Bound            `json:"overall"`
	ByCategory map[string]Bound `json:"byCategory,omitempty"`
	Tolerance  *float64         `json:"regressionTolerance,omitempty"`
}

// Thresholds is DEV/ocrlab/thresholds.json.
type Thresholds struct {
	SchemaVersion int    `json:"schemaVersion"`
	DerivedFrom   string `json:"derivedFrom"` // the run id the bounds were read off
	DerivedOn     string `json:"derivedOn"`   // YYYY-MM-DD
	Caveat        string `json:"caveat"`      // what the corpus could not yet support

	Hard       HardGates            `json:"hard"`
	Dimensions map[string]Dimension `json:"dimensions"`
}

// The dimension keys, one per row of strategic section 3.2. Named constants because the gate, the
// file and the report must agree on the spelling, and a typo would silently drop a whole row of
// the contract rather than fail.
const (
	DimRecognition = "recognition"
	DimGrouping    = "grouping"
	DimPosition    = "position"
	DimConcealment = "concealment"
	DimDamage      = "damage"
	DimReplacement = "replacement"
	DimReview      = "review"
	DimCost        = "cost"
)

// Dimensions lists the eight in table order.
func Dimensions() []string {
	return []string{
		DimRecognition, DimGrouping, DimPosition, DimConcealment,
		DimDamage, DimReplacement, DimReview, DimCost,
	}
}

// LoadThresholds reads and validates the acceptance file.
func LoadThresholds(path string) (*Thresholds, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var th Thresholds
	if err := json.Unmarshal(data, &th); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if th.SchemaVersion != ThresholdsSchemaVersion {
		return nil, fmt.Errorf("%s: schemaVersion %d, this build understands %d",
			path, th.SchemaVersion, ThresholdsSchemaVersion)
	}
	if problems := th.Validate(); len(problems) > 0 {
		return nil, fmt.Errorf("%s:\n  %s", path, strings.Join(problems, "\n  "))
	}
	return &th, nil
}

// Validate reports what is wrong with the file itself, before it judges anything.
func (t *Thresholds) Validate() []string {
	var out []string
	if t.DerivedFrom == "" {
		out = append(out, "derivedFrom is empty - a bound with no run behind it is a bound somebody chose")
	}
	for _, name := range Dimensions() {
		d, ok := t.Dimensions[name]
		if !ok {
			out = append(out, fmt.Sprintf("dimension %q is missing", name))
			continue
		}
		for label, b := range allBounds(d) {
			if b.Max == nil && b.Min == nil {
				out = append(out, fmt.Sprintf("%s/%s sets neither max nor min", name, label))
			}
			if b.Baseline == nil && b.Note == "" {
				out = append(out, fmt.Sprintf("%s/%s has no baseline and no note saying why", name, label))
			}
		}
	}
	for label, v := range map[string]int{
		"protectedDamagePx": t.Hard.ProtectedDamagePx, "merges": t.Hard.Merges,
		"clippedPlates": t.Hard.ClippedPlates, "crossGroupOverlaps": t.Hard.CrossGroupOverlaps,
	} {
		if v != 0 {
			out = append(out, fmt.Sprintf("hard gate %s is %d; the strategic spec fixes all four at 0", label, v))
		}
	}
	sort.Strings(out)
	return out
}

func allBounds(d Dimension) map[string]Bound {
	out := map[string]Bound{"overall": d.Overall}
	for k, v := range d.ByCategory {
		out[k] = v
	}
	return out
}

// Check is one verdict line.
type Check struct {
	Dimension string  `json:"dimension"`
	Scope     string  `json:"scope"` // "overall" or a category name
	Measure   string  `json:"measure"`
	Value     float64 `json:"value"`
	Limit     float64 `json:"limit"`
	Pass      bool    `json:"pass"`
	Detail    string  `json:"detail,omitempty"`
}

// GateResult is the whole verdict, kept as data so the report and the exit code read the same
// thing rather than two renderings of it.
type GateResult struct {
	RunID   string  `json:"runId"`
	Edition string  `json:"edition"`
	Pass    bool    `json:"pass"`
	Checks  []Check `json:"checks"`
}

// Gate judges a summary. prev may be nil - the first run has nothing to regress against, and
// saying so is better than inventing a comparison.
func Gate(sum *metrics.Summary, th *Thresholds, prev *metrics.Summary) GateResult {
	res := GateResult{RunID: sum.RunID, Edition: string(sum.Edition), Pass: true}
	add := func(c Check) {
		if !c.Pass {
			res.Pass = false
		}
		res.Checks = append(res.Checks, c)
	}

	// The four hard gates first: they are counts, they are fixed at zero, and a table of means
	// underneath them must never be what a reader sees first.
	for _, h := range []struct {
		measure string
		got     int
		limit   int
	}{
		{"protected-area damage (px)", sum.Overall.ProtectedHitPx, th.Hard.ProtectedDamagePx},
		{"reading groups merged", sum.Overall.Merges, th.Hard.Merges},
		{"clipped plates after stress", sum.Overall.Clipped, th.Hard.ClippedPlates},
		{"plates crossing another group", sum.Overall.CrossGroup, th.Hard.CrossGroupOverlaps},
	} {
		add(Check{
			Dimension: "hard", Scope: "overall", Measure: h.measure,
			Value: float64(h.got), Limit: float64(h.limit), Pass: h.got <= h.limit,
		})
	}

	for _, name := range Dimensions() {
		d, ok := th.Dimensions[name]
		if !ok {
			continue
		}
		add(bucketCheck(name, "overall", d.Overall, &sum.Overall))
		for _, cat := range corpus.Categories() {
			b, ok := d.ByCategory[string(cat)]
			if !ok {
				continue
			}
			bucket := sum.ByCategory[cat]
			if bucket == nil || bucket.Scenes == 0 {
				add(Check{
					Dimension: name, Scope: string(cat), Measure: measureOf(name),
					Pass: true, Detail: "no scored scene in this category - not a pass, an absence",
				})
				continue
			}
			add(bucketCheck(name, string(cat), b, bucket))
		}
		if d.Tolerance != nil && prev != nil {
			add(regressionCheck(name, *d.Tolerance, sum, prev))
		}
	}
	return res
}

// measureOf names the one number each dimension is judged on. One number per dimension, chosen so
// the direction is unambiguous: every bound below is "must not exceed" except the two the file
// spells with min.
func measureOf(dimension string) string {
	switch dimension {
	case DimRecognition:
		return "mean recall"
	case DimGrouping:
		return "merges + splits"
	case DimPosition:
		return "worst drift (px)"
	case DimConcealment:
		return "worst residual ink"
	case DimDamage:
		return "protected-area damage (px)"
	case DimReplacement:
		return "clipped + cross-group"
	case DimReview:
		return "scenes with a named failure"
	case DimCost:
		return "total OCR time (ms)"
	}
	return dimension
}

func valueOf(dimension string, b *metrics.Bucket) float64 {
	switch dimension {
	case DimRecognition:
		return b.MeanRecall
	case DimGrouping:
		return float64(b.Merges + b.Splits)
	case DimPosition:
		return b.WorstDrift
	case DimConcealment:
		return b.WorstResidual
	case DimDamage:
		return float64(b.ProtectedHitPx)
	case DimReplacement:
		return float64(b.Clipped + b.CrossGroup)
	case DimReview:
		return float64(b.FailingScenes)
	case DimCost:
		return float64(b.TotalOcrMs)
	}
	return 0
}

func bucketCheck(dimension, scope string, b Bound, bucket *metrics.Bucket) Check {
	c := Check{Dimension: dimension, Scope: scope, Measure: measureOf(dimension), Pass: true}
	c.Value = valueOf(dimension, bucket)
	switch {
	case b.Min != nil:
		c.Limit = *b.Min
		c.Pass = c.Value >= *b.Min
		c.Detail = "must not fall below"
	case b.Max != nil:
		c.Limit = *b.Max
		c.Pass = c.Value <= *b.Max
		c.Detail = "must not exceed"
	}
	return c
}

// regressionCheck compares this run's holdout with the last accepted one. The holdout is the only
// split it looks at: the dev split is what tuning is allowed to move, and comparing it would turn
// an intended improvement into a failure.
func regressionCheck(dimension string, tolerance float64, sum, prev *metrics.Summary) Check {
	c := Check{Dimension: dimension, Scope: "holdout", Measure: measureOf(dimension) + " vs the last accepted run", Pass: true}
	now, before := sum.BySplit[corpus.SplitHoldout], prev.BySplit[corpus.SplitHoldout]
	if now == nil || before == nil || now.Scenes == 0 || before.Scenes == 0 {
		c.Detail = "no holdout scenes on one side - nothing to compare"
		return c
	}
	nowV, beforeV := valueOf(dimension, now), valueOf(dimension, before)
	c.Value, c.Limit = nowV, beforeV
	// Recognition is the one dimension where higher is better, so a regression is a fall.
	if dimension == DimRecognition {
		c.Pass = nowV >= beforeV-tolerance
		c.Detail = fmt.Sprintf("may not fall more than %g below %g", tolerance, beforeV)
		return c
	}
	c.Pass = nowV <= beforeV+tolerance
	c.Detail = fmt.Sprintf("may not rise more than %g above %g", tolerance, beforeV)
	return c
}

// Render writes the verdict the way it must be read: the failures first, then everything, then one
// line that is the exit code in words.
func (r GateResult) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "gate: %s (%s)\n\n", r.RunID, r.Edition)
	for _, want := range []bool{false, true} {
		for _, c := range r.Checks {
			if c.Pass != want {
				continue
			}
			verdict := "PASS"
			if !c.Pass {
				verdict = "FAIL"
			}
			fmt.Fprintf(&b, "  %s %-12s %-10s %-34s %10.4g", verdict, c.Dimension, c.Scope, c.Measure, c.Value)
			if c.Detail != "" && c.Limit == 0 && c.Value == 0 {
				fmt.Fprintf(&b, "   %s", c.Detail)
			} else {
				fmt.Fprintf(&b, "   limit %.4g", c.Limit)
			}
			b.WriteString("\n")
		}
	}
	if r.Pass {
		b.WriteString("\nPASS\n")
	} else {
		b.WriteString("\nFAIL\n")
	}
	return b.String()
}
