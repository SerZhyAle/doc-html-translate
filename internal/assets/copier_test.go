package assets

import (
	"os"
	"path/filepath"
	"testing"
)

// A stylesheet whose write fails after the files it references were copied must be forgotten,
// not the last of those files: otherwise a later reference to the same sheet resolves, by file
// identity, to the name of a copy that was never written.
func TestFailedSheetIsForgottenNotItsLastImport(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.css"), []byte(`body{background:url(img.png)}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "img.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A directory under the sheet's output name makes that one write fail; the image still copies.
	if err := os.Mkdir(filepath.Join(out, "a.css"), 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := NewCopier(src, out)
	if err != nil {
		t.Fatal(err)
	}
	if name, ok := c.LinkedStylesheet("a.css", ""); ok {
		t.Fatalf("first link reported %q as written", name)
	}
	if _, err := os.Stat(filepath.Join(out, "img.png")); err != nil {
		t.Fatalf("the referenced image was not copied: %v", err)
	}

	// The same sheet spelled differently: the failed copy must not be reused.
	name, ok := c.LinkedStylesheet("./a.css", "")
	if !ok {
		t.Fatal("second link of the sheet failed")
	}
	if st, err := os.Stat(filepath.Join(out, name)); err != nil || st.IsDir() {
		t.Fatalf("second link points at %q, which is not a written stylesheet (err %v)", name, err)
	}
	// The image copied by the failed attempt is still known, so it is not copied twice.
	if got, out := c.place("img.png", c.Root(), false); out != copied || got != "img.png" {
		t.Errorf("image resolves to %q, %v; want the first copy img.png", got, out)
	}
}
