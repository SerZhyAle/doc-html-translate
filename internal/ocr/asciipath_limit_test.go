package ocr

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useEngineLimit injects the engine path limit for one test (0 = unlimited).
func useEngineLimit(t *testing.T, limit int) {
	t.Helper()
	saved := maxEnginePath
	maxEnginePath = limit
	t.Cleanup(func() { maxEnginePath = saved })
}

// longASCIIFile writes data to a file whose path is longer than any root a test uses, so a limit
// set between the two stages the file without touching the root.
func longASCIIFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), strings.Repeat("d", 90))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A path is accepted up to and including the limit and refused one character past it.
func TestAsciiFormHonoursTheEnginePathLimit(t *testing.T) {
	// Paths are absolute, as the limit applies to the absolute form; n is the total length.
	root := t.TempDir()
	of := func(n int, fill string) string {
		return filepath.Join(root, strings.Repeat(fill, n-len(root)-1))
	}
	const limit = 20
	limitN := len(root) + 1 + limit
	useEngineLimit(t, limitN)
	for _, tc := range []struct {
		name string
		path string
		ok   bool
	}{
		{"below", of(limitN-1, "a"), true},
		{"at", of(limitN, "a"), true},
		{"above", of(limitN+1, "a"), false},
	} {
		got, ok := asciiForm(tc.path, unchanged)
		if ok != tc.ok || (ok && got != tc.path) {
			t.Errorf("%s: asciiForm(%d chars) = %q, %v; want ok=%v", tc.name, len(tc.path), got, ok, tc.ok)
		}
	}

	// A short name that fits rescues an over-long path, as it does a non-ASCII one.
	long := of(limitN+20, "a")
	alias := of(limitN-5, "b")
	short := func(p string) string {
		if p == long {
			return alias
		}
		return p
	}
	if got, ok := asciiForm(long, short); !ok || got != alias {
		t.Errorf("asciiForm with a short name = %q, %v; want %q", got, ok, alias)
	}
	// A short name that is itself over the limit does not.
	if _, ok := asciiForm(long, func(string) string { return of(limitN+10, "c") }); ok {
		t.Error("asciiForm accepted a short name over the limit")
	}

	useEngineLimit(t, 0)
	if _, ok := asciiForm(of(len(root)+5000, "a"), unchanged); !ok {
		t.Error("asciiForm refused a long path although no limit applies")
	}
}

// An over-long ASCII image is copied under the short staging root, byte for byte, and the copy is
// removed by the cleanup - before this, the long path went to Tesseract as it was and the read failed.
func TestStageASCIIPathCopiesAnOverLongPath(t *testing.T) {
	root := t.TempDir()
	data := []byte("not really a png, but bytes all the same")
	src := longASCIIFile(t, "page.png", data)
	limit := len(root) + stagingNameReserve + 20
	if len(src) <= limit {
		t.Fatalf("test setup: source path (%d) must exceed the limit (%d)", len(src), limit)
	}
	useEngineLimit(t, limit)
	useStagingRoot(t, []string{root}, unchanged)

	staged, cleanup := stageASCIIPath(src)
	if staged == src || filepath.Dir(staged) != root {
		t.Fatalf("stageASCIIPath = %q, want a copy under %s", staged, root)
	}
	if len(staged) > limit {
		t.Errorf("staged path is %d characters, over the limit %d", len(staged), limit)
	}
	if filepath.Ext(staged) != ".png" {
		t.Errorf("staged copy lost its extension: %q", staged)
	}
	if got, err := os.ReadFile(staged); err != nil || !bytes.Equal(got, data) {
		t.Errorf("staged bytes = %q, %v; want the original bytes", got, err)
	}
	cleanup()
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Errorf("cleanup left %s behind (stat err = %v)", staged, err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("cleanup removed the source: %v", err)
	}

	// A path within the limit still goes through untouched.
	short := filepath.Join(root, "in.png")
	if got, cl := stageASCIIPath(short); got != short {
		t.Errorf("stageASCIIPath of a short path = %q, want it unchanged", got)
	} else {
		cl()
	}
}

// A staging root that cannot hold the names created under it is as unusable as a non-ASCII one.
func TestStagingRootRejectsAnOverLongCandidate(t *testing.T) {
	fits := t.TempDir()
	tooLong := filepath.Join(t.TempDir(), strings.Repeat("l", 60))
	useEngineLimit(t, len(fits)+stagingNameReserve+5)

	got, err := resolveStagingRoot([]string{tooLong, fits}, unchanged)
	if err != nil || got != fits {
		t.Errorf("resolveStagingRoot = %q, %v; want the shorter %q", got, err, fits)
	}
	_, err = resolveStagingRoot([]string{tooLong}, unchanged)
	if err == nil || !strings.Contains(err.Error(), tooLong) {
		t.Errorf("resolveStagingRoot with only a long root: err = %v, want one naming %s", err, tooLong)
	}
}

// The in-place branch of prepareForOCR (an image large enough to need no upscale) used to hand the
// long path straight to the engine; the small images that were staged anyway hid it.
func TestPrepareForOCRStagesAnOverLongPathForALargeImage(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 1400, 900))); err != nil {
		t.Fatal(err)
	}
	src := longASCIIFile(t, "big.png", buf.Bytes())
	limit := len(root) + stagingNameReserve + 20
	useEngineLimit(t, limit)
	useStagingRoot(t, []string{root}, unchanged)

	frame, scale, _, cleanup := prepareForOCR(src)
	defer cleanup()
	if scale != 1 {
		t.Fatalf("scale = %d, want 1 (an image this size is not enlarged)", scale)
	}
	if frame.path == src || len(frame.path) > limit || filepath.Dir(frame.path) != root {
		t.Errorf("frame.path = %q (%d chars), want a copy under %s within %d", frame.path, len(frame.path), root, limit)
	}
	if got, err := os.ReadFile(frame.path); err != nil || !bytes.Equal(got, buf.Bytes()) {
		t.Errorf("staged image differs from the source (read err = %v)", err)
	}
}

// A relative path is short but the engine resolves it against its working directory, so the limit
// must be measured on the absolute form (the OCR lab's 260-character cartoon path was relative).
func TestFitsEngineMeasuresTheAbsolutePath(t *testing.T) {
	abs, err := filepath.Abs("scene.png")
	if err != nil {
		t.Fatal(err)
	}
	useEngineLimit(t, len(abs)-1)
	if fitsEngine("scene.png", 0) {
		t.Errorf("a relative name whose absolute form is %d characters fits a limit of %d", len(abs), len(abs)-1)
	}
	useEngineLimit(t, len(abs))
	if !fitsEngine("scene.png", 0) {
		t.Errorf("a relative name exactly at the limit was refused")
	}
}
