package evidence

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// Observation proves a successful browser observation even when there are zero plates.
type Observation struct {
	Viewport   string `json:"viewport"`
	StressCase string `json:"stressCase"`
	Rendered   string `json:"rendered"`
	Concealed  string `json:"concealed"`
}

// Issues is an absence list, not product defects. Geometric defects stay scoreable.
func Issues(dir string, r *Run, d *Declaration) []string {
	if d == nil {
		return []string{"legacy evidence: no frozen declaration; exploratory inspection only"}
	}
	var out []string
	bad := func(s string) { out = append(out, s) }
	if d.Version != 1 || (d.Purpose != "exploratory" && d.Purpose != "selected-dev" && d.Purpose != "full-benchmark") {
		bad("unsupported declaration version or purpose")
	}
	declared := map[string]bool{}
	for _, id := range d.SceneIDs {
		if id == "" || declared[id] {
			bad("empty or duplicate declared scene")
		}
		declared[id] = true
		if _, ok := d.Scenes[id]; !ok {
			bad(id + ": input identity absent")
		}
	}
	for id := range d.Scenes {
		if !declared[id] {
			bad(id + ": input outside declared selection")
		}
	}
	viewports, stresses := map[string]bool{}, map[string]bool{}
	for _, v := range d.Viewports {
		if v.Name == "" || viewports[v.Name] || v.Width <= 0 || v.Height <= 0 || v.DeviceScaleFactor <= 0 {
			bad("invalid or duplicate declared viewport")
		}
		viewports[v.Name] = true
	}
	for _, c := range d.StressCases {
		if c == "" || stresses[c] {
			bad("invalid or duplicate declared stress")
		}
		stresses[c] = true
	}
	if d.Procedure != Procedure {
		bad(fmt.Sprintf("measurement procedure differs: the run declares %q, this scorer is %q; a run cannot certify acceptance under a procedure it was not declared for, and its numbers are not comparable across procedures", d.Procedure, Procedure))
	}
	if d.Purpose == "exploratory" {
		bad("exploratory scope cannot certify acceptance")
	}
	if d.SourceDigest == "" || d.ScorerDigest == "" || d.SourceRevision == "" {
		bad("source/scorer provenance unavailable")
	}
	if len(d.SceneIDs) == 0 || len(d.Viewports) == 0 || len(d.StressCases) == 0 {
		bad("empty declared selection or observation matrix")
	}
	if len(r.Viewports) != len(d.Viewports) {
		bad("viewport declaration changed")
	} else {
		for i, v := range r.Viewports {
			if v != d.Viewports[i] {
				bad("viewport declaration changed")
				break
			}
		}
	}
	seen := map[string]bool{}
	for _, sc := range r.Scenes {
		if seen[sc.SceneID] {
			bad(sc.SceneID + ": duplicate scene")
		}
		seen[sc.SceneID] = true
		if _, ok := d.Scenes[sc.SceneID]; !ok {
			bad(sc.SceneID + ": undeclared scene")
		}
	}
	for _, id := range d.SceneIDs {
		s := r.Find(id)
		if s == nil {
			bad(id + ": required scene absent")
			continue
		}
		if s.Error != "" {
			bad(id + ": observation failed: " + s.Error)
			continue
		}
		if s.Unmeasured != "" {
			// One line, not the cascade of missing captures the absent observations would otherwise be.
			bad(id + ": unmeasured: " + s.Unmeasured)
			continue
		}
		if want := d.Scenes[id].Lang; want != "" && s.Lang != "" && s.Lang != want {
			bad(fmt.Sprintf("%s: read with language %s, declared %s", id, s.Lang, want))
		}
		if s.ImageWidth <= 0 || s.ImageHeight <= 0 {
			bad(id + ": invalid coordinate space")
			continue
		}
		if err := CheckShot(dir, s.Screenshots.Source, s.ImageWidth, s.ImageHeight); err != nil {
			bad(id + ": source: " + err.Error())
		}
		obs := map[string]Observation{}
		for _, o := range s.Observations {
			k := o.Viewport + "/" + o.StressCase
			if !viewports[o.Viewport] || !stresses[o.StressCase] {
				bad(id + ": undeclared observation " + k)
			}
			if _, ok := obs[k]; ok {
				bad(id + ": duplicate observation " + k)
			}
			obs[k] = o
		}
		for _, p := range s.Plates {
			if _, ok := obs[p.Viewport+"/"+p.StressCase]; !ok {
				bad(id + ": plate without observation")
			}
		}
		for _, v := range d.Viewports {
			for _, c := range d.StressCases {
				k := v.Name + "/" + c
				o, ok := obs[k]
				if !ok {
					bad(id + ": missing observation " + k)
					continue
				}
				for label, path := range map[string]string{"rendered": o.Rendered, "concealed": o.Concealed} {
					if err := CheckShot(dir, path, s.ImageWidth, s.ImageHeight); err != nil {
						bad(id + ": " + k + " " + label + ": " + err.Error())
					}
				}
			}
		}
	}
	return out
}

