package tests

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fb2"
	"doc-html-translate/internal/htmlconv"
	"doc-html-translate/internal/limits"
	"doc-html-translate/internal/md"
	"doc-html-translate/internal/rtf"
	"doc-html-translate/internal/txt"
)

// E27: every whole-file text input is refused from its size on disk, before it is read, when
// it is over limits.MaxTextInputBytes. The file is sparse, so the test allocates nothing and
// an extractor that reads it anyway shows up as a missing refusal, not as a slow test.
func TestTextInputsRefuseOversizedFiles(t *testing.T) {
	extractors := []struct {
		ext     string
		extract func(string, string) (*epub.Book, error)
	}{
		{".txt", txt.Extract},
		{".md", md.Extract},
		{".fb2", fb2.Extract},
		{".rtf", rtf.Extract},
		{".html", htmlconv.Extract},
	}
	for _, x := range extractors {
		t.Run(x.ext, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "huge"+x.ext)
			f, err := os.Create(src)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Truncate(limits.MaxTextInputBytes + 1); err != nil {
				t.Fatal(err)
			}
			_ = f.Close()
			out := filepath.Join(dir, "out")
			if err := os.MkdirAll(out, 0o755); err != nil {
				t.Fatal(err)
			}
			_, err = x.extract(src, out)
			if !errors.Is(err, limits.ErrTooLarge) {
				t.Errorf("Extract(%s over the budget) = %v, want a size-limit refusal", x.ext, err)
			}
		})
	}
}
