package main

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// writeScreens lays out a corpus root the way capture.mjs leaves it: one PNG per scene and the index beside them.
func writeScreens(t *testing.T, idx screensIndex, w, h int) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ScreensDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, sc := range idx.Scenes {
		f, err := os.Create(filepath.Join(root, filepath.FromSlash(sc.File)))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, image.NewGray(image.Rect(0, 0, w, h))); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	data, _ := json.Marshal(idx)
	if err := os.WriteFile(filepath.Join(dir, "index.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func screensSample(script string, box [4]int) screensIndex {
	return screensIndex{
		Producer:    "tools/ocrlab/screens/capture.mjs",
		GeneratedOn: "2026-10-30",
		Scenes: []screenScene{{
			ID: "screen-gui-page-ocr-ru-light", File: "screens/screen-gui-page-ocr-ru-light.png",
			Locale: "ru", Script: script, Theme: "light", Setup: "page-ocr", Width: 120, Height: 80, Version: "v26.1028.0224",
			Groups: []screenGroup{
				{Kind: "title", Lines: []screenLine{{Text: "OCR", Box: [4]int{10, 5, 40, 20}, FontPx: 18}}},
				{Kind: "paragraph", Lines: []screenLine{
					{Text: "Язык распознавания", Box: box, FontPx: 13},
					{Text: "по умолчанию", Box: [4]int{10, 45, 70, 58}, FontPx: 13},
				}},
			},
		}},
	}
}

func TestScreensBecomeScreenshotScenesWithDOMTruth(t *testing.T) {
	root := writeScreens(t, screensSample("Cyrl", [4]int{10, 30, 100, 43}), 120, 80)
	scenes, anns, err := screensFromIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 || len(anns) != 1 {
		t.Fatalf("got %d scenes and %d annotations, want 1 and 1", len(scenes), len(anns))
	}
	s, a := scenes[0], anns[0]
	if s.Licence != corpus.LicenceOwn || !s.RightsReviewed() || s.Split != corpus.SplitDev {
		t.Errorf("licence %s, rights reviewed %v, split %s: want OWN, true, dev", s.Licence, s.RightsReviewed(), s.Split)
	}
	if len(s.Categories) != 2 || s.Categories[0] != corpus.CatScreenshot || s.Categories[1] != corpus.CatScript {
		t.Errorf("categories %v, want [screenshot script] for a Cyrillic window", s.Categories)
	}
	if s.Width != 120 || s.Height != 80 || s.SHA256 == "" || s.Bytes == 0 {
		t.Errorf("measured %dx%d, sha %q, %d bytes", s.Width, s.Height, s.SHA256, s.Bytes)
	}
	if ps := corpus.Validate(&corpus.Manifest{SchemaVersion: corpus.SchemaVersion, Scenes: scenes}, root); hasDefect(ps) {
		t.Errorf("the scene is not a valid corpus entry: %v", ps)
	}
	if !a.IsTruth() || strings.TrimSpace(a.Review.CheckedBy) != "" || a.Review.AnnotatedBy != ScreensAnnotator {
		t.Errorf("truth %v, checked by %q, annotated by %q: want truth, unsigned, the DOM annotator", a.IsTruth(), a.Review.CheckedBy, a.Review.AnnotatedBy)
	}
	if ps := truth.Validate(a, &s); len(ps) != 0 {
		t.Errorf("the annotation does not validate: %v", ps)
	}
	g := a.Groups[1]
	if g.Type != truth.GroupParagraph || g.Transcript != "Язык распознавания по умолчанию" || len(g.Lines) != 2 {
		t.Errorf("group %s %q with %d lines", g.Type, g.Transcript, len(g.Lines))
	}
	if x0, y0, x1, y1 := g.Bounds.Bounds(); x0 != 10 || y0 != 30 || x1 != 100 || y1 != 58 {
		t.Errorf("bounds %d,%d-%d,%d, want the union of the lines 10,30-100,58", x0, y0, x1, y1)
	}

	latin := writeScreens(t, screensSample("Latn", [4]int{10, 30, 100, 43}), 120, 80)
	if scenes, _, err := screensFromIndex(latin); err != nil || len(scenes[0].Categories) != 1 {
		t.Errorf("a Latin window is screenshot only: %v, %v", scenes, err)
	}
}

func TestScreensRefuseGeometryTheImageCannotHold(t *testing.T) {
	outside := writeScreens(t, screensSample("Cyrl", [4]int{10, 30, 130, 43}), 120, 80)
	if _, _, err := screensFromIndex(outside); err == nil || !strings.Contains(err.Error(), "not inside") {
		t.Errorf("a line box past the image edge: err %v", err)
	}
	resized := writeScreens(t, screensSample("Cyrl", [4]int{10, 30, 100, 43}), 100, 80)
	if _, _, err := screensFromIndex(resized); err == nil || !strings.Contains(err.Error(), "the index says") {
		t.Errorf("an image whose size is not the window's: err %v", err)
	}
	if _, _, err := screensFromIndex(t.TempDir()); err == nil {
		t.Error("no index: want an error naming the capture step")
	}
}

func hasDefect(ps []corpus.Problem) bool {
	defects, _ := corpus.SplitProblems(ps)
	return len(defects) > 0
}
