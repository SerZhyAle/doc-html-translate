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

func TestProgressUsesHostedMarkerOnly(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	out := withHostPipes(t, "", func() {
		Progress("extracting", 0, 12)
		Progress("extracting", 12, 12)
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("progress lines = %q", out)
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, ProgressPrefix) {
			t.Fatalf("not a progress marker: %q", line)
		}
		var event struct {
			Stage string `json:"stage"`
			Done  int    `json:"done"`
			Total int    `json:"total"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, ProgressPrefix)), &event); err != nil {
			t.Fatal(err)
		}
		if event.Stage != "extracting" || event.Total != 12 {
			t.Fatalf("unexpected progress: %+v", event)
		}
	}
	t.Setenv(HostEnv, "")
	if out := withHostPipes(t, "", func() { Progress("saving", 0, 0) }); out != "" {
		t.Fatalf("console run received marker: %q", out)
	}
}

// A secret question carries its retry flag on the marker line; the value comes back as one JSON
// line on stdin and is never written to stdout.
func TestHostedSecretSpeaksTheMarkerProtocol(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	for _, retry := range []bool{false, true} {
		var got []byte
		var ok bool
		out := withHostPipes(t, "{\"cancel\":false,\"value\":\"pw\"}\n", func() { got, ok = AskSecret("FileDO", "Enter it", retry) })
		if !ok || string(got) != "pw" {
			t.Errorf("retry=%v: got %q ok=%v", retry, got, ok)
		}
		if !strings.HasPrefix(out, SecretPrefix) || strings.Count(out, "\n") != 1 {
			t.Fatalf("stdout = %q, want exactly one secret marker line", out)
		}
		var q secretLine
		if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(out), SecretPrefix)), &q); err != nil {
			t.Fatal(err)
		}
		if q.Title != "FileDO" || q.Message != "Enter it" || q.Retry != retry {
			t.Errorf("question = %+v", q)
		}
		if strings.Contains(out, "pw") {
			t.Errorf("the value must never be written out: %q", out)
		}
	}
}

func TestHostedSecretValueSurvivesAwkwardCharacters(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	for _, want := range []string{"", "a\"b\\c", "пароль-密码-✓", " lead and trail "} {
		line, _ := json.Marshal(secretAnswer{Value: want})
		var got []byte
		var ok bool
		withHostPipes(t, string(line)+"\r\n", func() { got, ok = AskSecret("t", "m", false) })
		if !ok || string(got) != want {
			t.Errorf("value %q came back as %q (ok=%v)", want, got, ok)
		}
	}
}

func TestHostedSecretUnansweredIsNotOK(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	for name, answer := range map[string]string{
		"closed pipe": "",
		"cancel":      "{\"cancel\":true,\"value\":\"x\"}\n",
		"bad json":    "not json\n",
		"blank line":  "\n",
	} {
		var got []byte
		var ok bool
		withHostPipes(t, answer, func() { got, ok = AskSecret("t", "m", false) })
		if ok || got != nil {
			t.Errorf("%s: got %q ok=%v, want not ok", name, got, ok)
		}
	}
}
