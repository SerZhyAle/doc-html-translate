package tests

// Product-site guards (PAGE-STYLE, PAGE-CONTENT, SITE-FAMILY-MAP; docs/contracts/). The site is
// static HTML served from the repository root by GitHub Pages, so nothing but these tests stops a
// page from drifting off the kit, dropping a sibling from the family footer, or bringing back the
// old contact address. What is checked here is what the contracts call machine-checkable: the kit
// bytes, the footer URLs, the contact string, the language value space, the release links, and
// the house typography of the author-language pages.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// kitSHA256 is the SHA-256 of the PAGE-STYLE reference kit (contracts catalog,
// product-web-pages/reference/sza-kit.css, 13037 bytes, LF) measured 2026-09-25. assets/sza-kit.css
// must stay a byte copy; page-local rules go into assets/site.css. Re-vendor and update this
// constant only together with a catalog change of the reference.
const kitSHA256 = "72bd903e7edd4d883106eb296c50b64a6e11731125fab89017320b250332593f"

var localeLandings = []string{"ar", "bn", "de", "es", "fr", "hi", "it", "pt", "ur", "zh"}

// authorPages are written in the author languages (RU/EN/UA) and keep the house typography.
var authorPages = []string{
	"index.html", "extension.html", "docs.html", "docs.ru.html", "docs.uk.html",
	"privacy.html", "install-trust.html", "extension-privacy.html",
}

func sitePages() []string {
	pages := append([]string{}, authorPages...)
	for _, c := range localeLandings {
		pages = append(pages, c+"/index.html")
	}
	return pages
}

// SITE-FAMILY-MAP 1.1 section 2, every row but this product's own.
var familyURLs = []string{
	"https://serzhyale.github.io/FastMediaSorter_mob_v2/",
	"https://serzhyale.github.io/FastMediaSorter_Lite/",
	"https://serzhyale.github.io/CyrFlip/",
	"https://serzhyale.github.io/FileDO/",
	"https://serzhyale.github.io/StreamsPlayer/",
	"https://serzhyale.github.io/OneClickRunner/",
	"https://serzhyale.github.io/universal-agent-kit/",
	"https://sza.od.ua",
}

func TestSiteKitIsReferenceCopy(t *testing.T) {
	// Raw bytes on purpose: .gitattributes keeps this file -text, so a CRLF checkout is a failure.
	b, err := os.ReadFile(filepath.Join("..", "assets", "sza-kit.css"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	if got := hex.EncodeToString(sum[:]); got != kitSHA256 {
		t.Errorf("assets/sza-kit.css is not the PAGE-STYLE reference kit\n got  %s\n want %s\nmove page-local rules to assets/site.css", got, kitSHA256)
	}
}

func TestSitePagesListedInCanonStamp(t *testing.T) {
	var stamp struct {
		Site struct {
			Pages []string `json:"pages"`
		} `json:"site"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, ".sza-canon.json")), &stamp); err != nil {
		t.Fatal(err)
	}
	got := append([]string{}, stamp.Site.Pages...)
	want := sitePages()
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf(".sza-canon.json site.pages does not list the served pages\n got  %v\n want %v", got, want)
	}
}

func TestSitePagesUseKitThenSiteStylesheet(t *testing.T) {
	re := regexp.MustCompile(`<link rel="stylesheet" href="(?:\.\./)?assets/sza-kit\.css">\s*<link rel="stylesheet" href="(?:\.\./)?assets/site\.css">`)
	for _, p := range sitePages() {
		if !re.MatchString(readRepoFile(t, strings.Split(p, "/")...)) {
			t.Errorf("%s does not link assets/sza-kit.css immediately followed by assets/site.css", p)
		}
	}
}

func TestSiteFooterCarriesTheFamilyMap(t *testing.T) {
	footerRe := regexp.MustCompile(`(?s)<footer class="site-footer">.*?</footer>`)
	for _, p := range sitePages() {
		footer := footerRe.FindString(readRepoFile(t, strings.Split(p, "/")...))
		if footer == "" {
			t.Errorf("%s has no site-footer", p)
			continue
		}
		if !strings.Contains(footer, `class="tools-grid"`) || !strings.Contains(footer, "<h2") {
			t.Errorf("%s: footer lacks the headed tools grid", p)
		}
		for _, u := range familyURLs {
			if !strings.Contains(footer, `href="`+u+`"`) {
				t.Errorf("%s: footer does not link %s", p, u)
			}
		}
		if strings.Contains(footer, "https://serzhyale.github.io/doc-html-translate/") {
			t.Errorf("%s: footer grid lists this product itself", p)
		}
		if !strings.Contains(footer, "mailto:sza@ukr.net") {
			t.Errorf("%s: footer lacks the contact sza@ukr.net", p)
		}
	}
}

func TestSiteOneContactAddress(t *testing.T) {
	files := append(sitePages(), "extension/store/PRIVACY.md", "README.md", "README_RU.md", "README_UK.md")
	for _, f := range files {
		if strings.Contains(readRepoFile(t, strings.Split(f, "/")...), "serzhyale@gmail.com") {
			t.Errorf("%s states serzhyale@gmail.com; the one contact is sza@ukr.net (SITE-FAMILY-MAP rule 3)", f)
		}
	}
}

// Every SZA page shares one origin and one localStorage 'sza-lang'. The value space is ru|en|ua;
// a reader maps a stored 'uk' to 'ua' so a page written by an older build never shows all three
// languages at once.
func TestSiteLanguageValueSpace(t *testing.T) {
	bad := regexp.MustCompile(`data-(?:l|lang|set-lang)="uk"`)
	for _, p := range sitePages() {
		raw := readRepoFile(t, strings.Split(p, "/")...)
		if m := bad.FindString(raw); m != "" {
			t.Errorf("%s keys Ukrainian as %s; the value is ua", p, m)
		}
		if strings.Contains(raw, "getItem('sza-lang')") && !strings.Contains(raw, "if(l==='uk')l='ua'") {
			t.Errorf("%s reads sza-lang without mapping a stored uk to ua", p)
		}
	}
	if js := readRepoFile(t, "assets", "site.js"); !strings.Contains(js, "l === 'uk' ? 'ua'") {
		t.Error("assets/site.js no longer maps uk to ua")
	}
}

func TestSiteNoBranchLinksForBinaries(t *testing.T) {
	re := regexp.MustCompile(`github\.com/SerZhyAle/doc-html-translate/(?:tree|blob|raw)/(?:master|main)/build`)
	for _, p := range sitePages() {
		if m := re.FindString(readRepoFile(t, strings.Split(p, "/")...)); m != "" {
			t.Errorf("%s links a binary on a branch (%s); link /releases/latest", p, m)
		}
	}
}

// House typography on the author-language pages: no long dash outside <title>, no ellipsis
// character, no three-dot ellipsis. Scripts and styles are code, not prose.
func TestSiteAuthorPagesTypography(t *testing.T) {
	strip := regexp.MustCompile(`(?s)<script\b.*?</script>|<style\b.*?</style>|<title>.*?</title>`)
	for _, p := range authorPages {
		raw := readRepoFile(t, p)
		prose := strip.ReplaceAllString(raw, "")
		for _, bad := range []string{"—", "…", "..."} {
			if strings.Contains(prose, bad) {
				t.Errorf("%s: prose carries %q (house style: plain hyphen, '..')", p, bad)
			}
		}
		for _, m := range regexp.MustCompile(`(?s)<script\b.*?</script>`).FindAllString(raw, -1) {
			if strings.Contains(m, "—") {
				t.Errorf("%s: a script string carries a long dash", p)
			}
		}
	}
}
