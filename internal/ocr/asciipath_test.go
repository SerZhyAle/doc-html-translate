package ocr

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// useStagingRoot points the staging root at candidates for one test, resolved as the app would.
func useStagingRoot(t *testing.T, candidates []string, short func(string) string) {
	t.Helper()
	saved := stagingRoot
	stagingRoot = sync.OnceValues(func() (string, error) { return resolveStagingRoot(candidates, short) })
	t.Cleanup(func() { stagingRoot = saved })
}

// nonASCIIDir makes a folder named the way a Cyrillic Windows profile is, or skips when the file
// system here cannot hold such a name.
func nonASCIIDir(t *testing.T, parent string) string {
	t.Helper()
	dir := filepath.Join(parent, "Пользователь")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skipf("cannot create a non-ASCII folder on this file system: %v", err)
	}
	return dir
}

func unchanged(p string) string { return p }

// A profile named outside the ANSI code page puts the temp folder outside it too. Staging there
// handed Tesseract a path it mangles, so the upscale, rotate, rescue and screen passes all failed.
func TestStagingRootSkipsANonASCIITempFolder(t *testing.T) {
	temp := nonASCIIDir(t, t.TempDir())
	fallback := filepath.Join(t.TempDir(), "fallback")

	got, err := resolveStagingRoot([]string{temp, fallback}, unchanged)
	if err != nil || got != fallback {
		t.Errorf("resolveStagingRoot = %q, %v; want the ASCII fallback %q", got, err, fallback)
	}

	// Where the volume keeps 8.3 names the temp folder itself is usable through its short form.
	alias := t.TempDir()
	short := func(p string) string {
		if p == temp {
			return alias
		}
		return p
	}
	if got, err := resolveStagingRoot([]string{temp, fallback}, short); err != nil || got != alias {
		t.Errorf("resolveStagingRoot with a short name = %q, %v; want %q", got, err, alias)
	}

	// Nothing usable is a reason to skip, and the reason names what was tried.
	_, err = resolveStagingRoot([]string{temp}, unchanged)
	if err == nil || !strings.Contains(err.Error(), temp) {
		t.Errorf("resolveStagingRoot with no ASCII folder: err = %v, want one naming %s", err, temp)
	}
}

// Every image handed to the engine - a derived rendition or a copy of the book's own file - lands
// on an ASCII path even when the system temp folder is not one.
func TestStagedImagesAvoidANonASCIITempFolder(t *testing.T) {
	temp := nonASCIIDir(t, t.TempDir())
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	fallback := filepath.Join(t.TempDir(), "fallback")
	useStagingRoot(t, []string{os.TempDir(), fallback}, unchanged)

	path, cleanup, ok := writeTempPNG(image.NewGray(image.Rect(0, 0, 4, 4)))
	if !ok {
		t.Fatal("writeTempPNG failed")
	}
	defer cleanup()
	if !isASCIIPath(path) || filepath.Dir(path) != fallback {
		t.Errorf("writeTempPNG wrote %q, want an ASCII path under %s", path, fallback)
	}

	src := filepath.Join(temp, "страница.png")
	if err := os.WriteFile(src, []byte("png bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged, cleanupCopy := stageASCIIPath(src)
	defer cleanupCopy()
	if !isASCIIPath(staged) || filepath.Dir(staged) != fallback {
		t.Errorf("stageASCIIPath = %q, want an ASCII copy under %s", staged, fallback)
	}
}

// The per-user data folder sits under the same profile. Handed over as --tessdata-dir it failed
// every run once a pack was downloaded, so the packs are mirrored under an ASCII root instead.
func TestPrepareEngineMirrorsANonASCIIDataFolder(t *testing.T) {
	data := nonASCIIDir(t, t.TempDir())
	pack := []byte("eng traineddata")
	if err := os.WriteFile(langFile(data, "eng"), pack, 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	useStagingRoot(t, []string{root}, unchanged)

	got, err := PrepareEngine(data)
	if err != nil {
		t.Fatalf("PrepareEngine: %v", err)
	}
	if !isASCIIPath(got) || !strings.HasPrefix(got, root) {
		t.Fatalf("PrepareEngine = %q, want an ASCII folder under %s", got, root)
	}
	if b, err := os.ReadFile(langFile(got, "eng")); err != nil || !bytes.Equal(b, pack) {
		t.Errorf("mirrored eng pack = %q, %v; want the original bytes", b, err)
	}
	// A second run reuses the mirror rather than copying the pack again.
	if again, err := PrepareEngine(data); err != nil || again != got {
		t.Errorf("second PrepareEngine = %q, %v; want %q", again, err, got)
	}

	ascii := t.TempDir()
	if err := os.WriteFile(langFile(ascii, "eng"), pack, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := PrepareEngine(ascii); err != nil || got != ascii {
		t.Errorf("PrepareEngine of an ASCII folder = %q, %v; want it unchanged", got, err)
	}
	// A folder without packs is never handed to the engine, so it needs no mirror.
	empty := nonASCIIDir(t, t.TempDir())
	if got, err := PrepareEngine(empty); err != nil || got != empty {
		t.Errorf("PrepareEngine of an empty folder = %q, %v; want it unchanged", got, err)
	}
}

// When no ASCII root exists anywhere, OCR is skipped with the folder named, not attempted.
func TestPrepareEngineNamesTheFolderWhenNothingIsUsable(t *testing.T) {
	data := nonASCIIDir(t, t.TempDir())
	if err := os.WriteFile(langFile(data, "eng"), []byte("eng"), 0o644); err != nil {
		t.Fatal(err)
	}
	useStagingRoot(t, []string{data}, unchanged)
	_, err := PrepareEngine(data)
	if err == nil || !strings.Contains(err.Error(), data) {
		t.Errorf("PrepareEngine with no ASCII root: err = %v, want one naming %s", err, data)
	}
}
