package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// stubBrowser doubles the test binary as an instrumented browser: it serves the debugging
// endpoint over loopback (the marker target among its pages, a CDP websocket beside it) and
// stays alive until killed, so the session's ownership discipline is exercised for real.
func stubBrowser(args []string) int {
	port := 0
	markerURL := ""
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--remote-debugging-port="):
			_, _ = fmt.Sscanf(strings.TrimPrefix(args[i], "--remote-debugging-port="), "%d", &port)
		case strings.HasPrefix(args[i], "--"):
		default:
			markerURL = args[i]
		}
	}
	if port == 0 {
		return 2
	}
	// The merged-launch mode: a browser that exits at once, the way a real launch does when a
	// running instance of the same profile absorbs it.
	if os.Getenv("AUDITSTAND_STUB_BROWSER_MODE") == "exit" {
		return 0
	}
	appOrigin := "http://127.0.0.1:8123/"
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Browser":"stub/1"}`)
	})
	mux.HandleFunc("/json/list", func(w http.ResponseWriter, r *http.Request) {
		wsHost := fmt.Sprintf("ws://127.0.0.1:%d", port)
		fmt.Fprintf(w, `[{"id":"stub-marker","type":"page","title":"stub marker","url":%q,"webSocketDebuggerUrl":%q},{"id":"stub-app","type":"page","title":"stub app","url":%q,"webSocketDebuggerUrl":%q}]`,
			markerURL, wsHost+"/ws", appOrigin, wsHost+"/ws")
	})
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			var request struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
			}
			if websocket.JSON.Receive(ws, &request) != nil {
				return
			}
			var result any
			switch request.Method {
			case "Memory.getDOMCounters":
				result = map[string]int{"documents": 1, "nodes": 42, "jsEventListeners": 3}
			case "Runtime.getHeapUsage":
				result = map[string]int{"usedSize": 1024, "totalSize": 2048}
			default:
				result = map[string]any{"result": map[string]any{"value": true}}
			}
			if websocket.JSON.Send(ws, map[string]any{"id": request.ID, "result": result}) != nil {
				return
			}
		}
	}))
	if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), mux); err != nil {
		return 1
	}
	return 0
}

func getServerHost(port int) string { return fmt.Sprintf("http://127.0.0.1:%d", port) }

// TestBrowserPrelaunchMarkerClose pins step 00.08 against the stub browser: the launch opens
// the debugging endpoint, the marker target is found by its URL, the DOM counters flow
// through the websocket, and closing the session kills the instance and the port.
func TestBrowserPrelaunchMarkerClose(t *testing.T) {
	if testing.Short() {
		t.Skip("the stub browser is a real child process")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUDITSTAND_STUB_BROWSER", "1")
	marker, err := startMarkerServer()
	if err != nil {
		t.Fatal(err)
	}
	defer marker.close()
	if !strings.Contains(marker.title, markerTitlePrefix) {
		t.Fatalf("marker title %q lost its prefix", marker.title)
	}
	port, err := freeLoopbackPort()
	if err != nil {
		t.Fatal(err)
	}
	session, err := prelaunchBrowser(exe, port, marker.url, marker.title)
	if err != nil {
		t.Fatalf("prelaunch against the stub browser failed: %v", err)
	}
	if !session.owned {
		t.Fatal("a fresh stub launch must be owned by the session")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	target, err := session.markerTarget(ctx)
	if err != nil {
		t.Fatalf("marker target not found: %v", err)
	}
	if target.ID != "stub-marker" {
		t.Fatalf("wrong marker target: %+v", target)
	}
	app, err := session.appTarget(ctx, "http://127.0.0.1:8123/")
	if err != nil {
		t.Fatalf("app target not found: %v", err)
	}
	if app.ID != "stub-app" {
		t.Fatalf("wrong app target: %+v", app)
	}
	probe, err := connectBrowserProbe(ctx, getServerHost(port), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	fields, err := probe.sampleDOM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fields["dom_nodes"] != 42 {
		t.Fatalf("wrong counters: %v", fields)
	}
	_ = probe.conn.Close()
	if err := session.close(); err != nil {
		t.Fatalf("close must leave the port shut: %v", err)
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		t.Fatal("the debugging port is still open after the session closed")
	}
}

// TestBrowserPrelaunchMergedLaunch pins the honest fallback: a browser that never opens the
// endpoint reports errBrowserPortUnavailable, and the session it returns refuses to kill
// anything it did not start.
func TestBrowserPrelaunchMergedLaunch(t *testing.T) {
	if testing.Short() {
		t.Skip("the stub browser is a real child process")
	}
	oldTimeout := browserReadyTimeout
	browserReadyTimeout = 500 * time.Millisecond
	defer func() { browserReadyTimeout = oldTimeout }()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// A stub that exits at once never opens the endpoint: the merged-launch path.
	t.Setenv("AUDITSTAND_STUB_BROWSER", "1")
	t.Setenv("AUDITSTAND_STUB_BROWSER_MODE", "exit")
	port, err := freeLoopbackPort()
	if err != nil {
		t.Fatal(err)
	}
	// Occupy nothing: the port stays closed, so readiness never succeeds.
	session, err := prelaunchBrowser(exe, port, "http://127.0.0.1:1/", "merged")
	if err == nil {
		_ = session.close()
		t.Fatal("a launch whose endpoint never opened must report the fallback sentinel")
	}
	if err != errBrowserPortUnavailable {
		t.Fatalf("wrong error: %v", err)
	}
	if session == nil || session.owned {
		t.Fatal("an unopened endpoint must leave the session unowned")
	}
	if killErr := session.close(); killErr != nil {
		t.Fatalf("an unowned session must close without killing anything: %v", killErr)
	}
}

// TestBrowserEndpointRefusesNonLoopback keeps the discovery loopback-only.
func TestBrowserEndpointRefusesNonLoopback(t *testing.T) {
	if err := browserEndpointReady("http://example.com/json/version"); err == nil {
		t.Fatal("a non-loopback endpoint was accepted")
	}
}

// TestGridMaxDiff pins the capture-diff comparator: incomparable grids must read as changed,
// a small cursor blink must read as unchanged and a real repaint as changed.
func TestGridMaxDiff(t *testing.T) {
	a := make([]byte, 64*48)
	if got := gridMaxDiff(a, a); got != 0 {
		t.Fatalf("identical grids differ by %d", got)
	}
	if got := gridMaxDiff(a, nil); got < captureDiffSensitivity {
		t.Fatal("incomparable grids must never read as unchanged")
	}
	b := make([]byte, 64*48)
	copy(b, a)
	b[100] = byte(captureDiffSensitivity - 1)
	if got := gridMaxDiff(a, b); got >= captureDiffSensitivity {
		t.Fatalf("a sub-threshold change read as a reaction (%d)", got)
	}
	b[100] = byte(captureDiffSensitivity + 8)
	if got := gridMaxDiff(a, b); got < captureDiffSensitivity {
		t.Fatal("a real repaint did not reach the sensitivity threshold")
	}
}
