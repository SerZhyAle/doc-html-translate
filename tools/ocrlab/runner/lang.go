package runner

import (
	"slices"
	"strings"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
)

// missingLangData lists the packs of lang that no tessdata folder the app reads holds. The lab
// never downloads: it has no opt-in flag for network access, so a scene whose language is not
// installed is recorded as unmeasured instead of being read with the wrong language.
func missingLangData(lang string) []string {
	var missing []string
	for _, code := range strings.Split(lang, "+") {
		if code = strings.TrimSpace(code); code != "" && !ocr.IsInstalled(code) {
			missing = append(missing, code)
		}
	}
	return missing
}

// runLangs names the run's language for the evidence header and the packs whose bytes identify its
// engine. An explicit -lang is both, as before; per-scene it is the label and the union of the
// installed packs the selected scenes will actually read.
func runLangs(scenes []*corpus.Scene, opt Options) (label, packs string) {
	if opt.Lang != "" {
		return opt.Lang, opt.Lang
	}
	var codes []string
	for _, s := range scenes {
		for _, code := range strings.Split(evidence.SceneLang(opt.Annotations, "", s).Lang, "+") {
			if len(missingLangData(code)) == 0 && !slices.Contains(codes, code) {
				codes = append(codes, code)
			}
		}
	}
	slices.Sort(codes)
	return evidence.LangPerScene, strings.Join(codes, "+")
}
