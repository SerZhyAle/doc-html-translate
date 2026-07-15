package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/syslocale"
	"doc-html-translate/internal/translator"
	"doc-html-translate/internal/windowsreg"
)

//go:embed ui.html
var uiHTML string

//go:embed favicon.ico
var faviconICO []byte

// Version is set at build time via -ldflags.
var Version = "dev"

const cliName = "doc-html-translate.exe"

// maxDropBytes caps an uploaded (dropped) document. The window runs in a plain
// browser where a dropped file has no filesystem path, so its bytes are uploaded
// instead; the cap is a sanity backstop, not a real limit for any ebook/PDF.
const maxDropBytes = 2 << 30 // 2 GiB

var initialFile string

func main() {
	if len(os.Args) > 1 {
		initialFile = os.Args[1]
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	addr := ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleUI)
	mux.HandleFunc("/favicon.ico", handleFavicon)
	mux.HandleFunc("/api/version", handleVersion)
	mux.HandleFunc("/api/initial", handleInitial)
	mux.HandleFunc("/api/ping", handlePing)
	mux.HandleFunc("/api/browse-file", handleBrowseFile)
	mux.HandleFunc("/api/browse-folder", handleBrowseFolder)
	mux.HandleFunc("/api/drop", handleDrop)
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/google-key", handleGoogleKey)
	mux.HandleFunc("/api/preview", handlePreview)
	mux.HandleFunc("/api/output-status", handleOutputStatus)
	mux.HandleFunc("/api/delete-output", handleDeleteOutput)
	mux.HandleFunc("/api/run", handleRun)
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/unregister", handleUnregister)
	mux.HandleFunc("/api/assoc-status", handleAssocStatus)
	mux.HandleFunc("/api/env", handleEnv)
	mux.HandleFunc("/api/ocr-langs", handleOCRLangs)
	mux.HandleFunc("/api/ocr-download", handleOCRDownload)

	srv := &http.Server{Handler: mux}

	go watchHeartbeat(srv)
	go openAppWindow("http://" + addr)
	go ensureRightClickRegistered()

	_ = srv.Serve(ln)
}

// ensureRightClickRegistered advertises the app in Windows' "Open with" list AND adds
// the "Convert to HTML" right-click verb on every launch, so users find it there without
// making it the default handler. Both are non-destructive - they never change the default
// handler - and a no-op under MSIX (the package manifest already declares the associations)
// or when the bundled CLI cannot be located. The associations point at the CLI so the
// right-click entry converts and opens the result, matching double-click behavior.
func ensureRightClickRegistered() {
	if isPackaged() || !cliAvailable() {
		return
	}
	cli := findCLI()
	_, _ = windowsreg.RegisterOpenWithFor(cli)
	_, _ = windowsreg.RegisterContextMenuFor(cli)
}

// ── HTTP handlers ───────────────────────────────────────────

var lastPing atomic.Int64

// activeRuns counts in-flight conversions. The heartbeat watchdog must not shut the
// server down while one is running, even if the browser's ping lapses (its main
// thread can stall while rendering a long log), or a long conversion gets cut off
// mid-stream - which surfaces in the UI as a bare "network error".
var activeRuns atomic.Int64

func handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, uiHTML)
}

// handleFavicon serves the project icon so the app window shows our logo instead of the
// browser's default globe (the GUI runs as a Chrome --app window rendering this page).
func handleFavicon(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	_, _ = w.Write(faviconICO)
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"version": Version})
}

func handleInitial(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"file": initialFile})
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	lastPing.Store(time.Now().Unix())
	w.WriteHeader(http.StatusOK)
}

func handleBrowseFile(w http.ResponseWriter, _ *http.Request) {
	path, err := browseFile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": path})
}

func handleBrowseFolder(w http.ResponseWriter, _ *http.Request) {
	path, err := browseFolder()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": path})
}

