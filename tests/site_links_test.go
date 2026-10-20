package tests

// DOC-EXTERNAL-QUALITY rule 7, link integrity. Every internal reference on every announced page must
// land on a file Pages serves, and every fragment on an id of the page it names. An address on this
// site written in full (canonical, hreflang, og:image) is resolved the same way, so a renamed
// screenshot breaks the build instead of the share card. External hosts are not fetched: the suite
// runs offline, and a third-party outage is not a defect of this repository.

import (
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// linkAttrs are the attributes that carry an address a reader or a crawler follows. data-href is
// the docs trio's language switch, navigation in all but name.
var linkAttrs = map[string]bool{"href": true, "src": true, "data-href": true, "poster": true}

// metaAddresses are the head fields whose value is an address rather than prose.
var metaAddresses = map[string]bool{"og:url": true, "og:image": true, "twitter:image": true}

type siteRef struct {
	page, raw string
}

// resolveSiteRef maps a reference on page to the repository file it lands on and its fragment.
// ok=false means the reference leaves this site and is not checked.
func resolveSiteRef(page, raw string) (file, fragment string, ok bool) {
	if strings.HasPrefix(raw, siteBase) {
		raw = "/" + strings.TrimPrefix(raw, siteBase)
	} else if rawPrefix := strings.TrimSuffix(siteBase, "/"); raw == rawPrefix {
		raw = "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "", "", false
	}
	if u.Path == "" {
		return page, u.Fragment, true
	}
	target := u.Path
	if strings.HasPrefix(target, "/") {
		target = path.Clean(target)
	} else {
		target = path.Join("/", path.Dir(page), target)
	}
	if strings.HasSuffix(u.Path, "/") || target == "/" {
		target = path.Join(target, "index.html")
	}
	target = strings.TrimPrefix(target, "/")
	return target, u.Fragment, true
}

func collectSiteRefs(page string, doc *html.Node) []siteRef {
	var refs []siteRef
	walkHTML(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		for _, a := range n.Attr {
			if linkAttrs[a.Key] {
				refs = append(refs, siteRef{page, a.Val})
			}
			if a.Key == "srcset" {
				for _, part := range strings.Split(a.Val, ",") {
					if f := strings.Fields(part); len(f) > 0 {
						refs = append(refs, siteRef{page, f[0]})
					}
				}
			}
		}
		if prop, ok := attrOf(n, "property"); ok && metaAddresses[prop] {
			v, _ := attrOf(n, "content")
			refs = append(refs, siteRef{page, v})
		}
		if name, ok := attrOf(n, "name"); ok && metaAddresses[name] {
			v, _ := attrOf(n, "content")
			refs = append(refs, siteRef{page, v})
		}
		return true
	})
	return refs
}

