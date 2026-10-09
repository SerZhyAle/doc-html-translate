package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// verdictFolder writes a result.json (and the text files the judge reads) the way run.mjs leaves a
// case folder, and returns the folder.
func verdictFolder(t *testing.T, r *Result) string {
	t.Helper()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, "result.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return r.dir
}

func TestVerdictGradesOneCaseFolderWithTodaysJudge(t *testing.T) {
	m := &Manifest{Cases: []Case{baseCase("text-pdf")}}
	clean := resultWith(t, &Probe{Lang: "en", TocEntries: 1}, "", "")
	var out bytes.Buffer
	if err := cmdVerdict(m, t.TempDir(), verdictFolder(t, clean), &out); err != nil {
		t.Fatal(err)
	}
	var got struct{ Auto, Campaign string }
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Auto != Pass {
		t.Fatalf("a clean page is PASS, got %q (%v)", out.String(), err)
	}

	broken := resultWith(t, &Probe{Lang: "en", TocEntries: 1}, "", "")
	broken.Convert.ExitCode = exitCode(3)
	out.Reset()
	if err := cmdVerdict(m, t.TempDir(), verdictFolder(t, broken), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"auto":"FAIL"`) {
		t.Fatalf("a failed conversion is FAIL, got %s", out.String())
	}
}

func TestVerdictOfAnUnknownCaseOrFolderIsCouldNotVerify(t *testing.T) {
	m := &Manifest{Cases: []Case{baseCase("text-pdf")}}
	stray := resultWith(t, &Probe{Lang: "en"}, "", "")
	stray.CaseID = "not-in-the-manifest"
	for name, dir := range map[string]string{"unknown case": verdictFolder(t, stray), "no result": t.TempDir()} {
		var out bytes.Buffer
		err := cmdVerdict(m, t.TempDir(), dir, &out)
		if !errors.Is(err, errCouldNotVerify) || out.Len() != 0 {
			t.Errorf("%s: want COULD NOT VERIFY and no verdict, got %v / %q", name, err, out.String())
		}
	}
}
