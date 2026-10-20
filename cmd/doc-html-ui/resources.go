package main

import (
	"encoding/json"
	"net/http"
)

// resourceFacts is what the About page's resource index shows: whether each helper the program can
// use is found on this machine. The answers come from the same discovery functions the conversion
// uses (readinessProbes), so the index and a real run never disagree, and the lookup is local - a
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
	_, ocrErr := readinessProbes.ocrLocate()
	return resourceFacts{
		Tesseract: ocrErr == nil,
		Calibre:   readinessProbes.calibre(),
		SevenZip:  readinessProbes.sevenZip(),
		FileDO:    readinessProbes.filedo(),
	}
}

func handleResources(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(currentResources())
}
