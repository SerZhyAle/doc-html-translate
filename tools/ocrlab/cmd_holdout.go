package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/holdout"
	"doc-html-translate/tools/ocrlab/truth"
)

// The holdout lifecycle: freeze its membership before tuning, record every acceptance use, and
// make any later replacement a visible decision. The record is DEV/ocrlab/holdout.json, and it
// is written by these commands only - never by editing, never by a run.

func cmdHoldout(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ocrlab holdout freeze|used|replace|audit [flags]")
	}
	switch args[0] {
	case "freeze":
		return cmdHoldoutFreeze(args[1:])
	case "used":
		return cmdHoldoutUsed(args[1:])
	case "replace":
		return cmdHoldoutReplace(args[1:])
	case "audit":
		return cmdHoldoutAudit(args[1:])
	default:
		return fmt.Errorf("unknown holdout subcommand %q (freeze, used, replace, audit)", args[0])
	}
}

func holdoutFlags(fs *flag.FlagSet, p **paths) {
	*p = addCommonFlags(fs)
	fs.StringVar(&(*p).holdout, "record", filepath.Join("DEV", "ocrlab", "holdout.json"), "holdout freeze record")
}

// cmdHoldoutFreeze writes the record from the corpus as it stands. It refuses on any ineligible
// candidate, listing every reason, because a freeze that quietly skipped a scene would read as
// complete. The current corpus has no holdout scenes yet, so today this reports exactly that.
func cmdHoldoutFreeze(args []string) error {
	fs := flag.NewFlagSet("holdout freeze", flag.ExitOnError)
	var p *paths
	holdoutFlags(fs, &p)
	by := fs.String("by", "", "the operator freezing the holdout (required)")
	on := fs.String("on", time.Now().UTC().Format("2006-01-02"), "freeze date, YYYY-MM-DD")
	runs := fs.String("runs", filepath.Join("temp", "ocrlab"), "run history scanned for prior judged use")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return cannotVerify(err)
	}
	anns, err := truth.LoadDir(p.annotations)
	if err != nil {
		return cannotVerify(err)
	}
	history, err := holdout.ScanRuns(*runs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ocrlab holdout freeze: run history unavailable, freezing without it: %v\n", err)
		history = nil
	}
	f, err := holdout.Build(m, anns, history, *by, *on)
	if err != nil {
		return err
	}
	if err := holdout.Save(p.holdout, f); err != nil {
		return err
	}
	fmt.Printf("ocrlab holdout freeze: %d scene(s), frozen on %s by %s; %s\n",
		len(f.Scenes), f.FrozenOn, f.FrozenBy, p.holdout)
	return nil
}

// cmdHoldoutUsed records one acceptance use. The run must exist, declare the full benchmark and
// be scored: a use ledger entry that names nothing is a claim, not a record.
func cmdHoldoutUsed(args []string) error {
	fs := flag.NewFlagSet("holdout used", flag.ExitOnError)
	var p *paths
	holdoutFlags(fs, &p)
	run := fs.String("run", "", "the judged run directory to record (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *run == "" {
		return errors.New("holdout used requires -run <run-dir>")
	}
	var summary struct {
		RunID   string `json:"runId"`
		Purpose string `json:"purpose"`
	}
	if err := readJSON(filepath.Join(*run, "summary.json"), &summary); err != nil {
		return cannotVerify(fmt.Errorf("run %s is not scored: %w (judge it with ocrlab score first)", *run, err))
	}
	if summary.Purpose != "full-benchmark" {
		return fmt.Errorf("run %s declares purpose %q; the holdout is used for acceptance, not iterative parameter selection", summary.RunID, summary.Purpose)
	}
	f, err := holdout.Load(p.holdout)
	if err != nil {
		return cannotVerify(err)
	}
	for _, u := range f.Use {
		if u.Run == summary.RunID {
			return fmt.Errorf("run %s is already recorded as used on %s", summary.RunID, u.On)
		}
	}
	f.Use = append(f.Use, holdout.Use{Run: summary.RunID, On: time.Now().UTC().Format("2006-01-02"), Purpose: "full-benchmark"})
	if err := holdout.Save(p.holdout, f); err != nil {
		return err
	}
	fmt.Printf("ocrlab holdout used: run %s recorded, %d use(s) total\n", summary.RunID, len(f.Use))
	return nil
}

