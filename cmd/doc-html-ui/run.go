package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// procTree is a started converter and everything it spawned.
type procTree interface {
	Kill() error // stop the whole tree now
	Release()    // the run ended on its own: let go without killing what it left behind
}

// resolveCLI is findCLI, indirected so a test can run a fake converter.
var resolveCLI = findCLI

// pipeGrace bounds how long the relay waits for output after the converter exited. A
// grandchild that inherited the pipe could otherwise hold the run open indefinitely.
const pipeGrace = 3 * time.Second

// maxLogLine caps one relayed line. A longer one is passed on in pieces rather than held,
// so a runaway line costs bounded memory and never stalls the child.
const maxLogLine = 8 << 20

// ── one run per output location ─────────────────────────────

type activeRun struct {
	cancel    context.CancelFunc
	cancelled bool
}

var runs = struct {
	sync.Mutex
	m map[string]*activeRun
}{m: map[string]*activeRun{}}

var errRunActive = errors.New("a conversion into this output folder is already running")

// runKey names the output location a request writes to. Two requests that would write the
// same folder share a key, whatever spelling of the path they used.
func runKey(input, output string) string {
	key := input
	if t, err := resolveOutputDir(input, output); err == nil {
		key = t.Dir
	} else if abs, err := filepath.Abs(input); err == nil {
		key = abs
	}
	key = filepath.Clean(key)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}

func claimRun(key string, cancel context.CancelFunc) (*activeRun, error) {
	runs.Lock()
	defer runs.Unlock()
	if _, busy := runs.m[key]; busy {
		return nil, errRunActive
	}
	r := &activeRun{cancel: cancel}
	runs.m[key] = r
	return r, nil
}

func releaseRun(key string) {
	runs.Lock()
	delete(runs.m, key)
	runs.Unlock()
}

// cancelRun stops the run writing to key, if there is one.
func cancelRun(key string) bool {
	runs.Lock()
	defer runs.Unlock()
	r, ok := runs.m[key]
	if ok {
		r.cancelled = true
		r.cancel()
	}
	return ok
}

// cancelAllRuns stops every run; the GUI is going away.
func cancelAllRuns() {
	runs.Lock()
	defer runs.Unlock()
	for _, r := range runs.m {
		r.cancelled = true
		r.cancel()
	}
}

func wasCancelled(key string) bool {
	runs.Lock()
	defer runs.Unlock()
	r, ok := runs.m[key]
	return ok && r.cancelled
}

// handleCancel stops the conversion writing to the request's output location.
//
//	POST {"input":"..","output":".."} → {"ok":true,"cancelled":bool}
func handleCancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "cancelled": cancelRun(runKey(req.Input, req.Output))})
}

// ── the run itself ──────────────────────────────────────────

func handleRun(w http.ResponseWriter, r *http.Request) {
	activeRuns.Add(1)
	defer activeRuns.Add(-1)

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

	// The child lives no longer than the request: a page that went away, a Cancel press and
	// the GUI shutting down all end here.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	key := runKey(req.Input, req.Output)
	if _, err := claimRun(key, cancel); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	defer releaseRun(key)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	args := assembleArgs(req)
	bin := resolveCLI()
	fmt.Fprintf(w, "> %s\n\n", formatCommandLine(bin, args))
	flusher.Flush()

	err := runConverter(ctx, bin, args, w, flusher)
	switch {
	case wasCancelled(key):
		fmt.Fprintf(w, "\nCancelled.\n")
	case err != nil:
		fmt.Fprintf(w, "\nExit: %v\n", err)
	default:
		fmt.Fprintf(w, "\nDone.\n")
		if outputDir, err := previousResult(req.Input, req.Output); err == nil {
			fp := fingerprintFor(req)
			_ = updateParamsHistory(func(m map[string]string) { m[outputDir] = fp })
		}
	}
	flusher.Flush()
}