// handleDrop saves the bytes of a file dropped onto the app window to a per-user
// folder and returns the resulting path. The GUI runs in a plain browser window
// (Edge/Chrome --app mode), where a dropped file exposes no filesystem path, so the
// page uploads the raw bytes here and we materialize them for the CLI to read. The
// destination lives under %LOCALAPPDATA% (writable even under the read-only MSIX/Store
// install) so the converted output, which lands next to the input, persists too.
func handleDrop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// The browser only ever sends a bare file name; Base strips any stray path
	// components or traversal so we cannot be steered outside the drop folder.
	name := filepath.Base(filepath.FromSlash(strings.TrimSpace(r.URL.Query().Get("name"))))
	if name == "" || name == "." || name == string(os.PathSeparator) {
		http.Error(w, "missing or invalid filename", http.StatusBadRequest)
		return
	}

	dir := droppedFilesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dest := filepath.Join(dir, name)
	f, err := os.Create(dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(f, http.MaxBytesReader(w, r.Body, maxDropBytes)); err != nil {
		_ = f.Close()
		http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := f.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": dest})
}

// droppedFilesDir is the writable, per-user folder where files dropped onto the
// window are saved before conversion. It mirrors the Google-key location's choice of
// %LOCALAPPDATA% so it keeps working under the read-only MSIX/Store install.
func droppedFilesDir() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "doc-html-translate", "dropped")
	}
	return filepath.Join(os.TempDir(), "doc-html-translate-dropped")
}

// handleSettings persists the GUI's form state across sessions. The browser's own
// localStorage cannot: the server binds a random port each launch, so the page origin
// (and thus the store) differs every time. We keep the raw JSON blob the page sends -
// its shape is the page's business - under %LOCALAPPDATA% so it survives restarts and
// the read-only MSIX/Store install.
//
//	GET  → the saved JSON (or {} if none yet)
//	POST → save the request body (must be valid JSON, capped at 64 KiB)
func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		data, err := os.ReadFile(settingsPath())
		if err != nil || !json.Valid(data) {
			_, _ = io.WriteString(w, "{}")
			return
		}
		_, _ = w.Write(data)
	case http.MethodPost:
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !json.Valid(data) {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		p := settingsPath()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(p, data, 0o600); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// settingsPath is the writable, per-user location of the saved GUI state.
func settingsPath() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "doc-html-translate", "ui-settings.json")
	}
	return filepath.Join(os.TempDir(), "doc-html-translate-ui-settings.json")
}

// handleGoogleKey lets the GUI inspect and save the Google Translate API key.
//
//	GET  → {"exists": bool, "path": "<writable per-user path>"}
//	POST → {"key": "..."} saves the key to the writable per-user location and
//	       returns {"exists": bool, "path": "..."}.
//
// The key is stored at %LOCALAPPDATA%\doc-html-translate\google_api.key so it
// also works under the read-only Microsoft Store (MSIX) install directory.
func handleGoogleKey(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := saveGoogleAPIKey(req.Key); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodGet:
		// fallthrough to status response below
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := translator.LoadGoogleAPIKey()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"exists": err == nil,
		"path":   writableGoogleKeyPath(),
	})
}

// writableGoogleKeyPath returns the per-user, writable key location. It mirrors
// translator.GoogleAPIKeyPaths by picking the %LOCALAPPDATA% candidate (the only
// one guaranteed writable under MSIX); falls back to the last candidate.
func writableGoogleKeyPath() string {
	paths := translator.GoogleAPIKeyPaths()
	for _, p := range paths {
		if appData := os.Getenv("LOCALAPPDATA"); appData != "" && strings.HasPrefix(p, appData) {
			return p
		}
	}
	if len(paths) > 0 {
		return paths[len(paths)-1]
	}
	return ""
}

func saveGoogleAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("key is empty")
	}
	path := writableGoogleKeyPath()
	if path == "" {
		return fmt.Errorf("cannot determine a writable key location")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create key folder: %w", err)
	}
	if err := os.WriteFile(path, []byte(key), 0o600); err != nil {
		return fmt.Errorf("write key file: %w", err)
	}
	return nil
}

