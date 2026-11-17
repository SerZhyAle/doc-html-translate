package tests

// DOC-EXTERNAL-QUALITY rule 5, the length half of the SEO block (scripts/doc-registry.ps1 checks that
// the fields exist). A search result cuts a title after about 60 characters and a snippet after
// about 160, so a longer one reaches the reader truncated. Lengths are counted in characters
// (runes) after entity decoding - what the reader sees, not the bytes of the source.
//
// The in-page trio pages also switch title and description by script (window.SITE strings, applied
// by assets/site.js to <title>, og:title, twitter:title and the three descriptions), so a ?l=ru
// address renders the ru strings; those are held to the same limits.

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

const (
	maxTitleRunes       = 60
	maxDescriptionRunes = 160
)

// The trio pages key the strings by language (ru/en/ua, bare keys); a locale landing has one set
// under page, written as JSON.
var siteStringRe = regexp.MustCompile(`\b(ru|en|ua|page):\{\s*"?title"?:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")(?:,\s*"?desc"?:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"))?`)

func unquoteJS(s string) string {
	if len(s) < 2 {
		return s
	}
	s = s[1 : len(s)-1]
	return strings.NewReplacer(`\'`, `'`, `\"`, `"`, `\\`, `\`).Replace(s)
}

func TestSiteSEOLengths(t *testing.T) {
	for _, p := range sitePages() {
		doc := parseSitePage(t, p)
		title := headTitle(doc)
		if n := utf8.RuneCountInString(title); n > maxTitleRunes {
			t.Errorf("%s: <title> is %d characters, the limit is %d: %q", p, n, maxTitleRunes, title)
		}
		desc, _ := metaContent(doc, "name", "description")
		if n := utf8.RuneCountInString(desc); n > maxDescriptionRunes {
			t.Errorf("%s: meta description is %d characters, the limit is %d", p, n, maxDescriptionRunes)
		}
		// og:title is the share-card title and has always equalled <title>; twitter:title may be a
		// shorter card variant but not a longer one.
		if og, ok := metaContent(doc, "property", "og:title"); ok && og != title {
			t.Errorf("%s: og:title %q differs from <title> %q", p, og, title)
		}
		if tw, ok := metaContent(doc, "name", "twitter:title"); ok && utf8.RuneCountInString(tw) > maxTitleRunes {
			t.Errorf("%s: twitter:title is over %d characters: %q", p, maxTitleRunes, tw)
		}
		for _, key := range []string{"og:description", "twitter:description"} {
			attr := "name"
			if strings.HasPrefix(key, "og:") {
				attr = "property"
			}
			if v, ok := metaContent(doc, attr, key); ok && utf8.RuneCountInString(v) > maxDescriptionRunes {
				t.Errorf("%s: %s is over %d characters", p, key, maxDescriptionRunes)
			}
		}

		for _, m := range siteStringRe.FindAllStringSubmatch(readRepoFile(t, strings.Split(p, "/")...), -1) {
			if s := unquoteJS(m[2]); utf8.RuneCountInString(s) > maxTitleRunes {
				t.Errorf("%s: SITE.strings.%s.title is %d characters, the limit is %d: %q", p, m[1], utf8.RuneCountInString(s), maxTitleRunes, s)
			}
			if m[3] != "" {
				if s := unquoteJS(m[3]); utf8.RuneCountInString(s) > maxDescriptionRunes {
					t.Errorf("%s: SITE.strings.%s.desc is %d characters, the limit is %d", p, m[1], utf8.RuneCountInString(s), maxDescriptionRunes)
				}
			}
		}
	}
}

// The script strings are read by a regex over the page; a page that switches language by script
// must yield one title per language it offers, or the length check above reads nothing.
func TestSiteSEOStringsAreRead(t *testing.T) {
	for _, p := range inPageTrioPages {
		got := map[string]bool{}
		for _, m := range siteStringRe.FindAllStringSubmatch(readRepoFile(t, p), -1) {
			got[m[1]] = true
		}
		for _, l := range []string{"en", "ru", "ua"} {
			if !got[l] {
				t.Errorf("%s: no SITE.strings.%s.title was read", p, l)
			}
		}
	}
	for _, c := range localeLandings {
		p := c + "/index.html"
		if m := siteStringRe.FindStringSubmatch(readRepoFile(t, c, "index.html")); m == nil || m[1] != "page" || m[3] == "" {
			t.Errorf("%s: SITE.strings.page title and desc were not read", p)
		}
	}
}

// DOC-EXTERNAL-QUALITY 0.14 item U: every multilingual page whose ?lang= views canonicalize
// together declares all offered languages in JSON-LD "inLanguage"; single-language pages
// declare their single language.
func TestSiteJSONLDInLanguage(t *testing.T) {
	ldRe := regexp.MustCompile(`(?s)<script\s+type="application/ld\+json">\s*(\{.*?\})\s*</script>`)
	expected := map[string][]string{
		"index.html":                {"ar", "bn", "de", "en", "es", "fr", "hi", "it", "pt", "ru", "uk", "ur", "zh"},
		"extension.html":            {"en", "ru", "uk"},
		"extension-privacy.html":    {"en", "ru", "uk"},
		"install-trust.html":        {"en", "ru", "uk"},
		"privacy.html":              {"en", "ru", "uk"},
		"release-notes.html":        {"en", "ru", "uk"},
		"support.html":              {"en", "ru", "uk"},
		"docs.html":                 {"en"},
		"docs.ru.html":              {"ru"},
		"docs.uk.html":              {"uk"},
		"capability-matrix.html":    {"en"},
		"capability-matrix.ru.html": {"ru"},
		"capability-matrix.uk.html": {"uk"},
	}
	for _, c := range localeLandings {
		expected[c+"/index.html"] = []string{c}
	}
	for page, want := range expected {
		raw := readRepoFile(t, strings.Split(page, "/")...)
		m := ldRe.FindStringSubmatch(raw)
		if m == nil {
			t.Errorf("%s: missing <script type=\"application/ld+json\"> block", page)
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(m[1]), &doc); err != nil {
			t.Errorf("%s: invalid JSON-LD: %v", page, err)
			continue
		}
		v, ok := doc["inLanguage"]
		if !ok {
			if graph, gok := doc["@graph"].([]any); gok {
				for _, node := range graph {
					if obj, ok2 := node.(map[string]any); ok2 {
						if iv, iok := obj["inLanguage"]; iok {
							v = iv
							ok = true
							break
						}
					}
				}
			}
		}
		if !ok {
			t.Errorf("%s: JSON-LD lacks \"inLanguage\" (DOC-EXTERNAL-QUALITY 0.14 item U)", page)
			continue
		}
		var got []string
		switch x := v.(type) {
		case string:
			got = []string{x}
		case []any:
			for _, item := range x {
				if s, ok := item.(string); ok {
					got = append(got, s)
				}
			}
		default:
			t.Errorf("%s: unexpected JSON-LD inLanguage type %T", page, v)
			continue
		}
		sort.Strings(got)
		wantSorted := append([]string(nil), want...)
		sort.Strings(wantSorted)
		if !reflect.DeepEqual(got, wantSorted) {
			t.Errorf("%s: JSON-LD inLanguage = %v, want %v", page, got, wantSorted)
		}
	}
}
