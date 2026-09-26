package epub

import (
	"encoding/json"
	"path"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

type targetCase struct {
	Href  string  `json:"href"`
	Want  *string `json:"want"`
	Frag  string  `json:"frag"`
	About string  `json:"about"`
}

type targetCases struct {
	OPFDir   string       `json:"opfDir"`
	Chapters []string     `json:"chapters"`
	TOCFile  string       `json:"tocFile"`
	TOC      []targetCase `json:"toc"`
	LinkFile string       `json:"linkFile"`
	Links    []targetCase `json:"links"`
}

func loadTargetCases(t *testing.T) targetCases {
	t.Helper()
	var fx targetCases
	if err := json.Unmarshal(sharedFixture(t, "epub_target_cases.json"), &fx); err != nil {
		t.Fatal(err)
	}
	return fx
}

// TestTOCTargetSharedCases: a TOC entry reaches the same chapter as in the extension, which runs the
// same fixture through resolveTocAnchor (audit finding E36).
func TestTOCTargetSharedCases(t *testing.T) {
	fx := loadTargetCases(t)
	book := &Book{BasePath: fx.OPFDir}
	manifest := map[string]bool{}
	for _, ch := range fx.Chapters {
		manifest[relToBase(fx.OPFDir, ch)] = true
	}
	tocDir := path.Dir(relToBase(fx.OPFDir, fx.TOCFile))
	for _, c := range fx.TOC {
		got, ok := resolveTOCHref(c.Href, tocDir, book, manifest)
		if c.Want == nil {
			if ok {
				t.Errorf("%q: resolved to %q, want no target", c.Href, got)
			}
			continue
		}
		want := URLPath(relToBase(fx.OPFDir, *c.Want))
		if c.Frag != "" {
			want += "#" + c.Frag
		}
		if !ok || got != want {
			t.Errorf("%q %s: got %q, %v; want %q", c.Href, c.About, got, ok, want)
		}
	}
}

// TestLinkTargetSharedCases: a chapter link reaches the same chapter as in the extension, which runs
// the same fixture through rewriteAnchor (audit finding E37: a root-relative link was left pointing
// at the drive root).
func TestLinkTargetSharedCases(t *testing.T) {
	fx := loadTargetCases(t)
	fileHref := relToBase(fx.OPFDir, fx.LinkFile)
	noRename := func(string) (string, bool) { return "", false }
	for _, c := range fx.Links {
		doc, err := gohtml.Parse(strings.NewReader(`<a href="` + gohtml.EscapeString(c.Href) + `">x</a>`))
		if err != nil {
			t.Fatal(err)
		}
		rewriteLinks(doc, fx.OPFDir, fileHref, noRename)
		a := findElementByTag(doc, "a")
		href := ""
		for _, at := range a.Attr {
			if at.Key == "href" {
				href = at.Val
			}
		}
		file, frag := splitFragment(href)
		got, err := resolveBookPath(path.Dir(fx.LinkFile), file)
		if strings.HasPrefix(href, "/") || strings.HasPrefix(href, `\`) {
			t.Errorf("%q %s: still root-relative after the rewrite: %q", c.Href, c.About, href)
		}
		if c.Want == nil {
			continue
		}
		if err != nil || got != *c.Want || frag != c.Frag {
			t.Errorf("%q %s: rewritten to %q, reaching %q#%s (%v); want %q#%s", c.Href, c.About, href, got, frag, err, *c.Want, c.Frag)
		}
	}
}

func findElementByTag(n *gohtml.Node, tag string) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findElementByTag(c, tag); f != nil {
			return f
		}
	}
	return nil
}
