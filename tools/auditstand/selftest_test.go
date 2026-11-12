package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCapabilitiesDetectsStalePin(t *testing.T) {
	claude := "something without the pin phrase"
	agents := "plain text"
	defects, notes := checkCapabilities(claude, agents, "")
	found := false
	for _, d := range defects {
		if strings.Contains(d.Error(), "stale pin") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a vanished pin must be a stale-pin defect; got defects=%v notes=%v", defects, notes)
	}
}

func TestCapabilitiesDetectsUncoveredCapability(t *testing.T) {
	// Synthetic docs shaped like the real ones (the scoped section headers) that contain every
	// pin and every claim (so no stale-pin noise) plus one extra internal/ package the
	// inventory never claims: direction 1 must flag it.
	var claude, agents strings.Builder
	claude.WriteString("## What this is\n\n## High-value invariants\n\n")
	agents.WriteString("## Architecture Map\n\n")
	seen := map[string]bool{}
	addPin := func(b *strings.Builder, p pin) {
		if seen[p.phrase] {
			return
		}
		seen[p.phrase] = true
		b.WriteString(p.phrase + "\n")
	}
	for _, row := range capabilityInventory {
		for _, p := range row.pins {
			switch p.file {
			case claudemd:
				addPin(&claude, p)
			case agentsmd:
				addPin(&agents, p)
			}
		}
	}
	// Claims must also stay derivable: bullet claims go under the Architecture Map headline.
	for _, row := range capabilityInventory {
		for _, c := range row.covers {
			if strings.ContainsAny(c, "/") && !strings.HasPrefix(c, "internal/") && !strings.HasPrefix(c, "cmd/") {
				continue // phrase claims live as pins or bullets already
			}
			agents.WriteString("- " + c + ": x\n")
		}
	}
	agents.WriteString("- internal/totallynewthing: a capability nobody covers\n")
	defects, _ := checkCapabilities(claude.String(), agents.String(), "")
	found := false
	for _, d := range defects {
		if strings.Contains(d.Error(), "totallynewthing") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a derived capability with no row must be a defect; got %v", defects)
	}
}

func TestCapabilitiesAgainstRealRepoDocs(t *testing.T) {
	// The real files must pass right now: the pins were verified against them. When this test
	// fails, either the docs dropped a pinned phrase (a stale pin) or a new capability bullet
	// needs an inventory row. The sources are read relative to the repository root, where the stand
	// is started; go test starts in the package directory, so stand in the root for this read.
	t.Chdir(filepath.Join("..", ".."))
	claude, agents, roster, err := loadCapabilitySources()
	if err != nil {
		t.Skip("input absent:", err)
	}
	defects, _ := checkCapabilities(claude, agents, roster)
	if len(defects) > 0 {
		t.Fatalf("capability defects against the live repo docs: %v", defects)
	}
	if len(outOfScopeMatches) == 0 {
		t.Fatal("expected the developer-tooling bullets to be matched out of scope")
	}
}

func TestSelftestFixtureStates(t *testing.T) {
	dir := t.TempDir()

	// No fixture root at all: could-not-verify, not a pass.
	res := runSelfTest(&runEnv{fixturesDir: filepath.Join(dir, "absent"), artifactsDir: filepath.Join(dir, "artifacts")})
	if res.verdict != verdictCouldNotVerify {
		t.Fatalf("absent fixture root must be COULD NOT VERIFY, got %v (%s)", res.verdict, res.detail)
	}
	if !regexp.MustCompile(`^run=\d skipped=\d, `).MatchString(res.detail) {
		t.Errorf("the self-test line must count its checks (CHECK-VERDICT 0.14 A1), got %q", res.detail)
	}

	// Generated fixtures: a clean pass needs the repo docs, which the package dir two levels up
	// holds when the tests run from tools/auditstand.
	if _, err := generateFixtures(filepath.Join(dir, "fx"), false); err != nil {
		t.Fatalf("fixture generation: %v", err)
	}
	missing, corrupt, err := checkFixtures(filepath.Join(dir, "fx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 || len(corrupt) != 0 {
		t.Fatalf("fresh fixtures reported missing=%v corrupt=%v", missing, corrupt)
	}

	// A mutated fixture is corrupt, not missing.
	f := filepath.Join(dir, "fx", "epub-small.epub")
	if err := os.WriteFile(f, []byte("corrupted"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, corrupt, err = checkFixtures(filepath.Join(dir, "fx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(corrupt) != 1 || corrupt[0] != "epub-small" {
		t.Fatalf("mutated fixture must be reported corrupt, got %v", corrupt)
	}
}
