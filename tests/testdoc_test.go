package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/pipeline"
)

// testDocDir returns the directory holding the local sample documents.
// It is gitignored (see .gitignore: test_doc/), so it is present only on
// developer machines. Override with DOC_HTML_TEST_DOC for a custom location.
func testDocDir(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("DOC_HTML_TEST_DOC"); env != "" {
		return env
	}
	// `go test` runs with the working directory set to this package dir
	// (<repo>/tests), so the sample folder is one level up.
	return filepath.Join("..", "test_doc")
}

// supportedExts are the input formats the pipeline knows how to convert.
// Anything else in test_doc (e.g. .docx, .jpg) is ignored by these tests.
var supportedExts = map[string]bool{
	".epub": true,
	".pdf":  true,
	".txt":  true,
	".md":   true,
	".fb2":  true,
	".rtf":  true,
	".html": true,
	".htm":  true,
	".mobi": true,
	".azw3": true,
}

// TestConvertTestDoc converts every supported sample in test_doc/ through the
// full pipeline (extract + build HTML, no translation, no browser) and asserts
// that a valid index.html is produced. Each file is a subtest so failures are
// isolated and named. The whole test skips cleanly when test_doc/ is absent,
// which is the normal state in CI (the folder is gitignored).
func TestConvertTestDoc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test_doc conversion in short mode")
	}

	dir := testDocDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("sample directory %q not present - skipping (set DOC_HTML_TEST_DOC to override)", dir)
		}
		t.Fatalf("read sample dir %q: %v", dir, err)
	}

	var converted int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !supportedExts[ext] {
			continue
		}

		name := entry.Name()
		inputPath := filepath.Join(dir, name)

		t.Run(name, func(t *testing.T) {
			// MOBI/AZW3 extraction shells out to Calibre's ebook-convert.
			if ext == ".mobi" || ext == ".azw3" {
				if _, err := exec.LookPath("ebook-convert"); err != nil {
					t.Skipf("ebook-convert (Calibre) not on PATH - skipping %s", name)
				}
			}

			// Output goes to a throwaway temp dir so the sample folder stays
			// clean. On success it is deleted; on failure it is kept (and its
			// path logged) so the broken output can be inspected.
			outFolder, err := os.MkdirTemp("", "doc-html-testdoc-*")
			if err != nil {
				t.Fatalf("create temp output dir: %v", err)
			}
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("kept output for inspection: %s", outFolder)
					return
				}
				_ = os.RemoveAll(outFolder)
			})

			cfg := config.Config{
				InputFile:    inputPath,
				OutputFolder: outFolder,
				NoTranslate:  true,
				NoOpen:       true,
				SourceLang:   "en",
				TargetLang:   "ru",
			}

			exitCode, err := pipeline.NewRunner(cfg).Run()
			if err != nil {
				t.Fatalf("convert %s: exit=%d err=%v", name, exitCode, err)
			}
			if exitCode != pipeline.ExitOK {
				t.Fatalf("convert %s: exit=%d, want %d", name, exitCode, pipeline.ExitOK)
			}

			// The pipeline writes an index.html under OutputFolder (in a
			// sub-dir named after the input file). Locate it without
			// reconstructing the sanitized name, then check it is non-empty.
			indexPath := findIndexHTML(outFolder)
			if indexPath == "" {
				t.Fatalf("no index.html produced for %s under %s", name, outFolder)
			}
			info, err := os.Stat(indexPath)
			if err != nil {
				t.Fatalf("stat %s: %v", indexPath, err)
			}
			if info.Size() == 0 {
				t.Fatalf("index.html for %s is empty", name)
			}
		})
		converted++
	}

	if converted == 0 {
		t.Skipf("no supported samples found in %q - skipping", dir)
	}
}

// findIndexHTML returns the path of the first index.html found anywhere under
// root, or "" if there is none.
func findIndexHTML(root string) string {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "index.html") {
			found = path
		}
		return nil
	})
	return found
}
