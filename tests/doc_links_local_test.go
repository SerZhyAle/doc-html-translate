package tests

// DOC-INTERNAL-QUALITY 0.10 (ticket 122, item 7): the local, git-ignored engineering corpus - the ticket store
// DEV/plan and the private notes DEV/private - is judged for links independently of the committed one. A relative
// link of a local document resolves to a file or folder that is on disk, in the local corpus or in the repository,
// and a heading anchor names a heading or id the target renders. The registry half (every local document is a record
// of DEV/DOCUMENT_REGISTRY.jsonl, and the record names no file the committed registry holds) is
// scripts/doc-registry-local.ps1, which also runs this test.
//
// The check is opt-in: the corpus is edited by many sessions at once and exists only on the owner's machine, so it
// is not part of the Go suite's verdict. scripts/doc-registry-local.ps1 sets DOCHT_LOCAL_CORPUS=1.

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// localTrees are the git-ignored trees of the engineering folder (.gitignore), in the repository's spelling.
var localTrees = []string{"DEV/plan", "DEV/private"}

// localCorpusFiles is every file on disk under the local trees, repository-relative with forward slashes.
func localCorpusFiles(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, tree := range localTrees {
		root := filepath.Join("..", filepath.FromSlash(tree))
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "node_modules" {
					return fs.SkipDir
				}
				return nil
			}
			rel, rerr := filepath.Rel("..", p)
			if rerr != nil {
				return rerr
			}
			out[filepath.ToSlash(rel)] = true
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
	return out
}

func TestLocalDocLinks(t *testing.T) {
	if os.Getenv("DOCHT_LOCAL_CORPUS") != "1" {
		t.Skip("opt-in: scripts/doc-registry-local.ps1 judges the local corpus (DOCHT_LOCAL_CORPUS=1)")
	}
	local := localCorpusFiles(t)
	if len(local) == 0 {
		t.Skip("the local corpus (DEV/plan, DEV/private) is not on this machine")
	}
	files := repoFiles(t)
	for f := range local {
		files[f] = true
	}
	dirs := repoDirs(files)
	cache := map[string]map[string]bool{}
	anchorsOf := func(f string) map[string]bool {
		if a, ok := cache[f]; ok {
			return a
		}
		a := headingAnchors(readRepoFile(t, f))
		cache[f] = a
		return a
	}
	judgeEverything := func(string) bool { return false }

	var docs []string
	for f := range local {
		if strings.HasSuffix(strings.ToLower(f), ".md") {
			docs = append(docs, f)
		}
	}
	sort.Strings(docs)
	if len(docs) == 0 {
		t.Skip("the local corpus holds no Markdown document")
	}
	problems := 0
	for _, f := range docs {
		for _, l := range docLinks(readRepoFile(t, f)) {
			if p := linkProblemIn(f, l, files, dirs, anchorsOf, judgeEverything); p != "" {
				t.Errorf("%s:%d: %s (%s)", f, l.line, p, l.dest)
				problems++
			}
		}
	}
	t.Logf("%d local document(s) read, %d link problem(s)", len(docs), problems)
}

// TestLocalCorpusScanPins the judgement on fixed inputs: a link into the local tree is judged here (the committed
// corpus skips it), so a regression cannot pass silently by finding nothing.
func TestLocalCorpusScanPins(t *testing.T) {
	files := map[string]bool{"DEV/plan/a.md": true, "DEV/plan/b/INDEX.md": true, "README.md": true}
	dirs := repoDirs(files)
	anchorsOf := func(string) map[string]bool { return map[string]bool{} }
	judge := func(string) bool { return false }
	cases := []struct {
		from, dest string
		bad        bool
	}{
		{"DEV/plan/a.md", "b/INDEX.md", false},
		{"DEV/plan/a.md", "b/MISSING.md", true},
		{"DEV/plan/b/INDEX.md", "../a.md", false},
		{"DEV/plan/b/INDEX.md", "../../../README.md", false},
		{"DEV/plan/a.md", "../../nope.md", true},
	}
	for _, c := range cases {
		got := linkProblemIn(c.from, docLink{line: 1, dest: c.dest}, files, dirs, anchorsOf, judge) != ""
		if got != c.bad {
			t.Errorf("%s -> %s: problem=%v, want %v", c.from, c.dest, got, c.bad)
		}
	}
	// the committed corpus does not judge the same link, on purpose
	if linkProblem("docs/x.md", docLink{line: 1, dest: "../DEV/plan/gone.md"}, files, dirs, anchorsOf) != "" {
		t.Error("a committed document's link into DEV/plan must stay unjudged")
	}
}
