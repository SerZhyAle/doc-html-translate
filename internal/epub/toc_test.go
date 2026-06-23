package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// buildEPUB writes a zip of the given (name, content) entries and returns its path.
func buildEPUB(t *testing.T, dir, filename string, entries [][2]string) string {
	t.Helper()
	epubPath := filepath.Join(dir, filename)
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, e := range entries {
		addFile(t, w, e[0], e[1])
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return epubPath
}

const containerOPF = `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

func chapterXHTML(title string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>` + title + `</title></head>
<body><h1 id="top">` + title + `</h1><p>Body.</p><h2 id="sec2">Section Two</h2><p>More.</p></body></html>`
}

const ncxXML = `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1"><navLabel><text>Chapter One</text></navLabel><content src="chapter1.xhtml"/>
      <navPoint id="np1-1"><navLabel><text>Section Two</text></navLabel><content src="chapter1.xhtml#sec2"/></navPoint>
    </navPoint>
    <navPoint id="np2"><navLabel><text>Chapter Two</text></navLabel><content src="chapter2.xhtml"/></navPoint>
    <navPoint id="np3"><navLabel><text>Ghost</text></navLabel><content src="ghost.xhtml"/></navPoint>
  </navMap>
</ncx>`

func TestParseTOC_NCX(t *testing.T) {
	dir := t.TempDir()
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>NCX Book</dc:title></metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx"><itemref idref="ch1"/><itemref idref="ch2"/></spine>
</package>`

	epubPath := buildEPUB(t, dir, "ncx.epub", [][2]string{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", containerOPF},
		{"OEBPS/content.opf", opf},
		{"OEBPS/toc.ncx", ncxXML},
		{"OEBPS/chapter1.xhtml", chapterXHTML("Chapter One")},
		{"OEBPS/chapter2.xhtml", chapterXHTML("Chapter Two")},
	})

	book, err := Extract(epubPath, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// Two top-level entries; the dangling "Ghost" target was dropped.
	if len(book.TOC) != 2 {
		t.Fatalf("expected 2 top-level TOC entries, got %d: %+v", len(book.TOC), book.TOC)
	}
	if book.TOC[0].Title != "Chapter One" || book.TOC[0].Href != "chapter1.html" {
		t.Errorf("entry 0 = %+v, want {Chapter One chapter1.html}", book.TOC[0])
	}
	if len(book.TOC[0].Children) != 1 {
		t.Fatalf("expected 1 child under entry 0, got %d", len(book.TOC[0].Children))
	}
	if c := book.TOC[0].Children[0]; c.Title != "Section Two" || c.Href != "chapter1.html#sec2" {
		t.Errorf("child = %+v, want {Section Two chapter1.html#sec2}", c)
	}
	if book.TOC[1].Title != "Chapter Two" || book.TOC[1].Href != "chapter2.html" {
		t.Errorf("entry 1 = %+v, want {Chapter Two chapter2.html}", book.TOC[1])
	}
}

func TestParseTOC_NavPreferredOverNCX(t *testing.T) {
	dir := t.TempDir()
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Nav Book</dc:title></metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx"><itemref idref="ch1"/><itemref idref="ch2"/></spine>
</package>`

	nav := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><title>TOC</title></head><body>
<nav epub:type="toc"><ol>
  <li><a href="chapter1.xhtml">Nav Chapter One</a>
    <ol><li><a href="chapter1.xhtml#sec2">Nav Section Two</a></li></ol>
  </li>
  <li><a href="chapter2.xhtml">Nav Chapter Two</a></li>
</ol></nav></body></html>`

	epubPath := buildEPUB(t, dir, "nav.epub", [][2]string{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", containerOPF},
		{"OEBPS/content.opf", opf},
		{"OEBPS/nav.xhtml", nav},
		{"OEBPS/toc.ncx", ncxXML},
		{"OEBPS/chapter1.xhtml", chapterXHTML("Chapter One")},
		{"OEBPS/chapter2.xhtml", chapterXHTML("Chapter Two")},
	})

	book, err := Extract(epubPath, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(book.TOC) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(book.TOC), book.TOC)
	}
	// Titles must come from nav.xhtml, not toc.ncx.
	if book.TOC[0].Title != "Nav Chapter One" {
		t.Errorf("expected nav to be preferred (title %q), got %q", "Nav Chapter One", book.TOC[0].Title)
	}
	if book.TOC[0].Href != "chapter1.html" {
		t.Errorf("entry 0 href = %q, want chapter1.html", book.TOC[0].Href)
	}
	if len(book.TOC[0].Children) != 1 || book.TOC[0].Children[0].Href != "chapter1.html#sec2" {
		t.Errorf("expected nested section href chapter1.html#sec2, got %+v", book.TOC[0].Children)
	}
}

func TestParseTOC_ReservedNameRename(t *testing.T) {
	dir := t.TempDir()
	// With the OPF at the EPUB root, a chapter literally named index.xhtml
	// normalizes to index.html, which collides with the generated index.html
	// and is renamed to _content_index.html; the TOC must follow the rename.
	container := `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Reserved</dc:title></metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="idx" href="index.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx"><itemref idref="idx"/><itemref idref="ch2"/></spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap>
  <navPoint id="n1"><navLabel><text>Intro</text></navLabel><content src="index.xhtml"/></navPoint>
  <navPoint id="n2"><navLabel><text>Two</text></navLabel><content src="chapter2.xhtml"/></navPoint>
</navMap></ncx>`

	epubPath := buildEPUB(t, dir, "reserved.epub", [][2]string{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", container},
		{"content.opf", opf},
		{"toc.ncx", ncx},
		{"index.xhtml", chapterXHTML("Intro")},
		{"chapter2.xhtml", chapterXHTML("Chapter Two")},
	})

	book, err := Extract(epubPath, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.TOC) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(book.TOC), book.TOC)
	}
	if book.TOC[0].Href != "_content_index.html" {
		t.Errorf("reserved-renamed target = %q, want _content_index.html", book.TOC[0].Href)
	}
}
