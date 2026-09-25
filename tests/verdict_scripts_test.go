package tests

// Drives scripts/check.ps1 and scripts/parity-check.ps1 to every outcome they document and reads
// the result the way a caller does: the exit code and the last line (CHECK-VERDICT section 7,
// rung 1). A documented code that no test reaches is how "exit 2" silently became "exit 1" in the
// contract's reference incident.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// runScript runs a PowerShell script in dir and returns its exit code, its last non-empty line and
// the whole output.
func runScript(t *testing.T, pwsh, dir, script string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(pwsh, append([]string{"-NoProfile", "-NonInteractive", "-File", script}, args...)...)
	cmd.Dir = dir
	outBytes, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %s: %v", script, err)
		}
		code = ee.ExitCode()
	}
	out := ansiEscape.ReplaceAllString(strings.ReplaceAll(string(outBytes), "\r\n", "\n"), "")
	lines := strings.Split(strings.TrimRight(out, "\n "), "\n")
	return code, strings.TrimSpace(lines[len(lines)-1]), out
}

func findPwsh(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh (PowerShell 7) not on PATH - the gate scripts cannot be driven here")
	}
	return p
}

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join(append([]string{".."}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAggregatorOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	check := repoPath(t, "scripts", "check.ps1")
	// check.ps1 writes temp/logs relative to its working directory, so it runs in a scratch
	// directory - never over the repository's own gate evidence.
	dir := t.TempDir()
	child := func(name string, code int, line string) string {
		p := filepath.Join(dir, name+".ps1")
		body := ""
		if line != "" {
			body = "Write-Host '" + line + "'\n"
		}
		writeFile(t, p, body+"exit "+string(rune('0'+code))+"\n")
		return p
	}
	pass := child("okay", 0, "okay: PASS")
	adv := child("drift", 3, "drift: PASS WITH ADVISORIES (1: x)")
	cnv := child("notool", 2, "notool: COULD NOT VERIFY (tool absent)")
	fail := child("broken", 1, "broken: FAIL (1)")
	mute := child("mute", 7, "")

	cases := []struct {
		name     string
		plan     []string
		code     int
		lastLine string
	}{
		{"all pass", []string{pass, pass}, 0, "check: PASS"},
		{"an advisory colours a clean run", []string{pass, adv}, 3, "check: PASS WITH ADVISORIES (1: drift)"},
		{"could-not-verify is never a pass", []string{pass, adv, cnv}, 2, "check: COULD NOT VERIFY (1: notool)"},
		{"a failure outranks everything", []string{fail, cnv, adv, pass}, 1, "check: FAIL (1: broken)"},
		{"an unknown code without a line is a failure", []string{pass, mute}, 1, "check: FAIL (1: mute)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, last, out := runScript(t, pwsh, dir, check, "-Plan", strings.Join(c.plan, ","))
			if code != c.code || last != c.lastLine {
				t.Fatalf("exit %d, last line %q; want exit %d, %q\n%s", code, last, c.code, c.lastLine, out)
			}
			// Rule 9: every child ran, whatever the ones before it returned.
			for _, p := range c.plan {
				name := strings.TrimSuffix(filepath.Base(p), ".ps1")
				if !strings.Contains(out, "== "+name+" ==") {
					t.Errorf("child %s did not run:\n%s", name, out)
				}
			}
		})
	}
}

func TestParityCheckOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.autocrlf=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	writeFile(t, filepath.Join(dir, "scripts", "parity-check.ps1"), readRepoFile(t, "scripts", "parity-check.ps1"))
	writeFile(t, filepath.Join(dir, "scripts", "lib", "verdict.ps1"), readRepoFile(t, "scripts", "lib", "verdict.ps1"))
	writeFile(t, filepath.Join(dir, "configs", "parity-map.json"),
		`{"pairs":[{"name":"PDF","go":["internal/pdf/"],"js":["extension/src/reflow.js"]}],"acknowledge":["docs/PARITY.md"]}`)
	writeFile(t, filepath.Join(dir, "internal", "pdf", "a.go"), "package pdf\n")
	writeFile(t, filepath.Join(dir, "extension", "src", "reflow.js"), "// js\n")
	writeFile(t, filepath.Join(dir, "docs", "PARITY.md"), "# parity\n")
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	script := filepath.Join(dir, "scripts", "parity-check.ps1")

	run := func(args ...string) (int, string, string) { return runScript(t, pwsh, dir, script, args...) }
	expect := func(what string, wantCode int, wantPrefix string, args ...string) {
		t.Helper()
		code, last, out := run(args...)
		if code != wantCode || !strings.HasPrefix(last, wantPrefix) {
			t.Fatalf("%s: exit %d, last line %q; want exit %d, prefix %q\n%s", what, code, last, wantCode, wantPrefix, out)
		}
	}

	expect("empty change set", 0, "parity-check: PASS (0 file(s) inspected")
	expect("an undeterminable change set", 2, "parity-check: COULD NOT VERIFY", "-Range", "no-such-ref..HEAD")

	writeFile(t, filepath.Join(dir, "internal", "pdf", "a.go"), "package pdf\n\n// changed\n")
	expect("one-sided change", 3, "parity-check: PASS WITH ADVISORIES (1: PDF)")
	expect("one-sided change, strict", 1, "parity-check: FAIL (1: PDF)", "-Strict")

	writeFile(t, filepath.Join(dir, "docs", "PARITY.md"), "# parity\n\nrecorded\n")
	expect("acknowledged in PARITY.md", 0, "parity-check: PASS")

	git("checkout", "-q", "--", "docs/PARITY.md")
	writeFile(t, filepath.Join(dir, "extension", "src", "reflow.js"), "// js changed\n")
	expect("both sides moved", 0, "parity-check: PASS (2 file(s) inspected")
}

