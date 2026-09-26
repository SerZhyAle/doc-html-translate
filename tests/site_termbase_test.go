package tests

// DOC-EXTERNAL-QUALITY rule 4, the termbase. configs/termbase.json names the product's key terms per
// locale: the stem the site uses and the variants it must not. Without it each fan-out of the
// landing picks its own word for "extension" or "overlay", and a reader who meets two words for one
// thing assumes two things.

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type termForm struct {
	Use   string   `json:"use"`
	Avoid []string `json:"avoid"`
}

type termEntry struct {
	ID       string              `json:"id"`
	Concept  string              `json:"concept"`
	Glossary bool                `json:"glossary"`
	Forms    map[string]termForm `json:"forms"`
}

type termbase struct {
	Shape   int         `json:"shape"`
	Locales []string    `json:"locales"`
	Terms   []termEntry `json:"terms"`
}

func loadTermbase(t *testing.T) termbase {
	t.Helper()
	var tb termbase
	if err := json.Unmarshal([]byte(readRepoFile(t, "configs", "termbase.json")), &tb); err != nil {
		t.Fatalf("configs/termbase.json: %v", err)
	}
	return tb
}

// docsPageFor is the docs trio member written in each author language.
var docsPageFor = map[string]string{"en": "docs.html", "ru": "docs.ru.html", "uk": "docs.uk.html"}

// localeTexts is the reader-facing text of the site per termbase locale: the <main> of each page
// written in it. The in-page trio pages mark their Russian and Ukrainian blocks data-l="ru"/"ua";
// their unmarked text is English.
func localeTexts(t *testing.T) map[string]string {
	t.Helper()
	parts := map[string][]string{}
	mainOf := func(p string) *html.Node {
		doc := parseSitePage(t, p)
		m := findElement(doc, byAtom(atom.Main))
		if m == nil {
			t.Fatalf("%s has no <main>", p)
		}
		return m
	}
	for _, p := range inPageTrioPages {
		m := mainOf(p)
		parts["en"] = append(parts["en"], visibleText(m, "en", true))
		parts["ru"] = append(parts["ru"], visibleText(m, "ru", false))
		parts["uk"] = append(parts["uk"], visibleText(m, "ua", false))
	}
	for lang, p := range docsPageFor {
		parts[lang] = append(parts[lang], visibleText(mainOf(p), "", true))
	}
	for _, c := range localeLandings {
		parts[c] = append(parts[c], visibleText(mainOf(c+"/index.html"), "", true))
	}
	out := map[string]string{}
	for k, v := range parts {
		out[k] = strings.Join(v, "\n")
	}
	return out
}

// containsTerm matches term case-insensitively. Latin and Cyrillic terms must start a word, so
// "addon" does not fire inside "Add-ons" and a stem still matches its inflections; other scripts
// attach particles to words or have no spaces at all, so there a substring is the only fair test.
func containsTerm(text, term string) bool {
	text, term = strings.ToLower(text), strings.ToLower(term)
	first := []rune(term)[0]
	if !unicode.In(first, unicode.Latin, unicode.Cyrillic) {
		return strings.Contains(text, term)
	}
	for i := 0; ; {
		j := strings.Index(text[i:], term)
		if j < 0 {
			return false
		}
		at := i + j
		prev := []rune(text[:at])
		if len(prev) == 0 || !(unicode.IsLetter(prev[len(prev)-1]) || unicode.IsDigit(prev[len(prev)-1]) || unicode.IsMark(prev[len(prev)-1])) {
			return true
		}
		i = at + len(term)
	}
}

func TestTermbaseShape(t *testing.T) {
	tb := loadTermbase(t)
	want := append([]string{"en", "ru", "uk"}, localeLandings...)
	got := append([]string{}, tb.Locales...)
	sort.Strings(want)
	sort.Strings(got)
	if tb.Shape != 1 || strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("termbase shape %d, locales %v; want shape 1 and the site's locales %v", tb.Shape, got, want)
	}
	for _, id := range []string{"convert", "translate", "ocr-overlay", "browser-extension", "desktop-app", "microsoft-store"} {
		found := false
		for _, e := range tb.Terms {
			found = found || e.ID == id
		}
		if !found {
			t.Errorf("termbase lacks the key term %q", id)
		}
	}
	for _, e := range tb.Terms {
		if e.Concept == "" {
			t.Errorf("term %s: no concept", e.ID)
		}
		if _, all := e.Forms["*"]; all {
			if len(e.Forms) != 1 {
				t.Errorf("term %s: a '*' proper name takes no per-locale forms", e.ID)
			}
			continue
		}
		for _, l := range tb.Locales {
			if f, ok := e.Forms[l]; !ok || f.Use == "" {
				t.Errorf("term %s: no form for locale %s", e.ID, l)
			}
		}
	}
}

func TestSiteFollowsTermbase(t *testing.T) {
	tb := loadTermbase(t)
	texts := localeTexts(t)
	all := strings.Join(func() []string {
		var s []string
		for _, v := range texts {
			s = append(s, v)
		}
		return s
	}(), "\n")
	for _, e := range tb.Terms {
		for loc, f := range e.Forms {
			text := texts[loc]
			if loc == "*" {
				text = all
			}
			if text == "" {
				t.Errorf("term %s: no site text for locale %s", e.ID, loc)
				continue
			}
			if !containsTerm(text, f.Use) {
				t.Errorf("term %s [%s]: the site never writes %q - update the termbase or the pages", e.ID, loc, f.Use)
			}
			for _, bad := range f.Avoid {
				if containsTerm(text, bad) {
					t.Errorf("term %s [%s]: the site writes %q; the termbase says %q", e.ID, loc, bad, f.Use)
				}
			}
		}
	}
}

func TestContainsTerm(t *testing.T) {
	cases := []struct {
		text, term string
		want       bool
	}{
		{"Edge Add-ons", "addon", false},
		{"an addon here", "addon", true},
		{"Plugins are", "plugin", true},
		{"Расширения для браузера", "расширени", true},
		{"перерасширение", "расширени", false},
		{"додатково", "додаток", false},
		{"浏览器插件", "插件", true},
		{"يحوّل الكتب", "حوّل", true},
	}
	for _, c := range cases {
		if got := containsTerm(c.text, c.term); got != c.want {
			t.Errorf("containsTerm(%q, %q) = %v, want %v", c.text, c.term, got, c.want)
		}
	}
}
