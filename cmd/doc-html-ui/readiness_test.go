package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"doc-html-translate/internal/readiness"
)

// The check itself and its tests live in internal/readiness (ticket E0146). What the window owns is the request it sends:
// its `/api/readiness` handler must hand every field the window knows to the shared check and answer exactly what the
// check answers, so the window and the command `input check` cannot disagree.
func TestReadinessHandlerAnswersTheSharedCheck(t *testing.T) {
	saved := readiness.Probes
	t.Cleanup(func() { readiness.Probes = saved })
	readiness.Probes.Calibre = func() bool { return false }
	readiness.Probes.SevenZip = func() bool { return true }
	readiness.Probes.FileDO = func() bool { return true }
	readiness.Probes.OCRLocate = func() (string, error) { return "tesseract", nil }
	readiness.Probes.OCRMissing = func(_ context.Context, _, lang string) []string { return []string{"asked:" + lang} }
	readiness.Probes.GoogleKey = func() (string, error) { return "", errors.New("missing") }
	readiness.Probes.Ollama = func(_ context.Context, model string) (bool, error) { return false, nil }

	file := func(ext string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "book"+ext)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for name, req := range map[string]runRequest{
		"mobi":         {Input: file(".mobi")},
		"ocr language": {Input: file(".epub"), OCR: true, OCRLang: "deu+fra"},
		"ocr from src": {Input: file(".epub"), OCR: true, SrcLang: "ru"},
		"google":       {Input: file(".pdf"), Google: true, MaxCost: "2.5"},
		"ollama":       {Input: file(".pdf"), Ollama: true, OllamaModel: "m1"},
		"no translate": {Input: file(".pdf"), Google: true, NoTranslate: true},
		"nothing":      {},
	} {
		body, _ := json.Marshal(req)
		rec := httptest.NewRecorder()
		handleReadiness(rec, httptest.NewRequest("POST", "/api/readiness", strings.NewReader(string(body))))
		var got readiness.Result
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: not JSON: %v: %s", name, err, rec.Body.String())
		}
		want := readiness.Check(context.Background(), req.readinessRequest())
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the window answered %+v, the shared check answers %+v", name, got, want)
		}
	}

	// A window request field reaches the check: the language of the OCR data it asks for is the one the window chose.
	var got readiness.Result
	body, _ := json.Marshal(runRequest{Input: file(".epub"), OCR: true, OCRLang: "deu+fra"})
	rec := httptest.NewRecorder()
	handleReadiness(rec, httptest.NewRequest("POST", "/api/readiness", strings.NewReader(string(body))))
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got.Issues) != 1 || got.Issues[0].Detail != "asked:deu+fra" {
		t.Errorf("the OCR language of the window did not reach the check: %+v", got)
	}

	rec = httptest.NewRecorder()
	handleReadiness(rec, httptest.NewRequest("POST", "/api/readiness", strings.NewReader("{")))
	if rec.Code != 400 {
		t.Errorf("a body that is not JSON answered %d, want 400", rec.Code)
	}
}
