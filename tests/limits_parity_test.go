package tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/internal/limits"
)

// TestParityInputLimits: both editions refuse the same archives, so the published limits must
// hold the same values in internal/limits (and the per-format caps in internal/epub and
// internal/comic) and in extension/src/limits.js. See docs/PARITY.md "Input limits".
func TestParityInputLimits(t *testing.T) {
	js := readRepoFile(t, "extension", "src", "limits.js")
	epubGo := readRepoFile(t, "internal", "epub", "epub.go")
	comicGo := readRepoFile(t, "internal", "comic", "extract.go")

	pairs := []struct {
		name string
		goV  int64
		jsV  int64
	}{
		{"archive entry count", limits.MaxArchiveEntries, jsConst(t, js, "ARCHIVE_MAX_ENTRIES")},
		{"archive unpacked total", limits.MaxArchiveTotalBytes, jsConst(t, js, "ARCHIVE_MAX_TOTAL_BYTES")},
		{"EPUB per-file cap", goShiftConst(t, epubGo, "maxEntryBytes"), jsConst(t, js, "EPUB_MAX_ENTRY_BYTES")},
		{"comic per-page cap", goShiftConst(t, comicGo, "maxPageBytes"), jsConst(t, js, "COMIC_MAX_PAGE_BYTES")},
		{"whole-file text input", limits.MaxTextInputBytes, jsConst(t, js, "TEXT_MAX_INPUT_BYTES")},
	}
	for _, p := range pairs {
		if p.goV != p.jsV {
			t.Errorf("%s drift: Go=%d limits.js=%d (must match - see docs/PARITY.md)", p.name, p.goV, p.jsV)
		}
	}

	// The published numbers are pinned too: a change is an owner decision, and README.md and
	// docs/PARITY.md state them.
	if limits.MaxImagePixels != 100_000_000 || limits.MaxImageSide != 32768 ||
		limits.MaxArchiveEntries != 20000 || limits.MaxArchiveTotalBytes != 4<<30 ||
		limits.MaxTextInputBytes != 100<<20 {
		t.Error("a published input limit changed; update README.md, extension/README.md and docs/PARITY.md with it")
	}
}

// jsConst evaluates `export const NAME = a * b * ..;` from limits.js.
func jsConst(t *testing.T, src, name string) int64 {
	t.Helper()
	m := regexp.MustCompile(`export const ` + name + ` = ([0-9_ *]+);`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("limits.js: %s not found", name)
	}
	v := int64(1)
	for _, f := range strings.Split(m[1], "*") {
		n, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(f), "_", ""), 10, 64)
		if err != nil {
			t.Fatalf("limits.js: %s: %v", name, err)
		}
		v *= n
	}
	return v
}

// goShiftConst evaluates `const name = a << b` from a Go source.
func goShiftConst(t *testing.T, src, name string) int64 {
	t.Helper()
	m := regexp.MustCompile(`const ` + name + ` = (\d+) << (\d+)`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("Go: %s not found", name)
	}
	a, _ := strconv.ParseInt(m[1], 10, 64)
	b, _ := strconv.ParseInt(m[2], 10, 64)
	return a << b
}
