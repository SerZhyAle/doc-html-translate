package main

import (
	"encoding/json"
	"net/http"

	"doc-html-translate/internal/readiness"
)

// readinessRequest is what the window's conversion request tells the shared check (internal/readiness). The check itself
// - its probes, its issues, its states - lives there, so the window and the command `input check` give one answer.
func (r runRequest) readinessRequest() readiness.Request {
	return readiness.Request{
		Input:       r.Input,
		NoTranslate: r.NoTranslate,
		Google:      r.Google,
		Ollama:      r.Ollama,
		OllamaModel: r.OllamaModel,
		MaxCost:     r.MaxCost,
		SrcLang:     r.SrcLang,
		OCR:         r.OCR,
		OCRLang:     r.OCRLang,
	}
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(readiness.Check(r.Context(), req.readinessRequest()))
}
