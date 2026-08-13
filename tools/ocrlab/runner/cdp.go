package runner

// The DevTools transport the runner drives the browser with.
//
// Phase 04 drove the browser through its one-shot command line - `--dump-dom` for the geometry,
// `--screenshot` for the pixels - on the grounds that it needed no dependency and no open port.
// That premise expired: measured on 2026-08-12 with Chrome 151.0.7922.76 and the matching Edge,
// under `--headless=new`, `--headless=old` and plain `--headless`, `--dump-dom` writes zero bytes
// to stdout and `--screenshot` writes no file. Both switches simply do nothing now, silently, so
// the failure looked like an empty page rather than a missing feature.
//
// So the runner speaks the DevTools protocol instead - the same protocol, and for the same reason,
// as the extension edition's scripts/_ocrlab-cdp.mjs. It costs one WebSocket client, which
// golang.org/x/net already provides and this module already depends on, so nothing new enters
// go.mod. It also buys back what the one-shot design cost: one browser for the whole run instead
// of one launch per page state, and a viewport that is emulated rather than relaunched.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"doc-html-translate/tools/ocrlab/evidence"
)

// portTimeout bounds the launch itself. Separate from browserTimeout because a browser that never
// opens its debugging port has failed to start, which is a different fault from a page that never
// finishes, and reporting them as one would send the reader looking in the wrong place.
const portTimeout = 30 * time.Second

// devtools is one live browser session: the process, the connection, and the page target every
// command is addressed to.
type devtools struct {
	cmd     *exec.Cmd
	conn    *websocket.Conn
	session string
	stderr  string // path to the browser's stderr, read only when something has gone wrong

	writeMu sync.Mutex // one writer at a time; the protocol is a single stream

	mu      sync.Mutex
	nextID  int
	pending map[int]chan cdpReply
	dead    error
}

type cdpReply struct {
	result json.RawMessage
	err    error
}

