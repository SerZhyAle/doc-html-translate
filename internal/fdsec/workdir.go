package fdsec

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"doc-html-translate/internal/outputpath"
)

// workRootOverride lets a test point the work root at its own folder.
var workRootOverride string

// workRoot is the product-named folder inside the user's temp folder that holds the per-run
// folders. It is not a folder of the per-user application-data root: a packaged process's new
// folder there is redirected into the package's private store and FileDO would not see what it
// wrote (ticket 94 research note, section 4.1). The temp folder is the one place measured to be
// the same real path for packaged and unpackaged programs alike.
func workRoot() string {
	if workRootOverride != "" {
		return workRootOverride
	}
	return filepath.Join(os.TempDir(), "doc-html-translate-fdsec")
}

// staleAfter is how old an unlocked per-run folder must be before the sweep removes it, so the
// sweep never races a run that has just created its folder.
const staleAfter = time.Minute

// acquireOneCopy takes the lock that lets at most one plain copy exist at a time, waiting while
// another run (a parallel GUI worker, another process) holds it. FDSEC-BEHAVIOUR: nothing
// bulk - a batch of plain copies on disk at once is the shape of malware staging.
func acquireOneCopy(ctx context.Context) (*outputpath.Lock, error) {
	if err := os.MkdirAll(workRoot(), 0o700); err != nil {
		return nil, &Error{Class: IO}
	}
	for {
		l, err := outputpath.AcquireLock(workRoot())
		if err == nil {
			return l, nil
		}
		if !errors.Is(err, outputpath.ErrLocked) {
			return nil, &Error{Class: IO}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// newWorkDir creates the private per-run folder (8 random hex characters, restricted to the
// current user) and locks it. The caller must already hold the one-copy lock.
func newWorkDir() (string, *outputpath.Lock, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", nil, &Error{Class: IO}
	}
	dir := filepath.Join(workRoot(), hex.EncodeToString(b[:]))
	if err := restrictToUser(dir); err != nil {
		return "", nil, &Error{Class: IO}
	}
	l, err := outputpath.AcquireLock(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, &Error{Class: IO}
	}
	return dir, l, nil
}

// SweepStale removes per-run folders a killed or crashed run left behind - a plain copy must
// not outlive its run. It runs at the start of the CLI and the GUI, never returns an error and
// never reports a path below the work root.
func SweepStale() {
	entries, err := os.ReadDir(workRoot())
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(workRoot(), e.Name())
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < staleAfter {
			continue
		}
		if outputpath.Locked(sub) {
			continue
		}
		_ = os.RemoveAll(sub)
	}
}
