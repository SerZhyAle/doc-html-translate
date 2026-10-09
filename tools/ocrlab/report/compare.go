package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
)

type SceneChange struct {
	SceneID  string              `json:"sceneId"`
	State    string              `json:"state"`
	Before   *metrics.SceneScore `json:"before,omitempty"`
	After    *metrics.SceneScore `json:"after,omitempty"`
	Findings []string            `json:"findings,omitempty"`
}

type Comparison struct {
	Before      string        `json:"before"`
	After       string        `json:"after"`
	Edition     string        `json:"edition"`
	Compatible  bool          `json:"compatible"`
	Limitations []string      `json:"limitations"`
	Scenes      []SceneChange `json:"scenes"`
}

// Compare never drops additions/removals into a common aggregate. Engine changes are the
// subject of a quality comparison; corpus/truth, layout, language and measurement must agree.
func Compare(before, after *Data, bd, ad *evidence.Declaration) Comparison {
	c := Comparison{Before: before.Run.RunID, After: after.Run.RunID, Edition: string(after.Run.Edition), Compatible: true, Limitations: []string{}}
	bad := func(s string) { c.Compatible = false; c.Limitations = append(c.Limitations, s) }
	if len(before.Scores) == 0 || len(after.Scores) == 0 {
		bad("empty scored selection")
	}
	if bd == nil || ad == nil {
		bad("legacy run lacks frozen provenance")
	} else {
		switch {
		case bd.Procedure != ad.Procedure || before.Summary.Procedure != after.Summary.Procedure:
			bad(fmt.Sprintf("measurement procedure differs (%s -> %s): %s", describeProcedure(bd, before), describeProcedure(ad, after), procedureChangeNote))
		case before.Summary.ScorerDigest == "" || before.Summary.ScorerDigest != after.Summary.ScorerDigest:
			bad("scorer differs: the measurement code changed between the two runs")
		}
		if !reflect.DeepEqual(bd.Viewports, ad.Viewports) || !reflect.DeepEqual(bd.StressCases, ad.StressCases) || !reflect.DeepEqual(bd.Settings, ad.Settings) || before.Run.Browser != after.Run.Browser {
			bad("viewport, stress, settings or browser differs")
		}
	}
	if before.Run.Edition != after.Run.Edition {
		bad("different editions: side-by-side diagnosis only")
	}
	bs, as := map[string]*metrics.SceneScore{}, map[string]*metrics.SceneScore{}
	ids := map[string]bool{}
	for _, s := range before.Run.Scenes {
		ids[s.SceneID] = true
	}
	for _, s := range after.Run.Scenes {
		ids[s.SceneID] = true
	}
	for _, s := range before.Scores {
		bs[s.SceneID] = s
	}
	for _, s := range after.Scores {
		as[s.SceneID] = s
	}
	var ordered []string
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	for _, id := range ordered {
		s := SceneChange{SceneID: id, Before: bs[id], After: as[id], State: "unchanged"}
		switch {
		case before.Run.Find(id) == nil:
			s.State = "added"
			bad(id + ": added scene outside shared subset")
		case after.Run.Find(id) == nil:
			s.State = "removed"
			bad(id + ": removed scene outside shared subset")
		case bd == nil || ad == nil:
			s.State = "incompatible"
		case bd.Scenes[id].MediaSHA256 == "" || ad.Scenes[id].MediaSHA256 == "" || bd.Scenes[id].AnnotationSHA256 == "" || ad.Scenes[id].AnnotationSHA256 == "" || bd.Scenes[id] != ad.Scenes[id]:
			s.State = "changed-input"
			bad(id + ": media, scene or truth changed")
		case s.Before == nil || s.After == nil:
			s.State = "unavailable"
			bad(id + ": score unavailable")
		default:
			for _, f := range s.After.Failures {
				if !contains(s.Before.Failures, f) {
					s.Findings = append(s.Findings, "new failure: "+f)
				}
			}
			if len(s.Findings) > 0 {
				s.State = "regressed"
			} else if len(s.After.Failures) < len(s.Before.Failures) {
				s.State = "improved"
			}
			if s.After.Text.Measured != s.Before.Text.Measured {
				s.Findings = append(s.Findings, "text accuracy availability changed")
				bad(id + ": text accuracy availability changed")
				s.State = "unavailable"
			}
			if s.After.Text.Measured && s.Before.Text.Measured {
				if s.After.Text.StrictCER > s.Before.Text.StrictCER {
					s.Findings = append(s.Findings, "strict character errors increased")
					s.State = "regressed"
				} else if s.After.Text.StrictCER < s.Before.Text.StrictCER && s.State == "unchanged" {
					s.State = "improved"
				}
			}
			if s.After.Detection.Recall < s.Before.Detection.Recall || s.After.Detection.Precision < s.Before.Detection.Precision {
				s.Findings = append(s.Findings, "detection recall or precision decreased")
				s.State = "regressed"
			}
		}
		c.Scenes = append(c.Scenes, s)
	}
	if len(before.Summary.EvidenceIssues) > 0 || len(after.Summary.EvidenceIssues) > 0 || len(before.Summary.Skipped) > 0 || len(after.Summary.Skipped) > 0 {
		bad("one or both runs have unavailable required evidence")
	}
	return c
}

// procedureChangeNote says what a procedure difference means for the numbers, so the refusal is
// an explanation and not a bare "incompatible".
const procedureChangeNote = "concealment residual, readability, background contrast and protected-area damage changed meaning between procedures " +
	"(ocrlab-110-v1 judges residual on the text-hidden capture, reports drawn glyphs separately from plate-rectangle contrast, and counts painted pixels), " +
	"so their numbers are not comparable; compare runs of one procedure, or rescore the earlier evidence under the new one (the previous scores are kept in score-history/)"

// describeProcedure names the procedure a run declared and the one its scores were written under,
// which differ when old evidence was rescored.
func describeProcedure(d *evidence.Declaration, data *Data) string {
	scored := data.Summary.Procedure
	if scored == "" || scored == d.Procedure {
		return d.Procedure
	}
	return d.Procedure + ", scored as " + scored
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func WriteComparison(dir string, c Comparison) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "comparison.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	text := fmt.Sprintf("# Run comparison: %s -> %s (%s)\n\nRegression acceptance eligible: %t. This is measurement comparison, not a baseline approval.\n\n", c.Before, c.After, c.Edition, c.Compatible)
	for _, s := range c.Limitations {
		text += "- " + s + "\n"
	}
	text += "\n| Scene | Change | Findings |\n| --- | --- | --- |\n"
	for _, s := range c.Scenes {
		text += fmt.Sprintf("| %s | %s | %v |\n", s.SceneID, s.State, s.Findings)
	}
	return os.WriteFile(filepath.Join(dir, "comparison.md"), []byte(text), 0644)
}
