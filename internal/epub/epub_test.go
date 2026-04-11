package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createMinimalEPUB creates a minimal valid EPUB file for testing.
func createMinimalEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "test.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)

	// mimetype
	addFile(t, w, "mimetype", "application/epub+zip")

	// container.xml
	addFile(t, w, "META-INF/container.xml", `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	// content.opf
	addFile(t, w, "OEBPS/content.opf", `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="css1" href="style.css" media-type="text/css"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`)

	// Chapter files
	addFile(t, w, "OEBPS/chapter1.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html><head><title>Chapter 1</title></head><body><h1>Chapter 1</h1><p>Hello world</p></body></html>`)
	addFile(t, w, "OEBPS/chapter2.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html><head><title>Chapter 2</title></head><body><h1>Chapter 2</h1><p>Goodbye world</p></body></html>`)

	// CSS
	addFile(t, w, "OEBPS/style.css", `body { margin: 1em; }`)

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return epubPath
}

func addFile(t *testing.T, w *zip.Writer, name, content string) {
	t.Helper()
	fw, err := w.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
}

func TestExtractMinimalEPUB(t *testing.T) {
	tmpDir := t.TempDir()
	epubPath := createMinimalEPUB(t, tmpDir)

	outputDir := filepath.Join(tmpDir, "output")
	book, err := Extract(epubPath, outputDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if book.Title != "Test Book" {
		t.Errorf("expected title 'Test Book', got '%s'", book.Title)
	}

	if len(book.Spine) != 2 {
		t.Errorf("expected 2 spine items, got %d", len(book.Spine))
	}

	if len(book.Manifest) != 3 {
		t.Errorf("expected 3 manifest items, got %d", len(book.Manifest))
	}

	if book.BasePath != "OEBPS" {
		t.Errorf("expected BasePath 'OEBPS', got '%s'", book.BasePath)
	}

	// Verify files were extracted
	ch1 := filepath.Join(outputDir, "OEBPS", "chapter1.xhtml")
	if _, err := os.Stat(ch1); os.IsNotExist(err) {
		t.Error("chapter1.xhtml not extracted")
	}
}

func TestSpineHrefs(t *testing.T) {
	book := &Book{
		Manifest: []ManifestItem{
			{ID: "ch1", Href: "chapter1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "ch2", Href: "chapter2.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "css", Href: "style.css", MediaType: "text/css"},
		},
		Spine: []SpineItem{
			{IDRef: "ch2"},
			{IDRef: "ch1"},
		},
	}

	hrefs := book.SpineHrefs()
	if len(hrefs) != 2 {
		t.Fatalf("expected 2 hrefs, got %d", len(hrefs))
	}
	if hrefs[0] != "chapter2.xhtml" {
		t.Errorf("expected first href 'chapter2.xhtml', got '%s'", hrefs[0])
	}
	if hrefs[1] != "chapter1.xhtml" {
		t.Errorf("expected second href 'chapter1.xhtml', got '%s'", hrefs[1])
	}
}

func TestPathTraversalProtection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a malicious EPUB with path traversal
	epubPath := filepath.Join(tmpDir, "evil.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)
	addFile(t, w, "mimetype", "application/epub+zip")
	addFile(t, w, "META-INF/container.xml", `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	addFile(t, w, "content.opf", `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata><title>Evil</title></metadata>
  <manifest></manifest>
  <spine></spine>
</package>`)
	// This file should be skipped (path traversal attempt logged as warning)
	addFile(t, w, "../../../etc/evil.txt", "pwned")

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	_, err = Extract(epubPath, outputDir)
	// Should succeed (best-effort) but the evil file should NOT exist outside output
	if err != nil {
		t.Fatalf("Extract should succeed with best-effort, got: %v", err)
	}

	evilPath := filepath.Join(tmpDir, "etc", "evil.txt")
	if _, err := os.Stat(evilPath); err == nil {
		t.Error("path traversal succeeded — evil.txt was created outside output dir")
	}
}
