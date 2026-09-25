package epub

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
)

// parseBody parses an HTML body fragment into a full document.
func parseBody(t *testing.T, body string) *gohtml.Node {
	t.Helper()
	doc, err := gohtml.Parse(strings.NewReader("<html><head></head><body>" + body + "</body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// renderBody renders the children of <body>.
func renderBody(t *testing.T, doc *gohtml.Node) string {
	t.Helper()
	body := findElementByName(doc, "body")
	if body == nil {
		t.Fatal("no body")
	}
	var buf bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if err := gohtml.Render(&buf, c); err != nil {
			t.Fatal(err)
		}
	}
	return buf.String()
}

func findElementByName(n *gohtml.Node, name string) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.Data == name {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findElementByName(c, name); f != nil {
			return f
		}
	}
	return nil
}

func TestRewriteCoverSVGs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // "" means the input must stay an <svg> with no <img>
	}{
		{
			name: "stretching cover (preserveAspectRatio=none, xlink:href)",
			in:   `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="100%" height="100%" viewBox="0 0 960 1500" preserveAspectRatio="none"><image width="960" height="1500" xlink:href="cover.jpeg"/></svg>`,
			want: `<img src="cover.jpeg" alt=""/>`,
		},
		{
			name: "calibre cover with attrs after href, title and whitespace",
			in:   "<svg class=\"calibre\">\n  <title>Cover</title>\n  <image xlink:href=\"image_rsrcSB.jpg\" height=\"1920\" width=\"1200\"/>\n</svg>",
			want: `<img src="image_rsrcSB.jpg" alt=""/>`,
		},
		{
			name: "plain href without xlink prefix",
			in:   `<svg width="100%" height="100%"><image href="cover.png"/></svg>`,
			want: `<img src="cover.png" alt=""/>`,
		},
		{name: "multi-image svg left untouched", in: `<svg><image xlink:href="a.jpg"/><image xlink:href="b.jpg"/></svg>`},
		{name: "icon svg without raster image left untouched", in: `<svg viewBox="0 0 24 24"><path d="M0 0h24v24H0z"/></svg>`},
		{name: "svg with text left untouched", in: `<svg><image href="a.jpg"/><text>Caption</text></svg>`},
		{
			name: "two separate cover svgs both rewritten",
			in:   `<svg><image href="a.jpg"/></svg><hr/><svg><image href="b.jpg"/></svg>`,
			want: `<img src="a.jpg" alt=""/><hr/><img src="b.jpg" alt=""/>`,
		},
		{
			// E6: the old regex matched from the first <svg> to the second
			// </svg> and replaced the prose in between with one <img>.
			name: "paragraphs between two single-image svgs survive",
			in:   `<svg><image href="a.jpg"/></svg><p>First paragraph.</p><p>Second paragraph.</p><svg><image href="b.jpg"/></svg>`,
			want: `<img src="a.jpg" alt=""/><p>First paragraph.</p><p>Second paragraph.</p><img src="b.jpg" alt=""/>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseBody(t, tc.in)
			changed := rewriteCoverSVGs(doc)
			got := renderBody(t, doc)
			if tc.want == "" {
				if changed || strings.Contains(got, "<img") || !strings.Contains(got, "<svg") {
					t.Errorf("svg must stay untouched, got: %s", got)
				}
				return
			}
			if got != tc.want {
				t.Errorf("rewriteCoverSVGs()\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestXHTMLToHTMLSyntax(t *testing.T) {
	in := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title/><script type="text/javascript" src="x.js"/></head>
<body><p><a id="p1"/>One<br/>two</p><div/><span/><p>Three</p><svg><rect width="1"/><title/></svg><img src="a.png"/></body></html>`
	got := string(xhtmlToHTMLSyntax([]byte(in)))
	for _, want := range []string{
		`<title></title>`,
		`<script type="text/javascript" src="x.js"></script>`,
		`<a id="p1"></a>One<br/>two`,
		`<div></div><span></span>`,
		`<svg><rect width="1"/><title/></svg>`,
		`<img src="a.png"/>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<?xml") {
		t.Errorf("XML declaration kept:\n%s", got)
	}
}

func TestRewriteSrcset(t *testing.T) {
	fn := func(target string) (string, bool) {
		if target == "img/a.png" {
			return "img/b.png", true
		}
		return "", false
	}
	got, ok := rewriteSrcset("text", "../img/a.png 1x, data:image/png;base64,AA,BB 2x,../img/a.png", fn)
	want := "../img/b.png 1x, data:image/png;base64,AA,BB 2x,../img/b.png"
	if !ok || got != want {
		t.Errorf("rewriteSrcset() = %q, %v; want %q", got, ok, want)
	}
}

func TestDecodeToUTF8(t *testing.T) {
	text := "Привет, мир"
	enc, err := charmap.Windows1251.NewEncoder().String(`<?xml version="1.0" encoding="windows-1251"?><p>` + text + `</p>`)
	if err != nil {
		t.Fatal(err)
	}
	out, recoded, err := decodeToUTF8([]byte(enc))
	if err != nil || !recoded || !strings.Contains(string(out), text) {
		t.Errorf("windows-1251: recoded=%v err=%v out=%q", recoded, err, out)
	}

	metaEnc, _ := charmap.Windows1251.NewEncoder().String(`<html><head><meta charset="windows-1251"></head><body>` + text + `</body></html>`)
	out, recoded, _ = decodeToUTF8([]byte(metaEnc))
	if !recoded || !strings.Contains(string(out), text) {
		t.Errorf("meta charset: recoded=%v out=%q", recoded, out)
	}

	utf := `<?xml version="1.0" encoding="UTF-8"?><p>` + text + `</p>`
	out, recoded, _ = decodeToUTF8([]byte(utf))
	if recoded || string(out) != utf {
		t.Errorf("utf-8 must pass through unchanged, recoded=%v", recoded)
	}
}

// fidelityOPF places the chapters under text/ or other/ so links cross
// directories, and keeps the OPF at the root so index.xhtml hits the reserved
// generated index.html name.
const fidelityContainer = `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

const fidelityOPF = `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Fidelity</dc:title></metadata>
  <manifest>
    <item id="idx" href="index.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch1" href="text/ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="text/ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="sub" href="other/sub.xhtml" media-type="application/xhtml+xml"/>
    <item id="cp" href="text/cp1251.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="idx"/><itemref idref="ch1"/><itemref idref="ch2"/><itemref idref="sub"/><itemref idref="cp"/></spine>
</package>`

const calibreChapter = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="ru">
<head><title>Ch1</title><script type="text/javascript" src="../js/calibre.js"/></head>
<body>
<p class="calibre"><a id="p1"/>First words of the chapter.</p>
<p>See index.html and chapter.xhtml for details.</p>
<p><a href="ch2.xhtml#s2">Next</a> <a href="../index.xhtml#top">Home</a> <a href="https://example.com/index.html">Web</a></p>
</body></html>`

const subChapter = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Sub</title></head>
<body><p><a href="../text/ch1.xhtml#p1">Back</a> <a href="../text/ch2.xhtml">Two</a> <a href="#local">Here</a></p></body></html>`

func extractFidelityBook(t *testing.T) (string, *Book) {
	t.Helper()
	dir := t.TempDir()
	cp1251, err := charmap.Windows1251.NewEncoder().String(`<?xml version="1.0" encoding="windows-1251"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>CP</title><meta http-equiv="Content-Type" content="text/html; charset=windows-1251"/></head>
<body><p>Съешь же ещё этих мягких французских булок</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	epubPath := buildEPUB(t, dir, "fidelity.epub", [][2]string{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", fidelityContainer},
		{"content.opf", fidelityOPF},
		{"index.xhtml", chapterXHTML("Intro")},
		{"text/ch1.xhtml", calibreChapter},
		{"text/ch2.xhtml", chapterXHTML("Two")},
		{"other/sub.xhtml", subChapter},
		{"text/cp1251.xhtml", cp1251},
	})
	out := filepath.Join(dir, "out")
	book, err := Extract(epubPath, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return out, book
}

func readOut(t *testing.T, out, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// E12: the XML self-closing <a id/> and <script/> must not swallow the text.
func TestExtract_CalibreSelfClosingTagsKeepBodyText(t *testing.T) {
	out, _ := extractFidelityBook(t)
	html := readOut(t, out, "text/ch1.html")
	doc, err := gohtml.Parse(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	body := findElementByName(doc, "body")
	if body == nil || !strings.Contains(textContent(body), "First words of the chapter.") {
		t.Fatalf("body text lost:\n%s", html)
	}
	for _, tag := range []string{"a", "script"} {
		var check func(*gohtml.Node)
		check = func(n *gohtml.Node) {
			if n.Type == gohtml.ElementNode && n.Data == tag && n.Namespace == "" {
				if strings.Contains(textContent(n), "First words") {
					t.Errorf("chapter text ended up inside <%s>", tag)
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				check(c)
			}
		}
		check(doc)
	}
	if strings.Contains(html, "<?xml") {
		t.Error("XML declaration kept in HTML output")
	}
	if !strings.Contains(html, `lang="ru"`) {
		t.Error("xml:lang not mapped to lang")
	}
}

// E11: only real link attributes are rewritten, relative to their own file.
func TestExtract_LinkRewriteTouchesOnlyLinks(t *testing.T) {
	out, book := extractFidelityBook(t)
	ch1 := readOut(t, out, "text/ch1.html")
	for _, want := range []string{
		"See index.html and chapter.xhtml for details.",
		`href="ch2.html#s2"`,
		`href="../_content_index.html#top"`,
		`href="https://example.com/index.html"`,
	} {
		if !strings.Contains(ch1, want) {
			t.Errorf("text/ch1.html missing %q:\n%s", want, ch1)
		}
	}
	sub := readOut(t, out, "other/sub.html")
	for _, want := range []string{`href="../text/ch1.html#p1"`, `href="../text/ch2.html"`, `href="#local"`} {
		if !strings.Contains(sub, want) {
			t.Errorf("other/sub.html missing %q:\n%s", want, sub)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "_content_index.html")); err != nil {
		t.Errorf("reserved rename not written: %v", err)
	}
	if got := book.SpineHrefs()[0]; got != "_content_index.html" {
		t.Errorf("spine[0] = %q, want _content_index.html", got)
	}
}

// E14: a windows-1251 chapter is transcoded and declared as UTF-8.
func TestExtract_Windows1251ChapterBecomesUTF8(t *testing.T) {
	out, _ := extractFidelityBook(t)
	html := readOut(t, out, "text/cp1251.html")
	if !strings.Contains(html, "Съешь же ещё этих мягких французских булок") {
		t.Errorf("text not decoded:\n%s", html)
	}
	if !strings.Contains(html, `<meta charset="utf-8"/>`) || strings.Contains(strings.ToLower(html), "windows-1251") {
		t.Errorf("charset declaration not replaced:\n%s", html)
	}
}
