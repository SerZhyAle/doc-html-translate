package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClosingRecordRequiresRecoveryBudgetsAndSameDayControl(t *testing.T) {
	now := time.Now()
	within := true
	ctrl := &record{RunID: "control", Kind: "control", Valid: true, BaselineApproved: true, CreatedAt: now.Add(-time.Minute), Measurements: []measurement{{Probe: "P2.host", N: 30, P95Ms: 5}}}
	scene := &record{RunID: "scene", Kind: "scene", Scene: "S03", CreatedAt: now, Valid: true, ControlRef: "control", Isolation: "B-full", RestoreVerified: true, RecoverySHA256: "digest", TreeBefore: "tree", TreeAfter: "tree", Cells: []cellResult{{Scene: "S03", Property: 1, State: "measured", Evidence: "latency distribution", Budgets: []budgetVerdict{{ID: "H5", Verdict: "pass", Observed: "5ms", Limit: "10ms", Within2xControl: &within}}}}}
	recs := []*record{ctrl, scene}
	if closingRecord(recs, "S03", 1) != scene {
		t.Fatal("complete evidence refused")
	}
	for _, what := range []string{"restore", "control", "budget", "advisory", "invalid", "tree", "relative"} {
		t.Run(what, func(t *testing.T) {
			copyRecord := *scene
			copyRecord.Cells = append([]cellResult(nil), scene.Cells...)
			copyRecord.Cells[0].Budgets = append([]budgetVerdict(nil), scene.Cells[0].Budgets...)
			switch what {
			case "restore":
				copyRecord.RestoreVerified = false
			case "control":
				copyRecord.ControlRef = "absent"
			case "budget":
				copyRecord.Cells[0].Budgets = nil
			case "advisory":
				copyRecord.Cells[0].Budgets[0].Verdict = "advisory"
			case "invalid":
				copyRecord.InvalidReasons = []string{"sleep"}
			case "tree":
				copyRecord.TreeAfter = "changed"
			case "relative":
				copyRecord.Cells[0].Budgets[0].Within2xControl = nil
			}
			if closingRecord([]*record{ctrl, &copyRecord}, "S03", 1) != nil {
				t.Fatal("incomplete evidence closed a cell")
			}
		})
	}
	ctrl.CreatedAt = now.Add(-48 * time.Hour)
	if closingRecord(recs, "S03", 1) != nil {
		t.Fatal("control from another day closed a cell")
	}
	ctrl.CreatedAt = now.Add(-time.Minute)
	ctrl.BaselineApproved = false
	if closingRecord(recs, "S03", 1) != nil {
		t.Fatal("unsigned hypotheses closed a matrix cell")
	}
}

func TestNewestInvalidControlInvalidatesEarlierBaseline(t *testing.T) {
	e := &runEnv{artifactsDir: t.TempDir()}
	now := time.Now()
	for i, r := range []*record{{RunID: newRunID("control"), Kind: "control", Valid: true, CreatedAt: now.Add(-time.Minute), Measurements: []measurement{{Probe: "probe", N: 1, P95Ms: 1}}}, {RunID: newRunID("control"), Kind: "control", Valid: false, CreatedAt: now, InvalidReasons: []string{"machine noisy"}}} {
		if _, err := saveRecord(filepath.Join(e.artifactsDir, string(rune('a'+i))), r); err != nil {
			t.Fatal(err)
		}
	}
	if latestControl(e) != nil {
		t.Fatal("old valid control bypassed a newer invalid run")
	}
}

func TestFixtureManifestRequiredEvenWhenFilesExist(t *testing.T) {
	dir := t.TempDir()
	if _, err := generateFixtures(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	_, corrupt, err := checkFixtures(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(corrupt) == 0 {
		t.Fatal("unhashed corpus accepted")
	}
}

func TestHostBudgetKeepsRelativeComparison(t *testing.T) {
	ctrl := &record{Measurements: []measurement{{Probe: "P2.host", N: 30, P95Ms: 5}}}
	if v := budgetCheck(findBudget("H1"), "50ms", 50, ctrl, "P2.host"); v.Verdict != "fail" {
		t.Fatal("host budget ignored 2x control")
	}
	if v := budgetCheck(findBudget("H1"), "5100ms", 5100, nil, "P2.host"); v.Verdict != "fail" {
		t.Fatal("watchdog breach downgraded without control")
	}
}
