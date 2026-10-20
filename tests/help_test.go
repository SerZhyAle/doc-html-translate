package tests

// The in-app help gate inside the Go suite (CONTEXT-HELP rule 19, ticket 118). The judging is internal/helpcheck, the
// same code scripts/help-check.ps1 runs, so a structural break fails `go test` as well as the gate: a surface with no
// row, a row with no surface, a page missing in a shipped locale, an unresolved link, a request to another origin, a
// `none` or `inherit` row without a reason, a baseline that grew. Freshness against the inputs is an advisory.

import (
	"strings"
	"testing"

	"doc-html-translate/internal/helpcheck"
	"doc-html-translate/internal/i18n"
)

func TestHelpGate(t *testing.T) {
	r, err := helpcheck.Run("..")
	if err != nil {
		t.Fatalf("the help set cannot be read: %v", err)
	}
	for _, f := range r.Failures {
		t.Error(f)
	}
	for _, a := range r.Advisories {
		t.Logf("advisory: %s", a)
	}
	if r.Pages == 0 || r.Links == 0 {
		t.Errorf("the gate counted %d pages and %d links - the scan is wrong, not the set", r.Pages, r.Links)
	}
}

// Every GUI language names the help button and the About link in its own words: the vocabulary's app.help name,
// the key appended (CONTEXT-HELP rule 2).
func TestHelpButtonNameInEveryLanguage(t *testing.T) {
	js := readRepoFile(t, "cmd", "doc-html-ui", "i18n.js")
	want := map[string]string{
		"en": "Help", "ru": "Справка", "uk": "Довідка", "de": "Hilfe", "it": "Aiuto", "es": "Ayuda", "fr": "Aide",
		"pt": "Ajuda", "ar": "مساعدة", "hi": "सहायता", "bn": "সহায়তা", "ur": "مدد", "zh": "帮助",
	}
	for _, code := range i18n.Codes {
		name := want[code]
		if name == "" {
			t.Fatalf("no expected help name for %s", code)
		}
		if !strings.Contains(js, `titleHelp: "`+name+` (F1)",`) || !strings.Contains(js, `aboutHelp: "`+name+`",`) {
			t.Errorf("i18n.js: the %s dictionary does not carry titleHelp/aboutHelp as %q (the app.help name of the vocabulary)", code, name)
		}
	}
}
