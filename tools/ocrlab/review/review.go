// Package review produces a portable offline annotation workspace. Drafts download to the
// reviewer's filesystem; importing validates them and never overwrites reviewed truth.
package review

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

type Item struct {
	Scene      corpus.Scene      `json:"scene"`
	Annotation *truth.Annotation `json:"annotation"`
	Source     string            `json:"source"`
	Queues     []string          `json:"queues"`
}

func Write(out, root string, m *corpus.Manifest, anns map[string]*truth.Annotation) error {
	items := []Item{}
	for _, s := range m.Scenes {
		a := anns[s.ID]
		if a == nil {
			a = &truth.Annotation{SchemaVersion: truth.SchemaVersion, SceneID: s.ID, ImageWidth: s.Width, ImageHeight: s.Height, Ambiguity: truth.AmbiguityClear, Groups: []truth.Group{}, Protected: []truth.Region{}}
		}
		i := Item{Scene: s, Annotation: a, Queues: []string{}}
		if !s.RightsReviewed() {
			i.Queues = append(i.Queues, "rights")
		}
		if !a.IsTruth() || len(truth.Validate(a, &s)) > 0 {
			i.Queues = append(i.Queues, "annotation")
		}
		if s.Split == corpus.SplitHoldout && (a.Review.CheckedBy == "" || a.Review.CheckedBy == a.Review.AnnotatedBy) {
			i.Queues = append(i.Queues, "independent")
		}
		if data, err := os.ReadFile(s.Path(root)); err == nil {
			i.Source = "data:" + http.DetectContentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)
		}
		items = append(items, i)
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return err
	}
	page := fmt.Sprintf(pageTemplate, html.EscapeString(filepath.Base(out)), string(data))
	return os.WriteFile(out, []byte(page), 0644)
}

func Import(path, dir string, m *corpus.Manifest) (string, error) {
	a, err := truth.Load(path)
	if err != nil {
		return "", err
	}
	s := m.Find(a.SceneID)
	if s == nil {
		return "", fmt.Errorf("unknown scene %q", a.SceneID)
	}
	// Review signatures do not validate geometry. Validate as dev before routing to review.
	meta := *s
	meta.Split = corpus.SplitDev
	if ps := truth.Validate(a, &meta); len(ps) > 0 {
		return "", fmt.Errorf("draft validation: %s", ps[0].String())
	}
	a = truth.Draft(a)
	dest := truth.DraftPath(dir, s.ID)
	if err := truth.Save(dest, a); err != nil {
		return "", err
	}
	return dest, nil
}
