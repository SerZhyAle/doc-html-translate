// Package report keeps a bounded history of run logs and packs them, with an environment
// summary, into a single archive the user can mail to the author.
//
// Nothing here ever sends anything: the archive is a file on the user's disk and the user
// decides what happens to it.
package report

import (
	"errors"
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

// RunLogPath names the log file of a run started at `at`. The timestamp is a parameter
// rather than a clock read so callers and tests agree on the name.
func RunLogPath(at time.Time) string {
	return filepath.Join(LogsDir(), "run-"+at.Format("20060102-150405")+".log")
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
