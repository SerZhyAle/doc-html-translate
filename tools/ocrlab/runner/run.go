// Package runner produces the desktop edition's evidence: it converts each corpus scene through
// the real pipeline, renders the result in a pinned headless browser, applies the deterministic
// translation-stress cases and records what the browser actually laid out.
//
// It measures the shipped program, not a copy of it. The plate geometry comes from the DOM the
// app produced and from the app's own diagnostics sidecar; nothing here re-implements
// clustering, plate styling or colour sampling, because a benchmark that scored its own
// reimplementation would be measuring the wrong thing.
package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
)

// Options configure one run.
type Options struct {
	Manifest string
	Root     string
	OutDir   string
	Split    string
	SceneIDs []string
	Lang     string
	Log      io.Writer
}

// Layout of a run directory. Everything is relative to OutDir so the folder can be moved,
// zipped or attached to a report and still open.
const (
	EvidenceFile = "evidence.json"
	ScoresFile   = "scores.json"
	SummaryFile  = "summary.json"
	SelfFile     = "selftest.json"
	DiagFile     = "ocr-diag.jsonl"
	ShotsDir     = "shots"
	PagesDir     = "pages"
)

// Run converts, renders and records every selected scene.
//
// It refuses to start on an incomplete corpus. A missing asset or a hash mismatch on a selected
// scene stops the run rather than shrinking it: a report over the scenes that happened to be
// present is the one failure mode the strategic spec names outright.
func Run(opt Options) (*evidence.Run, error) {
	if opt.Log == nil {
		opt.Log = io.Discard
	}
	if opt.Lang == "" {
		opt.Lang = "eng"
	}
	m, err := corpus.Load(opt.Manifest)
	if err != nil {
		return nil, err
	}
	scenes, missing := m.Select(opt.Split, opt.SceneIDs)
	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown scene id(s): %v", missing)
	}
	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes selected (split %q)", opt.Split)
	}
	if err := requireIntactMedia(scenes, opt.Root); err != nil {
		return nil, err
	}

	bin, err := ocr.Locate()
	if err != nil {
		return nil, err
	}
	profile := filepath.Join(opt.OutDir, ".browser-profile")
	browser, err := FindBrowser(profile)
	if err != nil {
		return nil, err
	}
	// Headless helpers outlive the launch; leaving them behind after every run would slowly fill
	// the machine with orphaned browsers.
	defer browser.Close()

	runID := filepath.Base(opt.OutDir)
	out := &evidence.Run{
		SchemaVersion: evidence.SchemaVersion,
		RunID:         runID,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		Edition:       evidence.EditionDesktop,
		Engine:        engineFor(bin, opt.Lang),
		Browser:       evidence.Browser{Name: browser.Name, Version: browser.Version},
		Viewports:     Viewports,
	}

	fmt.Fprintf(opt.Log, "ocrlab run: %d scene(s), %s, %s\n", len(scenes), browser.Version, out.Engine.Tesseract)
	for _, s := range scenes {
		sc := runScene(browser, bin, s, opt)
		out.Scenes = append(out.Scenes, sc)
		if sc.Error != "" {
			fmt.Fprintf(opt.Log, "  FAIL %-40s %s\n", s.ID, sc.Error)
			continue
		}
		fmt.Fprintf(opt.Log, "  ok   %-40s %d plate-record(s), ocr %dms\n", s.ID, len(sc.Plates), sc.OcrMs)
	}
	if err := out.Save(filepath.Join(opt.OutDir, EvidenceFile)); err != nil {
		return nil, err
	}
	return out, nil
}

