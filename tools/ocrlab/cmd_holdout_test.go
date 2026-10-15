package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/holdout"
)

// The gate-side rule: a full-benchmark verdict may only name a scope whose holdout was frozen
// and matches what the run judged. Anything else is missing evidence, never a pass.
func TestHoldoutGateIssuesRequireFrozenMembership(t *testing.T) {
	root := t.TempDir()
	record := filepath.Join(root, "holdout.json")

	// Not a full benchmark: nothing to require.
	if issues := holdoutGateIssues(record, "selected-dev", root); issues != nil {
		t.Fatalf("selected-dev must not need a holdout freeze, got %v", issues)
	}
	// Full benchmark without a record.
	issues := holdoutGateIssues(record, "full-benchmark", root)
	if len(issues) != 1 || !strings.Contains(issues[0], "holdout freeze unavailable") {
		t.Fatalf("missing freeze must be reported, got %v", issues)
	}

	f := &holdout.Freeze{SchemaVersion: 1, FrozenOn: "2026-10-15", FrozenBy: "operator",
		Scenes: map[string]holdout.FrozenScene{"h1": {SHA256: "sha-h1"}}}
	if err := holdout.Save(record, f); err != nil {
		t.Fatal(err)
	}
	// A run directory with no frozen corpus snapshot.
	issues = holdoutGateIssues(record, "full-benchmark", root)
	if len(issues) != 1 || !strings.Contains(issues[0], "corpus snapshot unavailable") {
		t.Fatalf("missing run corpus must be reported, got %v", issues)
	}

	runDir := filepath.Join(root, "run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestJSON(t, filepath.Join(runDir, "corpus.json"), corpus.Manifest{SchemaVersion: corpus.SchemaVersion, Scenes: []corpus.Scene{
		{ID: "h1", Split: corpus.SplitHoldout, SHA256: "sha-h1"},
	}})
	if issues = holdoutGateIssues(record, "full-benchmark", runDir); len(issues) != 0 {
		t.Fatalf("matching membership must be clean, got %v", issues)
	}

	// The run judged a holdout the freeze does not know.
	writeTestJSON(t, filepath.Join(runDir, "corpus.json"), corpus.Manifest{SchemaVersion: corpus.SchemaVersion, Scenes: []corpus.Scene{
		{ID: "h1", Split: corpus.SplitHoldout, SHA256: "sha-h1"},
		{ID: "h2", Split: corpus.SplitHoldout, SHA256: "sha-h2"},
	}})
	issues = holdoutGateIssues(record, "full-benchmark", runDir)
	if len(issues) != 1 || !strings.Contains(issues[0], "not in the frozen holdout") {
		t.Fatalf("drifted membership must be reported, got %v", issues)
	}
}
