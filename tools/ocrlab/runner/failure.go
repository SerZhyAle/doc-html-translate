package runner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
)

// pathLimit is the longest path, in characters, a run may produce; 0 means no limit. It is the
// OCR engine's own limit, read from the package that stages around it, so the lab cannot drift
// from the app. A variable so a test can inject another limit.
var pathLimit = ocr.EnginePathLimit

// classifyFailure decides why a scene failed from the file system alone. The order is the
// diagnosis: an input that is not there explains everything else; a path past the limit explains
// a helper that could not open it; anything left is the engine's own failure. The error text is
// never read, because it is a flat string that says "cannot read input file" for both.
func classifyFailure(input string, pathChars int) evidence.ErrorKind {
	if _, err := os.Stat(input); err != nil {
		return evidence.ErrInputMissing
	}
	if limit := pathLimit(); limit > 0 && pathChars > limit {
		return evidence.ErrEnginePathIncompatible
	}
	return evidence.ErrEngineFailed
}

// scenePathChars is the longest absolute path, in characters, among the scene's input and the
// files the run wrote for it so far (pages and screenshots).
func scenePathChars(input string, dirs ...string) int {
	longest := absLen(input)
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
			if err == nil {
				longest = max(longest, absLen(p))
			}
			return nil
		})
	}
	return longest
}

func absLen(p string) int {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return len(p)
}

// pageHTMLName and pageLabName are the pages the browser opens for a one-image scene; the image
// copy sits beside them and is loaded by the page.
const (
	pageHTMLName = "page_001.html"
	pageLabName  = "page_001.ocrlab.html"
)

// requirePathBudget refuses a run whose browser-loaded files would sit on too long a path. The
// browser (Chrome and Edge alike) opens a page and its image from a file: URL, and on a path of
// 260 characters or more it renders the image as corrupt noise without any error - a scene that
// looks collected and is not (measured on the Brandeis cartoon, 2026-10-09: the same scene on a
// shorter -out root is correct). The OCR engine no longer matters here, because internal/ocr stages
// a long path itself. Screenshots are written by this program, which handles long paths, so they
// are not counted. The names are known before the run starts, so a root that is too deep is
// reported once, with the limit, instead of producing bad evidence.
func requirePathBudget(scenes []*corpus.Scene, outDir string) error {
	limit := pathLimit()
	if limit <= 0 {
		return nil
	}
	root, err := filepath.Abs(outDir)
	if err != nil {
		root = outDir
	}
	type over struct {
		path string
		n    int
	}
	var bad []over
	for _, s := range scenes {
		page := filepath.Join(root, PagesDir, s.ID)
		for _, p := range []string{
			filepath.Join(page, filepath.Base(filepath.FromSlash(s.File))),
			filepath.Join(page, pageHTMLName),
			filepath.Join(page, pageLabName),
		} {
			if len(p) > limit {
				bad = append(bad, over{p, len(p)})
			}
		}
	}
	if len(bad) == 0 {
		return nil
	}
	sort.Slice(bad, func(i, j int) bool { return bad[i].n > bad[j].n })
	return fmt.Errorf("refusing to run: %d browser-loaded path(s) exceed the supported limit of %d characters "+
		"(longest %d: %s); choose a shorter -out directory", len(bad), limit, bad[0].n, bad[0].path)
}

// failureFor records a failed scene: the reason, its kind and the path length that decided it.
func failureFor(sc *evidence.Scene, input, outDir string, msg string) {
	sc.Error = msg
	sc.PathChars = scenePathChars(input,
		filepath.Join(outDir, PagesDir, sc.SceneID), filepath.Join(outDir, ShotsDir, sc.SceneID))
	sc.ErrorKind = classifyFailure(input, sc.PathChars)
}
