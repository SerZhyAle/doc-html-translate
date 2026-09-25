package txt

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

func utf16NoBOM(s string, order binary.ByteOrder) []byte {
	units := utf16.Encode([]rune(s))
	buf := make([]byte, len(units)*2)
	for i, u := range units {
		order.PutUint16(buf[i*2:], u)
	}
	return buf
}

// BOM-less UTF-16 is valid UTF-8 for ASCII and Cyrillic, so it used to reach the page with a
// NUL (or a 0x04) between every letter.
func TestDecodeTextBOMlessUTF16(t *testing.T) {
	for _, text := range []string{
		"Plain English line.\nSecond line.",
		"Привет, мир. Это текст без метки порядка байтов.",
	} {
		for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
			if got := decodeText(utf16NoBOM(text, order)); got != text {
				t.Errorf("%v %q: got %q", order, text, got)
			}
		}
	}
}

func TestSniffUTF16RejectsText(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte("ordinary text with no NUL bytes at all"),
		[]byte("Привет"),
		{0, 0, 0, 0, 0, 0, 0, 0}, // NULs on both parities: binary, not UTF-16
		{'a', 0},                 // too short to judge
	} {
		if _, ok := sniffUTF16(raw); ok {
			t.Errorf("sniffUTF16(%q) = true", raw)
		}
	}
}

// One damaged byte in a Russian UTF-8 book used to send the whole file to the legacy detector.
func TestDecodeTextToleratesDamagedUTF8(t *testing.T) {
	body := strings.Repeat("Обычный русский текст в кодировке UTF-8. ", 20)
	truncated := []byte(body + "конец")
	truncated = truncated[:len(truncated)-1]
	got := decodeText(truncated)
	if !strings.HasSuffix(got, "коне\U0000FFFD") || !strings.HasPrefix(got, "Обычный") {
		t.Errorf("truncated tail: got suffix %q", got[len(got)-20:])
	}

	mid := []byte(body)
	mid[len(mid)/2+strings.IndexByte(body[len(body)/2:], ' ')] = 0xFF
	got = decodeText(mid)
	if strings.Count(got, "\U0000FFFD") != 1 || !strings.HasPrefix(got, "Обычный") {
		t.Errorf("mid-text damage should cost exactly one replacement character")
	}
}

// Damage above the threshold is not UTF-8 any more: 2 bad bytes against 10 valid sequences.
func TestAcceptAsUTF8Threshold(t *testing.T) {
	if acceptAsUTF8([]byte("ПриветПрив\xff\xff")) {
		t.Error("20% damage accepted as UTF-8")
	}
	if !acceptAsUTF8([]byte("abc\xd0")) {
		t.Error("a truncated tail alone must be accepted")
	}
	if acceptAsUTF8([]byte("caf\xe9 cr\xe8me")) {
		t.Error("Latin-1 accepted as UTF-8")
	}
}

func TestDecodeTextOutputAlwaysValid(t *testing.T) {
	for _, raw := range [][]byte{
		{0xFF, 0xFE, 'a', 0, 0x00, 0xD8}, // UTF-16LE BOM with a lone surrogate
		{0xEF, 0xBB, 0xBF, 'a', 0xFF},
		{0x81, 0x8D, 0x8F, 0x90, 0x9D},
	} {
		if got := decodeText(raw); !utf8.ValidString(got) {
			t.Errorf("decodeText(% x) = %q is not valid UTF-8", raw, got)
		}
	}
}
