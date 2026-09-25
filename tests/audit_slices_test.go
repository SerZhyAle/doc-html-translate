package tests

// Drives scripts/audit-slices.ps1 (ticket 34) over a scratch tree built here: the partition it
// promises (every class-A file in exactly one slice, limits kept, platform twins and a declared
// parity pair never parted, an oversized package cut into adjacent siblings, small neighbours
// merged), the same manifest from the same tree, the class-B flag changing membership only, and
// the summary's counts and exit codes as the campaign moves from open to closed.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type auditManifest struct {
	Base   string `json:"base"`
	Slices []struct {
		ID       string   `json:"id"`
		Units    []string `json:"units"`
		Siblings []string `json:"siblings"`
		Lines    int      `json:"lines"`
		Changed  int      `json:"changed"`
		Recheck  []string `json:"recheck"`
		Files    []struct {
			Path  string `json:"path"`
			Lines int    `json:"lines"`
		} `json:"files"`
	} `json:"slices"`
	RegisterClassB []string `json:"registerClassB"`
}

func goLines(pkg string, n int) string {
	var b strings.Builder
	b.WriteString("package " + pkg + "\n")
	for i := 1; i < n; i++ {
		b.WriteString("// line\n")
	}
	return b.String()
}

func readManifest(t *testing.T, dir string) (auditManifest, string) {
	t.Helper()
	raw := readScratch(t, dir, "plan/34/slices.json")
	var m auditManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("slices.json: %v\n%s", err, raw)
	}
	return m, raw
}

// holder maps each file to the id of the slice that holds it, failing on a file held twice.
func holder(t *testing.T, m auditManifest) map[string]string {
	t.Helper()
	h := map[string]string{}
	for _, s := range m.Slices {
		for _, f := range s.Files {
			if prev, ok := h[f.Path]; ok {
				t.Fatalf("%s is in %s and %s", f.Path, prev, s.ID)
			}
			h[f.Path] = s.ID
		}
	}
	return h
}

