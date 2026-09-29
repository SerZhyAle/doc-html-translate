package textutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The cases match extension/test/reflow.test.mjs "normalizeLangTag", so the two editions read a
// declared language the same way.
func TestNormalizeLangTag(t *testing.T) {
	cases := map[string]string{
		"en-US":   "en-US",
		"RU":      "ru",
		"fr_FR":   "fr-FR",
		" ru ":    "ru",
		"":        "",
		"zh-Hans": "zh",
		"russian": "",
		"uk-UA-x": "uk-UA",
		"123":     "",
	}
	for in, want := range cases {
		if got := NormalizeLangTag(in); got != want {
			t.Errorf("NormalizeLangTag(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDominantScript pins the block counter the declaration guard reads, including the two
// shapes the fixture cannot reach: no letters at all, and the ASCII-only Latin definition.
func TestDominantScript(t *testing.T) {
	cases := map[string]string{
		"消除贫穷和饥饿": "han",
		"это сказаніе въ старинныхъ книгахъ": "cyrillic",
		"the quick brown fox": "latin",
		"_ [ ] ^ ` 12345":     "unknown",
		"":                    "unknown",
	}
	for in, want := range cases {
		if got, _ := DominantScript(in); got != want {
			t.Errorf("DominantScript(%q) = %q, want %q", in, got, want)
		}
	}
	if _, letters := DominantScript("abc _123 消"); letters != 4 {
		t.Errorf("DominantScript letter count = %d, want 4", letters)
	}
}

// TestDeclarationContradicted runs the shared PDF-language fixture (ticket 76) through the
// guard. The extension runs the same cases through the viewer's whole decision - guard, then
// heuristic - and checks the final <html lang> (extension/test/reflow.test.mjs,
// "pdfDocumentLang: declaration guard, shared Go/JS fixture").
func TestDeclarationContradicted(t *testing.T) {
	var fx struct {
		Cases []struct {
			Name     string `json:"name"`
			Declared string `json:"declared"`
			Text     string `json:"text"`
			Agree    bool   `json:"agree"`
		} `json:"cases"`
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "pdf_lang_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		if got := DeclarationContradicted(c.Declared, c.Text); got != !c.Agree {
			t.Errorf("%s: DeclarationContradicted(%q, sample) = %v, want %v", c.Name, c.Declared, got, !c.Agree)
		}
	}
}
