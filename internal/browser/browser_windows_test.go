//go:build windows

package browser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// TestOpenPassesTargetVerbatim: names that cmd.exe would read as syntax reach the
// shell unchanged, as a file argument with no parameters, and no interpreter runs.
func TestOpenPassesTargetVerbatim(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"a&calc&b.html", "Q&A.html", "50%.html", "%PATH%.html", "a^b.html", "!x!.html",
		"(1).html", "a,b;c.html", "with space.html", "Книга.html",
	}
	orig := shellExecute
	t.Cleanup(func() { shellExecute = orig })

	for _, name := range append(names, dir+`\`) {
		target := name
		if name != dir+`\` {
			target = filepath.Join(dir, name)
		}
		var gotFile string
		var gotArgs *uint16
		shellExecute = func(_ windows.Handle, verb, file, args, _ *uint16, _ int32) error {
			if windows.UTF16PtrToString(verb) != "open" {
				t.Errorf("verb = %q", windows.UTF16PtrToString(verb))
			}
			gotFile = windows.UTF16PtrToString(file)
			gotArgs = args
			return nil
		}
		if err := Open(target); err != nil {
			t.Fatalf("Open(%q): %v", target, err)
		}
		if gotFile != target {
			t.Errorf("shell got %q, want %q", gotFile, target)
		}
		if gotArgs != nil {
			t.Errorf("Open(%q) passed parameters to the shell", target)
		}
	}
}

// fakeShell records the file each Open hands to the shell and answers with err.
func fakeShell(t *testing.T, err error) *[]string {
	t.Helper()
	var files []string
	orig := shellExecute
	t.Cleanup(func() { shellExecute = orig })
	shellExecute = func(_ windows.Handle, _, file, _, _ *uint16, _ int32) error {
		files = append(files, windows.UTF16PtrToString(file))
		return err
	}
	return &files
}

func TestOpenReportsShellFailure(t *testing.T) {
	cause := windows.ERROR_FILE_NOT_FOUND
	fakeShell(t, cause)
	err := Open(`C:\missing\index.html`)
	if !errors.Is(err, cause) || !strings.HasPrefix(err.Error(), "open browser: ") {
		t.Errorf("Open = %v, want the shell error wrapped", err)
	}
}

// A target that cannot be encoded for the shell is refused before anything is launched.
func TestOpenRejectsNULInTarget(t *testing.T) {
	files := fakeShell(t, nil)
	if err := Open("book\x00.html"); err == nil {
		t.Error("Open accepted a target with a NUL byte")
	}
	if len(*files) != 0 {
		t.Errorf("shell was called with %q", *files)
	}
}

// A legacy chapter link opens the book's index.html instead, found by walking up.
func TestOpenRedirectsLegacyChapterToIndex(t *testing.T) {
	root := t.TempDir()
	chapter := filepath.Join(root, "OEBPS", "Text", "ch01.xhtml")
	writeFile(t, chapter)
	index := filepath.Join(root, "index.html")
	writeFile(t, index)

	files := fakeShell(t, nil)
	if err := Open(chapter); err != nil {
		t.Fatal(err)
	}
	if len(*files) != 1 || (*files)[0] != index {
		t.Errorf("shell got %q, want %q", *files, index)
	}
}

func TestNormalizeTarget(t *testing.T) {
	root := t.TempDir()
	book := filepath.Join(root, "book")
	writeFile(t, filepath.Join(book, "index.html"))
	nearest := filepath.Join(book, "part", "index.html")
	writeFile(t, nearest)
	// A directory named index.html is not a page to open.
	if err := os.MkdirAll(filepath.Join(root, "orphan", "index.html"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct{ name, in, want string }{
		{"nearest index wins", filepath.Join(book, "part", "ch.XHTM"), nearest},
		{"walks up to book index", filepath.Join(book, "a", "b", "ch.xhtml"), filepath.Join(book, "index.html")},
		{"no index anywhere", filepath.Join(root, "orphan", "ch.xhtml"), filepath.Join(root, "orphan", "ch.xhtml")},
		{"html is opened as-is", filepath.Join(book, "part", "ch.html"), filepath.Join(book, "part", "ch.html")},
		{"url is opened as-is", "HTTPS://example.com/ch.xhtml", "HTTPS://example.com/ch.xhtml"},
		{"blank is left alone", "  ", "  "},
	}
	for _, tc := range cases {
		if got := normalizeTarget(tc.in); got != tc.want {
			t.Errorf("%s: normalizeTarget(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func writeFile(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
}
