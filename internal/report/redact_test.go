package report

import (
	"path/filepath"
	"strings"
	"testing"
)

// setHome points both location variables at a temporary profile so the path rules are the
// same on every machine that runs the test.
func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	return home
}

func TestRedactRemovesGoogleAPIKeyShapes(t *testing.T) {
	setHome(t)

	secrets := map[string]string{
		"bare":     "using AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r for translation",
		"key=":     "google_key=AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r",
		"token:":   "token: abcdefghijklmnopqrstuvwxyz012345",
		"password": "Password hunter2hunter2hunter2hunter2",
		"json":     `{"googleKey":"AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r"}`,
	}
	for name, in := range secrets {
		got := Redact(in)
		if !strings.Contains(got, "<redacted>") {
			t.Errorf("%s: nothing was redacted in %q -> %q", name, in, got)
		}
		for _, leak := range []string{"AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r", "abcdefghijklmnopqrstuvwxyz012345", "hunter2hunter2hunter2hunter2"} {
			if strings.Contains(got, leak) {
				t.Errorf("%s: secret survived redaction: %q", name, got)
			}
		}
	}
}

func TestRedactShortensUserProfilePaths(t *testing.T) {
	home := setHome(t)

	in := filepath.Join(home, "Documents", "Books", "War and Peace.epub")
	got := Redact(in)
	if !strings.HasPrefix(got, "%USERPROFILE%") {
		t.Errorf("Redact(%q) = %q, want it to start with %%USERPROFILE%%", in, got)
	}
	if strings.Contains(got, home) {
		t.Errorf("the home directory survived: %q", got)
	}
	if !strings.Contains(got, "War and Peace.epub") {
		t.Errorf("the document's own name was lost: %q", got)
	}
}

func TestRedactKeepsDocumentFileNames(t *testing.T) {
	home := setHome(t)

	in := filepath.Join(home, "Загрузки", "Война и мир.fb2")
	got := Redact(in)
	if !strings.Contains(got, "Война и мир.fb2") {
		t.Errorf("a Cyrillic document name did not survive: %q", got)
	}
}

func TestRedactPrefersTheMoreSpecificLocation(t *testing.T) {
	home := setHome(t)

	in := filepath.Join(home, "AppData", "Local", "doc-html-translate", "logs", "run-20260729-130509.log")
	got := Redact(in)
	if !strings.HasPrefix(got, "%LOCALAPPDATA%") {
		t.Errorf("Redact(%q) = %q, want it to start with %%LOCALAPPDATA%%", in, got)
	}
}
