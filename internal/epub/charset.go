package epub

import (
	"bytes"
	"regexp"
	"strings"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

// charsetPrescanBytes bounds the declaration search, mirroring the HTML
// encoding-sniffing prescan window.
const charsetPrescanBytes = 1024

var xmlDeclEncodingRe = regexp.MustCompile(`^\s*<\?xml\s[^>]*?\bencoding\s*=\s*["']([A-Za-z0-9._:-]+)["']`)

// decodeToUTF8 returns a chapter's bytes as UTF-8. The encoding comes from the
// BOM, then the XML declaration, then a <meta> charset; undeclared content is
// taken as UTF-8, the XML default. recoded reports that the bytes were
// transcoded, so the file must be rewritten even if nothing else changed.
func decodeToUTF8(data []byte) (out []byte, recoded bool, err error) {
	switch {
	case bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}):
		return data[3:], false, nil
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
		return transcode(unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM), data)
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		return transcode(unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM), data)
	}

	head := data
	if len(head) > charsetPrescanBytes {
		head = head[:charsetPrescanBytes]
	}
	label := ""
	if m := xmlDeclEncodingRe.FindSubmatch(head); m != nil {
		label = string(m[1])
	} else {
		label = metaCharset(head)
	}
	if label == "" {
		return data, false, nil
	}
	enc, name := charset.Lookup(label)
	// A declaration readable as ASCII rules out UTF-16 without a BOM, so such a
	// label is wrong and the content is treated as UTF-8 (as HTML does).
	if enc == nil || name == "utf-8" || strings.HasPrefix(name, "utf-16") {
		return data, false, nil
	}
	return transcode(enc, data)
}

func transcode(enc encoding.Encoding, data []byte) ([]byte, bool, error) {
	out, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return data, false, err
	}
	return bytes.TrimPrefix(out, []byte{0xEF, 0xBB, 0xBF}), true, nil
}

// metaCharset returns the charset declared by <meta charset> or a
// Content-Type http-equiv <meta> within head, or "".
func metaCharset(head []byte) string {
	z := gohtml.NewTokenizer(bytes.NewReader(head))
	for {
		tt := z.Next()
		if tt == gohtml.ErrorToken {
			return ""
		}
		if tt != gohtml.StartTagToken && tt != gohtml.SelfClosingTagToken {
			continue
		}
		tok := z.Token()
		if tok.Data != "meta" {
			continue
		}
		var httpEquiv, content string
		for _, a := range tok.Attr {
			switch strings.ToLower(a.Key) {
			case "charset":
				return strings.TrimSpace(a.Val)
			case "http-equiv":
				httpEquiv = a.Val
			case "content":
				content = a.Val
			}
		}
		if strings.EqualFold(strings.TrimSpace(httpEquiv), "content-type") {
			if i := strings.Index(strings.ToLower(content), "charset="); i >= 0 {
				v := strings.Trim(strings.TrimSpace(content[i+len("charset="):]), `"'`)
				if j := strings.IndexAny(v, "; \t"); j >= 0 {
					v = v[:j]
				}
				return v
			}
		}
	}
}