// requireIntactMedia checks only the selected scenes, so working on one scene does not require
// the whole corpus to be present - but a scene that IS selected must be exactly what the
// manifest claims.
func requireIntactMedia(scenes []*corpus.Scene, root string) error {
	var bad []string
	for _, s := range scenes {
		path := s.Path(root)
		if _, err := os.Stat(path); err != nil {
			bad = append(bad, s.ID+": no media at "+path)
			continue
		}
		if s.SHA256 == "" {
			bad = append(bad, s.ID+": no sha256 recorded, so the bytes prove nothing")
			continue
		}
		got, err := corpus.HashFile(path)
		if err != nil {
			bad = append(bad, s.ID+": "+err.Error())
			continue
		}
		if got != s.SHA256 {
			bad = append(bad, s.ID+": hash differs from the manifest")
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("refusing to run on an incomplete corpus:\n  %s", strings.Join(bad, "\n  "))
	}
	return nil
}

// runScene is one scene end to end. Every failure is captured into the scene's Error rather
// than aborting the run, because one broken scene must not cost the other 199.
func runScene(browser *Browser, bin string, s *corpus.Scene, opt Options) evidence.Scene {
	sc := evidence.Scene{SceneID: s.ID}
	fail := func(format string, args ...any) evidence.Scene {
		sc.Error = fmt.Sprintf(format, args...)
		return sc
	}

	workDir := filepath.Join(opt.OutDir, PagesDir, s.ID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fail("workdir: %v", err)
	}

	// The scene is converted the way a user's image is: one page holding the picture, with the
	// OCR overlay applied by the shipped code.
	start := time.Now()
	page, res, err := convertScene(bin, s.Path(opt.Root), workDir, opt)
	if err != nil {
		return fail("convert: %v", err)
	}
	sc.OcrMs = time.Since(start).Milliseconds()
	sc.ImageWidth, sc.ImageHeight = res.Width, res.Height
	if sc.ImageWidth == 0 {
		sc.ImageWidth, sc.ImageHeight = s.Width, s.Height
	}

	probePage, err := injectProbe(page)
	if err != nil {
		return fail("inject probe: %v", err)
	}

	renderStart := time.Now()
	var imageRect *probeRect
	for _, v := range Viewports {
		dom, err := browser.DumpDOM(probePage, "#ocrlab-collect", v)
		if err != nil {
			return fail("render at %s: %v", v.Name, err)
		}
		pr, err := extractProbeResult(dom)
		if err != nil {
			return fail("read probe at %s: %v", v.Name, err)
		}
		if !pr.OK {
			return fail("probe at %s: %s", v.Name, strings.Join(pr.Errors, "; "))
		}
		if v.Name == Viewports[0].Name {
			imageRect = pr.ImageRect
		}
		for _, p := range pr.Plates {
			sc.Plates = append(sc.Plates, evidence.Plate{
				Text:           p.Text,
				Rect:           evidence.Rect{X0: p.Rect.X0, Y0: p.Rect.Y0, X1: p.Rect.X1, Y1: p.Rect.Y1},
				Viewport:       v.Name,
				StressCase:     p.StressCase,
				FontPx:         p.FontPx,
				Background:     p.Background,
				Ink:            p.Ink,
				Mode:           evidence.Mode(p.Mode),
				ModeConfidence: p.ModeConfidence,
				ScrollHeight:   p.ScrollHeight,
				ClientHeight:   p.ClientHeight,
			})
		}
	}

	if err := captureShots(browser, probePage, s, imageRect, sc.ImageWidth, sc.ImageHeight, opt, &sc); err != nil {
		return fail("screenshots: %v", err)
	}
	sc.RenderMs = time.Since(renderStart).Milliseconds()
	sc.PeakRSSBytes = peakRSS()
	return sc
}

// captureShots writes the source image and one render per stress case, each mapped back into
// the source image's pixel space so the concealment measurement can compare them directly.
func captureShots(browser *Browser, page string, s *corpus.Scene, rect *probeRect, w, h int, opt Options, sc *evidence.Scene) error {
	v := Viewports[0] // the primary viewport; drift is measured from geometry, not from pixels
	shots := filepath.Join(opt.OutDir, ShotsDir, s.ID)
	if err := os.MkdirAll(shots, 0o755); err != nil {
		return err
	}

	// The source needs no browser: it is the corpus file itself, copied so the run folder is
	// self-contained.
	srcOut := filepath.Join(shots, "source.png")
	if err := copyFile(s.Path(opt.Root), srcOut); err != nil {
		return err
	}
	sc.Screenshots.Source = relTo(opt.OutDir, srcOut)
	sc.Screenshots.Stress = map[string]string{}

	// No image rect means the page carries no overlay at all - the recognizer found nothing in
	// this scene. That is a measurement, not a malfunction: the scene stays in the run, the
	// metrics report zero recall against its annotation, and the page is still captured so a
	// reviewer can look at what the engine was given. What cannot be done is mapping the
	// screenshot into image space, so the pixel-level dimensions get no sample rather than a
	// wrong one.
	mapped := rect != nil

	for _, c := range StressCases {
		raw := filepath.Join(shots, "raw-"+c.Name+".png")
		if err := browser.Screenshot(page, "#ocrlab-stress="+c.Name, v, raw); err != nil {
			return err
		}
		if !mapped {
			sc.Screenshots.Stress[c.Name] = relTo(opt.OutDir, raw)
			continue
		}
		out := filepath.Join(shots, c.Name+".png")
		if err := CropToImage(raw, rect, v.DeviceScaleFactor, w, h, out); err != nil {
			return err
		}
		_ = os.Remove(raw) // the mapped copy is the one every measurement and the report use
		if c.Name == PrimaryStress {
			sc.Screenshots.Rendered = relTo(opt.OutDir, out)
		}
		sc.Screenshots.Stress[c.Name] = relTo(opt.OutDir, out)
	}
	return nil
}

func relTo(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
