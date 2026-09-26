package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// E30: the package's dc:language is read, and a page that declares no language of its own
// carries it, while a page that declares one keeps its own.
func TestExtractDeclaresPackageLanguage(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "lang.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	addFile(t, w, "mimetype", "application/epub+zip")
	addFile(t, w, "META-INF/container.xml", `<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`)
	addFile(t, w, "OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Книга</dc:title>
    <dc:language> ru-ru </dc:language>
  </metadata>
  <manifest>
    <item id="a" href="a.xhtml" media-type="application/xhtml+xml"/>
    <item id="b" href="b.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="a"/><itemref idref="b"/></spine>
</package>`)
	addFile(t, w, "OEBPS/a.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>a</title></head><body><p>Текст</p></body></html>`)
	addFile(t, w, "OEBPS/b.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en"><head><title>b</title></head><body><p>Quote</p></body></html>`)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out")
	book, err := Extract(epubPath, out)
	if err != nil {
		t.Fatal(err)
	}
	if book.Language != "ru-RU" {
		t.Errorf("Language = %q, want ru-RU", book.Language)
	}
	for name, want := range map[string]string{"a.html": `lang="ru-RU"`, "b.html": `lang="en"`} {
		data, err := os.ReadFile(filepath.Join(out, "OEBPS", name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s: want %s, got:\n%.200s", name, want, data)
		}
		if name == "b.html" && strings.Contains(string(data), "ru-RU") {
			t.Errorf("b.html: the package language overrode the page's own")
		}
	}
}
