// verdict.go speaks the shared CHECK-VERDICT vocabulary (docs/contracts/CHECK-VERDICT.md): four
// outcomes, one machine-readable line at the end of every run, advisories named rather than
// counted, and no stopping at the first failure.

package main

import "fmt"

// verdict is the four-outcome vocabulary. Only PASS is clean.
type verdict int

const (
	verdictPass verdict = iota
	verdictAdvisory
	verdictFail
	verdictCouldNotVerify
)

func (v verdict) word() string {
	switch v {
	case verdictPass:
		return "PASS"
	case verdictAdvisory:
		return "PASS WITH ADVISORIES"
	case verdictFail:
		return "FAIL"
	default:
		return "COULD NOT VERIFY"
	}
}

// emitVerdict prints the run's final line: `<check>: <VERDICT> (detail)` - detail on anything
// but a clean pass, per the contract's "advisories named rather than counted".
func emitVerdict(check string, v verdict, detail string) {
	line := fmt.Sprintf("auditstand %s: %s", check, v.word())
	if detail != "" {
		line += fmt.Sprintf(" (%s)", detail)
	}
	fmt.Println(line)
}

// worstVerdict folds child verdicts: any fail wins, then could-not-verify, then advisories -
// the same precedence check.ps1 uses.
func worstVerdict(vs ...verdict) verdict { //nolint:unused // verdict fold for the run summary, kept for ticket 113
	w := verdictPass
	for _, v := range vs {
		switch {
		case v == verdictFail:
			return verdictFail
		case v == verdictCouldNotVerify && w != verdictFail:
			w = verdictCouldNotVerify
		case v == verdictAdvisory && w == verdictPass:
			w = verdictAdvisory
		}
	}
	return w
}
