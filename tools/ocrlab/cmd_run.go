package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"time"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
	"doc-html-translate/tools/ocrlab/report"
	"doc-html-translate/tools/ocrlab/runner"
	"doc-html-translate/tools/ocrlab/truth"
)

// stringList collects a repeatable flag.
type stringList []string

func (s *stringList) String() string     { return fmt.Sprint(*s) }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	p := addCommonFlags(fs)
	split := fs.String("split", "dev", "which scenes to run: dev, holdout or all")
	out := fs.String("out", "", "run directory (default temp/ocrlab/<timestamp>)")
	lang := fs.String("lang", "eng", "tesseract language")
	var ids stringList
	fs.Var(&ids, "scene", "run only this scene id (repeatable)")
	noScore := fs.Bool("no-score", false, "stop after writing evidence.json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := *out
	if dir == "" {
		dir = filepath.Join("temp", "ocrlab", time.Now().Format("20060102-150405"))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if _, err := runner.Run(runner.Options{
		Manifest: p.manifest,
		Root:     p.root,
		OutDir:   dir,
		Split:    *split,
		SceneIDs: ids,
		Lang:     *lang,
		Log:      os.Stdout,
	}); err != nil {
		return err
	}
	fmt.Printf("\nevidence: %s\n", filepath.Join(dir, runner.EvidenceFile))
	if *noScore {
		return nil
	}
	if err := scoreDir(dir, p); err != nil {
		return err
	}
	return reportDir(dir)
}

// cmdScore grades a saved run. It touches no browser and no recognizer, so a change to the
// measurement can be re-applied to yesterday's evidence and the two results compared - which is
// the only way to tell "the app got better" from "the scoring changed".
func cmdScore(args []string) error {
	fs := flag.NewFlagSet("score", flag.ExitOnError)
	p := addCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: ocrlab score <run-dir>")
	}
	return scoreDir(fs.Arg(0), p)
}

func scoreDir(dir string, p *paths) error {
	run, err := evidence.LoadRun(filepath.Join(dir, runner.EvidenceFile))
	if err != nil {
		return err
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	anns, err := truth.LoadDir(p.annotations)
	if err != nil {
		return err
	}

	var scores []*metrics.SceneScore
	var selfScores []*metrics.SelfScore
	var skipped []metrics.Skipped
	for i := range run.Scenes {
		sc := &run.Scenes[i]
		src := openShot(dir, sc.Screenshots.Source)
		rendered := openShot(dir, sc.Screenshots.Rendered)

		// Every scene that ran gets the annotation-free diagnosis, annotated or not. It costs a
		// pass over two images and it is what makes a freshly harvested batch readable the same
		// afternoon instead of after somebody has drawn a hundred polygons.
		if sc.Error == "" {
			selfScores = append(selfScores, metrics.SelfDiagnose(run, sc, src, rendered))
		}

		scene := m.Find(sc.SceneID)
		if scene == nil {
			skipped = append(skipped, metrics.Skipped{SceneID: sc.SceneID, Reason: "not in the manifest"})
			continue
		}
		ann := anns[sc.SceneID]
		if ann == nil {
			skipped = append(skipped, metrics.Skipped{SceneID: sc.SceneID, Reason: "no annotation - self-diagnosis only"})
			continue
		}
		score, err := metrics.Score(run, sc, ann, scene, src, rendered)
		if err != nil {
			skipped = append(skipped, metrics.Skipped{SceneID: sc.SceneID, Reason: err.Error()})
			continue
		}
		scores = append(scores, score)
	}

	summary := metrics.Aggregate(run.RunID, run.Edition, scores, skipped)
	selfSummary := metrics.SummarizeSelf(selfScores)
	if err := writeJSON(filepath.Join(dir, runner.ScoresFile), scores); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, runner.SummaryFile), summary); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, runner.SelfFile), selfScores); err != nil {
		return err
	}

	// The self-diagnosis first, because on a fresh harvest it is the whole picture.
	fmt.Printf("\n== what the run looks like, no ground truth needed (%d scene(s)) ==\n", selfSummary.Scenes)
	fmt.Printf("scenes with no plates at all : %d\n", selfSummary.ScenesWithNoPlate)
	fmt.Printf("scenes with a finding        : %d\n", selfSummary.ScenesWithFinding)
	if selfSummary.MeanResidual > 0 || selfSummary.WorstResidualID != "" {
		fmt.Printf("original still visible under the plates: mean %.0f%%, worst %.0f%% (%s)\n",
			selfSummary.MeanResidual*100, selfSummary.WorstResidual*100, selfSummary.WorstResidualID)
		fmt.Printf("strokes carrying on past the plate edges: mean %.0f%% of the covered ink\n",
			selfSummary.MeanCutGlyphInk*100)
	}
	for _, k := range sortedCounts(selfSummary.FindingCounts) {
		fmt.Printf("  %-56s %d scene(s)\n", k, selfSummary.FindingCounts[k])
	}

	fmt.Printf("\n== against ground truth ==\nscored %d scene(s), %d without an annotation\n", len(scores), len(skipped))
	var failing int
	for _, s := range scores {
		if len(s.Failures) > 0 {
			failing++
			fmt.Printf("  FAIL %-40s %s\n", s.SceneID, s.Failures[0])
			for _, f := range s.Failures[1:] {
				fmt.Printf("       %-40s %s\n", "", f)
			}
		}
	}
	fmt.Printf("scenes with a hard failure: %d of %d\n", failing, len(scores))
	return nil
}

