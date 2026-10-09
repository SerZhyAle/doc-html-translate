// Package report turns a scored run into something a person can act on: a Markdown table for
// the ledger and a self-contained HTML page that puts the source and the result side by side
// with every failure named in words.
//
// The rule the whole design serves: an aggregate may never be the only thing a reader sees. A
// mean recall of 0.94 and a balloon outline painted over are both true at once, and only one of
// them matters. So every scene that failed appears with its own pictures and its own sentence,
// and the skipped scenes appear too - a scene absent from the denominator is a result, not a
// gap in the page.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
)

// Data is everything a report needs, already loaded.
type Data struct {
	Run     *evidence.Run
	Scores  []*metrics.SceneScore
	Summary *metrics.Summary
}

// LoadData reads a run directory produced by `ocrlab run` + `ocrlab score`.
func LoadData(dir string) (*Data, error) {
	run, err := evidence.LoadRun(filepath.Join(dir, "evidence.json"))
	if err != nil {
		return nil, err
	}
	var scores []*metrics.SceneScore
	if err := readJSON(filepath.Join(dir, "scores.json"), &scores); err != nil {
		return nil, err
	}
	var summary metrics.Summary
	if err := readJSON(filepath.Join(dir, "summary.json"), &summary); err != nil {
		return nil, err
	}
	d, _ := evidence.LoadDeclaration(dir)
	summary.EvidenceIssues = append(summary.EvidenceIssues, evidence.Issues(dir, run, d)...)
	summary.EvidenceIssues = append(summary.EvidenceIssues, evidence.SnapshotIssues(dir, run, d)...)
	for _, pair := range []struct{ name, hash string }{{"evidence.json", summary.EvidenceDigest}, {"scores.json", summary.ScoresDigest}, {"declaration.json", summary.DeclarationDigest}} {
		data, err := os.ReadFile(filepath.Join(dir, pair.name))
		if err != nil || pair.hash == "" || evidence.Digest(data) != pair.hash {
			summary.EvidenceIssues = append(summary.EvidenceIssues, pair.name+": unavailable or changed since scoring")
		}
	}
	return &Data{Run: run, Scores: scores, Summary: &summary}, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// WriteMarkdown renders report.md.
func WriteMarkdown(dir string, d *Data) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# OCR visual-fidelity report - %s (%s)\n\n", d.Run.RunID, d.Run.Edition)
	fmt.Fprintf(&b, "**Started:** %s  \n", d.Run.StartedAt)
	fmt.Fprintf(&b, "**Scope:** %s; procedure %s. No hard failure is not an acceptance PASS.\n\n", or(d.Summary.Purpose, "legacy unspecified"), or(d.Summary.Procedure, "legacy"))
	for _, issue := range d.Summary.EvidenceIssues {
		fmt.Fprintf(&b, "- Unavailable evidence: %s\n", issue)
	}
	fmt.Fprintf(&b, "**Engine:** %s, tessdata %s, lang %s  \n",
		or(d.Run.Engine.Tesseract, "unknown"), or(d.Run.Engine.TessdataVersion, "unknown"), d.Run.Engine.Lang)
	fmt.Fprintf(&b, "**Browser:** %s  \n", or(d.Run.Browser.Version, d.Run.Browser.Name))
	fmt.Fprintf(&b, "**Viewports:** %s\n\n", viewportList(d.Run))

	// Coverage first, and deliberately so: every number below is over this many scenes and no
	// more, and a reader who skips this line will misread the rest of the page.
	fmt.Fprintf(&b, "**Coverage:** %d scene(s) in the run, %d scored, %d skipped.\n\n",
		len(d.Run.Scenes), len(d.Scores), len(d.Summary.Skipped))
	fmt.Fprintf(&b, "**Evidence:** %s Scoring, the gate and this report always run on the evidence as it stands.\n\n", evidenceLine(d.Run))
	writeReusedMarkdown(&b, d.Run)
	fmt.Fprintf(&b, "**Not measured:** concealment in %d scene(s), painted damage in %d scene(s); those carry no number in the tables below.\n\n",
		d.Summary.Overall.UnmeasuredConcealment, d.Summary.Overall.UnmeasuredDamage)
	if len(d.Summary.Skipped) > 0 {
		b.WriteString("| Skipped scene | Why |\n|---|---|\n")
		for _, s := range d.Summary.Skipped {
			fmt.Fprintf(&b, "| `%s` | %s |\n", s.SceneID, s.Reason)
		}
		b.WriteString("\n")
	}

	b.WriteString("## By category\n\n")
	writeBucketTable(&b, "Category", categoryRows(d.Summary))
	b.WriteString("\n## By split\n\n")
	writeBucketTable(&b, "Split", splitRows(d.Summary))

	b.WriteString("\n## Failing scenes\n\n")
	failing := failingScores(d.Scores)
	if len(failing) == 0 {
		b.WriteString("None.\n")
	} else {
		b.WriteString("| Scene | Failure |\n|---|---|\n")
		for _, s := range failing {
			for _, f := range s.Failures {
				fmt.Fprintf(&b, "| `%s` | %s |\n", s.SceneID, f)
			}
			if s.Loss != nil {
				fmt.Fprintf(&b, "| `%s` | loss point: %s |\n", s.SceneID, s.Loss)
			}
		}
	}

	b.WriteString("\n## Per scene\n\n")
	b.WriteString("| Scene | Recall | CER | IoU | Covered | Residual | Painted damage px | Rectangle intrusion px | Replacement glyphs | Merges | Clipped | OCR ms |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---|---:|---:|---:|\n")
	for _, s := range sortedScores(d.Scores) {
		fmt.Fprintf(&b, "| `%s` | %.2f | %.2f | %.2f | %.2f | %s | %s | %d | %s | %d | %d | %d |\n",
			s.SceneID, s.Detection.Recall, s.Text.MeanCER, s.Placement.MeanIoU, s.Covered,
			residualCell(s.Residual), damageCell(s.Damage), s.RectangleIntrusion.ProtectedHit,
			readabilityCell(s.Readability), s.Grouping.Merges, totalClipped(s), s.Cost.OcrMs)
	}
	b.WriteString("\nResidual is the share of the known old lettering still showing in the text-hidden capture; " +
		"it is `unmeasured` where the background is not a single tone and no lettering mask is declared. " +
		"Painted damage counts protected pixels the overlay changed; rectangle intrusion is the plates' bounding boxes over protected content. " +
		"Replacement glyphs describe the text actually drawn, not the plate box.\n")
	b.WriteString("\nOpen `report.html` for the side-by-side pictures of every scene.\n")

	return os.WriteFile(filepath.Join(dir, "report.md"), []byte(b.String()), 0o644)
}

