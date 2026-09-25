package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"

	"doc-html-translate/internal/browser"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/report"
	"doc-html-translate/internal/syslocale"
	"doc-html-translate/internal/translator"
	"doc-html-translate/internal/windowsreg"
)

//go:embed ui.html
var uiHTML string

//go:embed i18n.js
var uiI18nJS string

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

	srv := &http.Server{Handler: newAPIGuard(newMux(), addr, uiToken)}

	go watchHeartbeat(srv)
	go openAppWindow("http://" + addr)

	_ = srv.Serve(ln)
}

// uiToken is this launch's API secret. It is baked into the served page and nowhere else.
var uiToken = newToken()

// newMux wires every route to its handler and pins each one to the method (and, for a
// JSON body, the content type) it is meant for. See guard.go for why.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", getOnly(handleUI))
	mux.HandleFunc("/i18n.js", getOnly(handleI18nJS))
	mux.HandleFunc("/favicon.ico", getOnly(handleFavicon))
	mux.HandleFunc("/api/version", getOnly(handleVersion))
	mux.HandleFunc("/api/initial", getOnly(handleInitial))
	mux.HandleFunc("/api/env", getOnly(handleEnv))
	mux.HandleFunc("/api/assoc-status", getOnly(handleAssocStatus))
	mux.HandleFunc("/api/ocr-langs", getOnly(handleOCRLangs))
	mux.HandleFunc("/api/alive", getOnly(handleAlive))
	mux.HandleFunc("/api/ping", postAction(handlePing))
	mux.HandleFunc("/api/browse-file", postAction(handleBrowseFile))
	mux.HandleFunc("/api/browse-folder", postAction(handleBrowseFolder))
	mux.HandleFunc("/api/drop", postAction(handleDrop))
	mux.HandleFunc("/api/register", postAction(handleRegister))
	mux.HandleFunc("/api/unregister", postAction(handleUnregister))
	mux.HandleFunc("/api/open-default-apps", postAction(handleOpenDefaultApps))
	mux.HandleFunc("/api/report", postAction(handleReport))
	mux.HandleFunc("/api/logs-clear", postAction(handleLogsClear))
	mux.HandleFunc("/api/settings", getOrPostJSON(handleSettings))
	mux.HandleFunc("/api/google-key", getOrPostJSON(handleGoogleKey))
	mux.HandleFunc("/api/preview", jsonPost(handlePreview))
	mux.HandleFunc("/api/output-status", jsonPost(handleOutputStatus))
	mux.HandleFunc("/api/open-output", jsonPost(handleOpenOutput))
	mux.HandleFunc("/api/delete-output", jsonPost(handleDeleteOutput))
	mux.HandleFunc("/api/run", jsonPost(handleRun))
	mux.HandleFunc("/api/cancel", jsonPost(handleCancel))
	mux.HandleFunc("/api/answer", jsonPost(handleAnswer))
	mux.HandleFunc("/api/shell-entries", jsonPost(handleShellEntries))
	mux.HandleFunc("/api/ocr-download", jsonPost(handleOCRDownload))
	mux.HandleFunc("/api/report-reveal", jsonPost(handleReportReveal))
	mux.HandleFunc("/api/report-open", jsonPost(handleReportOpen))
	return mux
}

// The GUI writes nothing to the registry on its own. The right-click "Convert to HTML" entry and
// the "Open with" advertisement are added only when the user says yes - the installer's
// "openwith" task, the first-run question in the window, or the toggle under "Windows
// integration" - so a user who unticked the installer task is not overridden by the next launch
// (APP-BEHAVIOUR rules 4 and 11).

// ── HTTP handlers ───────────────────────────────────────────

// handleUI serves the page with this launch's token in it. The page must not be cached:
// a copy from an earlier launch would carry a dead token.
func handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	_, _ = io.WriteString(w, strings.Replace(uiHTML, tokenPlaceholder, uiToken, 1))
}