func TestSiteInternalLinksResolve(t *testing.T) {
	anchors := map[string]map[string]bool{}
	anchorsOf := func(file string) map[string]bool {
		if a, ok := anchors[file]; ok {
			return a
		}
		a := pageAnchors(parseSitePage(t, file))
		anchors[file] = a
		return a
	}
	checked := 0
	for _, p := range sitePages() {
		for _, r := range collectSiteRefs(p, parseSitePage(t, p)) {
			file, frag, ok := resolveSiteRef(p, r.raw)
			if !ok {
				continue
			}
			checked++
			if !repoPathExists(file) {
				t.Errorf("%s: %q lands on %s, which the site does not serve", p, r.raw, file)
				continue
			}
			if frag == "" || !strings.HasSuffix(file, ".html") {
				continue
			}
			if !anchorsOf(file)[frag] {
				t.Errorf("%s: %q names #%s, which %s does not define", p, r.raw, frag, file)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no internal reference was found on any site page; the collector is broken")
	}
}

// Pages is served over https only; a plain http reference is mixed content or a downgrade.
func TestSiteNoPlainHTTP(t *testing.T) {
	for _, p := range sitePages() {
		if strings.Contains(readRepoFile(t, strings.Split(p, "/")...), "http://") {
			t.Errorf("%s carries an http:// reference; use https://", p)
		}
	}
}

func TestResolveSiteRef(t *testing.T) {
	cases := []struct{ page, raw, file, frag string }{
		{"index.html", "#get", "index.html", "get"},
		{"index.html", "de/", "de/index.html", ""},
		{"index.html", "?lang=ru", "index.html", ""},
		{"index.html", "?l=ru", "index.html", ""},
		{"docs.html", "./", "index.html", ""},
		{"docs.html", "index.html#get", "index.html", "get"},
		{"de/index.html", "../", "index.html", ""},
		{"de/index.html", "../?lang=uk", "index.html", ""},
		{"de/index.html", "../?l=uk", "index.html", ""},
		{"de/index.html", "../fr/", "fr/index.html", ""},
		{"de/index.html", "../assets/site.css", "assets/site.css", ""},
		{"de/index.html", siteBase + "de/", "de/index.html", ""},
		{"de/index.html", siteBase, "index.html", ""},
		{"docs.html", siteBase + "tools/store/gui-de.png", "tools/store/gui-de.png", ""},
	}
	for _, c := range cases {
		file, frag, ok := resolveSiteRef(c.page, c.raw)
		if !ok || file != c.file || frag != c.frag {
			t.Errorf("resolveSiteRef(%q, %q) = %q, %q, %v; want %q, %q", c.page, c.raw, file, frag, ok, c.file, c.frag)
		}
	}
	for _, raw := range []string{"https://github.com/SerZhyAle/doc-html-translate", "mailto:sza@ukr.net"} {
		if _, _, ok := resolveSiteRef("index.html", raw); ok {
			t.Errorf("resolveSiteRef treats %q as internal", raw)
		}
	}
}

// languageQueryPages read the locale query in the head, before first paint: the pages that carry
// their three languages in one file, and the 404 page, which does the same for any path it answers.
func languageQueryPages() []string { return append(append([]string{}, inPageTrioPages...), "404.html") }

// PAGE-STYLE 3.1 amendment 1: ?lang= is the canonical locale query, ?l= the legacy spelling a reader
// still accepts, and a valid lang wins over l. The reader is a page-local script that never touches
// sza-theme (TestSitePagesCarryTheCatalogResolver), so it is found by what it reads.
func TestSitePagesReadTheLanguageQuery(t *testing.T) {
	script := regexp.MustCompile(`(?s)<script>(.*?)</script>`)
	for _, p := range languageQueryPages() {
		src := readRepoFile(t, p)
		var reader string
		for _, m := range script.FindAllStringSubmatch(src[:strings.Index(src, "</head>")], -1) {
			if strings.Contains(m[1], "location.search") {
				reader = m[1]
			}
		}
		lang := strings.Index(reader, "[?&]lang=(ru|uk|ua|en)")
		legacy := strings.Index(reader, "[?&]l=(ru|uk|ua|en)")
		if lang < 0 || legacy < 0 || lang > legacy {
			t.Errorf("%s: the head query reader must try ?lang= first and ?l= second, over ru|uk|ua|en only: %q", p, reader)
		}
	}
}

// The addresses the site emits - hreflang, the language links of the landings, links between its own
// pages, the sitemap - use the canonical spelling. The legacy one stays readable, not written.
func TestSiteEmitsTheCanonicalLanguageQuery(t *testing.T) {
	legacy := regexp.MustCompile(`[?&]l=[a-z]{2}\b`)
	files := append(append([]string{}, languageQueryPages()...), sitePages()...)
	files = append(files, "sitemap.xml")
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f] {
			continue
		}
		seen[f] = true
		if m := legacy.FindString(readRepoFile(t, strings.Split(f, "/")...)); m != "" {
			t.Errorf("%s writes %q; the canonical spelling is ?lang= (docs/SITE_ADDRESSES.md, policy 6)", f, m)
		}
	}
}

// SITE-STRUCTURE 8.B: a page opened by ?lang= (or a page of one language) hands its language to the pages that carry
// their languages in one file. assets/site.js names those pages in one pattern; a page added to the in-page set
// without being named there would silently drop the visitor's language, so the two lists are held together.
func TestSiteCarriesTheVisitorLanguageToEveryInPagePage(t *testing.T) {
	js := readRepoFile(t, "assets", "site.js")
	m := regexp.MustCompile(`var CARRY = /\^\(\?:([^)]*)\)`).FindStringSubmatch(js)
	if m == nil {
		t.Fatal("assets/site.js no longer declares the CARRY pattern of the pages that take the carried language")
	}
	named := map[string]bool{}
	for _, alt := range strings.Split(m[1], "|") {
		named[strings.ReplaceAll(alt, `\`, "")] = true
	}
	for _, p := range inPageTrioPages {
		if !named[p] {
			t.Errorf("assets/site.js CARRY does not name %s; a visitor on a ?lang= page would lose the language there", p)
		}
	}
	if !named["./"] {
		t.Error("assets/site.js CARRY does not name the landing address ./")
	}
	for _, p := range languageQueryPages() {
		if !strings.Contains(readRepoFile(t, p), "'data-lang-src','query'") {
			t.Errorf("%s: the head reader does not mark a query-selected language (data-lang-src), so site.js cannot carry it", p)
		}
	}
}
