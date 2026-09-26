package fb2_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/fb2"
)

// TestFB2SharedCase converts the FB2 book extension/test/fb2.test.mjs parses, so both editions show
// the cover first, keep a stanza's own title and subtitle, and leave a visible note for a picture
// with no binary (audit findings B34, B35).
func TestFB2SharedCase(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "fb2_parity_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		XML         string   `json:"xml"`
		FirstText   string   `json:"firstText"`
		Order       []string `json:"order"`
		Subtitle    string   `json:"subtitle"`
		Placeholder string   `json:"placeholder"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "poems.fb2")
	if err := os.WriteFile(src, []byte(fx.XML), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := fb2.Extract(src, out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "page_001.html"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(page)
	body = body[strings.Index(body, "<body>"):]

	if img, first := strings.Index(body, "<img"), strings.Index(body, fx.FirstText); img < 0 || img > first {
		t.Errorf("the cover does not open the book:\n%s", body)
	}
	at := 0
	for _, text := range fx.Order {
		i := strings.Index(body[at:], text)
		if i < 0 {
			t.Fatalf("%q missing or out of order:\n%s", text, body)
		}
		at += i + len(text)
	}
	if !strings.Contains(body, `class="subtitle">`+fx.Subtitle) {
		t.Errorf("%q is not a subtitle paragraph:\n%s", fx.Subtitle, body)
	}
	if !strings.Contains(body, fx.Placeholder) {
		t.Errorf("no placeholder %q:\n%s", fx.Placeholder, body)
	}
}