// handleI18nJS serves the GUI dictionary. It is a separate file rather than an inline block
// because thirteen languages of ~87 keys each would otherwise bury the markup in ui.html.
func handleI18nJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = io.WriteString(w, uiI18nJS)
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

// busy marks a request that the watchdog must outlive, such as an open dialog.
func busy() func() {
	activeRuns.Add(1)
	return func() { activeRuns.Add(-1) }
}

func handleBrowseFile(w http.ResponseWriter, r *http.Request) {
	defer busy()()
	path, err := browseFile(r.Context(), dialogTitle(r, "Select input file"))
	if err != nil {
		logFailure("browse file", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": path})
}

func handleBrowseFolder(w http.ResponseWriter, r *http.Request) {
	defer busy()()
	path, err := browseFolder(r.Context(), dialogTitle(r, "Select output folder"))
	if err != nil {
		logFailure("browse folder", err)
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
	if name == "" || name == "." || name == ".." || name == string(os.PathSeparator) {
		http.Error(w, "missing or invalid filename", http.StatusBadRequest)
		return
	}

	dest, err := saveDropped(http.MaxBytesReader(w, r.Body, maxDropBytes), name)
	if err != nil {
		logFailure("drop", err)
		http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": dest})
}

// saveDropped stores an upload as <drop dir>/<content hash>/<name>. Keying the folder by
// content means two different files both called "download.pdf" no longer overwrite each
// other (and the second one no longer reopens the first one's conversion), while
// re-dropping the same file lands on the same path and reuses its result. The bytes go
// to a temporary file first, so a failed upload never replaces a complete copy.
func saveDropped(body io.Reader, name string) (string, error) {
	root := droppedFilesDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(root, "upload-*.part")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), body); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	dir := filepath.Join(root, hex.EncodeToString(h.Sum(nil))[:16])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, name)
	if fi, err := os.Stat(dest); err == nil && fi.Mode().IsRegular() {
		// Same content already dropped under this name: keep it, and its output.
		return dest, nil
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return "", err
	}
	return dest, nil
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
//	GET  → the saved JSON (or {} if none yet). An unreadable file is set aside, and the
//	       X-Settings-Corrupt header says where, so the page can tell the user.
//	POST → save the request body (must be valid JSON, capped at 64 KiB)
func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		data, corruptAt := readSettings()
		if corruptAt != "" {
			w.Header().Set("X-Settings-Corrupt", corruptAt)
		}
		if data == nil {
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
		if err := writeSettings(data); err != nil {
			logFailure("save settings", err)
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
			logFailure("save google key", err)
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
	if err := writeFileAtomic(path, []byte(key), 0o600); err != nil {
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
	UILang         string `json:"uiLang"`
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

// resolveOutputDir mirrors internal/pipeline's choice of output directory so the GUI can
// look at (and clear) a previous result without shelling out to the CLI. It shares the
// resolver itself, not a copy of its rules: the directory for "book.pdf" may be
// "book (pdf)" when "book" belongs to another document, and the GUI must agree.
func resolveOutputDir(input, folder string) (outputpath.Target, error) {
	abs, err := filepath.Abs(input)
	if err != nil {
		return outputpath.Target{}, err
	}
	return outputpath.Resolve(abs, folder)
}

// previousResult returns the output directory of an earlier conversion of input, if one
// exists that the converter owns. A folder that merely contains an index.html - a saved
// website, a user's own folder - is never reported, opened or deleted as a result.
func previousResult(input, folder string) (string, error) {
	target, err := resolveOutputDir(input, folder)
	if err != nil {
		return "", err
	}
	if !target.State.Ours() {
		return "", errNoPreviousResult
	}
	if _, err := os.Stat(filepath.Join(target.Dir, "index.html")); err != nil {
		return "", errNoPreviousResult
	}
	return target.Dir, nil
}

var errNoPreviousResult = errors.New("no previous result found")

// handleOutputStatus reports whether a previous conversion result exists for the
// request's input/output, and whether it was built with different options than the
// request currently describes - so the GUI can offer to delete and rebuild instead of
// silently reopening a stale result. The answer comes from the completion record the CLI
// writes into the output itself, so the GUI and a run from the command line agree on it.
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
	outputDir, err := previousResult(req.Input, req.Output)
	if err != nil {
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp["exists"] = true
	resp["outputDir"] = outputDir
	resp["paramsChanged"] = optionsChanged(req, outputDir)
	_ = json.NewEncoder(w).Encode(resp)
}

// optionsChanged reports whether the finished output in outputDir was built with other
// result-affecting options than req asks for. The options are derived by parsing the very
// command line the run would execute, so CLI defaults apply exactly as they will in the run.
// An unfinished output is not "changed": the CLI rebuilds it on its own, without asking.
func optionsChanged(req runRequest, outputDir string) bool {
	cfg, err := config.ParseArgs(assembleArgs(req))
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(cfg.InputFile)
	if err != nil {
		return false
	}
	reason, _ := outputpath.CheckReuse(outputDir, abs, outputpath.OptionsFor(cfg))
	return reason == outputpath.ReuseOptionsChanged
}

// openTarget hands a path to the OS. Indirected so the guard below can be tested without
// a test run spawning a browser or a file manager.
var openTarget = browser.Open

// handleOpenOutput opens a previous conversion result: the page itself, or the folder
// holding it when the caller asks for that. A dropped file is copied into the app's own
// data folder and the result lands beside it, so without this the reader is left with a
// result they cannot find again once the browser tab is gone. Like handleDeleteOutput it
// only ever acts on a result the converter owns (previousResult), so a stray
// output-folder value can't steer it into opening something unrelated.
func handleOpenOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Input  string `json:"input"`
		Output string `json:"output"`
		Folder bool   `json:"folder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	outputDir, err := previousResult(req.Input, req.Output)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	target := filepath.Join(outputDir, "index.html")
	if req.Folder {
		target = outputDir
	}
	if err := openTarget(target); err != nil {
		logFailure("open result", err)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// handleDeleteOutput removes a previous conversion result so the next run starts
// clean. It only ever deletes a directory carrying the converter's ownership marker (or a
// recognised pre-marker output) for this very input - a folder that merely holds an
// index.html, a saved website or the user's own, is refused - and never one a
// conversion is still writing.
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
	outputDir, err := previousResult(req.Input, req.Output)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if outputpath.Locked(outputDir) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "cause": "locked", "error": outputpath.ErrLocked.Error()})
		return
	}
	if err := os.RemoveAll(outputDir); err != nil {
		logFailure("delete result", err)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// handleRegister sets this app as the default Windows handler for the supported
// document types (the opt-in association, off by default). It shells out to the bundled
// CLI's -register flow so the file associations point at doc-html-translate.exe (the
// headless converter) rather than at the GUI - matching the unpackaged double-click
// behavior. Under an MSIX/Store install this is a no-op (HKCU is virtualized); the GUI
// hides the association toggle there.
func handleRegister(w http.ResponseWriter, r *http.Request) {
	defer busy()()
	ctx, cancel := context.WithTimeout(r.Context(), registerTimeout)
	defer cancel()
	bin := findCLI()
	cmd := exec.CommandContext(ctx, bin, "-register")
	// The CLI's -register flow prints a splash and waits on Scanln; feed a newline
	// so it returns immediately instead of blocking this request.
	cmd.Stdin = strings.NewReader("\n")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logFailure("register", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out))))
	}
	// The child's exit code says only that something was written. Whether Windows now uses
	// it is read back from the registry, the same way the status endpoint does.
	resp := assocStatus()
	resp["ok"] = err == nil
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		resp["error"] = msg
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleShellEntries adds or removes the right-click "Convert to HTML" entry and the "Open
// with" advertisement - the first-run question's yes, and the toggle under "Windows
// integration". Both point at the CLI, so the right-click entry converts and opens the
// result, like a double-click on an associated file. Hidden under MSIX, where the package
// manifest declares the associations and HKCU is virtualized.
//
//	POST {"on":bool} → assocStatus() + {"ok":bool}
func handleShellEntries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		On bool `json:"on"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var err error
	switch {
	case !req.On:
		_, err = windowsreg.RemoveShellEntries()
	case !cliAvailable():
		err = errors.New("the converter is not next to the app or on PATH")
	default:
		cli := findCLI()
		_, openWithErr := windowsreg.RegisterOpenWithFor(cli)
		_, menuErr := windowsreg.RegisterContextMenuFor(cli)
		err = errors.Join(openWithErr, menuErr)
	}
	logFailure("shell entries", err)
	resp := assocStatus()
	resp["ok"] = err == nil
	_ = json.NewEncoder(w).Encode(resp)
}

// registerTimeout bounds the -register child. Registration is a handful of registry writes;
// a child still running after this is stuck, not slow.
const registerTimeout = 2 * time.Minute

// handleUnregister releases the default-handler association (the "off" side of the
// association toggle), leaving the non-destructive "Convert to HTML" right-click verb and
// "Open with" entry in place. Deletion only clears HKCU values that point at our ProgID,
// so it needs no CLI round-trip and does not depend on which exe wrote them.
func handleUnregister(w http.ResponseWriter, _ *http.Request) {
	_, err := windowsreg.Unregister()
	logFailure("unregister", err)
	resp := assocStatus()
	resp["ok"] = err == nil
	if err != nil {
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleOpenDefaultApps opens Settings > Default apps, the one place the user's own choice
// can be changed. It runs only on a click: popping Settings unasked would be a surprise.
func handleOpenDefaultApps(w http.ResponseWriter, _ *http.Request) {
	err := windowsreg.OpenDefaultAppsSettings()
	logFailure("open default apps", err)
	resp := map[string]any{"ok": err == nil}
	if err != nil {
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleAssocStatus reports which handler Windows actually uses for the supported types, so
// the GUI can reflect the association toggle's state, tell "registered, not default" apart
// from "default", and decide whether to show the one-time first-run opt-in prompt.
func handleAssocStatus(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(assocStatus())
}

// assocStatus is the association state the GUI renders:
//
//	default  → Windows opens every supported type with the app
//	defaults → the types Windows opens with the app
//	state   → default, blocked, unknown, partial or none (windowsreg.Status.Summary)
//	blocked → registered, but the user's own choice in Windows wins
//	unknown → the user's choice could not be read
//	other   → types that open with something else
//	shell   → the right-click "Convert to HTML" entry is registered
func assocStatus() map[string]any {
	st := windowsreg.HandlerStatus()
	return map[string]any{
		"shell":    windowsreg.HasShellEntries(),
		"default":  st.IsDefault(),
		"state":    st.Summary(),
		"defaults": nonNil(st.Default),
		"blocked":  nonNil(st.Blocked),
		"unknown":  nonNil(st.Unknown),
		"other":    nonNil(st.Other),
	}
}

// nonNil keeps an empty list a JSON [] rather than null, so the page can join it unguarded.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// handleEnv reports environment facts the GUI adapts to on load:
//
//	packaged → running from inside an MSIX/Store package, where -register is a no-op
//	           (file associations come from the package manifest instead).
//	cli      → the bundled converter exe is resolvable next to the app or on PATH;
//	           without it, Convert cannot run.
//	lang     → the Windows UI language, one of the thirteen the app ships; the GUI opens
//	           in it by default, unless the user has picked a language before.
//	font     → the UI font that language's script needs (Nirmala UI, Microsoft YaHei UI),
//	           empty when the default stack covers it.
//	logs     → how many run logs are held and how many bytes they take, for the About section.
//	author   → the address a report is mailed to. It is served rather than written into the
//	           page a second time, so the product has one place that defines it.
func handleEnv(w http.ResponseWriter, _ *http.Request) {
	lang := i18n.Resolve("", "", syslocale.Lang())
	logCount, logBytes := logStoreSize()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"packaged": isPackaged(),
		"cli":      cliAvailable(),
		"lang":     lang,
		"font":     i18n.FontFamily(lang),
		"logs":     map[string]any{"count": logCount, "bytes": logBytes},
		"author":   authorEmail,
	})
}

// authorEmail is where a report goes. Same address as the page's feedback link.
const authorEmail = "sza@ukr.net"

// logStoreSize measures the run-log store. A store that is not there yet is empty, not an
// error - the About section says "0 logs" and the next conversion creates it.
func logStoreSize() (count int, bytes int64) {
	entries, err := os.ReadDir(report.LogsDir())
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		count++
		bytes += info.Size()
	}
	return count, bytes
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
		ISO       string `json:"iso"` // ISO-639-1 code of -src that selects this data, "" if none
		Installed bool   `json:"installed"`
	}
	out := make([]row, 0, len(ocr.Available))
	for _, l := range ocr.Available {
		out = append(out, row{Code: l.Code, Name: l.Name, ISO: ocr.ISOFor(l.Code), Installed: installed[l.Code]})
	}
	_ = json.NewEncoder(w).Encode(out)
}

// handleOCRDownload downloads a single OCR language pack on request from the GUI. The answer is
// a stream of JSON lines: {"done":N,"total":N} as the pack arrives, then one final
// {"ok":bool,...}. Closing the request cancels the download, which leaves nothing behind
// (APP-BEHAVIOUR rule 3). A failure carries a cause code the page words itself; the raw error
// goes to the GUI log, and "error" keeps the localized text for callers that want it.
func handleOCRDownload(w http.ResponseWriter, r *http.Request) {
	defer busy()()
	var req struct {
		Lang   string `json:"lang"`
		UILang string `json:"uiLang"` // the page's language, for the error text
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	enc := json.NewEncoder(w)
	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	err := ocr.CheckLang(req.Lang)
	if err == nil {
		var last time.Time
		err = ocr.DownloadContext(r.Context(), req.Lang, func(done, total int64) {
			// A line per network read would be thousands; a few a second is what a bar needs.
			if done < total && time.Since(last) < 150*time.Millisecond {
				return
			}
			last = time.Now()
			_ = enc.Encode(map[string]int64{"done": done, "total": total})
			flush()
		})
	}
	resp := map[string]any{"ok": err == nil}
	if err != nil {
		cause := ocrFailureCause(err)
		// A refused code and a cancel are answers, not failures; only a real failure is logged.
		if cause == "network" || cause == "mismatch" {
			logFailure("ocr download "+req.Lang, err)
		}
		resp["cause"] = cause
		resp["error"] = ocr.ErrorText(err, req.UILang)
	}
	_ = enc.Encode(resp)
	flush()
}

// ocrFailureCause names why a download failed, for the page to word: the user cancelled, the
// pack did not match its pinned digest, the code is not in the catalogue, or the transfer or the
// disk failed.
func ocrFailureCause(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, ocr.ErrPackMismatch):
		return "mismatch"
	case errors.Is(err, ocr.ErrUnknownLang):
		return "lang"
	}
	return "network"
}

// ── args assembly ───────────────────────────────────────────

// assembleArgs turns the form into the converter's command line. Every field is checked
// before it is forwarded: the CLI rejects a malformed value outright, so one stray character
// in a box the user is not even using used to break every run. An empty or malformed field
// means the GUI's documented default, and a field that belongs to one engine is sent only
// when that engine is selected.
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
		if m := strings.TrimSpace(req.OllamaModel); m != "" && m != "gemma3:12b" && !strings.HasPrefix(m, "-") {
			a = append(a, "-ollama-model", m)
		}
		if n, ok := intField(req.OllamaParallel, 1); ok && n != 1 {
			a = append(a, "-ollama-parallel", strconv.Itoa(n))
		}
		if n, ok := intField(req.OllamaCtx, 1); ok && n != 8192 {
			a = append(a, "-ollama-ctx", strconv.Itoa(n))
		}
	}
	// The GUI's split default is 0 (off) while the CLI's is 5000, so the value is always
	// sent: leaving it out would silently switch splitting on.
	split, ok := intField(req.SplitSize, 0)
	if !ok {
		split = 0
	}
	a = append(a, "-split", strconv.Itoa(split))
	// toc-depth/max-cost default to 0 (unlimited / no limit) in the CLI, so only
	// forward them when the user picked a non-default value - keeps the command line clean.
	if n, ok := intField(req.TOCDepth, 0); ok && n != 0 {
		a = append(a, "-toc-depth", strconv.Itoa(n))
	}
	if !req.SinglePage {
		a = append(a, "-multipage")
	}
	// Max cost guards the paid engine only.
	if req.Google {
		if f, err := strconv.ParseFloat(strings.TrimSpace(req.MaxCost), 64); err == nil && f > 0 && !math.IsInf(f, 0) {
			a = append(a, "-max-cost", strconv.FormatFloat(f, 'f', -1, 64))
		}
	}
	if isLangCode(req.SrcLang) {
		a = append(a, "-src", req.SrcLang)
	}
	if isLangCode(req.DstLang) {
		a = append(a, "-dst", req.DstLang)
	}
	if req.OCR {
		a = append(a, "-ocr")
		if ocrLangPattern.MatchString(req.OCRLang) {
			a = append(a, "-ocr-lang", req.OCRLang)
		}
	}
	if req.Force {
		a = append(a, "-force")
	}
	if req.Verbose {
		a = append(a, "-v")
	}
	// The window's own language selector is the control here, so the converted page's
	// navigation speaks the language the user just read the GUI in. Empty means "whatever
	// the OS says", which is the CLI's own default; a code the CLI does not know is dropped
	// rather than failing the run.
	if req.UILang != "" && slices.Contains(i18n.Codes, req.UILang) {
		a = append(a, "-ui-lang", req.UILang)
	}
	if req.Output != "" {
		a = append(a, "-folder", trimTrailingSeparators(req.Output))
	}
	if req.Input != "" {
		// "--" ends flag parsing, so an input named "-force.epub" is converted, not obeyed.
		a = append(a, "--", trimTrailingSeparators(req.Input))
	}
	return a
}

// intField parses a numeric form field. ok is false for an empty, non-integer or
// below-minimum value, which the caller treats as "use the default".
func intField(s string, minimum int) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < minimum {
		return 0, false
	}
	return n, true
}

