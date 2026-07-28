package i18n

import "testing"

func TestCodesAndNamesAgree(t *testing.T) {
	if len(Codes) != len(Names) {
		t.Fatalf("Codes has %d entries, Names has %d", len(Codes), len(Names))
	}
	if len(Codes) != 13 {
		t.Fatalf("the shipped language set is 13, got %d", len(Codes))
	}
	if Codes[0] != "en" {
		t.Fatalf("index 0 must be the key language, got %q", Codes[0])
	}
	seen := map[string]bool{}
	for i, c := range Codes {
		if seen[c] {
			t.Errorf("duplicate code %q", c)
		}
		seen[c] = true
		if Names[i] == "" {
			t.Errorf("%s has no endonym", c)
		}
	}
}

// TestEveryKeyHasTwelveNonEmptyTranslations is the guard the later phases inherit: a key with an
// empty column falls back to English at runtime and looks fine, so only a static check catches
// a language that was left behind.
func TestEveryKeyHasTwelveNonEmptyTranslations(t *testing.T) {
	for en, tr := range All() {
		if len(tr) != len(Codes)-1 {
			t.Errorf("%q has %d translations, want %d", en, len(tr), len(Codes)-1)
			continue
		}
		for i, v := range tr {
			if v == "" {
				t.Errorf("%q has no %s translation", en, Codes[i+1])
			}
		}
	}
}

func TestTranslateFallbacks(t *testing.T) {
	Add("phase-02 fixture", "ру", "укр", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh")
	t.Cleanup(func() { delete(translations, "phase-02 fixture") })

	if got := T("ru", "phase-02 fixture"); got != "ру" {
		t.Errorf("ru: got %q", got)
	}
	if got := T("en", "phase-02 fixture"); got != "phase-02 fixture" {
		t.Errorf("en must return the key itself, got %q", got)
	}
	if got := T("ru", "never registered"); got != "never registered" {
		t.Errorf("unknown key must return itself, got %q", got)
	}
	if got := T("xx", "phase-02 fixture"); got != "phase-02 fixture" {
		t.Errorf("unknown language must fall back to English, got %q", got)
	}

	translations["phase-02 empty"] = make([]string, len(Codes)-1)
	t.Cleanup(func() { delete(translations, "phase-02 empty") })
	if got := T("de", "phase-02 empty"); got != "phase-02 empty" {
		t.Errorf("empty translation must fall back to English, got %q", got)
	}
}

func TestTranslateFormats(t *testing.T) {
	Add("Chapters: %d", "Глав: %d", "Розділів: %d", "Kapitel: %d", "Capitoli: %d", "Capítulos: %d",
		"Chapitres: %d", "Capítulos: %d", "الفصول: %d", "अध्याय: %d", "অধ্যায়: %d", "ابواب: %d", "章节：%d")
	t.Cleanup(func() { delete(translations, "Chapters: %d") })

	if got := T("ru", "Chapters: %d", 7); got != "Глав: 7" {
		t.Errorf("got %q", got)
	}
}

func TestDirection(t *testing.T) {
	for _, c := range []string{"ar", "ur"} {
		if !IsRTL(c) {
			t.Errorf("%s must be right-to-left", c)
		}
		if Dir(c) != "rtl" {
			t.Errorf("%s: Dir = %q", c, Dir(c))
		}
	}
	for _, c := range []string{"en", "ru", "zh", "hi", "xx"} {
		if IsRTL(c) {
			t.Errorf("%s must not be right-to-left", c)
		}
		if Dir(c) != "ltr" {
			t.Errorf("%s: Dir = %q", c, Dir(c))
		}
	}
}

func TestFontFamily(t *testing.T) {
	cases := map[string]string{
		"hi": "Nirmala UI", "bn": "Nirmala UI", "zh": "Microsoft YaHei UI",
		"en": "", "ru": "", "ar": "", "ur": "",
	}
	for code, want := range cases {
		if got := FontFamily(code); got != want {
			t.Errorf("%s: got %q, want %q", code, got, want)
		}
	}
}

func TestResolvePrecedence(t *testing.T) {
	cases := []struct{ explicit, saved, system, want string }{
		{"de", "ru", "uk", "de"},
		{"", "ru", "uk", "ru"},
		{"", "", "uk", "uk"},
		{"", "", "", "en"},
		{"xx", "ru", "uk", "ru"},
		{"", "xx", "xx", "en"},
	}
	for _, c := range cases {
		if got := Resolve(c.explicit, c.saved, c.system); got != c.want {
			t.Errorf("Resolve(%q,%q,%q) = %q, want %q", c.explicit, c.saved, c.system, got, c.want)
		}
	}
}

func TestProcessLanguageRoundTrip(t *testing.T) {
	t.Cleanup(func() { SetLanguage("en") })

	SetLanguage("bn")
	if Language() != "bn" {
		t.Errorf("Language = %q", Language())
	}
	SetLanguage("xx")
	if Language() != "en" {
		t.Errorf("an unknown code must select English, got %q", Language())
	}

	Add("phase-02 process", "ру", "укр", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh")
	t.Cleanup(func() { delete(translations, "phase-02 process") })
	SetLanguage("uk")
	if got := S("phase-02 process"); got != "укр" {
		t.Errorf("S = %q", got)
	}
}

func TestAddRejectsWrongColumnCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Add must panic on a wrong translation count")
		}
	}()
	Add("phase-02 short", "only one")
}
