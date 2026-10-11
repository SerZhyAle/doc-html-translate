// probes.go defines the probe shapes of strategic section 6 and the platform-neutral citizen
// probes. Platform probes live in probes_windows.go; probes_other.go stubs them so the tree
// compiles everywhere a vet can reach, with every stub refusing to measure.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// latencyProbe measures one thing repeatedly for a duration and hands every observation to the
// sink. A probe that cannot run reports why through Available, and the runner turns that into
// COULD NOT VERIFY - never into silence and never into a pass.
type latencyProbe struct {
	name string
	// budgetNames are the budget ids this probe's numbers feed (for the jitter gate).
	budgetNames []string
	available   func() (bool, string)
	run         func(ctx context.Context, d time.Duration, sink func(ms float64)) error
}

// probeRegistry returns the probes a control run exercises, platform-dependent parts included.
// Order is display order; every probe runs its own duration.
func probeRegistry() []latencyProbe {
	return append([]latencyProbe{
		{
			name:        "P6.fsync-writer",
			budgetNames: []string{"H10"},
			available:   func() (bool, string) { return true, "" },
			run:         runFsyncProbe,
		},
		{
			name:        "P6.http-fetch",
			budgetNames: []string{"H10"},
			available:   func() (bool, string) { return true, "" },
			run:         runHTTPFetchProbe,
		},
	}, platformProbes()...)
}

// probeByName finds one registry probe by name (scenes look their citizen probes up by name so
// the platform-neutral code never names a platform function).
func probeByName(name string) (latencyProbe, bool) {
	for _, p := range probeRegistry() {
		if p.name == name {
			return p, true
		}
	}
	return latencyProbe{}, false
}

// runFsyncProbe is the foreign fsync writer (P6): time a write+sync round to a scratch file.
// It measures the machine's own disk latency, which a scene's citizen comparison subtracts.
func runFsyncProbe(ctx context.Context, d time.Duration, sink func(ms float64)) error {
	f, err := os.CreateTemp("", "auditstand-fsync-*.tmp")
	if err != nil {
		return fmt.Errorf("fsync probe: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		start := time.Now()
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := f.Write(payload); err != nil {
			return fmt.Errorf("fsync probe: %w", err)
		}
		if err := f.Sync(); err != nil {
			return fmt.Errorf("fsync probe: %w", err)
		}
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		{
			// Keep the probe light: it must never be the load the scene measures.
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Millisecond):
			}
		}
	}
}

// runHTTPFetchProbe is the foreign HTTP fetch (P6): time a loopback GET against a server the
// stand owns. It never sends anything to the network - the app's own hosts (update check,
// tessdata) are the app's business, and the probe only needs the local stack's latency.
func runHTTPFetchProbe(ctx context.Context, d time.Duration, sink func(ms float64)) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("http probe: %w", err)
	}
	defer ln.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/probe", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()
	url := "http://" + ln.Addr().String() + "/probe"
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("http probe: %w", err)
		}
		_, err = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// tempResidue walks a directory and returns the files and their total size - the P4 folder
// sampler and the P7 consistency check share it. Directories carry no meaning here; a zero-file
// walk is the "temp residue zero" outcome of H13.
func tempResidue(dir string) (files []string, totalBytes int64, err error) {
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, path)
		totalBytes += info.Size()
		return nil
	})
	return files, totalBytes, err
}

// --- shared probe types (platform-neutral; implementations in probes_windows.go) ---

// procSample is one P4 observation of a live process.
type procSample struct {
	PID            int     `json:"pid"`
	Handles        int64   `json:"handles"`
	GDI            int64   `json:"gdi"`
	USER           int64   `json:"user"`
	Threads        int64   `json:"threads"`
	PrivateBytes   float64 `json:"private_bytes"`
	WorkingSetByte float64 `json:"working_set_bytes"`
}

// focusEvent is one foreground-window transition the focus observer saw.
type focusEvent struct {
	At     time.Time `json:"at"`
	Window uintptr   `json:"window"`
	Title  string    `json:"title"`
}

// auditWindowMarker is the title substring the app's visible window carries (the browser window
// the GUI host opens is titled DOC-HTML-UI; the host itself owns no window - strategic
// section 4). Overridable for tests and stand-in windows.
var auditWindowMarker = "DOC-HTML-UI"
