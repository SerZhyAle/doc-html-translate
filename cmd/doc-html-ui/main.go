package main

import (
	"bufio"
	"context"
	_ "embed"
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

	"doc-html-translate/internal/translator"
)

//go:embed ui.html
var uiHTML string

// Version is set at build time via -ldflags.
var Version = "dev"

const cliName = "doc-html-translate.exe"

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
	mux.HandleFunc("/api/version", handleVersion)
	mux.HandleFunc("/api/initial", handleInitial)
	mux.HandleFunc("/api/ping", handlePing)
	mux.HandleFunc("/api/browse-file", handleBrowseFile)
	mux.HandleFunc("/api/browse-folder", handleBrowseFolder)
	mux.HandleFunc("/api/google-key", handleGoogleKey)
	mux.HandleFunc("/api/run", handleRun)
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/env", handleEnv)

	srv := &http.Server{Handler: mux}

	go watchHeartbeat(srv)
	go openAppWindow("http://" + addr)

	_ = srv.Serve(ln)
}

// ── HTTP handlers ───────────────────────────────────────────

var lastPing atomic.Int64

func handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, uiHTML)
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
	MaxCost        string `json:"maxCost"`
	SrcLang        string `json:"srcLang"`
	DstLang        string `json:"dstLang"`
	Force          bool   `json:"force"`
	Verbose        bool   `json:"verbose"`
}

func handleRun(w http.ResponseWriter, r *http.Request) {
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

	fmt.Fprintf(w, "> %s %s\n\n", bin, strings.Join(args, " "))
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
	}
	flusher.Flush()
}

// handleRegister sets this app as the default Windows handler for the supported
// document types. It shells out to the bundled CLI's -register flow so the file
// associations point at doc-html-translate.exe (the headless converter) rather than
// at the GUI - matching the unpackaged double-click behavior. Under an MSIX/Store
// install this is a no-op (HKCU is virtualized); the GUI hides the button there.
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

// handleEnv reports environment facts the GUI adapts to on load:
//
//	packaged → running from inside an MSIX/Store package, where -register is a no-op
//	           (file associations come from the package manifest instead).
//	cli      → the bundled converter exe is resolvable next to the app or on PATH;
//	           without it, Convert cannot run.
func handleEnv(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"packaged": isPackaged(),
		"cli":      cliAvailable(),
	})
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
	if req.MaxCost != "" && req.MaxCost != "0" {
		a = append(a, "-max-cost", req.MaxCost)
	}
	a = append(a, "-src", req.SrcLang, "-dst", req.DstLang)
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

func browseFile() (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
$f = New-Object System.Windows.Forms.OpenFileDialog
$f.Filter = "Documents|*.epub;*.mobi;*.azw3;*.fb2;*.pdf;*.txt;*.md;*.html;*.htm;*.rtf|All files|*.*"
$f.Title = "Select input file"
if ($f.ShowDialog() -eq 'OK') { $f.FileName }`
	return runPowershell(script)
}

func browseFolder() (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
$f = New-Object System.Windows.Forms.FolderBrowserDialog
$f.Description = "Select output folder"
if ($f.ShowDialog() -eq 'OK') { $f.SelectedPath }`
	return runPowershell(script)
}

func runPowershell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
				_ = exec.Command(p, "--app="+url, "--window-size=780,700").Start()
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
				_ = exec.Command(p, "--app="+url, "--window-size=780,700").Start()
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
		if time.Now().Unix()-lastPing.Load() > 15 {
			_ = srv.Shutdown(context.Background())
			os.Exit(0)
		}
	}
}
