package textutil

import "testing"

// The expected strings are what TextDecoder("utf-8") returns for the same bytes (WHATWG
// maximal-subpart replacement), so both editions show damage identically.
func TestDecodeUTF8MatchesWHATWG(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"a\xffb", "a\U0000FFFDb"},
		{"\xff\xfe", "\U0000FFFD\U0000FFFD"},
		{"a\xe2\x82", "a\U0000FFFD"},                       // truncated 3-byte sequence: one replacement
		{"\xe2\x82a", "\U0000FFFDa"},                       // cut short by an ASCII byte, which survives
		{"\xed\xa0\x80", "\U0000FFFD\U0000FFFD\U0000FFFD"}, // surrogate: ED only accepts 80-9F next
		{"\xf0\x9f\x98\x80", "\U0001F600"},
		{"\xc0\xaf", "\U0000FFFD\U0000FFFD"}, // overlong
	}
	for _, c := range cases {
		if got := DecodeUTF8([]byte(c.in)); got != c.want {
			t.Errorf("DecodeUTF8(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMeasureUTF8(t *testing.T) {
	d := MeasureUTF8([]byte("Привет\xd1"))
	if d.Multi != 6 || d.Errors != 1 || d.InvalidBytes != 1 || !d.TruncatedTail {
		t.Errorf("truncated tail: got %+v", d)
	}
	d = MeasureUTF8([]byte("Пр\xffивет"))
	if d.Multi != 6 || d.Errors != 1 || d.TruncatedTail {
		t.Errorf("mid-text damage: got %+v", d)
	}
	d = MeasureUTF8([]byte("\xcf\xf0\xe8\xe2\xe5\xf2")) // "Привет" in cp1251
	if d.Multi != 0 || d.Errors == 0 {
		t.Errorf("cp1251 bytes: got %+v", d)
	}
}

func TestLookupCodec(t *testing.T) {
	c, ok := LookupCodec("cp1251")
	if !ok || c.Name() != "windows-1251" {
		t.Fatalf("cp1251: ok=%v", ok)
	}
	if got := c.Decode([]byte{0xcf, 0xf0, 0x98}); got != "Пр\u0098" {
		t.Errorf("windows-1251 decode = %q (0x98 must be the WHATWG C1 control)", got)
	}
	latin, ok := LookupCodec("iso-8859-1")
	if !ok || latin.Name() != "windows-1252" {
		t.Fatalf("iso-8859-1 must resolve to windows-1252 as in the browser")
	}
	if got := latin.Decode([]byte("caf\xe9 \x80 \x81")); got != "café € \u0081" {
		t.Errorf("windows-1252 decode = %q", got)
	}
	for _, l := range []string{"utf-8", "utf-16", "no-such-charset"} {
		if _, ok := LookupCodec(l); ok {
			t.Errorf("LookupCodec(%q) should be refused", l)
		}
	}
}
