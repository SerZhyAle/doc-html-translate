package bundledtools

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
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

func TestCacheRoot(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("the cache-dir environment variable is set only for linux and windows")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("LocalAppData", dir)
	if got, want := CacheRoot(), filepath.Join(dir, "doc-html-translate"); got != want {
		t.Errorf("CacheRoot = %q, want %q", got, want)
	}
}

// With no per-user cache dir the tools still need somewhere to live, so the temp dir is used.
func TestCacheRootFallsBackToTempDir(t *testing.T) {
	for _, v := range []string{"XDG_CACHE_HOME", "HOME", "LocalAppData"} {
		t.Setenv(v, "")
	}
	if got, want := CacheRoot(), filepath.Join(os.TempDir(), "doc-html-translate"); got != want {
		t.Errorf("CacheRoot = %q, want %q", got, want)
	}
}

// The folder name covers file names too, so a renamed runtime DLL is a different set.
func TestExtractSetHashCoversNames(t *testing.T) {
	base := t.TempDir()
	a, err := extractSet(fstest.MapFS{"tool/a.dll": {Data: []byte("same")}}, "tool", base, "tool")
	if err != nil {
		t.Fatal(err)
	}
	b, err := extractSet(fstest.MapFS{"tool/b.dll": {Data: []byte("same")}}, "tool", base, "tool")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("a renamed file reused the old folder")
	}
}

func TestExtractSetFailures(t *testing.T) {
	t.Run("root missing from the bundle", func(t *testing.T) {
		if _, err := extractSet(fixtureSet("x"), "absent", t.TempDir(), "tool"); err == nil {
			t.Error("extractSet succeeded with nothing to extract")
		}
	})
	t.Run("cache base is a file", func(t *testing.T) {
		base := filepath.Join(t.TempDir(), "cache")
		if err := os.WriteFile(base, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := extractSet(fixtureSet("x"), "tool", base, "tool"); err == nil {
			t.Error("extractSet succeeded under a file")
		}
	})
	t.Run("destination occupied by a folder", func(t *testing.T) {
		base := t.TempDir()
		set := fixtureSet("x")
		hash, _, err := setHash(set, "tool")
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(base, "tool-"+hash)
		if err := os.MkdirAll(filepath.Join(dir, "tool.exe", "junk"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err = extractSet(set, "tool", base, "tool")
		if err == nil || !strings.Contains(err.Error(), "tool.exe") {
			t.Fatalf("extractSet = %v, want an error naming tool.exe", err)
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.Contains(e.Name(), ".tmp-") {
				t.Errorf("temporary file %s left behind after a failed placement", e.Name())
			}
		}
	})
}

// A file that already holds the right bytes is not rewritten: on Windows it may be running
// in another instance, and replacing it would fail.
func TestPlaceFileLeavesIdenticalFileAlone(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(dst, []byte("same"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(dst, old, old); err != nil {
		t.Fatal(err)
	}
	if err := placeFile(dst, []byte("same")); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !st.ModTime().Equal(old) {
		t.Errorf("identical file was rewritten (mtime %v, want %v)", st.ModTime(), old)
	}
}

func TestPlaceFileFailsWithoutFolder(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "gone", "tool.exe")
	if err := placeFile(dst, []byte("x")); err == nil {
		t.Error("placeFile succeeded into a folder that does not exist")
	}
}
