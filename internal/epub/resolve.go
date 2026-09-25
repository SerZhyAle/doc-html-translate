package epub

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"
)

// resolveBookPath is the single gate through which a name the book supplies
// (container full-path, manifest href) becomes a path. The rule, pinned for
// both editions in docs/PARITY.md "EPUB href resolution" and exercised by the
// shared fixture tests/testdata/epub_href_cases.json:
//
//  1. cut ?query and #fragment off the raw value, before decoding, so an
//     encoded %23 stays part of the name;
//  2. percent-decode once (a malformed or non-UTF-8 escape keeps the raw text);
//  3. treat "\" as "/", as Windows and browsers do;
//  4. refuse what Windows would read as leaving the tree or aliasing a device:
//     a colon (drive letter, scheme, alternate data stream), a leading "//"
//     (UNC, protocol-relative), control characters, a segment ending in a dot
//     or space (Windows strips them, so "..." or ".. " could become ".."), and
//     reserved device names;
//  5. a leading "/" is root-relative (the book root), anything else resolves
//     against baseDir;
//  6. the cleaned result must name something strictly inside the book root.
//
// baseDir is a clean root-relative slash path ("" or "." for the root).
// Returns the root-relative slash path.
func resolveBookPath(baseDir, raw string) (string, error) {
	p := raw
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if u, err := url.PathUnescape(p); err == nil && utf8.ValidString(u) {
		p = u
	}
	p = strings.ReplaceAll(p, `\`, "/")
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.HasPrefix(p, "//") {
		return "", fmt.Errorf("network path")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("control character")
		}
		if r == ':' {
			return "", fmt.Errorf("drive letter, scheme or stream")
		}
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			continue
		}
		if strings.HasSuffix(seg, ".") || strings.HasSuffix(seg, " ") {
			return "", fmt.Errorf("segment %q ends in a dot or space", seg)
		}
		if isWindowsDeviceName(seg) {
			return "", fmt.Errorf("reserved device name %q", seg)
		}
	}

	var joined string
	if strings.HasPrefix(p, "/") {
		joined = path.Clean(p[1:])
	} else {
		joined = path.Clean(path.Join(baseDir, p))
	}
	if joined == "." || joined == ".." || strings.HasPrefix(joined, "../") {
		return "", fmt.Errorf("outside the book")
	}
	return joined, nil
}

// isWindowsDeviceName reports whether seg names a DOS device (CON, NUL, COM1..),
// with or without an extension - Windows opens the device, not a file.
func isWindowsDeviceName(seg string) bool {
	stem := seg
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	stem = strings.ToUpper(strings.TrimRight(stem, " "))
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) {
		return stem[3] >= '1' && stem[3] <= '9'
	}
	return false
}

// relToBase expresses a root-relative path relative to baseDir, the form
// ManifestItem.Href carries (it may start with "../" and still be in the book).
func relToBase(baseDir, rootRel string) string {
	if baseDir == "" || baseDir == "." {
		return rootRel
	}
	from := strings.Split(baseDir, "/")
	to := strings.Split(rootRel, "/")
	common := 0
	for common < len(from) && common < len(to)-1 && from[common] == to[common] {
		common++
	}
	return strings.Repeat("../", len(from)-common) + strings.Join(to[common:], "/")
}

// URLPath escapes a book path (a decoded ManifestItem.Href, possibly prefixed
// with BasePath) for use as a relative URL in generated HTML. Only the bytes
// that would change how the URL parses are escaped; letters in any script stay
// readable, and a plain ASCII name comes out unchanged.
func URLPath(p string) string {
	const special = `%#?"<>\^` + "`{|}"
	if !strings.ContainsAny(p, special) && !strings.ContainsFunc(p, func(r rune) bool { return r <= 0x20 || r == 0x7f }) {
		return p
	}
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c <= 0x20 || c == 0x7f || strings.IndexByte(special, c) >= 0 {
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
