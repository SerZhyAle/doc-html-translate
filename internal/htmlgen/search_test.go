package htmlgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchIndexUsesFinalChapterTextAndOCRPlates(t *testing.T) {
	dir := t.TempDir()
	book := writeTextBook(t, dir, 2)
	if err := InjectNavBars(book, dir, "source.epub"); err != nil {
		t.Fatal(err)
	}
	if page := readFile(t, filepath.Join(dir, "ch_001.html")); !strings.Contains(page, `id="dht-search-button"`) || !strings.Contains(page, `id="dht-search-script"`) {
		t.Fatal("chapter has no search control and runtime")
	}
	chapter := filepath.Join(dir, "ch_002.html")
	content := readFile(t, chapter)
	content = strings.Replace(content, "Text.", `<span class="ocr-plate">مرحبا</span> Translated phrase`, 1)
	if err := os.WriteFile(chapter, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteSearchIndex(book, dir); err != nil {
		t.Fatal(err)
	}
	js := readFile(t, filepath.Join(dir, "dht-search-index.js"))
	const prefix = "window.dhtSearchIndex="
	if !strings.HasPrefix(js, prefix) {
		t.Fatal("missing local script assignment")
	}
	var data struct {
		Pages []struct {
			Href  string   `json:"href"`
			Text  []string `json:"text"`
			Plate []bool   `json:"plate"`
		} `json:"pages"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimPrefix(js, prefix), ";\n")), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Pages) != 2 || data.Pages[1].Href != "ch_002.html" {
		t.Fatalf("unexpected pages: %+v", data.Pages)
	}
	joined := strings.Join(data.Pages[1].Text, " ")
	if !strings.Contains(joined, "مرحبا") || !strings.Contains(joined, "Translated phrase") {
		t.Fatalf("final text missing: %q", joined)
	}
	if strings.Contains(joined, "Previous page") || strings.Contains(joined, "Search text") {
		t.Fatalf("reader chrome indexed: %q", joined)
	}
	foundPlate := false
	for i, value := range data.Pages[1].Text {
		if value == "مرحبا" && data.Pages[1].Plate[i] {
			foundPlate = true
		}
	}
	if !foundPlate {
		t.Fatal("OCR text missing plate marker")
	}
}