// sortedCounts orders finding classes by how many scenes hit them, worst first - the order a
// person fixing the program wants to read.
func sortedCounts(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// openShot decodes a screenshot if it is there. A missing one is not an error: the geometric
// dimensions still score and the pixel ones report a zero sample, which is what lets a scoring
// pass run over an evidence file whose images have been cleaned up.
func openShot(dir, rel string) image.Image {
	if rel == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func cmdReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: ocrlab report <run-dir>")
	}
	return reportDir(fs.Arg(0))
}

func reportDir(dir string) error {
	d, err := report.LoadData(dir)
	if err != nil {
		return err
	}
	if err := report.WriteMarkdown(dir, d); err != nil {
		return err
	}
	if err := report.WriteHTML(dir, d); err != nil {
		return err
	}
	fmt.Printf("report:   %s\n", filepath.Join(dir, "report.md"))
	fmt.Printf("          %s\n", filepath.Join(dir, "report.html"))
	return nil
}

// cmdGate is the acceptance decision, and the only subcommand whose exit code means "this change
// may not be accepted". It reads a scored run and the thresholds file; -against points at an
// earlier run directory, whose summary becomes the non-regression reference.
func cmdGate(args []string) error {
	fs := flag.NewFlagSet("gate", flag.ExitOnError)
	thresholds := fs.String("thresholds", filepath.Join("DEV", "ocrlab", "thresholds.json"), "acceptance bounds")
	against := fs.String("against", "", "run directory of the last accepted run, for the holdout non-regression check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: ocrlab gate [-thresholds <path>] [-against <run-dir>] <run-dir>")
	}
	dir := fs.Arg(0)

	th, err := report.LoadThresholds(*thresholds)
	if err != nil {
		return err
	}
	var summary metrics.Summary
	if err := readJSON(filepath.Join(dir, runner.SummaryFile), &summary); err != nil {
		return fmt.Errorf("%s: %w (run `ocrlab score` first)", dir, err)
	}
	var prev *metrics.Summary
	if *against != "" {
		var p metrics.Summary
		if err := readJSON(filepath.Join(*against, runner.SummaryFile), &p); err != nil {
			return fmt.Errorf("-against %s: %w", *against, err)
		}
		prev = &p
	}

	res := report.Gate(&summary, th, prev)
	fmt.Print(res.Render())
	if err := writeJSON(filepath.Join(dir, "gate.json"), res); err != nil {
		return err
	}
	if !res.Pass {
		os.Exit(1)
	}
	return nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(buf, '\n'), 0o644)
}