func TestAuditSlicesPartition(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "audit-slices.ps1", "lib/verdict.ps1")
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }

	w("configs/parity-map.json", `{"pairs":[{"name":"pair","go":["internal/pair/"],"js":["extension/src/pair.js"]}]}`)
	w("internal/big/a.go", goLines("big", 40))
	w("internal/big/b.go", goLines("big", 40))
	w("internal/big/c.go", goLines("big", 40))
	w("internal/big/d_windows.go", goLines("big", 30))
	w("internal/big/d_nonwindows.go", goLines("big", 30))
	w("internal/big/a_test.go", goLines("big", 10))
	w("internal/small1/x.go", goLines("small1", 10))
	w("internal/small2/y.go", goLines("small2", 10))
	w("internal/pair/p.go", goLines("pair", 30))
	w("extension/src/pair.js", strings.Repeat("// js\n", 20))
	w("extension/src/other.js", strings.Repeat("// js\n", 5))
	w("tools/lab/main.go", goLines("main", 10))
	w("scripts/release.ps1", strings.Repeat("# ps\n", 12))
	w("docs/notes.md", "# not code\n")
	w("reg.md", "- P1 - high - conf - a thing - `big/a.go:3` - 01\n- Q9 - low - conf - a test - `internal/big/a_test.go:1` - 17\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	w("internal/big/a.go", goLines("big", 40)+"// added\n// added\n")

	args := []string{"-Ticket", "plan/34", "-Register", "reg.md", "-Base", "HEAD", "-MaxLines", "100", "-MaxFiles", "6"}
	expectRun(t, pwsh, dir, "audit-slices.ps1", "slicing", 0, "audit-slices: PASS", "", append(args, "-Write")...)
	m, first := readManifest(t, dir)
	h := holder(t, m)

	classA := []string{"internal/big/a.go", "internal/big/b.go", "internal/big/c.go", "internal/big/d_windows.go",
		"internal/big/d_nonwindows.go", "internal/small1/x.go", "internal/small2/y.go", "internal/pair/p.go",
		"extension/src/pair.js", "extension/src/other.js", "scripts/release.ps1",
		// the copies of the script under test and its library are release-path scripts too
		"scripts/audit-slices.ps1", "scripts/lib/verdict.ps1"}
	for _, f := range classA {
		if h[f] == "" {
			t.Errorf("class-A file %s is in no slice", f)
		}
	}
	if len(h) != len(classA) {
		t.Errorf("slices hold %d files, want the %d class-A ones: %v", len(h), len(classA), h)
	}
	for _, s := range m.Slices {
		if s.Lines > 100 && len(s.Files) > 1 {
			t.Errorf("%s has %d lines over the limit of 100", s.ID, s.Lines)
		}
		if len(s.Files) > 6 {
			t.Errorf("%s has %d files over the limit of 6", s.ID, len(s.Files))
		}
	}
	if h["internal/big/d_windows.go"] != h["internal/big/d_nonwindows.go"] {
		t.Errorf("platform twins parted: %s / %s", h["internal/big/d_windows.go"], h["internal/big/d_nonwindows.go"])
	}
	if h["internal/pair/p.go"] != h["extension/src/pair.js"] {
		t.Errorf("parity pair parted: %s / %s", h["internal/pair/p.go"], h["extension/src/pair.js"])
	}
	if h["internal/small1/x.go"] != h["internal/small2/y.go"] {
		t.Errorf("small neighbours not merged: %s / %s", h["internal/small1/x.go"], h["internal/small2/y.go"])
	}

	// The oversized package is cut into adjacent siblings that name each other.
	var bigParts []string
	for _, s := range m.Slices {
		if len(s.Units) == 1 && s.Units[0] == "internal/big" {
			bigParts = append(bigParts, s.ID)
			if len(s.Siblings) == 0 {
				t.Errorf("%s is a part of internal/big but names no sibling", s.ID)
			}
		}
	}
	if len(bigParts) < 2 {
		t.Fatalf("internal/big (180 lines) was not cut: %v", bigParts)
	}
	for i := 1; i < len(bigParts); i++ {
		a, errA := strconv.Atoi(bigParts[i-1][1:])
		b, errB := strconv.Atoi(bigParts[i][1:])
		if errA != nil || errB != nil {
			t.Fatalf("slice ids %v are not S<number>", bigParts)
		}
		if b != a+1 {
			t.Errorf("the parts of internal/big are not adjacent: %v", bigParts)
		}
	}

	// Changed lines count against the base; the register id lands on the slice holding its file.
	for _, s := range m.Slices {
		for _, f := range s.Files {
			if f.Path == "internal/big/a.go" && s.Changed < 2 {
				t.Errorf("%s: 2 lines were added to a.go since the base, the manifest counts %d", s.ID, s.Changed)
			}
		}
		for _, id := range s.Recheck {
			if id == "P1" && s.ID != h["internal/big/a.go"] {
				t.Errorf("P1 cites internal/big/a.go, placed on %s instead of %s", s.ID, h["internal/big/a.go"])
			}
		}
	}
	if strings.Join(m.RegisterClassB, ",") != "Q9" {
		t.Errorf("Q9 cites only a test file; registerClassB = %v", m.RegisterClassB)
	}

	// The same tree gives the same manifest.
	expectRun(t, pwsh, dir, "audit-slices.ps1", "slicing again", 0, "audit-slices: PASS", "", append(args, "-Write")...)
	if _, again := readManifest(t, dir); again != first {
		t.Fatalf("a rerun on the same tree wrote a different manifest")
	}

	// Class B changes the membership only.
	expectRun(t, pwsh, dir, "audit-slices.ps1", "slicing with class B", 0, "audit-slices: PASS", "", append(args, "-Write", "-IncludeClassB")...)
	mb, _ := readManifest(t, dir)
	hb := holder(t, mb)
	var got []string
	for f := range hb {
		got = append(got, f)
	}
	want := append(append([]string{}, classA...), "internal/big/a_test.go", "tools/lab/main.go")
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("with -IncludeClassB the slices hold\n%v\nwant\n%v", got, want)
	}
}

