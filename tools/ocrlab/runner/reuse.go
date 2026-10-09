package runner

import (
	"fmt"
	"io"
	"path/filepath"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
)

// DefaultReuseRoot is where earlier runs are searched when -reuse-from names nothing.
var DefaultReuseRoot = filepath.Join("temp", "ocrlab")

// CurrentScorerDigest is the scorer digest of the working tree. It is a variable so a test, which
// does not run from the repository root, can supply one.
var CurrentScorerDigest = func() (string, error) { return evidence.ScorerDigest(".") }

// reuser decides, scene by scene, whether an earlier complete run can stand in for collecting.
// Collection is the expensive stage (conversion, recognition, three viewports by six stress cases
// of browser work); scoring, gating and reporting are cheap and always rerun on whatever evidence
// the run ends up with, so a reused scene is judged by the scorer in force now.
type reuser struct {
	finder  *evidence.Finder
	decl    *evidence.Declaration
	engine  evidence.Engine
	browser evidence.Browser
	outDir  string
	log     io.Writer
}

// newReuser prepares the search. It returns nil when -fresh asks for every scene to be collected.
func newReuser(opt Options, decl *evidence.Declaration, engine evidence.Engine, browser evidence.Browser) (*reuser, error) {
	if opt.Fresh {
		return nil, nil
	}
	root := opt.ReuseFrom
	if root == "" {
		root = DefaultReuseRoot
	}
	scorer, err := CurrentScorerDigest()
	if err != nil {
		return nil, fmt.Errorf("scorer digest for reuse: %w", err)
	}
	finder, err := evidence.NewFinder(root, opt.OutDir, scorer)
	if err != nil {
		return nil, fmt.Errorf("search %s for reusable scenes: %w", root, err)
	}
	return &reuser{finder: finder, decl: decl, engine: engine, browser: browser, outDir: opt.OutDir, log: opt.Log}, nil
}

// obtain returns the scene's record and whether it was reused. A scene is collected whenever
// reuse is off, no earlier record is complete and unchanged, or the copy fails.
func (r *reuser) obtain(s *corpus.Scene, opt Options, collect func() evidence.Scene) (evidence.Scene, bool) {
	if r == nil {
		return collect(), false
	}
	key, err := evidence.NewKey(evidence.EditionDesktop, r.decl, s.ID, r.engine, r.browser)
	if err != nil {
		fmt.Fprintf(r.log, "  fresh  %s: cannot be reused: %v\n", s.ID, err)
		return collect(), false
	}
	found, why := r.finder.Find(key, s.ID)
	if found == nil {
		fmt.Fprintf(r.log, "  fresh  %s: %s\n", s.ID, why)
		return collect(), false
	}
	sc, err := evidence.Apply(found, r.outDir)
	if err != nil {
		fmt.Fprintf(r.log, "  fresh  %s: copy from %s failed: %v\n", s.ID, filepath.ToSlash(found.Bundle), err)
		return collect(), false
	}
	// The earlier record may have been read under another flag for the same language; this run's
	// choice is the one the declaration froze.
	sc.LangSource = evidence.SceneLang(opt.Annotations, opt.Lang, s).Source
	fmt.Fprintf(r.log, "  reused %s from %s\n", s.ID, filepath.ToSlash(found.Bundle))
	return sc, true
}
