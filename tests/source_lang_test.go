package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/config"
)

// htmlRootRe captures the attributes of a page's <html> start tag.
var htmlRootRe = regexp.MustCompile(`<html([^>]*)>`)

// TestConvertedSourceLanguage pins ticket 40 end to end: the entry page declares the language
// the source states and nothing when it states none. A guessed "en" on a Russian book is what
// stops Chrome offering "Translate page", the product's free flow. Both the merged page and the
// multi-page TOC index are covered, since each builds its own <html>.
func TestConvertedSourceLanguage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping conversion in short mode")
	}

	// Enough paragraphs for two FB2 pages and two Markdown chapters, so the multi-page run
	// builds a real TOC index rather than a redirect to its only page.
	russianFB2 := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title>Проверка</book-title><lang>ru</lang></title-info></description>
  <body><section><title><p>Глава первая</p></title>` +
		strings.Repeat("<p>Это обычный русский абзац, достаточно длинный для текста.</p>", 40) +
		`</section></body>
</FictionBook>`
	markdown := "# One\n\nAn ordinary paragraph of body text.\n\n# Two\n\nAnother paragraph of body text.\n"

	cases := []struct {
		name, file, src, wantAttrs string
	}{
		{"fb2", "book.fb2", russianFB2, ` lang="ru"`},
		{"markdown", "book.md", markdown, ""},
	}
	for _, c := range cases {
		for _, single := range []bool{true, false} {
			mode := "multi"
			if single {
				mode = "single"
			}
			t.Run(c.name+"/"+mode, func(t *testing.T) {
				in := filepath.Join(t.TempDir(), c.file)
				if err := os.WriteFile(in, []byte(c.src), 0o644); err != nil {
					t.Fatal(err)
				}
				out := t.TempDir()
				cfg := config.Config{
					InputFile:    in,
					OutputFolder: out,
					NoTranslate:  true,
					NoOpen:       true,
					SinglePage:   single,
					SourceLang:   "en",
					TargetLang:   "ru",
					UILang:       "en",
				}
				if code, err := app.New(cfg).Run(); err != nil {
					t.Fatalf("convert: exit=%d err=%v", code, err)
				}
				indexPath := findIndexHTML(out)
				if indexPath == "" {
					t.Fatalf("no index.html under %s", out)
				}
				raw, err := os.ReadFile(indexPath)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(raw), "<script>location.replace(") {
					t.Fatalf("index.html is a redirect; the fixture should build a TOC index or a merged page")
				}
				m := htmlRootRe.FindStringSubmatch(string(raw))
				if m == nil {
					t.Fatalf("no <html> in %s", indexPath)
				}
				if got := strings.TrimRight(m[1], " "); got != c.wantAttrs {
					t.Errorf("<html%s>, want <html%s>", got, c.wantAttrs)
				}
			})
		}
	}
}
