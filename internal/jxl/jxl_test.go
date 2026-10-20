package jxl

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gen2brain/jpegxl"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/limits"
)

// The guard only proves the tag was passed; this proves the tag still does what ticket 115 relies on.
// Built with nodynamic, the module reports exactly this error and its init never calls LoadLibrary.
// A module update that renamed the tag would bring the libjxl.dll search back while the guard still
// compiled, and the native path's error (a failed load, or nil with a libjxl.dll found) differs.
func TestNoNativeDecoder(t *testing.T) {
	if err := jpegxl.Dynamic(); err == nil || err.Error() != "jpegxl: dynamic disabled" {
		t.Fatalf("jpegxl.Dynamic() = %v, want the nodynamic build's \"jpegxl: dynamic disabled\"", err)
	}
}

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../tests/fixtures/jxl/rgba.jxl")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPNG(t *testing.T) {
	raw := fixture(t)
	// The same independently generated picture in both legal file envelopes.
	box := []byte{0, 0, 0, 12, 'J', 'X', 'L', ' ', 13, 10, 135, 10, 0, 0, 0, 20, 'f', 't', 'y', 'p', 'j', 'x', 'l', ' ', 0, 0, 0, 0, 'j', 'x', 'l', ' '}
	box = append(box, make([]byte, 8)...)
	binary.BigEndian.PutUint32(box[32:36], uint32(len(raw)+8))
	copy(box[36:], "jxlc")
	box = append(box, raw...)
	for _, b := range [][]byte{raw, box} {
		if !Is(b) {
			t.Fatal("signature not detected")
		}
		out, err := PNG(b)
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatal(err)
		}
		if im.Bounds().Dx() != 3 || im.Bounds().Dy() != 2 {
			t.Fatal(im.Bounds())
		}
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				got := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
				want := color.NRGBA{R: 200, G: uint8(x * 60), B: uint8(y * 80), A: uint8(55 + x*100)}
				if got != want {
					t.Fatalf("pixel %d,%d = %v, want %v", x, y, got, want)
				}
			}
		}
	}
	for _, b := range [][]byte{nil, {255, 10}, []byte("not an image")} {
		if _, err := PNG(b); err == nil {
			t.Fatal("accepted malformed picture")
		}
	}
}

func TestNormalizeBook(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "art"), 0700); err != nil {
		t.Fatal(err)
	}
	put := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	put("art/pic #1.JXL", fixture(t))
	put("art/pic #1.JXL.png", []byte("existing asset"))
	put("chapter.html", []byte(`<p>pic #1.JXL stays prose</p><img src="art/pic%20%231.JXL?x=1#view"><picture><source type="image/jxl" srcset="art/pic%20%231.JXL 1x, art/pic%20%231.JXL 2x"></picture><style>.a{background:url('art/pic%20%231.JXL')}</style><img src="https://example.org/pic.JXL"><img src="data:image/jxl;base64,`+base64.StdEncoding.EncodeToString(fixture(t))+`">`))
	put("art/style.css", []byte(`.a{background:url("pic%20%231.JXL")}`))
	book := &epub.Book{Manifest: []epub.ManifestItem{{Href: "art/pic #1.JXL", MediaType: "image/jxl"}}}
	if err := NormalizeBook(context.Background(), book, dir); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "chapter.html"))
	for _, want := range []string{`<p>pic #1.JXL stays prose</p>`, `src="art/pic%20%231.JXL.1.png?x=1#view"`, `type="image/png"`, `art/pic%20%231.JXL.1.png 2x`, `https://example.org/pic.JXL`, `data:image/png;base64,`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s in %s", want, b)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "art/pic #1.JXL.png")); string(b) != "existing asset" {
		t.Fatal("overwrote existing asset")
	}
	if book.Manifest[0].Href != "art/pic #1.JXL.1.png" || book.Manifest[0].MediaType != "image/png" {
		t.Fatal(book.Manifest)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "art/style.css"))
	if !strings.Contains(string(b), `pic%20%231.JXL.1.png`) {
		t.Fatal(string(b))
	}
}

func TestNormalizeBookCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NormalizeBook(ctx, &epub.Book{}, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestPixelBomb(t *testing.T) {
	// Raw codestream header: large dimensions (60000 squared), no pixel payload.
	b := []byte{255, 10}
	bit := 16
	write := func(v, n int) {
		for i := 0; i < n; i++ {
			if bit/8 == len(b) {
				b = append(b, 0)
			}
			b[bit/8] |= byte((v>>i)&1) << (bit % 8)
			bit++
		}
	}
	write(0, 1)
	write(2, 2)
	write(59999, 18)
	write(1, 3)
	// Reuse the valid metadata after the fixture's 31-bit size header.
	raw := fixture(t)
	for i := 31; i < len(raw)*8; i++ {
		write(int(raw[i/8]>>(i%8))&1, 1)
	}
	_, err := PNG(b)
	if !errors.Is(err, limits.ErrTooLarge) {
		t.Fatalf("want pixel-budget refusal, got %v", err)
	}
}

func TestNormalizeBookBasePath(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "OEBPS")
	if err := os.Mkdir(base, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "a,b.jxl"), fixture(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "chapter.html"), []byte(`<img srcset="a%2Cb.jxl 1x">`), 0600); err != nil {
		t.Fatal(err)
	}
	book := &epub.Book{BasePath: "OEBPS", Manifest: []epub.ManifestItem{{Href: "a,b.jxl", MediaType: "image/jxl"}}}
	if err := NormalizeBook(context.Background(), book, dir); err != nil {
		t.Fatal(err)
	}
	if book.Manifest[0].Href != "a,b.jxl.png" || book.Manifest[0].MediaType != "image/png" {
		t.Fatal(book.Manifest)
	}
	b, err := os.ReadFile(filepath.Join(base, "chapter.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `a%2Cb.jxl.png 1x`) {
		t.Fatal(string(b))
	}
}
