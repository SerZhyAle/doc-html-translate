package tests

// DOC-EXTERNAL-QUALITY rule 7, link integrity. Every internal reference on every announced page must
// land on a file Pages serves, and every fragment on an id of the page it names. An address on this
// site written in full (canonical, hreflang, og:image) is resolved the same way, so a renamed
// screenshot breaks the build instead of the share card. External hosts are not fetched: the suite
// runs offline, and a third-party outage is not a defect of this repository.

import (
	"net/url"
	"path"
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
		{"index.html", "?l=ru", "index.html", ""},
		{"docs.html", "./", "index.html", ""},
		{"docs.html", "index.html#get", "index.html", "get"},
		{"de/index.html", "../", "index.html", ""},
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
