package tests

// SITE-REPRESENTATION rules 1 to 4 (docs/contracts/SITE-REPRESENTATION.md). The positioning source is
// docs/POSITIONING.md (the ordered pillars) and the facts typed once are in docs/positioning.json (the
// public editions with one display name each, the channels, the interface-language count). What is
// compared here: the facts file against the code it derives from, the editions list of the README trio
// and the documentation trio against the declared names, every page that states a language count
// against the file, the pillar order of the use-case cards on the landing and its ten language pages,
// and a list of names that were retired.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
)

type positioningEdition struct {
	ID          string            `json:"id"`
	Names       map[string]string `json:"names"`
	DerivesFrom []string          `json:"derivesFrom"`
	Channels    []string          `json:"channels"`
}

type positioningFacts struct {
	Editions []positioningEdition `json:"editions"`
	Channels []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channels"`
	InterfaceLanguages struct {
		Count       int      `json:"count"`
		Codes       []string `json:"codes"`
		DerivesFrom []string `json:"derivesFrom"`
	} `json:"interfaceLanguages"`
	DeviceClasses []string `json:"deviceClasses"`
	RetiredNames  []string `json:"retiredNames"`
}

func loadPositioning(t *testing.T) positioningFacts {
	t.Helper()
	var f positioningFacts
	if err := json.Unmarshal([]byte(readRepoFile(t, "docs", "positioning.json")), &f); err != nil {
		t.Fatalf("docs/positioning.json: %v", err)
	}
	return f
}

// positioningSurfaces are the pages and documents the retired-name check reads.
func positioningSurfaces() []string {
	s := []string{
		"README.md", "README_RU.md", "README_UK.md",
		"docs.html", "docs.ru.html", "docs.uk.html",
		"extension.html", "extension/README.md", "extension/store/LISTING.md",
	}
	return append(s, sitePages()...)
}

func TestPositioningFactsAreConsistent(t *testing.T) {
	f := loadPositioning(t)
	if len(f.Editions) == 0 || len(f.DeviceClasses) == 0 {
		t.Fatal("docs/positioning.json declares no editions or no device classes")
	}
	channelOwner := map[string]string{}
	channelKnown := map[string]bool{}
	for _, c := range f.Channels {
		if c.ID == "" || c.Name == "" || channelKnown[c.ID] {
			t.Errorf("channel %q: empty field or duplicate id", c.ID)
		}
		channelKnown[c.ID] = true
	}
	seenName := map[string]string{}
	for _, e := range f.Editions {
		for _, lang := range []string{"en", "ru", "uk"} {
			n := strings.TrimSpace(e.Names[lang])
			if n == "" {
				t.Errorf("edition %s has no %s display name", e.ID, lang)
				continue
			}
			if other, dup := seenName[lang+"|"+n]; dup {
				t.Errorf("display name %q (%s) is used by %s and %s", n, lang, other, e.ID)
			}
			seenName[lang+"|"+n] = e.ID
		}
		for _, d := range e.DerivesFrom {
			if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(d))); err != nil {
				t.Errorf("edition %s derives from %s, which is not in the tree", e.ID, d)
			}
		}
		for _, c := range e.Channels {
			if !channelKnown[c] {
				t.Errorf("edition %s names the channel %q, which is not declared", e.ID, c)
			}
			if o, dup := channelOwner[c]; dup {
				t.Errorf("channel %s belongs to both %s and %s", c, o, e.ID)
			}
			channelOwner[c] = e.ID
		}
	}
	for _, c := range f.Channels {
		if channelOwner[c.ID] == "" {
			t.Errorf("channel %s belongs to no edition", c.ID)
		}
	}
}

func TestPositioningLanguageCountMatchesTheCode(t *testing.T) {
	f := loadPositioning(t)
	n := f.InterfaceLanguages.Count
	if n != len(f.InterfaceLanguages.Codes) {
		t.Errorf("count %d but %d codes", n, len(f.InterfaceLanguages.Codes))
	}
	if n != len(i18n.Codes) {
		t.Errorf("positioning.json says %d interface languages, internal/i18n.Codes has %d", n, len(i18n.Codes))
	}
	for i, c := range i18n.Codes {
		if i < len(f.InterfaceLanguages.Codes) && f.InterfaceLanguages.Codes[i] != c {
			t.Errorf("code %d: positioning.json has %q, internal/i18n.Codes has %q", i, f.InterfaceLanguages.Codes[i], c)
		}
	}
	dirs, err := os.ReadDir(filepath.Join("..", "extension", "_locales"))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != n {
		t.Errorf("extension/_locales has %d languages, positioning.json says %d", len(dirs), n)
	}
	listings, _ := filepath.Glob(filepath.Join("..", "tools", "store", "listing", "*.txt"))
	if len(listings) != n {
		t.Errorf("tools/store/listing has %d languages, positioning.json says %d", len(listings), n)
	}
}

