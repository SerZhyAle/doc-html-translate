package rtf

import (
	"math"
	"strings"
	"unicode/utf8"

	"doc-html-translate/internal/textutil"
)

// destKind says what a group's text is for.
type destKind uint8

const (
	destBody      destKind = iota // visible document text
	destSkip                      // a destination that is never body text
	destFontTable                 // the font table: read for charsets, never shown
)

// groupState is the part of the reader state RTF scopes to a group: entering '{' copies it and
// the matching '}' restores it.
type groupState struct {
	dest destKind
	uc   int // \ucN: fallback characters that follow each \uN
	font int // \fN
}

// maxParamDigits bounds a control word's numeric parameter; the spec allows a signed 16-bit
// value, and a longer run of digits is garbage that must not overflow.
const maxParamDigits = 10

// maxParamValue saturates a parameter's magnitude: ten digits exceed a 32-bit int, and on the
// 386 build the wrapped value used to reach the \bin skip arithmetic and panic.
const maxParamValue = math.MaxInt32

// reader turns RTF into plain text. It is a single forward pass: a group stack for the scoped
// state, a destination per group so tables, metadata and pictures never reach the text, and
// code-page decoding of \'XX bytes batched so a whole run goes through one decoder call.
type reader struct {
	in    []byte
	pos   int
	out   strings.Builder
	st    groupState
	stack []groupState

	ansiCP     int
	defFont    int
	fontCP     map[int]int // font number -> code page, from \fcharsetN or \cpgN
	definingFt int         // the font whose table entry is being read

	codecs    map[int]*textutil.Codec
	pending   []byte // \'XX and raw high bytes not yet decoded
	pendingCP int

	skip int  // fallback characters still to drop after a \uN
	high rune // a UTF-16 high surrogate waiting for its low half
}

// stripRTF returns the text of an RTF document. Paragraph breaks come out as blank lines.
func stripRTF(data []byte) string {
	r := &reader{
		in:     data,
		st:     groupState{uc: 1},
		ansiCP: defaultCodePage,
		fontCP: make(map[int]int),
		codecs: make(map[int]*textutil.Codec),
	}
	r.out.Grow(len(data) / 2)
	for r.pos < len(r.in) {
		c := r.in[r.pos]
		r.pos++
		switch c {
		case '{':
			r.flush()
			r.skip = 0
			r.stack = append(r.stack, r.st)
		case '}':
			r.flush()
			r.skip = 0
			if n := len(r.stack); n > 0 {
				r.st = r.stack[n-1]
				r.stack = r.stack[:n-1]
			}
		case '\\':
			r.control()
		case '\r', '\n':
			// Line breaks in RTF source are formatting of the file, not of the text.
		default:
			r.textByte(c)
		}
	}
	r.flush()
	r.resolveHigh()
	return r.out.String()
}

// textByte handles one literal byte of document text.
func (r *reader) textByte(c byte) {
	if r.skip > 0 {
		r.skip--
		return
	}
	if r.st.dest != destBody {
		return
	}
	if c >= 0x80 {
		r.addByte(c)
		return
	}
	r.flush()
	r.resolveHigh()
	r.out.WriteByte(c)
}

// control handles everything after a backslash.
func (r *reader) control() {
	if r.pos >= len(r.in) {
		return
	}
	c := r.in[r.pos]
	if !isLetter(c) {
		r.pos++
		r.symbol(c)
		return
	}
	start := r.pos
	for r.pos < len(r.in) && isLetter(r.in[r.pos]) {
		r.pos++
	}
	word := string(r.in[start:r.pos])
	param, hasParam := r.readParam()
	if r.pos < len(r.in) && r.in[r.pos] == ' ' {
		r.pos++
	}
	if word == "bin" {
		// Raw binary follows; it may contain braces and backslashes, so it is skipped by count
		// before anything could read it as RTF.
		r.flush()
		if hasParam && param > 0 {
			// Compared against what is left, never added first: r.pos+param can overflow.
			r.pos += min(param, len(r.in)-r.pos)
		}
		return
	}
	if r.skip > 0 {
		r.skip--
		return
	}
	r.word(word, param, hasParam)
}

// readParam reads an optional signed decimal parameter.
func (r *reader) readParam() (int, bool) {
	neg := false
	if r.pos+1 < len(r.in) && r.in[r.pos] == '-' && isDigit(r.in[r.pos+1]) {
		neg = true
		r.pos++
	}
	start := r.pos
	var n int64
	for r.pos < len(r.in) && isDigit(r.in[r.pos]) {
		if r.pos-start < maxParamDigits {
			n = min(n*10+int64(r.in[r.pos]-'0'), maxParamValue)
		}
		r.pos++
	}
	if r.pos == start {
		return 0, false
	}
	if neg {
		n = -n
	}
	return int(n), true
}

