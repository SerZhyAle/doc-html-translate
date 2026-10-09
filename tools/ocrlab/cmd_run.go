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
	lang := fs.String("lang", "", "tesseract language for every scene (default: each scene's declared language, else eng)")
	var ids stringList
	fs.Var(&ids, "scene", "run only this scene id (repeatable)")
	purpose := fs.String("purpose", "exploratory", "exploratory, selected-dev or full-benchmark")
	noScore := fs.Bool("no-score", false, "stop after writing evidence.json")
	fresh := fs.Bool("fresh", false, "collect every scene even when an earlier complete run with identical inputs could be reused")
	reuseFrom := fs.String("reuse-from", "", "search only this run directory (or folder of runs) for reusable scenes (default temp/ocrlab)")
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
		Manifest:    p.manifest,
		Annotations: p.annotations, Purpose: *purpose,
		Root:     p.root,
		OutDir:   dir,
		Split:    *split,
		SceneIDs: ids,
		Lang:     *lang,
		Log:      os.Stdout,
		Fresh:    *fresh, ReuseFrom: *reuseFrom,
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
	p, decl, err := snapshotInputs(dir, p)
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

	// A sidecar that cannot be read costs the loss-point diagnosis, not the scoring.
	sidecar, err := evidence.LoadSidecar(filepath.Join(dir, runner.DiagFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ocrlab: loss points unavailable: %v\n", err)
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
		if sc.Error == "" && sc.Unmeasured == "" {
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
		lettering, err := truth.LoadLettering(p.annotations, sc.SceneID, ann.ImageWidth, ann.ImageHeight)
		if err != nil {
			skipped = append(skipped, metrics.Skipped{SceneID: sc.SceneID, Reason: err.Error()})
			continue
		}
		score, err := metrics.Score(run, sc, ann, scene, metrics.Captures{
			Source: src, Rendered: rendered, Lettering: lettering,
			Open: func(rel string) image.Image { return openShot(dir, rel) },
			Diag: sidecar[sc.SceneID],
		})
		if err != nil {
			skipped = append(skipped, metrics.Skipped{SceneID: sc.SceneID, Reason: err.Error()})
			continue
		}
		scores = append(scores, score)
	}

	summary := metrics.Aggregate(run.RunID, run.Edition, scores, skipped)
	for _, sc := range run.Scenes {
		if sc.ReusedFrom != nil {
			summary.ReusedScenes++
		} else {
			summary.CollectedScenes++
		}
	}
	summary.EvidenceIssues = evidence.Issues(dir, run, decl)
	if decl != nil {
		summary.ScorerDigest, err = evidence.ScorerDigest(".")
		if err != nil {
			return err
		}
		data, _ := os.ReadFile(filepath.Join(dir, "declaration.json"))
		summary.DeclarationDigest = evidence.Digest(data)
		summary.Procedure = evidence.Procedure
		summary.Purpose = decl.Purpose
		buf, _ := json.Marshal(struct {
			Inputs    map[string]evidence.Input
			Viewports []evidence.Viewport
			Settings  map[string]string
			Browser   evidence.Browser
			Stresses  []string
			Scorer    string
		}{decl.Scenes, decl.Viewports, decl.Settings, run.Browser, decl.StressCases, summary.ScorerDigest})
		summary.InputDigest = evidence.Digest(buf)
		summary.EvidenceIssues = append(summary.EvidenceIssues, evidence.SnapshotIssues(dir, run, decl)...)
	}
	if err := preserveScores(dir); err != nil {
		return err
	}
	selfSummary := metrics.SummarizeSelf(selfScores)
	if err := writeJSON(filepath.Join(dir, runner.ScoresFile), scores); err != nil {
		return err
	}
	evBytes, _ := os.ReadFile(filepath.Join(dir, runner.EvidenceFile))
	scoreBytes, _ := os.ReadFile(filepath.Join(dir, runner.ScoresFile))
	summary.EvidenceDigest = evidence.Digest(evBytes)
	summary.ScoresDigest = evidence.Digest(scoreBytes)
	if err := writeJSON(filepath.Join(dir, runner.SummaryFile), summary); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, runner.SelfFile), selfScores); err != nil {
		return err
	}

	fmt.Printf("\nevidence: %d scene(s) collected in this run, %d reused from earlier runs (reusedFrom)\n", summary.CollectedScenes, summary.ReusedScenes)

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
			if s.Loss != nil {
				fmt.Printf("       %-40s loss point: %s\n", "", s.Loss)
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
	path, err := evidence.SafePath(dir, rel)
	if err != nil {
		return nil
	}
	f, err := os.Open(path)
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
		return cannotVerify(errors.New("usage: ocrlab gate [-thresholds <path>] [-against <run-dir>] <run-dir>"))
	}
	dir := fs.Arg(0)

	th, err := report.LoadThresholds(*thresholds)
	if err != nil {
		return cannotVerify(err)
	}
	var summary metrics.Summary
	if err := readJSON(filepath.Join(dir, runner.SummaryFile), &summary); err != nil {
		return cannotVerify(fmt.Errorf("%s: %w (run `ocrlab score` first)", dir, err))
	}
	var prev *metrics.Summary
	if *against != "" {
		var p metrics.Summary
		if err := readJSON(filepath.Join(*against, runner.SummaryFile), &p); err != nil {
			summary.EvidenceIssues = append(summary.EvidenceIssues, fmt.Sprintf("reference run unavailable: %v", err))
		}
		if p.RunID != "" {
			prev = &p
		}
		if data, err := report.LoadData(*against); err != nil || len(data.Summary.EvidenceIssues) > 0 {
			summary.EvidenceIssues = append(summary.EvidenceIssues, "reference run lacks complete unchanged evidence")
			prev = nil
		}
	}

	run, loadErr := evidence.LoadRun(filepath.Join(dir, runner.EvidenceFile))
	decl, _ := evidence.LoadDeclaration(dir)
	declarationBytes, _ := os.ReadFile(filepath.Join(dir, "declaration.json"))
	if decl != nil && summary.DeclarationDigest != evidence.Digest(declarationBytes) {
		summary.EvidenceIssues = append(summary.EvidenceIssues, "declaration changed since scoring")
	}
	if loadErr != nil {
		summary.EvidenceIssues = append(summary.EvidenceIssues, "evidence unavailable: "+loadErr.Error())
	} else {
		summary.EvidenceIssues = append(summary.EvidenceIssues, evidence.Issues(dir, run, decl)...)
		summary.EvidenceIssues = append(summary.EvidenceIssues, evidence.SnapshotIssues(dir, run, decl)...)
	}
	evBytes, _ := os.ReadFile(filepath.Join(dir, runner.EvidenceFile))
	scoreBytes, _ := os.ReadFile(filepath.Join(dir, runner.ScoresFile))
	if summary.EvidenceDigest != evidence.Digest(evBytes) || summary.ScoresDigest != evidence.Digest(scoreBytes) {
		summary.EvidenceIssues = append(summary.EvidenceIssues, "evidence or scores changed since scoring; rescore before judging")
	}
	res := report.Gate(&summary, th, prev)

	if err := writeJSON(filepath.Join(dir, "gate.json"), res); err != nil {
		return err
	}
	fmt.Print(res.Render())
	// 0 pass, 1 a bound failed, 2 a declared bound had nothing to judge (CHECK-VERDICT rule 2).
	if code := res.ExitCode(); code != 0 {
		os.Exit(code)
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
