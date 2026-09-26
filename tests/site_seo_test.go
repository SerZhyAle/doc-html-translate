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
	"regexp"
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
