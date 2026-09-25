package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The GUI's download endpoint is an entry point of its own: a code outside the catalogue is
// refused in the page's language and nothing is written (ticket 13, done criterion 1).
func TestHandleOCRDownloadRefusesATraversalCode(t *testing.T) {
	root := t.TempDir()
	// os.UserCacheDir reads these; the per-user tessdata folder would land under root.
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LocalAppData", root)
	t.Setenv("HOME", root)

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"lang":"../../x","uiLang":"de"}`)
	handleOCRDownload(rec, httptest.NewRequest(http.MethodPost, "/api/ocr-download", body))

	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
	}
	if resp.OK || !strings.Contains(resp.Error, "keine OCR-Sprache") {
		t.Errorf("response = %+v, want a refusal in German", resp)
	}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && p != root {
			t.Errorf("a refused code created %s", p)
		}
		return nil
	})
}
