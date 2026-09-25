package assets

import (
	"reflect"
	"testing"
)

func TestParseSrcset(t *testing.T) {
	cases := map[string][]srcsetCandidate{
		"a.png":                                 {{url: "a.png"}},
		"a.png 1x, b.png 2x":                    {{"a.png", "1x"}, {"b.png", "2x"}},
		"a.png, b.png 480w":                     {{url: "a.png"}, {"b.png", "480w"}},
		"data:image/png;base64,AA 1x, c.png 2x": {{"data:image/png;base64,AA", "1x"}, {"c.png", "2x"}},
		"  x.png   100w  ,  ":                   {{"x.png", "100w"}},
	}
	for in, want := range cases {
		if got := parseSrcset(in); !reflect.DeepEqual(got, want) {
			t.Errorf("parseSrcset(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		ref, path string
		out       outcome
	}{
		{"img/a%20b.png?v=2#x", "img/a b.png", copied},
		{`img\a.png`, "img/a.png", copied},
		{"https://x/a.png", "", kept},
		{"//cdn/a.png", "", kept},
		{"data:image/png;base64,AA", "", kept},
		{"#frag", "", kept},
		{"", "", kept},
		{"/abs/a.png", "", refused},
		{`C:\pics\a.png`, "", refused},
		{`\\server\share\a.png`, "", refused},
		{"file:///etc/a.png", "", refused},
	}
	for _, c := range cases {
		path, out := classify(c.ref)
		if path != c.path || out != c.out {
			t.Errorf("classify(%q) = %q, %v; want %q, %v", c.ref, path, out, c.path, c.out)
		}
	}
}

func TestReservedAndSanitize(t *testing.T) {
	for _, name := range []string{"favicon.ico", "FAVICON.ICO", "index.html", "page_001.html", ".doc-html-translate.json"} {
		if !reserved(name) {
			t.Errorf("%s should be reserved", name)
		}
	}
	if reserved("page_001_2.html") || reserved("pic.png") {
		t.Error("ordinary names reported reserved")
	}
	if got := sanitizeName(".."); got != "file" {
		t.Errorf("sanitizeName(..) = %q", got)
	}
}
