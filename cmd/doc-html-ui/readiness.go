package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"doc-html-translate/internal/comic"
	"doc-html-translate/internal/img"
	"doc-html-translate/internal/mobi"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/pipeline"
	"doc-html-translate/internal/translator"
)

type readinessIssue struct {
	Code   string `json:"code"`
	Level  string `json:"level"` // action or optional
	Detail string `json:"detail,omitempty"`
}

type readinessResult struct {
	State       string           `json:"state"`
	Issues      []readinessIssue `json:"issues"`
	GoogleLimit string           `json:"googleLimit,omitempty"`
}

// Probes call the same discovery functions as conversion. Indirection keeps the readiness
// combinations testable without installing helpers or contacting a local service.
var readinessProbes = struct {
	calibre    func() bool
	sevenZip   func() bool
	ocrLocate  func() (string, error)
	ocrMissing func(context.Context, string, string) []string
	googleKey  func() (string, error)
	ollama     func(context.Context, string) (bool, error)
}{mobi.Available, comic.SevenZipAvailable, ocr.Locate, ocr.MissingLangs, translator.LoadGoogleAPIKey, translator.OllamaModelAvailable}

func checkReadiness(ctx context.Context, req runRequest) readinessResult {
	out := readinessResult{State: "ready", Issues: []readinessIssue{}}
	add := func(code, level, detail string) {
		out.Issues = append(out.Issues, readinessIssue{Code: code, Level: level, Detail: detail})
		if level == "action" {
			out.State = "action"
		} else if out.State == "ready" {
			out.State = "optional"
		}
	}
	if strings.TrimSpace(req.Input) == "" {
		return out
	}
	info, err := os.Stat(req.Input)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() || info.Size() == 0 {
		add("input", "action", "")
		return out
	}
	ext := strings.ToLower(filepath.Ext(req.Input))
	if (ext == ".mobi" || ext == ".azw3") && !readinessProbes.calibre() {
		add("calibre", "action", "")
	}
	if (ext == ".cbr" || ext == ".cb7") && !readinessProbes.sevenZip() {
		add("sevenzip", "action", "")
	}
	forcedOCR := img.IsImage(ext) || comic.IsComic(ext)
	if (req.OCR && pipeline.SupportsImageOCR(ext)) || forcedOCR {
		bin, err := readinessProbes.ocrLocate()
		if err != nil {
			add("tesseract", "optional", "")
		} else {
			lang := req.OCRLang
			if lang == "" {
				lang = ocr.TessLang(req.SrcLang)
			}
			if missing := readinessProbes.ocrMissing(ctx, bin, lang); len(missing) > 0 {
				add("ocrData", "optional", strings.Join(missing, ", "))
			}
		}
	}
	if !req.NoTranslate && req.Google {
		if _, err := readinessProbes.googleKey(); err != nil {
			add("googleKey", "optional", "")
		}
		out.GoogleLimit = "0"
		if f, err := strconv.ParseFloat(strings.TrimSpace(req.MaxCost), 64); err == nil && f > 0 && !math.IsInf(f, 0) {
			out.GoogleLimit = strconv.FormatFloat(f, 'f', -1, 64)
		}
	}
	if !req.NoTranslate && !req.Google && req.Ollama {
		model := strings.TrimSpace(req.OllamaModel)
		if model == "" {
			model = "gemma3:12b"
		}
		available, err := readinessProbes.ollama(ctx, model)
		if err != nil {
			add("ollamaService", "optional", "")
		} else if !available {
			add("ollamaModel", "optional", model)
		}
	}
	return out
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(checkReadiness(r.Context(), req))
}
