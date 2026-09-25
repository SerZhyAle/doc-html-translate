package app

import (
	"io"
	"os"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/windowsreg"
)

// captureStdout runs f and returns what it printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	f()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// A registration the user's own choice overrides must not be announced as done, and must say
// how to finish it.
func TestDefaultHandlerResultIsHonest(t *testing.T) {
	i18n.SetLanguage("en")
	t.Run("blocked", func(t *testing.T) {
		out := captureStdout(t, func() {
			printDefaultHandlerResult(windowsreg.Registration{Default: []string{".pdf"}, Blocked: []string{".epub"}})
		})
		for _, bad := range []string{"DONE", "will now open"} {
			if strings.Contains(out, bad) {
				t.Errorf("output claims success (%q):\n%s", bad, out)
			}
		}
		for _, want := range []string{"INCOMPLETE", "chose earlier", "* .epub", "Default apps"} {
			if !strings.Contains(out, want) {
				t.Errorf("output lacks %q:\n%s", want, out)
			}
		}
	})
	t.Run("failed", func(t *testing.T) {
		out := captureStdout(t, func() {
			printDefaultHandlerResult(windowsreg.Registration{Default: []string{".pdf"}, Failed: []string{".epub"}})
		})
		if !strings.Contains(out, "Could not register:") || !strings.Contains(out, "* .epub") {
			t.Errorf("the failed extension is not named:\n%s", out)
		}
	})
	t.Run("complete", func(t *testing.T) {
		out := captureStdout(t, func() {
			printDefaultHandlerResult(windowsreg.Registration{Default: []string{".epub", ".pdf"}})
		})
		if !strings.Contains(out, "DONE") || !strings.Contains(out, "will now open") {
			t.Errorf("a complete registration is not reported as done:\n%s", out)
		}
	})
}