// runConverter starts the converter, relays its output to w until it exits, and stops its
// whole process tree if ctx ends first.
func runConverter(ctx context.Context, bin string, args []string, w io.Writer, flusher http.Flusher) error {
	outR, outW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		_ = outR.Close()
		_ = outW.Close()
		return fmt.Errorf("stderr pipe: %w", err)
	}
	defer outR.Close()
	defer errR.Close()

	cmd := exec.Command(bin, args...)
	cmd.Stdout = outW
	cmd.Stderr = errW
	prepareTree(cmd)
	startErr := cmd.Start()
	// The child holds its own copies of the write ends; ours must go, or the readers never
	// see EOF.
	_ = outW.Close()
	_ = errW.Close()
	if startErr != nil {
		return fmt.Errorf("start: %w", startErr)
	}

	tree, err := attachTree(cmd)
	if err != nil {
		// Without the tree handle Cancel could not stop the child, so do not run it at all.
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("process control: %w", err)
	}
	// The watcher alone decides whether the tree is killed. An exit that has already
	// happened wins over a late cancel, so a finished run never takes down the browser it
	// just opened.
	exited := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			select {
			case <-exited:
			default:
				_ = tree.Kill()
			}
		case <-exited:
		}
	}()

	lines := make(chan string, 256)
	stop := make(chan struct{}) // closed when the relay gives up on output still pending
	var readers sync.WaitGroup
	readers.Add(2)
	go pumpLines(outR, "", lines, stop, &readers)
	go pumpLines(errR, "[err] ", lines, stop, &readers)
	readersDone := make(chan struct{})
	go func() { readers.Wait(); close(lines); close(readersDone) }()

	waitErr := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		close(exited)
		select {
		case <-readersDone:
		case <-time.After(pipeGrace):
			// Something the converter left behind still holds the pipes. The relay stops
			// here; the pumps keep draining in the background so that process never blocks.
			// Closing the read ends is not enough on its own: on Windows it does not
			// interrupt a read already waiting on an anonymous pipe.
			close(stop)
		}
		waitErr <- err
	}()

	relayLines(lines, stop, w, flusher)
	err = <-waitErr
	<-watched
	tree.Release()
	return err
}

// relayLines is the single writer of the response: both pumps feed it, so their lines
// never interleave mid-line and the ResponseWriter is never touched concurrently. Once a
// write fails (the page went away) it keeps draining, so the pumps and the child are never
// blocked on a reader that is gone.
func relayLines(lines <-chan string, stop <-chan struct{}, w io.Writer, flusher http.Flusher) {
	writable := true
	for {
		var s string
		select {
		case l, ok := <-lines:
			if !ok {
				return
			}
			s = l
		case <-stop:
			return
		}
		if !writable {
			continue
		}
		if _, err := io.WriteString(w, s); err != nil {
			writable = false
			continue
		}
		if len(lines) == 0 {
			flusher.Flush()
		}
	}
}

// pumpLines turns one child stream into whole prefixed lines. It has no line-length limit
// that could stop it: an oversized line is forwarded in pieces. After a read error, or once
// the relay has stopped listening, the stream is still drained so the writer never blocks
// on a full pipe.
func pumpLines(rd io.Reader, prefix string, out chan<- string, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	br := bufio.NewReaderSize(rd, 64<<10)
	var buf []byte
	emit := func() bool {
		line := strings.TrimRight(string(buf), "\r\n")
		buf = buf[:0]
		select {
		case out <- prefix + line + "\n":
			return true
		case <-stop:
			return false
		}
	}
	for {
		chunk, err := br.ReadSlice('\n')
		buf = append(buf, chunk...)
		if errors.Is(err, bufio.ErrBufferFull) {
			if len(buf) >= maxLogLine && !emit() {
				_, _ = io.Copy(io.Discard, br)
				return
			}
			continue
		}
		if len(buf) > 0 && !emit() {
			_, _ = io.Copy(io.Discard, br)
			return
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				_, _ = io.Copy(io.Discard, rd)
			}
			return
		}
	}
}
