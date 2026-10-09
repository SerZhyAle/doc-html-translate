package main

import (
	"flag"
	"fmt"
	"math"
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

func cmdCost(args []string) error {
	f := flag.NewFlagSet("cost", flag.ContinueOnError)
	condition := f.String("condition", "unclassified", "operator-recorded cold, warm or unclassified condition")
	out := f.String("out", "temp/ocrlab/cost.json", "output report")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() < 2 {
		return fmt.Errorf("cost requires at least two run directories")
	}
	if *condition != "cold" && *condition != "warm" && *condition != "unclassified" {
		return fmt.Errorf("invalid condition")
	}
	type series struct {
		Conversion      timing  `json:"conversionRecognition"`
		Render          timing  `json:"browserRendering"`
		MemoryKind      string  `json:"memoryKind"`
		MemorySnapshots []int64 `json:"memorySnapshotsBytes"`
	}
	type samples struct {
		conversion, render []float64
		memory             []int64
		kind               string
	}
	collected := map[string]*samples{}
	seen := map[string]bool{}
	var first *report.Data
	var firstDecl *evidence.Declaration
	environment := ""
	var runIDs []string
	for _, dir := range f.Args() {
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
		runIDs = append(runIDs, data.Run.RunID)
		for _, s := range data.Run.Scenes {
			if s.Unmeasured != "" {
				continue // never ran, so its zero times are not a sample
			}
			if s.ReusedFrom != nil {
				continue // its times belong to the run it was copied from, not to a repeat
			}
			v := collected[s.SceneID]
			if v == nil {
				v = &samples{kind: s.MemoryKind}
				collected[s.SceneID] = v
			}
			if s.MemoryKind != v.kind {
				return fmt.Errorf("memory semantics changed")
			}
			v.conversion = append(v.conversion, float64(s.OcrMs))
			v.render = append(v.render, float64(s.RenderMs))
			v.memory = append(v.memory, s.MemoryBytes)
		}
	}
	rows := map[string]series{}
	for id, v := range collected {
		rows[id] = series{summarizeTiming(v.conversion), summarizeTiming(v.render), v.kind, v.memory}
	}
	result := struct {
		Runs        []string          `json:"runs"`
		Edition     string            `json:"edition"`
		Condition   string            `json:"condition"`
		Environment string            `json:"environment"`
		Limitations string            `json:"limitations"`
		Scenes      map[string]series `json:"scenes"`
	}{runIDs, string(first.Run.Edition), *condition, environment, "Operator-labelled cache condition; no cache reset is inferred. Stage wall times include instrument overhead. Memory observations are snapshots, not peak process-tree RSS. Small repetition counts do not establish a budget.", rows}
	if err := writeJSON(*out, result); err != nil {
		return err
	}
	fmt.Printf("cost: %d repeats, %s edition, %s condition; %s\n", len(runIDs), first.Run.Edition, *condition, *out)
	return nil
}
