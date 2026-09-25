package htmlgen

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"

	gohtml "golang.org/x/net/html"
)

// sigilBook is the layout Sigil writes: the package in OEBPS/, chapters in Text/, images
// in Images/, styles in Styles/. Chapter two holds the footnote chapter one links to, and
// both chapters use id="top", so the merge has to rename one of them.
func sigilBook() map[string]string {
	const opf = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Sigil Book</dc:title></metadata>
  <manifest>
    <item id="c1" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="Text/ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="pic" href="Images/pic.png" media-type="image/png"/>
    <item id="hi" href="Images/pic%20hi.png" media-type="image/png"/>
    <item id="bg" href="Images/bg.png" media-type="image/png"/>
    <item id="css" href="Styles/style.css" media-type="text/css"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`
	const ch1 = `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml" xml:lang="de"><head><title>1</title>
<link rel="stylesheet" href="../Styles/style.css"/></head><body>
<h1 id="top">One</h1>
<p id="p1">Text<a id="ref1" href="ch2.xhtml#note1">1</a>, <a href="ch2.xhtml">next</a>, <a href="#top">up</a>.</p>
<img src="../Images/pic.png" srcset="../Images/pic.png 1x, ../Images/pic%20hi.png 2x" alt=""/>
<div style="background:url('../Images/bg.png')">x</div>
<style>.deco { background: url(../Images/bg.png) }</style>
<label for="p1">label</label>
</body></html>`
	const ch2 = `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>2</title></head>
<body id="start2"><h1 id="top">Two</h1>
<p id="note1">Note. <a href="ch1.xhtml#ref1">back</a> <a href="#top">top</a> <a href="#start2">start</a></p>
<p><a href="https://example.com/x">web</a></p>
</body></html>`
	return map[string]string{
		"mimetype":                "application/epub+zip",
		"META-INF/container.xml":  container("OEBPS/content.opf"),
		"OEBPS/content.opf":       opf,
		"OEBPS/Text/ch1.xhtml":    ch1,
		"OEBPS/Text/ch2.xhtml":    ch2,
		"OEBPS/Images/pic.png":    "png",
		"OEBPS/Images/pic hi.png": "png",
		"OEBPS/Images/bg.png":     "png",
		"OEBPS/Styles/style.css":  "p { margin: 0 }",
	}
}

// convertSingle extracts files as an EPUB and runs the default single-page merge,
// returning the merged page's text and the directory it lives in.
func convertSingle(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	root := t.TempDir()
	epubPath := filepath.Join(root, "book.epub")
	writeEPUB(t, epubPath, files)
	out := filepath.Join(root, "out")
	book, err := epub.Extract(epubPath, out)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := GenerateSinglePage(book, out, "book.epub"); err != nil {
		t.Fatalf("GenerateSinglePage: %v", err)
	}
	mergedDir := filepath.Join(out, "OEBPS")
	return readFile(t, filepath.Join(mergedDir, "index.html")), mergedDir
}

// E3/E4: in the Sigil layout every image and style reference resolves from the merged
// page, every in-page link lands on an id that exists, and no id is used twice.
func TestSinglePageMergeSigilLayout(t *testing.T) {
	merged, mergedDir := convertSingle(t, sigilBook())

	doc, err := gohtml.Parse(strings.NewReader(merged))
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int{}
	var hrefs, srcs []string
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			if id := nodeAttr(n, "id"); id != "" {
				ids[id]++
			}
			if n.Data == "a" {
				hrefs = append(hrefs, nodeAttr(n, "href"))
			}
			if n.Data == "img" {
				srcs = append(srcs, nodeAttr(n, "src"))
				for _, c := range strings.Split(nodeAttr(n, "srcset"), ",") {
					if f := strings.Fields(c); len(f) > 0 {
						srcs = append(srcs, f[0])
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	for id, n := range ids {
		if n > 1 {
			t.Errorf("id %q appears %d times in the merged page", id, n)
		}
	}
	for _, h := range hrefs {
		if !strings.HasPrefix(h, "#") {
			continue
		}
		frag, _ := url.PathUnescape(h[1:])
		if ids[frag] == 0 {
			t.Errorf("link %q lands on no element", h)
		}
	}
	for _, s := range srcs {
		p, _ := url.PathUnescape(s)
		if _, err := os.Stat(filepath.Join(mergedDir, filepath.FromSlash(p))); err != nil {
			t.Errorf("image %q does not resolve from the merged page: %v", s, err)
		}
	}

	for _, want := range []string{
		`href="#note1"`,                // the footnote marker reaches the note in chapter two
		`href="#ref1"`,                 // and the note links back to the marker
		`href="#dht-ch-2"`,             // a bare chapter link lands on the chapter start
		`id="dht-ch-2"`,                // which exists
		`id="c2-top"`,                  // chapter two's colliding id was renamed..
		`href="#c2-top"`,               // ..and its own link followed it
		`href="#top"`,                  // chapter one kept its id and its link
		`url(&#34;Images/bg.png&#34;)`, // the style attribute is rebased
		`url("Images/bg.png")`,         // and so is the <style> block
		`href="https://example.com/x"`,
		`<label for="p1">`,
		`<html lang="de">`, // xml:lang honoured when lang is absent
	} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged page lacks %s", want)
		}
	}
	if strings.Contains(merged, "ch2.html") || strings.Contains(merged, "ch1.html") {
		t.Error("a link still points at a chapter file the merge deletes")
	}
	if !strings.Contains(merged, `href="Styles/style.css"`) {
		t.Error("the manifest stylesheet is not linked from the merged page's folder")
	}
}

// The merge must not change a flat book without cross-references: no markers, ids and
// references untouched.
func TestSinglePageMergeFlatBookUnchanged(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 2)
	if _, err := GenerateSinglePage(book, dir, "t.epub"); err != nil {
		t.Fatal(err)
	}
	merged := readFile(t, filepath.Join(dir, "index.html"))
	if strings.Contains(merged, "dht-chapter-anchor") {
		t.Error("a chapter marker was inserted although nothing links to a chapter")
	}
}

// E17: the page language is copied from the book into an attribute; it must be escaped,
// never able to open a new attribute.
func TestSinglePageLangIsEscaped(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 1)
	page := `<!DOCTYPE html><html lang='en" onload="alert(1)'><body><p>x</p></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "ch_001.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateSinglePage(book, dir, "t.epub"); err != nil {
		t.Fatal(err)
	}
	merged := readFile(t, filepath.Join(dir, "index.html"))
	if strings.Contains(merged, `onload="alert(1)"`) {
		t.Errorf("lang value broke out of its attribute:\n%s", merged[:200])
	}
	doc, err := gohtml.Parse(bytes.NewReader([]byte(merged)))
	if err != nil {
		t.Fatal(err)
	}
	if got := htmlLang(doc); got != `en" onload="alert(1)` {
		t.Errorf("lang round-trip = %q", got)
	}
}
