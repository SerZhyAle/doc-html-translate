package fdsec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/dialog"
)

func TestEnvSource(t *testing.T) {
	t.Setenv("FDSEC_TEST_PW", "from-env")
	a := NewAsker("FDSEC_TEST_PW")
	got, err := a.Secret("x.fd-sec", false)
	if err != nil || string(got) != "from-env" {
		t.Fatalf("got %q %v", got, err)
	}
	if a.CanRetry() {
		t.Error("a variable cannot give a different answer: no retry")
	}
	// Read once and kept: changing the variable mid-run changes nothing.
	t.Setenv("FDSEC_TEST_PW", "changed")
	again, _ := a.Secret("x.fd-sec", true)
	if string(again) != "from-env" {
		t.Errorf("the variable must be read once per run, got %q", again)
	}
}

func TestEnvSourceUnsetAndEmpty(t *testing.T) {
	for name, set := range map[string]bool{"FDSEC_TEST_UNSET": false, "FDSEC_TEST_EMPTY": true} {
		if set {
			t.Setenv(name, "")
		} else {
			_ = os.Unsetenv(name)
		}
		_, err := NewAsker(name).Secret("x.fd-sec", false)
		var fe *Error
		if !errors.As(err, &fe) || fe.Class != EmptyEnv || fe.Name != name {
			t.Errorf("%s: got %v", name, err)
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the message must name the variable: %v", err)
		}
	}
}

func TestNoSourceOffATerminal(t *testing.T) {
	t.Setenv(dialog.HostEnv, "")
	consoleAvailable = func() bool { return false }
	t.Cleanup(func() { consoleAvailable = stdinIsConsole })
	a := NewAsker("")
	_, err := a.Secret("x.fd-sec", false)
	var fe *Error
	if !errors.As(err, &fe) || fe.Class != NoSource {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "-fdsec-password-env") {
		t.Errorf("the usage message must name the source: %v", err)
	}
	if a.CanRetry() {
		t.Error("no retry without a source")
	}
}

func TestNamedVariableBeatsTheConsoleAndTheWindow(t *testing.T) {
	t.Setenv(dialog.HostEnv, dialog.HostStdio)
	consoleAvailable = func() bool { return true }
	t.Cleanup(func() { consoleAvailable = stdinIsConsole })
	if _, ok := NewAsker("SOME_VAR").(*envAsker); !ok {
		t.Error("an explicit variable name must win")
	}
	if _, ok := NewAsker("").(hostAsker); !ok {
		t.Error("the GUI host must win over the console")
	}
	t.Setenv(dialog.HostEnv, "")
	if _, ok := NewAsker("").(consoleAsker); !ok {
		t.Error("a console must be used when nothing else applies")
	}
}

// withStdio swaps the process's stdin and stdout for files, as the GUI's pipes would be.
func withStdio(t *testing.T, answer string, run func()) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in"), []byte(answer), 0o600); err != nil {
		t.Fatal(err)
	}
	fin, err := os.Open(filepath.Join(dir, "in"))
	if err != nil {
		t.Fatal(err)
	}
	defer fin.Close()
	fout, err := os.Create(filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	origIn, origOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = fin, fout
	run()
	os.Stdin, os.Stdout = origIn, origOut
	_ = fout.Close()
	b, _ := os.ReadFile(filepath.Join(dir, "out"))
	return string(b)
}

func TestHostSource(t *testing.T) {
	t.Setenv(dialog.HostEnv, dialog.HostStdio)
	a := NewAsker("")
	if !a.CanRetry() {
		t.Error("a person can type another password")
	}
	var got []byte
	var err error
	out := withStdio(t, "{\"cancel\":false,\"value\":\"typed\"}\n", func() { got, err = a.Secret("book.fd-sec", false) })
	if err != nil || string(got) != "typed" {
		t.Fatalf("got %q %v", got, err)
	}
	if !strings.HasPrefix(out, dialog.SecretPrefix) || !strings.Contains(out, `"retry":false`) || !strings.Contains(out, "book.fd-sec") {
		t.Errorf("marker = %q", out)
	}
	if strings.Contains(out, "typed") {
		t.Error("the value must not appear on stdout")
	}

	out = withStdio(t, "{\"cancel\":false,\"value\":\"again\"}\n", func() { got, err = a.Secret("book.fd-sec", true) })
	if err != nil || string(got) != "again" || !strings.Contains(out, `"retry":true`) {
		t.Errorf("retry: got %q %v, marker %q", got, err, out)
	}

	withStdio(t, "{\"cancel\":true}\n", func() { _, err = a.Secret("book.fd-sec", false) })
	var fe *Error
	if !errors.As(err, &fe) || fe.Class != Cancelled {
		t.Errorf("a cancel must be Cancelled, got %v", err)
	}
	withStdio(t, "", func() { _, err = a.Secret("book.fd-sec", false) })
	if !errors.As(err, &fe) || fe.Class != Cancelled {
		t.Errorf("a closed pipe must be Cancelled, got %v", err)
	}
}
