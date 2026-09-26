package epub

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

type contentFidelityCases struct {
	Charset []struct {
		Name   string `json:"name"`
		Reader string `json:"reader"`
		Bytes  string `json:"bytes"`
		Want   string `json:"want"`
	} `json:"charset"`
	XHTML []struct {
		Name string `json:"name"`
		In   string `json:"in"`
		Want string `json:"want"`
	} `json:"xhtml"`
	Cover []struct {
		Name  string `json:"name"`
		Body  string `json:"body"`
		Cover bool   `json:"cover"`
		Text  string `json:"text"`
	} `json:"cover"`
}

// TestContentFidelitySharedCases runs the chapter half of the fixture extension/test/epub-parity.test.mjs
// runs, so a chapter decodes, parses and keeps its cover text alike in both editions (audit finding
// B46; the HTML-input charset cases run in internal/htmlconv).
func TestContentFidelitySharedCases(t *testing.T) {
	var fx contentFidelityCases
	if err := json.Unmarshal(sharedFixture(t, "content_fidelity_cases.json"), &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Charset {
		if c.Reader != "chapter" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(c.Bytes)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		out, _, err := decodeToUTF8(raw)
		if err != nil || !strings.Contains(string(out), c.Want) {
			t.Errorf("charset %s: decoded %q (err %v), want it to hold %q", c.Name, out, err, c.Want)
		}
	}
	for _, c := range fx.XHTML {
		if got := string(xhtmlToHTMLSyntax([]byte(c.In))); got != c.Want {
			t.Errorf("xhtml %s:\n got %q\nwant %q", c.Name, got, c.Want)
		}
	}
	for _, c := range fx.Cover {
		doc, err := gohtml.Parse(strings.NewReader("<html><body>" + c.Body + "</body></html>"))
		if err != nil {
			t.Fatal(err)
		}
		rewriteCoverSVGs(doc)
		var buf bytes.Buffer
		if err := gohtml.Render(&buf, doc); err != nil {
			t.Fatal(err)
		}
		html := buf.String()
		isCover := !strings.Contains(html, "<svg") && strings.Contains(html, "<img")
		if isCover != c.Cover {
			t.Errorf("cover %s: became an <img> = %v, want %v:\n%s", c.Name, isCover, c.Cover, html)
		}
		if c.Text != "" && !strings.Contains(html, c.Text) {
			t.Errorf("cover %s: lost %q:\n%s", c.Name, c.Text, html)
		}
	}
}
