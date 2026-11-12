package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// ScreensDir is the sub-directory of the corpus root that tools/ocrlab/screens/capture.mjs writes: the PNGs of the
// GUI launcher's window and the index.json that carries their text geometry.
const ScreensDir = "screens"

// ScreensAnnotator names who drew a screenshot scene's truth. Like synth's, it is not a person: the boxes are the
// window's own rendered DOM, read from the page that painted them, so they are independent of OCR without being a
// human's reading. A human signs a scene by filling Review.CheckedBy; until then it stays in the dev split.
const ScreensAnnotator = "tools/ocrlab/screens (rendered DOM boxes)"

// The rights decision for the screenshot scenes. They are the product's own interface, the owner's own work, and the
// owner recorded that in the decision of ticket E0121 (2026-10-30) - the same OWN case `ocrlab add` records for an
// owner's own image.
const (
	screensRightsBy = "owner (own material; ticket E0121 decision)"
	screensRightsOn = "2026-10-30"
)

// screensIndex is what capture.mjs writes. Boxes are natural image pixels [x0, y0, x1, y1].
type screensIndex struct {
	Producer    string        `json:"producer"`
	GeneratedOn string        `json:"generatedOn"`
	Scenes      []screenScene `json:"scenes"`
}

type screenScene struct {
	ID      string        `json:"id"`
	File    string        `json:"file"`
	Locale  string        `json:"locale"`
	Script  string        `json:"script"`
	Theme   string        `json:"theme"`
	Setup   string        `json:"setup"`
	Width   int           `json:"width"`
	Height  int           `json:"height"`
	Version string        `json:"version"`
	Groups  []screenGroup `json:"groups"`
}

type screenGroup struct {
	Kind  string       `json:"kind"`
	Lines []screenLine `json:"lines"`
}

type screenLine struct {
	Text   string  `json:"text"`
	Box    [4]int  `json:"box"`
	FontPx float64 `json:"fontPx"`
}

// cmdScreens turns the capture's index into corpus scenes and their annotations. It never captures: the window is
// driven by capture.mjs, so this half needs no browser and can be tested on a written index.
func cmdScreens(args []string) error {
	fs := flag.NewFlagSet("screens", flag.ExitOnError)
	p := addCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	scenes, anns, err := screensFromIndex(p.root)
	if err != nil {
		return err
	}
	kept := 0
	for i, s := range scenes {
		// A signed record of the same bytes is a human's work and stays; a new capture voids the signature,
		// because the image it vouched for is gone.
		if old, err := truth.Load(truth.FinalPath(p.annotations, s.ID)); err == nil && strings.TrimSpace(old.Review.CheckedBy) != "" {
			if prev := m.Find(s.ID); prev != nil && prev.SHA256 == s.SHA256 {
				m.Upsert(s)
				kept++
				continue
			}
			fmt.Printf("screens: %s was signed by %s; the capture changed, so the new record is unsigned\n", s.ID, old.Review.CheckedBy)
		}
		m.Upsert(s)
		if err := truth.Save(truth.FinalPath(p.annotations, s.ID), anns[i]); err != nil {
			return err
		}
	}
	if err := corpus.Save(p.manifest, m); err != nil {
		return err
	}
	fmt.Printf("screens: %d scene(s) from %s, %d annotation(s) written to %s, %d signed record(s) kept\n",
		len(scenes), filepath.Join(p.root, ScreensDir), len(scenes)-kept, p.annotations, kept)
	return nil
}

// screensFromIndex reads <root>/screens/index.json and measures every PNG it names. A scene whose file is missing or
// whose size disagrees with the index is an error: the geometry was read from a window of that size.
func screensFromIndex(root string) ([]corpus.Scene, []*truth.Annotation, error) {
	data, err := os.ReadFile(filepath.Join(root, ScreensDir, "index.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("read the capture index (run tools/ocrlab/screens/capture.mjs first): %w", err)
	}
	var idx screensIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, nil, fmt.Errorf("parse the capture index: %w", err)
	}
	if len(idx.Scenes) == 0 {
		return nil, nil, errors.New("the capture index names no scene")
	}
	var scenes []corpus.Scene
	var anns []*truth.Annotation
	for _, sc := range idx.Scenes {
		s, err := screenSceneEntry(root, idx, sc)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", sc.ID, err)
		}
		a, err := screenAnnotation(idx, sc)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", sc.ID, err)
		}
		scenes = append(scenes, s)
		anns = append(anns, a)
	}
	return scenes, anns, nil
}

