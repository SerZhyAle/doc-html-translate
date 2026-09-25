//go:build windows

package browser

import (
	"path/filepath"
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
