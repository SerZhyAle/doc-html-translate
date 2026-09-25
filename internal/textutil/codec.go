package textutil

import (
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// Codec decodes one legacy encoding named by a WHATWG Encoding Standard label - the label set
// the browser's TextDecoder accepts - so a label resolves to the same table in both editions.
// Build it once per document and reuse it; a Codec is not safe for concurrent use.
type Codec struct {
	name  string
	enc   encoding.Encoding
	dec   *encoding.Decoder
	table *[256]rune // set for single-byte encodings
}

// LookupCodec resolves label to a Codec. ok is false for an unknown label, and for UTF-8 and
// UTF-16, which callers handle themselves (UTF-16 cannot be named by an ASCII declaration, and
// UTF-8 goes through DecodeUTF8).
func LookupCodec(label string) (*Codec, bool) {
	// htmlindex rather than x/net's charset.Lookup: the latter wraps the encoding, which hides
	// the *charmap.Charmap that marks a single-byte table.
	enc, err := htmlindex.Get(strings.TrimSpace(label))
	if err != nil {
		return nil, false
	}
	name, err := htmlindex.Name(enc)
	if err != nil || name == "utf-8" || strings.HasPrefix(name, "utf-16") {
		return nil, false
	}
	c := &Codec{name: name, enc: enc}
	if cm, ok := enc.(*charmap.Charmap); ok {
		c.table = singleByteTable(cm)
	} else {
		c.dec = enc.NewDecoder()
	}
	return c, true
}

// singleByteTable is the charmap as the browser reads it. Where a code page leaves a byte in
// 0x80-0x9F undefined (0x81 in windows-1252, 0x98 in windows-1251), the WHATWG index maps it to
// the C1 control of the same value and x/text to U+FFFD; the browser's result wins so the two
// editions agree.
func singleByteTable(cm *charmap.Charmap) *[256]rune {
	var t [256]rune
	for b := 0; b < 256; b++ {
		r := cm.DecodeByte(byte(b))
		if r == utf8.RuneError && b >= 0x80 && b <= 0x9F {
			r = rune(b)
		}
		t[b] = r
	}
	return &t
}

// Name is the canonical WHATWG name of the encoding.
func (c *Codec) Name() string { return c.name }

// Decode returns b decoded to valid UTF-8.
func (c *Codec) Decode(b []byte) string {
	if c.table != nil {
		var sb strings.Builder
		sb.Grow(len(b) * 2)
		for _, x := range b {
			sb.WriteRune(c.table[x])
		}
		return sb.String()
	}
	out, err := c.dec.Bytes(b)
	if err != nil {
		// The x/text decoders replace bad input rather than fail; an error here means the
		// transformer itself gave up, and the bytes must still reach the page visibly.
		return DecodeUTF8(b)
	}
	return string(out)
}

// NewReader returns r decoded to UTF-8 as a stream, for input too large to hold twice.
func (c *Codec) NewReader(r io.Reader) io.Reader {
	if c.table != nil {
		return transform.NewReader(r, singleByteDecoder{table: c.table})
	}
	return transform.NewReader(r, c.enc.NewDecoder())
}

// singleByteDecoder streams a single-byte table; it keeps no state between calls.
type singleByteDecoder struct {
	transform.NopResetter
	table *[256]rune
}

func (d singleByteDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		r := d.table[src[nSrc]]
		if nDst+utf8.RuneLen(r) > len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc++
	}
	return nDst, nSrc, nil
}
