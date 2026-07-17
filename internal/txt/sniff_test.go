package txt

import "testing"

func TestLooksBinaryNamesFormats(t *testing.T) {
	tests := []struct {
		name string
		head []byte
		want string
	}{
		{"zip (docx/xlsx/cbz)", []byte("PK\x03\x04rest"), "a ZIP archive (DOCX, XLSX, PPTX, ODT and CBZ are ZIP-based)"},
		{"rar (cbr)", []byte("Rar!\x1a\x07\x00rest"), "a RAR archive (CBR comic)"},
		{"7z (cb7)", append([]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, 0x00, 0x01), "a 7-Zip archive (CB7 comic)"},
		{"djvu", []byte("AT&TFORM\x00\x00\x00\x00DJVM"), "a DjVu document"},
		{"pdf", []byte("%PDF-1.7\n"), "a PDF"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2}, "an image"},
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0}, "an image"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksBinary(tt.head); got != tt.want {
				t.Errorf("LooksBinary() = %q, want %q", got, tt.want)
			}
		})
	}
}

// tar has no magic at offset 0 - that is the first entry's filename - so the "ustar" magic at
// offset 257 is what names it. Build a minimal header: a filename, padding to 257, then ustar.
func TestLooksBinaryTarByOffset257(t *testing.T) {
	head := make([]byte, 512)
	copy(head, []byte("some-file.jpg"))
	copy(head[257:], []byte("ustar"))
	if got := LooksBinary(head); got != "a TAR archive (CBT comic)" {
		t.Errorf("tar not named: got %q", got)
	}
}

// A binary with no signature we name must still be refused, via the NUL catch-all.
func TestLooksBinaryUnknownWithNUL(t *testing.T) {
	head := []byte{0x00, 0x01, 0x02, 'x', 'y', 'z'}
	if got := LooksBinary(head); got != "binary data" {
		t.Errorf("unsigned binary not refused: got %q", got)
	}
}

// Text must pass, and UTF-16 - full of NUL bytes - must pass because its BOM is checked first.
// This is the interaction with the NUL rule that the ticket flagged: get it wrong and a Notepad
// "Unicode" save is refused as binary.
func TestLooksBinaryAcceptsText(t *testing.T) {
	tests := []struct {
		name string
		head []byte
	}{
		{"plain ascii", []byte("The Project Gutenberg eBook of something.\n")},
		{"utf-8 cyrillic", []byte("Это обычное предложение на русском языке.\n")},
		{"utf-8 bom", append(bomUTF8, []byte("hello")...)},
		{"utf-16le bom (has NULs)", append(bomUTF16LE, []byte{'T', 0, 'h', 0, 'e', 0}...)},
		{"utf-16be bom (has NULs)", append(bomUTF16BE, []byte{0, 'T', 0, 'h', 0, 'e'}...)},
		{"text starting with BM (not a BMP)", []byte("BM this is a note, not a bitmap image file.")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksBinary(tt.head); got != "" {
				t.Errorf("text refused as %q", got)
			}
		})
	}
}

// A real BMP (BM + size + zero reserved fields) is still named, so the BM tightening did not
// break the true positive.
func TestLooksBinaryRealBMP(t *testing.T) {
	head := append([]byte("BM"), []byte{0x36, 0x84, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00}...)
	if got := LooksBinary(head); got != "an image" {
		t.Errorf("real BMP not named: got %q", got)
	}
	// And the tightening holds: "BM" + non-zero at the reserved offset is not treated as a BMP.
	notBMP := append([]byte("BM"), []byte("text follows here and there")...)
	if got := LooksBinary(notBMP); got == "an image" {
		t.Errorf(`"BM" text wrongly named an image`)
	}
}