// isLangCode accepts the shape of a translation language code ("en", "zh-CN", "auto").
func isLangCode(s string) bool { return langCodePattern.MatchString(s) }

var (
	langCodePattern = regexp.MustCompile(`^[A-Za-z]{2,8}(-[A-Za-z0-9]{1,8})?$`)
	// Tesseract data names: "eng", "chi_sim", "eng+rus".
	ocrLangPattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}(\+[A-Za-z0-9_]{1,32}){0,7}$`)
)

// trimTrailingSeparators drops a trailing "\" or "/" from a folder path (keeping a
// root such as `C:\`). The path means the same without it, and inside quotes a
// trailing backslash escapes the closing quote when Windows PowerShell hands the
// argument to a native program, so the copied command would not run as shown.
func trimTrailingSeparators(p string) string {
	t := strings.TrimRight(p, `\/`)
	if t == "" || (len(t) == 2 && t[1] == ':') {
		return p
	}
	return t
}

// formatCommandLine joins a binary and its args into a single command line that pastes
// into PowerShell and runs as shown. It starts with the call operator "& ", which
// PowerShell needs before a quoted program path and accepts before a bare one. This is
// also what the run log's header shows, so the live preview matches the command that
// actually executes.
func formatCommandLine(bin string, args []string) string {
	parts := make([]string, 0, len(args)+2)
	parts = append(parts, "&", quoteArg(bin))
	for _, a := range args {
		parts = append(parts, quoteArg(a))
	}
	return strings.Join(parts, " ")
}

// quoteArg renders one argument for PowerShell. A token made only of characters that
// PowerShell reads literally stays bare; anything else goes in single quotes, where
// nothing (no $, `, &, %, ;) is expanded. Inside them a quote is escaped by doubling,
// and PowerShell also treats the typographic quotes as single quotes. "--" is quoted
// because some PowerShell versions swallow a bare "--" instead of passing it on.
func quoteArg(s string) string {
	if s != "" && s != "--" && !strings.ContainsFunc(s, needsPSQuote) {
		return s
	}
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		if isPSSingleQuote(r) {
			b.WriteRune(r)
		}
		b.WriteRune(r)
	}
	b.WriteByte('\'')
	return b.String()
}

