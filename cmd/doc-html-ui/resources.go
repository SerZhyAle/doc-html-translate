package main

import (
	"encoding/json"
	"net/http"

	"doc-html-translate/internal/readiness"
)

// resourceFacts is what the About page's resource index shows: whether each helper the program can
// use is found on this machine. The answers come from the same discovery functions the conversion
// uses (readiness.Probes), so the index and a real run never disagree, and the lookup is local - a
// path search and a file check. Opening About contacts nothing remote and sends nothing anywhere
// (SETTINGS-NAVIGATION 3.4); Ollama is deliberately absent, because the only honest answer is a
// request to its service, and that is a thing the reader's own run makes.
type resourceFacts struct {
	Tesseract bool `json:"tesseract"`
	Calibre   bool `json:"calibre"`
	SevenZip  bool `json:"sevenzip"`
	FileDO    bool `json:"filedo"`
}

func currentResources() resourceFacts {
	_, ocrErr := readiness.Probes.OCRLocate()
	return resourceFacts{
		Tesseract: ocrErr == nil,
		Calibre:   readiness.Probes.Calibre(),
		SevenZip:  readiness.Probes.SevenZip(),
		FileDO:    readiness.Probes.FileDO(),
	}
}

func handleResources(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(currentResources())
}
