package evidence

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"

	"doc-html-translate/tools/ocrlab/corpus"
)

// completeScene reports why an earlier scene record cannot stand in for a collection, or "" when
// it is complete: no error, measured, the full viewport-by-stress observation matrix, and every
// file it names present, decodable at the scene's size and (for the source copy) byte-identical
// to the declared media. Only image headers are read, so the check stays cheap.
func completeScene(dir string, d *Declaration, edition Edition, sc *Scene, in Input) string {
	switch {
	case sc.Error != "":
		return "the scene failed in that run: " + sc.Error
	case sc.Unmeasured != "":
		return "the scene was unmeasured in that run: " + sc.Unmeasured
	case sc.ImageWidth <= 0 || sc.ImageHeight <= 0:
		return "the scene has no coordinate space"
	}

	declaredViewports, declaredStress := map[string]bool{}, map[string]bool{}
	for _, v := range d.Viewports {
		declaredViewports[v.Name] = true
	}
	for _, c := range d.StressCases {
		declaredStress[c] = true
	}
	seen := map[string]bool{}
	var files []string
	for _, o := range sc.Observations {
		k := o.Viewport + "/" + o.StressCase
		if !declaredViewports[o.Viewport] || !declaredStress[o.StressCase] || seen[k] {
			return "the observation matrix is not the declared one (" + k + ")"
		}
		if o.Rendered == "" || o.Concealed == "" {
			return "observation " + k + " lacks a capture"
		}
		seen[k] = true
		files = append(files, o.Rendered, o.Concealed)
	}
	if len(seen) != len(d.Viewports)*len(d.StressCases) {
		return fmt.Sprintf("only %d of %d observations were recorded", len(seen), len(d.Viewports)*len(d.StressCases))
	}
	for _, p := range sc.Plates {
		if !seen[p.Viewport+"/"+p.StressCase] {
			return "a plate has no observation"
		}
	}
	files = append(files, sc.Screenshots.Rendered)
	for _, rel := range sc.Screenshots.Stress {
		files = append(files, rel)
	}
	sort.Strings(files)
	for i, rel := range files {
		if i > 0 && rel == files[i-1] {
			continue
		}
		if err := checkShotHeader(dir, rel, sc.ImageWidth, sc.ImageHeight); err != nil {
			return "a capture is missing or damaged: " + err.Error()
		}
	}

	source, err := SafePath(dir, sc.Screenshots.Source)
	if err != nil {
		return "the source copy is not recorded"
	}
	if hash, err := corpus.HashFile(source); err != nil || hash != in.MediaSHA256 {
		return "the source copy is missing or is not the declared media"
	}
	if edition == EditionDesktop {
		if entries, err := os.ReadDir(filepath.Join(dir, PagesDir, sc.SceneID)); err != nil || len(entries) == 0 {
			return "the converted pages are missing"
		}
	}
	return ""
}

// checkShotHeader is CheckShot without decoding the pixels: the file must exist, be a readable
// image and have the scene's dimensions.
func checkShotHeader(dir, rel string, w, h int) error {
	p, err := SafePath(dir, rel)
	if err != nil {
		return err
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	if cfg.Width != w || cfg.Height != h {
		return fmt.Errorf("%s: dimensions %dx%d, expected %dx%d", rel, cfg.Width, cfg.Height, w, h)
	}
	return nil
}

// checkManifest compares a scene's files on disk with the manifest its run wrote when it
// finished. A run without a manifest predates it: its files are accepted on the dimension check
// alone and the provenance says so.
func checkManifest(dir, sceneID string) (integrity, why string) {
	sizes, present, err := loadFilesManifest(dir)
	if err != nil {
		return "", "that run's file manifest is unreadable: " + err.Error()
	}
	if !present {
		return "dimensions", ""
	}
	want := sizes[sceneID]
	if len(want) == 0 {
		return "", "that run's file manifest does not list the scene"
	}
	have, err := sceneFiles(dir, sceneID)
	if err != nil {
		return "", "the scene's files cannot be listed: " + err.Error()
	}
	if len(have) != len(want) {
		return "", fmt.Sprintf("the scene has %d files, the run recorded %d", len(have), len(want))
	}
	for rel, size := range want {
		if got, ok := have[rel]; !ok || got != size {
			return "", "a file changed since the run finished: " + rel
		}
	}
	return "sizes", ""
}
