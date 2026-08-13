package report

import (
	"os"
	"runtime"
	"strings"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/syslocale"
)

// ollamaDefaultModel mirrors the default in internal/translator. The summary states which
// model a default run would reach for, not which one this run used.
const ollamaDefaultModel = "gemma3:12b"

// Env is what the author needs to know about the installation before reading the logs.
type Env struct {
	AppVersion, Edition, OS, UILang, Tesseract, OllamaModel string
	OCRLangs                                                []string
}

// Environment describes the installation as a plain `key: value` block, one field per line.
//
// It is **always English**, whatever the interface language is: the author reads one format,
// and a summary they cannot read is worse than no summary. The field labels are a shared
// invariant with the extension's reportText - see docs/PARITY.md.
func Environment(appVersion string, packaged bool) string {
	env := Env{
		AppVersion:  appVersion,
		Edition:     "portable",
		OS:          runtime.GOOS + "/" + runtime.GOARCH,
		UILang:      i18n.Resolve("", "", syslocale.Lang()),
		Tesseract:   "not found",
		OllamaModel: ollamaDefaultModel,
		OCRLangs:    ocr.Installed(),
	}
	if packaged {
		env.Edition = "packaged (MSIX)"
	}
	if name := os.Getenv("OS"); name != "" {
		env.OS += " (" + name + ")"
	}
	if bin, err := ocr.Locate(); err == nil {
		env.Tesseract = bin
	}

	langs := "none"
	if len(env.OCRLangs) > 0 {
		langs = strings.Join(env.OCRLangs, ", ")
	}

	var b strings.Builder
	for _, f := range [][2]string{
		{"version", env.AppVersion},
		{"edition", env.Edition},
		{"platform", env.OS},
		{"interface language", env.UILang},
		{"ocr", env.Tesseract},
		{"ocr languages", langs},
		{"ocr data dir", ocr.DataDir()},
		{"ollama model", env.OllamaModel},
	} {
		b.WriteString(f[0] + ": " + f[1] + "\n")
	}
	// The summary names paths - the tesseract binary, the tessdata folder - so it goes through
	// the same gate as everything else in the archive.
	return Redact(b.String())
}
