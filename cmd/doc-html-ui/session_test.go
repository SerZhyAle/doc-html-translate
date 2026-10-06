package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ticket 63 - the GUI remembers its session context. The recent list, the window geometry
// and the log/keyboard work all ride surfaces that already exist (the settings blob, the
// relayed run stream); what can be tested in Go is the recent-status endpoint and the
// anchors the page's behaviour is pinned to.

// recentStatuses calls the endpoint and returns its answer as path -> exists.
func recentStatuses(t *testing.T, paths []string) map[string]bool {
	t.Helper()
	b, err := json.Marshal(map[string]any{"paths": paths})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/recent-status", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleRecentStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Statuses []struct {
			Path   string `json:"path"`
			Exists bool   `json:"exists"`
		} `json:"statuses"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
	}
	out := map[string]bool{}
	for _, s := range resp.Statuses {
		out[s.Path] = s.Exists
	}
	return out
}

// The recent list must show an entry that is gone as unavailable, so the answer has to
// distinguish a real file from a stale path - without saying anything else about it.
func TestRecentStatusAnswersExistence(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "book.epub")
	if err := os.WriteFile(live, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	gone := filepath.Join(dir, "deleted.epub")

	got := recentStatuses(t, []string{live, gone})
	if len(got) != 2 {
		t.Fatalf("got %d statuses, want 2", len(got))
	}
	if !got[live] {
		t.Errorf("existing file reported missing: %v", got)
	}
	if got[gone] {
		t.Errorf("missing file reported present: %v", got)
	}
}

// A directory is not a document, whatever os.Stat says about it: an entry that answers
// "exists" must point at a file the converter could actually open.
func TestRecentStatusRefusesDirectories(t *testing.T) {
	dir := t.TempDir()
	if got := recentStatuses(t, []string{dir}); got[dir] {
		t.Fatalf("a directory passed as an existing document: %v", got)
	}
}

// The cap bounds the stat loop; the page asks for far fewer.
func TestRecentStatusCapsThePathList(t *testing.T) {
	paths := make([]string, maxRecentStatusPaths+10)
	for i := range paths {
		paths[i] = filepath.Join(t.TempDir(), "x.epub")
	}
	if got := recentStatuses(t, paths); len(got) != maxRecentStatusPaths {
		t.Fatalf("got %d statuses, want the cap %d", len(got), maxRecentStatusPaths)
	}
}

// The recent list is unreachable if its section, its controls or its plumbing are missing.
func TestUIRecentSectionAnchors(t *testing.T) {
	for _, snippet := range []string{
		`id="recentSection"`, `id="recentList"`, `id="btnRecentClear"`,
		"function rememberRecent(", "function renderRecent(", "function scheduleRecentStatus(",
		"rememberRecent(paths);",                // every pick lands in the list
		"setInputPath(d.file); rememberRecent(", // the association-opened file too
		"recent:         recentDocs,",           // rides the settings blob
		"recentDocs = s.recent.filter(",         // and is restored from it
		"api('/api/recent-status'",              // existence comes from the server
	} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q - the recent list cannot work", snippet)
		}
	}
	// A gone entry is visibly unavailable and cannot be chosen, so a stale path is never
	// converted silently.
	for _, snippet := range []string{"is-gone", "name.disabled = gone;", "recentUnavailable"} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q - a stale recent entry could be converted", snippet)
		}
	}
}

// The window geometry: the page owns restore, clamp and tracking, and the geometry rides
// the settings blob.
func TestUIWindowGeometryRestores(t *testing.T) {
	for _, snippet := range []string{
		"function restoreWindow()", "function clampWindowToScreen()",
		"function trackWindowGeometry()", "function saneGeometry(",
		"restoreWindow(); // the saved size and position, clamped to the screen it opens on",
		"win:            winGeom,", "if (s.win) { const g = saneGeometry(s.win); if (g) winGeom = g; }",
		// A minimized window parks at -32000 on Windows; that is not a geometry to remember.
		"x < -30000 || y < -30000",
	} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q - the window would not reopen where it was", snippet)
		}
	}
}

// The log tools and the failure-line colouring.
func TestUILogToolsAndFailureClasses(t *testing.T) {
	for _, snippet := range []string{
		`id="btnLogCopy"`, `id="btnLogJump"`,
		"btnLogCopy').addEventListener", "btnLogJump').addEventListener",
		".log .log-err", ".log .log-cmd",
		`if (/^\[err\]/.test(text)) cls = 'log-err';`,
		`else if (/^> /.test(text)) cls = 'log-cmd';`,
	} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q - the log console did not gain its tools", snippet)
		}
	}
}

// Keyboard handling has one owner: exactly one document-level keydown listener, driven by
// the SHORTCUTS table, silent while a modal dialog is up (ticket 63, with ticket 57's audit).
// Escape is claimed - default prevented - only while a run can be cancelled (INPUT-PARITY):
// the `when` check sits before preventDefault, so an idle Escape keeps its browser default.
func TestUIShortcutsHaveOneOwner(t *testing.T) {
	if n := strings.Count(uiHTML, "document.addEventListener('keydown'"); n != 1 {
		t.Fatalf("ui.html has %d document keydown listeners, want exactly 1", n)
	}
	for _, snippet := range []string{
		"const SHORTCUTS = [",
		"{key: 'Enter', ctrl: true, fire: () => runPressed()}",
		"{key: 'Escape', when: () => !!(runAbort || queueBusy), fire: () => cancelPressed()}",
		"el('dropZone').focus()",
		"if (dlgOpen || document.querySelector('dialog[open]')) return;",
		"if (s.when && !s.when()) return;\n        ev.preventDefault();",
	} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q - the shortcut table is incomplete", snippet)
		}
	}
}

// ICON-EXTERNAL rule 6: translation pickers preserve each language's endonym across UI switches.
func TestUITranslationLangNamesUseEndonyms(t *testing.T) {
	start := strings.Index(uiI18nJS, "const TRANSLANG_ENDONYMS = {")
	if start < 0 {
		t.Fatal("missing translation endonyms")
	}
	end := strings.Index(uiI18nJS[start:], "\n};")
	if end < 0 {
		t.Fatal("endonym table never closes")
	}
	block := uiI18nJS[start : start+end]
	names := map[string]string{}
	for _, m := range regexp.MustCompile(`([a-z]{2}): "([^"]+)"`).FindAllStringSubmatch(block, -1) {
		names[m[1]] = m[2]
	}
	for _, m := range regexp.MustCompile(`\['([a-z]{2})', '([^']+)'\]`).FindAllStringSubmatch(uiHTML, -1) {
		if got := names[m[1]]; got == "" || got != m[2] {
			t.Errorf("%s: picker %q, dictionary %q", m[1], m[2], got)
		}
	}
	for code, want := range map[string]string{"ru": "Русский", "uk": "Українська", "de": "Deutsch", "ja": "日本語", "ar": "العربية"} {
		if names[code] != want {
			t.Errorf("%s: %q, want endonym %q", code, names[code], want)
		}
	}
	if strings.Contains(uiHTML, "TRANSLANGS[currentLang]") {
		t.Fatal("translation names still depend on interface language")
	}
	for _, snippet := range []string{"const names = TRANSLANG_ENDONYMS;", "fillLangs(el('srcLang'), el('srcLang').value);", "fillLangs(el('dstLang'), el('dstLang').value);"} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("missing %q", snippet)
		}
	}
}
