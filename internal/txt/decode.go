package txt

import (
	"encoding/binary"
	"unicode/utf8"

	"doc-html-translate/internal/textutil"
)

// utf16SniffBytes bounds the BOM-less UTF-16 check to the head of the file, like the binary
// sniff: the NUL pattern is established within the first lines of any real text.
const utf16SniffBytes = 4096

// sniffUTF16 recognizes UTF-16 saved without a byte-order mark. Such a file is often valid UTF-8
// (ASCII and Cyrillic code units are all bytes below 0x80), so without this check it reached the
// page full of NUL and 0x04 control bytes. The signal is where the NUL bytes sit: the high byte
// of every Latin letter, digit, space and punctuation mark is zero, so NULs pile up on one
// parity (odd offsets for little-endian, even for big-endian) and almost never on the other.
// Real 8-bit text contains no NUL at all.
//
// The dominant parity must hold at least 2 NULs and cover 5% of the code units - Cyrillic
// prose, whose letters have no zero byte, still carries a space every few words - and the
// other parity at most a tenth as many, which a binary blob with NULs everywhere fails.
// Mirrors sniffUtf16 in extension/src/txt.js.
func sniffUTF16(raw []byte) (binary.ByteOrder, bool) {
	n := len(raw)
	if n > utf16SniffBytes {
		n = utf16SniffBytes
	}
	n &^= 1
	units := n / 2
	if units < 2 {
		return nil, false
	}
	even, odd := 0, 0
	for i := 0; i < n; i += 2 {
		if raw[i] == 0 {
			even++
		}
		if raw[i+1] == 0 {
			odd++
		}
	}
	dominates := func(hi, lo int) bool {
		return hi >= 2 && hi*20 >= units && lo*10 <= hi
	}
	switch {
	case dominates(odd, even):
		return binary.LittleEndian, true
	case dominates(even, odd):
		return binary.BigEndian, true
	}
	return nil, false
}

// acceptAsUTF8 decides that bytes which are not strictly valid UTF-8 are still UTF-8 with a
// little damage, rather than a legacy code page. One bad byte in a Russian UTF-8 book used to
// send the whole file to the Cyrillic detector, which turned every letter into mojibake.
//
// Owner decision (ticket 10, section 6.1): the text is UTF-8 when it holds at least one valid
// multi-byte sequence and its invalid bytes are under 1% of those sequences, or when the only
// invalidity is a sequence cut off by the end of the file (at most 3 bytes). A legacy 8-bit
// file fails both: its high bytes almost never form valid multi-byte sequences. Mirrors
// acceptAsUtf8 in extension/src/txt.js.
func acceptAsUTF8(raw []byte) bool {
	if utf8.Valid(raw) {
		return true
	}
	d := textutil.MeasureUTF8(raw)
	if d.Errors == 0 {
		return true
	}
	if d.Errors == 1 && d.TruncatedTail {
		return true
	}
	return d.Multi > 0 && d.InvalidBytes*100 < d.Multi
}
