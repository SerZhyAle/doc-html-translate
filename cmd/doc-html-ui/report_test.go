package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"doc-html-translate/internal/report"
)

// seedLog puts one run log in the store so a report has something to carry.
func seedLog(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(report.LogsDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	p := report.RunLogPath(time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC))
	if err := os.WriteFile(p, []byte("[14:00:00] doc-html-translate test\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
}

func decodeResp(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return got
}

func TestHandleReportWritesArchive(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	seedLog(t)

	rec := httptest.NewRecorder()
	handleReport(rec, httptest.NewRequest(http.MethodPost, "/api/report", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/report = %d, body %s", rec.Code, rec.Body.String())
	}

	got := decodeResp(t, rec)
	if got["ok"] != true {
		t.Fatalf("ok = %v, error = %v", got["ok"], got["error"])
	}
	path, _ := got["path"].(string)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive %q not written: %v", path, err)
	}
	if bytes, _ := got["bytes"].(float64); bytes <= 0 {
		t.Errorf("bytes = %v, want the archive's size", got["bytes"])
	}
	if _, ok := got["dropped"]; !ok {
		t.Error("response carries no dropped count")
	}
}

func TestHandleReportRejectsGET(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	rec := httptest.NewRecorder()
	handleReport(rec, httptest.NewRequest(http.MethodGet, "/api/report", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /api/report = %d, want 405", rec.Code)
	}
}

func TestHandleReportRevealRefusesPathOutsideReportDir(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	outside := filepath.Join(t.TempDir(), "elsewhere.zip")
	revealed := ""
	saved := revealInExplorer
	revealInExplorer = func(path string) error {
		revealed = path
		return nil
	}
	t.Cleanup(func() { revealInExplorer = saved })

	body := `{"path":` + jsonString(outside) + `}`
	rec := httptest.NewRecorder()
	handleReportReveal(rec, httptest.NewRequest(http.MethodPost, "/api/report-reveal", strings.NewReader(body)))

	got := decodeResp(t, rec)
	if got["ok"] != false {
		t.Fatalf("ok = %v, want false for a path outside the report folder", got["ok"])
	}
	if revealed != "" {
		t.Fatalf("a process was started for %q", revealed)
	}

	// The same handler accepts a path the app itself wrote.
	inside := filepath.Join(report.Dir(), "reports", "report.zip")
	rec = httptest.NewRecorder()
	handleReportReveal(rec, httptest.NewRequest(http.MethodPost, "/api/report-reveal", strings.NewReader(`{"path":`+jsonString(inside)+`}`)))
	if got := decodeResp(t, rec); got["ok"] != true {
		t.Fatalf("ok = %v for a path inside the report folder, error = %v", got["ok"], got["error"])
	}
	if revealed != inside {
		t.Fatalf("revealed %q, want %q", revealed, inside)
	}
}

func TestHandleLogsClearEmptiesTheStore(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	seedLog(t)

	rec := httptest.NewRecorder()
	handleLogsClear(rec, httptest.NewRequest(http.MethodPost, "/api/logs-clear", nil))
	if got := decodeResp(t, rec); got["ok"] != true {
		t.Fatalf("ok = %v, error = %v", got["ok"], got["error"])
	}
	left, err := os.ReadDir(report.LogsDir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("%d logs left after /api/logs-clear, want 0", len(left))
	}
}

// jsonString quotes a Windows path so it survives as a JSON string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
