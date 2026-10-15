package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/report"
)

type timing struct {
	Samples int     `json:"samples"`
	Median  float64 `json:"medianMs"`
	P95     float64 `json:"p95Ms"`
	Min     float64 `json:"minMs"`
	Max     float64 `json:"maxMs"`
	StdDev  float64 `json:"stdDevMs"`
}

func summarizeTiming(samples []float64) timing {
	if len(samples) == 0 {
		return timing{}
	}
	values := append([]float64(nil), samples...)
	sort.Float64s(values)
	n := len(values)
	mid := values[n/2]
	if n%2 == 0 {
		mid = (values[n/2-1] + values[n/2]) / 2
	}
	var mean, variance float64
	for _, v := range values {
		mean += v
	}
	mean /= float64(n)
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	return timing{Samples: n, Median: mid, P95: values[int(math.Ceil(.95*float64(n)))-1], Min: values[0], Max: values[n-1], StdDev: math.Sqrt(variance / float64(n))}
}

// validCondition is the closed condition vocabulary. The tool does not reset caches, so a
// condition is an operator claim; the vocabulary at least keeps the claim spellable in only the
// three honest ways: cold, warm, or unclassified.
func validCondition(c string) bool { return c == "cold" || c == "warm" || c == "unclassified" }

// resolveConditions assigns one condition per run directory. A -conditions file maps a run id
// (the directory's base name) to its condition, so a cold first repetition and warm repeats can
// be summarized in one command without mixing their samples; runs absent from the file fall back
// to the global -condition. Duplicate and unknown-valued entries are errors rather than
// silently degrading to unclassified.
func resolveConditions(dirs []string, file, fallback string) ([]string, error) {
	perRun := map[string]string{}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read conditions: %w", err)
		}
		if err := json.Unmarshal(data, &perRun); err != nil {
			return nil, fmt.Errorf("parse conditions %s: %w", file, err)
		}
		for id, c := range perRun {
			if !validCondition(c) {
				return nil, fmt.Errorf("conditions %s: run %s has condition %q, want cold, warm or unclassified", file, id, c)
			}
		}
	}
	if !validCondition(fallback) {
		return nil, fmt.Errorf("invalid condition %q", fallback)
	}
	out := make([]string, len(dirs))
	for i, dir := range dirs {
		c, ok := perRun[filepath.Base(dir)]
		if !ok {
			c, ok = perRun[dir]
		}
		if !ok {
			c = fallback
		}
		out[i] = c
	}
	return out, nil
}

type costSeries struct {
	Conversion      timing  `json:"conversionRecognition"`
	Render          timing  `json:"browserRendering"`
	MemoryKind      string  `json:"memoryKind"`
	MemorySnapshots []int64 `json:"memorySnapshotsBytes"`
}

type costSamples struct {
	conversion, render []float64
	memory             []int64
	kind               string
}

// costCondition is the per-condition slice of one cost report.
type costCondition struct {
	Runs   []string               `json:"runs"`
	Scenes map[string]*costSeries `json:"scenes"`
}

