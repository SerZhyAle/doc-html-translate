package ocr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestIsTranslatableSharedCases runs the fixture extension/test/ocr-text.test.mjs runs, so the two
// editions keep and drop the same OCR text (audit finding B38: the extension's CJK class was a set
// of BMP ranges).
func TestIsTranslatableSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "ocr_translatable_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Text  string `json:"text"`
			Want  bool   `json:"want"`
			About string `json:"about"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		if got := isTranslatable(c.Text); got != c.Want {
			t.Errorf("isTranslatable(%q) = %v, want %v (%s)", c.Text, got, c.Want, c.About)
		}
	}
}
