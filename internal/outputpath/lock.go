package outputpath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LockName is the per-run lock inside an output directory. Two runs on one output (a
// double-click plus the context-menu verb, two GUI jobs) used to extract into the same
// folder, and one run's failure cleanup deleted the other's work mid-write.
const LockName = ".doc-html-translate.lock"

// staleLockAge is the backstop for a lock whose process cannot be checked (or whose pid
// was reused): no conversion runs for a day.
const staleLockAge = 24 * time.Hour

// ErrLocked means another live process is converting into the same directory.
var ErrLocked = errors.New("output folder is being written by another conversion")

// Lock is a held output-directory lock.
type Lock struct{ path string }

// AcquireLock takes the exclusive lock in dir, clearing one left behind by a process
// that is no longer running.
func AcquireLock(dir string) (*Lock, error) {
	p := filepath.Join(dir, LockName)
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, werr := fmt.Fprintf(f, "%d\n", os.Getpid())
			cerr := f.Close()
			if werr != nil || cerr != nil {
				_ = os.Remove(p)
				return nil, errors.Join(werr, cerr)
			}
			hideFile(p)
			return &Lock{path: p}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if !lockIsStale(p) {
			return nil, ErrLocked
		}
		_ = os.Remove(p)
	}
	return nil, ErrLocked
}

// Locked reports whether dir holds a live lock - a reuse check uses it so it does not
// open an output another process is still writing.
func Locked(dir string) bool {
	p := filepath.Join(dir, LockName)
	if _, err := os.Stat(p); err != nil {
		return false
	}
	return !lockIsStale(p)
}

// Release drops the lock. It tolerates the file being gone already: failure cleanup
// removes the whole directory, lock included.
func (l *Lock) Release() {
	if l == nil {
		return
	}
	_ = os.Remove(l.path)
}

func lockIsStale(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return true
	}
	if time.Since(fi.ModTime()) > staleLockAge {
		return true
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		// Mid-write by its owner, or garbage; only the age backstop may clear it.
		return false
	}
	return !processAlive(pid)
}
