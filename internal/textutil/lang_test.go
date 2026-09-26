package textutil

import "testing"

// The cases match extension/test/reflow.test.mjs "normalizeLangTag", so the two editions read a
// declared language the same way.
func TestNormalizeLangTag(t *testing.T) {
	cases := map[string]string{
		"en-US":   "en-US",
		"RU":      "ru",
		"fr_FR":   "fr-FR",
		" ru ":    "ru",
		"":        "",
		"zh-Hans": "zh",
		"russian": "",
		"uk-UA-x": "uk-UA",
		"123":     "",
	}
	for in, want := range cases {
		if got := NormalizeLangTag(in); got != want {
			t.Errorf("NormalizeLangTag(%q) = %q, want %q", in, got, want)
		}
	}
}
