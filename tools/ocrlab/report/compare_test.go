package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
)

func TestComparisonCompatibilityAndRegression(t *testing.T) {
	makeData := func(id string) *Data {
		return &Data{Run: &evidence.Run{RunID: id, Edition: evidence.EditionDesktop, Scenes: []evidence.Scene{{SceneID: "s"}}}, Summary: &metrics.Summary{Procedure: evidence.Procedure, ScorerDigest: "scorer"}, Scores: []*metrics.SceneScore{{SceneID: "s", Text: metrics.TextScore{Measured: true, StrictCER: 0}, Detection: metrics.DetectionScore{Recall: 1, Precision: 1}}}}
	}
	makeDecl := func() *evidence.Declaration {
		return &evidence.Declaration{Procedure: evidence.Procedure, ScorerDigest: "scorer", Scenes: map[string]evidence.Input{"s": {MediaSHA256: "media", AnnotationSHA256: "truth"}}}
	}
	for _, name := range []string{"compatible", "regression", "changed-truth", "removed", "missing-evidence", "different-edition", "missing-text"} {
		t.Run(name, func(t *testing.T) {
			b, a := makeData("before"), makeData("after")
			bd, ad := makeDecl(), makeDecl()
			want := true
			switch name {
			case "regression":
				a.Scores[0].Text.StrictCER = .5
			case "changed-truth":
				ad.Scenes["s"] = evidence.Input{MediaSHA256: "media", AnnotationSHA256: "other"}
				want = false
			case "removed":
				a.Run.Scenes = nil
				want = false
			case "missing-evidence":
				a.Summary.EvidenceIssues = []string{"missing render"}
				want = false
			case "different-edition":
				a.Run.Edition = evidence.EditionExtension
				want = false
			case "missing-text":
				a.Scores[0].Text.Measured = false
				want = false
			}
			c := Compare(b, a, bd, ad)
			if c.Compatible != want {
				t.Fatalf("%+v", c)
			}
			if name == "regression" && c.Scenes[0].State != "regressed" {
				t.Fatal(c.Scenes)
			}
			dir := t.TempDir()
			if err := WriteComparison(dir, c); err != nil {
				t.Fatal(err)
			}
			md, _ := os.ReadFile(filepath.Join(dir, "comparison.md"))
			if strings.Contains(string(md), "%!") {
				t.Fatalf("broken format: %s", md)
			}
		})
	}
}

// Evidence scored under an older procedure is not comparable with a new one. The refusal has to
// say why, not only that, so a reader can tell an instrument change from a product regression.
func TestComparisonRefusesAcrossProceduresWithAnExplanation(t *testing.T) {
	data := func(procedure string) (*Data, *evidence.Declaration) {
		d := &Data{
			Run:     &evidence.Run{RunID: procedure, Edition: evidence.EditionDesktop, Scenes: []evidence.Scene{{SceneID: "s"}}},
			Summary: &metrics.Summary{Procedure: procedure, ScorerDigest: "scorer"},
			Scores:  []*metrics.SceneScore{{SceneID: "s", Text: metrics.TextScore{Measured: true}, Detection: metrics.DetectionScore{Recall: 1, Precision: 1}}},
		}
		decl := &evidence.Declaration{Procedure: procedure, ScorerDigest: "scorer", Scenes: map[string]evidence.Input{"s": {MediaSHA256: "media", AnnotationSHA256: "truth"}}}
		return d, decl
	}
	before, bd := data("ocrlab-109-v1")
	after, ad := data(evidence.Procedure)
	if evidence.Procedure != "ocrlab-110-v1" {
		t.Fatalf("the current procedure is %q; update this test with the next measurement change", evidence.Procedure)
	}
	c := Compare(before, after, bd, ad)
	if c.Compatible {
		t.Fatal("a 109 bundle and a 110 bundle must not compare")
	}
	var reason string
	for _, l := range c.Limitations {
		if strings.Contains(l, "measurement procedure differs") {
			reason = l
		}
	}
	for _, want := range []string{"ocrlab-109-v1 -> ocrlab-110-v1", "changed meaning", "not comparable", "score-history"} {
		if !strings.Contains(reason, want) {
			t.Errorf("the refusal should explain itself (missing %q): %q", want, reason)
		}
	}
}
