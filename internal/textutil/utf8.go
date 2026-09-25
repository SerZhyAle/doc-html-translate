package textutil

import (
	"strings"
	"unicode/utf8"
)

// UTF8Damage summarizes how a byte slice fails to be UTF-8, measured with the WHATWG decoder
// algorithm so the extension (TextDecoder) counts exactly the same errors.
type UTF8Damage struct {
	// Multi is the number of well-formed multi-byte sequences: the evidence that the text is
	// UTF-8 at all, since ASCII is valid in every candidate encoding.
	Multi int
	// InvalidBytes is the number of bytes that decode to U+FFFD.
	InvalidBytes int
	// Errors is the number of U+FFFD the decoder emits (one per maximal invalid subpart).
	Errors int
	// TruncatedTail reports that the data ends partway through an otherwise well-formed
	// sequence - a file cut off mid-character, not one written in another encoding. That
	// error is included in Errors.
	TruncatedTail bool
}

// MeasureUTF8 scans b once and reports its UTF-8 damage. It never allocates.
func MeasureUTF8(b []byte) UTF8Damage {
	var d UTF8Damage
	walkUTF8(b, func(r rune, size int, bad bool) {
		switch {
		case bad:
			d.Errors++
			d.InvalidBytes += size
		case size > 1:
			d.Multi++
		}
	}, &d.TruncatedTail)
	return d
}

// DecodeUTF8 returns b as a valid UTF-8 string. Each maximal invalid subpart becomes one U+FFFD,
// which is what TextDecoder("utf-8") emits, so both editions show the same damage in the same
// places. Go's strings.ToValidUTF8 would collapse a whole run into one replacement instead.
func DecodeUTF8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var sb strings.Builder
	sb.Grow(len(b) + 8)
	start := 0
	walkUTF8(b, func(r rune, size int, bad bool) {
		if bad {
			sb.WriteRune(utf8.RuneError)
		} else {
			sb.Write(b[start : start+size])
		}
		start += size
	}, nil)
	return sb.String()
}

// walkUTF8 is the WHATWG UTF-8 decoder: it reports every decoded code point or error with the
// number of input bytes it covers. An unexpected byte ends the current sequence as an error and
// is then reprocessed as the start of the next one - the "maximal subpart" rule.
func walkUTF8(b []byte, emit func(r rune, size int, bad bool), truncatedTail *bool) {
	var cp rune
	needed, seen := 0, 0
	lower, upper := byte(0x80), byte(0xBF)
	for i := 0; i < len(b); {
		c := b[i]
		if needed == 0 {
			i++
			switch {
			case c < 0x80:
				emit(rune(c), 1, false)
			case c >= 0xC2 && c <= 0xDF:
				needed, cp = 1, rune(c&0x1F)
			case c >= 0xE0 && c <= 0xEF:
				// E0 would otherwise admit overlong forms, ED the UTF-16 surrogates.
				switch c {
				case 0xE0:
					lower = 0xA0
				case 0xED:
					upper = 0x9F
				}
				needed, cp = 2, rune(c&0x0F)
			case c >= 0xF0 && c <= 0xF4:
				// F0 would otherwise admit overlong forms, F4 code points past U+10FFFF.
				switch c {
				case 0xF0:
					lower = 0x90
				case 0xF4:
					upper = 0x8F
				}
				needed, cp = 3, rune(c&0x07)
			default:
				emit(utf8.RuneError, 1, true)
			}
			continue
		}
		if c < lower || c > upper {
			emit(utf8.RuneError, seen+1, true)
			needed, seen, lower, upper = 0, 0, 0x80, 0xBF
			continue
		}
		i++
		lower, upper = 0x80, 0xBF
		cp = cp<<6 | rune(c&0x3F)
		seen++
		if seen == needed {
			emit(cp, needed+1, false)
			needed, seen = 0, 0
		}
	}
	if needed > 0 {
		emit(utf8.RuneError, seen+1, true)
	}
	if truncatedTail != nil {
		*truncatedTail = needed > 0
	}
}
