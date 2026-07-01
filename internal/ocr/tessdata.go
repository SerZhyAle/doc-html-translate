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

// Available is the catalog of languages the app offers for download. It mirrors the
// extension's set plus a few common Latin/CJK ones.
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

// Bundled languages are meant to ship with the app (English works offline out of the box).
var Bundled = []string{"eng"}

// tessdata_fast plain (non-gzipped) files via GitHub raw - no decompression needed.
const cdnBase = "https://github.com/tesseract-ocr/tessdata_fast/raw/main"

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

// LangName returns the display name for a code, or the code itself if unknown.
func LangName(code string) string {
	for _, l := range Available {
		if l.Code == code {
			return l.Name
		}
	}
	return code
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
