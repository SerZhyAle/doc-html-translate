package bundledtools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
)

// CacheRoot is the per-user folder bundled tools are unpacked under. It is what a user
// excludes from antivirus scanning when a scanner blocks the bundled pdftotext.
func CacheRoot() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "doc-html-translate")
}

// hashLen is how much of the content hash names a cache folder: enough to tell builds apart,
// short enough to keep paths readable.
const hashLen = 12

// setHash fingerprints every file under root in fsys, names included, so any change to the
// bundled set - a new pdftotext, one more DLL - yields a new folder.
func setHash(fsys fs.FS, root string) (string, []string, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return "", nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		data, err := fs.ReadFile(fsys, path.Join(root, name))
		if err != nil {
			return "", nil, err
		}
		_, _ = fmt.Fprintf(h, "%s\x00%d\x00", name, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:hashLen], names, nil
}

// extractSet unpacks the files under root in fsys into <base>/<name>-<hash>/ and returns that
// folder. The folder is keyed by the content hash, so an upgraded app never runs the previous
// build's copy. Each file is written under a temporary name in the same folder and renamed into
// place, so a crash or a second instance never sees a half-written executable. A file already
// in place with the right bytes is left alone: on Windows it may be running in another
// instance, and replacing a running executable fails.
func extractSet(fsys fs.FS, root, base, name string) (string, error) {
	hash, names, err := setHash(fsys, root)
	if err != nil {
		return "", fmt.Errorf("read embedded %s: %w", name, err)
	}
	dir := filepath.Join(base, name+"-"+hash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s cache dir: %w", name, err)
	}
	for _, file := range names {
		data, err := fs.ReadFile(fsys, path.Join(root, file))
		if err != nil {
			return "", fmt.Errorf("read embedded %s: %w", file, err)
		}
		if err := placeFile(filepath.Join(dir, file), data); err != nil {
			return "", fmt.Errorf("write %s: %w", file, err)
		}
	}
	pruneOldSets(base, name, filepath.Base(dir))
	return dir, nil
}

// placeFile makes dst hold exactly data.
func placeFile(dst string, data []byte) error {
	if sameContent(dst, data) {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmpName, 0o755)
	}
	if werr != nil {
		_ = os.Remove(tmpName)
		return werr
	}
	if err := os.Rename(tmpName, dst); err != nil {
		_ = os.Remove(tmpName)
		// Another instance got there first and the file is now locked by a running copy;
		// what matters is that the right bytes are in place.
		if sameContent(dst, data) {
			return nil
		}
		return err
	}
	return nil
}

func sameContent(p string, data []byte) bool {
	existing, err := os.ReadFile(p)
	return err == nil && bytes.Equal(existing, data)
}

// pruneOldSets removes the folders earlier builds unpacked, including the unversioned one from
// before the hash was part of the name. Best-effort: a folder still in use by a running older
// instance cannot be removed on Windows, and the next run will try again.
func pruneOldSets(base, name, keep string) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	versioned := regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `-[0-9a-f]{` + fmt.Sprint(hashLen) + `}$`)
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() || n == keep || (n != name && !versioned.MatchString(n)) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(base, n))
	}
}
