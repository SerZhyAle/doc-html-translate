package ocr

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeOSD is a stand-in tesseract that behaves like the Windows build on a non-ASCII path: it
// fails unless the image path it is given is plain ASCII and exists, and otherwise prints the
// script lines of `--psm 0`. It records the path it was handed.
const fakeOSD = `#!/bin/sh
printf '%s' "$1" > "$FAKE_TESS_ARG"
if printf '%s' "$1" | LC_ALL=C grep -q '[^ -~]'; then exit 1; fi
test -f "$1" || exit 2
echo "Script: Cyrillic"
echo "Script confidence: 8.24"
`

// Done criterion 4: a book in a Cyrillic-named folder still gets script detection, because the
// detector is handed the same ASCII staging recognition uses.
func TestDetectScriptStagesANonASCIIPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in engine is a shell script")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "tesseract")
	if err := os.WriteFile(bin, []byte(fakeOSD), 0o755); err != nil {
		t.Fatal(err)
	}
	argFile := filepath.Join(dir, "arg.txt")
	t.Setenv("FAKE_TESS_ARG", argFile)

	book := filepath.Join(dir, "Книга")
	if err := os.MkdirAll(book, 0o755); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(book, "страница.png")
	f, err := os.Create(img)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewGray(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	script, conf, ok := DetectScript(bin, img, "")
	if !ok || script != "Cyrillic" || conf != 8.24 {
		t.Fatalf("DetectScript = %q %v %v, want Cyrillic 8.24 true", script, conf, ok)
	}
	got, err := os.ReadFile(argFile)
	if err != nil {
		t.Fatal(err)
	}
	if !isASCIIPath(string(got)) || strings.Contains(string(got), "Книга") {
		t.Errorf("the engine was handed %q, want an ASCII staging copy", got)
	}
	if _, err := os.Stat(string(got)); !os.IsNotExist(err) {
		t.Errorf("the staging copy %q was not removed", got)
	}
}
