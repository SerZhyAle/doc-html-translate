package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func s03Run(t *testing.T, mode string) map[string]outcomeCheck {
	t.Helper()
	t.Setenv("AUDITSTAND_STUB_CLI", "1")
	t.Setenv("AUDITSTAND_STUB_MODE", mode)
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	fixtures := filepath.Join(root, "fixtures")
	if err := os.MkdirAll(fixtures, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtures, "epub-small.epub"), []byte("not read by the stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &runEnv{artifactsDir: filepath.Join(root, "artifacts"), fixturesDir: fixtures, appPath: bin, isolation: "B-env", sceneRunID: "RUN"}
	rec, err := runSceneS03(env, context.Background(), 1)
	if err != nil {
		t.Fatalf("scene: %v", err)
	}
	out := map[string]outcomeCheck{}
	for _, oc := range rec.OutcomeChecks {
		out[oc.Injection] = oc
	}
	if len(out) != 3 {
		t.Fatalf("outcome checks = %d, want 3 (notes: %s)", len(out), rec.Notes)
	}
	return out
}

// -folder names the parent; the record is one level down. Before this was pinned the scene looked
// at <out>/ and would have reported "state inconsistent" for a healthy first conversion.
func TestS03ReuseAndForceAgainstACorrectCLI(t *testing.T) {
	checks := s03Run(t, "")
	for name, oc := range checks {
		if !oc.NamedOutcome || !oc.BoundaryLogFound || !oc.StateConsistent {
			t.Errorf("%s: %+v", name, oc)
		}
	}
}

func TestS03RebuildInsteadOfReuseIsCaught(t *testing.T) {
	checks := s03Run(t, "no-reuse")
	if oc := checks["S03-reopen"]; oc.NamedOutcome || oc.BoundaryLogFound {
		t.Fatalf("a run that never reuses a finished output must fail the reopen check: %+v", oc)
	}
}
