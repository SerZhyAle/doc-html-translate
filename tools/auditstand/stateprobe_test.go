package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const toggleRoot = `Software\Classes\.audit`

func newToggleStore() *memoryRegistry {
	return &memoryRegistry{keys: map[string]map[string]rawRegValue{
		`Software`:                      {},
		`Software\Classes`:              {},
		toggleRoot:                      {"": {1, []byte("Audit.File\x00")}},
		toggleRoot + `\OpenWithProgids`: {"Audit.File": {0, nil}},
	}}
}

func captureToggle(t *testing.T, store registryStore, dir string, facts map[string]string) *stateSnapshot {
	t.Helper()
	s, err := captureState(store, []string{toggleRoot}, map[string]string{"settings": dir}, facts)
	if err != nil {
		t.Fatalf("captureState: %v", err)
	}
	return s
}

func TestToggleFlipIsObservableAndRevertIsByteIdentical(t *testing.T) {
	store := newToggleStore()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"updateCheck":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	before := captureToggle(t, store, dir, map[string]string{"update check": "off"})

	// The flip: a registry value and a settings file change, and the program reports the new fact.
	store.keys[toggleRoot][""] = rawRegValue{1, []byte("Other.File\x00")}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"updateCheck":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	flipped := captureToggle(t, store, dir, map[string]string{"update check": "on"})

	// The revert: same bytes as before, written again (a different mtime must not matter).
	store.keys[toggleRoot][""] = rawRegValue{1, []byte("Audit.File\x00")}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"updateCheck":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reverted := captureToggle(t, store, dir, map[string]string{"update check": "off"})

	c := evaluateToggle("update check", before, flipped, reverted)
	if !c.Observable || !c.Reverted || c.Changes != 3 || c.Leftover != 0 {
		t.Fatalf("check = %+v, want observable, reverted, 3 changes, 0 leftover", c)
	}
	if !c.ok() {
		t.Error("a clean toggle must be ok")
	}
}

func TestToggleThatChangesNothingIsNotObservable(t *testing.T) {
	store := newToggleStore()
	dir := t.TempDir()
	s := captureToggle(t, store, dir, nil)
	c := evaluateToggle("silent", s, s, s)
	if c.Observable || c.ok() {
		t.Fatalf("a flip that changed nothing must fail H14: %+v", c)
	}
	if !strings.Contains(c.Detail, "nothing") {
		t.Errorf("detail = %q", c.Detail)
	}
}

func TestRevertThatLeavesAResidueIsReported(t *testing.T) {
	store := newToggleStore()
	dir := t.TempDir()
	before := captureToggle(t, store, dir, nil)

	store.keys[toggleRoot+`\shell`] = map[string]rawRegValue{}
	store.keys[toggleRoot+`\shell`]["Extra"] = rawRegValue{1, []byte("secret-path\x00")}
	flipped := captureToggle(t, store, dir, nil)
	// The "revert" removes the value but forgets the key itself.
	delete(store.keys[toggleRoot+`\shell`], "Extra")
	reverted := captureToggle(t, store, dir, nil)

	c := evaluateToggle("integration", before, flipped, reverted)
	if !c.Observable || c.Reverted || c.Leftover == 0 {
		t.Fatalf("check = %+v, want observable and not reverted", c)
	}
	if strings.Contains(c.Detail, "secret-path") {
		t.Errorf("a registry value's data leaked into the detail: %q", c.Detail)
	}
	if !strings.Contains(c.Detail, `key `+toggleRoot+`\shell`) {
		t.Errorf("the leftover key is not named: %q", c.Detail)
	}
}

func TestFolderSnapshotDistinguishesAbsentFromEmptyAndIgnoresMtime(t *testing.T) {
	root := t.TempDir()
	absent, err := snapshotFolder(filepath.Join(root, "nope"))
	if err != nil || absent != nil {
		t.Fatalf("absent folder = %v, %v; want nil, nil", absent, err)
	}
	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	tree, err := snapshotFolder(empty)
	if err != nil || tree == nil || len(tree) != 0 {
		t.Fatalf("empty folder = %v, %v; want a non-nil empty tree", tree, err)
	}
	// A removed folder differs from an emptied one.
	a := &stateSnapshot{Folders: map[string]folderTree{"f": absent}}
	b := &stateSnapshot{Folders: map[string]folderTree{"f": tree}}
	if d := diffState(a, b); len(d) != 1 || d[0].Kind != "added" {
		t.Errorf("absent -> empty diff = %v", d)
	}
}

func TestDiffIsDeterministic(t *testing.T) {
	a := &stateSnapshot{Facts: map[string]string{"a": "1", "b": "1", "c": "1"}}
	b := &stateSnapshot{Facts: map[string]string{"a": "2", "b": "2", "c": "2"}}
	first := diffState(a, b).String()
	for i := 0; i < 20; i++ {
		if got := diffState(a, b).String(); got != first {
			t.Fatalf("diff order is not stable: %q vs %q", got, first)
		}
	}
}

func TestH14ToggleVerdictNeverPassesAlone(t *testing.T) {
	good := toggleCheck{Toggle: "a", Observable: true, Reverted: true}
	bad := toggleCheck{Toggle: "b", Observable: true, Reverted: false}
	silent := toggleCheck{Toggle: "c"}

	if v := h14ToggleVerdict(nil); v.Verdict != "advisory" {
		t.Errorf("no toggles: %+v", v)
	}
	if v := h14ToggleVerdict([]toggleCheck{good}); v.Verdict != "advisory" {
		t.Errorf("a clean toggle half must stay advisory until locale/announcement/keyboard report: %+v", v)
	}
	v := h14ToggleVerdict([]toggleCheck{good, bad, silent})
	if v.Verdict != "fail" || !strings.Contains(v.Observed, "b, c") {
		t.Errorf("mixed set: %+v", v)
	}
}