func TestAuditSlicesSummary(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "audit-slices.ps1", "lib/verdict.ps1")
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }
	w("internal/one/a.go", goLines("one", 20))
	w("internal/two/b.go", goLines("two", 20))
	w("reg.md", "- P1 - high - conf - a thing - `one/a.go:3` - 01\n")
	w("plan/34/PHASE_TEMPLATE.md", "# Phase {{PHASE}}: {{SLICE}} - {{NAME}}\n\nRe-check: {{RECHECK_IDS}}\n\n{{FILES_TABLE}}\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")

	args := []string{"-Ticket", "plan/34", "-Register", "reg.md", "-Base", "HEAD", "-MaxLines", "30", "-MaxFiles", "5"}
	run := func(what string, code int, prefix, fragment string, extra ...string) {
		t.Helper()
		expectRun(t, pwsh, dir, "audit-slices.ps1", what, code, prefix, fragment, append(append([]string{}, args...), extra...)...)
	}
	run("no manifest yet", 2, "audit-slices: COULD NOT VERIFY", "", "-Summary")
	run("slicing", 0, "audit-slices: PASS (4 slices", "", "-Write")
	index := readScratch(t, dir, "plan/34/INDEX.md")
	if strings.Count(index, "| S0") != 4 {
		t.Fatalf("INDEX.md should have one phase row per slice:\n%s", index)
	}
	phases, _ := filepath.Glob(filepath.Join(dir, "plan", "34", "PHASE_0*.md"))
	if len(phases) != 4 {
		t.Fatalf("want 4 phase files, got %v", phases)
	}
	run("fresh campaign", 1, "audit-slices: FAIL (campaign open: 4 slices open, 1 re-checks missing", "", "-Summary")

	done := func() {
		t.Helper()
		idx := readScratch(t, dir, "plan/34/INDEX.md")
		w("plan/34/INDEX.md", strings.ReplaceAll(idx, "⬜ Not started", "✅ Done"))
	}
	done()
	w("plan/34/FINDINGS.md", "# Findings\n\n## Re-check of the 2026-09-24 register\n\n- P1 - still fixed - S02 - a.go:3\n\n"+
		"## New findings\n\n- P25 - high - conf - a new thing - one/a.go:5 - #07\n- P26 - low - conf - a nit - one/a.go:9 - -\n")
	w("DEV/plan/07_2026-09-26_a-new-thing.md", "# A new thing\n\n**Status:** Draft\n")
	run("every slice closed, re-checked, triaged", 0, "audit-slices: PASS (campaign closed: 4 slices, 2 findings)", "ticket #07 07_2026-09-26_a-new-thing - Draft", "-Summary")

	w("plan/34/FINDINGS.md", readScratch(t, dir, "plan/34/FINDINGS.md")+"- P27 - crit - plaus - untriaged - two/b.go:2 - -\n")
	run("a crit with no ticket", 1, "audit-slices: FAIL (campaign open: 1 crit/high untriaged)", "P27 is crit/high", "-Summary")
	w("plan/34/FINDINGS.md", strings.Replace(readScratch(t, dir, "plan/34/FINDINGS.md"), "two/b.go:2 - -", "two/b.go:2 - inline: go test ./internal/two exit 0", 1))
	run("the crit fixed inline", 0, "audit-slices: PASS", "", "-Summary")

	w("internal/three/c.go", goLines("three", 10))
	run("a file added after slicing", 1, "audit-slices: FAIL (campaign open: 1 uncovered", "uncovered: internal/three/c.go", "-Summary")
	run("a tail slice", 0, "audit-slices: PASS (1 tail slice(s) appended for 1 file(s))", "", "-Tail")
	run("the tail slice is open", 1, "audit-slices: FAIL (campaign open: 1 slices open)", "S05", "-Summary")
	done()
	run("closed again", 0, "audit-slices: PASS (campaign closed: 5 slices", "", "-Summary")

	if err := os.Remove(filepath.Join(dir, "internal", "two", "b.go")); err != nil {
		t.Fatal(err)
	}
	run("a sliced file deleted is reported, not failed", 0, "audit-slices: PASS", "gone since slicing: internal/two/b.go", "-Summary")
}