type runRequest struct {
	Input          string `json:"input"`
	Output         string `json:"output"`
	NoTranslate    bool   `json:"noTranslate"`
	NoOpen         bool   `json:"noOpen"`
	Google         bool   `json:"google"`
	Ollama         bool   `json:"ollama"`
	OllamaModel    string `json:"ollamaModel"`
	OllamaParallel string `json:"ollamaParallel"`
	OllamaCtx      string `json:"ollamaCtx"`
	SplitSize      string `json:"splitSize"`
	TOCDepth       string `json:"tocDepth"`
	SinglePage     bool   `json:"singlePage"`
	MaxCost        string `json:"maxCost"`
	SrcLang        string `json:"srcLang"`
	DstLang        string `json:"dstLang"`
	Force          bool   `json:"force"`
	Verbose        bool   `json:"verbose"`
	OCR            bool   `json:"ocr"`
	OCRLang        string `json:"ocrLang"`
}

// handlePreview returns the exact command line the current settings would run,
// so the GUI can show it live next to a Copy button. It reuses assembleArgs and
// findCLI so the preview always matches what handleRun actually executes.
func handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	cmd := formatCommandLine(findCLI(), assembleArgs(req))
	_ = json.NewEncoder(w).Encode(map[string]string{"cmd": cmd})
}

// resolveOutputDir mirrors internal/pipeline's reuse-check target directory so the
// GUI can look at (and clear) a previous result without shelling out to the CLI.
func resolveOutputDir(input, folder string) (string, error) {
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	return outputpath.OutputDirFor(abs, folder), nil
}

// outputFingerprint captures the runRequest fields that change what actually gets
// written to the output directory - as opposed to execution-only flags like Force,
// Verbose, NoOpen, or OllamaParallel - so paramsHistory can detect "this result was
// built with different options than what's selected now".
type outputFingerprint struct {
	SinglePage  bool
	SplitSize   string
	TOCDepth    string
	MaxCost     string
	SrcLang     string
	DstLang     string
	NoTranslate bool
	Google      bool
	Ollama      bool
	OllamaModel string
	OllamaCtx   string
	OCR         bool
	OCRLang     string
}

func fingerprintFor(req runRequest) string {
	data, _ := json.Marshal(outputFingerprint{
		SinglePage:  req.SinglePage,
		SplitSize:   req.SplitSize,
		TOCDepth:    req.TOCDepth,
		MaxCost:     req.MaxCost,
		SrcLang:     req.SrcLang,
		DstLang:     req.DstLang,
		NoTranslate: req.NoTranslate,
		Google:      req.Google,
		Ollama:      req.Ollama,
		OllamaModel: req.OllamaModel,
		OllamaCtx:   req.OllamaCtx,
		OCR:         req.OCR,
		OCRLang:     req.OCRLang,
	})
	return string(data)
}

// paramsHistoryPath is the writable, per-user location of the fingerprint each output
// directory was last built with, so a later run with different options can be told
// apart from a plain re-run of the same settings.
func paramsHistoryPath() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "doc-html-translate", "output-params.json")
	}
	return filepath.Join(os.TempDir(), "doc-html-translate-output-params.json")
}

