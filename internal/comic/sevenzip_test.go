package comic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"doc-html-translate/internal/limits"
)

// fake7z is a stand-in for the 7-Zip CLI: `l` prints the listing the test prepared, and `x`
// records that it ran and writes each name from the list file as a small page. It is what lets
// the listing-first rule be tested on a machine without 7-Zip, and proves the refusal happens
// before extraction rather than after it.
const fake7z = `#!/bin/sh
cmd=$1; shift
if [ "$cmd" = "l" ]; then cat "$FAKE7Z_LISTING"; exit 0; fi
touch "$FAKE7Z_MARK"
out=""; list=""
for a in "$@"; do
  case "$a" in
    -o*) out="${a#-o}";;
    @*) list="${a#@}";;
  esac
done
while IFS= read -r name; do
  mkdir -p "$out/$(dirname "$name")"
  printf 'PAGE %s' "$name" > "$out/$name"
done < "$list"
`

// withFake7z puts the fake on PATH and returns the path of the marker `x` touches.
func withFake7z(t *testing.T, listing string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the 7-Zip stand-in is a POSIX shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "7z"), []byte(fake7z), 0o755); err != nil {
		t.Fatal(err)
	}
	listPath := filepath.Join(dir, "listing.txt")
	if err := os.WriteFile(listPath, []byte(listing), 0o644); err != nil {
		t.Fatal(err)
	}
	mark := filepath.Join(dir, "extracted")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH")) // the script needs cat, mkdir, touch
	t.Setenv("FAKE7Z_LISTING", listPath)
	t.Setenv("FAKE7Z_MARK", mark)
	return mark
}

// sevenZipListing renders a `7z l -slt` listing for the given entries.
func sevenZipListing(items ...string) string {
	var b strings.Builder
	b.WriteString("7-Zip 16.02\n\nListing archive: x\n\n--\nPath = x\nType = Rar\n\n----------\n")
	for _, it := range items {
		b.WriteString(it)
		b.WriteString("\n")
	}
	return b.String()
}

func item(path string, size int64) string {
	return fmt.Sprintf("Path = %s\nFolder = -\nSize = %d\nPacked Size = 10\nAttributes = A_ -rw-r--r--\n", path, size)
}

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Done criterion 2: a 7z bomb is refused from its listing, and 7-Zip is never asked to unpack
// anything, so the temp drive does not grow at all.
func TestSevenZipBombRefusedFromListing(t *testing.T) {
	var items []string
	for i := range 50 {
		items = append(items, item(fmt.Sprintf("page%02d.jpg", i), 90<<20))
	}
	mark := withFake7z(t, sevenZipListing(items...))
	path := writeFile(t, "bomb.cb7", []byte("7z\xbc\xaf\x27\x1c\x00\x04"))

	_, err := Extract(context.Background(), path, t.TempDir())
	if !errors.Is(err, limits.ErrTooLarge) {
		t.Fatalf("Extract = %v, want the listing refusal", err)
	}
	if _, serr := os.Stat(mark); serr == nil {
		t.Error("7-Zip extraction ran for an archive the listing had already refused")
	}
}

func TestSevenZipUnknownSizeRefused(t *testing.T) {
	mark := withFake7z(t, sevenZipListing("Path = page1.jpg\nFolder = -\nSize = \n"))
	path := writeFile(t, "nosize.cbr", []byte("Rar!\x1a\x07\x01\x00"))
	_, err := Extract(context.Background(), path, t.TempDir())
	if !errors.Is(err, limits.ErrTooLarge) || !strings.Contains(err.Error(), "page1.jpg") {
		t.Fatalf("Extract = %v, want the unchecked-size refusal naming the entry", err)
	}
	if _, serr := os.Stat(mark); serr == nil {
		t.Error("7-Zip extraction ran for an archive whose listing could not be checked")
	}
}

// Done criterion 5: a RAR renamed to .cbz is routed by its signature to 7-Zip and converts.
// Only the pages are unpacked: the symlink and the metadata never reach the list file.
func TestRARNamedCBZConverts(t *testing.T) {
	withFake7z(t, sevenZipListing(
		item("page2.jpg", 10),
		item("page1.jpg", 10),
		item("ComicInfo.xml", 10),
		"Path = link.jpg\nFolder = -\nSize = 8\nAttributes = A_ lrwxrwxrwx\n",
		"Path = sub\nFolder = +\nSize = 0\nAttributes = D_ drwxr-xr-x\n",
	))
	path := writeFile(t, "renamed.cbz", []byte("Rar!\x1a\x07\x01\x00"))
	out := t.TempDir()
	book, err := Extract(context.Background(), path, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != 2 {
		t.Fatalf("pages = %d, want 2", len(book.Spine))
	}
	if got, _ := os.ReadFile(filepath.Join(out, "page_001.jpg")); string(got) != "PAGE page1.jpg" {
		t.Errorf("page_001.jpg = %q, want page1.jpg's bytes", got)
	}
}

// Without 7-Zip, the renamed RAR is refused with a message that says what the file is,
// instead of "not a valid zip file".
func TestRARNamedCBZWithout7Zip(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if find7Zip() != "" {
		t.Skip("7-Zip is installed at a probed location")
	}
	path := writeFile(t, "renamed.cbz", []byte("Rar!\x1a\x07\x01\x00"))
	_, err := Extract(context.Background(), path, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "RAR archive with a .cbz extension") || !strings.Contains(err.Error(), "7-Zip not found") {
		t.Fatalf("Extract = %v, want the accurate 7-Zip notice", err)
	}
}

func TestParseSevenZipListing(t *testing.T) {
	got := parseSevenZipListing(strings.ReplaceAll(sevenZipListing(
		item("a/p1.jpg", 42),
		"Path = l.jpg\nSize = 3\nSymbolic Link = /etc/passwd\n",
		"Path = d\nSize = 0\nAttributes = D\n",
		"Path = u.jpg\nSize = \n",
	), "\n", "\r\n"))
	if len(got) != 4 {
		t.Fatalf("items = %d, want 4 (the archive's own block is not an entry): %+v", len(got), got)
	}
	want := []sevenZipItem{
		{path: "a/p1.jpg", size: 42, sizeKnown: true},
		{path: "l.jpg", size: 3, sizeKnown: true, link: true},
		{path: "d", sizeKnown: true, dir: true},
		{path: "u.jpg"},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
