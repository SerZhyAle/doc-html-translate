package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTimingSummaryPreservesSamplesAndDoesNotMutateInputs(t *testing.T) {
	samples := []float64{40, 10, 30, 20}
	s := summarizeTiming(samples)
	if s.Samples != 4 || s.Median != 25 || s.P95 != 40 || s.Min != 10 || s.Max != 40 || s.StdDev <= 0 {
		t.Fatalf("%+v", s)
	}
	if samples[0] != 40 {
		t.Fatal("input mutated")
	}
	if summarizeTiming(nil).Samples != 0 {
		t.Fatal("invented sample")
	}
}

func TestResolveConditionsSeparatesColdFromWarmRepetitions(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "conditions.json")
	if err := os.WriteFile(file, []byte(`{"rep-1": "cold", "rep-2": "warm"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dirs := []string{filepath.Join(root, "rep-1"), filepath.Join(root, "rep-2"), filepath.Join(root, "rep-3")}
	got, err := resolveConditions(dirs, file, "warm")
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "cold" || got[1] != "warm" || got[2] != "warm" {
		t.Fatalf("conditions misassigned: %v", got)
	}
	// Without a file every run takes the global label.
	got, err = resolveConditions(dirs, "", "unclassified")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c != "unclassified" {
			t.Fatalf("fallback not applied: %v", got)
		}
	}
	// A condition outside the vocabulary is refused, not silently degraded.
	bad := filepath.Join(root, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"rep-1": "lukewarm"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveConditions(dirs, bad, "warm"); err == nil {
		t.Fatal("unknown condition must be refused")
	}
	if _, err := resolveConditions(dirs, file, "hot"); err == nil {
		t.Fatal("unknown fallback must be refused")
	}
	// A missing conditions file is an error, never an unlabelled report.
	if _, err := resolveConditions(dirs, filepath.Join(root, "absent.json"), "warm"); err == nil {
		t.Fatal("missing conditions file must be refused")
	}
}