// SnapshotIssues revalidates portable input bytes at both scoring and judging time.
func SnapshotIssues(dir string, r *Run, d *Declaration) []string {
	if d == nil {
		return nil
	}
	var out []string
	var execution struct {
		SourceDigest string `json:"sourceDigest"`
		Unchanged    bool   `json:"unchanged"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "execution.json"))
	if err != nil || json.Unmarshal(data, &execution) != nil || !execution.Unchanged || execution.SourceDigest != d.SourceDigest {
		out = append(out, "source changed during execution or final source identity unavailable")
	}
	m, err := corpus.Load(filepath.Join(dir, "corpus.json"))
	if err != nil {
		return []string{"frozen corpus unavailable: " + err.Error()}
	}
	data, err = os.ReadFile(filepath.Join(dir, "family-corpus.json"))
	if err != nil || d.FamilyDigest == "" || Digest(data) != d.FamilyDigest {
		out = append(out, "source-family snapshot missing or changed")
	} else {
		family, err := corpus.Load(filepath.Join(dir, "family-corpus.json"))
		if err != nil {
			out = append(out, "invalid source-family snapshot")
		} else {
			for _, p := range corpus.FamilyProblems(family) {
				out = append(out, p.String())
			}
		}
	}
	for _, id := range d.SceneIDs {
		input := d.Scenes[id]
		s := m.Find(id)
		if s == nil {
			out = append(out, id+": frozen corpus record missing")
			continue
		}
		data, _ := json.Marshal(s)
		if Digest(data) != input.SceneSHA256 {
			out = append(out, id+": frozen corpus record changed")
		}
		if input.MediaSHA256 == "" || input.MediaSHA256 != s.SHA256 {
			out = append(out, id+": media identity missing or inconsistent")
		}
		if sc := r.Find(id); sc != nil {
			path, err := SafePath(dir, sc.Screenshots.Source)
			if err == nil {
				hash, err := corpus.HashFile(path)
				if err != nil || hash != input.MediaSHA256 {
					out = append(out, id+": source media changed or unavailable")
				}
			}
			if sc.ImageWidth != s.Width || sc.ImageHeight != s.Height {
				out = append(out, id+": observation coordinate space differs from corpus")
			}
		}
		path := truth.FinalPath(filepath.Join(dir, "truth"), id)
		data, err = os.ReadFile(path)
		if err != nil || input.AnnotationSHA256 == "" || Digest(data) != input.AnnotationSHA256 {
			out = append(out, id+": frozen annotation changed or unavailable")
			continue
		}
		letteringData, letteringErr := os.ReadFile(truth.LetteringPath(filepath.Join(dir, "truth"), id))
		switch {
		case input.LetteringSHA256 == "" && letteringErr == nil:
			out = append(out, id+": lettering mask present but not declared")
		case input.LetteringSHA256 != "" && (letteringErr != nil || Digest(letteringData) != input.LetteringSHA256):
			out = append(out, id+": frozen lettering mask changed or unavailable")
		}
		a, err := truth.Load(path)
		if err != nil {
			out = append(out, id+": invalid frozen annotation: "+err.Error())
			continue
		}
		if !a.IsTruth() {
			out = append(out, id+": "+a.NotTruthReason())
		}
		for _, p := range truth.Validate(a, s) {
			out = append(out, p.String())
		}
		if !s.RightsReviewed() {
			out = append(out, id+": rights review pending")
		}
		if d.Purpose == "selected-dev" && s.Split != corpus.SplitDev {
			out = append(out, id+": selected-dev contains holdout")
		}
	}
	for _, p := range corpus.FamilyProblems(m) {
		out = append(out, p.String())
	}
	if d.Purpose == "full-benchmark" {
		for _, p := range corpus.Validate(m, "") {
			out = append(out, p.String())
		}
	}
	return out
}

func SafePath(dir, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", fmt.Errorf("missing or non-relative path")
	}
	p := filepath.Clean(filepath.FromSlash(rel))
	if p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path leaves run directory")
	}
	return filepath.Join(dir, p), nil
}

func CheckShot(dir, rel string, w, h int) error {
	p, err := SafePath(dir, rel)
	if err != nil {
		return err
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	b := im.Bounds()
	if b.Dx() != w || b.Dy() != h {
		return fmt.Errorf("dimensions %dx%d, expected %dx%d", b.Dx(), b.Dy(), w, h)
	}
	return nil
}
