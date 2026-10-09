package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/runner"
)

// Both producers use this local snapshot implementation; no second hashing algorithm.
func cmdDeclare(args []string) error {
	f := flag.NewFlagSet("declare", flag.ContinueOnError)
	p := addCommonFlags(f)
	out := f.String("out", "", "new run directory")
	purpose := f.String("purpose", "exploratory", "exploratory, selected-dev, full-benchmark")
	lang := f.String("lang", "", "tesseract language for every scene (default: each scene's declared language, else eng)")
	var ids stringList
	f.Var(&ids, "scene", "selected scene (repeatable)")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("-out required")
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	scenes, missing := m.Select("all", ids)
	if len(missing) > 0 {
		return fmt.Errorf("unknown scenes: %v", missing)
	}
	_, err = evidence.Freeze(*out, *purpose, p.root, p.annotations, *lang, scenes, runner.Viewports, runner.StressNames(), m)
	return err
}

func snapshotInputs(dir string, p *paths) (*paths, *evidence.Declaration, error) {
	d, err := evidence.LoadDeclaration(dir)
	if err != nil {
		return p, nil, nil
	}
	return &paths{manifest: filepath.Join(dir, "corpus.json"), root: p.root, annotations: filepath.Join(dir, "truth")}, d, nil
}
