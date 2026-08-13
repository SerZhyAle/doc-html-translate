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

	"doc-html-translate/internal/config"
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

// The GUI does not parse its own command line - it builds one for the CLI, and the CLI's parser
// is free to change under it. This runs the assembled vector through the real parser and checks
// every setting arrived, so a parser change that quietly drops a flag fails here instead of in a
// user's conversion. It is the GUI's half of the argument-order fix (DEV/plan, P41).
func TestAssembledArgsSurviveTheCLIParser(t *testing.T) {
	// Every field the window fills in, including the ones it always sends - assembleArgs
	// forwards those whenever they differ from the default, so leaving them empty here would
	// build a command line the window never produces.
	args := assembleArgs(runRequest{
		Input:          `C:\books\My Story.epub`,
		Output:         `D:\out folder`,
		OllamaModel:    "llama3",
		OllamaParallel: "2",
		OllamaCtx:      "4096",
		SplitSize:      "0",
		TOCDepth:       "3",
		MaxCost:        "5",
		SrcLang:        "ru",
		DstLang:        "en",
		SinglePage:     false,
		Force:          true,
		Verbose:        true,
		OCR:            true,
		OCRLang:        "rus",
		Google:         true,
	})

	cfg, err := config.ParseArgs(args)
	if err != nil {
		t.Fatalf("the GUI's own command line no longer parses: %v (args: %v)", err, args)
	}
	if cfg.InputFile != `C:\books\My Story.epub` {
		t.Errorf("InputFile = %q", cfg.InputFile)
	}
	if cfg.OutputFolder != `D:\out folder` {
		t.Errorf("OutputFolder = %q", cfg.OutputFolder)
	}
	if cfg.SplitSize != 0 || cfg.TOCDepth != 3 || cfg.MaxCost != 5 {
		t.Errorf("numbers dropped: split=%d toc=%d maxCost=%v", cfg.SplitSize, cfg.TOCDepth, cfg.MaxCost)
	}
	if cfg.OllamaModel != "llama3" || cfg.OllamaParallel != 2 || cfg.OllamaNumCtx != 4096 {
		t.Errorf("ollama settings dropped: model=%q parallel=%d ctx=%d", cfg.OllamaModel, cfg.OllamaParallel, cfg.OllamaNumCtx)
	}
	if cfg.SourceLang != "ru" || cfg.TargetLang != "en" || cfg.OCRLang != "rus" {
		t.Errorf("languages dropped: src=%q dst=%q ocr=%q", cfg.SourceLang, cfg.TargetLang, cfg.OCRLang)
	}
	if !cfg.OCR || !cfg.Force || !cfg.Verbose || !cfg.UseGoogle || cfg.SinglePage {
		t.Errorf("switches dropped: ocr=%v force=%v verbose=%v google=%v singlePage=%v",
			cfg.OCR, cfg.Force, cfg.Verbose, cfg.UseGoogle, cfg.SinglePage)
	}
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

// The OCR path is the GUI half of DEV/plan/2026-07-01_app-ocr-image-overlay.md criterion 4 ("GUI
// exposes the toggle + language buttons and drives the CLI"). It was a by-hand release check with
// no test behind it, so a silently dropped -ocr flag would have looked exactly like OCR being off.
func TestAssembleArgsForwardsOCRAndLanguage(t *testing.T) {
	args := assembleArgs(runRequest{
		Input:   `C:\books\comic.pdf`,
		OCR:     true,
		OCRLang: "rus",
		SrcLang: "en",
		DstLang: "ru",
	})
	if !slices.Contains(args, "-ocr") {
		t.Fatalf("expected -ocr to be forwarded, got %v", args)
	}
	assertFlagValue(t, args, "-ocr-lang", "rus")
}

func TestAssembleArgsOmitsOCRWhenToggleIsOff(t *testing.T) {
	args := assembleArgs(runRequest{
		Input:   `C:\books\comic.pdf`,
		OCRLang: "rus", // a remembered language must not enable the feature on its own
		SrcLang: "en",
		DstLang: "ru",
	})
	if slices.Contains(args, "-ocr") || slices.Contains(args, "-ocr-lang") {
		t.Fatalf("expected no OCR flags when the toggle is off, got %v", args)
	}
}

func TestHandleOCRLangsListsCatalogWithInstalledFlag(t *testing.T) {
	rec := httptest.NewRecorder()
	handleOCRLangs(rec, httptest.NewRequest(http.MethodGet, "/api/ocr-langs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var rows []struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Installed bool   `json:"installed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
	}
	if len(rows) == 0 {
		t.Fatal("catalog is empty - the GUI language pickers would render blank")
	}
	// English is bundled with the app, so it must be offered whatever else is installed. The
	// installed flag itself depends on the machine's tessdata, so it is not asserted here.
	for _, r := range rows {
		if r.Code == "eng" {
			if r.Name == "" {
				t.Error(`the "eng" row has no display name`)
			}
			return
		}
	}
	t.Errorf(`no "eng" row in the OCR catalog: %s`, rec.Body.String())
}

func TestHandleOCRLangsCarriesTheISOAlias(t *testing.T) {
	// The GUI derives the OCR language from the -src language through this alias. Without it
	// the page would need its own copy of ocr.iso2tess, which is exactly how the two drift.
	rec := httptest.NewRecorder()
	handleOCRLangs(rec, httptest.NewRequest(http.MethodGet, "/api/ocr-langs", nil))
	var rows []struct {
		Code string `json:"code"`
		ISO  string `json:"iso"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
	}
	for _, r := range rows {
		if r.Code == "rus" {
			if r.ISO != "ru" {
				t.Errorf(`"rus" row carries iso %q, want "ru"`, r.ISO)
			}
			return
		}
	}
	t.Errorf(`no "rus" row in the OCR catalog: %s`, rec.Body.String())
}

func TestUIFollowsTheSourceLanguageForOCR(t *testing.T) {
	// The GUI always sends an explicit -ocr-lang, so a stale pick silently OCRs a page in the
	// wrong language and finds nothing. Every path that changes the source must resync.
	for _, snippet := range []string{
		"function syncOcrLangToSource()",
		"document.getElementById('srcLang').addEventListener('change', syncOcrLangToSource)",
	} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q - the OCR language would not follow -src", snippet)
		}
	}
	if strings.Count(uiHTML, "syncOcrLangToSource()") < 4 {
		t.Error("syncOcrLangToSource is not called from all of: definition, load, swap, settings-applied")
	}
}

func TestUIMarkupExposesOCRControls(t *testing.T) {
	// Guards the other half of criterion 4: the flags above are unreachable if the section that
	// sets them is not in the page.
	for _, id := range []string{`id="chkOCR"`, `id="ocrLang"`, `id="ocrDownloadSel"`, `id="btnOcrDownload"`} {
		if !strings.Contains(uiHTML, id) {
			t.Errorf("ui.html has no %s - the GUI cannot drive OCR without it", id)
		}
	}
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
