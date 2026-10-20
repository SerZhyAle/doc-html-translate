package tests

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/config"
)

func TestJPEGXLConversion(t *testing.T) {
	picture, err := os.ReadFile("fixtures/jxl/rgba.jxl")
	if err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".jxl", ".cbz", ".html", ".epub", ".fb2"} {
		t.Run(ext, func(t *testing.T) {
			dir, out := t.TempDir(), t.TempDir()
			input := filepath.Join(dir, "picture"+ext)
			data := picture
			if ext == ".epub" {
				var archive bytes.Buffer
				z := zip.NewWriter(&archive)
				files := map[string][]byte{
					"mimetype":               []byte("application/epub+zip"),
					"META-INF/container.xml": []byte(`<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OPS/book.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
					"OPS/book.opf":           []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>JPEG XL Book</dc:title></metadata><manifest><item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/><item id="art" href="art.JXL" media-type="image/jxl"/></manifest><spine><itemref idref="chapter"/></spine></package>`),
					"OPS/chapter.xhtml":      []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Art</title></head><body><p>A picture in a book.</p><img src="art.JXL"/></body></html>`),
					"OPS/art.JXL":            picture,
				}
				for name, b := range files {
					w, err := z.Create(name)
					if err != nil {
						t.Fatal(err)
					}
					if _, err = w.Write(b); err != nil {
						t.Fatal(err)
					}
				}
				if err := z.Close(); err != nil {
					t.Fatal(err)
				}
				data = archive.Bytes()
			} else if ext == ".fb2" {
				data = []byte(`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink"><description><title-info><book-title>JPEG XL Book</book-title></title-info></description><body><section><p>A picture in a book.</p><image l:href="#art"/></section></body><binary id="art" content-type="image/jxl">` + base64.StdEncoding.EncodeToString(picture) + `</binary></FictionBook>`)
			} else if ext == ".cbz" {
				var archive bytes.Buffer
				z := zip.NewWriter(&archive)
				w, err := z.Create("page01.JXL")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = w.Write(picture); err != nil {
					t.Fatal(err)
				}
				if err = z.Close(); err != nil {
					t.Fatal(err)
				}
				data = archive.Bytes()
			} else if ext == ".html" {
				if err := os.WriteFile(filepath.Join(dir, "art.JXL"), picture, 0600); err != nil {
					t.Fatal(err)
				}
				data = []byte(`<html><body><p>A picture in a document.</p><img src="art.JXL"></body></html>`)
			}
			if err := os.WriteFile(input, data, 0600); err != nil {
				t.Fatal(err)
			}
			code, err := app.New(config.Config{InputFile: input, OutputFolder: out, NoOpen: true, NoTranslate: true, SinglePage: true, UILang: "en"}).Run()
			if err != nil || code != 0 {
				t.Fatalf("conversion exit=%d: %v", code, err)
			}
			index := findIndexHTML(out)
			b, err := os.ReadFile(index)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), ".png") {
				t.Fatal("output does not reference a PNG")
			}
			found := false
			err = filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || filepath.Ext(path) != ".png" {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				im, err := png.Decode(bytes.NewReader(b))
				if err != nil {
					return err
				}
				if im.Bounds().Dx() == 3 && im.Bounds().Dy() == 2 {
					found = true
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !found {
				t.Fatal("converted picture missing")
			}
		})
	}
}
