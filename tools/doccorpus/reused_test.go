package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestReportNamesReusedCasesAndMarksTheirCells(t *testing.T) {
	m := &Manifest{Cases: []Case{baseCase("epub")}}
	reused := &Result{CaseID: "en-x-case", ReusedFrom: &ReusedFrom{
		CaseDir: "temp/doccorpus/earlier/windows-default/en-x-case", CollectedAt: "2026-10-09T10:00:00Z", ReusedAt: "2026-10-09T11:00:00Z", Verdict: Pass,
	}}
	passes := []runPass{
		{Name: "windows-default", Results: map[string]*Result{"en-x-case": reused}},
		{Name: "extension-default", Results: map[string]*Result{"en-x-case": {CaseID: "en-x-case"}}},
	}

	var b strings.Builder
	writeReused(func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }, passes, m)
	got := b.String()
	for _, want := range []string{"## Reused cases", "`en-x-case` | windows-default | `temp/doccorpus/earlier/windows-default/en-x-case`", "2026-10-09T10:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "extension-default") {
		t.Error("a collected case must not be listed as reused")
	}
	if reusedMark(passes[0].Results["en-x-case"]) != "*" || reusedMark(passes[1].Results["en-x-case"]) != "" || reusedMark(nil) != "" {
		t.Error("only a reused result is marked")
	}

	b.Reset()
	passes[0].Results["en-x-case"].ReusedFrom = nil
	writeReused(func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }, passes, m)
	if b.Len() != 0 {
		t.Errorf("a run that collected everything has no reuse section, got %q", b.String())
	}
}
