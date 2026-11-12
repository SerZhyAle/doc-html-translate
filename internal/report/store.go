// Package report keeps a bounded history of run logs and packs them, with an environment
// summary, into a single archive the user can mail to the author.
//
// Nothing here ever sends anything: the archive is a file on the user's disk and the user
// decides what happens to it.
package report

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// The store is bounded twice over: a heavy user converting large scanned documents produces
// far bigger logs than a light one, so a file count alone would not cap the disk and a byte
// budget alone would keep a single huge run and drop the history around it.
const (
	// MaxLogFiles is how many run logs are kept.
	MaxLogFiles = 20
	// MaxLogBytes is the total size the store may occupy.
	MaxLogBytes = 20 << 20
	// MaxRunLogBytes caps a single run's log. Without it one runaway run (a verbose OCR of a
	// thousand-page scan) could outgrow MaxLogBytes on its own and make Trim evict the whole
	// history it was meant to sit beside.
	MaxRunLogBytes = MaxLogBytes / 4
	// RunLogHeadBytes is how much of a run's start a compacted log keeps: the arguments, the
	// settings and the first pages, which say what the run was asked to do.
	RunLogHeadBytes = 1 << 20
	// compactedMarkerRoom is reserved for the marker line (three decimal int64 fields stay under
	// 160 bytes), so head + marker + tail never exceed MaxRunLogBytes.
	compactedMarkerRoom = 256
	// runLogTailBytes is how much of a run's end a compacted log keeps: the failure sequence.
	runLogTailBytes = MaxRunLogBytes - RunLogHeadBytes - compactedMarkerRoom
)

// Dir is the per-user, writable root of the report state. It mirrors the choice of
// %LOCALAPPDATA% made by the GUI's settingsPath and by the Google-key location: the packaged
// MSIX/Store install directory is read-only, so nothing may be written next to the binary.
func Dir() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "doc-html-translate")
	}
	return filepath.Join(os.TempDir(), "doc-html-translate")
}

// LogsDir is where per-run log files accumulate.
func LogsDir() string {
	return filepath.Join(Dir(), "logs")
}

// RunLogPath names the log file of a run started at `at` by process pid. The timestamp is a
// parameter rather than a clock read so callers and tests agree on the name. The pid keeps two
// runs started in the same second - a batch over a folder does exactly that - out of one file,
// where O_APPEND would interleave their lines.
func RunLogPath(at time.Time, pid int) string {
	return filepath.Join(LogsDir(), fmt.Sprintf("run-%s-%d.log", at.Format("20060102-150405"), pid))
}

// GUILogPath names the log of one GUI launch started at `at` by process pid. It sits in the run
// log store under a run- name, so it is sent, trimmed and cleared with the run logs, in the same
// chronological order.
func GUILogPath(at time.Time, pid int) string {
	return filepath.Join(LogsDir(), fmt.Sprintf("run-%s-gui%d.log", at.Format("20060102-150405"), pid))
}

// CapRunLog wraps a run log so it never holds more than MaxRunLogBytes. The first
// RunLogHeadBytes go to w as they come; everything after is held in memory, whole lines only,
// and Finish writes the last runLogTailBytes of it behind one DIAGNOSTIC-REPORT marker naming
// what was dropped from the middle. A run that fits the head is written through untouched, and
// a run that fits head + tail loses nothing. Every write reports success: the run log is a side
// channel whose failure must never reach the conversion. Not safe for concurrent use on its
// own; logging serializes its writes.
//
// The caller must call Finish before the process exits: os.Exit skips deferred calls, and a
// process killed from outside (the GUI's Cancel) loses the held tail, so such a log ends at its head.
func CapRunLog(w io.Writer) *CappedLog {
	return &CappedLog{w: w, headLeft: RunLogHeadBytes}
}

// CappedLog is the run log CapRunLog returns.
type CappedLog struct {
	w         io.Writer
	headLeft  int // bytes the head may still take
	headKept  int
	tailing   bool // the head is over: every later line goes to tail
	tail      [][]byte
	tailBytes int
	dropped   int64
	finished  bool
}

func (c *CappedLog) Write(p []byte) (int, error) {
	if c.finished {
		_, _ = c.w.Write(p)
		return len(p), nil
	}
	if !c.tailing {
		if len(p) <= c.headLeft {
			c.headLeft -= len(p)
			c.headKept += len(p)
			_, _ = c.w.Write(p)
			return len(p), nil
		}
		// The head ends at the first line that does not fit whole, so no line is ever cut.
		c.tailing = true
	}
	if len(p) > runLogTailBytes {
		c.dropped += int64(len(p))
		return len(p), nil
	}
	c.tail = append(c.tail, append([]byte(nil), p...))
	c.tailBytes += len(p)
	for c.tailBytes > runLogTailBytes {
		c.tailBytes -= len(c.tail[0])
		c.dropped += int64(len(c.tail[0]))
		c.tail[0] = nil
		c.tail = c.tail[1:]
	}
	return len(p), nil
}

// Finish writes the held tail, behind the marker when anything was dropped, and ends the
// compaction; a later write goes straight through. Calling it again does nothing.
func (c *CappedLog) Finish() {
	if c.finished {
		return
	}
	c.finished = true
	if c.dropped > 0 {
		_, _ = fmt.Fprintf(c.w, "[Diag] LOG COMPACTED | dropped_middle_bytes=%d | kept_head_bytes=%d | kept_tail_bytes=%d\n",
			c.dropped, c.headKept, c.tailBytes)
	}
	for _, line := range c.tail {
		_, _ = c.w.Write(line)
	}
	c.tail = nil
}

// Trim deletes the oldest run logs until both bounds hold, and reports how many it removed.
// A store that does not exist yet is already within its bounds, not an error.
func Trim() (removed int, err error) {
	entries, err := logFiles()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	// Names carry a sortable timestamp, so name order is chronological order.
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	var total int64
	for _, e := range entries {
		total += e.size
	}
	for i := 0; i < len(entries) && (len(entries)-i > MaxLogFiles || total > MaxLogBytes); i++ {
		if rmErr := os.Remove(filepath.Join(LogsDir(), entries[i].name)); rmErr != nil {
			if !errors.Is(rmErr, fs.ErrNotExist) {
				err = rmErr
			}
			continue
		}
		total -= entries[i].size
		removed++
	}
	return removed, err
}

// ClearLogs empties the store and leaves the directory behind, so the next run does not have
// to recreate it. A missing directory is already empty.
func ClearLogs() error {
	entries, err := logFiles()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if rmErr := os.Remove(filepath.Join(LogsDir(), e.name)); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = rmErr
		}
	}
	return err
}

type logFile struct {
	name string
	size int64
}

// logFiles lists the store's regular files with their sizes. An entry that vanished between
// the listing and the stat is simply gone - another process trimmed it.
func logFiles() ([]logFile, error) {
	entries, err := os.ReadDir(LogsDir())
	if err != nil {
		return nil, err
	}
	files := make([]logFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, logFile{name: e.Name(), size: info.Size()})
	}
	return files, nil
}