// symbol handles a control symbol: a backslash followed by one non-letter.
func (r *reader) symbol(c byte) {
	if c == '\'' {
		b, ok := r.readHex()
		if !ok {
			return
		}
		if r.skip > 0 {
			r.skip--
			return
		}
		if r.st.dest == destBody {
			r.addByte(b)
		}
		return
	}
	if r.skip > 0 {
		r.skip--
		return
	}
	switch c {
	case '*':
		// No \* destination carries body text this reader understands.
		r.st.dest = destSkip
	case '~':
		r.emitRune(0x00A0)
	case '_':
		r.emitRune(0x2011)
	case '{', '}', '\\':
		r.emitRune(rune(c))
	case '\r', '\n':
		r.emitString("\n\n")
	}
	// \- (optional hyphen), \| and \: (index formatting) are invisible in running text.
}

func (r *reader) readHex() (byte, bool) {
	if r.pos+2 > len(r.in) {
		r.pos = len(r.in)
		return 0, false
	}
	hi, ok1 := hexVal(r.in[r.pos])
	lo, ok2 := hexVal(r.in[r.pos+1])
	r.pos += 2
	return hi<<4 | lo, ok1 && ok2
}

// word applies one control word.
func (r *reader) word(word string, param int, hasParam bool) {
	if r.st.dest == destFontTable {
		switch word {
		case "f":
			r.definingFt = param
			return
		case "fcharset":
			if cp, ok := charsetCodePages[param]; ok {
				r.fontCP[r.definingFt] = cp
			}
			return
		case "cpg":
			r.fontCP[r.definingFt] = param
			return
		}
	}
	switch {
	case word == "fonttbl":
		r.st.dest = destFontTable
	case skippedDestinations[word]:
		r.st.dest = destSkip
	case word == "ansicpg" && hasParam:
		r.ansiCP = param
	case word == "mac":
		r.ansiCP = 10000
	case word == "deff" && hasParam:
		r.defFont = param
		r.st.font = param
	case word == "f" && hasParam:
		r.st.font = param
	case word == "plain":
		r.st.font = r.defFont
	case word == "uc" && hasParam && param >= 0:
		r.st.uc = param
	case word == "u" && hasParam:
		r.unicode(param)
		r.skip = r.st.uc
	case breakWords[word]:
		r.emitString("\n\n")
	case word == "tab" || word == "cell" || word == "nestcell":
		r.emitString("\t")
	default:
		if sym, ok := symbolWords[word]; ok {
			r.emitRune(sym)
		}
	}
}

// unicode emits a \uN value. N is a signed 16-bit number, so characters above U+7FFF arrive
// negative, and characters beyond the BMP arrive as a surrogate pair of two \u words.
func (r *reader) unicode(n int) {
	if r.st.dest != destBody {
		return
	}
	if n < 0 {
		n += 0x10000
	}
	r.flush()
	switch {
	case n >= 0xD800 && n <= 0xDBFF:
		r.resolveHigh()
		r.high = rune(n)
		return
	case n >= 0xDC00 && n <= 0xDFFF:
		if r.high != 0 {
			r.out.WriteRune(0x10000 + (r.high-0xD800)<<10 + rune(n) - 0xDC00)
			r.high = 0
			return
		}
		r.out.WriteRune(utf8.RuneError)
		return
	}
	r.resolveHigh()
	if n > 0 && n <= utf8.MaxRune {
		r.out.WriteRune(rune(n))
	}
}

// resolveHigh writes a high surrogate that never met its low half as U+FFFD.
func (r *reader) resolveHigh() {
	if r.high != 0 {
		r.out.WriteRune(utf8.RuneError)
		r.high = 0
	}
}

func (r *reader) emitRune(c rune) {
	if r.st.dest != destBody {
		return
	}
	r.flush()
	r.resolveHigh()
	r.out.WriteRune(c)
}

func (r *reader) emitString(s string) {
	if r.st.dest != destBody {
		return
	}
	r.flush()
	r.resolveHigh()
	r.out.WriteString(s)
}

// addByte queues a code-page byte. Consecutive bytes are decoded together, which is what lets
// a double-byte code page (Shift-JIS, GBK) see both halves of a character.
func (r *reader) addByte(b byte) {
	cp := r.codePage()
	if len(r.pending) > 0 && cp != r.pendingCP {
		r.flush()
	}
	r.pending = append(r.pending, b)
	r.pendingCP = cp
}

// flush decodes the queued bytes with the code page they were written in.
func (r *reader) flush() {
	if len(r.pending) == 0 {
		return
	}
	r.resolveHigh()
	r.out.WriteString(r.codec(r.pendingCP).Decode(r.pending))
	r.pending = r.pending[:0]
}

// codePage is the code page of the current font, else the document's.
func (r *reader) codePage() int {
	if cp, ok := r.fontCP[r.st.font]; ok {
		return cp
	}
	return r.ansiCP
}

// codec returns the decoder for a code page, built once per code page per document.
func (r *reader) codec(cp int) *textutil.Codec {
	if c, ok := r.codecs[cp]; ok {
		return c
	}
	label, ok := codePageLabels[cp]
	if !ok {
		label = codePageLabels[defaultCodePage]
	}
	c, ok := textutil.LookupCodec(label)
	if !ok {
		c, _ = textutil.LookupCodec(codePageLabels[defaultCodePage])
	}
	r.codecs[cp] = c
	return c
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
