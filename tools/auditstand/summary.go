// summary.go implements -Summary (strategic section 5.4): the campaign state from the recorded
// runs. Exit 0 only when every matrix cell is measured or reasoned n/a and no P0/P1 finding is
// open - the audit-slices precedent: the campaign is closed by the script, not by prose.

package main

import (
	"fmt"
	"os"
	"strings"
)

// summaryResult is what main.go turns into the verdict.
type summaryResult struct {
	verdict verdict
	detail  string
}

// runSummary reads every record under the artifact root and renders the matrix.
func runSummary(env *runEnv) summaryResult {
	recs, errs := loadAllRecords(env.artifactsDir)
	for _, e := range errs {
		fmt.Printf("unreadable record: %v\n", e)
	}
	if len(errs) > 0 {
		return summaryResult{verdictCouldNotVerify, "campaign records unreadable; closure cannot be verified"}
	}

	open := 0
	na := 0
	measured := 0
	var openP0P1 []string
	printMatrixState(recs)

	// Open P0/P1 findings keep the campaign open even with a full matrix (acceptance 4).
	if f, err := os.ReadFile(findingsPath()); err == nil {
		openP0P1 = scanOpenP0P1(string(f))
	} else {
		return summaryResult{verdictCouldNotVerify, "findings register unreadable"}
	}
	for _, r := range recs {
		for _, f := range r.Findings {
			if f.Severity == "P0" || f.Severity == "P1" {
				openP0P1 = append(openP0P1, r.RunID+": "+f.Detail)
			}
		}
	}

	for _, c := range matrix {
		if closingRecord(recs, c.scene, c.prop) != nil {
			measured++
			continue
		}
		if c.state == cellNA {
			na++
			continue
		}
		open++
	}

	fmt.Println()
	fmt.Printf("cells: %d measured, %d n/a, %d open; runs recorded: %d\n", measured, na, open, countSceneRuns(recs))
	for _, id := range openP0P1 {
		fmt.Printf("open P0/P1 finding: %s\n", id)
	}

	// A same-day control must exist for anything to have been measured at all.
	if measured > 0 && latestControl(env) == nil {
		fmt.Println("warning: measured cells exist without a same-day control record")
	}

	switch {
	case open > 0:
		return summaryResult{verdictAdvisory, fmt.Sprintf("campaign open: %d cell(s) not yet measured (a partial matrix is the honest state while the phases run)", open)}
	case len(openP0P1) > 0:
		return summaryResult{verdictAdvisory, fmt.Sprintf("matrix covered but %d P0/P1 finding(s) still open", len(openP0P1))}
	default:
		return summaryResult{verdictPass, fmt.Sprintf("%d measured, %d n/a", measured, na)}
	}
}

func countSceneRuns(recs []*record) int {
	n := 0
	for _, r := range recs {
		if r.Kind == "scene" {
			n++
		}
	}
	return n
}

// scanOpenP0P1 finds FINDINGS.md rows with severity P0 or P1 whose routing column is still
// empty ("-"): a triaged finding has a ticket or an owner question in that column.
func scanOpenP0P1(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "| DYN-") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 4 {
			continue
		}
		id := strings.TrimSpace(cols[1])
		sev := strings.TrimSpace(cols[3])
		routing := strings.TrimSpace(cols[len(cols)-2])
		if (sev == "P0" || sev == "P1") && (routing == "-" || routing == "") {
			out = append(out, id+" ("+sev+")")
		}
	}
	return out
}
