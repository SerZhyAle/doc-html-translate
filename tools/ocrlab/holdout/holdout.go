// Package holdout governs the holdout split's lifecycle: how its membership is frozen before
// tuning, how every acceptance use is recorded, and how a later replacement stays visible
// instead of quietly becoming "a fresh holdout".
//
// The freeze record is DEV/ocrlab/holdout.json, versioned beside the corpus and the thresholds.
// It is created by `ocrlab holdout freeze` from the corpus as it stands, never by editing, and
// the audit command is what confronts the record with the corpus and the run history.
package holdout

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// SchemaVersion guards a record written against different meanings from being read as if it
// agreed with this build.
const SchemaVersion = 1

// PriorUse is an earlier run that declared the scene, recorded at freeze time so the history a
// reviewer needs travels with the record. Exploratory diagnosis before the freeze is allowed -
// diagnosing that a scene renders is not tuning against it - but it is disclosed, never silent.
type PriorUse struct {
	Run     string `json:"run"`
	On      string `json:"on,omitempty"`
	Purpose string `json:"purpose"`
}

// FrozenScene is one holdout member as frozen.
type FrozenScene struct {
	SHA256     string            `json:"sha256"`
	Licence    corpus.Licence    `json:"licence"`
	Categories []corpus.Category `json:"categories,omitempty"`
	PriorUse   []PriorUse        `json:"priorUse,omitempty"`
}

// Use is one recorded acceptance use of the holdout.
type Use struct {
	Run     string `json:"run"`
	On      string `json:"on"`
	Purpose string `json:"purpose"`
}

// Replacement is one recorded membership change after the freeze. Added=false is a removal.
type Replacement struct {
	SceneID string `json:"sceneId"`
	On      string `json:"on"`
	By      string `json:"by"`
	Reason  string `json:"reason"`
	Added   bool   `json:"added"`
}

// Freeze is DEV/ocrlab/holdout.json.
type Freeze struct {
	SchemaVersion int                    `json:"schemaVersion"`
	FrozenOn      string                 `json:"frozenOn"`
	FrozenBy      string                 `json:"frozenBy"`
	Limitations   string                 `json:"limitations"`
	Scenes        map[string]FrozenScene `json:"scenes"`
	Use           []Use                  `json:"use,omitempty"`
	Replacements  []Replacement          `json:"replacements,omitempty"`
}

// Load reads the freeze record. A missing file is an error, not an empty freeze: silently
// treating "never frozen" as "nothing to check" is exactly the failure mode the spec names.
func Load(path string) (*Freeze, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read holdout freeze %s: %w", path, err)
	}
	var f Freeze
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse holdout freeze %s: %w", path, err)
	}
	if f.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%s: schemaVersion %d, this build understands %d", path, f.SchemaVersion, SchemaVersion)
	}
	if strings.TrimSpace(f.FrozenOn) == "" || strings.TrimSpace(f.FrozenBy) == "" {
		return nil, fmt.Errorf("%s: a freeze without a date and an operator is a label somebody moved", path)
	}
	return &f, nil
}

// Save writes the record with stable ordering so two audits of unchanged state diff cleanly.
func Save(path string, f *Freeze) error {
	f.SchemaVersion = SchemaVersion
	if f.Use != nil {
		sort.Slice(f.Use, func(i, j int) bool {
			if f.Use[i].On != f.Use[j].On {
				return f.Use[i].On < f.Use[j].On
			}
			return f.Use[i].Run < f.Use[j].Run
		})
	}
	if f.Replacements != nil {
		sort.Slice(f.Replacements, func(i, j int) bool {
			if f.Replacements[i].On != f.Replacements[j].On {
				return f.Replacements[i].On < f.Replacements[j].On
			}
			return f.Replacements[i].SceneID < f.Replacements[j].SceneID
		})
	}
	buf, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(buf, '\n'), 0o644)
}

// RunUse is one run directory found in the local run history.
type RunUse struct {
	Dir       string
	RunID     string
	Purpose   string
	StartedAt string // RFC3339, "" when the evidence file carries none
	Scored    bool
	SceneIDs  []string
}