func screenSceneEntry(root string, idx screensIndex, sc screenScene) (corpus.Scene, error) {
	s := corpus.Scene{
		ID:                sc.ID,
		File:              sc.File,
		SourceItem:        fmt.Sprintf("doc-html-ui %s window, state %s, locale %s, theme %s", sc.Version, sc.Setup, sc.Locale, sc.Theme),
		RetrievedOn:       idx.GeneratedOn,
		Licence:           corpus.LicenceOwn,
		LicenceVerifiedBy: screensRightsBy,
		LicenceVerifiedOn: screensRightsOn,
		Languages:         []string{sc.Locale},
		Scripts:           []string{sc.Script},
		Categories:        []corpus.Category{corpus.CatScreenshot},
		Split:             corpus.SplitDev,
	}
	if sc.Script != "Latn" {
		s.Categories = append(s.Categories, corpus.CatScript)
	}
	path := s.Path(root)
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	cfg, _, err := image.DecodeConfig(f)
	f.Close()
	if err != nil {
		return s, fmt.Errorf("not a readable image: %w", err)
	}
	if cfg.Width != sc.Width || cfg.Height != sc.Height {
		return s, fmt.Errorf("image is %dx%d, the index says %dx%d", cfg.Width, cfg.Height, sc.Width, sc.Height)
	}
	s.Width, s.Height = cfg.Width, cfg.Height
	info, err := os.Stat(path)
	if err != nil {
		return s, err
	}
	s.Bytes = info.Size()
	if s.SHA256, err = corpus.HashFile(path); err != nil {
		return s, err
	}
	return s, nil
}

var screenGroupTypes = map[string]truth.GroupType{
	"title":     truth.GroupTitle,
	"paragraph": truth.GroupParagraph,
	"label":     truth.GroupLabel,
}

func screenAnnotation(idx screensIndex, sc screenScene) (*truth.Annotation, error) {
	a := &truth.Annotation{
		SchemaVersion: truth.SchemaVersion,
		SceneID:       sc.ID,
		Origin:        truth.OriginHuman,
		ImageWidth:    sc.Width,
		ImageHeight:   sc.Height,
		Ambiguity:     truth.AmbiguityClear,
		Review:        truth.Review{AnnotatedBy: ScreensAnnotator, AnnotatedOn: idx.GeneratedOn},
	}
	for gi, g := range sc.Groups {
		kind, ok := screenGroupTypes[g.Kind]
		if !ok {
			return nil, fmt.Errorf("group %d: kind %q is not title, paragraph or label", gi+1, g.Kind)
		}
		if len(g.Lines) == 0 {
			return nil, fmt.Errorf("group %d has no line", gi+1)
		}
		id := fmt.Sprintf("g%03d", gi+1)
		var lines []truth.Region
		var texts []string
		x0, y0, x1, y1 := g.Lines[0].Box[0], g.Lines[0].Box[1], g.Lines[0].Box[2], g.Lines[0].Box[3]
		for li, l := range g.Lines {
			b := l.Box
			if b[0] < 0 || b[1] < 0 || b[2] > sc.Width || b[3] > sc.Height || b[2] <= b[0] || b[3] <= b[1] {
				return nil, fmt.Errorf("group %d line %d: box %v is not inside the %dx%d image", gi+1, li+1, b, sc.Width, sc.Height)
			}
			lines = append(lines, truth.Box(fmt.Sprintf("%s-l%d", id, li+1), b[0], b[1], b[2], b[3]))
			texts = append(texts, l.Text)
			x0, y0, x1, y1 = min(x0, b[0]), min(y0, b[1]), max(x1, b[2]), max(y1, b[3])
		}
		a.Groups = append(a.Groups, truth.Group{
			ID:           id,
			Type:         kind,
			Transcript:   strings.Join(texts, " "),
			Language:     sc.Locale,
			Direction:    truth.DirLTR,
			ReadingOrder: gi + 1,
			Lines:        lines,
			Bounds:       truth.Box(id+"-bounds", x0, y0, x1, y1),
			ReplaceArea:  truth.Box(id+"-replace", x0, y0, x1, y1),
		})
	}
	return a, nil
}
