package htmlgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

var bookVarRe = regexp.MustCompile(`var BOOK = ("[^"]*");`)

func scriptBookKey(t *testing.T, page string) string {
	t.Helper()
	m := bookVarRe.FindStringSubmatch(readFile(t, page))
	if m == nil {
		t.Fatalf("%s carries no reader script", page)
	}
	return m[1]
}

// E2: chapter pages are written before translation and index.html after it, when the
// title is already translated. Both must namespace the saved position under one key, or
// "Continue reading" never appears - with the key the pipeline sets and with the one
// derived on first use.
func TestReaderKeySameOnChaptersAndIndexAcrossTranslation(t *testing.T) {
	for _, preset := range []bool{true, false} {
		dir := t.TempDir()
		book := writeTextBook(t, dir, 3)
		if preset {
			book.ReaderKey = ReaderKey("book.epub", 1234, book.Title, len(book.Spine))
		}
		if err := InjectNavBars(book, dir, "book.epub"); err != nil {
			t.Fatal(err)
		}
		book.Title = "Translated Title"
		if _, err := GenerateIndex(book, dir); err != nil {
			t.Fatal(err)
		}
		index := scriptBookKey(t, filepath.Join(dir, "index.html"))
		for _, ch := range []string{"ch_001.html", "ch_003.html"} {
			if got := scriptBookKey(t, filepath.Join(dir, ch)); got != index {
				t.Errorf("preset=%v: %s key %s, index key %s", preset, ch, got, index)
			}
		}
	}
}

// E24: two different books that share a title and page count must not share a saved
// position; the same book converted twice must.
func TestReaderKeyDistinguishesBooks(t *testing.T) {
	a := ReaderKey("a.epub", 100, "Title", 3)
	if a != ReaderKey("a.epub", 100, "Title", 3) {
		t.Error("the key is not deterministic")
	}
	for _, other := range []string{
		ReaderKey("b.epub", 100, "Title", 3),
		ReaderKey("a.epub", 101, "Title", 3),
		ReaderKey("a.epub", 100, "Title", 4),
		ReaderKey("a.epub", 100, "", 3),
	} {
		if other == a {
			t.Errorf("distinct books share key %s", a)
		}
	}
}

// E13: a URL fragment is an explicit destination, so the saved position is restored only
// when the page was opened without one.
func TestReaderScriptRestoresOnlyWithoutFragment(t *testing.T) {
	s := readerScript("k", "OEBPS/ch.html", 1, 2)
	if !strings.Contains(s, `if (!location.hash && saved && saved.href === SELF`) {
		t.Errorf("restore is not gated on an empty location.hash:\n%s", s)
	}
}

// E19: values reach the script as JS literals that cannot end the <script> element.
func TestReaderScriptEscapesValues(t *testing.T) {
	s := readerScript(`k</script><img src=x>`, "a\u2028b\"c.html", 1, 2)
	if strings.Contains(s, "</script><img") {
		t.Error("a value closed the script element")
	}
	for _, want := range []string{`var BOOK = "k\u003c/script\u003e\u003cimg src=x\u003e";`, `var SELF = "a\u2028b\"c.html";`} {
		if !strings.Contains(s, want) {
			t.Errorf("script lacks %s", want)
		}
	}
}

// E23: a spine that lists one file twice, or a second injection pass, must leave the page
// with one bar and one reader script.
func TestInjectNavBarsIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 2)
	book.Spine = append(book.Spine, book.Spine[0])
	for pass := 0; pass < 2; pass++ {
		if err := InjectNavBars(book, dir, "book.epub"); err != nil {
			t.Fatal(err)
		}
	}
	page := readFile(t, filepath.Join(dir, "ch_001.html"))
	if n := strings.Count(page, `class="dht-navbar"`); n != 1 {
		t.Errorf("ch_001.html has %d navbars, want 1", n)
	}
	if n := strings.Count(page, readerMarker); n != 1 {
		t.Errorf("ch_001.html has %d reader scripts, want 1", n)
	}
}

// E18: an external TOC target is the author's URL - kept as written, never put under the
// base folder - and only web and mail links stay clickable.
func TestRenderTOCExternalLinks(t *testing.T) {
	out := renderTOCTree([]epub.TOCEntry{
		{Title: "Web", Href: "https://example.com/a?b=1&c=2"},
		{Title: "Mail", Href: "mailto:a@example.com"},
		{Title: "Evil", Href: "javascript:alert(1)"},
		{Title: "Data", Href: "data:text/html,x"},
		{Title: "Inner", Href: "ch.html#x"},
	}, "OEBPS", 0)
	for _, want := range []string{
		`<a href="https://example.com/a?b=1&amp;c=2">Web</a>`,
		`<a href="mailto:a@example.com">Mail</a>`,
		`<span class="toc-section">Evil</span>`,
		`<span class="toc-section">Data</span>`,
		`<a href="OEBPS/ch.html#x">Inner</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("TOC lacks %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "javascript:") || strings.Contains(out, "OEBPS/https") {
		t.Errorf("unsafe or base-prefixed external link rendered:\n%s", out)
	}
}

// E19: a heading id goes into the fallback TOC href as a URL fragment, escaped.
func TestFallbackTOCEscapesHeadingIDs(t *testing.T) {
	got := nestHeadings([]flatHeading{{level: 1, title: "T", id: "a b#c%d"}}, "p.html")
	if len(got) != 1 || got[0].Href != "p.html#a%20b%23c%25d" {
		t.Errorf("heading href = %+v", got)
	}
}

// E17/E23: a redirect target reaches the script as a JS literal.
func TestSinglePageIndexRedirectIsJSLiteral(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 1)
	if err := os.Rename(filepath.Join(dir, "ch_001.html"), filepath.Join(dir, "a#b.html")); err != nil {
		t.Fatal(err)
	}
	book.Manifest[0].Href = "a#b.html"
	if _, err := GenerateSinglePageIndex(book, dir); err != nil {
		t.Fatal(err)
	}
	if idx := readFile(t, filepath.Join(dir, "index.html")); !strings.Contains(idx, `location.replace("a%23b.html")`) {
		t.Errorf("redirect not encoded:\n%s", idx)
	}
}
