package htmlconv_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/htmlconv"

	"golang.org/x/text/encoding/charmap"
)

// convert writes page into a fresh source dir (plus whatever setup adds), converts it, and
// returns the output dir and the generated page.
func convert(t *testing.T, page []byte, setup func(srcDir string)) (string, string) {
	t.Helper()
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "out")
	for _, d := range []string{srcDir, outDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if setup != nil {
		setup(srcDir)
	}
	htmlPath := filepath.Join(srcDir, "page.html")
	if err := os.WriteFile(htmlPath, page, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := htmlconv.Extract(htmlPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatal(err)
	}
	return outDir, string(out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeImage(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG(t, path)
}

func encode1251(t *testing.T, s string) []byte {
	t.Helper()
	b, err := charmap.Windows1251.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A windows-1251 page must come out as readable Cyrillic in a UTF-8 page, whether it
// declares its charset with <meta charset> or the older http-equiv form.
func TestExtract_HTMLWindows1251(t *testing.T) {
	const text = "Привет, мир"
	for name, meta := range map[string]string{
		"meta-charset": `<meta charset="windows-1251">`,
		"http-equiv":   `<meta http-equiv="Content-Type" content="text/html; charset=windows-1251">`,
	} {
		t.Run(name, func(t *testing.T) {
			src := encode1251(t, "<html><head>"+meta+"<title>Заголовок</title></head><body><p>"+text+"</p></body></html>")
			_, page := convert(t, src, nil)
			if !strings.Contains(page, text) {
				t.Errorf("Cyrillic text not decoded: %q", page)
			}
			if !strings.Contains(page, "<title>Заголовок</title>") {
				t.Error("title not decoded")
			}
			if !strings.Contains(page, `<meta charset="UTF-8">`) || strings.Contains(page, "windows-1251") {
				t.Error("output must declare UTF-8 only")
			}
		})
	}
}

// A UTF-8 page with no declaration and a long ASCII head: the WHATWG sniffer only sees the
// first 1024 bytes and would call it windows-1252.
func TestExtract_HTMLUndeclaredUTF8(t *testing.T) {
	src := "<html><body><p>" + strings.Repeat("ascii ", 300) + "</p><p>Ünïcödé и кириллица</p></body></html>"
	_, page := convert(t, []byte(src), nil)
	if !strings.Contains(page, "Ünïcödé и кириллица") {
		t.Error("undeclared UTF-8 was mis-decoded")
	}
}

// The "Save page as" shape: page.html + page_files/. Every candidate of img srcset and
// <picture><source srcset> is copied and rewritten with its descriptor kept.
func TestExtract_HTMLSavedPageSrcset(t *testing.T) {
	src := `<html><body>
<img src="page_files/a.png" srcset="page_files/a.png 1x, page_files/b@2x.png 2x">
<picture><source srcset="page_files/b@2x.png 480w, https://cdn.example/c.png 800w"><img src="page_files/a.png"></picture>
<video><source src="page_files/clip.mp4"></video>
</body></html>`
	outDir, page := convert(t, []byte(src), func(dir string) {
		writeImage(t, filepath.Join(dir, "page_files", "a.png"))
		writeImage(t, filepath.Join(dir, "page_files", "b@2x.png"))
	})
	if strings.Contains(page, "page_files/a.png") || strings.Contains(page, "page_files/b@2x.png") {
		t.Errorf("source-relative image refs left in output: %s", page)
	}
	for _, want := range []string{`src="a.png"`, `srcset="a.png 1x, b_2x.png 2x"`, `srcset="b_2x.png 480w, https://cdn.example/c.png 800w"`, `page_files/clip.mp4`} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %s in %s", want, page)
		}
	}
	pngs, _ := filepath.Glob(filepath.Join(outDir, "*.png"))
	if len(pngs) != 2 {
		t.Errorf("want 2 copied images (one per source file), got %v", pngs)
	}
}

// Two images that differ only in case must not overwrite each other on a case-insensitive
// filesystem.
func TestExtract_HTMLNamesCaseInsensitive(t *testing.T) {
	src := `<html><body><img src="x/A.png"><img src="y/a.png"></body></html>`
	outDir, page := convert(t, []byte(src), func(dir string) {
		writeFile(t, filepath.Join(dir, "x", "A.png"), "upper")
		writeFile(t, filepath.Join(dir, "y", "a.png"), "lower")
	})
	srcs := regexp.MustCompile(`src="([^"]+)"`).FindAllStringSubmatch(page, -1)
	if len(srcs) != 2 {
		t.Fatalf("want 2 srcs, got %v", srcs)
	}
	if strings.EqualFold(srcs[0][1], srcs[1][1]) {
		t.Fatalf("names collide case-insensitively: %q vs %q", srcs[0][1], srcs[1][1])
	}
	for i, want := range []string{"upper", "lower"} {
		got, err := os.ReadFile(filepath.Join(outDir, srcs[i][1]))
		if err != nil || string(got) != want {
			t.Errorf("%s holds %q (err %v), want %q", srcs[i][1], got, err, want)
		}
	}
}

// An image named like a file the generator writes (the tab icon) is renamed so the icon
// does not overwrite it.
func TestExtract_HTMLReservedNameAvoided(t *testing.T) {
	src := `<html><body><img src="img/favicon.ico"><img src="img/INDEX.html"></body></html>`
	_, page := convert(t, []byte(src), func(dir string) {
		writeFile(t, filepath.Join(dir, "img", "favicon.ico"), "icon")
		writeFile(t, filepath.Join(dir, "img", "INDEX.html"), "x")
	})
	if strings.Contains(page, `src="favicon.ico"`) || strings.Contains(strings.ToLower(page), `src="index.html"`) {
		t.Errorf("an asset took a reserved output name: %s", page)
	}
	if !strings.Contains(page, `src="favicon_2.ico"`) {
		t.Errorf("favicon.ico not renamed: %s", page)
	}
}

// A symlink inside the source folder that points outside it is refused: nothing is copied
// and the src is dropped.
func TestExtract_HTMLSymlinkOutsideRefused(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "private.png")
	writePNG(t, outside)
	src := `<html><body><img src="leak.png" alt="x"></body></html>`
	outDir, page := convert(t, []byte(src), func(dir string) {
		if err := os.Symlink(outside, filepath.Join(dir, "leak.png")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	})
	if _, err := os.Stat(filepath.Join(outDir, "leak.png")); err == nil {
		t.Error("a symlink out of the source tree was followed and copied")
	}
	if strings.Contains(page, "leak.png") || !strings.Contains(page, `alt="x"`) {
		t.Errorf("want the <img> kept without its src: %s", page)
	}
}

// The source's declared language and direction reach the output <html>; a page that
// declares none gets none.
func TestExtract_HTMLLangDir(t *testing.T) {
	_, page := convert(t, []byte(`<html lang="ar" dir="rtl"><body><p>مرحبا</p></body></html>`), nil)
	if !strings.Contains(page, `<html lang="ar" dir="rtl">`) {
		t.Errorf("lang/dir not carried: %s", page[:80])
	}
	_, page = convert(t, []byte(`<html><body><p>text</p></body></html>`), nil)
	if !strings.Contains(page, "<html>\n") {
		t.Errorf("an undeclared language must not be invented: %s", page[:60])
	}
}

// Head CSS: inline <style> and a local linked sheet are kept (with their url() assets
// copied), a remote sheet and a remote @import are dropped, and the kept sheets come after
// the baseline style so the reader CSS injected later still wins.
func TestExtract_HTMLHeadStylesKept(t *testing.T) {
	src := `<html><head>
<link rel="stylesheet" href="https://fonts.example/remote.css">
<style>body { background: url("img/bg.png"); }</style>
<link rel="stylesheet" href="css/site.css">
<link rel="stylesheet" href="css/print.css" media="print">
<script src="app.js"></script>
</head><body><p style="background-image: url('img/bg.png')">x</p></body></html>`
	outDir, page := convert(t, []byte(src), func(dir string) {
		writeImage(t, filepath.Join(dir, "img", "bg.png"))
		writeImage(t, filepath.Join(dir, "img", "tex.png"))
		writeFile(t, filepath.Join(dir, "css", "site.css"),
			"@import url(\"https://fonts.example/x.css\");\n.t { background: url(../img/tex.png) }\n.u { background: url(../../outside.png) }\n")
		writeFile(t, filepath.Join(dir, "css", "print.css"), "body { color: red }\n")
	})
	if strings.Contains(page, "remote.css") || strings.Contains(page, "app.js") {
		t.Error("remote stylesheet or script kept")
	}
	links := regexp.MustCompile(`<link rel="stylesheet" href="([^"]+)">`).FindAllStringSubmatch(page, -1)
	if len(links) != 3 {
		t.Fatalf("want 3 kept stylesheets, got %v", links)
	}
	if strings.Index(page, "</style>") > strings.Index(page, links[0][0]) {
		t.Error("kept stylesheets must follow the baseline style")
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(b)
	}
	if got := read(links[0][1]); !strings.Contains(got, `url("bg.png")`) {
		t.Errorf("inline style url not rewritten: %s", got)
	}
	site := read(links[1][1])
	if strings.Contains(site, "fonts.example") || !strings.Contains(site, `url("tex.png")`) || !strings.Contains(site, "none") {
		t.Errorf("linked sheet not rewritten: %s", site)
	}
	if got := read(links[2][1]); !strings.Contains(got, "print;") {
		t.Errorf("media=print sheet not scoped: %s", got)
	}
	for _, f := range []string{"bg.png", "tex.png"} {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Errorf("%s not copied", f)
		}
	}
	if !strings.Contains(page, `url(&#34;bg.png&#34;)`) && !strings.Contains(page, `url("bg.png")`) {
		t.Errorf("style attribute not rewritten: %s", page)
	}
}
