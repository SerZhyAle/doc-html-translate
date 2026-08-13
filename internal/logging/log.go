// Package logging provides simple timestamp-prefixed console output.
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// AppVersion is set by main before any pipeline output so it appears in the first log line.
var AppVersion = "dev"

var stdoutIsTerminal = detectStdoutTerminal()

// The run log is a side channel: the console output is unchanged whether or not one is
// installed, and a failing sink is ignored rather than reported, because a log that cannot be
// written must never be able to fail the conversion it was only observing.
var (
	runLogMu sync.Mutex
	runLog   io.Writer
)

// StartRunLog tees every subsequent log line into w until StopRunLog.
func StartRunLog(w io.Writer) {
	runLogMu.Lock()
	defer runLogMu.Unlock()
	runLog = w
}

// StopRunLog detaches the run log sink.
func StopRunLog() {
	runLogMu.Lock()
	defer runLogMu.Unlock()
	runLog = nil
}

// emit prints console to dst and appends logLine to the run log when one is installed. The
// two texts differ only where the console uses in-place redrawing, which a file must not
// inherit. The write is done under the lock so concurrent loops cannot interleave a line.
func emit(dst io.Writer, console, logLine string) {
	fmt.Fprint(dst, console)
	runLogMu.Lock()
	defer runLogMu.Unlock()
	if runLog != nil {
		_, _ = io.WriteString(runLog, logLine)
	}
}

func ts() string {
	return time.Now().Format("[15:04:05] ")
}

// Printf prints a timestamped message to stdout.
func Printf(format string, args ...any) {
	line := ts() + fmt.Sprintf(format, args...)
	emit(os.Stdout, line, line)
}

// Println prints a timestamped line to stdout.
func Println(s string) {
	line := ts() + s + "\n"
	emit(os.Stdout, line, line)
}

// Errorf prints a timestamped message to stderr.
func Errorf(format string, args ...any) {
	line := ts() + fmt.Sprintf(format, args...)
	emit(os.Stderr, line, line)
}

// Progress prints an in-place progress update line (uses \r to overwrite, no trailing newline).
func Progress(format string, args ...any) {
	line := ts() + fmt.Sprintf(format, args...)
	// The run log always gets a whole line: the padding an in-place update uses to erase the
	// previous, longer text is noise once the text is in a file.
	logLine := strings.TrimRight(line, " \n") + "\n"
	if stdoutIsTerminal {
		emit(os.Stdout, "\r"+line, logLine)
		return
	}
	// In non-interactive outputs (pipes/UI log capture), carriage return creates
	// broken lines. Emit normal timestamped lines instead.
	emit(os.Stdout, line+"\n", logLine)
}

// StdoutIsTerminal reports whether stdout is an interactive console rather than a pipe, a
// file, or a stream captured by a parent process.
//
// Use it to gate a *pause* - "press Enter so the window does not vanish" only serves a human
// looking at a console, and blocks forever when the output is piped into a script, CI, or the
// GUI's shell-out. Do not use it to gate a *question*: whether an answer can be read depends
// on stdin, not stdout, and reading one already handles EOF by taking the safe default.
func StdoutIsTerminal() bool { return stdoutIsTerminal }

func detectStdoutTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// progressInterval is how often a Ticker is allowed to speak. A loop that runs for
// minutes in silence is indistinguishable from a hang, but every line is flushed
// separately into the GUI's log pane, and a rate + ETA that redraws every second is
// noise: the numbers barely move and they bury the lines around them. Four seconds is
// often enough to prove the run is alive and rare enough to stay readable.
const progressInterval = 4 * time.Second

// Ticker rate-limits an in-place progress line for a long per-item loop, then closes it
// with a summary. Shared by the PDF page loops and the OCR overlay.
type Ticker struct {
	label string
	unit  string
	start time.Time
	last  time.Time
}

// NewTicker starts timing a loop. label names the phase ("Extracting images"), unit names
// what is counted ("pages").
func NewTicker(label, unit string) *Ticker {
	now := time.Now()
	return &Ticker{label: label, unit: unit, start: now, last: now}
}

// Report announces `done` of `total` items. Intermediate calls are dropped unless
// progressInterval has passed since the last one, so callers can call it every iteration;
// done >= total always prints and carries the newline that releases the line for the next
// message.
func (t *Ticker) Report(done, total int) {
	if done < total && time.Since(t.last) < progressInterval {
		return
	}
	t.last = time.Now()
	elapsed := time.Since(t.start).Seconds()
	rate := 0.0
	if elapsed > 0 {
		rate = float64(done) / elapsed
	}
	if done >= total {
		Progress("  %s: %d %s in %.1fs (%.1f/s)     \n", t.label, total, t.unit, elapsed, rate)
		return
	}
	eta := ""
	if rate > 0 {
		remaining := time.Duration(float64(total-done) / rate * float64(time.Second))
		eta = fmt.Sprintf(" ETA %s", remaining.Round(time.Second))
	}
	Progress("  %s: %d/%d %s  %.1f/s%s     ", t.label, done, total, t.unit, rate, eta)
}

// Quiet reports whether the loop finished before Report ever spoke, so a caller can skip
// the closing summary for a run too short to have looked stuck.
func (t *Ticker) Quiet() bool { return time.Since(t.start) < progressInterval }
