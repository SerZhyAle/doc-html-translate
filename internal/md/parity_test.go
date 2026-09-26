package md

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

var (
	leadingHeading = regexp.MustCompile(`^\s*<h[12][^>]*>([\s\S]*?)</h[12]>`)
	anyTag         = regexp.MustCompile(`<[^>]*>`)
)

// TestSectionsSharedCases splits the Markdown the extension's md.js splits, so a document pages at
// the same headings in both editions (audit finding E26).
func TestSectionsSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "md_section_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Name     string   `json:"name"`
			MD       string   `json:"md"`
			Sections []string `json:"sections"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		var buf bytes.Buffer
		if err := goldmark.Convert([]byte(c.MD), &buf); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, s := range splitBySections(buf.String()) {
			label := ""
			if m := leadingHeading.FindStringSubmatch(s); m != nil {
				label = strings.Join(strings.Fields(anyTag.ReplaceAllString(m[1], "")), " ")
			}
			got = append(got, label)
		}
		if !reflect.DeepEqual(got, c.Sections) {
			t.Errorf("%s: sections %q, want %q\n%s", c.Name, got, c.Sections, buf.String())
		}
	}
}