// ScanRuns reads every run directory under root (one run or a folder of runs). A history that
// cannot be scanned is reported as an error, never as an empty history: "no runs found" would
// silently drop exactly the tuning use the freeze and the audit exist to catch.
func ScanRuns(root string) ([]RunUse, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("run history %s unavailable: %w", root, err)
	}
	var out []RunUse
	seen := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == root {
			return nil
		}
		decl, err := readDeclaration(filepath.Join(path, "declaration.json"))
		if err != nil {
			if os.IsNotExist(err) {
				return nil // not a run directory
			}
			return err
		}
		if seen[decl.purpose+filepath.Base(path)] {
			return nil
		}
		seen[decl.purpose+filepath.Base(path)] = true
		_, scoreErr := os.Stat(filepath.Join(path, "summary.json"))
		out = append(out, RunUse{
			Dir: path, RunID: filepath.Base(path), Purpose: decl.purpose,
			StartedAt: readStartedAt(path), Scored: scoreErr == nil, SceneIDs: decl.sceneIDs,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RunID < out[j].RunID })
	return out, nil
}

type declShape struct {
	purpose  string
	sceneIDs []string
}

func readDeclaration(path string) (declShape, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return declShape{}, err
	}
	var d struct {
		Purpose  string   `json:"purpose"`
		SceneIDs []string `json:"sceneIds"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return declShape{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return declShape{purpose: d.Purpose, sceneIDs: d.SceneIDs}, nil
}

func readStartedAt(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "evidence.json"))
	if err != nil {
		return ""
	}
	var r struct {
		StartedAt string `json:"startedAt"`
	}
	if json.Unmarshal(data, &r) != nil {
		return ""
	}
	return r.StartedAt
}

// RunDate is the run's UTC calendar date, "" when undated.
func (r RunUse) RunDate() string {
	if len(r.StartedAt) < 10 {
		return ""
	}
	return r.StartedAt[:10]
}

// Refusal is one reason a candidate cannot enter the holdout.
type Refusal struct {
	SceneID string
	Rule    string
	Detail  string
}

func (r Refusal) String() string {
	return fmt.Sprintf("%-34s %-40s %s", r.Rule, r.SceneID, r.Detail)
}

// Eligibility judges every holdout scene in the manifest against the rules a freeze demands:
// an independent, rights-reviewed, human-annotated and independently checked record that was
// never judged in a development selection. Family violations are reported against the
// candidate whether the pair's problem line names the candidate or its relative.
func Eligibility(m *corpus.Manifest, anns map[string]*truth.Annotation, runs []RunUse) (eligible map[string]FrozenScene, refusals []Refusal) {
	eligible = map[string]FrozenScene{}
	family := corpus.FamilyProblems(m)
	for i := range m.Scenes {
		s := &m.Scenes[i]
		if s.Split != corpus.SplitHoldout {
			continue
		}
		add := func(rule, detail string) {
			refusals = append(refusals, Refusal{SceneID: s.ID, Rule: rule, Detail: detail})
		}
		for _, p := range family {
			if p.SceneID == s.ID || strings.Contains(p.Detail, s.ID) {
				add("holdout-family-conflict", p.Detail)
			}
		}
		if s.DerivedFrom != "" || s.Licence == corpus.LicenceSynthetic {
			add("holdout-not-independent", "a derived or synthetic scene is tunable by definition")
		}
		if !s.RightsReviewed() {
			add("holdout-rights-pending", "no human has recorded reading the asset's own licence page")
		}
		ann := anns[s.ID]
		if ann == nil {
			add("holdout-annotation-missing", "no annotation: a holdout scene cannot be gated without truth")
		} else if !ann.IsTruth() {
			add("holdout-annotation-not-truth", ann.NotTruthReason())
		} else {
			for _, p := range truth.Validate(ann, s) {
				add("holdout-annotation-invalid", p.String())
			}
		}
		var prior []PriorUse
		for _, r := range runs {
			if !contains(r.SceneIDs, s.ID) {
				continue
			}
			switch r.Purpose {
			case "selected-dev", "full-benchmark":
				// The scene was already part of a judged selection: accepting it later as
				// holdout would be moving a label onto a scene tuning has seen.
				add("holdout-previously-judged", fmt.Sprintf("declared in run %s (purpose %s) before the freeze", r.RunID, r.Purpose))
			default:
				prior = append(prior, PriorUse{Run: r.RunID, On: r.RunDate(), Purpose: r.Purpose})
			}
		}
		// A scene with any refusal is not eligible; the clean ones are frozen with their
		// recorded prior exploratory diagnosis.
		if allFor(refusals, s.ID) == nil {
			eligible[s.ID] = FrozenScene{SHA256: s.SHA256, Licence: s.Licence, Categories: s.Categories, PriorUse: prior}
		}
	}
	sort.Slice(refusals, func(i, j int) bool {
		if refusals[i].SceneID != refusals[j].SceneID {
			return refusals[i].SceneID < refusals[j].SceneID
		}
		return refusals[i].Rule < refusals[j].Rule
	})
	return eligible, refusals
}

func allFor(rs []Refusal, id string) []Refusal {
	var out []Refusal
	for _, r := range rs {
		if r.SceneID == id {
			out = append(out, r)
		}
	}
	return out
}

func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Build creates the record from the corpus as it stands. It refuses - listing every reason -
// rather than freezing a partial set, because a freeze that quietly skipped an ineligible
// scene would read as complete.
func Build(m *corpus.Manifest, anns map[string]*truth.Annotation, runs []RunUse, by, on string) (*Freeze, error) {
	by, on = strings.TrimSpace(by), strings.TrimSpace(on)
	if by == "" {
		return nil, fmt.Errorf("a freeze names the operator who froze it; -by is required")
	}
	if _, err := time.Parse("2006-01-02", on); err != nil {
		return nil, fmt.Errorf("-on must be YYYY-MM-DD: %w", err)
	}
	var holdout int
	for i := range m.Scenes {
		if m.Scenes[i].Split == corpus.SplitHoldout {
			holdout++
		}
	}
	if holdout == 0 {
		return nil, fmt.Errorf("the corpus has no holdout scenes; a freeze of an empty set is a file that can never fire")
	}
	eligible, refusals := Eligibility(m, anns, runs)
	if len(refusals) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "%d holdout scene(s) cannot be frozen:", len(refusals))
		for _, r := range refusals {
			fmt.Fprintf(&b, "\n  %s", r)
		}
		return nil, fmt.Errorf("%s", b.String())
	}
	return &Freeze{
		FrozenOn: on, FrozenBy: by, Scenes: eligible,
		Limitations: "Frozen before the section 4.1 corpus coverage targets are met; full-benchmark acceptance additionally requires them. Exploratory diagnosis recorded per scene under priorUse is not tuning; any later membership change must be recorded with holdout replace.",
	}, nil
}

// Problem is one audit finding.
type Problem struct {
	Rule   string
	Detail string
}

func (p Problem) String() string { return fmt.Sprintf("%-34s %s", p.Rule, p.Detail) }

// Audit confronts the freeze record with the corpus, the annotations and the local run
// history. It reports and never repairs.
func Audit(m *corpus.Manifest, f *Freeze, runs []RunUse) []Problem {
	var out []Problem
	add := func(rule, detail string) { out = append(out, Problem{Rule: rule, Detail: detail}) }

	corpusHoldout := map[string]*corpus.Scene{}
	for i := range m.Scenes {
		if m.Scenes[i].Split == corpus.SplitHoldout {
			corpusHoldout[m.Scenes[i].ID] = &m.Scenes[i]
		}
	}
	for id, s := range corpusHoldout {
		frozen, ok := f.Scenes[id]
		if !ok {
			add("holdout-addition-unrecorded", fmt.Sprintf("%s entered the corpus holdout without holdout replace -add", id))
			continue
		}
		if s.SHA256 != frozen.SHA256 {
			add("holdout-media-changed", fmt.Sprintf("%s media hash differs from the frozen record", id))
		}
	}
	for id := range f.Scenes {
		if _, ok := corpusHoldout[id]; !ok {
			add("holdout-removal-unrecorded", fmt.Sprintf("%s left the corpus holdout without holdout replace -remove", id))
		}
	}

	recorded := map[string]bool{}
	for _, u := range f.Use {
		if u.Purpose != "full-benchmark" {
			add("holdout-use-purpose", fmt.Sprintf("use of run %s records purpose %q; the holdout is for acceptance", u.Run, u.Purpose))
		}
		if recorded[u.Run] {
			add("holdout-use-duplicate", "run "+u.Run+" is recorded twice")
		}
		recorded[u.Run] = true
		if u.On < f.FrozenOn {
			add("holdout-use-before-freeze", fmt.Sprintf("run %s recorded used on %s, before the freeze %s", u.Run, u.On, f.FrozenOn))
		}
		found := false
		for _, r := range runs {
			if r.RunID == u.Run {
				found = true
				if r.Purpose != "full-benchmark" {
					add("holdout-use-purpose", fmt.Sprintf("run %s declares purpose %q", u.Run, r.Purpose))
				}
				if !r.Scored {
					add("holdout-use-unscored", "run "+u.Run+" has no scores; a declared run is not a judged use")
				}
			}
		}
		if !found {
			add("holdout-use-unknown-run", fmt.Sprintf("run %s is not in the scanned history; a recorded use must name evidence that exists", u.Run))
		}
	}
	for _, r := range runs {
		if r.Purpose == "full-benchmark" && r.Scored && !recorded[r.RunID] {
			add("holdout-use-unrecorded", fmt.Sprintf("run %s judged the full benchmark and is not in the use ledger", r.RunID))
		}
		if r.Purpose == "selected-dev" {
			for id := range f.Scenes {
				if contains(r.SceneIDs, id) {
					when := r.RunDate()
					if when == "" || when >= f.FrozenOn {
						add("holdout-tuned-after-freeze", fmt.Sprintf("run %s (%s) declared holdout scene %s in a development selection", r.RunID, when, id))
					}
				}
			}
		}
	}
	for _, rep := range f.Replacements {
		if strings.TrimSpace(rep.Reason) == "" {
			add("holdout-replacement-reason-missing", fmt.Sprintf("replacement of %s has no reason", rep.SceneID))
		}
		if strings.TrimSpace(rep.By) == "" {
			add("holdout-replacement-operator-missing", fmt.Sprintf("replacement of %s names no operator", rep.SceneID))
		}
		if rep.On < f.FrozenOn {
			add("holdout-replacement-before-freeze", fmt.Sprintf("replacement of %s is dated %s, before the freeze %s", rep.SceneID, rep.On, f.FrozenOn))
		}
		if rep.Added {
			if _, ok := f.Scenes[rep.SceneID]; !ok {
				add("holdout-replacement-orphan", fmt.Sprintf("%s was recorded added but is not in the frozen set", rep.SceneID))
			}
		} else if _, ok := f.Scenes[rep.SceneID]; ok {
			add("holdout-replacement-orphan", fmt.Sprintf("%s was recorded removed but is still in the frozen set", rep.SceneID))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rule != out[j].Rule {
			return out[i].Rule < out[j].Rule
		}
		return out[i].Detail < out[j].Detail
	})
	return out
}

// MembershipIssues checks a run's frozen corpus against the freeze record: what a full-benchmark
// gate judged must be exactly the frozen holdout, not a set that drifted since.
func MembershipIssues(f *Freeze, frozen *corpus.Manifest) []string {
	var out []string
	for i := range frozen.Scenes {
		s := &frozen.Scenes[i]
		if s.Split != corpus.SplitHoldout {
			continue
		}
		rec, ok := f.Scenes[s.ID]
		if !ok {
			out = append(out, s.ID+": in the run's holdout but not in the frozen holdout")
			continue
		}
		if s.SHA256 != rec.SHA256 {
			out = append(out, s.ID+": media hash differs from the frozen holdout")
		}
	}
	for id := range f.Scenes {
		if s := frozen.Find(id); s == nil || s.Split != corpus.SplitHoldout {
			out = append(out, id+": in the frozen holdout but not in the run's")
		}
	}
	sort.Strings(out)
	return out
}
