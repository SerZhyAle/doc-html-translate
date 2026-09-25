package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The GUI's small state files (settings, output history, the Google key) used to be
// rewritten in place. A crash mid-write left a truncated file that the next read treated as
// "nothing saved", and two GUI windows doing read-modify-write on the history dropped each
// other's entries. Writes now go to a temporary file that replaces the target in one rename,
// and read-modify-write cycles run under a lock that other instances honour too.

// writeFileAtomic replaces path with data, or leaves the old file untouched.
func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// storeMu serializes this process's own read-modify-write cycles; the lock file below
// extends that to other GUI instances.
var storeMu sync.Mutex

const (
	lockWait  = 5 * time.Second
	lockStale = 30 * time.Second // a lock older than any real update belongs to a dead process
)

var errStoreBusy = errors.New("settings store is busy")

// withFileLock runs fn while holding <path>.lock.
func withFileLock(path string, fn func() error) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	lock := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		return err
	}
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		if fi, statErr := os.Stat(lock); statErr == nil && time.Since(fi.ModTime()) > lockStale {
			_ = os.Remove(lock)
			continue
		}
		if time.Now().After(deadline) {
			return errStoreBusy
		}
		time.Sleep(20 * time.Millisecond)
	}
	defer os.Remove(lock)
	return fn()
}

// setAsideCorrupt moves an unreadable state file out of the way instead of letting the next
// save overwrite it, so what the user had can still be recovered by hand. It returns where
// the file went.
func setAsideCorrupt(path string) string {
	dest := fmt.Sprintf("%s.corrupt-%s", path, time.Now().Format("20060102-150405"))
	if err := os.Rename(path, dest); err != nil {
		return ""
	}
	return dest
}

// readSettings returns the saved settings blob. A file that exists but is not valid JSON is
// set aside, and its new location is returned as corruptAt so the page can say so.
func readSettings() (data []byte, corruptAt string) {
	p := settingsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, ""
	}
	if json.Valid(data) {
		return data, ""
	}
	_ = withFileLock(p, func() error {
		corruptAt = setAsideCorrupt(p)
		return nil
	})
	return nil, corruptAt
}

func writeSettings(data []byte) error {
	p := settingsPath()
	return withFileLock(p, func() error { return writeFileAtomic(p, data, 0o600) })
}

// loadParamsHistoryLocked reads the history; the caller holds its lock. An unreadable file
// is set aside rather than silently becoming an empty map that the next save writes over.
func loadParamsHistoryLocked() map[string]string {
	p := paramsHistoryPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil {
		setAsideCorrupt(p)
		return map[string]string{}
	}
	if m == nil {
		m = map[string]string{}
	}
	return m
}

func loadParamsHistory() map[string]string {
	var m map[string]string
	if err := withFileLock(paramsHistoryPath(), func() error {
		m = loadParamsHistoryLocked()
		return nil
	}); err != nil {
		return map[string]string{}
	}
	return m
}

// updateParamsHistory applies change to the history as one locked read-modify-write.
func updateParamsHistory(change func(map[string]string)) error {
	p := paramsHistoryPath()
	return withFileLock(p, func() error {
		m := loadParamsHistoryLocked()
		change(m)
		data, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return writeFileAtomic(p, data, 0o600)
	})
}
