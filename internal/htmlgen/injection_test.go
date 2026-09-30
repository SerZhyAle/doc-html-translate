package htmlgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTurkishTitleMultipageInjection(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 2)
	for _, name := range []string{"ch_001.html", "ch_002.html"} {
		content := `<!doctype html><html lang="tr"><HEAD><title>İstanbul Kitap</title><style>.book { color: red }</style></HEAD><BODY><p>İçerik</p></BODY></html>`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := InjectNavBars(book, dir, "İstanbul.epub"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ch_001.html", "ch_002.html"} {
		content := readFile(t, filepath.Join(dir, name))
		headEnd := indexASCIIFold(content, "</head>")
		bodyStart := indexASCIIFold(content, "<body")
		if headEnd < 0 || bodyStart < 0 || headEnd > bodyStart {
			t.Fatalf("%s: broken head/body boundaries", name)
		}
		head := content[:headEnd]
		body := content[bodyStart:]
		if !strings.Contains(head, `.book { color: red }</style>`) || !strings.Contains(head, `<style id="dht-nav">`) {
			t.Errorf("%s: book style or navbar CSS missing from head", name)
		}
		if !strings.Contains(body, `class="dht-navbar"`) || !strings.Contains(body, `<p>İçerik</p>`) {
			t.Errorf("%s: navbar or book text missing from body", name)
		}
		if strings.Contains(body, "style>") {
			t.Errorf("%s: stray style text in body", name)
		}
		if n := strings.Count(content, `rel="icon"`); n != 1 {
			t.Errorf("%s: %d favicon links, want one", name, n)
		}
	}
}

func TestSingleSpineFaviconInsertedOnce(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 1)
	page := filepath.Join(dir, "ch_001.html")
	if err := os.WriteFile(page, []byte(`<html><head><title>İstanbul</title></head><body>İçerik</body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateSinglePageIndex(book, dir); err != nil {
		t.Fatal(err)
	}
	if err := InjectNavBars(book, dir, "İstanbul.epub"); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, page)
	if n := strings.Count(content, `rel="icon"`); n != 1 {
		t.Errorf("single-spine page has %d favicon links, want one", n)
	}
	if !strings.Contains(content[:indexASCIIFold(content, "</head>")], `<style id="dht-nav">`) {
		t.Error("navbar CSS missing from head")
	}
}

func TestIndexASCIIFoldPreservesByteOffsets(t *testing.T) {
	content := "<title>İ K " + string([]byte{0xff}) + "</title></HEAD><BODY class='book'>"
	for _, tag := range []string{"</head>", "<body"} {
		idx := indexASCIIFold(content, tag)
		if idx < 0 || !strings.EqualFold(content[idx:idx+len(tag)], tag) {
			t.Errorf("%s: incorrect byte offset %d", tag, idx)
		}
	}
}