func loadParamsHistory() map[string]string {
	data, err := os.ReadFile(paramsHistoryPath())
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

func saveParamsHistory(m map[string]string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	p := paramsHistoryPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// handleOutputStatus reports whether a previous conversion result exists for the
// request's input/output, and whether it was built with different options than the
// request currently describes - so the GUI can offer to delete and rebuild instead of
// silently reopening a stale result (mirrors the CLI's reuse-unless--force rule).
func handleOutputStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp := map[string]any{"exists": false, "paramsChanged": false}
	if req.Input == "" {
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	outputDir, err := resolveOutputDir(req.Input, req.Output)
	if err != nil {
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if _, err := os.Stat(filepath.Join(outputDir, "index.html")); err != nil {
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp["exists"] = true
	resp["outputDir"] = outputDir
	if prev, ok := loadParamsHistory()[outputDir]; ok && prev != fingerprintFor(req) {
		resp["paramsChanged"] = true
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleDeleteOutput removes a previous conversion result so the next run starts
// clean. It only ever deletes a directory that actually contains index.html - the
// same marker the CLI's reuse check uses - so a stray output-folder value can't steer
// it into deleting something unrelated.
func handleDeleteOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	outputDir, err := resolveOutputDir(req.Input, req.Output)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if _, err := os.Stat(filepath.Join(outputDir, "index.html")); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "no previous result found"})
		return
	}
	if err := os.RemoveAll(outputDir); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if hist := loadParamsHistory(); len(hist) > 0 {
		if _, ok := hist[outputDir]; ok {
			delete(hist, outputDir)
			_ = saveParamsHistory(hist)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	activeRuns.Add(1)
	defer activeRuns.Add(-1)

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	args := assembleArgs(req)
	bin := findCLI()

	fmt.Fprintf(w, "> %s\n\n", formatCommandLine(bin, args))
	flusher.Flush()

	cmd := exec.Command(bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(w, "[ERROR] stdout pipe: %v\n", err)
		flusher.Flush()
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(w, "[ERROR] stderr pipe: %v\n", err)
		flusher.Flush()
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "[ERROR] start: %v\n", err)
		flusher.Flush()
		return
	}

	var wg sync.WaitGroup
	stream := func(rd io.Reader, prefix string) {
		defer wg.Done()
		sc := bufio.NewScanner(rd)
		for sc.Scan() {
			fmt.Fprintf(w, "%s%s\n", prefix, sc.Text())
			flusher.Flush()
		}
	}
	wg.Add(2)
	go stream(stdout, "")
	go stream(stderr, "[err] ")
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(w, "\nExit: %v\n", err)
	} else {
		fmt.Fprintf(w, "\nDone.\n")
		if outputDir, err := resolveOutputDir(req.Input, req.Output); err == nil && outputDir != "" {
			hist := loadParamsHistory()
			hist[outputDir] = fingerprintFor(req)
			_ = saveParamsHistory(hist)
		}
	}
	flusher.Flush()
}

// handleRegister sets this app as the default Windows handler for the supported
// document types (the opt-in association, off by default). It shells out to the bundled
// CLI's -register flow so the file associations point at doc-html-translate.exe (the
// headless converter) rather than at the GUI - matching the unpackaged double-click
// behavior. Under an MSIX/Store install this is a no-op (HKCU is virtualized); the GUI
// hides the association toggle there.
func handleRegister(w http.ResponseWriter, _ *http.Request) {
	bin := findCLI()
	cmd := exec.Command(bin, "-register")
	// The CLI's -register flow prints a splash and waits on Scanln; feed a newline
	// so it returns immediately instead of blocking this request.
	cmd.Stdin = strings.NewReader("\n")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	resp := map[string]any{"ok": err == nil}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		resp["error"] = msg
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleUnregister releases the default-handler association (the "off" side of the
// association toggle), leaving the non-destructive "Convert to HTML" right-click verb and
// "Open with" entry in place. Deletion only clears HKCU values that point at our ProgID,
// so it needs no CLI round-trip and does not depend on which exe wrote them.
func handleUnregister(w http.ResponseWriter, _ *http.Request) {
	_, err := windowsreg.Unregister()
	resp := map[string]any{"ok": err == nil}
	if err != nil {
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleAssocStatus reports whether this app is currently the default handler for the
// supported types, so the GUI can reflect the association toggle's on/off state and decide
// whether to show the one-time first-run opt-in prompt.
func handleAssocStatus(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{"default": windowsreg.IsDefaultHandler()})
}

// handleEnv reports environment facts the GUI adapts to on load:
//
//	packaged → running from inside an MSIX/Store package, where -register is a no-op
//	           (file associations come from the package manifest instead).
//	cli      → the bundled converter exe is resolvable next to the app or on PATH;
//	           without it, Convert cannot run.
//	lang     → the Windows UI language ("ru"/"uk"/"en"); the GUI opens in it by
//	           default, unless the user has picked a language before.
func handleEnv(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"packaged": isPackaged(),
		"cli":      cliAvailable(),
		"lang":     syslocale.Lang(),
	})
}

// handleOCRLangs reports the OCR language catalog with an installed flag, so the GUI can
// show which languages are ready and which need downloading.
func handleOCRLangs(w http.ResponseWriter, _ *http.Request) {
	installed := map[string]bool{}
	for _, c := range ocr.Installed() {
		installed[c] = true
	}
	type row struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Installed bool   `json:"installed"`
	}
	out := make([]row, 0, len(ocr.Available))
	for _, l := range ocr.Available {
		out = append(out, row{Code: l.Code, Name: l.Name, Installed: installed[l.Code]})
	}
	_ = json.NewEncoder(w).Encode(out)
}

// handleOCRDownload downloads a single OCR language pack on request from the GUI.
func handleOCRDownload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Lang string `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp := map[string]any{"ok": true}
	if err := ocr.Download(req.Lang); err != nil {
		resp["ok"] = false
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ── args assembly ───────────────────────────────────────────

func assembleArgs(req runRequest) []string {
	var a []string
	if req.NoTranslate {
		a = append(a, "-notranslate")
	}
	if req.NoOpen {
		a = append(a, "-noopen")
	}
	if req.Google {
		a = append(a, "-google")
	}
	if req.Ollama {
		a = append(a, "-ollama")
	}
	if req.OllamaModel != "gemma3:12b" {
		a = append(a, "-ollama-model", req.OllamaModel)
	}
	if req.OllamaParallel != "1" {
		a = append(a, "-ollama-parallel", req.OllamaParallel)
	}
	if req.OllamaCtx != "8192" {
		a = append(a, "-ollama-ctx", req.OllamaCtx)
	}
	if req.SplitSize != "" {
		a = append(a, "-split", req.SplitSize)
	}
	// toc-depth/max-cost default to 0 (unlimited / no limit) in the CLI, so only
	// forward them when the user picked a non-default value - keeps the command line clean.
	if req.TOCDepth != "" && req.TOCDepth != "0" {
		a = append(a, "-toc-depth", req.TOCDepth)
	}
	if !req.SinglePage {
		a = append(a, "-multipage")
	}
	if req.MaxCost != "" && req.MaxCost != "0" {
		a = append(a, "-max-cost", req.MaxCost)
	}
	a = append(a, "-src", req.SrcLang, "-dst", req.DstLang)
	if req.OCR {
		a = append(a, "-ocr")
		if req.OCRLang != "" {
			a = append(a, "-ocr-lang", req.OCRLang)
		}
	}
	if req.Force {
		a = append(a, "-force")
	}
	if req.Verbose {
		a = append(a, "-v")
	}
	if req.Output != "" {
		a = append(a, "-folder", req.Output)
	}
	if req.Input != "" {
		a = append(a, req.Input)
	}
	return a
}

// formatCommandLine joins a binary and its args into a single copy-pasteable command
// line, quoting any token that contains spaces (Windows paths routinely do) so the
// result stays runnable in a terminal. This is also what the run log's header shows,
// so the live preview matches the command that actually executes.
func formatCommandLine(bin string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteArg(bin))
	for _, a := range args {
		parts = append(parts, quoteArg(a))
	}
	return strings.Join(parts, " ")
}

func quoteArg(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, " \t\"") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

// ── find CLI binary ─────────────────────────────────────────

func findCLI() string {
	if exe, err := os.Executable(); err == nil {
		c := filepath.Join(filepath.Dir(exe), cliName)
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath(cliName); err == nil {
		return p
	}
	return cliName
}

// cliAvailable reports whether the bundled converter exe can actually be located
// (next to this app or on PATH). findCLI always returns a bare name as a last resort,
// so it cannot answer this on its own.
func cliAvailable() bool {
	if exe, err := os.Executable(); err == nil {
		if _, err := os.Stat(filepath.Join(filepath.Dir(exe), cliName)); err == nil {
			return true
		}
	}
	_, err := exec.LookPath(cliName)
	return err == nil
}

// isPackaged reports whether the app runs from inside an MSIX/Store package.
// Such installs live under ...\WindowsApps\... and virtualize HKCU writes, so the
// -register flow is a no-op there (associations come from the package manifest).
func isPackaged() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(exe), `\windowsapps\`)
}

// ── native file/folder dialogs via PowerShell ───────────────

// The dialogs run from a hidden PowerShell process that owns no window. A dialog with
// no owner (or owned by an off-screen helper form) is not topmost, so it opens behind
// our active app window, and a background process is denied SetForegroundWindow to fix
// that. The reliable cure is to own the dialog to the *current foreground window* -
// our DOC-HTML-UI (Edge --app) window, which still has focus from the click that
// triggered the browse. An owned dialog always sits above its owner, so it lands in
// front. dialogOwner compiles a tiny IWin32Window wrapper around GetForegroundWindow()
// and leaves the handle in $owner.
const dialogOwner = `Add-Type -ReferencedAssemblies System.Windows.Forms -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Windows.Forms;
public class Fg : IWin32Window {
    [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
    public IntPtr Handle { get; set; }
    public static Fg Current() { return new Fg { Handle = GetForegroundWindow() }; }
}
'@
$owner = [Fg]::Current()`

func browseFile() (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
` + dialogOwner + `
$f = New-Object System.Windows.Forms.OpenFileDialog
$f.Filter = "Documents & images|*.epub;*.mobi;*.azw3;*.fb2;*.pdf;*.txt;*.md;*.html;*.htm;*.rtf;*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp;*.tif;*.tiff|Documents|*.epub;*.mobi;*.azw3;*.fb2;*.pdf;*.txt;*.md;*.html;*.htm;*.rtf|Images|*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp;*.tif;*.tiff|All files|*.*"
$f.Title = "Select input file"
$res = $f.ShowDialog($owner)
if ($res -eq 'OK') { [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($f.FileName)) }`
	out, err := runPowershell(script)
	if err != nil {
		return "", err
	}
	return decodeDialogPath(out)
}

func browseFolder() (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
` + dialogOwner + `
$f = New-Object System.Windows.Forms.FolderBrowserDialog
$f.Description = "Select output folder"
$res = $f.ShowDialog($owner)
if ($res -eq 'OK') { [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($f.SelectedPath)) }`
	out, err := runPowershell(script)
	if err != nil {
		return "", err
	}
	return decodeDialogPath(out)
}

func runPowershell(script string) (string, error) {
	// -EncodedCommand (base64 of UTF-16LE) sidesteps every -Command quoting pitfall
	// for multi-line scripts that embed here-strings, inline C# (Add-Type), and double
	// quotes - which the dialog scripts now do.
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePSCommand(script))
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// encodePSCommand encodes a script the way powershell -EncodedCommand expects:
// base64 of its UTF-16LE bytes.
func encodePSCommand(script string) string {
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, len(u16)*2)
	for i, r := range u16 {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// decodeDialogPath decodes a path emitted by the dialog scripts as base64(UTF-8).
// We cannot just print the raw path: launched detached (no console), Windows
// PowerShell writes stdout in the OEM code page (e.g. cp866) and Go, reading it as
// UTF-8, mangles non-ASCII paths like Cyrillic folder names. Base64 is plain ASCII,
// so it survives any code page - and, unlike forcing [Console]::OutputEncoding, it
// never throws in a console-less process (which silently broke the dialog).
func decodeDialogPath(out string) (string, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return "", nil // dialog cancelled
	}
	b, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ── browser / app window ────────────────────────────────────

func openAppWindow(url string) {
	time.Sleep(200 * time.Millisecond) // let server start

	if runtime.GOOS == "windows" {
		// Try Edge (shipped with Win10/11)
		for _, p := range []string{
			filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
		} {
			if _, err := os.Stat(p); err == nil {
				_ = exec.Command(p, "--app="+url, "--window-size=1160,760").Start()
				return
			}
		}
		// Try Chrome
		for _, p := range []string{
			filepath.Join(os.Getenv("ProgramFiles"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
		} {
			if _, err := os.Stat(p); err == nil {
				_ = exec.Command(p, "--app="+url, "--window-size=1160,760").Start()
				return
			}
		}
		// Fallback: default browser
		_ = exec.Command("cmd", "/c", "start", "", url).Start()
		return
	}

	// Non-Windows fallback
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}

// ── heartbeat auto-shutdown ─────────────────────────────────

func watchHeartbeat(srv *http.Server) {
	lastPing.Store(time.Now().Unix())
	for {
		time.Sleep(5 * time.Second)
		// A running conversion counts as alive: keep the timer fresh so we neither
		// abort it nor shut down the instant a long one finishes (the browser needs
		// a moment to resume pinging).
		if activeRuns.Load() > 0 {
			lastPing.Store(time.Now().Unix())
			continue
		}
		if time.Now().Unix()-lastPing.Load() > 15 {
			_ = srv.Shutdown(context.Background())
			os.Exit(0)
		}
	}
}
