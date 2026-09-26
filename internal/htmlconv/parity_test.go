package htmlconv

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

func sharedFixture(t *testing.T, name string, v any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}

// TestCharsetSharedCases runs the HTML-input half of the content-fidelity fixture, which the
// extension runs through charset.js decodeHtml (audit finding B46, the extension half of E14).
func TestCharsetSharedCases(t *testing.T) {
	var fx struct {
		Charset []struct {
			Name   string `json:"name"`
			Reader string `json:"reader"`
			Bytes  string `json:"bytes"`
			Want   string `json:"want"`
		} `json:"charset"`
	}
	sharedFixture(t, "content_fidelity_cases.json", &fx)
	for _, c := range fx.Charset {
		if c.Reader != "html" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(c.Bytes)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		doc, err := parseDocument(raw)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		if got := textContent(doc); !strings.Contains(got, c.Want) {
			t.Errorf("%s: text %q, want it to hold %q", c.Name, got, c.Want)
		}
	}
}

var langAttr = regexp.MustCompile(` lang="([^"]*)"`)

// TestLangSharedCases: the converted page declares the language the extension's html.js reads from
// the same source (audit finding E28).
func TestLangSharedCases(t *testing.T) {
	var fx struct {
		Cases []struct {
			Name string `json:"name"`
			HTML string `json:"html"`
			Want string `json:"want"`
		} `json:"cases"`
	}
	sharedFixture(t, "html_lang_cases.json", &fx)
	for _, c := range fx.Cases {
		doc, err := gohtml.Parse(strings.NewReader(c.HTML))
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		if m := langAttr.FindStringSubmatch(rootAttrs(doc)); m != nil {
			got = m[1]
		}
		if got != c.Want {
			t.Errorf("%s: lang %q, want %q", c.Name, got, c.Want)
		}
	}
}
