package main

import (
	"context"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// The server has no window of its own; it exits when its page is gone. It used to decide
// that from a 3-second ping with a 15-second grace, but Chrome throttles the timers of a
// hidden page to about one wake-up a minute, so a GUI left minimized shut itself down. The
// page now also holds a long-lived request open (/api/alive). Throttling slows timers, not
// open connections, so the stream stays up while the window is minimized and drops the
// moment the page closes or reloads. The ping remains as a second signal, and the grace is
// well above the throttled timer interval, so a reload or a stalled stream is survived.

var lastPing atomic.Int64

// activeRuns counts in-flight work (conversions, dialogs, registration). The watchdog never
// shuts the server down under it.
var activeRuns atomic.Int64

// aliveStreams counts open /api/alive requests.
var aliveStreams atomic.Int64

const (
	aliveGrace     = 90 * time.Second
	aliveKeepalive = 20 * time.Second
	watchdogTick   = 5 * time.Second
)

func handlePing(w http.ResponseWriter, _ *http.Request) {
	lastPing.Store(time.Now().Unix())
	w.WriteHeader(http.StatusOK)
}

// handleAlive holds the request open for as long as the page lives. The periodic newline
// makes a dead peer show up as a failed write instead of a connection that looks open.
func handleAlive(w http.ResponseWriter, r *http.Request) {
	aliveStreams.Add(1)
	defer func() {
		aliveStreams.Add(-1)
		lastPing.Store(time.Now().Unix())
	}()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	flusher, _ := w.(http.Flusher)
	tick := time.NewTicker(aliveKeepalive)
	defer tick.Stop()
	for {
		if _, err := w.Write([]byte("\n")); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
	}
}

// shouldExit reports whether the page is gone: no stream, no work, no ping for the grace.
func shouldExit(now time.Time) bool {
	if activeRuns.Load() > 0 || aliveStreams.Load() > 0 {
		lastPing.Store(now.Unix())
		return false
	}
	return now.Unix()-lastPing.Load() > int64(aliveGrace/time.Second)
}

func watchHeartbeat(srv *http.Server) {
	lastPing.Store(time.Now().Unix())
	for {
		time.Sleep(watchdogTick)
		if !shouldExit(time.Now()) {
			continue
		}
		cancelAllRuns()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
		os.Exit(0)
	}
}
