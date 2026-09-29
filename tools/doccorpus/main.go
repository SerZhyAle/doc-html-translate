// Command doccorpus is the ticket 68 cross-edition document corpus: the manifest check, the draft
// expectations and the campaign report. The runs themselves are made by run.mjs next to it, which
// drives both editions through one headless browser probe.
//
// It is a developer tool. No build script compiles it and it ships in nothing.
//
//	doccorpus verify           validate the manifest, hash the media, print the coverage matrix
//	doccorpus draft [id..]     write draft expectations from the sources' own bytes
//	doccorpus report <run-dir> grade a run and write report.md / report.json into it
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const usage = `doccorpus - ticket 68 multilingual document corpus

Usage:
  go run ./tools/doccorpus verify
  go run ./tools/doccorpus draft [-force] [case-id..]
  go run ./tools/doccorpus report <run-dir>

Runs are made by: node tools/doccorpus/run.mjs --edition windows|extension (see tools/doccorpus/README.md)

Flags:
  -manifest <path>   default DEV/doccorpus/cases.json
`

// errCouldNotVerify marks "the input was not there", exit 2 - never confused with a judged failure.
var errCouldNotVerify = errors.New("COULD NOT VERIFY")

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	manifestPath := fs.String("manifest", filepath.Join("DEV", "doccorpus", "cases.json"), "manifest")
	force := fs.Bool("force", false, "draft: overwrite existing drafts (never a reviewed expectation)")
	_ = fs.Parse(os.Args[2:])

	m, err := LoadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccorpus:", err)
		os.Exit(2)
	}
	base := filepath.Dir(*manifestPath)
	root := filepath.Join(filepath.Dir(filepath.Dir(base)), m.Root) // DEV/doccorpus -> repo root
	expectDir := filepath.Join(base, "expect")

	switch os.Args[1] {
	case "verify":
		err = cmdVerify(m, root, expectDir)
	case "draft":
		err = cmdDraft(m, root, expectDir, fs.Args(), *force)
	case "report":
		if fs.NArg() != 1 {
			err = fmt.Errorf("report needs one run directory: %w", errCouldNotVerify)
			break
		}
		var md string
		_, md, err = BuildReport(m, expectDir, fs.Arg(0))
		if err == nil {
			fmt.Print(md)
		}
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccorpus "+os.Args[1]+":", err)
		if errors.Is(err, errCouldNotVerify) {
			fmt.Println("doccorpus " + os.Args[1] + ": COULD NOT VERIFY")
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func cmdVerify(m *Manifest, root, expectDir string) error {
	if bad := m.Validate(); len(bad) > 0 {
		for _, b := range bad {
			fmt.Println("  manifest:", b)
		}
		return fmt.Errorf("%d manifest problem(s)", len(bad))
	}
	var missing, changed, badExpect []string
	reviewed, truth := 0, 0
	for _, c := range m.Cases {
		st, err := CheckMedia(root, c)
		if err != nil {
			return err
		}
		switch st {
		case MediaMissing:
			missing = append(missing, c.ID)
		case MediaChanged:
			changed = append(changed, c.ID)
		}
		if c.Rights.HumanReviewed() {
			reviewed++
		}
		e, err := LoadExpectation(filepath.Join(expectDir, c.ID+".json"))
		switch {
		case err != nil:
			badExpect = append(badExpect, c.ID+": "+err.Error())
		case e.CaseID != c.ID || e.SchemaVersion != SchemaVersion:
			badExpect = append(badExpect, c.ID+": caseId or schemaVersion does not match")
		case e.IsTruth():
			truth++
		}
	}
	printMatrix(m)
	fmt.Printf("\ncases: %d; rights reviewed by a person: %d; expectations reviewed by a person: %d\n", len(m.Cases), reviewed, truth)
	for _, id := range changed {
		fmt.Println("  CHANGED:", id)
	}
	for _, id := range missing {
		fmt.Println("  MISSING:", id)
	}
	for _, b := range badExpect {
		fmt.Println("  EXPECTATION:", b)
	}
	if len(changed) > 0 {
		return fmt.Errorf("%d case(s) whose bytes differ from the manifest", len(changed))
	}
	if len(badExpect) > 0 {
		return fmt.Errorf("%d case(s) without a usable expectation", len(badExpect))
	}
	if len(missing) > 0 {
		return fmt.Errorf("%d case(s) have no media on this machine: %w", len(missing), errCouldNotVerify)
	}
	fmt.Println("doccorpus verify: manifest and media consistent")
	return nil
}

func printMatrix(m *Manifest) {
	fmt.Printf("%-16s", "class")
	for _, l := range Languages {
		fmt.Printf(" %-4s", l)
	}
	fmt.Println()
	for _, cl := range Classes {
		fmt.Printf("%-16s", cl)
		for _, l := range Languages {
			n := 0
			for _, c := range m.Cases {
				if c.Class == cl && c.Language == l && c.Role == "primary" {
					n++
				}
			}
			mark := "gap"
			if n > 0 {
				mark = fmt.Sprint(n)
			}
			fmt.Printf(" %-4s", mark)
		}
		fmt.Println()
	}
}

func cmdDraft(m *Manifest, root, expectDir string, ids []string, force bool) error {
	inv, err := loadInventory(root)
	if err != nil {
		return fmt.Errorf("test_doc/INVENTORY.json: %v: %w", err, errCouldNotVerify)
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	if err := os.MkdirAll(expectDir, 0o755); err != nil {
		return err
	}
	for _, c := range m.Cases {
		if len(want) > 0 && !want[c.ID] {
			continue
		}
		p := filepath.Join(expectDir, c.ID+".json")
		if old, err := LoadExpectation(p); err == nil {
			if old.IsTruth() || old.Origin == "human" {
				fmt.Println("  keep (reviewed):", c.ID)
				continue
			}
			if !force {
				fmt.Println("  keep (exists):  ", c.ID)
				continue
			}
		}
		e, err := Draft(root, inv, c)
		if err != nil {
			return err
		}
		raw, err := json.MarshalIndent(e, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(p, append(raw, '\n'), 0o644); err != nil {
			return err
		}
		fmt.Printf("  draft: %-40s pages %d, %d snippet(s), %d lettering line(s) from %s\n",
			c.ID, e.Pages, len(e.Snippets), len(e.Lettering), strings.SplitN(e.DraftSource, ":", 2)[0])
	}
	return nil
}
