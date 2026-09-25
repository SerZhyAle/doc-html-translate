package tests

// configs/check-placement.jsonl declares which runner owns each check (CHECK-PLACEMENT rule 1).
// A declaration nothing compares with the wiring is a comment, so this test holds it in both
// directions (rule 2): a check recorded for the gate must be in scripts/check.ps1's plan, a check in
// the plan must be recorded, a build-time check must be called by the build scripts it names, and
// every script that speaks the verdict vocabulary must be placed somewhere.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type placement struct {
	Check   string `json:"check"`
	Class   string `json:"class"`
	Runner  string `json:"runner"`
	Decided string `json:"decided"`
	Ticket  string `json:"ticket"`
	Basis   string `json:"basis"`
	Reason  string `json:"reason"`
}

// "release" is a check the release checklist (scripts/release.ps1) runs before the tag step: its
// input exists only on the machine a release is cut on, so the gate cannot hold it.
var placementClasses = map[string]bool{"gate": true, "build": true, "release": true, "hand-run": true, "none": true}

func loadPlacement(t *testing.T) map[string]placement {
	t.Helper()
	out := map[string]placement{}
	for i, line := range strings.Split(readRepoFile(t, "configs", "check-placement.jsonl"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var p placement
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("configs/check-placement.jsonl line %d: %v", i+1, err)
		}
		switch {
		case p.Check == "" || p.Runner == "" || p.Reason == "" || p.Decided == "":
			t.Errorf("line %d (%q): check, runner, decided and reason are all required", i+1, p.Check)
		case !placementClasses[p.Class]:
			t.Errorf("line %d (%q): unknown class %q", i+1, p.Check, p.Class)
		case p.Basis != "judged" && p.Basis != "seeded":
			t.Errorf("line %d (%q): basis must be judged or seeded, got %q", i+1, p.Check, p.Basis)
		case p.Basis == "seeded" && p.Ticket == "":
			t.Errorf("line %d (%q): a seeded record names the open ticket that owns judging it (rule 7)", i+1, p.Check)
		}
		if _, dup := out[p.Check]; dup {
			t.Errorf("line %d: %q is recorded twice", i+1, p.Check)
		}
		out[p.Check] = p
	}
	return out
}

// checkPlan reads the default children out of scripts/check.ps1's $defaultPlan array.
func checkPlan(t *testing.T) []string {
	t.Helper()
	src := readRepoFile(t, "scripts", "check.ps1")
	m := regexp.MustCompile(`(?s)\$defaultPlan = @\((.*?)\n\)`).FindStringSubmatch(src)
	if m == nil {
		t.Fatal("scripts/check.ps1: no $defaultPlan = @( .. ) block")
	}
	var plan []string
	for _, q := range regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(m[1], -1) {
		plan = append(plan, q[1])
	}
	if len(plan) == 0 {
		t.Fatal("scripts/check.ps1: $defaultPlan is empty")
	}
	return plan
}

func TestCheckPlacement(t *testing.T) {
	records := loadPlacement(t)
	plan := checkPlan(t)

	inPlan := map[string]bool{}
	for _, c := range plan {
		inPlan[c] = true
		r, ok := records[c]
		if !ok {
			t.Errorf("scripts/check.ps1 runs %s, which configs/check-placement.jsonl does not record", c)
			continue
		}
		if r.Class != "gate" {
			t.Errorf("scripts/check.ps1 runs %s, recorded as class %q - update the record, with a reason, if it moved", c, r.Class)
		}
	}

	for _, r := range records {
		if strings.HasPrefix(r.Check, "scripts/") {
			if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(r.Check))); err != nil {
				t.Errorf("%s is recorded but does not exist", r.Check)
			}
		}
		switch r.Class {
		case "gate":
			if r.Runner != "scripts/check.ps1" || !inPlan[r.Check] {
				t.Errorf("%s is recorded as a gate check but scripts/check.ps1 does not run it", r.Check)
			}
		case "build", "release":
			for _, runner := range strings.Split(r.Runner, ",") {
				runner = strings.TrimSpace(runner)
				if !strings.Contains(readRepoFile(t, filepath.FromSlash(runner)), filepath.Base(r.Check)) {
					t.Errorf("%s is recorded as run by %s, which does not call it", r.Check, runner)
				}
			}
		}
	}

	// Every script that speaks the CHECK-VERDICT vocabulary is a check and must be placed; the
	// aggregator itself is the runner, not a check.
	scripts, err := filepath.Glob(filepath.Join("..", "scripts", "*.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scripts {
		rel := "scripts/" + filepath.Base(s)
		if rel == "scripts/check.ps1" {
			continue
		}
		if strings.Contains(readRepoFile(t, "scripts", filepath.Base(s)), "lib/verdict.ps1") {
			if _, ok := records[rel]; !ok {
				t.Errorf("%s reports a CHECK-VERDICT verdict but has no record in configs/check-placement.jsonl", rel)
			}
		}
	}
}
