package tests

// The pitch catalog (SITE-REPRESENTATION rules 1 and 3, ticket E0130): docs/pitch.json is the one
// copy source for the product story, and the landing surfaces take their wording from it. The
// catalog's title and hero tagline land on index.html and on the ru and uk landings; the short
// pitch is the lead of the store summaries and the README trio, which are read by hand against it
// (the hub-card string is applied on the hub, never here). What is compared here: the landing head
// (<title>, og:title, twitter:title), the title the page's own strings rewrite it to, and the hero
// tagline - each against the catalog string of its language.

import (
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

type pitchStrings struct {
	Title              string `json:"title"`
	ShortPitch         string `json:"short_pitch"`
	HubCardDescription string `json:"hub_card_description"`
	HeroTagline        string `json:"hero_tagline"`
}

func loadPitch(t *testing.T) map[string]pitchStrings {
	t.Helper()
	var f struct {
		Schema  int                     `json:"schema"`
		Strings map[string]pitchStrings `json:"strings"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, "docs", "pitch.json")), &f); err != nil {
		t.Fatalf("docs/pitch.json: %v", err)
	}
	if f.Schema != 1 {
		t.Fatalf("docs/pitch.json: schema %d, want 1", f.Schema)
	}
	if len(f.Strings) != 3 {
		t.Fatalf("docs/pitch.json: %d languages, want the three author languages", len(f.Strings))
	}
	for _, l := range []string{"en", "ru", "uk"} {
		s, ok := f.Strings[l]
		if !ok {
			t.Fatalf("docs/pitch.json: no strings for %s", l)
		}
		for field, v := range map[string]string{
			"title": s.Title, "short_pitch": s.ShortPitch,
			"hub_card_description": s.HubCardDescription, "hero_tagline": s.HeroTagline,
		} {
			if strings.TrimSpace(v) == "" {
				t.Fatalf("docs/pitch.json %s: %s is empty", l, field)
			}
		}
		if s.HeroTagline == s.Title {
			t.Fatalf("docs/pitch.json %s: hero_tagline equals the title", l)
		}
	}
	return f.Strings
}

// taglineSentence is the part of the hero tagline the landing shows under the h1: the catalog's
// second sentence, closing period and all - the first one is the h1 itself.
func taglineSentence(t *testing.T, hero string) string {
	t.Helper()
	parts := strings.Split(hero, ". ")
	if len(parts) != 2 {
		t.Fatalf("hero tagline is not two sentences: %q", hero)
	}
	return parts[1]
}

func findClass(n *html.Node, class string) *html.Node {
	return findElement(n, func(n *html.Node) bool {
		v, ok := attrOf(n, "class")
		return n.Type == html.ElementNode && ok && strings.Contains(" "+v+" ", " "+class+" ")
	})
}

func pitchTitleAssertion(t *testing.T, page, want string) {
	t.Helper()
	doc := parseSitePage(t, page)
	for _, m := range []struct{ what, key, name string }{
		{"<title>", "", ""},
		{"og:title", "property", "og:title"},
		{"twitter:title", "name", "twitter:title"},
	} {
		var got string
		if m.what == "<title>" {
			got = headTitle(doc)
		} else {
			got, _ = metaContent(doc, m.key, m.name)
		}
		if got != want {
			t.Errorf("%s: %s is %q, the pitch catalog says %q", page, m.what, got, want)
		}
	}
}

func TestSitePitchCatalogOnTheLanding(t *testing.T) {
	pitch := loadPitch(t)

	// The root: every surface it carries in-page, in that language.
	pitchTitleAssertion(t, "index.html", pitch["en"].Title)
	doc := parseSitePage(t, "index.html")
	tag := findClass(doc, "tagline")
	if tag == nil {
		t.Fatal("index.html: no .tagline element")
	}
	if want := taglineSentence(t, pitch["en"].HeroTagline); visibleText(tag, "en", true) != want {
		t.Errorf("index.html: the en tagline is %q, the pitch catalog says %q", visibleText(tag, "en", true), want)
	}
	if want := taglineSentence(t, pitch["ru"].HeroTagline); visibleText(tag, "ru", true) != want {
		t.Errorf("index.html: the ru tagline is %q, the pitch catalog says %q", visibleText(tag, "ru", true), want)
	}
	if want := taglineSentence(t, pitch["uk"].HeroTagline); visibleText(tag, "ua", true) != want {
		t.Errorf("index.html: the ua tagline is %q, the pitch catalog says %q", visibleText(tag, "ua", true), want)
	}

	// The physical author-language landings.
	pitchTitleAssertion(t, "ru/index.html", pitch["ru"].Title)
	pitchTitleAssertion(t, "uk/index.html", pitch["uk"].Title)
	for _, c := range []string{"ru", "uk"} {
		doc := parseSitePage(t, c+"/index.html")
		tag := findClass(doc, "tagline")
		if tag == nil {
			t.Fatalf("%s/index.html: no .tagline element", c)
		}
		if want := taglineSentence(t, pitch[c].HeroTagline); visibleText(tag, "", true) != want {
			t.Errorf("%s/index.html: the tagline is %q, the pitch catalog says %q", c, visibleText(tag, "", true), want)
		}
	}
}

// The page's own strings rewrite document.title after the visitor picks a language; they carry the
// catalog title too, so the rewritten title is the same wording the crawler sees.
func TestSitePitchCatalogInThePageStrings(t *testing.T) {
	pitch := loadPitch(t)
	raw := readRepoFile(t, "index.html")
	for _, l := range []string{"en", "ru", "ua"} {
		src := map[string]string{"en": "en", "ru": "ru", "ua": "uk"}[l]
		want := pitch[src].Title
		if !strings.Contains(raw, "title:'"+want+"'") {
			t.Errorf("index.html: SITE.strings %s title is not the pitch catalog title %q", l, want)
		}
	}
}
