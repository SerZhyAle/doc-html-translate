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
	mux.HandleFunc("/api/run", handleRun)

	srv := &http.Server{Handler: mux}

	go watchHeartbeat(srv)
	go openAppWindow("http://" + addr)

	srv.Serve(ln)
}

// ── HTTP handlers ───────────────────────────────────────────

var lastPing atomic.Int64

func handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, uiHTML)
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"version": Version})
}

func handleInitial(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"file": initialFile})
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
	json.NewEncoder(w).Encode(map[string]string{"path": path})
}

func handleBrowseFolder(w http.ResponseWriter, _ *http.Request) {
	path, err := browseFolder()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"path": path})
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
	if req.SplitSize != "0" && req.SplitSize != "" {
		a = append(a, "-split", req.SplitSize)
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

// ── native file/folder dialogs via PowerShell ───────────────

func browseFile() (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
$f = New-Object System.Windows.Forms.OpenFileDialog
$f.Filter = "Documents|*.epub;*.fb2;*.pdf;*.txt;*.html;*.htm;*.rtf;*.md|All files|*.*"
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
				exec.Command(p, "--app="+url, "--window-size=750,950").Start()
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
				exec.Command(p, "--app="+url, "--window-size=750,950").Start()
				return
			}
		}
		// Fallback: default browser
		exec.Command("cmd", "/c", "start", "", url).Start()
		return
	}

	// Non-Windows fallback
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

// ── heartbeat auto-shutdown ─────────────────────────────────

func watchHeartbeat(srv *http.Server) {
	lastPing.Store(time.Now().Unix())
	for {
		time.Sleep(5 * time.Second)
		if time.Now().Unix()-lastPing.Load() > 15 {
			srv.Shutdown(context.Background())
			os.Exit(0)
		}
	}
}
