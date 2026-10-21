package settingsinv

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The gate is only worth having if it fails when it should. Each case copies the files the gate reads into a scratch
// tree, breaks one thing APP-SETTINGS rule 11 names, and expects exactly that to be reported.

const repoRoot = "../.."

var gateFiles = []string{
	UIPath, I18NPath, ServerPath, FlagsPath, DeckDir, InventoryPath,
	"README.md", "README_RU.md", "README_UK.md", "cmd/doc-html-ui/help/help-map.json",
}

func scratch(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	for _, rel := range gateFiles {
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			r, _ := filepath.Rel(repoRoot, p)
			to := filepath.Join(dst, r)
			if d.IsDir() {
				return os.MkdirAll(to, 0o755)
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return err
			}
			return os.WriteFile(to, raw, 0o644)
		})
		if err != nil {
			t.Fatalf("copy %s: %v", rel, err)
		}
	}
	return dst
}

func edit(t *testing.T, root, rel string, f func(string) string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	out := f(string(raw))
	if out == string(raw) {
		t.Fatalf("the edit of %s changed nothing - the case tests nothing", rel)
	}
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, root string) *Report {
	t.Helper()
	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func mustFail(t *testing.T, r *Report, want string) {
	t.Helper()
	for _, f := range r.Failures {
		if strings.Contains(f, want) {
			return
		}
	}
	t.Errorf("want a failure naming %q, got %q", want, r.Failures)
}

func TestTheRepositoryPasses(t *testing.T) {
	r := run(t, repoRoot)
	for _, f := range r.Failures {
		t.Error(f)
	}
	if r.Settings < 20 || r.Context != 6 || r.Flags < 15 {
		t.Errorf("the counts look wrong: %+v", r)
	}
}

func TestAPersistedKeyThatIsNeitherASettingNorContextFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, "        pageScrolls:    pageScrolls,\n", "        pageScrolls:    pageScrolls,\n        mystery:        someState,\n", 1)
	})
	mustFail(t, run(t, root), `persists "mystery"`)
}

func TestASettingTheManualDoesNotNameFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, DeckDir+"/converter.json", func(s string) string { return strings.ReplaceAll(s, "{k:queueParallel}", "the parallel setting") })
	mustFail(t, run(t, root), `has neither a nameKey nor a {k:..} reference to "queueParallel"`)
}

func TestARenderThatDriftedFromTheWindowFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, InventoryPath, func(s string) string {
		return strings.Replace(s, `"captionEN": "Split size"`, `"captionEN": "Chunk size"`, 1)
	})
	mustFail(t, run(t, root), "differs from what the window says now")

	// a changed default of the window is a changed render too
	root = scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, `id="splitSize" value="0"`, `id="splitSize" value="5"`, 1)
	})
	mustFail(t, run(t, root), "differs from what the window says now")

	// and a missing render is not a pass
	root = scratch(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(InventoryPath))); err != nil {
		t.Fatal(err)
	}
	mustFail(t, run(t, root), "is missing")
}

func TestASettingThatMapsToAFlagNeedsTheReadmeRowInEveryLanguage(t *testing.T) {
	root := scratch(t)
	edit(t, root, "README_UK.md", func(s string) string {
		var keep []string
		for _, l := range strings.Split(s, "\n") {
			if !strings.HasPrefix(l, "| `-max-cost`") {
				keep = append(keep, l)
			}
		}
		return strings.Join(keep, "\n")
	})
	mustFail(t, run(t, root), "-max-cost, which has no row in the flag table of README_UK.md")
}

func TestARegionTheHelpMapDoesNotListFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, `<div class="opts" id="togglesRow" data-help="main.options">`, `<div class="opts" id="togglesRow" data-help="main.nowhere">`, 1)
	})
	mustFail(t, run(t, root), "main.nowhere is not a row of")
}

func TestRenderWritesWhatTheGateReads(t *testing.T) {
	root := scratch(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(InventoryPath))); err != nil {
		t.Fatal(err)
	}
	if r, err := Render(root); err != nil || len(r.Failures) > 0 {
		t.Fatalf("Render: %v %q", err, r.Failures)
	}
	if r := run(t, root); len(r.Failures) != 0 {
		t.Fatalf("a rendered inventory must pass: %q", r.Failures)
	}
	a, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(InventoryPath)))
	b, _ := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(InventoryPath)))
	if string(a) != string(b) {
		t.Error("the render of a copy of the tree is not the committed render")
	}
}
