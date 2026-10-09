package main

import (
	"fmt"
	"strings"
)

// ReusedFrom is what run.mjs writes into a result it copied from an earlier run instead of
// collecting again (tools/doccorpus/reuse.mjs). The report names such cases: a reused case is the
// earlier measurement, graded again, and not a new observation of this run.
type ReusedFrom struct {
	CaseDir     string `json:"caseDir"`
	KeyDigest   string `json:"keyDigest"`
	Producer    string `json:"producer"`
	CollectedAt string `json:"collectedAt"`
	ReusedAt    string `json:"reusedAt"`
	Verdict     string `json:"verdict"`
}

// reusedMark is the suffix of a reused case's cell in the per-case table.
func reusedMark(r *Result) string {
	if r != nil && r.ReusedFrom != nil {
		return "*"
	}
	return ""
}

// writeReused renders the section naming every reused case, or nothing when the run collected all.
func writeReused(w func(string, ...any), passes []runPass, m *Manifest) {
	var lines []string
	for _, c := range m.Cases {
		for _, p := range passes {
			if r := p.Results[c.ID]; r != nil && r.ReusedFrom != nil {
				f := r.ReusedFrom
				lines = append(lines, fmt.Sprintf("| `%s` | %s | `%s` | %s | %s | %s |", c.ID, p.Name, f.CaseDir, f.CollectedAt, f.ReusedAt, f.Verdict))
			}
		}
	}
	if len(lines) == 0 {
		return
	}
	w("\n## Reused cases\n\n`*` in the tables marks a case copied from an earlier run whose inputs were identical (source, edition, OCR mode, code, browser, tools, expectation). It is that run's measurement, graded again by today's judge.\n\n")
	w("| case | pass | from | collected | reused | earlier verdict |\n|---|---|---|---|---|---|\n%s\n", strings.Join(lines, "\n"))
}