type cdpFrame struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"error"`
}

// start launches the browser and attaches to a fresh page target.
func (b *Browser) start() (*devtools, error) {
	portFile := filepath.Join(b.profileDir, "DevToolsActivePort")
	_ = os.Remove(portFile) // a stale file from a previous run would point at a dead port

	errFile := filepath.Join(b.profileDir, "browser-stderr.log")
	errOut, err := os.Create(errFile)
	if err != nil {
		return nil, err
	}
	defer errOut.Close()

	cmd := exec.Command(b.Path, launchArgs(b.profileDir)...)
	// Handed real files rather than an io.Writer for the reason the one-shot launcher documented:
	// os/exec would otherwise build a pipe whose handle the browser's helper processes inherit, and
	// nothing here would ever see the far end close.
	cmd.Stdout, cmd.Stderr = nil, errOut
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	port, err := waitForPort(portFile)
	if err != nil {
		killTree(cmd)
		return nil, fmt.Errorf("%w%s", err, tail(errFile))
	}
	conn, err := dialBrowser(port)
	if err != nil {
		killTree(cmd)
		return nil, fmt.Errorf("%w%s", err, tail(errFile))
	}

	d := &devtools{cmd: cmd, conn: conn, stderr: errFile, pending: map[int]chan cdpReply{}}
	go d.read()
	if err := d.attach(); err != nil {
		d.close()
		return nil, err
	}
	return d, nil
}

// debuggerOrigin is the Origin this client presents and the only one the browser is told to
// accept.
//
// Both halves are forced. Since Chrome 111 the debugger refuses a WebSocket carrying an Origin
// header unless that origin is allow-listed - its defence against a web page reaching the debugger
// by DNS rebinding - and the refusal arrives as a bare "bad status" from the handshake, which says
// nothing about the cause. x/net/websocket, for its part, refuses to dial without an Origin at
// all. So the header is sent and exactly this one value is permitted; `*` would also work and is
// what most tooling reaches for, but there is no reason to widen a door on a debugging port.
const debuggerOrigin = "http://127.0.0.1"

// launchArgs is the browser's command line. Separate from start so the guard test can assert the
// property that cost an afternoon - the profile path must be absolute and in native separators, or
// headless Edge hangs forever instead of failing.
func launchArgs(profileDir string) []string {
	return []string{
		"--headless=new", "--disable-gpu", "--no-first-run", "--hide-scrollbars",
		"--no-default-browser-check", "--disable-extensions",
		"--disable-features=Translate,MediaRouter",
		"--disable-background-timer-throttling",
		"--user-data-dir=" + profileDir,
		"--remote-debugging-port=0",
		"--remote-allow-origins=" + debuggerOrigin,
		"about:blank",
	}
}

// waitForPort reads the port the browser chose. `--remote-debugging-port=0` means "pick one and
// write it down", which is what keeps concurrent runs and the user's own browser out of each
// other's way.
func waitForPort(portFile string) (string, error) {
	deadline := time.Now().Add(portTimeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(portFile); err == nil {
			if line := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0]); line != "" {
				return line, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("the browser never opened a debugging port within %s", portTimeout)
}

func dialBrowser(port string) (*websocket.Conn, error) {
	resp, err := http.Get("http://127.0.0.1:" + port + "/json/version")
	if err != nil {
		return nil, fmt.Errorf("ask the browser for its debugger URL: %w", err)
	}
	defer resp.Body.Close()
	var v struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("parse /json/version: %w", err)
	}
	if v.WebSocketDebuggerURL == "" {
		return nil, errors.New("the browser reported no debugger URL")
	}
	cfg, err := websocket.NewConfig(v.WebSocketDebuggerURL, debuggerOrigin)
	if err != nil {
		return nil, err
	}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", v.WebSocketDebuggerURL, err)
	}
	// A page's outerHTML is the largest thing that crosses this connection and a scanned page's
	// overlay is not small; the default cap is generous but not obviously so.
	conn.MaxPayloadBytes = 64 << 20
	return conn, nil
}

// attach opens a page target and enables the two domains the runner uses.
func (d *devtools) attach() error {
	raw, err := d.sendTo("", "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		return err
	}
	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(raw, &target); err != nil {
		return err
	}
	raw, err = d.sendTo("", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true})
	if err != nil {
		return err
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(raw, &attached); err != nil {
		return err
	}
	d.session = attached.SessionID
	for _, domain := range []string{"Page.enable", "Runtime.enable"} {
		if _, err := d.send(domain, nil); err != nil {
			return err
		}
	}
	return nil
}

// read delivers replies to whoever is waiting for them. Events carry no id and are dropped: the
// runner polls the page for the one state it cares about rather than subscribing, which keeps the
// waiting rule in one readable place instead of spread across event handlers.
func (d *devtools) read() {
	for {
		var f cdpFrame
		if err := websocket.JSON.Receive(d.conn, &f); err != nil {
			d.fail(err)
			return
		}
		if f.ID == 0 {
			continue
		}
		d.mu.Lock()
		ch := d.pending[f.ID]
		delete(d.pending, f.ID)
		d.mu.Unlock()
		if ch == nil {
			continue
		}
		if f.Error != nil {
			ch <- cdpReply{err: fmt.Errorf("%s %s", f.Error.Message, strings.TrimSpace(string(f.Error.Data)))}
			continue
		}
		ch <- cdpReply{result: f.Result}
	}
}

// fail marks the session dead and releases everyone waiting, so a browser that died mid-run
// reports that rather than every caller timing out one after another.
func (d *devtools) fail(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dead == nil {
		d.dead = fmt.Errorf("the browser connection ended: %w", err)
	}
	for id, ch := range d.pending {
		ch <- cdpReply{err: d.dead}
		delete(d.pending, id)
	}
}

func (d *devtools) send(method string, params map[string]any) (json.RawMessage, error) {
	return d.sendTo(d.session, method, params)
}

// sendTo addresses one command. An empty session is the browser itself, which is where the target
// commands live; everything else goes to the attached page.
func (d *devtools) sendTo(session, method string, params map[string]any) (json.RawMessage, error) {
	d.mu.Lock()
	if d.dead != nil {
		err := d.dead
		d.mu.Unlock()
		return nil, err
	}
	d.nextID++
	id := d.nextID
	ch := make(chan cdpReply, 1)
	d.pending[id] = ch
	d.mu.Unlock()

	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if session != "" {
		msg["sessionId"] = session
	}
	d.writeMu.Lock()
	err := websocket.JSON.Send(d.conn, msg)
	d.writeMu.Unlock()
	if err != nil {
		d.mu.Lock()
		delete(d.pending, id)
		d.mu.Unlock()
		return nil, fmt.Errorf("%s: %w", method, err)
	}

	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("%s: %w", method, r.err)
		}
		return r.result, nil
	case <-time.After(browserTimeout):
		d.mu.Lock()
		delete(d.pending, id)
		d.mu.Unlock()
		return nil, fmt.Errorf("%s: no reply after %s%s", method, browserTimeout, tail(d.stderr))
	}
}

// evaluate runs an expression in the page and returns its value as a string. A non-string value
// comes back empty rather than as a Go rendering of someone else's JSON.
func (d *devtools) evaluate(expr string) (string, error) {
	raw, err := d.send("Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true})
	if err != nil {
		return "", err
	}
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", err
	}
	if r.Exception != nil {
		return "", fmt.Errorf("page threw: %s", r.Exception.Text)
	}
	var s string
	_ = json.Unmarshal(r.Result.Value, &s)
	return s, nil
}

// waitForProbe blocks until the page has published its measurement.
//
// This is the honest replacement for `--virtual-time-budget`, which was never a statement about
// the page being ready - it was a guess at how long everything takes. Waiting for the probe's own
// output waits for exactly the thing the caller is about to read.
func (d *devtools) waitForProbe() error {
	expr := `(function(){var e=document.getElementById(` + strconv.Quote(ProbeElementID) + `);return e?e.textContent:"";})()`
	deadline := time.Now().Add(browserTimeout)
	var last error
	for time.Now().Before(deadline) {
		got, err := d.evaluate(expr)
		if err == nil && got != "" {
			return nil
		}
		last = err
		time.Sleep(100 * time.Millisecond)
	}
	if last != nil {
		return fmt.Errorf("the page produced no %s element within %s (last: %v)", ProbeElementID, browserTimeout, last)
	}
	return fmt.Errorf("the page produced no %s element within %s - the probe did not run", ProbeElementID, browserTimeout)
}

// capture returns the page as PNG bytes, beyond the viewport as well, so a tall page is not
// silently cropped to the window before the crop that is meant to happen.
func (d *devtools) capture() ([]byte, error) {
	raw, err := d.send("Page.captureScreenshot", map[string]any{"format": "png", "captureBeyondViewport": true})
	if err != nil {
		return nil, err
	}
	var r struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(r.Data)
}

func (d *devtools) close() {
	if d.conn != nil {
		_ = d.conn.Close()
	}
	killTree(d.cmd)
}

// tail returns the end of the browser's stderr, prefixed for an error message. Empty when there is
// nothing to say, so a message never ends in a dangling colon.
func tail(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	const keep = 600
	if len(data) > keep {
		data = data[len(data)-keep:]
	}
	return "\nthe browser said: " + strings.TrimSpace(string(data))
}

// applyViewport emulates one pinned geometry. Emulation rather than a relaunch: the point of the
// three viewports is the same page measured at each, and a relaunch would also throw away the
// warmed browser between them.
func (d *devtools) applyViewport(v evidence.Viewport) error {
	_, err := d.send("Emulation.setDeviceMetricsOverride", map[string]any{
		"width": v.Width, "height": v.Height, "deviceScaleFactor": v.DeviceScaleFactor, "mobile": false,
	})
	return err
}

// openProbed navigates to one page state and waits for its measurement.
//
// The detour through about:blank is not decoration: every state this runner asks for is selected
// by the URL fragment, and navigating from one fragment to another in the same document does not
// reload - the probe would never re-run and the caller would read the previous state's numbers.
func (b *Browser) openProbed(pagePath, fragment string, v evidence.Viewport) (*devtools, error) {
	d, err := b.session()
	if err != nil {
		return nil, err
	}
	if err := d.applyViewport(v); err != nil {
		return nil, err
	}
	url, err := fileURL(pagePath)
	if err != nil {
		return nil, err
	}
	if _, err := d.send("Page.navigate", map[string]any{"url": "about:blank"}); err != nil {
		return nil, err
	}
	if _, err := d.send("Page.navigate", map[string]any{"url": url + fragment}); err != nil {
		return nil, err
	}
	if err := d.waitForProbe(); err != nil {
		return nil, err
	}
	return d, nil
}
