package rtf

// The tables below are mirrored in extension/src/rtf.js and pinned by tests/parity_test.go; a
// change here is a change there (docs/PARITY.md, "RTF text decoding").

// defaultCodePage applies when the document names none, or names one no browser can decode:
// RTF's \ansi default is Windows-1252.
const defaultCodePage = 1252

// codePageLabels maps a Windows code page number (\ansicpgN, \cpgN) to its WHATWG Encoding
// Standard label - the set the extension's TextDecoder understands. A code page missing here
// (437 and 850 for \pc and \pca, for instance) falls back to defaultCodePage in both editions.
var codePageLabels = map[int]string{
	866:   "ibm866",
	874:   "windows-874",
	932:   "shift_jis",
	936:   "gbk",
	949:   "euc-kr",
	950:   "big5",
	1250:  "windows-1250",
	1251:  "windows-1251",
	1252:  "windows-1252",
	1253:  "windows-1253",
	1254:  "windows-1254",
	1255:  "windows-1255",
	1256:  "windows-1256",
	1257:  "windows-1257",
	1258:  "windows-1258",
	10000: "macintosh",
	10007: "x-mac-cyrillic",
	20866: "koi8-r",
	21866: "koi8-u",
	28592: "iso-8859-2",
	28595: "iso-8859-5",
	28597: "iso-8859-7",
	28605: "iso-8859-15",
	54936: "gb18030",
}

// charsetCodePages maps a font's \fcharsetN to the code page its \'XX bytes are in. ANSI (0)
// and DEFAULT (1) are absent on purpose: they mean "the document's \ansicpg", which is how
// naive generators label Cyrillic fonts in a \ansicpg1251 document.
var charsetCodePages = map[int]int{
	77:  10000,
	128: 932,
	129: 949,
	134: 936,
	136: 950,
	161: 1253,
	162: 1254,
	163: 1258,
	177: 1255,
	178: 1256,
	186: 1257,
	204: 1251,
	222: 874,
	238: 1250,
}

// skippedDestinations are groups whose content is never body text: tables, metadata, pictures,
// embedded objects, running headers and footers, field instructions and index entries. Every
// {\*\..} group is skipped as well, since \* marks a destination a reader may ignore. Only
// known names are listed, so an unknown generator's real text is never hidden.
var skippedDestinations = map[string]bool{
	"annotation":         true,
	"atnauthor":          true,
	"atnid":              true,
	"bkmkend":            true,
	"bkmkstart":          true,
	"colorschememapping": true,
	"colortbl":           true,
	"datastore":          true,
	"docvar":             true,
	"falt":               true,
	"filetbl":            true,
	"fldinst":            true,
	"fontemb":            true,
	"fontfile":           true,
	"footer":             true,
	"footerf":            true,
	"footerl":            true,
	"footerr":            true,
	"ftncn":              true,
	"ftnsep":             true,
	"ftnsepc":            true,
	"generator":          true,
	"header":             true,
	"headerf":            true,
	"headerl":            true,
	"headerr":            true,
	"info":               true,
	"latentstyles":       true,
	"listoverridetable":  true,
	"listpicture":        true,
	"listtable":          true,
	"nonshppict":         true,
	"object":             true,
	"objdata":            true,
	"panose":             true,
	"pgdsctbl":           true,
	"pict":               true,
	"private":            true,
	"revtbl":             true,
	"rsidtbl":            true,
	"stylesheet":         true,
	"tc":                 true,
	"template":           true,
	"themedata":          true,
	"userprops":          true,
	"xe":                 true,
	"xmlnsdecl":          true,
}

// symbolWords are control words that stand for one character. Code points, not literals: the
// house typography guard rightly rejects a long dash in a source string, but a document's
// \emdash is content and must survive.
var symbolWords = map[string]rune{
	"bullet":    0x2022,
	"emdash":    0x2014,
	"emspace":   0x2003,
	"endash":    0x2013,
	"enspace":   0x2002,
	"ldblquote": 0x201C,
	"lquote":    0x2018,
	"ltrmark":   0x200E,
	"qmspace":   0x2005,
	"rdblquote": 0x201D,
	"rquote":    0x2019,
	"rtlmark":   0x200F,
	"zwj":       0x200D,
	"zwnj":      0x200C,
}

// breakWords end a paragraph; cell ends a table cell, which reads as a tab within the row.
var breakWords = map[string]bool{
	"line": true,
	"page": true,
	"par":  true,
	"row":  true,
	"sect": true,
}
