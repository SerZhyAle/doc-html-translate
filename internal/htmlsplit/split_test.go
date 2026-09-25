package htmlsplit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"doc-html-translate/internal/epub"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// testBook writes files (OPF-relative href -> content) under dir/OEBPS and
// returns a book whose spine lists spine hrefs in order.
func testBook(t *testing.T, dir string, files map[string]string, spine []string) *epub.Book {
	t.Helper()
	book := &epub.Book{BasePath: "OEBPS"}
	for href, content := range files {
		p := filepath.Join(dir, "OEBPS", filepath.FromSlash(href))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i, href := range spine {
		id := fmt.Sprintf("item%d", i)
		book.Manifest = append(book.Manifest, epub.ManifestItem{ID: id, Href: href, MediaType: "application/xhtml+xml"})
		book.Spine = append(book.Spine, epub.SpineItem{IDRef: id})
	}
	return book
}

func readPart(t *testing.T, dir, href string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "OEBPS", filepath.FromSlash(href)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func spineHrefs(book *epub.Book) []string {
	byID := map[string]string{}
	for _, m := range book.Manifest {
		byID[m.ID] = m.Href
	}
	var out []string
	for _, s := range book.Spine {
		out = append(out, byID[s.IDRef])
	}
	return out
}

// partHolding returns the spine href whose file contains id="<id>".
func partHolding(t *testing.T, dir string, book *epub.Book, id string) string {
	t.Helper()
	for _, h := range spineHrefs(book) {
		if strings.Contains(readPart(t, dir, h), `id="`+id+`"`) {
			return h
		}
	}
	t.Fatalf("no part holds id %q", id)
	return ""
}

func parseDoc(t *testing.T, s string) *gohtml.Node {
	t.Helper()
	doc, err := gohtml.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func findElement(n *gohtml.Node, a atom.Atom) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findElement(c, a); f != nil {
			return f
		}
	}
	return nil
}

func attr(n *gohtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// wrappedChapter builds a Calibre-style chapter: one wrapper div holding
// paragraphs of roughly total characters, with <h2 id="last"> near the end.
func wrappedChapter(total int) string {
	var b strings.Builder
	b.WriteString(`<html><head><title>Ch</title></head><body><div class="calibre" id="wrap">`)
	para := strings.Repeat("Lorem ipsum dolor sit amet. ", 10)
	b.WriteString(`<p><a href="#last">jump</a></p>`)
	for n := 0; n < total; n += len(para) {
		b.WriteString("<p>" + para + "</p>\n")
	}
	b.WriteString(`<h2 id="last">Last section</h2><p>The end.</p></div></body></html>`)
	return b.String()
}

func TestSplitSingleWrapperRewritesTOC(t *testing.T) {
	dir := t.TempDir()
	book := testBook(t, dir, map[string]string{"text/chapter.html": wrappedChapter(50000)}, []string{"text/chapter.html"})
	book.TOC = []epub.TOCEntry{{Title: "Ch", Href: "text/chapter.html", Children: []epub.TOCEntry{
		{Title: "Last", Href: "text/chapter.html#last"},
	}}}

	added, err := SplitIfNeeded(book, dir, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if added < 5 {
		t.Fatalf("added = %d, want several parts", added)
	}

	for i, h := range spineHrefs(book) {
		doc := parseDoc(t, readPart(t, dir, h))
		div := findElement(findElement(doc, atom.Body), atom.Div)
		if div == nil || attr(div, "class") != "calibre" {
			t.Fatalf("part %s lost the calibre wrapper", h)
		}
		if gotID := attr(div, "id"); (i == 0) != (gotID == "wrap") {
			t.Errorf("part %d wrapper id = %q; only the first part keeps it", i+1, gotID)
		}
	}

	holder := partHolding(t, dir, book, "last")
	if holder == "text/chapter.html" {
		t.Fatalf("#last stayed in the first part; test chapter too small")
	}
	if got, want := book.TOC[0].Children[0].Href, holder+"#last"; got != want {
		t.Errorf("TOC child href = %q, want %q", got, want)
	}
	if got := book.TOC[0].Href; got != "text/chapter.html" {
		t.Errorf("fragment-less TOC href = %q, want it to stay on part 1", got)
	}
}

func TestSplitKeepsRootAndBodyAttributes(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="ar" dir="rtl"><head><title>x</title></head><body dir="rtl" class="x">`)
	for i := 0; i < 40; i++ {
		b.WriteString("<p>" + strings.Repeat("مرحبا بالعالم ", 20) + "</p>")
	}
	b.WriteString("</body></html>")
	book := testBook(t, dir, map[string]string{"ar.html": b.String()}, []string{"ar.html"})

	if added, err := SplitIfNeeded(book, dir, 1000); err != nil || added == 0 {
		t.Fatalf("SplitIfNeeded = %d, %v; want a split", added, err)
	}
	for _, h := range spineHrefs(book) {
		content := readPart(t, dir, h)
		if !strings.HasPrefix(strings.ToLower(content), "<!doctype html>") {
			t.Errorf("%s: missing doctype", h)
		}
		doc := parseDoc(t, content)
		root, body := findElement(doc, atom.Html), findElement(doc, atom.Body)
		if attr(root, "lang") != "ar" || attr(root, "dir") != "rtl" {
			t.Errorf("%s: <html> attrs = %v", h, root.Attr)
		}
		if attr(body, "dir") != "rtl" || attr(body, "class") != "x" {
			t.Errorf("%s: <body> attrs = %v", h, body.Attr)
		}
		if findElement(doc, atom.Title) == nil {
			t.Errorf("%s: head lost", h)
		}
	}
}

func TestSplitCountsCharactersNotBytes(t *testing.T) {
	dir := t.TempDir()
	const limit = 1000
	para := strings.Repeat("Привет мир ", 9) // 99 characters, ~180 bytes
	var b strings.Builder
	b.WriteString("<html><body>")
	for i := 0; i < 30; i++ {
		b.WriteString("<p>" + para + "</p>")
	}
	b.WriteString(`<p>` + strings.Repeat("Длинный абзац ", 100) + `</p></body></html>`)
	book := testBook(t, dir, map[string]string{"ru.html": b.String()}, []string{"ru.html"})

	if _, err := SplitIfNeeded(book, dir, limit); err != nil {
		t.Fatal(err)
	}
	hrefs := spineHrefs(book)
	// 30 x 98 trimmed characters at 1000 per part is 3 parts plus the oversized
	// block alone; counting bytes would roughly double the part count.
	if len(hrefs) != 4 {
		t.Errorf("parts = %d, want 4", len(hrefs))
	}
	for _, h := range hrefs {
		body := findElement(parseDoc(t, readPart(t, dir, h)), atom.Body)
		p := &planner{max: limit, sizes: map[*gohtml.Node]int{}}
		blocks := 0
		for c := body.FirstChild; c != nil; c = c.NextSibling {
			if isSignificant(c) {
				blocks++
			}
		}
		if n := p.measure(body); n > limit && blocks > 1 {
			t.Errorf("%s: %d characters in %d blocks exceeds %d", h, n, blocks, limit)
		}
	}
	if n := utf8.RuneCountInString(para); n != 99 {
		t.Fatalf("fixture paragraph = %d characters", n)
	}
}

func TestSplitRewritesCrossFileAndInFileLinks(t *testing.T) {
	dir := t.TempDir()
	other := `<html><body><p><a href="../text/chapter.html#last">to last</a>
<a href="../text/chapter.html">to start</a> <a href="https://example.com/chapter.html#last">ext</a></p></body></html>`
	book := testBook(t, dir, map[string]string{
		"misc/other.html":   other,
		"text/chapter.html": wrappedChapter(20000),
	}, []string{"misc/other.html", "text/chapter.html"})

	if _, err := SplitIfNeeded(book, dir, 4000); err != nil {
		t.Fatal(err)
	}
	holder := partHolding(t, dir, book, "last")
	if holder == "text/chapter.html" {
		t.Fatal("#last stayed in part 1; test chapter too small")
	}

	got := readPart(t, dir, "misc/other.html")
	want := `href="../` + holder + `#last"`
	if !strings.Contains(got, want) {
		t.Errorf("cross-file link not rewritten; want %s in:\n%s", want, got)
	}
	for _, keep := range []string{`href="../text/chapter.html"`, `href="https://example.com/chapter.html#last"`} {
		if !strings.Contains(got, keep) {
			t.Errorf("link %s should be untouched in:\n%s", keep, got)
		}
	}

	first := readPart(t, dir, "text/chapter.html")
	wantIn := `href="` + filepath.Base(holder) + `#last"`
	if !strings.Contains(first, wantIn) {
		t.Errorf("in-file link not rewritten; want %s in part 1", wantIn)
	}
}

func TestSplitUntouchedKeepsBytes(t *testing.T) {
	dir := t.TempDir()
	const src = "<?xml version=\"1.0\"?>\n<html xmlns=\"http://www.w3.org/1999/xhtml\"><body><p>short</p><br/></body></html>"
	other := `<html><body><a href="small.html#x">x</a></body></html>`
	book := testBook(t, dir, map[string]string{"small.html": src, "other.html": other}, []string{"small.html", "other.html"})

	added, err := SplitIfNeeded(book, dir, 5000)
	if err != nil || added != 0 {
		t.Fatalf("SplitIfNeeded = %d, %v", added, err)
	}
	if got := readPart(t, dir, "small.html"); got != src {
		t.Errorf("unsplit file changed:\n%s", got)
	}
	if got := readPart(t, dir, "other.html"); got != other {
		t.Errorf("unrelated file changed:\n%s", got)
	}
}

func TestSplitDoesNotDescendIntoProse(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("<html><body><div>")
	for i := 0; i < 50; i++ {
		b.WriteString("sentence <b>bold</b> ")
	}
	b.WriteString("</div></body></html>")
	dir := t.TempDir()
	p := filepath.Join(dir, "p.html")
	if err := os.WriteFile(p, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	parts, err := chunkHTMLFile(p, 100)
	if err != nil {
		t.Fatal(err)
	}
	if parts != nil {
		t.Errorf("a wrapper holding inline prose was split into %d parts", len(parts))
	}
}