func writeBucketTable(b *strings.Builder, label string, rows []bucketRow) {
	fmt.Fprintf(b, "| %s | Scenes | Recall | Precision | CER | IoU | Worst IoU | Covered | Worst residual (measured) | Merges | Splits | Painted damage px | Clipped | Cross-group | Drift | Failing |\n", label)
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	if len(rows) == 0 {
		fmt.Fprintf(b, "| _no scored scenes_ | | | | | | | | | | | | | | | |\n")
		return
	}
	for _, r := range rows {
		k := r.bucket
		fmt.Fprintf(b, "| %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %d | %d | %d | %d | %d | %.3f | %d |\n",
			r.name, k.Scenes, k.MeanRecall, k.MeanPrecision, k.MeanCER, k.MeanIoU, k.WorstIoU,
			k.MeanCovered, k.WorstResidual, k.Merges, k.Splits, k.ProtectedHitPx,
			k.Clipped, k.CrossGroup, k.WorstDrift, k.FailingScenes)
	}
}

type bucketRow struct {
	name   string
	bucket *metrics.Bucket
}

func categoryRows(s *metrics.Summary) []bucketRow {
	var out []bucketRow
	for _, c := range corpus.Categories() {
		if b := s.ByCategory[c]; b != nil {
			out = append(out, bucketRow{string(c), b})
		}
	}
	if s.Overall.Scenes > 0 {
		out = append(out, bucketRow{"**all**", &s.Overall})
	}
	return out
}

func splitRows(s *metrics.Summary) []bucketRow {
	var out []bucketRow
	for _, sp := range []corpus.Split{corpus.SplitDev, corpus.SplitHoldout} {
		if b := s.BySplit[sp]; b != nil {
			out = append(out, bucketRow{string(sp), b})
		}
	}
	return out
}

func failingScores(scores []*metrics.SceneScore) []*metrics.SceneScore {
	var out []*metrics.SceneScore
	for _, s := range scores {
		if len(s.Failures) > 0 {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SceneID < out[j].SceneID })
	return out
}

func sortedScores(scores []*metrics.SceneScore) []*metrics.SceneScore {
	out := append([]*metrics.SceneScore(nil), scores...)
	sort.Slice(out, func(i, j int) bool { return out[i].SceneID < out[j].SceneID })
	return out
}

func totalClipped(s *metrics.SceneScore) int {
	n := 0
	for _, r := range s.Stress {
		n += r.Clipped
	}
	return n
}

func viewportList(r *evidence.Run) string {
	var parts []string
	for _, v := range r.Viewports {
		parts = append(parts, fmt.Sprintf("%s %dx%d@%gx", v.Name, v.Width, v.Height, v.DeviceScaleFactor))
	}
	return strings.Join(parts, ", ")
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
