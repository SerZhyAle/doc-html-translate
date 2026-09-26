package ocr

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Tesseract and Leptonica open every path through the Windows ANSI code page and mangle any
// character outside it, failing the pass without saying why. So every path this package hands
// the engine - the image, a staged rendition of it, and --tessdata-dir - must be ASCII, or be
// turned into an ASCII form (an 8.3 short name) before it is handed over. The temp folder and the
// per-user data folder both sit under the user profile, so a profile named in Cyrillic loses OCR
// unless they are checked too, not only the book's own paths. The platform files supply the short
// name (shortPath) and the extra places to stage in (platformStagingRoots).

// stagingDirName is the subfolder a fallback root gets, so the app's files never sit loose in a
// shared folder such as %PUBLIC%.
const stagingDirName = "doc-html-translate-ocr"

// mirrorDirName is the folder under the staging root that receives a copy of the language data
// when the data folder itself has no ASCII form.
const mirrorDirName = "doc-html-translate-tessdata"

// stagingRoot is the folder temp images for Tesseract are written to, resolved once per process.
// A variable so a test can point it elsewhere.
var stagingRoot = sync.OnceValues(func() (string, error) {
	return resolveStagingRoot(stagingCandidates(), shortPath)
})

// stagingCandidates lists where images may be staged, best first: the system temp folder (where
// they always went), then the platform's shared ASCII roots, then a folder next to the executable.
func stagingCandidates() []string {
	out := []string{os.TempDir()}
	out = append(out, platformStagingRoots()...)
	if exe, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(exe), stagingDirName))
	}
	return out
}

// resolveStagingRoot returns the first candidate that exists (or can be made), has an ASCII form,
// and can be written to. The error names every folder tried, so a machine where none works says
// why OCR was skipped instead of failing image by image.
func resolveStagingRoot(candidates []string, short func(string) string) (string, error) {
	var tried []string
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tried = append(tried, dir)
			continue
		}
		safe, ok := asciiForm(dir, short)
		if !ok || !writable(safe) {
			tried = append(tried, dir)
			continue
		}
		return safe, nil
	}
	return "", fmt.Errorf("no writable folder with an ASCII path to hand Tesseract its images (tried %s)",
		strings.Join(tried, "; "))
}

// asciiForm returns p itself when it is ASCII, else its short name when that is ASCII. A volume
// with short names turned off gives the long name back, which fails the check - that is the case
// the fallback roots exist for.
func asciiForm(p string, short func(string) string) (string, bool) {
	if isASCIIPath(p) {
		return p, true
	}
	if s := short(p); isASCIIPath(s) {
		return s, true
	}
	return "", false
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, "docht-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name) == nil
}

func isASCIIPath(p string) bool {
	for _, r := range p {
		if r > 127 {
			return false
		}
	}
	return true
}

// PrepareEngine checks, once per book, that every path the engine will be handed can be made
// ASCII, and returns the form of dataDir to pass as --tessdata-dir. The error says which folder and
// why, so the caller skips OCR with that reason instead of failing on every image.
func PrepareEngine(dataDir string) (string, error) {
	if _, err := stagingRoot(); err != nil {
		return "", err
	}
	return engineDataDir(dataDir)
}

// engineDataDir returns dir itself or its short name when either is ASCII, else a copy of its
// language packs under the staging root. A folder with no packs is returned unchanged, because
// tesseractArgs never hands such a folder over.
func engineDataDir(dir string) (string, error) {
	packs := packsIn(dir)
	if len(packs) == 0 {
		return dir, nil
	}
	if safe, ok := asciiForm(dir, shortPath); ok {
		return safe, nil
	}
	root, err := stagingRoot()
	if err != nil {
		return "", fmt.Errorf("the OCR language folder %s has no ASCII path, and %w", dir, err)
	}
	mirror := filepath.Join(root, mirrorDirName)
	if err := mirrorPacks(dir, mirror, packs); err != nil {
		return "", err
	}
	return mirror, nil
}

// mirrorPacks copies the named packs from src into dst, skipping any already there at the same
// size - the packs are pinned by digest, so a same-size copy is the same file and a later run
// reuses it instead of copying megabytes again.
func mirrorPacks(src, dst string, packs []string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("could not create %s for the OCR language data: %w", dst, err)
	}
	for _, code := range packs {
		from := langFile(src, code)
		fi, err := os.Stat(from)
		if err != nil {
			return fmt.Errorf("could not read %s: %w", from, err)
		}
		if have, err := os.Stat(langFile(dst, code)); err == nil && have.Size() == fi.Size() {
			continue
		}
		if err := copyInto(dst, from); err != nil {
			return fmt.Errorf("could not copy %s into %s: %w", filepath.Base(from), dst, err)
		}
	}
	return nil
}

// writeTempPNG encodes an image to a temp PNG under the staging root and returns it with a cleanup
// func, removing a half-written file on failure. Shared by the passes that hand tesseract a
// derived image rather than the user's file.
func writeTempPNG(img image.Image) (path string, cleanup func(), ok bool) {
	root, err := stagingRoot()
	if err != nil {
		return "", nil, false
	}
	f, err := os.CreateTemp(root, "docht-ocr-*.png")
	if err != nil {
		return "", nil, false
	}
	name := f.Name()
	if err := png.Encode(f, img); err != nil {
		f.Close()
		_ = os.Remove(name)
		return "", nil, false
	}
	f.Close()
	return name, func() { _ = os.Remove(name) }, true
}

// stageASCIIPath returns a path safe to hand tesseract: the path itself when it is ASCII, its short
// name when that is, else a copy under the staging root. The cleanup func removes any copy and is
// never nil. A failed copy returns the original path, so the engine's own error reaches the report
// for that image. Mirrors internal/pdf's stagePDFForPDFToText.
func stageASCIIPath(imgPath string) (string, func()) {
	noop := func() {}
	if safe, ok := asciiForm(imgPath, shortPath); ok {
		return safe, noop
	}
	root, err := stagingRoot()
	if err != nil {
		return imgPath, noop
	}
	ext := filepath.Ext(imgPath)
	if !isASCIIPath(ext) {
		ext = "" // tesseract sniffs the format from content; a mangled ext is worse than none
	}
	f, err := os.CreateTemp(root, "docht-ocr-*"+ext)
	if err != nil {
		return imgPath, noop
	}
	name := f.Name()
	src, err := os.Open(imgPath)
	if err != nil {
		f.Close()
		os.Remove(name)
		return imgPath, noop
	}
	_, cerr := io.Copy(f, src)
	_ = src.Close()
	f.Close()
	if cerr != nil {
		os.Remove(name)
		return imgPath, noop
	}
	return name, func() { os.Remove(name) }
}
