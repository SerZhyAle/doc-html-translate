package app

import (
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
)

// TestSplashResourcesExistForEveryLanguage keeps the embedded welcome screen in step with the
// shipped language set: a missing file silently falls back to English, which reads as "this
// language is done" while it is not.
func TestSplashResourcesExistForEveryLanguage(t *testing.T) {
	for _, code := range i18n.Codes {
		raw, err := splashFS.ReadFile("splash/" + code + ".txt")
		if err != nil {
			t.Errorf("%s: no splash resource: %v", code, err)
			continue
		}
		text := string(raw)
		if !strings.Contains(text, "DOC-HTML-TRANSLATE") {
			t.Errorf("%s: splash does not carry the product name", code)
		}
		if !strings.Contains(text, "doc-html-translate.exe") {
			t.Errorf("%s: splash does not show the usage examples", code)
		}
		if !strings.Contains(text, splashRule) {
			t.Errorf("%s: splash has no %s placeholder", code, splashRule)
		}
	}
}

// TestSplashKeepsItsOpeningLines pins the first three printed lines, which are what a user sees
// when the window pops up and the only part of the screen the layout depends on.
func TestSplashKeepsItsOpeningLines(t *testing.T) {
	raw, err := splashFS.ReadFile("splash/en.txt")
	if err != nil {
		t.Fatalf("read en splash: %v", err)
	}
	rule := strings.Repeat("=", 62)
	lines := strings.Split(strings.ReplaceAll(string(raw), splashRule, rule), "\n")
	if len(lines) < 3 {
		t.Fatalf("splash has %d lines", len(lines))
	}
	want := []string{
		rule,
		"  DOC-HTML-TRANSLATE",
		"  Document converter to HTML, with translation for the rest of us",
	}
	for i, w := range want {
		if got := strings.TrimRight(lines[i], "\r"); got != w {
			t.Errorf("line %d: got %q, want %q", i+1, got, w)
		}
	}
}
