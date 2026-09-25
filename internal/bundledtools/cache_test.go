package bundledtools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func fixtureSet(exe string) fstest.MapFS {
	return fstest.MapFS{
		"tool/tool.exe":   {Data: []byte(exe)},
		"tool/helper.dll": {Data: []byte("runtime library")},
	}
}

func TestExtractSetIsKeyedByContent(t *testing.T) {
	base := t.TempDir()
	v1, err := extractSet(fixtureSet("build one"), "tool", base, "tool")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(v1), "tool-") || len(filepath.Base(v1)) != len("tool-")+hashLen {
		t.Errorf("folder %q is not tool-<%d hex>", filepath.Base(v1), hashLen)
	}
	again, err := extractSet(fixtureSet("build one"), "tool", base, "tool")
	if err != nil || again != v1 {
		t.Fatalf("same content went to %q (err %v), want %q", again, err, v1)
	}

	// An app upgrade: the new build must land in a new folder and be the one returned.
	v2, err := extractSet(fixtureSet("build two"), "tool", base, "tool")
	if err != nil {
		t.Fatal(err)
	}
	if v2 == v1 {
		t.Fatal("a changed bundle reused the old folder")
	}
	got, err := os.ReadFile(filepath.Join(v2, "tool.exe"))
	if err != nil || string(got) != "build two" {
		t.Fatalf("tool.exe = %q (err %v), want the new build", got, err)
	}
	if _, err := os.Stat(v1); !os.IsNotExist(err) {
		t.Errorf("the old folder %s was not pruned", v1)
	}
}

func TestExtractSetPrunesOnlyItsOwnFolders(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"tool", "tool-0123456789ab", "tool-notahash", "tessdata"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := extractSet(fixtureSet("x"), "tool", base, "tool"); err != nil {
		t.Fatal(err)
	}
	for d, want := range map[string]bool{"tool": false, "tool-0123456789ab": false, "tool-notahash": true, "tessdata": true} {
		_, err := os.Stat(filepath.Join(base, d))
		if exists := err == nil; exists != want {
			t.Errorf("%s exists = %v, want %v", d, exists, want)
		}
	}
}

// A file left half-written by an older, non-atomic build is replaced, and no temporary file
// is left behind.
func TestExtractSetRepairsAPartialFile(t *testing.T) {
	base := t.TempDir()
	set := fixtureSet("full executable")
	dir, err := extractSet(set, "tool", base, "tool")
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "tool.exe")
	if err := os.WriteFile(exe, []byte("full ex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := extractSet(set, "tool", base, "tool"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "full executable" {
		t.Errorf("tool.exe = %q, want it repaired", got)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temporary file %s left behind", e.Name())
		}
	}
}

// Several instances starting at once all get a complete copy.
func TestExtractSetConcurrent(t *testing.T) {
	base := t.TempDir()
	payload := bytes.Repeat([]byte("pdftotext"), 1<<16)
	set := fstest.MapFS{"tool/tool.exe": {Data: payload}}

	const n = 8
	dirs := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			dirs[i], errs[i] = extractSet(set, "tool", base, "tool")
		}(i)
	}
	wg.Wait()
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("instance %d: %v", i, errs[i])
		}
		if dirs[i] != dirs[0] {
			t.Fatalf("instance %d used %s, instance 0 used %s", i, dirs[i], dirs[0])
		}
	}
	got, err := os.ReadFile(filepath.Join(dirs[0], "tool.exe"))
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("tool.exe is %d bytes (err %v), want %d", len(got), err, len(payload))
	}
}
