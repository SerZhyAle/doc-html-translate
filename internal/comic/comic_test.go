package comic

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestNaturalLessOrdering(t *testing.T) {
	in := []string{"page10.jpg", "page2.jpg", "page1.jpg", "page100.jpg", "cover.png", "page02.jpg"}
	// Numeric-aware order; equal value ("2" vs "02") breaks toward the shorter raw run.
	want := []string{"cover.png", "page1.jpg", "page2.jpg", "page02.jpg", "page10.jpg", "page100.jpg"}
	got := append([]string(nil), in...)
	sort.SliceStable(got, func(i, j int) bool { return naturalLess(got[i], got[j]) })
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("natural order wrong:\n got  %v\n want %v", got, want)
		}
	}
}

func TestNaturalLessNumericValue(t *testing.T) {
	// The whole point: 2 must precede 10, which lexicographic sort gets wrong.
	if !naturalLess("p2", "p10") {
		t.Error("p2 should sort before p10 (numeric-aware)")
	}
	if naturalLess("p10", "p2") {
		t.Error("p10 should not sort before p2")
	}
	// Equal value, different leading zeros: fewer leading zeros (shorter raw run) first.
	if !naturalLess("7", "07") {
		t.Error("7 should sort before 07 (fewer leading zeros)")
	}
	if naturalLess("07", "7") {
		t.Error("07 should not sort before 7")
	}
}

func TestIsPageEntry(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"page01.jpg", true},
		{"deep/dir/page01.JPG", true},
		{"art.PNG", true},
		{"scan.webp", true},
		{"ComicInfo.xml", false},
		{"Thumbs.db", false},
		{"folder/", false},
		{"__MACOSX/page01.jpg", false},
		{"sub/__MACOSX/._page.jpg", false},
		{".DS_Store", false},
		{"dir/._page01.jpg", false},
		{"notes.txt", false},
		{"scan.tif", false}, // TIFF deliberately excluded
	}
	for _, c := range cases {
		if got := isPageEntry(c.name); got != c.want {
			t.Errorf("isPageEntry(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

// makeCBZ builds a ZIP in a temp file from name->bytes and returns its path.
func makeCBZ(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.cbz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, werr := zw.Create(name)
		if werr != nil {
			t.Fatal(werr)
		}
		if _, werr := w.Write(data); werr != nil {
			t.Fatal(werr)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func makeCBT(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.cbt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	for name, data := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// entries used by the extraction tests: three pages out of order, plus non-page
// cruft that must be filtered.
func sampleEntries() map[string][]byte {
	return map[string][]byte{
		"page10.jpg":       []byte("TEN"),
		"page2.jpg":        []byte("TWO"),
		"page1.jpg":        []byte("ONE"),
		"ComicInfo.xml":    []byte("<meta/>"),
		"Thumbs.db":        []byte("junk"),
		"__MACOSX/x.jpg":   []byte("resource"),
		"notes.txt":        []byte("ignore me"),
	}
}

func assertExtracted(t *testing.T, comicPath string) {
	t.Helper()
	out := t.TempDir()
	book, err := Extract(comicPath, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(book.Spine))
	}
	// Natural order: page1 -> page_001, page2 -> page_002, page10 -> page_003.
	wantData := map[string]string{
		"page_001.jpg": "ONE",
		"page_002.jpg": "TWO",
		"page_003.jpg": "TEN",
	}
	for name, want := range wantData {
		got, rerr := os.ReadFile(filepath.Join(out, name))
		if rerr != nil {
			t.Fatalf("read %s: %v", name, rerr)
		}
		if !bytes.Equal(got, []byte(want)) {
			t.Errorf("%s = %q, want %q (page ordering wrong)", name, got, want)
		}
	}
	// Each page has an HTML wrapper referencing its image.
	for i, img := range []string{"page_001.jpg", "page_002.jpg", "page_003.jpg"} {
		htmlPath := filepath.Join(out, book.Spine[i].IDRef+".html")
		h, rerr := os.ReadFile(htmlPath)
		if rerr != nil {
			t.Fatalf("read page html %d: %v", i+1, rerr)
		}
		if !bytes.Contains(h, []byte(`src="`+img+`"`)) {
			t.Errorf("page %d html does not reference %s", i+1, img)
		}
	}
}

func TestExtractCBZ(t *testing.T) { assertExtracted(t, makeCBZ(t, sampleEntries())) }
func TestExtractCBT(t *testing.T) { assertExtracted(t, makeCBT(t, sampleEntries())) }

func TestExtractNoPages(t *testing.T) {
	path := makeCBZ(t, map[string][]byte{"ComicInfo.xml": []byte("<x/>"), "readme.txt": []byte("hi")})
	if _, err := Extract(path, t.TempDir()); err == nil {
		t.Fatal("expected error when archive has no page images")
	}
}
