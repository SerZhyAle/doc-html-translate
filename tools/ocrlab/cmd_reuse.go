package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/runner"
)

// cmdReuse is the extension producer's door to scene reuse. The lookup, the completeness rule and
// the copy live once, in package evidence, so both editions apply the same rule and write the same
// reusedFrom field; the Node producer only supplies what it alone knows (the engine and browser
// it drives) and merges the record this prints.
//
// On a hit it copies the scene's captures and diagnostics lines into the new run and prints the
// scene record as one JSON line. On a miss it prints "null" and the reason on stderr, and still
// exits 0: not being able to reuse is an ordinary outcome, and the producer then collects.
func cmdReuse(args []string) error { return runReuse(args, os.Stdout, os.Stderr) }

func runReuse(args []string, stdout, stderr io.Writer) error {
	f := flag.NewFlagSet("reuse", flag.ContinueOnError)
	runDir := f.String("run", "", "the new run directory, already declared")
	edition := f.String("edition", "", "desktop or extension")
	scene := f.String("scene", "", "scene id")
	tesseract := f.String("tesseract", "", "the OCR engine's version, as the run records it")
	tessdata := f.String("tessdata", "", "the language data identity, as the run records it")
	browserName := f.String("browser-name", "", "the browser's name, as the run records it")
	browserVersion := f.String("browser-version", "", "the browser's version, as the run records it")
	reuseFrom := f.String("reuse-from", "", "search only this run directory (or folder of runs); default temp/ocrlab")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *runDir == "" || *scene == "" {
		return fmt.Errorf("reuse requires -run and -scene")
	}
	ed := evidence.Edition(*edition)
	if ed != evidence.EditionDesktop && ed != evidence.EditionExtension {
		return fmt.Errorf("-edition must be %q or %q", evidence.EditionDesktop, evidence.EditionExtension)
	}

	decl, err := evidence.LoadDeclaration(*runDir)
	if err != nil {
		return fmt.Errorf("the new run is not declared: %w", err)
	}
	miss := func(format string, a ...any) error {
		fmt.Fprintf(stderr, "no reuse for %s: %s\n", *scene, fmt.Sprintf(format, a...))
		_, err := fmt.Fprintln(stdout, "null")
		return err
	}
	key, err := evidence.NewKey(ed, decl, *scene,
		evidence.Engine{Tesseract: *tesseract, TessdataVersion: *tessdata},
		evidence.Browser{Name: *browserName, Version: *browserVersion})
	if err != nil {
		return miss("%v", err)
	}
	root := *reuseFrom
	if root == "" {
		root = runner.DefaultReuseRoot
	}
	scorer, err := runner.CurrentScorerDigest()
	if err != nil {
		return err
	}
	found, why, err := evidence.FindReusable(root, *runDir, scorer, key, *scene)
	if err != nil {
		return err
	}
	if found == nil {
		return miss("%s", why)
	}
	sc, err := evidence.Apply(found, *runDir)
	if err != nil {
		return miss("copy from %s failed: %v", found.Bundle, err)
	}
	buf, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "reused %s from %s\n", *scene, found.From.Bundle)
	_, err = fmt.Fprintln(stdout, string(buf))
	return err
}
