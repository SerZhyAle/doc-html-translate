package txt

import "bytes"

// LooksBinary decides whether a stretch of a file's leading bytes belong to a binary format
// rather than a text document, and if so names what it looks like. It returns "" for text.
//
// This is the guard on the pipeline's "unknown extension -> treat as plain text" arm. Without
// it, dragging a .docx, .djvu or a comic archive onto the app produced a multi-megabyte
// "document" of raw bytes rendered as prose, reported as success. The extension already routes
// on the byte signature (viewer.js detectFormat) and gets this right; this brings the Go side
// up to it. Keep the two in step - see docs/PARITY.md.
//
// The order matters: a BOM means text even though UTF-16 is full of NUL bytes, so it is tested
// before anything else; then the named signatures; then a NUL byte anywhere in the sampled head
// as the catch-all for binary data with no signature we recognize. head should be the first few
// KB - enough to reach tar's magic at offset 257.
func LooksBinary(head []byte) string {
	// A byte-order mark is a positive text signal. UTF-16 text carries a NUL after most
	// characters, so without this check the NUL catch-all below would refuse a legitimate
	// Notepad "Unicode" save.
	if bytes.HasPrefix(head, bomUTF8) ||
		bytes.HasPrefix(head, bomUTF16LE) ||
		bytes.HasPrefix(head, bomUTF16BE) {
		return ""
	}

	if desc := namedBinarySignature(head); desc != "" {
		return desc
	}

	// No signature we name, but a NUL byte in the first few KB is the reliable mark of binary
	// data: measured, every text fixture in the corpus has zero, every binary has many. Real
	// text does not embed NUL.
	if bytes.IndexByte(head, 0x00) >= 0 {
		return "binary data"
	}
	return ""
}

// namedBinarySignature recognizes the specific binary formats that actually reach the app's
// unknown-extension arm (ZIP-based office docs and CBZ, RAR/7z/tar comics, DjVu), plus the
// formats the app does handle - named defensively in case one arrives under the wrong
// extension. Returns "" when nothing matches. Mirrors the signatures in viewer.js detectFormat.
func namedBinarySignature(head []byte) string {
	switch {
	// ZIP local-file header. DOCX, XLSX, PPTX, ODT and CBZ are all ZIP containers, and so is
	// EPUB - which has its own arm, so a PK here is a format we do not read.
	case bytes.HasPrefix(head, []byte("PK\x03\x04")),
		bytes.HasPrefix(head, []byte("PK\x05\x06")), // empty archive
		bytes.HasPrefix(head, []byte("PK\x07\x08")): // spanned archive
		return "a ZIP archive (DOCX, XLSX, PPTX, ODT and CBZ are ZIP-based)"
	case bytes.HasPrefix(head, []byte("Rar!\x1a\x07")):
		return "a RAR archive (CBR comic)"
	case bytes.HasPrefix(head, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}):
		return "a 7-Zip archive (CB7 comic)"
	case bytes.HasPrefix(head, []byte("AT&TFORM")):
		return "a DjVu document"
	case bytes.HasPrefix(head, []byte("%PDF")):
		return "a PDF"
	case isMOBISignature(head):
		return "a MOBI ebook"
	case isImageSignature(head):
		return "an image"
	case isTarSignature(head):
		return "a TAR archive (CBT comic)"
	}
	return ""
}

// isTarSignature reports the POSIX "ustar" magic, which sits at offset 257 - a tar has no magic
// at offset 0 (that is the first entry's filename), so a naive first-bytes table misses it. The
// NUL catch-all would still refuse it; this only lets the refusal name it.
func isTarSignature(head []byte) bool {
	const off = 257
	return len(head) >= off+5 && bytes.Equal(head[off:off+5], []byte("ustar"))
}

// isMOBISignature reports the PDB "BOOKMOBI" type/creator at offset 60 - MOBI and AZW3 are PDB
// databases. Mirrors viewer.js isMobiBytes.
func isMOBISignature(head []byte) bool {
	const off = 60
	return len(head) >= off+8 && bytes.Equal(head[off:off+8], []byte("BOOKMOBI"))
}

// isImageSignature reports the raster formats the app otherwise accepts as standalone images,
// so one arriving under an unknown extension is named rather than lumped in as "binary data".
func isImageSignature(head []byte) bool {
	switch {
	case bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}): // PNG
		return true
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}): // JPEG
		return true
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return true
	case bytes.HasPrefix(head, []byte("BM")) && len(head) >= 10 &&
		head[6] == 0 && head[7] == 0 && head[8] == 0 && head[9] == 0:
		// BMP: "BM", then a 4-byte size, then two reserved 16-bit fields that are zero in
		// practice. Requiring those zeros keeps a text file that merely opens with "BM" from
		// being mistaken for an image (a bare 2-byte match would).
		return true
	case bytes.HasPrefix(head, []byte{0x49, 0x49, 0x2A, 0x00}), // TIFF little-endian
		bytes.HasPrefix(head, []byte{0x4D, 0x4D, 0x00, 0x2A}): // TIFF big-endian
		return true
	case len(head) >= 12 && bytes.HasPrefix(head, []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return true
	}
	return false
}
