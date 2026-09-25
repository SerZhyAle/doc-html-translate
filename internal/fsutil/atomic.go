// Package fsutil holds the one file-replacement primitive the conversion pipeline writes pages
// with. A page is rewritten several times in one run (navigation, OCR overlay, translation, TOC
// anchors), and an in-place truncating write that is interrupted - Ctrl+C, a crash, a full disk -
// leaves a half page the browser shows as the book. Writing a sibling temp file and renaming it
// over the target makes every rewrite all-or-nothing: the old page or the new one, never a mix.
package fsutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// WriteFile replaces path with data atomically. perm applies to the new file.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return Write(path, perm, func(w io.Writer) error {
		_, err := io.Copy(w, bytes.NewReader(data))
		return err
	})
}

// Write streams the new content of path through fill into a temp file in the same directory,
// then renames it over path. If fill fails, path is left exactly as it was.
//
// No fsync: the failure this guards against is the process dying mid-write, which a rename
// already survives, and a flush per page costs whole seconds on a book of thousands of pages.
func Write(path string, perm os.FileMode, fill func(io.Writer) error) (err error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	// Same directory, so the rename never crosses a volume; a leading dot keeps a stray temp
	// (left only by a hard kill) out of the way, and a rebuild clears the folder anyway.
	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()
	if err = fill(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return rename(tmpName, path)
}

// renameAttempts bounds the retry below. Windows refuses to replace a file that another
// process holds open without delete sharing - an antivirus scan or an indexer touching a page
// just written - and such a hold is released within milliseconds.
const renameAttempts = 5

func rename(from, to string) error {
	var err error
	for i := 0; i < renameAttempts; i++ {
		if err = os.Rename(from, to); err == nil || runtime.GOOS != "windows" {
			return err
		}
		time.Sleep(time.Duration(i+1) * 20 * time.Millisecond)
	}
	return err
}
