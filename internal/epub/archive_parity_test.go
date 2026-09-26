package epub

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sharedFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"..", "..", "tests", "testdata"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestContainerSharedCases runs the container fixture extension/test/epub.test.mjs runs through
// its parseContainer, so both editions open the same package document or refuse the same book
// (audit finding B48).
func TestContainerSharedCases(t *testing.T) {
	var fx struct {
		Cases []struct {
			Name      string              `json:"name"`
			RootFiles []map[string]string `json:"rootfiles"`
			Want      *string             `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(sharedFixture(t, "epub_container_cases.json"), &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles>`)
			for _, rf := range c.RootFiles {
				sb.WriteString("<rootfile")
				for _, k := range []string{"full-path", "media-type"} {
					if v, ok := rf[k]; ok {
						sb.WriteString(" " + k + `="` + html.EscapeString(v) + `"`)
					}
				}
				sb.WriteString("/>")
			}
			sb.WriteString("</rootfiles></container>")
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "META-INF"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "META-INF", "container.xml"), []byte(sb.String()), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := parseContainer(dir)
			switch {
			case c.Want == nil && err == nil:
				t.Errorf("parseContainer = %q, want a refusal", got)
			case c.Want != nil && (err != nil || got != *c.Want):
				t.Errorf("parseContainer = %q, %v; want %q", got, err, *c.Want)
			}
		})
	}
}

// TestArchiveParityEPUB extracts the shared EPUB fixtures extension/test/epub.test.mjs unzips: a
// symlink entry is never unpacked, and the chapter the spine named through it is dropped (audit
// finding E39; docs/PARITY.md, "Input limits").
func TestArchiveParityEPUB(t *testing.T) {
	var fx struct {
		EPUBs []struct {
			File    string `json:"file"`
			Kept    string `json:"kept"`
			Skipped string `json:"skipped"`
			About   string `json:"about"`
		} `json:"epubs"`
	}
	if err := json.Unmarshal(sharedFixture(t, "archive-parity", "cases.json"), &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.EPUBs {
		t.Run(c.File, func(t *testing.T) {
			out := t.TempDir()
			book, err := Extract(filepath.Join("..", "..", "tests", "testdata", "archive-parity", c.File), out)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(out, filepath.FromSlash(c.Skipped))); !os.IsNotExist(err) {
				t.Errorf("%s: %s was unpacked (lstat err %v)", c.About, c.Skipped, err)
			}
			if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(c.Kept))); err != nil {
				t.Errorf("%s: %s missing: %v", c.About, c.Kept, err)
			}
			// dropMissingContent takes the chapter out of the manifest; a spine idref left without
			// an item is skipped by every later stage.
			var chapters []string
			for _, it := range book.Manifest {
				if isHTMLMediaType(it.MediaType) {
					chapters = append(chapters, it.Href)
				}
			}
			if len(chapters) != 1 {
				t.Errorf("%s: chapters %q, want only the real one", c.About, chapters)
			}
		})
	}
}