// A page that states "N languages" in the author languages, and the ten language pages that carry the
// count as a bare number, must agree with the file.
func TestPositioningLanguageCountOnPages(t *testing.T) {
	f := loadPositioning(t)
	n := f.InterfaceLanguages.Count
	claim := regexp.MustCompile(`(\d+)\s+(?:interface\s+)?(?:languages|языках|языков|мовами)`)
	pages := []string{"README.md", "README_RU.md", "README_UK.md", "docs.html", "docs.ru.html", "docs.uk.html", "index.html", "extension.html"}
	stated := 0
	for _, p := range pages {
		for _, m := range claim.FindAllStringSubmatch(readRepoFile(t, strings.Split(p, "/")...), -1) {
			stated++
			if got, _ := strconv.Atoi(m[1]); got != n {
				t.Errorf("%s says %q, positioning.json says %d", p, m[0], n)
			}
		}
	}
	if stated == 0 {
		t.Error("no page states the interface-language count; the claim pattern may have drifted")
	}
	bare := regexp.MustCompile(`(?:^|[^0-9])` + strconv.Itoa(n) + `(?:[^0-9]|$)`)
	for _, c := range localeLandings {
		raw := readRepoFile(t, c, "index.html")
		if bare.MatchString(raw) {
			continue
		}
		// Two language pages write the number in their own way. The forms are for 13; a changed
		// count fails here until they are written again.
		if alt, ok := localeCountForms[c]; ok && n == 13 && strings.Contains(raw, alt) {
			continue
		}
		t.Errorf("%s/index.html does not carry the interface-language count %d", c, n)
	}
}

// localeCountForms is how a language page that does not use Latin digits writes thirteen.
var localeCountForms = map[string]string{
	"ar": "ثلاث عشرة",
	"bn": "১৩",
}

var sectionBoundary = regexp.MustCompile(`(?m)^## `)

// editionNamesIn returns the names a README or docs "Editions" section lists, in order.
func editionNamesIn(t *testing.T, path string) []string {
	t.Helper()
	raw := readRepoFile(t, path)
	var section string
	var item *regexp.Regexp
	if strings.HasSuffix(path, ".html") {
		i := strings.Index(raw, `<section id="editions">`)
		if i < 0 {
			t.Fatalf("%s: no editions section", path)
		}
		section = raw[i:]
		section = section[:strings.Index(section, "</section>")]
		item = regexp.MustCompile(`<li><strong>([^<]+)</strong>`)
	} else {
		heading := regexp.MustCompile(`(?m)^## (?:Editions|Издания|Видання)\s*$`).FindStringIndex(raw)
		if heading == nil {
			t.Fatalf("%s: no editions heading", path)
		}
		section = raw[heading[1]:]
		if next := sectionBoundary.FindStringIndex(section); next != nil {
			section = section[:next[0]]
		}
		item = regexp.MustCompile(`(?m)^- \*\*([^*]+)\*\*`)
	}
	var names []string
	for _, m := range item.FindAllStringSubmatch(section, -1) {
		names = append(names, m[1])
	}
	return names
}

func TestPositioningEditionsListedUnderTheirDisplayNames(t *testing.T) {
	f := loadPositioning(t)
	files := []struct{ lang, readme, docs string }{
		{"en", "README.md", "docs.html"},
		{"ru", "README_RU.md", "docs.ru.html"},
		{"uk", "README_UK.md", "docs.uk.html"},
	}
	for _, fl := range files {
		var want []string
		for _, e := range f.Editions {
			want = append(want, e.Names[fl.lang])
		}
		for _, p := range []string{fl.readme, fl.docs} {
			got := editionNamesIn(t, p)
			if strings.Join(got, " | ") != strings.Join(want, " | ") {
				t.Errorf("%s lists the editions as %q, positioning.json declares %q", p, got, want)
			}
		}
	}
}

func TestPositioningRetiredNamesAreGone(t *testing.T) {
	f := loadPositioning(t)
	for _, p := range positioningSurfaces() {
		raw := readRepoFile(t, strings.Split(p, "/")...)
		for _, r := range f.RetiredNames {
			if strings.Contains(raw, r) {
				t.Errorf("%s still says %q, a retired name (docs/positioning.json)", p, r)
			}
		}
	}
}

// The pillars are read from the table in docs/POSITIONING.md, between its markers.
func positioningPillars(t *testing.T) []string {
	t.Helper()
	raw := readRepoFile(t, "docs", "POSITIONING.md")
	a, b := strings.Index(raw, "<!-- pillars:begin -->"), strings.Index(raw, "<!-- pillars:end -->")
	if a < 0 || b < a {
		t.Fatal("docs/POSITIONING.md has no pillars table between its markers")
	}
	row := regexp.MustCompile("(?m)^\\|\\s*\\d+\\s*\\|\\s*`([A-Z]+)`\\s*\\|")
	var ids []string
	for _, m := range row.FindAllStringSubmatch(raw[a:b], -1) {
		ids = append(ids, m[1])
	}
	if len(ids) == 0 {
		t.Fatal("docs/POSITIONING.md: the pillars table has no rows")
	}
	return ids
}

func TestPositioningPillarOrderOnTheLandings(t *testing.T) {
	want := strings.Join(positioningPillars(t), " ")
	pill := regexp.MustCompile(`class="pill"[^>]*>([A-Z]+)<`)
	for _, p := range append([]string{"index.html"}, func() []string {
		var s []string
		for _, c := range localeLandings {
			s = append(s, c+"/index.html")
		}
		return s
	}()...) {
		raw := readRepoFile(t, strings.Split(p, "/")...)
		i := strings.Index(raw, `id="use-cases"`)
		if i < 0 {
			t.Errorf("%s has no use-cases section", p)
			continue
		}
		section := raw[i:]
		section = section[:strings.Index(section, "</section>")]
		var got []string
		for _, m := range pill.FindAllStringSubmatch(section, -1) {
			got = append(got, m[1])
		}
		if strings.Join(got, " ") != want {
			t.Errorf("%s lists the pillars as %q, docs/POSITIONING.md orders them %q", p, strings.Join(got, " "), want)
		}
	}
}
