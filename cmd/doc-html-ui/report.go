package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/internal/report"
)

// handleReport packs the recent run logs, an environment summary and the last saved settings
// into one archive and answers where it landed. Nothing is sent: the archive is a file on the
// user's disk, and the page hands it to the user's own mail program from there.
//
//	POST → {"ok":true,"path":"..","dropped":N,"bytes":N} | {"ok":false,"error":".."}
func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// A settings blob that cannot be read costs the report one section, not the report.
	settings, _ := os.ReadFile(settingsPath())

	path, dropped, err := report.Build(report.BuildOptions{
		AppVersion:   Version,
		Packaged:     isPackaged(),
		SettingsJSON: settings,
		At:           time.Now(),
	})
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	var size int64
	if info, statErr := os.Stat(path); statErr == nil {
		size = info.Size()
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"path":    path,
		"dropped": dropped,
		"bytes":   size,
	})
}

// revealInExplorer selects a file in a new Explorer window. Indirected so the containment
// guard can be tested without a test run opening file-manager windows.
var revealInExplorer = func(path string) error {
	cmd := exec.Command("explorer.exe", "/select,"+path)
	// Explorer reports a non-zero exit code even when it did open the window, so the only
	// honest signal available is whether the process started at all.
	return cmd.Start()
}

// insideReportDir reports whether path is a file the app itself wrote under report.Dir().
// The page sends back a path the server handed it, but a handler that opens or reveals an
// arbitrary path on request is a file-manager for the whole disk, so the claim is re-checked
// here rather than trusted.
func insideReportDir(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	root, err := filepath.Abs(report.Dir())
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// pathRequest decodes the shared {"path":".."} body and applies the containment check,
// answering the caller itself when either fails.
func pathRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return "", false
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return "", false
	}
	if !insideReportDir(req.Path) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "outside the report folder"})
		return "", false
	}
	return req.Path, true
}

// handleReportReveal opens the archive's folder with the file selected, so attaching it is
// one drag. The app never attaches it - see the ticket's ADR-3.
func handleReportReveal(w http.ResponseWriter, r *http.Request) {
	path, ok := pathRequest(w, r)
	if !ok {
		return
	}
	if err := revealInExplorer(path); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// handleReportOpen opens the archive itself, so the user can see what they are about to send
// before they send it.
func handleReportOpen(w http.ResponseWriter, r *http.Request) {
	path, ok := pathRequest(w, r)
	if !ok {
		return
	}
	if err := openTarget(path); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// handleLogsClear empties the run-log store for a user who would rather not keep a history.
func handleLogsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := report.ClearLogs(); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
