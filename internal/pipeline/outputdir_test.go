package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/outputpath"
)

func runOn(t *testing.T, input string, force bool) (int, error) {
	t.Helper()
	return NewRunner(config.Config{InputFile: input, NoOpen: true, NoTranslate: true, SinglePage: true, Force: force}).Run()
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("%s missing: %v", p, err)
	}
}

func mustNotExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("%s should not exist (err=%v)", p, err)
	}
}

// Reproduced data loss: a folder passed as input mapped its output onto itself and a
// failed plain-text read then deleted it recursively.
func TestRunRefusesFolderInputAndLeavesItIntact(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "photos")
	writeFile(t, filepath.Join(dir, "a.jpg"), "x")

	code, err := runOn(t, dir, false)
	if err == nil || code != ExitArgsError {
		t.Fatalf("code=%d err=%v, want an argument error", code, err)
	}
	mustExist(t, filepath.Join(dir, "a.jpg"))
}

func TestRunRefusesEmptyFile(t *testing.T) {
	in := filepath.Join(t.TempDir(), "empty.txt")
	writeFile(t, in, "")
	if code, err := runOn(t, in, false); err == nil || code != ExitArgsError {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

// A failed conversion next to the user's own same-named folder used to delete it.
func TestFailedRunKeepsForeignSameNamedFolder(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "Tolkien.epub")
	writeFile(t, in, "not a zip")
	writeFile(t, filepath.Join(dir, "Tolkien", "notes.txt"), "mine")

	if _, err := runOn(t, in, false); err == nil {
		t.Fatal("corrupt epub converted")
	}
	mustExist(t, filepath.Join(dir, "Tolkien", "notes.txt"))
	mustNotExist(t, filepath.Join(dir, "Tolkien (epub)"))
}

// A pre-existing empty folder is used, but it was not this run's to remove.
func TestFailedRunEmptiesButKeepsPreexistingEmptyFolder(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "book.epub")
	writeFile(t, in, "not a zip")
	out := filepath.Join(dir, "book")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := runOn(t, in, false); err == nil {
		t.Fatal("corrupt epub converted")
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatalf("pre-existing folder removed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed run left files behind: %v", entries)
	}
}

func TestFailedRunRemovesFolderItCreated(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "book.epub")
	writeFile(t, in, "not a zip")
	if _, err := runOn(t, in, false); err == nil {
		t.Fatal("corrupt epub converted")
	}
	mustNotExist(t, filepath.Join(dir, "book"))
}

func TestRunMarksReusesAndSeparatesOutputs(t *testing.T) {
	dir := t.TempDir()
	txt := filepath.Join(dir, "book.txt")
	writeFile(t, txt, "First paragraph.\n\nSecond paragraph.\n")
	out := filepath.Join(dir, "book")

	if code, err := runOn(t, txt, false); err != nil || code != ExitOK {
		t.Fatalf("convert: code=%d err=%v", code, err)
	}
	mustExist(t, filepath.Join(out, "index.html"))
	mustExist(t, filepath.Join(out, outputpath.MarkerName))
	mustNotExist(t, filepath.Join(out, outputpath.LockName))

	// A sentinel survives a reuse and is gone after -force: the rebuild happened.
	sentinel := filepath.Join(out, "sentinel")
	writeFile(t, sentinel, "")
	if _, err := runOn(t, txt, false); err != nil {
		t.Fatalf("reuse: %v", err)
	}
	mustExist(t, sentinel)
	if _, err := runOn(t, txt, true); err != nil {
		t.Fatalf("force: %v", err)
	}
	mustNotExist(t, sentinel)
	mustExist(t, filepath.Join(out, "index.html"))

	// A different document with the same base name gets its own folder.
	md := filepath.Join(dir, "book.md")
	writeFile(t, md, "# Other\n\nText.\n")
	if _, err := runOn(t, md, false); err != nil {
		t.Fatalf("md: %v", err)
	}
	mustExist(t, filepath.Join(dir, "book (md)", "index.html"))
	if data, _ := os.ReadFile(filepath.Join(out, "index.html")); !strings.Contains(string(data), "First paragraph") {
		t.Fatal("book.md overwrote book.txt's output")
	}
}

// A run that cannot take the lock must neither write nor delete anything.
func TestRunRefusesLockedOutputWithoutTouchingIt(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "book.epub")
	writeFile(t, in, "not a zip")
	out := filepath.Join(dir, "book")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	// Another live run of this same book is mid-conversion here.
	if err := outputpath.WriteMarker(out, in); err != nil {
		t.Fatal(err)
	}
	lock, err := outputpath.AcquireLock(out)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	writeFile(t, filepath.Join(out, "in-progress.html"), "")

	for _, force := range []bool{false, true} {
		if code, err := runOn(t, in, force); err == nil || code != ExitIOError {
			t.Fatalf("force=%v: code=%d err=%v, want a lock refusal", force, code, err)
		}
		mustExist(t, filepath.Join(out, "in-progress.html"))
	}

	// Its index.html may already exist; reuse must not open a half-written output.
	writeFile(t, filepath.Join(out, "index.html"), "")
	if code, err := runOn(t, in, false); err == nil || code != ExitIOError {
		t.Fatalf("reuse: code=%d err=%v, want a lock refusal", code, err)
	}
}
