package htmlgen

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

// Tests for ticket hotfix-epub-href-containment: a book-supplied name must never
// make the converter read, rewrite or delete a file outside the book.

const victimText = "VICTIM-FILE-MUST-SURVIVE"

func writeEPUB(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, body := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func container(fullPath string) string {
	return fmt.Sprintf(`<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path=%q media-type="application/oebps-package+xml"/></rootfiles>
</container>`, fullPath)
}

// opf builds a package document whose spine lists every manifest item in order.
func opf(hrefs ...string) string {
	var man, spine strings.Builder
	for i, h := range hrefs {
		fmt.Fprintf(&man, `<item id="i%d" href=%q media-type="application/xhtml+xml"/>`+"\n", i, h)
		fmt.Fprintf(&spine, `<itemref idref="i%d"/>`+"\n", i)
	}
	return `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>T</dc:title></metadata>
  <manifest>` + man.String() + `</manifest>
  <spine>` + spine.String() + `</spine>
</package>`
}

func chapter(text string) string {
	return `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>c</title></head><body><p>` + text + `</p></body></html>`
}

// convertBoth extracts the book and runs the single-page merge (which deletes
// what it absorbed) on one copy, and the multipage navbar injection (which
// rewrites every spine file) on another. It returns the merged page.
func convertBoth(t *testing.T, root string, files map[string]string) string {
	t.Helper()
	epubPath := filepath.Join(root, "book.epub")
	writeEPUB(t, epubPath, files)

	single := filepath.Join(root, "single")
	book, err := epub.Extract(epubPath, single)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := GenerateSinglePage(book, single, "book.epub"); err != nil {
		t.Fatalf("single page: %v", err)
	}
	merged, err := os.ReadFile(filepath.Join(single, filepath.FromSlash(book.BasePath), "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	multi := filepath.Join(root, "multi")
	book, err = epub.Extract(epubPath, multi)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if err := InjectNavBars(book, multi, "book.epub"); err != nil {
		t.Fatalf("navbar: %v", err)
	}
	return string(merged)
}

func TestOutOfBookHrefsLeaveOutsideFilesAlone(t *testing.T) {
	vectors := map[string]string{
		"spine parent":   "../../victim.xhtml",
		"encoded parent": "..%2F..%2Fvictim.xhtml",
		"backslash":      `..\..\victim.xhtml`,
		"root-relative":  "/../victim.xhtml",
		"drive letter":   "C:/victim.xhtml",
		"UNC":            `\\server\share\victim.xhtml`,
	}
	for name, href := range vectors {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			victim := filepath.Join(root, "victim.xhtml")
			if err := os.WriteFile(victim, []byte(chapter(victimText)), 0o644); err != nil {
				t.Fatal(err)
			}
			merged := convertBoth(t, root, map[string]string{
				"META-INF/container.xml": container("OEBPS/content.opf"),
				"OEBPS/content.opf":      opf("ch1.xhtml", href),
				"OEBPS/ch1.xhtml":        chapter("real chapter"),
			})
			got, err := os.ReadFile(victim)
			if err != nil {
				t.Fatalf("victim file gone: %v", err)
			}
			if string(got) != chapter(victimText) {
				t.Errorf("victim file rewritten:\n%s", got)
			}
			if strings.Contains(merged, victimText) {
				t.Error("victim content leaked into the merged page")
			}
			if !strings.Contains(merged, "real chapter") {
				t.Error("the in-book chapter was lost")
			}
		})
	}
}

func TestOutOfBookContainerPointerIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "evil.opf"), []byte(opf("x.xhtml")), 0o644); err != nil {
		t.Fatal(err)
	}
	epubPath := filepath.Join(root, "book.epub")
	writeEPUB(t, epubPath, map[string]string{
		"META-INF/container.xml": container("../evil.opf"),
	})
	if _, err := epub.Extract(epubPath, filepath.Join(root, "out")); err == nil {
		t.Fatal("an out-of-book rootfile must fail the book")
	}
}

func TestPercentEncodedChaptersConvert(t *testing.T) {
	root := t.TempDir()
	merged := convertBoth(t, root, map[string]string{
		"META-INF/container.xml": container("OEBPS/content.opf"),
		"OEBPS/content.opf":      opf("Chapter%201.xhtml", "Chapter%202.xhtml"),
		"OEBPS/Chapter 1.xhtml":  chapter("first chapter"),
		"OEBPS/Chapter 2.xhtml":  chapter("second chapter"),
	})
	for _, want := range []string{"first chapter", "second chapter"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged page lacks %q", want)
		}
	}
	nav, err := os.ReadFile(filepath.Join(root, "multi", "OEBPS", "Chapter 1.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nav), `href="Chapter%202.html"`) {
		t.Error("next link to the encoded chapter is not escaped")
	}
}

func TestMissingManifestItemSkipsOnlyThatItem(t *testing.T) {
	root := t.TempDir()
	merged := convertBoth(t, root, map[string]string{
		"META-INF/container.xml": container("OEBPS/content.opf"),
		"OEBPS/content.opf":      opf("ch1.xhtml", "gone.xhtml", "ch3.xhtml"),
		"OEBPS/ch1.xhtml":        chapter("chapter one"),
		"OEBPS/ch3.xhtml":        chapter("chapter three"),
	})
	for _, want := range []string{"chapter one", "chapter three"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged page lacks %q", want)
		}
	}
}

func TestIndexNamedChapterIsKept(t *testing.T) {
	for _, base := range []string{"", "OEBPS/"} {
		t.Run("base="+base, func(t *testing.T) {
			root := t.TempDir()
			merged := convertBoth(t, root, map[string]string{
				"META-INF/container.xml": container(base + "content.opf"),
				base + "content.opf":     opf("Index.xhtml", "ch2.xhtml"),
				base + "Index.xhtml":     chapter("index chapter"),
				base + "ch2.xhtml":       chapter("second chapter"),
			})
			for _, want := range []string{"index chapter", "second chapter"} {
				if !strings.Contains(merged, want) {
					t.Errorf("merged page lacks %q", want)
				}
			}
			// The merged page must survive the removal of the chapters it absorbed.
			if _, err := os.Stat(filepath.Join(root, "single", filepath.FromSlash(base), "index.html")); err != nil {
				t.Errorf("merged page removed: %v", err)
			}
		})
	}
}
