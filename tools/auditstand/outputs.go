// outputs.go holds the helpers scenes share for reading what a CLI run left on disk.

package main

import (
	"os"
	"path/filepath"

	"doc-html-translate/internal/outputpath"
)

// bookDirUnder returns the book folder a run left under out. The CLI treats -folder as the
// parent: the book lands in <out>/<name>/ with its ownership record inside, so a scene that looks
// for the record at <out>/ itself would find nothing and report a state defect that is its own.
// It returns "" when no book folder holds a record.
func bookDirUnder(out string) string {
	entries, err := os.ReadDir(out)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(out, e.Name())
		if fileExists(filepath.Join(dir, outputpath.MarkerName)) {
			return dir
		}
	}
	return ""
}

// stringIn reports whether s is one of the accepted values (exact match).
func stringIn(s string, accepted []string) bool {
	for _, a := range accepted {
		if s == a {
			return true
		}
	}
	return false
}

// completionOf reads the finished-output record of the book under out: the translation state it
// stores and whether the record says the run finished at all.
func completionOf(out string) (translation string, complete bool, err error) {
	dir := bookDirUnder(out)
	if dir == "" {
		return "", false, os.ErrNotExist
	}
	m, err := outputpath.ReadMarker(dir)
	if err != nil {
		return "", false, err
	}
	if m.Complete == nil {
		return "", false, nil
	}
	return m.Complete.Translation, true, nil
}
