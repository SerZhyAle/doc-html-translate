package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"doc-html-translate/internal/commands"
)

var uiTestSeq atomic.Int64

func newTestPipeMutex() (string, string) {
	n := uiTestSeq.Add(1)
	pid := os.Getpid()
	return fmt.Sprintf(`\\.\pipe\DocHtmlTranslate_UITest_%d_%d`, pid, n),
		fmt.Sprintf(`Local\DocHtmlTranslate_UITest_%d_%d`, pid, n)
}

func TestInstanceCommandStatusShowQuitOpenAndEvents(t *testing.T) {
	m, err := commands.Load()
	if err != nil {
		t.Fatal(err)
	}
	pipeName, mutexName := newTestPipeMutex()
	lock, acquired, err := commands.AcquireInstanceLock(mutexName)
	if err != nil || !acquired {
		t.Fatalf("AcquireInstanceLock: acquired=%v err=%v", acquired, err)
	}
	defer lock.Release()

	var winOpen atomic.Bool
	var runsActive atomic.Int64
	var cancelled atomic.Int64
	var openedWindow atomic.Int64
	var raisedWindow atomic.Int64
	var quitTriggered atomic.Bool

	st := &instanceState{
		version:     "26.1116.0001",
		subs:        map[chan openEventMsg]struct{}{},
		acks:        map[string]chan int{},
		windowOpen:  winOpen.Load,
		activeRuns:  func() int { return int(runsActive.Load()) },
		cancelRuns:  func() int { n := runsActive.Load(); cancelled.Store(n); runsActive.Store(0); return int(n) },
		isClosing:   func() bool { return false },
		setClosing:  func() {},
		openWindow:  func() { openedWindow.Add(1) },
		raiseWindow: func() (bool, bool) { raisedWindow.Add(1); return true, true },
		onQuit:      func() { quitTriggered.Store(true) },
	}

	ps, err := commands.StartPipeServer(pipeName, func(line []byte) ([]byte, func()) {
		return commands.HandlePipeRequest(context.Background(), m, st, "en", line)
	}, "en")
	if err != nil {
		t.Fatalf("StartPipeServer: %v", err)
	}
	defer ps.Close()

	var outBuf, errBuf bytes.Buffer
	env := commands.Env{
		Version:   "26.1116.0001",
		Open:      func() (io.Writer, io.Writer) { return &outBuf, &errBuf },
		Lang:      "en",
		PipeName:  pipeName,
		MutexName: mutexName,
		Launcher:  &commands.LauncherHooks{PipeName: pipeName, MutexName: mutexName},
	}

	// 1. status while no page is open -> windowOpen=false, acceptsRestrictedCommands=false.
	outBuf.Reset()
	errBuf.Reset()
	if handled, code := commands.Front([]string{"status", "--json"}, env); !handled || code != commands.ExitDone {
		t.Fatalf("status: handled=%v code=%d err=%q", handled, code, errBuf.String())
	}
	var statusResp commands.Response
	if err := json.Unmarshal(bytes.TrimSpace(outBuf.Bytes()), &statusResp); err != nil {
		t.Fatalf("invalid status json: %v", err)
	}
	var sd commands.StatusData
	if err := json.Unmarshal(statusResp.Data, &sd); err != nil || sd.WindowOpen || sd.AcceptsRestrictedCommands || sd.PID != os.Getpid() {
		t.Fatalf("unexpected status data: %+v (err=%v)", sd, err)
	}

	// 2. show while no page is open -> reopened=true, openWindow called once.
	outBuf.Reset()
	if handled, code := commands.Front([]string{"show", "--json"}, env); !handled || code != commands.ExitDone {
		t.Fatalf("show cold: handled=%v code=%d", handled, code)
	}
	var showResp commands.Response
	_ = json.Unmarshal(bytes.TrimSpace(outBuf.Bytes()), &showResp)
	var sh1 commands.ShowData
	_ = json.Unmarshal(showResp.Data, &sh1)
	if !sh1.Reopened || openedWindow.Load() != 1 {
		t.Fatalf("show without page: %+v, opened=%d", sh1, openedWindow.Load())
	}

	// 3. show while page is open -> reopened=false, raiseWindow called.
	winOpen.Store(true)
	outBuf.Reset()
	if handled, code := commands.Front([]string{"show", "--json"}, env); !handled || code != commands.ExitDone {
		t.Fatalf("show warm: handled=%v code=%d", handled, code)
	}
	_ = json.Unmarshal(bytes.TrimSpace(outBuf.Bytes()), &showResp)
	var sh2 commands.ShowData
	_ = json.Unmarshal(showResp.Data, &sh2)
	if sh2.Reopened || !sh2.Focused || raisedWindow.Load() != 1 {
		t.Fatalf("show with page: %+v, raised=%d", sh2, raisedWindow.Load())
	}

	// 4. open before /api/events connects -> populates /api/initial with both "file" and "files".
	dir := t.TempDir()
	f1 := filepath.Join(dir, "one.epub")
	f2 := filepath.Join(dir, "two.pdf")
	f3 := filepath.Join(dir, "three.txt")
	for _, p := range []string{f1, f2, f3} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	outBuf.Reset()
	if handled, code := commands.Front([]string{"open", "--in", f1, "--in", f2, "--json"}, env); !handled || code != commands.ExitDone {
		t.Fatalf("open initial: handled=%v code=%d err=%q", handled, code, errBuf.String())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/initial", getOnly(st.handleInitial))
	mux.HandleFunc("/api/events", getOnly(st.handleEvents))
	mux.HandleFunc("/api/events-ack", jsonPost(st.handleEventsAck))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	respInit, err := http.Get(srv.URL + "/api/initial")
	if err != nil {
		t.Fatal(err)
	}
	var initPayload struct {
		File  string   `json:"file"`
		Files []string `json:"files"`
	}
	_ = json.NewDecoder(respInit.Body).Decode(&initPayload)
	_ = respInit.Body.Close()
	if initPayload.File != f1 || len(initPayload.Files) != 2 || initPayload.Files[1] != f2 {
		t.Fatalf("/api/initial = %+v", initPayload)
	}

	// 5. Connect /api/events and verify open pushes f3 to the live stream and receives ack.
	evCtx, evCancel := context.WithCancel(context.Background())
	defer evCancel()
	evReq, _ := http.NewRequestWithContext(evCtx, http.MethodGet, srv.URL+"/api/events", nil)
	evResp, err := http.DefaultClient.Do(evReq)
	if err != nil {
		t.Fatal(err)
	}
	defer evResp.Body.Close()

	evCh := make(chan openEventMsg, 4)
	go func() {
		sc := bufio.NewScanner(evResp.Body)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var msg openEventMsg
			if json.Unmarshal([]byte(line), &msg) == nil && msg.Type == "open" {
				ackBody, _ := json.Marshal(map[string]any{"id": msg.ID, "queued": len(msg.Paths)})
				_, _ = http.Post(srv.URL+"/api/events-ack", "application/json", bytes.NewReader(ackBody))
				evCh <- msg
			}
		}
	}()

	// Wait briefly until subscriber is registered.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		st.mu.Lock()
		nSubs := len(st.subs)
		st.mu.Unlock()
		if nSubs > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	outBuf.Reset()
	if handled, code := commands.Front([]string{"open", "--in", f3, "--json"}, env); !handled || code != commands.ExitDone {
		t.Fatalf("open live: handled=%v code=%d err=%q", handled, code, errBuf.String())
	}
	select {
	case got := <-evCh:
		if len(got.Paths) != 1 || got.Paths[0] != f3 {
			t.Fatalf("event paths = %v, want [%s]", got.Paths, f3)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for /api/events open message")
	}

	// 6. quit busy (exit 11) vs quit --cancel-runs (exit 0).
	runsActive.Store(1)
	outBuf.Reset()
	if handled, code := commands.Front([]string{"quit"}, env); !handled || code != commands.ExitNotFound {
		t.Fatalf("quit busy: handled=%v code=%d", handled, code)
	}
	outBuf.Reset()
	if handled, code := commands.Front([]string{"quit", "--cancel-runs", "--json"}, env); !handled || code != commands.ExitDone {
		t.Fatalf("quit cancel-runs: handled=%v code=%d", handled, code)
	}
	if !quitTriggered.Load() || cancelled.Load() != 1 {
		t.Fatalf("quitTriggered=%v cancelled=%d", quitTriggered.Load(), cancelled.Load())
	}
}

func TestInstanceUIHTMLHasEventsHandlerWithoutNewControls(t *testing.T) {
	for _, needle := range []string{"/api/events", "/api/events-ack", "receiveOpenPaths", "d.files"} {
		if !strings.Contains(uiHTML, needle) {
			t.Errorf("ui.html missing %q", needle)
		}
	}
}
