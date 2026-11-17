package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"doc-html-translate/internal/commands"
)

type openEventMsg struct {
	Type  string   `json:"type"`
	ID    string   `json:"id,omitempty"`
	Paths []string `json:"paths"`
}

type instanceState struct {
	version     string
	mu          sync.Mutex
	initial     []string
	initialSent bool
	pendingOpen []string
	subs        map[chan openEventMsg]struct{}
	acks        map[string]chan int
	seq         atomic.Int64

	windowOpen  func() bool
	activeRuns  func() int
	cancelRuns  func() int
	isClosing   func() bool
	setClosing  func()
	openWindow  func()
	raiseWindow func() (found, focused bool)
	onQuit      func()
}

func newDefaultInstanceState() *instanceState {
	return &instanceState{
		version:     Version,
		subs:        map[chan openEventMsg]struct{}{},
		acks:        map[string]chan int{},
		windowOpen:  func() bool { return aliveStreams.Load() > 0 },
		activeRuns:  conversionsCount,
		cancelRuns:  cancelAllRuns,
		isClosing:   isClosing,
		setClosing:  setClosing,
		raiseWindow: func() (bool, bool) { return raiseLauncherWindow() },
	}
}

var uiState = newDefaultInstanceState()

func (s *instanceState) setInitialFiles(paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initial = dedupLocalPaths(paths)
	s.initialSent = false
	s.pendingOpen = nil
}

func (s *instanceState) handleInitial(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	files := append([]string(nil), s.initial...)
	s.initialSent = true
	s.mu.Unlock()
	first := ""
	if len(files) > 0 {
		first = files[0]
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"file":  first,
		"files": nonNil(files),
	})
}

func (s *instanceState) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)

	ch := make(chan openEventMsg, 16)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	if len(s.pendingOpen) > 0 {
		ch <- openEventMsg{Type: "open", Paths: append([]string(nil), s.pendingOpen...)}
		s.pendingOpen = nil
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()

	// Write one initial newline so headers flush at once and the client stream reader resolves.
	if _, err := w.Write([]byte("\n")); err != nil {
		return
	}
	if flusher != nil {
		flusher.Flush()
	}

	tick := time.NewTicker(aliveKeepalive)
	defer tick.Stop()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			if err := enc.Encode(msg); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		case <-tick.C:
			if _, err := w.Write([]byte("\n")); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func (s *instanceState) handleEventsAck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Queued int    `json:"queued"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	ch, ok := s.acks[req.ID]
	s.mu.Unlock()
	if ok {
		select {
		case ch <- req.Queued:
		default:
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *instanceState) pushOpenPaths(ctx context.Context, paths []string) (queued int, reopened bool) {
	unique := dedupLocalPaths(paths)
	if len(unique) == 0 {
		return 0, false
	}
	s.mu.Lock()
	if len(s.subs) == 0 {
		var added int
		if !s.initialSent {
			s.initial, added = appendUniquePaths(s.initial, unique)
		} else {
			s.pendingOpen, added = appendUniquePaths(s.pendingOpen, unique)
		}
		needOpen := !s.windowOpen()
		openFn := s.openWindow
		s.mu.Unlock()
		if needOpen && openFn != nil {
			lastPing.Store(time.Now().Unix())
			openFn()
			reopened = true
		}
		return added, reopened
	}

	id := fmt.Sprintf("ev-%d", s.seq.Add(1))
	ack := make(chan int, 1)
	s.acks[id] = ack
	msg := openEventMsg{Type: "open", ID: id, Paths: unique}
	for ch := range s.subs {
		select {
		case ch <- msg:
		default:
		}
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.acks, id)
		s.mu.Unlock()
	}()

	timer := time.NewTimer(400 * time.Millisecond)
	defer timer.Stop()
	select {
	case n := <-ack:
		return n, false
	case <-timer.C:
		return len(unique), false
	case <-ctx.Done():
		return len(unique), false
	}
}

func (s *instanceState) Status(_ context.Context) (commands.StatusData, int, string, string) {
	if s.isClosing() {
		return commands.StatusData{}, commands.ExitNotFound, commands.ReasonBusy, commands.MsgLauncherBusy
	}
	return commands.StatusData{
		Version:                   s.version,
		PID:                       os.Getpid(),
		WindowOpen:                s.windowOpen(),
		ActiveRuns:                s.activeRuns(),
		AcceptsRestrictedCommands: false,
	}, commands.ExitDone, "", commands.MsgStatusReady
}

func (s *instanceState) Show(_ context.Context) (commands.ShowData, int, string, string) {
	if s.isClosing() {
		return commands.ShowData{}, commands.ExitNotFound, commands.ReasonBusy, commands.MsgLauncherBusy
	}
	if s.windowOpen() {
		found, focused := s.raiseWindow()
		if found {
			return commands.ShowData{Focused: focused, Reopened: false}, commands.ExitDone, "", commands.MsgShowDone
		}
		return commands.ShowData{Focused: false, Reopened: false}, commands.ExitDone, "", commands.MsgShowDone
	}
	lastPing.Store(time.Now().Unix())
	if s.openWindow != nil {
		s.openWindow()
	}
	return commands.ShowData{Focused: true, Reopened: true}, commands.ExitDone, "", commands.MsgShowDone
}

func (s *instanceState) Quit(_ context.Context, cancelRuns bool) (commands.QuitData, int, string, string, func()) {
	if s.isClosing() {
		return commands.QuitData{}, commands.ExitNotFound, commands.ReasonBusy, commands.MsgLauncherBusy, nil
	}
	if s.activeRuns() > 0 && !cancelRuns {
		return commands.QuitData{}, commands.ExitNotFound, commands.ReasonBusy, commands.MsgQuitBusy, nil
	}
	s.setClosing()
	cancelled := s.cancelRuns()
	return commands.QuitData{CancelledRuns: cancelled}, commands.ExitDone, "", commands.MsgQuitDone, s.onQuit
}

func (s *instanceState) Open(ctx context.Context, paths []string) (commands.OpenData, int, string, string) {
	if s.isClosing() {
		return commands.OpenData{}, commands.ExitNotFound, commands.ReasonBusy, commands.MsgLauncherBusy
	}
	queued, reopened := s.pushOpenPaths(ctx, paths)
	return commands.OpenData{Queued: queued, Reopened: reopened}, commands.ExitDone, "", commands.MsgOpenDone
}

func dedupLocalPaths(paths []string) []string {
	out, _ := appendUniquePaths(nil, paths)
	return out
}

func appendUniquePaths(existing, incoming []string) ([]string, int) {
	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(incoming))
	keyOf := func(p string) string {
		c := filepath.Clean(p)
		if runtime.GOOS == "windows" {
			return strings.ToLower(c)
		}
		return c
	}
	for _, p := range existing {
		seen[keyOf(p)] = true
		out = append(out, p)
	}
	added := 0
	for _, p := range incoming {
		k := keyOf(p)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, p)
		added++
	}
	return out, added
}
