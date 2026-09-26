package tests

// Reader-chrome values docs/PARITY.md states and the code holds. Kept out of parity_test.go, which is
// past the file budget; same package, same readRepoFile helper.

import (
	"regexp"
	"strings"
	"testing"
)

// fontFamilies reads the serif/sans/mono entries of a `FAMILIES = { .. }` literal, with spaces and
// double quotes dropped: the two editions quote and space the same stacks differently, and neither
// changes what the browser picks.
func fontFamilies(t *testing.T, src, where string) map[string]string {
	t.Helper()
	block := between(src, "FAMILIES = {", "}")
	if block == "" {
		t.Fatalf("%s: no FAMILIES = { .. } literal", where)
	}
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`(serif|sans|mono):\s*'([^']*)'`).FindAllStringSubmatch(block, -1) {
		out[m[1]] = strings.NewReplacer(" ", "", `"`, "").Replace(m[2])
	}
	if len(out) != 3 {
		t.Fatalf("%s: FAMILIES parsed to %v, want serif, sans and mono", where, out)
	}
	return out
}

// TestParityReaderFonts: the reader's three font stacks are the same on both sides, and the values
// docs/PARITY.md "Reader fonts" prints are the ones the code holds (audit finding B42 found the
// section citing viewer lines that had moved). See docs/PARITY.md "Reader fonts".
func TestParityReaderFonts(t *testing.T) {
	goF := fontFamilies(t, readRepoFile(t, "internal", "htmlgen", "navbar.go"), "navbar.go")
	jsF := fontFamilies(t, readRepoFile(t, "extension", "src", "viewer.js"), "viewer.js")
	section := between(readRepoFile(t, "docs", "PARITY.md"), "### Reader fonts", "\n### ")
	for _, k := range []string{"serif", "sans", "mono"} {
		if goF[k] != jsF[k] {
			t.Errorf("reader font %q drift: navbar.go %q, viewer.js %q", k, goF[k], jsF[k])
		}
		doc := regexp.MustCompile("`" + k + "` = `([^`]*)`").FindStringSubmatch(section)
		if doc == nil {
			t.Errorf("docs/PARITY.md Reader fonts: no `%s` = `..` entry", k)
			continue
		}
		if got := strings.NewReplacer(" ", "", `"`, "").Replace(doc[1]); got != goF[k] {
			t.Errorf("docs/PARITY.md Reader fonts %q = %q, code has %q", k, got, goF[k])
		}
	}
}

// TestParityDocChunkConstants: the chunked PDF render is extension-only, so there is no Go twin to
// compare with - but docs/PARITY.md quotes its two numbers, and they had drifted from the code
// (audit finding B42: the doc said 50 and 2, the viewer held 100 and 5).
func TestParityDocChunkConstants(t *testing.T) {
	viewer := readRepoFile(t, "extension", "src", "viewer.js")
	doc := readRepoFile(t, "docs", "PARITY.md")
	for _, name := range []string{"PAGE_CHUNK", "CHUNK_LEAD"} {
		code := regexp.MustCompile(`const ` + name + `\s*=\s*(\d+)`).FindStringSubmatch(viewer)
		stated := regexp.MustCompile("`" + name + ` = (\d+)` + "`").FindStringSubmatch(doc)
		if code == nil || stated == nil {
			t.Fatalf("%s: viewer.js=%v PARITY.md=%v (not found)", name, code, stated)
		}
		if code[1] != stated[1] {
			t.Errorf("docs/PARITY.md says %s = %s, viewer.js has %s", name, stated[1], code[1])
		}
	}
}
