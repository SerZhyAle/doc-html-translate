package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/config"
)

func TestSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in short mode")
	}
}

// TestConvertedChromeLanguage converts one fixture per interface language and checks the two
// rules the chrome has to follow: it speaks the interface language and declares that language on
// itself, while <html lang> keeps carrying the document's language. The second half is what makes
// Chrome offer "Translate page" on the result, so it is worth a test rather than a comment.
func TestConvertedChromeLanguage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping conversion in short mode")
	}

	// Plain ASCII English prose: the source language must be unambiguous, or the assertion
	// about <html lang> tests the language detector instead of the chrome.
	const fixture = "The Adventures of a Test Fixture\n\n" +
		"This is an ordinary English paragraph, long enough that the converter treats it as " +
		"body text and not as a heading. It exists only so the generated page has something " +
		"to wrap navigation around.\n"

	// "Theme" is the reader control present in both the single-page bar and the per-chapter
	// one, so it is the string to key on; "Contents" only exists on the multi-page path.
	cases := []struct {
		lang  string
		theme string // the language's translation of "Theme"
		rtl   bool
	}{
		{lang: "en", theme: "Theme"},
		{lang: "zh", theme: "主题"},
		{lang: "ar", theme: "المظهر", rtl: true},
	}

	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			in := filepath.Join(t.TempDir(), "chrome-language.txt")
			if err := os.WriteFile(in, []byte(fixture), 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			out := t.TempDir()

			cfg := config.Config{
				InputFile:    in,
				OutputFolder: out,
				NoTranslate:  true,
				NoOpen:       true,
				SinglePage:   true,
				SourceLang:   "en",
				TargetLang:   "ru",
				UILang:       c.lang,
			}
			// app.New is where the interface language is resolved, so the conversion goes
			// through it rather than calling the pipeline directly.
			exitCode, err := app.New(cfg).Run()
			if err != nil {
				t.Fatalf("convert: exit=%d err=%v", exitCode, err)
			}

			indexPath := findIndexHTML(out)
			if indexPath == "" {
				t.Fatalf("no index.html produced under %s", out)
			}
			raw, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatalf("read %s: %v", indexPath, err)
			}
			page := string(raw)

			if !strings.Contains(page, `class="dht-navbar" lang="`+c.lang+`"`) {
				t.Errorf("chrome does not declare lang=%q", c.lang)
			}
			if !strings.Contains(page, c.theme) {
				t.Errorf("chrome is not in %s: %q not found", c.lang, c.theme)
			}
			if got := strings.Contains(page, `dir="rtl"`); got != c.rtl {
				t.Errorf("dir=rtl present = %v, want %v", got, c.rtl)
			}
			// The document is English prose; the interface language must not have leaked into
			// the element the browser reads to decide whether to offer a translation.
			if strings.Contains(page, `<html lang="`+c.lang+`"`) && c.lang != "en" {
				t.Errorf("<html lang> carries the interface language %q instead of the document's", c.lang)
			}
		})
	}
}
