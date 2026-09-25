package outputpath

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// MarkerName is the ownership record the converter writes into every output directory it
// creates. Deletion (failure cleanup, -force, the GUI's "delete previous result") is only
// ever allowed on a directory that carries it: before it existed, a failed run deleted
// whatever folder merely shared the book's name - including the input folder itself.
const MarkerName = ".doc-html-translate.json"

const markerTool = "doc-html-translate"

// Marker is the content of MarkerName. Source is the absolute input path; size and mtime
// are recorded so a later change can tell a rebuilt source from the one converted.
type Marker struct {
	Tool          string    `json:"tool"`
	Source        string    `json:"source"`
	SourceSize    int64     `json:"sourceSize"`
	SourceModTime time.Time `json:"sourceModTime"`
	Created       time.Time `json:"created"`
}

// State is what an existing path means for a conversion of a given source.
type State int

const (
	StateAbsent       State = iota // nothing there - a run may create it
	StateEmpty                     // an empty directory - usable, but not ours to delete
	StateOwned                     // carries our marker for this very source
	StateLegacy                    // pre-marker output of ours, source unknown or matching
	StateOwnedByOther              // our output, but for a different source document
	StateForeign                   // anything else: a user folder, a file, a symlink
)

// Ours reports whether the converter may reuse, rebuild or delete the directory.
func (s State) Ours() bool { return s == StateOwned || s == StateLegacy }

// Usable reports whether a conversion of the source may write into the directory.
func (s State) Usable() bool { return s == StateAbsent || s == StateEmpty || s.Ours() }

// Target is the resolved output location for one source.
type Target struct {
	Dir   string
	State State
	// Preferred is the plain name-derived location. It differs from Dir when that
	// location was taken by something else and a suffixed sibling was chosen.
	Preferred string
}

// maxCandidates bounds the "book (pdf) 2", "book (pdf) 3".. search. Hitting it means
// dozens of same-named foreign folders, which is better reported than silently dodged.
const maxCandidates = 20

// Resolve picks the output directory for input. It starts from the name-derived
// location (OutputDirFor) and moves to "<name> (<ext>)", then "<name> (<ext>) N", when
// the location belongs to a different document, is not ours, or would contain the input
// itself. It only reads the filesystem, so the GUI can call it to find a previous result
// and gets the same answer the CLI will.
func Resolve(input, folder string) (Target, error) {
	preferred := OutputDirFor(input, folder)
	label := strings.ToLower(strings.TrimPrefix(filepath.Ext(filepath.Base(input)), "."))
	if label == "" {
		label = "file"
	}
	label = sanitizeOutputName(label)
	for i := 0; i < maxCandidates; i++ {
		dir := preferred
		switch {
		case i == 1:
			dir = preferred + " (" + label + ")"
		case i > 1:
			dir = fmt.Sprintf("%s (%s) %d", preferred, label, i)
		}
		if containsOrEqual(dir, input) {
			continue
		}
		st := Inspect(dir, input)
		if st.Usable() {
			return Target{Dir: dir, State: st, Preferred: preferred}, nil
		}
	}
	return Target{}, fmt.Errorf("no free output folder next to %s: %s and %d suffixed variants are all taken by other content",
		filepath.Base(input), preferred, maxCandidates-1)
}

// Inspect classifies dir with respect to a conversion of source.
func Inspect(dir, source string) State {
	fi, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return StateAbsent
	}
	if err != nil || !fi.IsDir() {
		return StateForeign
	}

	if data, err := os.ReadFile(filepath.Join(dir, MarkerName)); err == nil {
		var m Marker
		if json.Unmarshal(data, &m) != nil || m.Tool != markerTool {
			return StateForeign
		}
		if samePath(m.Source, source) {
			return StateOwned
		}
		return StateOwnedByOther
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return StateForeign
	}
	meaningful := 0
	for _, e := range entries {
		if e.Name() != LockName {
			meaningful++
		}
	}
	if meaningful == 0 {
		return StateEmpty
	}
	return inspectLegacy(dir, source)
}

// legacySignature marks pages our generator writes: every reader layer, navbar and
// merged page carries dht- prefixed ids/classes. A saved website or a user folder that
// merely has an index.html does not.
const legacySignature = "dht-"

// legacyScanBytes bounds how much of index.html is read: the signature and the navbar's
// source-file label both sit in the head/first header, and merged pages can be huge.
const legacyScanBytes = 256 << 10

var redirectTarget = regexp.MustCompile(`location\.replace\("([^"]+)"\)`)

var navFileTitle = regexp.MustCompile(`class="nav-file" title="([^"]*)"`)

// inspectLegacy recognises outputs written before the ownership marker existed, so they
// keep opening (and can still be rebuilt with -force) instead of being orphaned.
func inspectLegacy(dir, source string) State {
	head := readHead(filepath.Join(dir, "index.html"))
	if head == "" {
		return StateForeign
	}
	// An EPUB with a base folder gets a redirect stub at the root; the real page is one
	// hop away and must stay inside dir.
	if m := redirectTarget.FindStringSubmatch(head); m != nil && !strings.Contains(head, legacySignature) {
		target := filepath.Join(dir, filepath.FromSlash(m[1]))
		if !containsOrEqual(dir, target) || samePath(dir, target) {
			return StateForeign
		}
		head = readHead(target)
	}
	if !strings.Contains(head, legacySignature) {
		return StateForeign
	}
	// The navbar names the source file. When it is present and names another document,
	// this is the book.epub output that book.pdf must not take over.
	if m := navFileTitle.FindStringSubmatch(head); m != nil {
		if !sameName(html.UnescapeString(m[1]), filepath.Base(source)) {
			return StateOwnedByOther
		}
	}
	return StateLegacy
}

func readHead(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf, _ := io.ReadAll(io.LimitReader(f, legacyScanBytes))
	return string(buf)
}

// WriteMarker records that dir is the output of source.
func WriteMarker(dir, source string) error {
	m := Marker{Tool: markerTool, Source: source, Created: time.Now().UTC()}
	if fi, err := os.Stat(source); err == nil {
		m.SourceSize = fi.Size()
		m.SourceModTime = fi.ModTime().UTC()
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(dir, MarkerName)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return err
	}
	hideFile(p)
	return nil
}

// ClearContents empties dir while keeping the directory itself and the named entries
// (the run's own lock). Used where the directory predates this run and so is not this
// run's to remove.
func ClearContents(dir string, keep ...string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var firstErr error
	for _, e := range entries {
		if contains(keep, e.Name()) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// containsOrEqual reports whether path is dir itself or lies inside it. An output
// directory like that would have the input (or its folder) deleted on cleanup.
func containsOrEqual(dir, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		// filepath.Rel is case-sensitive on the path components; NTFS is not.
		if strings.EqualFold(filepath.Clean(dir), filepath.Clean(path)) {
			return true
		}
		d := strings.ToLower(filepath.Clean(dir)) + string(os.PathSeparator)
		if strings.HasPrefix(strings.ToLower(filepath.Clean(path)), d) {
			return true
		}
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func sameName(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
