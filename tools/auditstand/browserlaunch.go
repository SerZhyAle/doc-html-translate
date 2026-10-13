// browserlaunch.go is step 00.08 of phase 0: pre-launch the browser the app's own
// openAppWindow will pick, with a debugging port and a stand-owned marker page, so the
// app's shell-open lands in the instrumented instance and P3/P4's DOM half have a target.
// The launch touches the real browser profile - it is the one step of the instrument the
// owner names before it may run. A launch that merges into an already-running instance
// never opens the port: the session reports that honestly and the capture-diff fallback
// carries the timing instead (the DOM-level cells become n/a with that reason).

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// markerTitlePrefix makes the marker tab's window findable by title without embedding
// anything private; the suffix is per-run so a stale tab from an earlier run cannot match.
const markerTitlePrefix = "auditstand marker"

// errBrowserPortUnavailable is returned when the debugging endpoint never answered: the
// browser the app would open was already running and merged the launch (or ignored the
// port flag). The caller records the capture-diff fallback and the DOM-level n/a reason.
var errBrowserPortUnavailable = fmt.Errorf("the instrumented browser's debugging port never opened (a running browser absorbed the launch, or the browser ignores the flag); the capture-diff fallback must carry the timing")

// browserTarget is one page target of the instrumented instance.
type browserTarget struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
	WS    string `json:"webSocketDebuggerUrl"`
}

// browserSession owns the instrumented browser instance for one run.
type browserSession struct {
	exe    string
	port   int
	marker string // marker page URL
	title  string // marker tab title (markerTitlePrefix + per-run suffix)
	proc   *exec.Cmd
	owned  bool // true when this stand started the process that owns the port
}

// appBrowserCandidates mirrors cmd/doc-html-ui's openAppWindow preference order (Edge,
// then Chrome, then the shell default): the instrument must predict where the app's window
// lands, not choose a different browser. Paths only, never a running instance.
func appBrowserCandidates() []string {
	var candidates []string
	if v := os.Getenv("ProgramFiles(x86)"); v != "" {
		candidates = append(candidates, filepath.Join(v, `Microsoft\Edge\Application\msedge.exe`))
	}
	if v := os.Getenv("ProgramFiles"); v != "" {
		candidates = append(candidates, filepath.Join(v, `Microsoft\Edge\Application\msedge.exe`))
	}
	if v := os.Getenv("ProgramFiles"); v != "" {
		candidates = append(candidates, filepath.Join(v, `Google\Chrome\Application\chrome.exe`))
	}
	if v := os.Getenv("ProgramFiles(x86)"); v != "" {
		candidates = append(candidates, filepath.Join(v, `Google\Chrome\Application\chrome.exe`))
	}
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		candidates = append(candidates, filepath.Join(v, `Google\Chrome\Application\chrome.exe`))
	}
	return candidates
}

// pickAppBrowser returns the first browser candidate that exists on this machine.
func pickAppBrowser() (string, error) {
	for _, p := range appBrowserCandidates() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("none of the browsers the app would open (Edge, then Chrome) is installed at the mirrored paths")
}

// freeLoopbackPort reserves an ephemeral loopback port and releases it again, so two runs
// never collide on a fixed port number.
func freeLoopbackPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// browserReadyTimeout bounds how long prelaunchBrowser waits for the debugging endpoint.
// It is a variable so the merged-launch path is testable without twenty real seconds.
var browserReadyTimeout = 20 * time.Second

