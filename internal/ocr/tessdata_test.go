package ocr

import (
	"os"
	"path/filepath"
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

// -src codes the catalog cannot read used to pass through (cs, hi, zh-CN) or map to packs it does
// not offer (nld), so the pass failed on data no command could install. Every derived value must be
// a catalog language, which is exactly what -ocr-download accepts.
func TestTessLangDerivesOnlyCatalogLanguages(t *testing.T) {
	cases := map[string]string{
		"":         "eng",
		"en":       "eng",
		"RU":       "rus",
		"pt-BR":    "por",
		"zh-CN":    "chi_sim",
		"zh_TW":    "chi_sim",
		"cs":       "eng",
		"sv":       "eng",
		"hi":       "eng",
		"nl":       "eng",
		"tr":       "eng",
		"ar":       "eng",
		"xyz":      "eng",
		"jpn_vert": "jpn_vert",
		"rus+eng":  "rus+eng",
		"ru+nl":    "rus",
	}
	for in, want := range cases {
		got := TessLang(in)
		if got != want {
			t.Errorf("TessLang(%q) = %q, want %q", in, got, want)
		}
		for _, code := range strings.Split(got, "+") {
			if err := CheckLang(code); err != nil {
				t.Errorf("TessLang(%q) derived %q, which the download catalog refuses", in, code)
			}
		}
	}
}

// The advice must name a command that works: -ocr-download refuses a code outside the catalog.
func TestMissingAdviceOffersOnlyCatalogDownloads(t *testing.T) {
	if got := MissingAdvice([]string{"nld"}); strings.Contains(got, "-ocr-download") {
		t.Errorf("advice for a non-catalog code offers a download: %q", got)
	}
	if got := MissingAdvice([]string{"nld", "rus"}); !strings.Contains(got, "-ocr-download rus") {
		t.Errorf("advice for rus = %q, want -ocr-download rus", got)
	}
}

// A bundled pack that could not be staged beside a downloaded one used to drop --tessdata-dir
// silently, and every image then failed with an engine error that named neither.
func TestDataDirForNamesAPackThatCouldNotBeStaged(t *testing.T) {
	user, bundled := dataDirs(t)
	for _, dir := range []string{user, bundled} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(langFile(user, "rus"), []byte("rus"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(langFile(bundled, "eng"), []byte("eng"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A non-empty folder in the pack's place makes the rename fail on every platform and for
	// every user, root included.
	if err := os.MkdirAll(filepath.Join(langFile(user, "eng"), "x"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := DataDirFor("rus+eng")
	if err == nil || !strings.Contains(err.Error(), "eng (English)") || !strings.Contains(err.Error(), user) {
		t.Errorf("DataDirFor(rus+eng) err = %v, want one naming the eng pack and %s", err, user)
	}
	// A language the failed pack is not needed for is unaffected.
	if dir, err := DataDirFor("rus"); err != nil || dir != user {
		t.Errorf("DataDirFor(rus) = %q, %v; want %q", dir, err, user)
	}
	// One the bundled folder serves alone falls back to it.
	if dir, err := DataDirFor("eng"); err != nil || dir != bundled {
		t.Errorf("DataDirFor(eng) = %q, %v; want %q", dir, err, bundled)
	}
}
