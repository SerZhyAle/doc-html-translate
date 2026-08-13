package ocr

import (
	"strings"
	"testing"
)

// A code alone tells the reader nothing - "eng" is only obvious to someone who already knows
// what went wrong. The code stays first because it is what has to be typed back into -ocr-lang.
func TestLangLabel(t *testing.T) {
	cases := map[string]string{
		"eng":     "eng (English)",
		"rus":     "rus (Russian)",
		"eng+rus": "eng (English) + rus (Russian)",
		"xyz":     "xyz",
		"":        "",
	}
	for in, want := range cases {
		if got := LangLabel(in); got != want {
			t.Errorf("LangLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// ISOFor is what lets the GUI keep the OCR language on the -src language; a wrong answer
// there sends Cyrillic pages to the English data, which recognizes nothing.
func TestISOForRoundTripsTessLang(t *testing.T) {
	for _, l := range Available {
		iso := ISOFor(l.Code)
		if iso == "" {
			continue // no ISO-639-1 code selects this data (jpn_vert) - nothing to check
		}
		if got := TessLang(iso); got != l.Code {
			t.Errorf("TessLang(ISOFor(%q)) = %q, want %q", l.Code, got, l.Code)
		}
	}
	if got := ISOFor("script/Cyrillic"); got != "" {
		t.Errorf("ISOFor of an unmapped name = %q, want empty", got)
	}
}

// A probe that could not run is not evidence. Reporting "not installed" because the engine
// could not be asked would send the user to download data they already have, and - worse - the
// caller skips the OCR pass on that answer.
func TestMissingLangsSaysNothingWhenTheEngineCannotBeAsked(t *testing.T) {
	if got := MissingLangs("no-such-tesseract-binary", "rus"); got != nil {
		t.Errorf("MissingLangs with an unusable engine = %v, want nil", got)
	}
}

// The header line of --list-langs names a directory and a count, so it carries spaces; every
// other line is one code. The header must go and jpn_vert / script/Cyrillic must survive - a
// dropped code would be reported to the user as missing data they actually have.
func TestParseLangList(t *testing.T) {
	sample := "List of available languages in \"C:\\Program Files\\Tesseract-OCR/tessdata/\" (3):\r\n" +
		"eng\r\njpn_vert\r\nscript/Cyrillic\r\n"
	got := strings.Join(parseLangList(sample), ",")
	if want := "eng,jpn_vert,script/Cyrillic"; got != want {
		t.Errorf("parseLangList = %q, want %q", got, want)
	}
}
