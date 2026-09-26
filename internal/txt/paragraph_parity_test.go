package txt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParagraphsSharedCases runs the fixture extension/test/txt.test.mjs runs through splitParagraphs,
// so the two editions split the same text into the same paragraphs (docs/PARITY.md, Plain-text
// paragraphs). The form feed case is audit finding X34: the extension read it as text.
func TestParagraphsSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "txt_paragraph_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Name string   `json:"name"`
			In   string   `json:"in"`
			Want []string `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		if got := parseParagraphs([]byte(c.In)); !reflect.DeepEqual(got, c.Want) {
			t.Errorf("%s: parseParagraphs(%q) = %q, want %q", c.Name, c.In, got, c.Want)
		}
	}
}
