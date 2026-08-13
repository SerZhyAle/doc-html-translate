// Command ocrlab is the OCR visual-fidelity lab: the instrument the strategic spec
// DEV/plan/2026-08-11_ocr-visual-fidelity-lab.md requires before any OCR or redraw change is
// accepted.
//
// It is a developer tool. No build script compiles it, it ships in no package, and it is not
// linked into doc-html-translate or doc-html-ui - it only reads what those produce.
//
//	ocrlab verify              validate the manifest, the media hashes and the coverage table
//	ocrlab fetch [id..]        idempotently download licence-verified media
//	ocrlab synth               regenerate the deterministic diagnostic scenes
//	ocrlab seed <id..>         write an OCR-seeded annotation draft for a human to correct
//	ocrlab run                 convert, render and record evidence
//	ocrlab score <dir>         grade a saved run offline
//	ocrlab report <dir>        render the reviewable report
//	ocrlab gate <dir>          judge a scored run against the acceptance thresholds
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

const usage = `ocrlab - OCR visual-fidelity lab

Usage:
  ocrlab <command> [flags]

Commands:
  verify        validate the manifest, media hashes, annotations and coverage
  fetch [id..]  download media for licence-verified scenes (idempotent; all scenes if no id)
  add <img..>   register your own images (or any local file) as corpus scenes
  harvest       pull licence-checked images from Wikimedia Commons by search or category
  synth         regenerate the deterministic diagnostic scenes and their exact annotations
  seed <id..>   write an OCR-seeded annotation draft (never counts as truth)
  run           convert, render and record evidence for the selected scenes
  score <dir>   grade a saved run offline against the annotations
  report <dir>  render report.md and a side-by-side report.html
  gate <dir>    judge a scored run against DEV/ocrlab/thresholds.json (exit 1 on FAIL)

Common flags:
  -manifest <path>      default DEV/ocrlab/corpus.json
  -root <path>          corpus media root, default test_doc/ocrlab
  -annotations <path>   default DEV/ocrlab/annotations
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "verify":
		err = cmdVerify(args)
	case "fetch":
		err = cmdFetch(args)
	case "add":
		err = cmdAdd(args)
	case "harvest":
		err = cmdHarvest(args)
	case "synth":
		err = cmdSynth(args)
	case "seed":
		err = cmdSeed(args)
	case "run":
		err = cmdRun(args)
	case "score":
		err = cmdScore(args)
	case "report":
		err = cmdReport(args)
	case "gate":
		err = cmdGate(args)
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ocrlab "+cmd+": "+err.Error())
		os.Exit(1)
	}
}

// paths holds the two locations every subcommand needs. Defaults are repo-relative, so the
// tool is run from the repository root like every other script here.
type paths struct {
	manifest    string
	root        string
	annotations string
}

func addCommonFlags(fs *flag.FlagSet) *paths {
	p := &paths{}
	fs.StringVar(&p.manifest, "manifest", filepath.Join("DEV", "ocrlab", "corpus.json"), "path to the corpus manifest")
	fs.StringVar(&p.root, "root", filepath.Join("test_doc", "ocrlab"), "corpus media root (gitignored)")
	fs.StringVar(&p.annotations, "annotations", filepath.Join("DEV", "ocrlab", "annotations"), "ground-truth directory")
	return p
}

// cmdVerify validates and prints. It exits non-zero on any problem, so it can gate a build,
// and it always prints the coverage table - a corpus that is merely thin should be visible
// even on a run with no rule violations.
func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	p := addCommonFlags(fs)
	strict := fs.Bool("strict", false, "also fail on coverage shortfalls (the full section 4.1 targets)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	anns, err := truth.LoadDir(p.annotations)
	if err != nil {
		return err
	}
	defects, shortfalls := corpus.SplitProblems(corpus.Validate(m, p.root))
	truthProblems := truth.ValidateAll(anns, m)
	printCoverage(m, anns)

	fmt.Printf("\ncorpus defects (%d) - these stop a run:\n", len(defects))
	for _, pr := range defects {
		fmt.Println("  " + pr.String())
	}
	fmt.Printf("\nannotation problems (%d):\n", len(truthProblems))
	for _, pr := range truthProblems {
		fmt.Println("  " + pr.String())
	}
	fmt.Printf("\ncoverage shortfalls (%d) - progress against the target, not errors:\n", len(shortfalls))
	for _, pr := range shortfalls {
		fmt.Println("  " + pr.String())
	}

	// Coverage never sets the exit code unless asked. A corpus grows one harvest at a time, and
	// a gate that fails until it is finished would stop the measure-look-fix loop from ever
	// starting - which is the opposite of what the targets are for. `-strict` is there for the
	// day the corpus is meant to be complete.
	failing := len(defects) + len(truthProblems)
	if *strict {
		failing += len(shortfalls)
	}
	if failing == 0 {
		fmt.Printf("\nverify: %d scene(s), nothing blocking\n", len(m.Scenes))
		return nil
	}
	os.Exit(1)
	return nil
}

// printCoverage renders the section 4.1 table against reality: what the corpus has, what it
// needs, and how the holdout is balanced.
func printCoverage(m *corpus.Manifest, anns map[string]*truth.Annotation) {
	c := corpus.Counts(m)
	fmt.Printf("scenes: %d total, %d scored, %d dev, %d holdout\n",
		c.Total, c.Scored, c.BySplit[corpus.SplitDev], c.BySplit[corpus.SplitHoldout])

	// How many scored scenes actually have usable truth. Printing this next to the scene count
	// is the difference between "200 scenes" and "200 scenes, 12 of which can be scored".
	var gradable int
	for i := range m.Scenes {
		s := &m.Scenes[i]
		if s.Scored() && anns[s.ID].IsTruth() {
			gradable++
		}
	}
	fmt.Printf("truth:  %d annotation(s) on disk, %d scene(s) gradable\n", len(anns), gradable)
	fmt.Printf("\n%-10s %8s %8s %10s %10s\n", "category", "have", "need", "holdout", "share")
	for _, cat := range corpus.Categories() {
		have := c.ByCategory[cat]
		need := corpus.CategoryMinimum(cat)
		ho := c.HoldoutByCategory[cat]
		share := "-"
		if h := c.BySplit[corpus.SplitHoldout]; h > 0 {
			share = fmt.Sprintf("%.0f%%", float64(ho)*100/float64(h))
		}
		mark := " "
		if have < need {
			mark = "!"
		}
		fmt.Printf("%-10s %7d%s %8d %10d %10s\n", cat, have, mark, need, ho, share)
	}

	licences := make([]string, 0, len(c.ByLicence))
	for l := range c.ByLicence {
		licences = append(licences, string(l))
	}
	sort.Strings(licences)
	fmt.Printf("\nlicences: ")
	for i, l := range licences {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%s=%d", l, c.ByLicence[corpus.Licence(l)])
	}
	fmt.Println()
}

func cmdFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	p := addCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	_, err = corpus.Fetch(m, p.root, fs.Args(), os.Stdout)
	return err
}
