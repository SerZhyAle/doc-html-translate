package evidence

import (
	"slices"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// The language a scene is read with. The lab used to read the whole campaign with one language
// while the truth declared Russian, French and English; a Cyrillic poster read as Latin yields
// debris the confidence gate then rightly drops, which looked like a product loss and was a lab
// setting. A scene now carries its own declared language unless the operator forces one.

const (
	// DefaultLang is what a scene with no declared (or no catalogue-readable) language is read with.
	DefaultLang = "eng"
	// LangPerScene is the run-level label when each scene chose its own language.
	LangPerScene = "per-scene"

	// Where a scene's language came from, recorded beside it.
	LangSourceFlag    = "flag"    // the operator passed -lang / --lang
	LangSourceTruth   = "truth"   // the annotation's group languages
	LangSourceCorpus  = "corpus"  // the manifest's languages field
	LangSourceDefault = "default" // nothing declared, or nothing the catalogue can read
)

// isoToTess maps a declared BCP-47 language to the OCR catalogue's Tesseract pack. Only catalogue
// packs are listed: a declared language the catalogue cannot read (ar, hi, nl..) has no entry and
// falls back to DefaultLang, because naming a pack the app cannot download would only move the
// failure. The same table, entry for entry, lives in extension/scripts/_ocrlab-lang.mjs, and
// TestParityOCRLabLangTable fails when either side or internal/ocr (TessLang) moves alone.
var isoToTess = map[string]string{
	"en": "eng", "ru": "rus", "uk": "ukr", "de": "deu", "fr": "fra",
	"es": "spa", "it": "ita", "pt": "por", "pl": "pol", "ja": "jpn",
	"zh": "chi_sim", "ko": "kor",
}

// regionToTess are the subtags that select a different pack than their base language: the
// traditional script and the regions that print it. Consulted before the base-subtag fallback.
var regionToTess = map[string]string{
	"zh-hant": "chi_tra", "zh-tw": "chi_tra", "zh-hk": "chi_tra", "zh-mo": "chi_tra",
}

// LangTables returns copies of the two mapping tables, for the parity test.
func LangTables() (iso, region map[string]string) {
	iso, region = map[string]string{}, map[string]string{}
	for k, v := range isoToTess {
		iso[k] = v
	}
	for k, v := range regionToTess {
		region[k] = v
	}
	return iso, region
}

// TessCode maps one declared language to its catalogue pack.
func TessCode(declared string) (string, bool) {
	norm := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(declared), "_", "-"))
	if t, ok := regionToTess[norm]; ok {
		return t, true
	}
	base, _, _ := strings.Cut(norm, "-")
	t, ok := isoToTess[base]
	return t, ok
}

// LangChoice is the language one scene is read with and where that came from.
type LangChoice struct {
	Lang   string // Tesseract code, "+"-joined when the scene declares several
	Source string
}

// SceneLang decides a scene's OCR language: an explicit operator choice wins, then the languages
// of the human-reviewed annotation's groups, then the manifest's languages, then DefaultLang. Several
// declared languages become one "+"-joined value in order of first appearance - Tesseract takes the
// first as primary, and the annotation's group order is stable.
func SceneLang(annotations, explicit string, s *corpus.Scene) LangChoice {
	if explicit != "" {
		return LangChoice{Lang: explicit, Source: LangSourceFlag}
	}
	if a, err := truth.Load(truth.FinalPath(annotations, s.ID)); err == nil && a.IsTruth() {
		declared := make([]string, 0, len(a.Groups))
		for _, g := range a.Groups {
			declared = append(declared, g.Language)
		}
		if lang := joinCodes(declared); lang != "" {
			return LangChoice{Lang: lang, Source: LangSourceTruth}
		}
	}
	if lang := joinCodes(s.Languages); lang != "" {
		return LangChoice{Lang: lang, Source: LangSourceCorpus}
	}
	return LangChoice{Lang: DefaultLang, Source: LangSourceDefault}
}

func joinCodes(declared []string) string {
	var codes []string
	for _, d := range declared {
		if t, ok := TessCode(d); ok && !slices.Contains(codes, t) {
			codes = append(codes, t)
		}
	}
	return strings.Join(codes, "+")
}
