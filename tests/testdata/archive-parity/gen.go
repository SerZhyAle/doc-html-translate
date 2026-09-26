//go:build ignore

// gen writes the shared archive fixtures: comic and EPUB containers built the way real tools
// build them, including the shapes the two editions used to list differently (audit findings
// B30, B31, B32, E39). Run from the repository root:
//
//	go run tests/testdata/archive-parity/gen.go
//
// The expected page lists in cases.json are written by hand, never computed by the readers under
// test. internal/comic, internal/epub and extension/test/{comic,epub}.test.mjs read the same files.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

const dir = "tests/testdata/archive-parity"

func main() {
	// B30: a symlink among the pages. Go's zip reader reports it as a link, not a regular file.
	must(write("symlink.cbz", zipOf(
		file("page1.jpg", "ONE", 0o644),
		file("page2.jpg", "page1.jpg", fs.ModeSymlink|0o777),
		file("page3.jpg", "THREE", 0o644),
	)))

	// B31: one entry named three ways - a PAX path=, a GNU long name, and the header's own
	// name. Go's archive/tar lets the GNU long name win.
	var tar bytes.Buffer
	tarRecord(&tar, "././@PaxHeader", 'x', paxRecord("path", "pax/page2.jpg"))
	tarRecord(&tar, "././@LongLink", 'L', []byte("gnu/page2.jpg\x00"))
	tarRecord(&tar, "short.jpg", '0', []byte("TWO"))
	tarRecord(&tar, "page1.jpg", '0', []byte("ONE"))
	tar.Write(make([]byte, 1024))
	must(write("names.cbt", tar.Bytes()))

	// B32: a ZIP behind a stub, the shape of a self-extracting archive saved as .cbz. No
	// signature at offset 0, so only the extension says it is a ZIP.
	stub := append([]byte("MZ"), bytes.Repeat([]byte{0x90}, 510)...)
	must(write("prefixed.cbz", append(stub, zipOf(
		file("page2.jpg", "TWO", 0o644),
		file("page1.jpg", "ONE", 0o644),
	)...)))

	// E39: an EPUB whose spine names a symlink. Neither edition may unpack it as a chapter.
	must(write("symlink.epub", zipOf(
		file("mimetype", "application/epub+zip", 0o644),
		file("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`, 0o644),
		file("OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Links</dc:title></metadata>
  <manifest>
    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="link.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`, 0o644),
		file("OEBPS/ch1.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head><body><p>Chapter one.</p></body></html>`, 0o644),
		file("OEBPS/link.xhtml", "../../../etc/hostname", fs.ModeSymlink|0o777),
	)))
}

type entry struct {
	name, data string
	mode       fs.FileMode
}

func file(name, data string, mode fs.FileMode) entry { return entry{name, data, mode} }

func zipOf(entries ...entry) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Store}
		h.SetMode(e.mode)
		f, err := w.CreateHeader(h)
		must(err)
		_, err = f.Write([]byte(e.data))
		must(err)
	}
	must(w.Close())
	return buf.Bytes()
}

// tarRecord writes one ustar header block (with a valid checksum, which Go's reader checks) and
// the data padded to 512 bytes.
func tarRecord(buf *bytes.Buffer, name string, typeflag byte, data []byte) {
	h := make([]byte, 512)
	copy(h[0:100], name)
	copy(h[100:108], "0000644\x00")
	copy(h[108:116], "0000000\x00")
	copy(h[116:124], "0000000\x00")
	copy(h[124:136], fmt.Sprintf("%011o\x00", len(data)))
	copy(h[136:148], "00000000000\x00")
	h[156] = typeflag
	copy(h[257:265], "ustar\x0000")
	for i := 148; i < 156; i++ {
		h[i] = ' '
	}
	sum := 0
	for _, b := range h {
		sum += int(b)
	}
	copy(h[148:156], fmt.Sprintf("%06o\x00 ", sum))
	buf.Write(h)
	buf.Write(data)
	if pad := (512 - len(data)%512) % 512; pad > 0 {
		buf.Write(make([]byte, pad))
	}
}

// paxRecord encodes "<len> key=value\n", where len counts itself.
func paxRecord(key, value string) []byte {
	body := " " + key + "=" + value + "\n"
	n := len(body)
	for {
		s := strconv.Itoa(n) + body
		if len(s) == n {
			return []byte(s)
		}
		n = len(s)
	}
}

func write(name string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
