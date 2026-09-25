// Package procrun is the one way the converter runs an external helper (pdftotext, Tesseract,
// Calibre, 7-Zip, ffmpeg/ImageMagick). Every call gets a deadline, a cancellation path that
// kills the helper's whole process tree, bounded captured output and one error type that names
// the tool. Six call sites with six ad-hoc exec.Command calls is how the converter ended up with
// no time limit anywhere (ticket bugfix-external-process-bounds, ADR-1).
package procrun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"doc-html-translate/internal/i18n"
)

// Output caps. A helper that floods its output must not take the converter's memory with it;
// what is past the cap is read and dropped so the helper is never blocked on a full pipe.
const (
	DefaultMaxStdout = 64 << 20
	DefaultMaxStderr = 64 << 10
)

// waitDelay bounds how long Wait keeps reading pipes after the helper itself has exited or been
// killed. A grandchild that inherited stdout would otherwise keep Wait blocked for as long as it
// lives. A variable so the test of that case does not have to sit out the full delay.
var waitDelay = 5 * time.Second

// Cmd describes one helper invocation.
type Cmd struct {
	// Tool is the name used in messages ("pdftotext", "tesseract").
	Tool string
	Path string
	Args []string
	// Env entries are appended to the converter's own environment.
	Env []string
	// Timeout is the deadline for this call; zero means only ctx bounds it.
	Timeout time.Duration
	// MaxStdout and MaxStderr cap the captured bytes; zero selects the defaults.
	MaxStdout int
	MaxStderr int
}

// Result is what a finished helper produced.
type Result struct {
	Stdout          []byte
	Stderr          []byte
	StdoutTruncated bool
	StderrTruncated bool
}

// Error is returned for every failed call: the helper could not start, exited non-zero, ran out
// of time or was cancelled. Err keeps the underlying cause, so errors.As still finds an
// *os.PathError from a blocked executable or an *exec.ExitError from a failed run.
type Error struct {
	Tool     string
	Timeout  time.Duration
	TimedOut bool
	Canceled bool
	Err      error
	// Stderr is the start of what the helper wrote to stderr, trimmed.
	Stderr string
}

func (e *Error) Error() string {
	var msg string
	switch {
	case e.TimedOut:
		msg = i18n.S("%s did not finish within %s and was stopped (set DOCHT_TOOL_TIMEOUT_SCALE to allow more time)",
			e.Tool, e.Timeout.Round(time.Second))
	case e.Canceled:
		msg = fmt.Sprintf("%s: cancelled", e.Tool)
	default:
		msg = fmt.Sprintf("%s: %v", e.Tool, e.Err)
	}
	if e.Stderr != "" {
		msg += ": " + e.Stderr
	}
	return msg
}

func (e *Error) Unwrap() error { return e.Err }

// ErrTimeout is matched by errors.Is for a call that ran out of time.
var ErrTimeout = errors.New("helper timed out")

// Is lets errors.Is(err, ErrTimeout) work without callers knowing the concrete type.
func (e *Error) Is(target error) bool { return target == ErrTimeout && e.TimedOut }

// Run starts the helper and waits for it. On deadline or cancellation the helper's whole process
// tree is killed; after a normal exit anything the helper left running is killed too, so no
// helper outlives its call.
func Run(ctx context.Context, c Cmd) (Result, error) {
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	stdout := &cappedBuffer{limit: orDefault(c.MaxStdout, DefaultMaxStdout)}
	stderr := &cappedBuffer{limit: orDefault(c.MaxStderr, DefaultMaxStderr)}

	cmd := exec.CommandContext(ctx, c.Path, c.Args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	cmd.WaitDelay = waitDelay
	prepareTree(cmd)

	var mu sync.Mutex
	var tree *processTree
	cmd.Cancel = func() error {
		mu.Lock()
		t := tree
		mu.Unlock()
		return killTree(cmd, t)
	}

	if err := cmd.Start(); err != nil {
		return Result{}, &Error{Tool: c.Tool, Timeout: c.Timeout, Err: err}
	}
	t := attachTree(cmd)
	mu.Lock()
	tree = t
	mu.Unlock()

	waitErr := cmd.Wait()
	releaseTree(t)

	res := Result{
		Stdout:          stdout.buf,
		Stderr:          stderr.buf,
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
	}
	if waitErr == nil {
		return res, nil
	}
	e := &Error{Tool: c.Tool, Timeout: c.Timeout, Err: waitErr, Stderr: strings.TrimSpace(string(stderr.buf))}
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		e.TimedOut = true
	case errors.Is(ctx.Err(), context.Canceled):
		e.Canceled = true
	case errors.Is(waitErr, exec.ErrWaitDelay):
		// The helper itself finished; only a leftover grandchild held the pipes open.
		return res, nil
	}
	return res, e
}

func orDefault(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

// cappedBuffer keeps the first limit bytes written to it and discards the rest while still
// reporting a full write, so the helper never sees a broken pipe because of the cap.
type cappedBuffer struct {
	buf       []byte
	limit     int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	room := b.limit - len(b.buf)
	if room >= len(p) {
		b.buf = append(b.buf, p...)
		return len(p), nil
	}
	if room > 0 {
		b.buf = append(b.buf, p[:room]...)
	}
	b.truncated = true
	return len(p), nil
}