func needsPSQuote(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	return !strings.ContainsRune(`-_.:\/=`, r)
}

func isPSSingleQuote(r rune) bool {
	return r == '\'' || r == '\u2018' || r == '\u2019' || r == '\u201A' || r == '\u201B'
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

// dialogTitle is the caption the page asked for, in the window's language, or fallback. It is
// capped, and it reaches the script only as base64 (psString), so no caption can be code.
func dialogTitle(r *http.Request, fallback string) string {
	t := strings.TrimSpace(r.URL.Query().Get("title"))
	if t == "" {
		return fallback
	}
	if rs := []rune(t); len(rs) > 120 {
		t = string(rs[:120])
	}
	return t
}

// psString renders s as a PowerShell expression that evaluates to s. The text travels as base64
// of its UTF-8 bytes, so neither a quote nor a $ in it can end the string or run anything.
func psString(s string) string {
	return "[System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('" + base64.StdEncoding.EncodeToString([]byte(s)) + "'))"
}

func browseFile(ctx context.Context, title string) (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
` + dialogOwner + `
$f = New-Object System.Windows.Forms.OpenFileDialog
$f.Filter = "Documents, images & comics|*.epub;*.mobi;*.azw3;*.fb2;*.pdf;*.txt;*.md;*.html;*.htm;*.rtf;*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp;*.tif;*.tiff;*.cbz;*.cbr;*.cb7;*.cbt|Documents|*.epub;*.mobi;*.azw3;*.fb2;*.pdf;*.txt;*.md;*.html;*.htm;*.rtf|Images|*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp;*.tif;*.tiff|Comics|*.cbz;*.cbr;*.cb7;*.cbt|All files|*.*"
$f.Title = ` + psString(title) + `
$res = $f.ShowDialog($owner)
if ($res -eq 'OK') { [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($f.FileName)) }`
	out, err := runPowershell(ctx, script)
	if err != nil {
		return "", err
	}
	return decodeDialogPath(out)
}

func browseFolder(ctx context.Context, title string) (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
` + dialogOwner + `
$f = New-Object System.Windows.Forms.FolderBrowserDialog
$f.Description = ` + psString(title) + `
$res = $f.ShowDialog($owner)
if ($res -eq 'OK') { [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($f.SelectedPath)) }`
	out, err := runPowershell(ctx, script)
	if err != nil {
		return "", err
	}
	return decodeDialogPath(out)
}

// runPowershell runs a dialog script for as long as the page that asked for it is there:
// the dialog waits on the user, so it has no timeout of its own, but a page that went away
// takes its dialog with it.
func runPowershell(ctx context.Context, script string) (string, error) {
	// -EncodedCommand (base64 of UTF-16LE) sidesteps every -Command quoting pitfall
	// for multi-line scripts that embed here-strings, inline C# (Add-Type), and double
	// quotes - which the dialog scripts now do.
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePSCommand(script))
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
				_ = startDetached(exec.Command(p, "--app="+url, "--window-size=1160,760"))
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
				_ = startDetached(exec.Command(p, "--app="+url, "--window-size=1160,760"))
				return
			}
		}
		// Fallback: default browser, through the shell API rather than cmd.exe.
		_ = openTarget(url)
		return
	}

	// Non-Windows fallback
	switch runtime.GOOS {
	case "darwin":
		_ = startDetached(exec.Command("open", url))
	default:
		_ = startDetached(exec.Command("xdg-open", url))
	}
}

// startDetached starts a program the GUI never waits for and releases its process
// handle at once, so a long-lived GUI does not collect one per window it opened.
func startDetached(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