// gateLine returns the "gate evidence" line of scripts/release.ps1's checklist header.
func gateLine(t *testing.T, out string) string {
	t.Helper()
	for _, l := range strings.Split(out, "\n") {
		if s := strings.TrimSpace(l); strings.HasPrefix(s, "gate evidence") {
			return s
		}
	}
	t.Fatalf("release.ps1 printed no gate evidence line:\n%s", out)
	return ""
}

// TestGateEvidenceBindsTheRelease drives scripts/check.ps1 and scripts/release.ps1 together over a
// scratch repository whose gate children are stubs, and reads the release checklist's gate line:
// only the full default plan, every child passing, on HEAD's tree unlocks the tag step (ticket 36,
// R1-R3). The contract gate is not copied, so release.ps1 always exits 1 here - the line is the
// verdict under test, not the exit code.
func TestGateEvidenceBindsTheRelease(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "check.ps1", "release.ps1", "lib/verdict.ps1")
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }
	placement := readRepoFile(t, "configs", "check-placement.jsonl")
	w("configs/check-placement.jsonl", placement)
	w(".gitignore", "temp/\n")
	w("DEV/COMMIT_LOG.md", "# COMMIT LOG\n")
	w("src.txt", "one\n")

	// A passing stub for every gate child the placement record names, at the path check.ps1 runs.
	var gate []string
	for _, l := range strings.Split(placement, "\n") {
		var r struct{ Check, Class, Runner string }
		if strings.TrimSpace(l) == "" {
			continue
		}
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("configs/check-placement.jsonl: %v", err)
		}
		if r.Class == "gate" && r.Runner == "scripts/check.ps1" {
			name := strings.TrimSuffix(filepath.Base(r.Check), ".ps1")
			w(r.Check, "Write-Host '"+name+": PASS'\nexit 0\n")
			gate = append(gate, r.Check)
		}
	}
	if len(gate) < 2 {
		t.Fatalf("configs/check-placement.jsonl names %d gate check(s); the subset case needs two", len(gate))
	}
	// A child that edits a tracked file while the gate runs.
	w("scripts/editor.ps1", "Add-Content -LiteralPath src.txt -Value 'edited'\nWrite-Host 'editor: PASS'\nexit 0\n")
	git("add", "-A")
	git("commit", "-q", "-m", "init")

	check := filepath.Join(dir, "scripts", "check.ps1")
	release := func(what, wantPrefix string) {
		t.Helper()
		_, _, out := runScript(t, pwsh, dir, filepath.Join(dir, "scripts", "release.ps1"))
		if line := gateLine(t, out); !strings.HasPrefix(line, "gate evidence   : "+wantPrefix) {
			t.Fatalf("%s: %q; want prefix %q\n%s", what, line, wantPrefix, out)
		}
	}

	if code, last, out := runScript(t, pwsh, dir, check); code != 0 || last != "check: PASS" {
		t.Fatalf("full plan: exit %d, %q\n%s", code, last, out)
	}
	release("the full plan on HEAD's tree", "check: PASS on tree")

	// R1: one child passing is not the gate passing.
	if code, last, out := runScript(t, pwsh, dir, check, "-Plan", gate[0]); code != 0 || last != "check: PASS" {
		t.Fatalf("subset plan: exit %d, %q\n%s", code, last, out)
	}
	release("a -Plan subset", "BLOCKED - the last gate ran a partial plan")

	// R3: an edit landing while the gate runs voids the evidence, whatever the children said.
	code, last, out := runScript(t, pwsh, dir, check, "-Plan", strings.Join(append(append([]string{}, gate...), "scripts/editor.ps1"), ","))
	if code != 2 || last != "check: COULD NOT VERIFY (the working tree changed during the run)" {
		t.Fatalf("edit during the run: exit %d, %q\n%s", code, last, out)
	}
	release("an edit during the run", "BLOCKED - the last gate did not pass")
	git("checkout", "-q", "--", "src.txt")

	// R2: the build log build-local appends after its commit is the one path allowed to differ.
	if code, last, out := runScript(t, pwsh, dir, check); code != 0 || last != "check: PASS" {
		t.Fatalf("full plan again: exit %d, %q\n%s", code, last, out)
	}
	w("DEV/COMMIT_LOG.md", "# COMMIT LOG\n\n| entry |\n")
	git("commit", "-q", "-am", "log")
	release("only DEV/COMMIT_LOG.md moved after the gate", "check: PASS on tree")
	w("src.txt", "two\n")
	git("commit", "-q", "-am", "ungated")
	release("a commit the gate never read", "BLOCKED - the last passing gate ran on tree")
}
