package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"doc-html-translate/internal/translator"
)

func TestAssembleArgsPassesExplicitSplitZero(t *testing.T) {
	args := assembleArgs(runRequest{
		Input:     `C:\books\story.epub`,
		SplitSize: "0",
		SrcLang:   "en",
		DstLang:   "ru",
	})

	if !slices.Contains(args, "-split") {
		t.Fatalf("expected -split to be passed when GUI explicitly sends 0, got %v", args)
	}

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-split" && args[i+1] == "0" {
			return
		}
	}
	t.Fatalf("expected -split 0 in args, got %v", args)
}

func assertFlagValue(t *testing.T, args []string, flag, val string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == val {
			return
		}
	}
	t.Fatalf("expected %s %s in args, got %v", flag, val, args)
}

func TestAssembleArgsForwardsTOCDepthAndMaxCost(t *testing.T) {
	args := assembleArgs(runRequest{
		Input:    `C:\books\story.epub`,
		Google:   true,
		TOCDepth: "1",
		MaxCost:  "2",
		SrcLang:  "en",
		DstLang:  "ru",
	})
	assertFlagValue(t, args, "-toc-depth", "1")
	assertFlagValue(t, args, "-max-cost", "2")
}

func TestAssembleArgsOmitsDefaultTOCDepthAndMaxCost(t *testing.T) {
	// 0 is the CLI default for both (unlimited TOC / no cost limit), so the GUI
	// should not clutter the command line with them.
	args := assembleArgs(runRequest{
		Input:    `C:\books\story.epub`,
		TOCDepth: "0",
		MaxCost:  "0",
		SrcLang:  "en",
		DstLang:  "ru",
	})
	if slices.Contains(args, "-toc-depth") {
		t.Fatalf("did not expect -toc-depth for default 0, got %v", args)
	}
	if slices.Contains(args, "-max-cost") {
		t.Fatalf("did not expect -max-cost for default 0, got %v", args)
	}
}

func TestSaveGoogleAPIKeyRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)

	want := filepath.Join(tmp, "doc-html-translate", "google_api.key")
	if got := writableGoogleKeyPath(); got != want {
		t.Fatalf("writableGoogleKeyPath() = %q, want %q", got, want)
	}

	if err := saveGoogleAPIKey("  AIzaSyTEST123  "); err != nil {
		t.Fatalf("saveGoogleAPIKey: %v", err)
	}

	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read saved key: %v", err)
	}
	if string(data) != "AIzaSyTEST123" {
		t.Fatalf("saved key = %q, want trimmed %q", string(data), "AIzaSyTEST123")
	}

	key, err := translator.LoadGoogleAPIKey()
	if err != nil {
		t.Fatalf("LoadGoogleAPIKey after save: %v", err)
	}
	if key != "AIzaSyTEST123" {
		t.Fatalf("LoadGoogleAPIKey = %q, want %q", key, "AIzaSyTEST123")
	}
}

func TestSaveGoogleAPIKeyRejectsEmpty(t *testing.T) {
	if err := saveGoogleAPIKey("   "); err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}

