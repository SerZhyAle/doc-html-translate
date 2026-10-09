package main

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
)

// cmdVerdict grades one case folder with the judge as it is now and prints the verdict as one JSON
// line. run.mjs asks it before reusing an earlier result: reuse needs the verdict of today's judge
// on that evidence, not a report.json an older judge may have written, and a case folder is graded
// without regrading the whole run.
func cmdVerdict(m *Manifest, expectDir, caseDir string, out io.Writer) error {
	r, err := LoadResult(filepath.Join(caseDir, "result.json"))
	if err != nil {
		return fmt.Errorf("%v: %w", err, errCouldNotVerify)
	}
	var found *Case
	for i := range m.Cases {
		if m.Cases[i].ID == r.CaseID {
			found = &m.Cases[i]
		}
	}
	if found == nil {
		return fmt.Errorf("case %q is not in the manifest: %w", r.CaseID, errCouldNotVerify)
	}
	exp, err := LoadExpectation(filepath.Join(expectDir, found.ID+".json"))
	if err != nil {
		exp = nil // a missing expectation grades as a draft would, as in BuildReport
	}
	j := Judge(*found, exp, r)
	line, err := json.Marshal(struct {
		CaseID   string `json:"caseId"`
		Edition  string `json:"edition"`
		OCRMode  string `json:"ocrMode"`
		Auto     string `json:"auto"`
		Campaign string `json:"campaign"`
	}{j.CaseID, j.Edition, j.OCRMode, j.Auto, j.Campaign})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(line))
	return err
}
