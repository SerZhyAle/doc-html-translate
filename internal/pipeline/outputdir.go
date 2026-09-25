package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"doc-html-translate/internal/outputpath"
)

// outputClaim is a run's hold on its output directory: the lock, and what this run may
// delete when it fails. The rule it enforces is that a run only ever deletes what it
// created or what carries its own ownership marker - never a folder that merely shares
// the book's name.
type outputClaim struct {
	dir     string
	created bool // this run made the directory, so failure removes it outright
	lock    *outputpath.Lock
}

// claimOutputDir creates or takes over target.Dir for a build from source, locks it and
// writes the ownership marker. An existing output of ours (a -force rebuild, or the
// remains of an interrupted run) is emptied first; a pre-existing empty folder is used
// but, not being ours, is only ever emptied again - never removed.
func claimOutputDir(target outputpath.Target, source string) (*outputClaim, error) {
	c := &outputClaim{dir: target.Dir}
	if err := os.MkdirAll(filepath.Dir(target.Dir), 0o755); err != nil {
		return nil, fmt.Errorf("create output parent: %w", err)
	}
	// Mkdir, not MkdirAll: only its success proves this run created the directory. A
	// concurrent run that got there first leaves ErrExist, and then the lock decides.
	switch err := os.Mkdir(target.Dir, 0o755); {
	case err == nil:
		c.created = true
	case !errors.Is(err, os.ErrExist):
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	lock, err := outputpath.AcquireLock(target.Dir)
	if err != nil {
		// Deliberately no cleanup: without the lock this run owns nothing in there,
		// even a directory it just created may already be another run's output.
		return nil, fmt.Errorf("%s: %w", target.Dir, err)
	}
	c.lock = lock

	// Re-check under the lock: a run that finished while we waited may have claimed
	// the directory for another document.
	switch st := outputpath.Inspect(target.Dir, source); {
	case c.created, st == outputpath.StateEmpty:
	case st.Ours():
		if err := outputpath.ClearContents(target.Dir, outputpath.LockName); err != nil {
			c.release()
			return nil, fmt.Errorf("clear previous output: %w", err)
		}
	default:
		c.release()
		return nil, fmt.Errorf("%s now holds other content - not writing into it", target.Dir)
	}

	if err := outputpath.WriteMarker(target.Dir, source); err != nil {
		c.cleanup()
		c.release()
		return nil, fmt.Errorf("write output marker: %w", err)
	}
	return c, nil
}

// cleanup undoes a failed build. It runs only while the lock is held.
func (c *outputClaim) cleanup() {
	if c.created {
		_ = os.RemoveAll(c.dir)
		return
	}
	_ = outputpath.ClearContents(c.dir, outputpath.LockName)
}

func (c *outputClaim) release() {
	c.lock.Release()
}
