// selftest.go implements -SelfTest (strategic section 5.4): matrix integrity, capability
// coverage against CLAUDE.md/AGENTS.md, fixture presence. Exit 0 pass, 1 a gap, 2 could not
// verify (a derivation source or the fixture root is unreadable), 3 pass with advisories.

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// selftestResult carries what main.go turns into the verdict and the exit code.
type selftestResult struct {
	verdict verdict
	detail  string
	defects []error
	notes   []string
}

// selfTestChecks is the fixed list the self-test runs; each is one test of its verdict line.
const selfTestChecks = 3

// runSelfTest performs the three checks and prints what it found. It never stops at the first
// failure - the verdict line at the end carries the whole count. CHECK-VERDICT 0.14 (amendment
// 2026-10-29 item A1): a self-test prints a test line, run=N skipped=M, each check one test.
func runSelfTest(env *runEnv) selftestResult {
	res := selftestResult{verdict: verdictPass}
	ran := 0
	counts := func() string { return fmt.Sprintf("run=%d skipped=%d", ran, selfTestChecks-ran) }

	// 1. Matrix integrity.
	ran++
	if errs := validateMatrix(); len(errs) > 0 {
		res.defects = append(res.defects, errs...)
	}

	// 2. Capability coverage. A missing derivation source is could-not-verify, never a pass.
	claude, agents, roster, err := loadCapabilitySources()
	if err != nil {
		res.verdict = verdictCouldNotVerify
		res.detail = counts() + ", " + err.Error()
		return res
	}
	ran++
	capDefects, capNotes := checkCapabilities(claude, agents, roster)
	res.defects = append(res.defects, capDefects...)
	res.notes = append(res.notes, capNotes...)

	// 3. Fixture presence. A fixture root never generated is an absent input (the ocrlab
	// precedent: COULD NOT VERIFY); a fixture the manifest no longer matches is a gap.
	missing, corrupt, err := checkFixtures(env.fixturesDir)
	if err != nil {
		res.verdict = verdictCouldNotVerify
		res.detail = counts() + ", " + err.Error()
		return res
	}
	if _, statErr := os.Stat(env.fixturesDir); statErr != nil && len(missing) == len(smallFixtures()) {
		res.verdict = verdictCouldNotVerify
		res.detail = counts() + fmt.Sprintf(", fixture root %s absent; run -Fixtures first", env.fixturesDir)
		return res
	}
	ran++
	for _, id := range missing {
		res.defects = append(res.defects, fmt.Errorf("fixture %s missing (run -Fixtures)", id))
	}
	for _, id := range corrupt {
		res.defects = append(res.defects, fmt.Errorf("fixture %s no longer matches its manifest hash", id))
	}

	// The verdict, from everything found. Every defect is named above the line; the detail
	// carries only the counts.
	res.detail = counts()
	switch {
	case len(res.defects) > 0:
		res.verdict = verdictFail
		res.detail += fmt.Sprintf(" defects=%d", len(res.defects))
	case len(res.notes) > 0:
		res.verdict = verdictAdvisory
		res.detail += fmt.Sprintf(" advisories=%d", len(res.notes))
	}
	return res
}

// smallFixtures lists the non-big catalog ids (the self-test's required set).
func smallFixtures() []string {
	var out []string
	for _, f := range fixtureCatalog {
		if !f.big {
			out = append(out, f.id)
		}
	}
	return out
}

// printMatrixState renders the current cell states the way -Summary does - the self-test shows
// the honest state of every applicable cell next to its integrity verdict.
func printMatrixState(recs []*record) {
	fmt.Println()
	fmt.Printf("%-5s %-22s %-10s %-10s %s\n", "scene", "title", "property", "state", "budgets / closing run")
	for _, s := range scenes {
		for _, c := range matrixCellsFor(s.id) {
			state := string(c.state)
			ref := c.na
			if c.state == cellOpen || c.state == cellMeasured {
				ref = joinBudgets(c.budgets)
			}
			if rec := closingRecord(recs, s.id, c.prop); rec != nil {
				state = "measured"
				ref += " <- " + rec.RunID
			}
			fmt.Printf("%-5s %-22s P%-9d %-10s %s\n", s.id, s.title, c.prop, state, ref)
		}
	}
}

func joinBudgets(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}

// closingRecord returns the newest valid scene record that measured this cell.
func closingRecord(recs []*record, id sceneID, prop int) *record {
	var best *record
	for _, r := range recs {
		if r.Kind != "scene" || !r.Valid || r.Scene != string(id) {
			continue
		}
		for _, c := range r.Cells {
			if c.Scene == string(id) && c.Property == prop && c.State == "measured" {
				if !cellEvidenceComplete(recs, r, c) {
					continue
				}
				if best == nil || r.CreatedAt.After(best.CreatedAt) {
					best = r
				}
			}
		}
	}
	return best
}

func cellEvidenceComplete(recs []*record, r *record, c cellResult) bool {
	if len(r.InvalidReasons) > 0 || r.Isolation != "B-full" || !r.RestoreVerified || r.RecoverySHA256 == "" || r.TreeBefore == "" || r.TreeBefore != r.TreeAfter || c.Evidence == "" {
		return false
	}
	cell, ok := cellFor(sceneID(c.Scene), c.Property)
	if !ok || cell.state == cellNA {
		return false
	}
	for _, id := range cell.budgets {
		found := false
		for _, b := range c.Budgets {
			if b.ID == id && (b.Verdict == "pass" || b.Verdict == "fail") && b.Observed != "" && b.Limit != "" {
				if id == "H1" || id == "H2" || id == "H3" || id == "H4" || id == "H5" || id == "H6" || id == "H7" || id == "H9" || id == "H10" || id == "H11" {
					if b.Within2xControl == nil {
						continue
					}
				}
				found = true
			}
		}
		if !found {
			return false
		}
	}
	for _, ctrl := range recs {
		if ctrl.RunID == r.ControlRef && ctrl.Kind == "control" && ctrl.Valid && ctrl.BaselineApproved && len(ctrl.InvalidReasons) == 0 && ctrl.CreatedAt.Before(r.CreatedAt) && ctrl.CreatedAt.Format("2006-01-02") == r.CreatedAt.Format("2006-01-02") {
			return true
		}
	}
	return false
}

// findingsPath is the findings register of this ticket's tactical folder.
func findingsPath() string {
	return filepath.Join("DEV", "plan", "113_2026-10-11_dynamic-behaviour-audit", "FINDINGS.md")
}
