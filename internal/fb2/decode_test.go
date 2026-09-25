package fb2

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func mustParse(t *testing.T, raw []byte) *fb2Doc {
	t.Helper()
	doc, err := parseFB2(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parseFB2: %v", err)
	}
	return doc
}

// A Russian FB2 declared windows-1251 used to fail outright: encoding/xml reads only UTF-8.
func TestParseFB2DeclaredWindows1251(t *testing.T) {
	src := `<?xml version="1.0" encoding="windows-1251"?>
<FictionBook><description><title-info><book-title>Книга</book-title></title-info></description>
<body><section><p>Привет, мир.</p></section></body></FictionBook>`
	raw, err := charmap.Windows1251.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	doc := mustParse(t, raw)
	if doc.title != "Книга" || len(doc.items) != 1 || doc.items[0].text != "Привет, мир." {
		t.Errorf("got title %q items %+v", doc.title, doc.items)
	}
}

func TestParseFB2DamagedUTF8IsVisible(t *testing.T) {
	doc := mustParse(t, []byte("<FictionBook><body><p>ab\xffcd</p></body></FictionBook>"))
	if len(doc.items) != 1 || doc.items[0].text != "ab\U0000FFFDcd" {
		t.Errorf("got %+v", doc.items)
	}
}

// Verse, subtitles, epigraph authors, citations and table cells used to be dropped.
func TestParseFB2ProseElements(t *testing.T) {
	doc := mustParse(t, []byte(`<FictionBook><body><section>
<subtitle>Part</subtitle>
<epigraph><p>Quote</p><text-author>Someone</text-author></epigraph>
<poem><title><p>Song</p></title>
  <stanza><v>Line one</v><v>Line <emphasis>two</emphasis></v></stanza>
  <stanza><v>Line three</v></stanza>
  <text-author>Poet</text-author></poem>
<cite><p>Cited</p></cite>
<table><tr><th>Head</th></tr><tr><td>Cell</td></tr></table>
</section></body></FictionBook>`))
	want := []fb2Item{
		{text: "Part", class: classSubtitle},
		{text: "Quote"},
		{text: "Someone", class: classTextAuthor},
		{text: "Song"},
		{lines: []string{"Line one", "Line two"}, class: classStanza},
		{lines: []string{"Line three"}, class: classStanza},
		{text: "Poet", class: classTextAuthor},
		{text: "Cited"},
		{text: "Head"},
		{text: "Cell"},
	}
	if len(doc.items) != len(want) {
		t.Fatalf("got %d items %+v, want %d", len(doc.items), doc.items, len(want))
	}
	for i, w := range want {
		g := doc.items[i]
		if g.text != w.text || g.class != w.class || strings.Join(g.lines, "|") != strings.Join(w.lines, "|") {
			t.Errorf("item %d: got %+v, want %+v", i, g, w)
		}
	}
}

// The base64 is decoded in chunks straight into the output; a binary spanning many chunks, with
// FB2's line wrapping, must come out byte-identical.
func TestReadBinaryAcrossChunks(t *testing.T) {
	data := make([]byte, 3*b64Chunk+7)
	for i := range data {
		data[i] = byte(i * 31)
	}
	enc := base64.StdEncoding.EncodeToString(data)
	var wrapped strings.Builder
	for i := 0; i < len(enc); i += 76 {
		wrapped.WriteString(enc[i:min(i+76, len(enc))])
		wrapped.WriteString("\r\n")
	}
	doc := mustParse(t, []byte(`<FictionBook><body><image href="#x"/></body><binary id="x" content-type="image/png">`+
		wrapped.String()+`</binary></FictionBook>`))
	if got := doc.binaries["x"].data; !bytes.Equal(got, data) {
		t.Errorf("decoded %d bytes, want %d identical bytes", len(got), len(data))
	}
}

func TestReadBinaryMalformedIsUnresolved(t *testing.T) {
	doc := mustParse(t, []byte(`<FictionBook><binary id="x">@@@@</binary></FictionBook>`))
	if _, ok := doc.binaries["x"]; ok {
		t.Error("malformed base64 should leave the binary unresolved")
	}
}

func TestImageFileNamesAreUnique(t *testing.T) {
	s := newNameSet()
	got := []string{
		s.imageFileName("cover.jpg", "image/png"),
		s.imageFileName("fig1", "image/png"),
		s.imageFileName("рис1", "image/jpeg"),
		s.imageFileName("рис2", "image/jpeg"),
		s.imageFileName("..", "image/png"),
		s.imageFileName("...", "image/png"),
		s.imageFileName("Fig1", "image/png"),
		s.imageFileName(strings.Repeat("x", 300), "image/gif"),
	}
	if got[0] != "cover.jpg" || got[1] != "fig1.png" {
		t.Errorf("plain ids must keep readable names, got %q %q", got[0], got[1])
	}
	seen := map[string]bool{}
	for _, n := range got {
		lower := strings.ToLower(n)
		if seen[lower] {
			t.Errorf("duplicate name %q in %q", n, got)
		}
		seen[lower] = true
		if strings.HasPrefix(n, ".") || strings.ContainsAny(n, `/\`) || filepath.Base(n) != n || len(n) > 70 {
			t.Errorf("unsafe name %q", n)
		}
	}
}

// An id of ".." used to fail the whole conversion when the image was written.
func TestExtractSurvivesPathologicalImageID(t *testing.T) {
	dir := t.TempDir()
	src := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns:l="http://www.w3.org/1999/xlink"><body><section>
<p>Text</p><image l:href="#.."/><image l:href="#рис1"/><image l:href="#рис2"/>
</section></body>
<binary id=".." content-type="image/png">iVBORw0KGgo=</binary>
<binary id="рис1" content-type="image/png">AAAA</binary>
<binary id="рис2" content-type="image/png">AQID</binary>
</FictionBook>`
	in := filepath.Join(dir, "book.fb2")
	if err := os.WriteFile(in, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(in, out); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	imgs, _ := filepath.Glob(filepath.Join(out, "*.png"))
	if len(imgs) != 3 {
		t.Errorf("want 3 distinct image files, got %v", imgs)
	}
}
