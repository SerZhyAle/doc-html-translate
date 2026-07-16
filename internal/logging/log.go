// Package logging provides simple timestamp-prefixed console output.
package logging

import (
	"fmt"
	"os"
	"time"
)

// AppVersion is set by main before any pipeline output so it appears in the first log line.
var AppVersion = "dev"

var stdoutIsTerminal = detectStdoutTerminal()

func ts() string {
	return time.Now().Format("[15:04:05] ")
}

// Printf prints a timestamped message to stdout.
func Printf(format string, args ...any) {
	fmt.Printf(ts()+format, args...)
}

// Println prints a timestamped line to stdout.
func Println(s string) {
	fmt.Printf(ts()+"%s\n", s)
}

// Errorf prints a timestamped message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, ts()+format, args...)
}

// Progress prints an in-place progress update line (uses \r to overwrite, no trailing newline).
func Progress(format string, args ...any) {
	if stdoutIsTerminal {
		fmt.Printf("\r"+ts()+format, args...)
		return
	}
	// In non-interactive outputs (pipes/UI log capture), carriage return creates
	// broken lines. Emit normal timestamped lines instead.
	fmt.Printf(ts()+format+"\n", args...)
}

func detectStdoutTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Ticker rate-limits an in-place progress line for a long per-item loop, then closes it
// with a summary. Two constraints meet here: a loop that runs for minutes with no output
// is indistinguishable from a hang, but a line per item would push thousands of lines
// through the GUI's log pane, which streams the CLI's stdout and flushes per line. One
// line per second satisfies both. Shared by the PDF page loops and the OCR overlay.
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

// Report announces `done` of `total` items. Intermediate calls are dropped unless a second
// has passed since the last one, so callers can call it every iteration; done >= total
// always prints and carries the newline that releases the line for the next message.
func (t *Ticker) Report(done, total int) {
	if done < total && time.Since(t.last) < time.Second {
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

// Quiet reports whether the loop finished fast enough that no progress line was ever
// worth printing, so a caller can skip the closing summary for a trivial run.
func (t *Ticker) Quiet() bool { return time.Since(t.start) < time.Second }
