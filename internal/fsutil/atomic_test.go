package fsutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileReplacesContent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(p, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "new" {
		t.Fatalf("got %q", got)
	}
	assertOnlyEntry(t, p)
}

// A rewrite that fails halfway - the stand-in for Ctrl+C or a full disk - must leave the old
// page whole and no temp file behind.
func TestWriteFailureKeepsOriginal(t *testing.T) {
	p := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(p, []byte("<html>whole page</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("interrupted")
	err := Write(p, 0o644, func(w io.Writer) error {
		_, _ = io.WriteString(w, "<html>half")
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "<html>whole page</html>" {
		t.Fatalf("original damaged: %q", got)
	}
	assertOnlyEntry(t, p)
}

func TestWriteFileCreatesNew(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fresh.html")
	if err := WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o044 == 0 {
		t.Errorf("perm %v: a page must stay readable, not the temp file's 0600", fi.Mode().Perm())
	}
}

func assertOnlyEntry(t *testing.T, p string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(p) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("directory holds %v, want only %s", names, filepath.Base(p))
	}
}
