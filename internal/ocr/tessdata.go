package ocr

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"doc-html-translate/internal/logging"
)

// LangInfo describes a supported OCR language.
type LangInfo struct {
	Code string
	Name string
}

// Available is the catalog of languages the app offers for download. It is the shared
// catalog with the extension's ocr-lang.js LANGS - keep the two in sync (see docs/PARITY.md).
var Available = []LangInfo{
	{"eng", "English"},
	{"rus", "Russian"},
	{"ukr", "Ukrainian"},
	{"jpn", "Japanese"},
	{"jpn_vert", "Japanese (vertical)"},
	{"deu", "German"},
	{"fra", "French"},
	{"spa", "Spanish"},
	{"ita", "Italian"},
	{"por", "Portuguese"},
	{"pol", "Polish"},
	{"chi_sim", "Chinese (simplified)"},
	{"kor", "Korean"},
}

// Bundled languages ship with the app so English OCR works offline out of the box.
// The eng.traineddata blob is not committed; scripts/build.ps1 provisions it into
// <exe>/tessdata at build time (copied from the extension's vendored copy, or downloaded
// from cdnBase). Both sources are tessdata_fast 4.0.0, so the bundled data matches the
// extension's (see docs/PARITY.md).
var Bundled = []string{"eng"}

// tessdata_fast plain (non-gzipped) files via GitHub raw - no decompression needed.
// Pinned to the 4.0.0 tag so downloaded languages match the extension, which loads
// tessdata_fast 4.0.0 from tesseract.js's CDN (ocr-lang.js CDN_LANG_PATH). Both sides
// must reference the same tessdata version - see docs/PARITY.md ("OCR").
const cdnBase = "https://github.com/tesseract-ocr/tessdata_fast/raw/4.0.0"

// appDirName is the per-user folder the app keeps writable state in - the same name the
// translator's key fallback and the report store use under %LOCALAPPDATA%, so all of the app's
// per-user state sits in one place.
const appDirName = "doc-html-translate"

// userDataDir is where downloads go: a per-user folder that is writable in every build flavour.
// The packaged (MSIX/Store) install directory is read-only, so no pack can be written next to
// the executable there. On Windows os.UserCacheDir is %LOCALAPPDATA%, the translator's precedent.
// A variable so tests can point it at a temp folder.
var userDataDir = func() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, appDirName, "tessdata")
}

// bundledDataDir is <exe dir>/tessdata: where the build provisions eng and where earlier versions
// installed downloads. Read-only under MSIX, so the app only ever reads from it.
var bundledDataDir = func() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "tessdata")
	}
	return "tessdata"
}

// UserDataDir is the writable per-user tessdata folder Download installs into.
func UserDataDir() string { return userDataDir() }

// DataDirs lists the tessdata folders the app reads, in lookup order: per-user, then bundled.
func DataDirs() []string {
	user, bundled := userDataDir(), bundledDataDir()
	if filepath.Clean(user) == filepath.Clean(bundled) {
		return []string{user}
	}
	return []string{user, bundled}
}

// DataDir returns the one folder to hand Tesseract as --tessdata-dir. Tesseract takes a single
// folder, so rus+eng cannot load when rus was downloaded into the per-user folder and eng is
// bundled next to the exe. A folder that already holds every pack is returned as is; only when
// both hold something is the bundled data copied into the per-user folder (once, a few MB), which
// then holds the union. A failed copy is logged and the per-user folder is still returned: its
// packs are the ones the user asked for.
func DataDir() string {
	user, bundled := userDataDir(), bundledDataDir()
	userPacks := packsIn(user)
	if len(userPacks) == 0 {
		if len(packsIn(bundled)) > 0 {
			return bundled
		}
		return user
	}
	if filepath.Clean(user) == filepath.Clean(bundled) {
		return user
	}
	have := make(map[string]bool, len(userPacks))
	for _, c := range userPacks {
		have[c] = true
	}
	for _, c := range packsIn(bundled) {
		if have[c] {
			continue
		}
		if err := copyInto(user, langFile(bundled, c)); err != nil {
			logging.RunLogf("OCR: could not stage %s into %s: %v\n", filepath.Base(langFile(bundled, c)), user, err)
		}
	}
	return user
}

func langFile(dir, code string) string { return filepath.Join(dir, code+".traineddata") }

// packsIn lists the language codes that have a traineddata file in dir.
func packsIn(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if code, ok := strings.CutSuffix(e.Name(), ".traineddata"); ok {
			out = append(out, code)
		}
	}
	return out
}

// copyInto copies src into dir under its own name through a unique temp file and a rename, so a
// reader - or a second process staging the same file - never sees half of it.
func copyInto(dir, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	tmp, err := os.CreateTemp(dir, filepath.Base(src)+"-*.tmp")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), filepath.Join(dir, filepath.Base(src)))
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
	}
	return err
}

// Installed lists the language codes present in any tessdata folder the app reads.
func Installed() []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range DataDirs() {
		for _, c := range packsIn(dir) {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	sort.Strings(out)
	return out
}

// IsInstalled reports whether a language's data is present in any tessdata folder the app reads.
func IsInstalled(code string) bool {
	for _, dir := range DataDirs() {
		if _, err := os.Stat(langFile(dir, code)); err == nil {
			return true
		}
	}
	return false
}

// iso2tess maps common ISO-639-1 codes (as used by the app's -src flag) to Tesseract
// traineddata names.
var iso2tess = map[string]string{
	"en": "eng", "ru": "rus", "uk": "ukr", "de": "deu", "fr": "fra",
	"es": "spa", "it": "ita", "pt": "por", "pl": "pol", "ja": "jpn",
	"zh": "chi_sim", "ko": "kor", "nl": "nld", "tr": "tur", "ar": "ara",
}

// TessLang converts a language code to a Tesseract traineddata name. A code that already
// looks like a Tesseract name (3+ letters, or contains "+") is returned unchanged.
func TessLang(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return "eng"
	}
	if t, ok := iso2tess[code]; ok {
		return t
	}
	return code
}

// ISOFor is the reverse of TessLang: the ISO-639-1 code that selects a Tesseract
// language, or "" when none maps to it. The GUI uses it to keep the OCR language
// following the -src language instead of duplicating the mapping in JavaScript.
func ISOFor(tess string) string {
	for iso, t := range iso2tess {
		if t == tess {
			return iso
		}
	}
	return ""
}

// LangName returns the display name for a code, or the code itself if unknown.
func LangName(code string) string {
	for _, l := range Available {
		if l.Code == code {
			return l.Name
		}
	}
	return code
}

// LangLabel renders a "+"-joined language string for a reader: each code keeps its
// traineddata name and gains the catalog's display name where there is one, so a line about
// "rus" also says Russian. The code stays first because it is what the user has to type back.
func LangLabel(lang string) string {
	codes := strings.Split(lang, "+")
	parts := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if name := LangName(c); name != c {
			c += " (" + name + ")"
		}
		parts = append(parts, c)
	}
	if len(parts) == 0 {
		return lang
	}
	return strings.Join(parts, " + ")
}