// The only thing between "open the result" and shell-executing an arbitrary path is the
// index.html check: the request carries an input and an output folder, both of which the
// page can set. Refusing a directory that holds no result is what keeps a stray (or
// crafted) Output value from steering the app into opening something unrelated.
func TestOpenOutputRefusesDirWithoutIndexHTML(t *testing.T) {
	dir := t.TempDir()
	body, err := json.Marshal(map[string]any{
		"input":  filepath.Join(dir, "book.pdf"),
		"output": "",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/open-output", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	handleOpenOutput(rec, req)

	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	if resp.OK {
		t.Fatal("open-output accepted a directory with no index.html")
	}
	if resp.Error == "" {
		t.Error("refusal carried no reason")
	}
}

// A real result is opened, and "folder" decides whether that means the page or the
// directory holding it - the answer to "where did my conversion go?".
func TestOpenOutputOpensExistingResult(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "book")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	index := filepath.Join(outDir, "index.html")
	if err := os.WriteFile(index, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var opened string
	prev := openTarget
	openTarget = func(target string) error { opened = target; return nil }
	t.Cleanup(func() { openTarget = prev })

	for _, tc := range []struct {
		folder bool
		want   string
	}{
		{false, index},
		{true, outDir},
	} {
		opened = ""
		// resolveOutputDir maps input "<dir>/book.pdf" with no explicit output folder
		// onto "<dir>/book" - the directory populated above.
		body, err := json.Marshal(map[string]any{
			"input":  filepath.Join(dir, "book.pdf"),
			"output": "",
			"folder": tc.folder,
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/open-output", strings.NewReader(string(body)))
		rec := httptest.NewRecorder()
		handleOpenOutput(rec, req)

		var resp struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
		}
		if !resp.OK {
			t.Fatalf("folder=%v: existing result refused: %s", tc.folder, resp.Error)
		}
		if opened != tc.want {
			t.Errorf("folder=%v: opened %q, want %q", tc.folder, opened, tc.want)
		}
	}
}

func TestDroppedFilesDirUsesLocalAppData(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	want := filepath.Join(tmp, "doc-html-translate", "dropped")
	if got := droppedFilesDir(); got != want {
		t.Fatalf("droppedFilesDir() = %q, want %q", got, want)
	}
}

func TestHandleDropSavesUploadedFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)

	// A non-ASCII name exercises the URL-encoded query path the GUI uses.
	req := httptest.NewRequest(http.MethodPost, "/api/drop?name=%D0%9A%D0%BD%D0%B8%D0%B3%D0%B0.epub", strings.NewReader("EPUB-BYTES"))
	rec := httptest.NewRecorder()
	handleDrop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := filepath.Join(tmp, "doc-html-translate", "dropped", "Книга.epub")
	if resp.Path != want {
		t.Fatalf("saved path = %q, want %q", resp.Path, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "EPUB-BYTES" {
		t.Fatalf("saved bytes = %q, want %q", string(data), "EPUB-BYTES")
	}
}

func TestHandleDropStripsPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)

	req := httptest.NewRequest(http.MethodPost, "/api/drop?name=..%2F..%2Fevil.epub", strings.NewReader("x"))
	rec := httptest.NewRecorder()
	handleDrop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	want := filepath.Join(tmp, "doc-html-translate", "dropped", "evil.epub")
	if resp.Path != want {
		t.Fatalf("path = %q, want %q (traversal must be stripped to a base name)", resp.Path, want)
	}
}

func TestDecodeDialogPathRoundTripsCyrillic(t *testing.T) {
	want := `C:\Users\serzh\OneDrive\Документы\file.pdf`
	enc := base64.StdEncoding.EncodeToString([]byte(want))

	// The dialog scripts print base64 with a trailing newline; decodeDialogPath trims it.
	got, err := decodeDialogPath("  " + enc + "\r\n")
	if err != nil {
		t.Fatalf("decodeDialogPath: %v", err)
	}
	if got != want {
		t.Fatalf("decodeDialogPath = %q, want %q", got, want)
	}
}

func TestDecodeDialogPathEmptyMeansCancelled(t *testing.T) {
	got, err := decodeDialogPath("  \r\n")
	if err != nil || got != "" {
		t.Fatalf("decodeDialogPath(empty) = %q, %v; want \"\", nil", got, err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)

	// No file yet → GET returns an empty object, not an error.
	rec := httptest.NewRecorder()
	handleSettings(rec, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "{}" {
		t.Fatalf("initial GET = %d %q, want 200 {}", rec.Code, rec.Body.String())
	}

	// POST saves the blob.
	body := `{"engine":"google","srcLang":"en","dstLang":"ru"}`
	rec = httptest.NewRecorder()
	handleSettings(rec, httptest.NewRequest(http.MethodPost, "/api/settings", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST = %d, body %s", rec.Code, rec.Body.String())
	}

	// GET returns exactly what was saved.
	rec = httptest.NewRecorder()
	handleSettings(rec, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if got := strings.TrimSpace(rec.Body.String()); got != body {
		t.Fatalf("GET after save = %q, want %q", got, body)
	}
}

func TestSettingsRejectsInvalidJSON(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	rec := httptest.NewRecorder()
	handleSettings(rec, httptest.NewRequest(http.MethodPost, "/api/settings", strings.NewReader("not json")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST invalid = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDropRejectsMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader("x"))
	rec := httptest.NewRecorder()
	handleDrop(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
