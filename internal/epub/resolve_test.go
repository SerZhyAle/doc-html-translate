package epub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveBookPathSharedCases runs the fixture the extension's
// extension/test/epub.test.mjs reads too, so the two editions cannot drift
// on which book-supplied names resolve and where.
func TestResolveBookPathSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "epub_href_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Base string  `json:"base"`
			Href string  `json:"href"`
			Want *string `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("no cases in fixture")
	}
	for _, c := range fx.Cases {
		got, err := resolveBookPath(c.Base, c.Href)
		switch {
		case c.Want == nil && err == nil:
			t.Errorf("resolveBookPath(%q, %q) = %q, want it refused", c.Base, c.Href, got)
		case c.Want != nil && err != nil:
			t.Errorf("resolveBookPath(%q, %q) refused (%v), want %q", c.Base, c.Href, err, *c.Want)
		case c.Want != nil && got != *c.Want:
			t.Errorf("resolveBookPath(%q, %q) = %q, want %q", c.Base, c.Href, got, *c.Want)
		}
	}
}

func TestRelToBase(t *testing.T) {
	cases := []struct{ base, p, want string }{
		{".", "a/b.html", "a/b.html"},
		{"OEBPS", "OEBPS/ch.html", "ch.html"},
		{"OEBPS/text", "OEBPS/images/c.jpg", "../images/c.jpg"},
		{"OEBPS", "root.html", "../root.html"},
	}
	for _, c := range cases {
		if got := relToBase(c.base, c.p); got != c.want {
			t.Errorf("relToBase(%q, %q) = %q, want %q", c.base, c.p, got, c.want)
		}
	}
}

func TestURLPath(t *testing.T) {
	cases := map[string]string{
		"OEBPS/ch1.html":    "OEBPS/ch1.html",
		"Chapter 1.html":    "Chapter%201.html",
		"a#b.html":          "a%23b.html",
		"100%.html":         "100%25.html",
		"q?.html":           "q%3F.html",
		"中.html":            "中.html",
		"a</script>b.html":  "a%3C/script%3Eb.html",
		"../images/c d.jpg": "../images/c%20d.jpg",
	}
	for in, want := range cases {
		if got := URLPath(in); got != want {
			t.Errorf("URLPath(%q) = %q, want %q", in, got, want)
		}
	}
}