func cmdCost(args []string) error {
	f := flag.NewFlagSet("cost", flag.ContinueOnError)
	condition := f.String("condition", "unclassified", "condition of every run not named by -conditions (cold, warm or unclassified)")
	conditionsFile := f.String("conditions", "", "JSON file mapping run id to condition (cold, warm, unclassified)")
	out := f.String("out", "temp/ocrlab/cost.json", "output report")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() < 2 {
		return fmt.Errorf("cost requires at least two run directories")
	}
	runConditions, err := resolveConditions(f.Args(), *conditionsFile, *condition)
	if err != nil {
		return err
	}

	// Samples are grouped per scene AND per condition, so a cold repetition never inflates a
	// warm median with a first-launch outlier the procedure did not intend to measure.
	collected := map[string]map[string]*costSamples{}
	seen := map[string]bool{}
	var first *report.Data
	var firstDecl *evidence.Declaration
	environment := ""
	var runIDs []string
	var conditionRuns []struct{ cond, id string }
	for i, dir := range f.Args() {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		if seen[abs] {
			return fmt.Errorf("same run supplied twice")
		}
		seen[abs] = true
		data, err := report.LoadData(dir)
		if err != nil {
			return err
		}
		decl, err := evidence.LoadDeclaration(dir)
		if err != nil {
			return err
		}
		if first == nil {
			first, firstDecl, environment = data, decl, decl.Environment
		} else {
			c := report.Compare(first, data, firstDecl, decl)
			if !c.Compatible || decl.Environment != environment || decl.SourceDigest != firstDecl.SourceDigest || data.Run.Engine != first.Run.Engine {
				return fmt.Errorf("cost runs have incompatible inputs, engine, environment or evidence: %v", c.Limitations)
			}
		}
		if len(data.Summary.EvidenceIssues) > 0 {
			return fmt.Errorf("cost run has unavailable evidence: %v", data.Summary.EvidenceIssues)
		}
		cond := runConditions[i]
		runIDs = append(runIDs, data.Run.RunID)
		conditionRuns = append(conditionRuns, struct{ cond, id string }{cond, data.Run.RunID})
		for _, s := range data.Run.Scenes {
			if s.Unmeasured != "" {
				continue // never ran, so its zero times are not a sample
			}
			if s.ReusedFrom != nil {
				continue // its times belong to the run it was copied from, not to a repeat
			}
			perScene := collected[cond]
			if perScene == nil {
				perScene = map[string]*costSamples{}
				collected[cond] = perScene
			}
			v := perScene[s.SceneID]
			if v == nil {
				v = &costSamples{kind: s.MemoryKind}
				perScene[s.SceneID] = v
			}
			if s.MemoryKind != v.kind {
				return fmt.Errorf("memory semantics changed")
			}
			v.conversion = append(v.conversion, float64(s.OcrMs))
			v.render = append(v.render, float64(s.RenderMs))
			v.memory = append(v.memory, s.MemoryBytes)
		}
	}
	conditions := map[string]*costCondition{}
	for cond, perScene := range collected {
		c := &costCondition{Scenes: map[string]*costSeries{}}
		for id, v := range perScene {
			c.Scenes[id] = &costSeries{summarizeTiming(v.conversion), summarizeTiming(v.render), v.kind, v.memory}
		}
		conditions[cond] = c
	}
	for _, r := range conditionRuns {
		if c := conditions[r.cond]; c != nil {
			c.Runs = append(c.Runs, r.id)
		}
	}
	for _, c := range conditions {
		sort.Strings(c.Runs)
	}
	result := struct {
		Runs        []string                  `json:"runs"`
		Edition     string                    `json:"edition"`
		Environment string                    `json:"environment"`
		Conditions  map[string]*costCondition `json:"conditions"`
		Limitations string                    `json:"limitations"`
	}{runIDs, string(first.Run.Edition), environment, conditions,
		"Operator-labelled cache condition; no cache reset is inferred. Every run uses its own fresh browser profile, so browser caches are cold in every repetition; cold vs warm separates first-session from immediately repeated state and claims no reboot-level cold. Stage wall times include instrument overhead. Memory observations are snapshots, not peak process-tree RSS. Small repetition counts do not establish a budget."}
	if err := writeJSON(*out, result); err != nil {
		return err
	}
	var order []string
	for cond := range conditions {
		order = append(order, cond)
	}
	sort.Strings(order)
	parts := make([]string, 0, len(order))
	for _, cond := range order {
		parts = append(parts, fmt.Sprintf("%s %d run(s)", cond, len(conditions[cond].Runs)))
	}
	fmt.Printf("cost: %d repeats, %s edition (%s); %s\n", len(runIDs), first.Run.Edition, joinCSV(parts), *out)
	return nil
}

func joinCSV(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