// prelaunchBrowser starts exe with the debugging port and the marker page. The real browser
// profile is used deliberately (step 00.08); no --user-data-dir is passed, because a
// throwaway profile would change what the audit measures. If the debugging endpoint does
// not answer in time the session is returned with owned=false and the fallback sentinel.
func prelaunchBrowser(exe string, port int, markerURL, markerTitle string) (*browserSession, error) {
	cmd := exec.Command(exe,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-first-run",
		"--no-default-browser-check",
		markerURL,
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s := &browserSession{exe: exe, port: port, marker: markerURL, title: markerTitle, proc: cmd, owned: true}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(browserReadyTimeout)
	for time.Now().Before(deadline) {
		if err := browserEndpointReady(endpoint); err == nil {
			return s, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	// The endpoint never answered: a running instance most likely absorbed the launch
	// (Chromium routes a new launch through an existing process, ignoring the port flag),
	// or the browser does not speak the debugging protocol at all.
	s.owned = false
	return s, errBrowserPortUnavailable
}

// browserEndpointReady checks /json/version on the debugging endpoint (loopback only).
func browserEndpointReady(endpoint string) error {
	if err := loopbackURL(endpoint, "http"); err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(endpoint + "/json/version")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("browser endpoint HTTP %d", resp.StatusCode)
	}
	return nil
}

// browserPageTargets lists the instance's page targets, refusing any discovery answer that
// is not loopback (the same discipline as connectBrowserProbe).
func (s *browserSession) browserPageTargets(ctx context.Context) ([]browserTarget, error) {
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", s.port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/json/list", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("target discovery HTTP %d", resp.StatusCode)
	}
	var targets []browserTarget
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&targets); err != nil {
		return nil, err
	}
	pages := make([]browserTarget, 0, len(targets))
	for _, t := range targets {
		if t.Type != "page" {
			continue
		}
		if t.WS != "" {
			if err := loopbackURL(t.WS, "ws"); err != nil {
				return nil, err
			}
		}
		pages = append(pages, t)
	}
	return pages, nil
}

// markerTarget finds the marker page among the instance's targets. A marker that has
// already been closed by the owner cannot be re-selected: the session reports it.
func (s *browserSession) markerTarget(ctx context.Context) (browserTarget, error) {
	targets, err := s.browserPageTargets(ctx)
	if err != nil {
		return browserTarget{}, err
	}
	for _, t := range targets {
		if t.Title == s.title || t.URL == s.marker {
			return t, nil
		}
	}
	return browserTarget{}, fmt.Errorf("the marker page %q is absent from the instrumented instance", s.title)
}

// appTarget finds the app's window among the page targets: the GUI host serves on loopback,
// so the app page is the target whose URL starts with the host's own origin.
func (s *browserSession) appTarget(ctx context.Context, hostOrigin string) (browserTarget, error) {
	targets, err := s.browserPageTargets(ctx)
	if err != nil {
		return browserTarget{}, err
	}
	for _, t := range targets {
		if strings.HasPrefix(t.URL, hostOrigin) {
			return t, nil
		}
	}
	return browserTarget{}, fmt.Errorf("no instrumented page under %s (the app window did not land in the instrumented instance)", hostOrigin)
}

// close terminates the instance this stand started and verifies the debugging port is gone.
// A session that never owned the port is never killed: the stand does not close a program
// it did not start (the same discipline as the control's app precondition).
func (s *browserSession) close() error {
	if !s.owned || s.proc == nil || s.proc.Process == nil {
		return nil
	}
	_ = s.proc.Process.Kill()
	_, _ = s.proc.Process.Wait()
	return s.waitPortClosed(5 * time.Second)
}

// waitPortClosed dials the debugging port until it refuses connections.
func (s *browserSession) waitPortClosed(timeout time.Duration) error {
	endpoint := fmt.Sprintf("127.0.0.1:%d", s.port)
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", endpoint, 250*time.Millisecond)
		if err != nil {
			return nil
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			return fmt.Errorf("the debugging port %d is still open after closing the instrumented instance", s.port)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// markerServer serves the stand-owned marker page on loopback. The page is inert: a title,
// a heading and one focusable button, so an injected Tab lands somewhere real.
type markerServer struct {
	srv    *http.Server
	url    string
	title  string
	server net.Listener
}

func startMarkerServer() (*markerServer, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	title := fmt.Sprintf("%s %s", markerTitlePrefix, randomMarkerSuffix())
	m := &markerServer{title: title}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>%s</title></head><body><h1>%s</h1><button id=\"audit-marker-target\">marker</button></body></html>", title, title)
	})
	m.srv = &http.Server{Handler: mux}
	m.server = l
	m.url = "http://" + l.Addr().String() + "/"
	go func() { _ = m.srv.Serve(l) }()
	return m, nil
}

func (m *markerServer) close() {
	_ = m.server.Close()
	_ = m.srv.Close()
}

// randomMarkerSuffix derives a per-run marker suffix. Cryptographic randomness is not
// needed here - the suffix only has to be unique among this stand's own runs.
func randomMarkerSuffix() string {
	now := time.Now().UnixNano()
	return fmt.Sprintf("%x-%d", now&0xffff, os.Getpid())
}