// cmdHoldoutReplace records one membership change with its reason and operator. An addition is
// re-judged by the same eligibility rules as a freeze, so a scene that was tuned meanwhile
// cannot slip in through the side door.
func cmdHoldoutReplace(args []string) error {
	fs := flag.NewFlagSet("holdout replace", flag.ExitOnError)
	var p *paths
	holdoutFlags(fs, &p)
	add := fs.String("add", "", "scene id entering the frozen holdout")
	remove := fs.String("remove", "", "scene id leaving the frozen holdout")
	reason := fs.String("reason", "", "why the membership changed (required)")
	by := fs.String("by", "", "the operator making the change (required)")
	runs := fs.String("runs", filepath.Join("temp", "ocrlab"), "run history scanned for prior judged use")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*add == "") == (*remove == "") {
		return errors.New("holdout replace needs exactly one of -add or -remove")
	}
	if *reason == "" || *by == "" {
		return errors.New("holdout replace requires -reason and -by; an unexplained membership change is a quiet swap")
	}
	f, err := holdout.Load(p.holdout)
	if err != nil {
		return cannotVerify(err)
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return cannotVerify(err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	rep := holdout.Replacement{On: today, By: *by, Reason: *reason}
	if *add != "" {
		s := m.Find(*add)
		if s == nil {
			return fmt.Errorf("scene %q is not in the manifest", *add)
		}
		if s.Split != corpus.SplitHoldout {
			return fmt.Errorf("scene %s is in the %s split; move it to the holdout in the manifest first", *add, s.Split)
		}
		anns, err := truth.LoadDir(p.annotations)
		if err != nil {
			return cannotVerify(err)
		}
		history, err := holdout.ScanRuns(*runs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ocrlab holdout replace: run history unavailable: %v\n", err)
			history = nil
		}
		// Eligibility is judged against the whole manifest so family conflicts are seen, and
		// only this scene's refusals block the addition.
		_, refusals := holdout.Eligibility(m, anns, history)
		var mine []string
		for _, r := range refusals {
			if r.SceneID == *add {
				mine = append(mine, r.String())
			}
		}
		if len(mine) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "%s cannot enter the frozen holdout:", *add)
			for _, r := range mine {
				fmt.Fprintf(&b, "\n  %s", r)
			}
			return errors.New(b.String())
		}
		f.Scenes[*add] = holdout.FrozenScene{SHA256: s.SHA256, Licence: s.Licence, Categories: s.Categories}
		rep.SceneID, rep.Added = *add, true
	} else {
		if _, ok := f.Scenes[*remove]; !ok {
			return fmt.Errorf("scene %s is not in the frozen holdout", *remove)
		}
		delete(f.Scenes, *remove)
		rep.SceneID = *remove
	}
	f.Replacements = append(f.Replacements, rep)
	if err := holdout.Save(p.holdout, f); err != nil {
		return err
	}
	fmt.Printf("ocrlab holdout replace: %s %s, reason recorded; %d scene(s) frozen\n",
		rep.SceneID, map[bool]string{true: "added", false: "removed"}[rep.Added], len(f.Scenes))
	return nil
}

// cmdHoldoutAudit is the standing integrity check over the record, the corpus and the local run
// history: membership drift, unrecorded acceptance use, holdout scenes used in development
// selections, and replacement bookkeeping. Exit 1 on any problem.
func cmdHoldoutAudit(args []string) error {
	fs := flag.NewFlagSet("holdout audit", flag.ExitOnError)
	var p *paths
	holdoutFlags(fs, &p)
	runs := fs.String("runs", filepath.Join("temp", "ocrlab"), "run history to audit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	f, err := holdout.Load(p.holdout)
	if err != nil {
		return cannotVerify(err)
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return cannotVerify(err)
	}
	history, err := holdout.ScanRuns(*runs)
	if err != nil {
		return cannotVerify(err)
	}
	problems := holdout.Audit(m, f, history)
	fmt.Printf("ocrlab holdout audit: subject = %s, frozen on %s by %s, %d scene(s), %d recorded use(s), %d replacement(s)\n\n",
		p.holdout, f.FrozenOn, f.FrozenBy, len(f.Scenes), len(f.Use), len(f.Replacements))
	if len(problems) > 0 {
		fmt.Printf("holdout problems (%d):\n", len(problems))
		for _, pr := range problems {
			fmt.Println("  " + pr.String())
		}
		fmt.Printf("\nocrlab holdout audit: FAIL (%d)\n", len(problems))
		os.Exit(1)
	}
	fmt.Printf("ocrlab holdout audit: PASS (%d scene(s), no drift, every judged use recorded)\n", len(f.Scenes))
	return nil
}

// holdoutGateIssues is the gate-side check: a full-benchmark verdict may only be produced over
// a frozen holdout whose membership matches what the run actually judged.
func holdoutGateIssues(record string, purpose, runDir string) []string {
	if purpose != "full-benchmark" {
		return nil
	}
	f, err := holdout.Load(record)
	if err != nil {
		return []string{"holdout freeze unavailable: " + err.Error()}
	}
	frozen, err := corpus.Load(filepath.Join(runDir, "corpus.json"))
	if err != nil {
		return []string{"run corpus snapshot unavailable: " + err.Error()}
	}
	var out []string
	for _, issue := range holdout.MembershipIssues(f, frozen) {
		out = append(out, "holdout membership: "+issue)
	}
	return out
}
