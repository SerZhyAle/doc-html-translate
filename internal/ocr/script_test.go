package ocr

import (
	"strings"
	"testing"
)

// TestResolveScriptLeavesAChosenLanguageAlone: the rule exists because the default comes from a
// flag about translation. A language the reader typed is an instruction, not a guess, and the
// detector - which gets a non-Latin script right on two of the corpus's 46 scenes - never overrides
// one.
func TestResolveScriptLeavesAChosenLanguageAlone(t *testing.T) {
	use, note, stop := resolveScript("eng", true, "Cyrillic", 9.0, true, []string{"rus"})
	if use != "eng" || note != "" || stop {
		t.Errorf("a chosen language was overridden: use=%q note=%q stop=%v", use, note, stop)
	}
}

// TestResolveScriptIgnoresAnUnsureDetector is the price side of the floor, and the corpus is what
// sets it: the detector calls an English Archie cover Cyrillic at 3.81 and an English cartoon
// Arabic at 5.00. Both must change nothing.
func TestResolveScriptIgnoresAnUnsureDetector(t *testing.T) {
	for _, c := range []struct {
		script string
		conf   float64
	}{{"Cyrillic", 3.81}, {"Arabic", 5.00}, {"Cyrillic", ocrScriptConfidenceFloor - 0.01}} {
		use, note, stop := resolveScript("eng", false, c.script, c.conf, true, nil)
		if use != "eng" || note != "" || stop {
			t.Errorf("%s at %.2f acted: use=%q note=%q stop=%v", c.script, c.conf, use, note, stop)
		}
	}
	// Nothing detected at all - no osd.traineddata, too few characters - is not evidence either.
	if use, note, stop := resolveScript("eng", false, "", 0, false, []string{"rus"}); use != "eng" || note != "" || stop {
		t.Errorf("a silent detector acted: use=%q note=%q stop=%v", use, note, stop)
	}
	// A script the default already reads is not a mismatch.
	if use, _, _ := resolveScript("rus", false, "Cyrillic", 9.0, true, []string{"rus"}); use != "rus" {
		t.Errorf("use = %q, want rus (the language already reads Cyrillic)", use)
	}
}

// TestResolveScriptAddsRatherThanReplaces: the correction has to survive being wrong. Tesseract
// reads a "+"-joined pair, so keeping the original beside the detected one costs a slower pass on a
// false positive instead of an unreadable page.
func TestResolveScriptAddsRatherThanReplaces(t *testing.T) {
	use, note, stop := resolveScript("eng", false, "Cyrillic", 8.24, true, []string{"rus", "ukr"})
	if use != "rus+eng" {
		t.Errorf("use = %q, want rus+eng", use)
	}
	if stop {
		t.Error("the pass was stopped even though data for the script is installed")
	}
	if !strings.Contains(note, "Cyrillic") || !strings.Contains(note, "-ocr-lang") {
		t.Errorf("note = %q, want the script named and the override offered", note)
	}
}

// TestResolveScriptRefusesRatherThanTransliterates is the ticket's criterion in one assertion:
// with no data that can read the page, the answer is no plates and a sentence saying why - never
// the transliterated debris an English recognizer produces from Cyrillic.
func TestResolveScriptRefusesRatherThanTransliterates(t *testing.T) {
	use, note, stop := resolveScript("eng", false, "Cyrillic", 8.24, true, nil)
	if !stop {
		t.Error("the pass ran anyway, so the page gets transliterated plates")
	}
	if use != "eng" {
		t.Errorf("use = %q, want the original language left alone when nothing can replace it", use)
	}
	if !strings.Contains(note, "-ocr-download rus") {
		t.Errorf("note = %q, want the download that fixes it", note)
	}
	// A script the app has no data for at all still refuses, and says so rather than naming a
	// download that does not exist.
	_, greek, stop := resolveScript("eng", false, "Greek", 9.0, true, nil)
	if !stop || strings.Contains(greek, "-ocr-download") {
		t.Errorf("Greek: stop=%v note=%q, want a refusal with no download to offer", stop, greek)
	}
}

// TestScriptOfReadsTheLeadingLanguage: the language string can be a "+"-joined pair once the rule
// has fired, and asking it again must not read the second half.
func TestScriptOfReadsTheLeadingLanguage(t *testing.T) {
	for lang, want := range map[string]string{
		"eng": "Latin", "rus": "Cyrillic", "rus+eng": "Cyrillic", "jpn_vert": "Japanese", "zzz": "Latin",
	} {
		if got := scriptOf(lang); got != want {
			t.Errorf("scriptOf(%q) = %q, want %q", lang, got, want)
		}
	}
}
