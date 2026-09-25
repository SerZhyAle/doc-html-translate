package tests

// Licence guards for the pdftotext set the Windows executables embed (internal/bundledtools,
// ticket 33). Every file present in the set is on record in internal/bundledtools/PROVENANCE.txt
// with the hash of the bytes that ship, and is named in THIRD-PARTY-NOTICES.txt, which every
// channel carrying the executables stages beside them.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBundledFilesAreInTheNotices(t *testing.T) {
	dir := repoPath(t, "internal", "bundledtools", "pdftotext")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	notices := readRepoFile(t, "THIRD-PARTY-NOTICES.txt")
	provenance := readRepoFile(t, "internal", "bundledtools", "PROVENANCE.txt")
	for _, want := range []string{"GNU GENERAL PUBLIC LICENSE", "Version 2, June 1991", "Version 3, 29 June 2007",
		"GCC RUNTIME LIBRARY EXCEPTION", "https://dl.xpdfreader.com/old/xpdf-4.00.tar.gz"} {
		if !strings.Contains(notices, want) {
			t.Errorf("THIRD-PARTY-NOTICES.txt lacks %q", want)
		}
	}
	files := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files++
		name := e.Name()
		if !strings.Contains(notices, name) {
			t.Errorf("THIRD-PARTY-NOTICES.txt does not name the bundled %s", name)
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if line := hex.EncodeToString(sum[:]); !provenanceHas(provenance, line, name) {
			t.Errorf("internal/bundledtools/PROVENANCE.txt has no line %q for %s - record the file's origin and hash", line, name)
		}
	}
	if files == 0 {
		t.Fatal("internal/bundledtools/pdftotext holds no files")
	}
}

// provenanceHas reports whether one line of the hash table names file with hash.
func provenanceHas(provenance, hash, file string) bool {
	for _, line := range strings.Split(provenance, "\n") {
		f := strings.Fields(line)
		if len(f) == 3 && f[0] == hash && f[2] == file {
			return true
		}
	}
	return false
}

// The CI release build embeds only what a clean checkout holds, so the whole set must be tracked:
// pdftotext.exe once matched the *.exe ignore rule and every CI-built asset shipped the DLLs
// alone (ticket 35). The workflow re-checks the set against PROVENANCE.txt before it builds.
func TestBundledSetIsTrackedAndGuarded(t *testing.T) {
	dir := repoPath(t, "internal", "bundledtools", "pdftotext")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if !slices.Contains(names, "pdftotext.exe") {
		t.Errorf("internal/bundledtools/pdftotext lacks pdftotext.exe; have %v", names)
	}
	if _, err := exec.LookPath("git"); err == nil {
		for _, name := range names {
			// check-ignore exits 0 when the path is ignored, 1 when it is not.
			cmd := exec.Command("git", "check-ignore", "-q", "--no-index", filepath.Join(dir, name))
			cmd.Dir = repoPath(t)
			if err := cmd.Run(); err == nil {
				t.Errorf("internal/bundledtools/pdftotext/%s is git-ignored, so a clean checkout would build without it", name)
			}
		}
	}
	wf := readRepoFile(t, ".github", "workflows", "release.yml")
	guard := strings.Index(wf, "name: Verify the bundled pdftotext set")
	build := strings.Index(wf, "go build")
	if guard < 0 || build < 0 || guard > build {
		t.Error(".github/workflows/release.yml must verify the bundled pdftotext set before the first go build")
	}
}

// The release workflow builds the portable executables and the zip winget installs: both
// carry the embedded set, so the notices go into the zip and up as a release asset of their own.
func TestReleaseWorkflowShipsTheNotices(t *testing.T) {
	wf := readRepoFile(t, ".github", "workflows", "release.yml")
	for _, want := range []string{"Copy-Item THIRD-PARTY-NOTICES.txt $stage/", "dist/THIRD-PARTY-NOTICES.txt"} {
		if !strings.Contains(wf, want) {
			t.Errorf(".github/workflows/release.yml lacks %q", want)
		}
	}
}
