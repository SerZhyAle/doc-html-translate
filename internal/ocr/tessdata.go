package ocr

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

// DataDir is the tessdata directory the app manages: <exe dir>/tessdata.
func DataDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "tessdata")
	}
	return "tessdata"
}

func langFile(dir, code string) string { return filepath.Join(dir, code+".traineddata") }

// Installed lists the language codes present in the tessdata directory.
func Installed() []string {
	entries, err := os.ReadDir(DataDir())
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
	sort.Strings(out)
	return out
}

// IsInstalled reports whether a language's data is present locally.
func IsInstalled(code string) bool {
	_, err := os.Stat(langFile(DataDir(), code))
	return err == nil
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

// Download fetches a language's traineddata into the tessdata directory. It writes to a
// temp file and renames on success so a partial download never looks installed.
func Download(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("empty language code")
	}
	dir := DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create tessdata dir: %w", err)
	}
	url := fmt.Sprintf("%s/%s.traineddata", cdnBase, code)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", code, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d (unknown language code?)", code, resp.StatusCode)
	}
	tmp := langFile(dir, code) + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("download %s: %w", code, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, langFile(dir, code))
}
