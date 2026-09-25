package dialog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withHostPipes points os.Stdin at a file holding the GUI's answer and returns what the code
// under test wrote to os.Stdout.
func withHostPipes(t *testing.T, answer string, run func()) string {
	t.Helper()
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	if err := os.WriteFile(in, []byte(answer), 0o600); err != nil {
		t.Fatal(err)
	}
	fin, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer fin.Close()
	fout, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	origIn, origOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = fin, fout
	run()
	os.Stdin, os.Stdout = origIn, origOut
	_ = fout.Close()
	b, _ := os.ReadFile(out)
	return string(b)
}

// The GUI reads the question off one marker line and answers on stdin. Only an explicit yes
// proceeds; a no, an empty line or a closed pipe all decline.
func TestHostedConfirmSpeaksTheMarkerProtocol(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	for answer, want := range map[string]bool{"yes\n": true, "yes\r\n": true, "no\n": false, "\n": false, "": false, "y\n": false} {
		var got bool
		out := withHostPipes(t, answer, func() { got = Confirm("Cost", "Line one\nline two") })
		if got != want {
			t.Errorf("answer %q: Confirm = %v, want %v", answer, got, want)
		}
		if !strings.HasPrefix(out, AskPrefix) || strings.Count(out, "\n") != 1 {
			t.Fatalf("stdout = %q, want exactly one ask marker line", out)
		}
		var q hostLine
		if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(out), AskPrefix)), &q); err != nil {
			t.Fatal(err)
		}
		if q.Title != "Cost" || q.Message != "Line one\nline two" {
			t.Errorf("question = %+v", q)
		}
	}
}

func TestHostedWarningIsANoteLine(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	out := withHostPipes(t, "", func() { ShowWarning("PDF", "pdftotext was blocked") })
	if !strings.HasPrefix(out, NotePrefix) || !strings.Contains(out, "pdftotext was blocked") {
		t.Errorf("stdout = %q, want one note marker line", out)
	}
}
