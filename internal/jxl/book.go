package jxl

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
)

var cssURL = regexp.MustCompile(`(?i)url\(\s*("[^"]*"|'[^']*'|[^)\s'"]*)\s*\)`)

// NormalizeBook converts extracted pictures before navigation, merging or OCR.
// References are changed as URLs, never by replacing prose in a chapter. Source
// pictures are retained, and generated names cannot overwrite another asset.
func NormalizeBook(ctx context.Context, book *epub.Book, root string) error {
	var files []string
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.Type().IsRegular() {
			files = append(files, p)
		}
		return nil
	}); err != nil {
		return err
	}
	converted := map[string]string{}
	for _, p := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		var head [12]byte
		_, _ = io.ReadFull(f, head[:])
		_ = f.Close()
		if !Is(head[:]) && !strings.EqualFold(filepath.Ext(p), ".jxl") {
			continue
		}
		dst := p + ".png"
		for n := 1; ; n++ {
			_, err := os.Stat(dst)
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return err
			}
			dst = fmt.Sprintf("%s.%d.png", p, n)
		}
		if err := Transcode(p, dst); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		converted[filepath.Clean(p)] = dst
	}

	for _, p := range files {
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".html" && ext != ".htm" && ext != ".xhtml" && ext != ".css" && ext != ".svg" {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if len(converted) == 0 && !bytes.Contains(bytes.ToLower(b), []byte("data:image/jxl")) {
			continue
		}
		var refErr error
		ref := func(s string) string {
			if strings.HasPrefix(strings.ToLower(s), "data:image/jxl;base64,") {
				encoded := s[len("data:image/jxl;base64,"):]
				if len(encoded) > (200<<20)*4/3+4 {
					refErr = fmt.Errorf("JPEG XL data URL exceeds 200 MB")
					return s
				}
				data, err := base64.StdEncoding.DecodeString(encoded)
				if err == nil {
					data, err = PNG(data)
				}
				if err != nil {
					refErr = err
					return s
				}
				return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
			}
			return rewriteRef(s, filepath.Dir(p), converted)
		}
		out := ""
		if ext == ".css" {
			out = rewriteCSS(string(b), ref)
		} else {
			out = rewriteHTML(b, ref)
		}
		if refErr != nil {
			return refErr
		}
		if out != string(b) {
			if err := fsutil.WriteFile(p, []byte(out), 0o644); err != nil {
				return err
			}
		}
	}
	for i := range book.Manifest {
		it := &book.Manifest[i]
		if dst, ok := converted[filepath.Join(root, filepath.FromSlash(book.BasePath), filepath.FromSlash(it.Href))]; ok {
			rel, err := filepath.Rel(filepath.Join(root, filepath.FromSlash(book.BasePath)), dst)
			if err != nil {
				return err
			}
			it.Href, it.MediaType = filepath.ToSlash(rel), "image/png"
		}
	}
	return nil
}

func rewriteRef(s, base string, converted map[string]string) string {
	u, err := url.Parse(s)
	if err != nil || u.IsAbs() || u.Host != "" || u.Path == "" || strings.HasPrefix(u.Path, "/") {
		return s
	}
	if dst, ok := converted[filepath.Clean(filepath.Join(base, filepath.FromSlash(u.Path)))]; ok {
		rel, err := filepath.Rel(base, dst)
		if err != nil {
			return s
		}
		u.Path, u.RawPath = filepath.ToSlash(rel), ""
		// Commas delimit srcset candidates, including in otherwise legal URL paths.
		u.RawPath = strings.ReplaceAll(u.EscapedPath(), ",", "%2C")
		return u.String()
	}
	return s
}

func rewriteCSS(s string, ref func(string) string) string {
	return cssURL.ReplaceAllStringFunc(s, func(m string) string {
		v := strings.Trim(cssURL.FindStringSubmatch(m)[1], "\"'")
		if r := ref(v); r != v {
			return `url("` + r + `")`
		}
		return m
	})
}

func rewriteHTML(b []byte, ref func(string) string) string {
	z := html.NewTokenizer(bytes.NewReader(b))
	var out strings.Builder
	inStyle := false
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			out.Write(z.Raw())
			break
		}
		raw := string(z.Raw())
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			t := z.Token()
			inStyle = t.Data == "style"
			changed := false
			for i := range t.Attr {
				a := &t.Attr[i]
				v := a.Val
				switch a.Key {
				case "src", "href", "xlink:href", "poster", "background", "data":
					a.Val = ref(v)
				case "srcset":
					if !strings.Contains(v, "data:") {
						parts := strings.Split(v, ",")
						updated := false
						for j, p := range parts {
							f := strings.Fields(p)
							if len(f) > 0 {
								r := ref(f[0])
								updated = updated || r != f[0]
								f[0] = r
								parts[j] = strings.Join(f, " ")
							}
						}
						if updated {
							a.Val = strings.Join(parts, ", ")
						}
					}
				case "style":
					a.Val = rewriteCSS(v, ref)

				}
				changed = changed || a.Val != v
			}
			if changed {
				for i := range t.Attr {
					if t.Attr[i].Key == "type" && strings.EqualFold(t.Attr[i].Val, "image/jxl") {
						t.Attr[i].Val = "image/png"
					}
				}
				raw = t.String()
			}
		} else if tt == html.EndTagToken {
			inStyle = false
		} else if tt == html.TextToken && inStyle {
			raw = rewriteCSS(raw, ref)
		}
		out.WriteString(raw)
	}
	return out.String()
}
