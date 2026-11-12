package report

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/syslocale"
)

// ollamaDefaultModel mirrors the default in internal/translator. The summary states which
// model a default run would reach for, not which one this run used.
const ollamaDefaultModel = "gemma3:12b"

// SchemaVersion is the version of the environment summary (DIAGNOSTIC-REPORT 8 B: the key
// `schema_version` in environment.txt; absent means 1). Bump it when a key changes meaning.
const SchemaVersion = 1

// Env is what the author needs to know about the installation before reading the logs.
type Env struct {
	AppVersion, Edition, OS, UILang, Tesseract, OllamaModel string
	OCRLangs                                                []string
}

// envField is one fact of the summary: the label the readable report prints and the key the
// machine-readable environment.txt prints.
type envField struct{ label, key, value string }

// environmentFields collects the installation facts once, for both renderings.
func environmentFields(appVersion string, packaged bool) []envField {
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

	// The labels are a shared invariant with the extension's reportText - see docs/PARITY.md.
	return []envField{
		{"version", "app_version", env.AppVersion},
		{"edition", "edition", env.Edition},
		{"platform", "os", env.OS},
		{"interface language", "ui_language", env.UILang},
		{"ocr", "ocr", env.Tesseract},
		{"ocr languages", "ocr_languages", langs},
		// DataDir would stage packs as a side effect; a report only reads.
		{"ocr data dir", "ocr_data_dir", strings.Join(ocr.DataDirs(), "; ")},
		{"ollama model", "ollama_model", env.OllamaModel},
	}
}

// Environment describes the installation as greppable `KEY=value` lines, one fact per line,
// opening with `schema_version` (DIAGNOSTIC-REPORT rule 2 and 8 B). This is the file a program
// reads; the labelled text a person reads is ReportText.
//
// Both are **always English**, whatever the interface language is: the author reads one
// format, and a summary they cannot read is worse than no summary.
func Environment(appVersion string, packaged bool) string {
	return renderEnvironment(environmentFields(appVersion, packaged))
}

// ReportText is the same facts as a labelled `label: value` block for the author to read, the
// human-readable report that travels beside the environment summary (DIAGNOSTIC-REPORT 8 B).
func ReportText(appVersion string, packaged bool) string {
	return renderReport(environmentFields(appVersion, packaged))
}

func renderEnvironment(fields []envField) string {
	var b strings.Builder
	b.WriteString("schema_version=" + strconv.Itoa(SchemaVersion) + "\n")
	for _, f := range fields {
		b.WriteString(f.key + "=" + f.value + "\n")
	}
	// The summary names paths - the tesseract binary, the tessdata folder - so it goes through
	// the same gate as everything else in the archive.
	return Redact(b.String())
}

func renderReport(fields []envField) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(f.label + ": " + f.value + "\n")
	}
	return Redact(b.String())
}
