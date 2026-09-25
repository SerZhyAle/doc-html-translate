//go:build !windows

package dialog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withStdio points os.Stdin at a file holding input and captures what the code under test
// writes to os.Stdout and os.Stderr.
func withStdio(t *testing.T, input string, run func()) (stdout, stderr string) {
	t.Helper()
	dir := t.TempDir()
	open := func(name, content string) *os.File {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(p, os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	in, out, errOut := open("in", input), open("out", ""), open("err", "")
	origIn, origOut, origErr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = in, out, errOut
	defer func() { os.Stdin, os.Stdout, os.Stderr = origIn, origOut, origErr }()
	run()
	o, _ := os.ReadFile(out.Name())
	e, _ := os.ReadFile(errOut.Name())
	return string(o), string(e)
}

func TestConfirm(t *testing.T) {
	t.Setenv(HostEnv, "")
	for input, want := range map[string]bool{
		"y\n":       true,
		" YES \n":   true,
		"yes":       true,
		"n\n":       false,
		"\n":        false,
		"":          false, // stdin closed: no answer is not consent
		"yess\n":    false,
		"sure\ny\n": false,
	} {
		var got bool
		stdout, _ := withStdio(t, input, func() { got = Confirm("Register", "Make this app the default?") })
		if got != want {
			t.Errorf("answer %q: Confirm = %v, want %v", input, got, want)
		}
		if !strings.Contains(stdout, "Register") || !strings.Contains(stdout, "Make this app the default?") {
			t.Errorf("prompt %q does not show the title and message", stdout)
		}
	}
}

func TestShowWarningGoesToStderr(t *testing.T) {
	stdout, stderr := withStdio(t, "", func() { ShowWarning("PDF", "pdftotext was blocked") })
	if stdout != "" {
		t.Errorf("warning leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "PDF") || !strings.Contains(stderr, "pdftotext was blocked") {
		t.Errorf("stderr = %q, want the title and message", stderr)
	}
}
