package runner

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
)

func useLimit(t *testing.T, limit int) {
	t.Helper()
	saved := pathLimit
	pathLimit = func() int { return limit }
	t.Cleanup(func() { pathLimit = saved })
}

func writeInput(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, makeGreyPNG(t, 32, 32), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// The three kinds are told apart by the file system and the limit, in diagnostic order.
func TestClassifyFailureUsesFilesystemFacts(t *testing.T) {
	input := writeInput(t, t.TempDir(), "in.png")
	missing := filepath.Join(t.TempDir(), "gone.png")

	useLimit(t, 259)
	for _, tc := range []struct {
		name  string
		input string
		chars int
		want  evidence.ErrorKind
	}{
		{"missing input", missing, 40, evidence.ErrInputMissing},
		{"missing wins over a long path", missing, 400, evidence.ErrInputMissing},
		{"path past the limit", input, 260, evidence.ErrEnginePathIncompatible},
		{"path at the limit", input, 259, evidence.ErrEngineFailed},
		{"short path", input, 80, evidence.ErrEngineFailed},
	} {
		if got := classifyFailure(tc.input, tc.chars); got != tc.want {
			t.Errorf("%s: classifyFailure = %q, want %q", tc.name, got, tc.want)
		}
	}

	useLimit(t, 0)
	if got := classifyFailure(input, 5000); got != evidence.ErrEngineFailed {
		t.Errorf("no limit: classifyFailure = %q, want %q", got, evidence.ErrEngineFailed)
	}
}

// failureFor measures what is on disk; the message plays no part in the kind.
func TestFailureForRecordsKindAndPathChars(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "run")
	input := writeInput(t, root, "in.png")
	page := filepath.Join(out, PagesDir, "s1")
	if err := os.MkdirAll(page, 0o755); err != nil {
		t.Fatal(err)
	}
	deep := writeInput(t, page, "a-rather-long-file-name-for-the-copied-page-image.png")

	useLimit(t, len(deep)-1)
	sc := evidence.Scene{SceneID: "s1"}
	failureFor(&sc, input, out, "convert: cannot read input file")
	if sc.ErrorKind != evidence.ErrEnginePathIncompatible || sc.PathChars != len(deep) {
		t.Errorf("kind = %q, pathChars = %d; want %q, %d", sc.ErrorKind, sc.PathChars,
			evidence.ErrEnginePathIncompatible, len(deep))
	}

	useLimit(t, len(deep)+50)
	sc = evidence.Scene{SceneID: "s1"}
	failureFor(&sc, input, out, "convert: cannot read input file")
	if sc.ErrorKind != evidence.ErrEngineFailed {
		t.Errorf("same message under the limit: kind = %q, want %q", sc.ErrorKind, evidence.ErrEngineFailed)
	}

	if err := os.Remove(input); err != nil {
		t.Fatal(err)
	}
	sc = evidence.Scene{SceneID: "s1"}
	failureFor(&sc, input, out, "convert: cannot read input file")
	if sc.ErrorKind != evidence.ErrInputMissing {
		t.Errorf("same message, input gone: kind = %q, want %q", sc.ErrorKind, evidence.ErrInputMissing)
	}
}

func budgetScenes() []*corpus.Scene {
	return []*corpus.Scene{{ID: "scene-one", File: "nested/photo.png"}}
}

func TestPathBudgetCountsOnlyWhatTheBrowserLoads(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	abs, _ := filepath.Abs(out)
	worst := len(filepath.Join(abs, PagesDir, "scene-one", pageLabName))
	if l := len(filepath.Join(abs, PagesDir, "scene-one", "photo.png")); l > worst {
		worst = l
	}

	useLimit(t, worst)
	if err := requirePathBudget(budgetScenes(), out); err != nil {
		t.Errorf("a path exactly at the limit was refused: %v", err)
	}
	useLimit(t, worst-1)
	err := requirePathBudget(budgetScenes(), out)
	if err == nil {
		t.Fatal("a path one past the limit was accepted")
	}
	for _, want := range []string{"refusing to run", "limit of " + strconv.Itoa(worst-1), "browser-loaded", "-out"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	// Screenshots are written by this program, so a limit that only they exceed refuses nothing.
	shot := len(filepath.Join(abs, ShotsDir, "scene-one", "phone-rtl-arabic-hidden.png"))
	if shot <= worst {
		t.Fatalf("test setup: the screenshot path (%d) should be the longest (page %d)", shot, worst)
	}
	useLimit(t, worst)
	if err := requirePathBudget(budgetScenes(), out); err != nil {
		t.Errorf("a screenshot path past the limit refused the run: %v", err)
	}
	useLimit(t, 0)
	if err := requirePathBudget(budgetScenes(), out); err != nil {
		t.Errorf("no limit: %v", err)
	}
}

// Run refuses before it freezes the corpus, starts a browser or creates anything under OutDir.
func TestRunRefusesAnOverLongOutputRootBeforeAnyScene(t *testing.T) {
	root := t.TempDir()
	media := writeInput(t, root, "photo.png")
	sum, err := corpus.HashFile(media)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "manifest.json")
	m := &corpus.Manifest{Scenes: []corpus.Scene{{ID: "scene-one", File: "photo.png", SHA256: sum, Split: corpus.SplitDev}}}
	if err := corpus.Save(manifest, m); err != nil {
		t.Fatal(err)
	}

	useLimit(t, 80)
	out := filepath.Join(root, strings.Repeat("deep", 20), "run")
	_, err = Run(Options{Manifest: manifest, Root: root, OutDir: out, Split: "all"})
	if err == nil || !strings.Contains(err.Error(), "supported limit of 80") {
		t.Fatalf("Run err = %v, want a refusal naming the limit 80", err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("Run created %s before refusing (stat err = %v)", out, statErr)
	}
}
