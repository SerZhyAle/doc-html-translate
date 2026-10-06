package fdsec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"doc-html-translate/internal/outputpath"
)

// Plain is the one decrypted copy of a container: a single file with a neutral name inside a
// private folder. Remove deletes the folder and releases the locks; it is idempotent.
type Plain struct {
	// Path is the plain file. Ext is its lower-cased extension, taken from the true name
	// sealed in the container - the only part of that name this app uses.
	Path, Ext string

	once    sync.Once
	dir     string
	dirLock *outputpath.Lock
	oneCopy *outputpath.Lock
}

// Remove deletes the plain copy and gives up the one-copy lock.
func (p *Plain) Remove() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		p.dirLock.Release()
		removeAll(p.dir)
		p.oneCopy.Release()
	})
}

// removeAll retries briefly: a scanner may still hold the file for a moment.
func removeAll(dir string) {
	for i := 0; i < 5; i++ {
		if os.RemoveAll(dir) == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Unwrap opens a container with the installed FileDO and returns its single plain file under a
// neutral name. Everything decidable without the password is decided first: FileDO present, the
// credential-free length screen. convertible says which inner extensions this app converts. On
// every error the plain copy is already gone. The sealed true name is never logged, wrapped or
// returned; the container's visible base name is the only name an error carries.
func Unwrap(ctx context.Context, container string, ask Asker, convertible func(ext string) bool) (*Plain, error) {
	name := filepath.Base(container)
	filedo, err := Locate()
	if err != nil {
		return nil, &Error{Class: NotFound, Name: name}
	}
	info, err := os.Stat(container)
	if err != nil {
		return nil, &Error{Class: IO, Name: name}
	}
	if info.IsDir() || !ScreenLength(info.Size()) {
		return nil, &Error{Class: Screened, Name: name}
	}

	oneCopy, err := acquireOneCopy(ctx)
	if err != nil {
		return nil, withName(err, name)
	}
	dir, dirLock, err := newWorkDir()
	if err != nil {
		oneCopy.Release()
		return nil, withName(err, name)
	}
	p := &Plain{dir: dir, dirLock: dirLock, oneCopy: oneCopy}
	fail := func(err error) (*Plain, error) {
		p.Remove()
		return nil, err
	}

	retry := false
	for {
		secret, err := ask.Secret(name, retry)
		if err != nil {
			return fail(withName(err, name))
		}
		_, err = restore(ctx, filedo, container, dir, secret)
		if err == nil {
			break
		}
		var fe *Error
		if errors.As(err, &fe) && fe.Class == Credential && ask.CanRetry() {
			clearDir(dir)
			retry = true
			continue
		}
		return fail(err)
	}

	file, ext, err := inspect(dir, name, convertible)
	if err != nil {
		return fail(err)
	}
	neutral := filepath.Join(dir, neutralStem(name)+ext)
	if err := os.Rename(file, neutral); err != nil {
		return fail(&Error{Class: IO, Name: name})
	}
	p.Path, p.Ext = neutral, ext
	return p, nil
}

// withName stamps the container's name on a typed error that lacks one.
func withName(err error, name string) error {
	var fe *Error
	if errors.As(err, &fe) && fe.Name == "" {
		fe.Name = name
	}
	return err
}

// clearDir empties dir without removing it or its lock, after a failed attempt.
func clearDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.Name() == outputpath.LockName {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
}

// inspect finds what FileDO restored and classifies it. It reports the file's path and its
// lower-cased extension; the sealed name itself goes no further than this function.
func inspect(dir, name string, convertible func(string) bool) (string, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", &Error{Class: IO, Name: name}
	}
	var kept []os.DirEntry
	for _, e := range entries {
		if e.Name() != outputpath.LockName {
			kept = append(kept, e)
		}
	}
	if len(kept) != 1 {
		return "", "", &Error{Class: Failed, Name: name}
	}
	e := kept[0]
	if e.IsDir() {
		return "", "", &Error{Class: Folder, Name: name}
	}
	ext := strings.ToLower(filepath.Ext(e.Name()))
	switch {
	case IsContainer(ext):
		return "", "", &Error{Class: Nested, Name: name}
	case !convertible(ext):
		return "", "", &Error{Class: InnerType, Name: name, Inner: shownExt(ext)}
	}
	return filepath.Join(dir, e.Name()), ext, nil
}

// shownExt makes an extension safe to print: printable and short.
func shownExt(ext string) string {
	ext = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, ext)
	if ext == "" {
		return "?"
	}
	if utf8.RuneCountInString(ext) > 16 {
		ext = string([]rune(ext)[:16])
	}
	return ext
}

// reservedNames are device names Windows refuses as a file stem.
var reservedNames = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// neutralStem is the container's visible name without its extension, made safe as a file stem
// and kept short, so no helper downstream meets a long path and a title taken from the file
// name falls back to the container's own name.
func neutralStem(visible string) string {
	stem := strings.TrimSuffix(visible, filepath.Ext(visible))
	stem = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune("<>:\"/\\|?*", r) {
			return '_'
		}
		return r
	}, stem)
	if r := []rune(stem); len(r) > 64 {
		stem = string(r[:64])
	}
	stem = strings.TrimRight(stem, ". ")
	if stem == "" {
		stem = "document"
	}
	if reservedNames[strings.ToLower(stem)] {
		stem = "_" + stem
	}
	return stem
}
