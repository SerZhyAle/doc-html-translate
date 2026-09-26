package tests

// DOC-EXTERNAL-QUALITY rule 3, translation freshness. Every localized page records which English it
// was made from: a comment in its <head>,
//
//	<!-- l10n-source: index.html sha256:0123456789abcdef -->
//
// naming the English source page and a hash of that page's English text (its <title>, its meta
// description and the English of its <main>). The ten landings are fanned out from index.html and
// docs.ru.html / docs.uk.html are written from docs.html. The in-page trio pages carry their ru and
// ua blocks beside the English, so their stamp names the page itself.
//
// A missing or malformed stamp fails: it is a page nobody can judge. A stale stamp - the English
// moved since the translation was made - is an advisory, not a failure: the landings are
// re-translated at the release boundary, not on every English edit. The test logs it as a line
// starting "advisory:", which scripts/test.ps1 turns into PASS WITH ADVISORIES (exit 3), and so
// scripts/check.ps1 does; a release wants a bare PASS.
//
// After re-translating (or checking that the translation still says what the English says), stamp
// the page again:
//
//	go test ./tests -run TestSiteTranslationFreshness -update-l10n

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var updateL10n = flag.Bool("update-l10n", false, "re-stamp every localized site page with the hash of its current English source")

var l10nStampRe = regexp.MustCompile(`<!-- l10n-source: (\S+) sha256:([0-9a-f]{16}) -->`)

// l10nSources maps each localized page to the English page it is made from. docs.html is English
// only and is a source, never a target.
func l10nSources() map[string]string {
	m := map[string]string{"docs.ru.html": "docs.html", "docs.uk.html": "docs.html"}
	for _, p := range inPageTrioPages {
		m[p] = p
	}
	for _, c := range localeLandings {
		m[c+"/index.html"] = "index.html"
	}
	return m
}

// englishFingerprint hashes the English a translation of the page source is made from.
func englishFingerprint(source string) (string, error) {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return "", err
	}
	desc, _ := metaContent(doc, "name", "description")
	body := ""
	if m := findElement(doc, byAtom(atom.Main)); m != nil {
		body = visibleText(m, "en", true)
	}
	sum := sha256.Sum256([]byte(headTitle(doc) + "\n" + desc + "\n" + body))
	return hex.EncodeToString(sum[:])[:16], nil
}

func TestSiteTranslationFreshness(t *testing.T) {
	sources := l10nSources()
	for _, p := range sitePages() {
		if _, ok := sources[p]; !ok && p != "docs.html" {
			t.Errorf("%s is served but has no translation source on record; add it to l10nSources", p)
		}
	}
	for page, src := range sources {
		want, err := englishFingerprint(readRepoFile(t, strings.Split(src, "/")...))
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		raw := readRepoFile(t, strings.Split(page, "/")...)
		stamp := fmt.Sprintf("<!-- l10n-source: %s sha256:%s -->", src, want)
		m := l10nStampRe.FindAllStringSubmatch(raw, -1)
		if *updateL10n {
			if len(m) > 1 {
				t.Fatalf("%s carries %d l10n-source stamps; leave one before re-stamping", page, len(m))
			}
			updated := raw
			if len(m) == 1 {
				updated = strings.Replace(raw, m[0][0], stamp, 1)
			} else {
				updated = strings.Replace(raw, `<meta charset="utf-8">`, `<meta charset="utf-8">`+"\n  "+stamp, 1)
			}
			if updated == raw && len(m) == 0 {
				t.Fatalf("%s: no <meta charset=\"utf-8\"> to anchor the stamp to", page)
			}
			if err := os.WriteFile(filepath.Join("..", filepath.FromSlash(page)), []byte(updated), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		switch {
		case len(m) == 0:
			t.Errorf("%s carries no l10n-source stamp; translate from %s and stamp it (-update-l10n)", page, src)
		case len(m) > 1:
			t.Errorf("%s carries %d l10n-source stamps, want one", page, len(m))
		case m[0][1] != src:
			t.Errorf("%s names %s as its source, want %s", page, m[0][1], src)
		case m[0][2] != want:
			t.Logf("advisory: %s was translated from %s at sha256:%s; its English is now sha256:%s - re-translate, then re-stamp with -update-l10n", page, src, m[0][2], want)
		}
	}
}

// The fingerprint must see an English edit and must not see a Russian one, or a stale landing
// would pass silently and a fixed Russian typo would flag ten landings.
func TestEnglishFingerprintScope(t *testing.T) {
	fp := func(source string) string {
		sum, err := englishFingerprint(source)
		if err != nil {
			t.Fatal(err)
		}
		return sum
	}
	page := func(en, ru string) string {
		return `<!doctype html><html><head><meta charset="utf-8"><title>T</title><meta name="description" content="D"></head><body><main><p><span data-l="ru">` + ru + `</span><span data-l="en">` + en + `</span></p></main></body></html>`
	}
	base := fp(page("Hello", "Привет"))
	if fp(page("Hello", "Здравствуйте")) != base {
		t.Error("a Russian edit changed the English fingerprint")
	}
	if fp(page("Hello there", "Привет")) == base {
		t.Error("an English edit did not change the English fingerprint")
	}
}
