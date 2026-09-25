package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/outputpath"
)

func postJSON(t *testing.T, h http.HandlerFunc, body map[string]any) map[string]any {
	t.Helper()
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(data))))
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	return resp
}

// A folder that merely shares the book's name and holds an index.html (a saved website,
// the user's own notes) used to be deleted by "delete previous result" with one confirm.
func TestDeleteOutputRefusesFolderItDoesNotOwn(t *testing.T) {
	dir := t.TempDir()
	site := filepath.Join(dir, "book")
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte("<html>my site</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := postJSON(t, handleDeleteOutput, map[string]any{"input": filepath.Join(dir, "book.pdf"), "output": ""})
	if resp["ok"] == true {
		t.Fatal("delete-output removed a folder the converter does not own")
	}
	if _, err := os.Stat(filepath.Join(site, "index.html")); err != nil {
		t.Fatalf("foreign folder damaged: %v", err)
	}
}

func TestDeleteOutputRemovesOwnResult(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "book.pdf")
	out := filepath.Join(dir, "book")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := outputpath.WriteMarker(out, input); err != nil {
		t.Fatal(err)
	}

	if resp := postJSON(t, handleDeleteOutput, map[string]any{"input": input, "output": ""}); resp["ok"] != true {
		t.Fatalf("own result not deleted: %v", resp)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output still present: %v", err)
	}
}

// The same result must not be reported for a different document that shares its name.
func TestOutputStatusIgnoresOtherDocumentsResult(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "book")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := outputpath.WriteMarker(out, filepath.Join(dir, "book.epub")); err != nil {
		t.Fatal(err)
	}

	resp := postJSON(t, handleOutputStatus, map[string]any{"input": filepath.Join(dir, "book.pdf")})
	if resp["exists"] == true {
		t.Fatalf("book.pdf reported book.epub's result: %v", resp)
	}
}

func dropBytes(t *testing.T, name, content string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	handleDrop(rec, httptest.NewRequest(http.MethodPost, "/api/drop?name="+name, strings.NewReader(content)))
	if rec.Code != http.StatusOK {
		t.Fatalf("drop status %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.Path
}

// Two different downloads both called "download.pdf" used to overwrite each other, and
// the second then reopened the first one's conversion.
func TestHandleDropKeepsSameNamedFilesApart(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	a := dropBytes(t, "download.pdf", "book A")
	b := dropBytes(t, "download.pdf", "book B")
	if a == b {
		t.Fatalf("different content saved to one path %q", a)
	}
	if data, _ := os.ReadFile(a); string(data) != "book A" {
		t.Fatalf("first drop overwritten: %q", data)
	}
	if again := dropBytes(t, "download.pdf", "book A"); again != a {
		t.Fatalf("re-dropping the same file moved it: %q vs %q", again, a)
	}
	parts, _ := filepath.Glob(filepath.Join(droppedFilesDir(), "*.part"))
	if len(parts) != 0 {
		t.Fatalf("temporary upload files left behind: %v", parts)
	}
}

func TestHandleDropRejectsDotDot(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	rec := httptest.NewRecorder()
	handleDrop(rec, httptest.NewRequest(http.MethodPost, "/api/drop?name=..", strings.NewReader("x")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// The GUI asks the output's own completion record, which the CLI wrote, whether the result was
// built with the settings now selected - not a history file of its own that a command-line run
// never updated.
func TestOutputStatusReadsCompletionRecord(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "book.epub")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "book")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := outputpath.WriteMarker(out, input); err != nil {
		t.Fatal(err)
	}
	req := map[string]any{
		"input": input, "noTranslate": true, "singlePage": true, "srcLang": "en", "dstLang": "ru",
		"ollamaModel": "gemma3:12b", "ollamaParallel": "1", "ollamaCtx": "8192",
	}
	// Unfinished: the CLI rebuilds it on its own, so the GUI does not ask.
	if resp := postJSON(t, handleOutputStatus, req); resp["exists"] != true || resp["paramsChanged"] != false {
		t.Fatalf("incomplete output: %v", resp)
	}

	cfg, err := config.ParseArgs(assembleArgs(runRequestFrom(t, req)))
	if err != nil {
		t.Fatal(err)
	}
	if err := outputpath.MarkComplete(out, input, outputpath.Completion{
		Options: outputpath.OptionsFor(cfg), Translation: outputpath.TranslationNone,
	}); err != nil {
		t.Fatal(err)
	}
	if resp := postJSON(t, handleOutputStatus, req); resp["paramsChanged"] != false {
		t.Fatalf("same settings reported as changed: %v", resp)
	}
	req["singlePage"] = false
	if resp := postJSON(t, handleOutputStatus, req); resp["paramsChanged"] != true {
		t.Fatalf("multipage after a single-page build not reported: %v", resp)
	}
}

func runRequestFrom(t *testing.T, m map[string]any) runRequest {
	t.Helper()
	data, _ := json.Marshal(m)
	var r runRequest
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r
}
