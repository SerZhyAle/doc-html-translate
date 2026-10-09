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
	Manifest    string
	Root        string
	OutDir      string
	Split       string
	SceneIDs    []string
	Annotations string
	Purpose     string
	Lang        string
	Log         io.Writer
	// Fresh collects every scene even when an earlier complete run could stand in for it.
	Fresh bool
	// ReuseFrom limits the search for reusable scenes to one run directory, or one folder of
	// runs; empty searches DefaultReuseRoot.
	ReuseFrom string
}

// Layout of a run directory. Everything is relative to OutDir so the folder can be moved,
// zipped or attached to a report and still open.
const (
	EvidenceFile = "evidence.json"
	ScoresFile   = "scores.json"
	SummaryFile  = "summary.json"
	SelfFile     = "selftest.json"
	DiagFile     = evidence.DiagFile
	ShotsDir     = evidence.ShotsDir
	PagesDir     = evidence.PagesDir
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
	if err := requirePathBudget(scenes, opt.OutDir); err != nil {
		return nil, err
	}

	if opt.Annotations == "" {
		opt.Annotations = "DEV/ocrlab/annotations"
	}
	if opt.Purpose == "" {
		opt.Purpose = "exploratory"
	}
	decl, err := evidence.Freeze(opt.OutDir, opt.Purpose, opt.Root, opt.Annotations, opt.Lang, scenes, Viewports, StressNames(), m)
	if err != nil {
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
	langLabel, langPacks := runLangs(scenes, opt)
	out := &evidence.Run{
		SchemaVersion: evidence.SchemaVersion,
		RunID:         runID,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		Edition:       evidence.EditionDesktop,
		Engine:        engineFor(bin, langLabel, langPacks),
		Browser:       evidence.Browser{Name: browser.Name, Version: browser.Version},
		Viewports:     Viewports,
	}

	reuse, err := newReuser(opt, decl, out.Engine, out.Browser)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(opt.Log, "ocrlab run: %d scene(s), %s, %s\n", len(scenes), browser.Version, out.Engine.Tesseract)
	reused := 0
	for _, s := range scenes {
		sc, wasReused := reuse.obtain(s, opt, func() evidence.Scene { return runScene(browser, bin, s, opt) })
		out.Scenes = append(out.Scenes, sc)
		if wasReused {
			reused++
			continue
		}
		if sc.Error != "" {
			fmt.Fprintf(opt.Log, "  FAIL %-40s [%s] %s\n", s.ID, sc.ErrorKind, sc.Error)
			continue
		}
		if sc.Unmeasured != "" {
			fmt.Fprintf(opt.Log, "  skip %-40s unmeasured: %s\n", s.ID, sc.Unmeasured)
			continue
		}
		fmt.Fprintf(opt.Log, "  ok   %-40s %-8s %d plate-record(s), ocr %dms\n", s.ID, sc.Lang, len(sc.Plates), sc.OcrMs)
	}
	fmt.Fprintf(opt.Log, "ocrlab run: collected %d, reused %d of %d scene(s)\n", len(scenes)-reused, reused, len(scenes))
	if err := out.Save(filepath.Join(opt.OutDir, EvidenceFile)); err != nil {
		return nil, err
	}
	if err := evidence.Finish(opt.OutDir); err != nil {
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
		failureFor(&sc, s.Path(opt.Root), opt.OutDir, fmt.Sprintf(format, args...))
		return sc
	}

	choice := evidence.SceneLang(opt.Annotations, opt.Lang, s)
	sc.Lang, sc.LangSource = choice.Lang, choice.Source
	if missing := missingLangData(choice.Lang); len(missing) > 0 {
		sc.ImageWidth, sc.ImageHeight = s.Width, s.Height
		sc.Unmeasured = "language data unavailable: " + strings.Join(missing, "+")
		return sc
	}

	workDir := filepath.Join(opt.OutDir, PagesDir, s.ID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fail("workdir: %v", err)
	}

	// The scene is converted the way a user's image is: one page holding the picture, with the
	// OCR overlay applied by the shipped code.
	start := time.Now()
	page, res, err := convertScene(bin, s.Path(opt.Root), workDir, choice.Lang, opt)
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
				ScrollWidth:    p.ScrollWidth, ClientWidth: p.ClientWidth,
			})
		}
	}

	if err := captureShots(browser, probePage, s, imageRect, sc.ImageWidth, sc.ImageHeight, opt, &sc); err != nil {
		return fail("screenshots: %v", err)
	}
	sc.RenderMs = time.Since(renderStart).Milliseconds()
	sc.MemoryKind = "go-runtime-sys-snapshot"
	sc.MemoryBytes = runtimeSys()
	return sc
}

// captureShots writes the source image and one render per stress case, each mapped back into
// the source image's pixel space so the concealment measurement can compare them directly.
func captureShots(browser *Browser, page string, s *corpus.Scene, rect *probeRect, w, h int, opt Options, sc *evidence.Scene) error {
	shots := filepath.Join(opt.OutDir, ShotsDir, s.ID)
	if err := os.MkdirAll(shots, 0755); err != nil {
		return err
	}
	source := filepath.Join(shots, "source.png")
	if err := copyFile(s.Path(opt.Root), source); err != nil {
		return err
	}
	sc.Screenshots.Source = relTo(opt.OutDir, source)
	sc.Screenshots.Stress = map[string]string{}
	for _, v := range Viewports {
		for _, c := range StressCases {
			dom, err := browser.DumpDOM(page, "#ocrlab-stress="+c.Name, v)
			if err != nil {
				return err
			}
			normal, err := extractProbeResult(dom)
			if err != nil {
				return err
			}
			if !normal.OK || normal.ImageRect == nil {
				return fmt.Errorf("missing image or failed observation %s/%s", v.Name, c.Name)
			}
			obs := evidence.Observation{Viewport: v.Name, StressCase: c.Name}
			for _, hidden := range []bool{false, true} {
				name := v.Name + "-" + c.Name
				fragment := "#ocrlab-stress=" + c.Name
				if hidden {
					name += "-hidden"
					fragment += "-hidden"
					dom, err := browser.DumpDOM(page, fragment, v)
					if err != nil {
						return err
					}
					diagnostic, err := extractProbeResult(dom)
					if err != nil {
						return err
					}
					if !diagnostic.OK || diagnostic.ImageRect == nil || *diagnostic.ImageRect != *normal.ImageRect || len(diagnostic.Plates) != len(normal.Plates) {
						return fmt.Errorf("diagnostic capture changed layout")
					}
					for i, p := range diagnostic.Plates {
						if p.Rect != normal.Plates[i].Rect || p.FontPx != normal.Plates[i].FontPx || p.ScrollWidth != normal.Plates[i].ScrollWidth || p.ScrollHeight != normal.Plates[i].ScrollHeight {
							return fmt.Errorf("diagnostic capture changed plate layout")
						}
					}
				}
				out := filepath.Join(shots, name+".png")
				if err := browser.ImageScreenshot(page, fragment, v, normal.ImageRect, w, h, out); err != nil {
					return err
				}
				if hidden {
					obs.Concealed = relTo(opt.OutDir, out)
				} else {
					obs.Rendered = relTo(opt.OutDir, out)
				}
			}
			sc.Observations = append(sc.Observations, obs)
			if v.Name == Viewports[0].Name {
				sc.Screenshots.Stress[c.Name] = obs.Rendered
				if c.Name == PrimaryStress {
					sc.Screenshots.Rendered = obs.Rendered
				}
			}
		}
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
