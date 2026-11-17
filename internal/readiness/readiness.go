// Package readiness answers, as data, what this machine can do for a conversion: whether one document is ready to
// convert with the chosen engine and OCR and what is missing (Check), and what the machine can do in general (Diagnose).
//
// It is the one check the app's window and the command line share (ticket E0146): the window's `/api/readiness` handler and
// the verb `input check` both call Check, and `doctor` is Diagnose, so a script and the window cannot disagree. Every
// answer comes from the very discovery functions a conversion uses (Probes), and nothing here writes a file, opens a log
// or sends anything anywhere; only an explicit Ollama question talks to a service, and only to the local one.
package readiness

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"doc-html-translate/internal/bundledtools"
	"doc-html-translate/internal/comic"
	"doc-html-translate/internal/fdsec"
	"doc-html-translate/internal/img"
	"doc-html-translate/internal/mobi"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/pdf"
	"doc-html-translate/internal/pipeline"
	"doc-html-translate/internal/translator"
)

// DefaultOllamaModel is the model a run reaches for when none is named (internal/config's default).
const DefaultOllamaModel = "gemma3:12b"

// Request is the part of a conversion request the check reads. Its zero value asks about nothing.
type Request struct {
	Input       string
	NoTranslate bool
	Google      bool
	Ollama      bool
	OllamaModel string
	MaxCost     string
	SrcLang     string
	OCR         bool
	OCRLang     string
}

// Issue is one thing the check found. Level is "action" (the run would fail or degrade) or "optional" (it only improves
// the result).
type Issue struct {
	Code   string `json:"code"`
	Level  string `json:"level"`
	Detail string `json:"detail,omitempty"`
}

// Result is the answer of Check. State is "ready", "optional" or "action": the worst level among the issues.
type Result struct {
	State       string  `json:"state"`
	Issues      []Issue `json:"issues"`
	GoogleLimit string  `json:"googleLimit,omitempty"`
}

// Probes call the same discovery functions as conversion. The indirection keeps the combinations testable without
// installing helpers or contacting a local service; a test swaps a field and puts the whole value back.
var Probes = struct {
	Calibre    func() bool
	SevenZip   func() bool
	FileDO     func() bool
	OCRLocate  func() (string, error)
	OCRMissing func(context.Context, string, string) []string
	GoogleKey  func() (string, error)
	Ollama     func(context.Context, string) (bool, error)

	// The members below are read by Diagnose only.

	// PDFToText reports a pdftotext on this machine or carried by this build, without unpacking anything.
	PDFToText func() bool
	// OCRLangs lists the installed recognition languages.
	OCRLangs func() []string
	// Version is the version number of a found tool ("tesseract", "pdftotext", "calibre", "sevenzip"), "" when unknown.
	Version func(ctx context.Context, tool string) string
	// Packaged reports whether the program runs from inside an MSIX package.
	Packaged func() bool
}{
	Calibre:    mobi.Available,
	SevenZip:   comic.SevenZipAvailable,
	FileDO:     fdsec.Available,
	OCRLocate:  ocr.Locate,
	OCRMissing: ocr.MissingLangs,
	GoogleKey:  translator.LoadGoogleAPIKey,
	Ollama:     translator.OllamaModelAvailable,
	PDFToText:  func() bool { return bundledtools.PDFToTextBundled() || pdf.SystemPDFToTextPath() != "" },
	OCRLangs:   ocr.Installed,
	Version:    toolVersion,
	Packaged:   Packaged,
}

// Check says whether one document is ready to convert with the chosen engine and OCR. An empty input is "ready" with no
// issues: there is nothing to check yet.
func Check(ctx context.Context, req Request) Result {
	out := Result{State: "ready", Issues: []Issue{}}
	add := func(code, level, detail string) {
		out.Issues = append(out.Issues, Issue{Code: code, Level: level, Detail: detail})
		if level == "action" {
			out.State = "action"
		} else if out.State == "ready" {
			out.State = "optional"
		}
	}
	if strings.TrimSpace(req.Input) == "" {
		return out
	}
	info, err := os.Stat(req.Input)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() || info.Size() == 0 {
		add("input", "action", "")
		return out
	}
	ext := strings.ToLower(filepath.Ext(req.Input))
	// A FileDO secret file needs FileDO, and its size must pass the format's credential-free
	// screen. Both are decided here, before Convert and before any password question; the
	// converter checks again, so a queue screens every container before it asks anything.
	if fdsec.IsContainer(ext) {
		switch {
		case !Probes.FileDO():
			add("filedo", "action", "")
		case !fdsec.ScreenLength(info.Size()):
			add("fdsecScreen", "action", "")
		}
		if out.State == "action" {
			return out
		}
	}
	if (ext == ".mobi" || ext == ".azw3") && !Probes.Calibre() {
		add("calibre", "action", "")
	}
	if (ext == ".cbr" || ext == ".cb7") && !Probes.SevenZip() {
		add("sevenzip", "action", "")
	}
	forcedOCR := img.IsImage(ext) || comic.IsComic(ext)
	if (req.OCR && pipeline.SupportsImageOCR(ext)) || forcedOCR {
		bin, err := Probes.OCRLocate()
		if err != nil {
			add("tesseract", "optional", "")
		} else {
			lang := req.OCRLang
			if lang == "" {
				lang = ocr.TessLang(req.SrcLang)
			}
			if missing := Probes.OCRMissing(ctx, bin, lang); len(missing) > 0 {
				add("ocrData", "optional", strings.Join(missing, ", "))
			}
		}
	}
	if !req.NoTranslate && req.Google {
		if _, err := Probes.GoogleKey(); err != nil {
			add("googleKey", "optional", "")
		}
		out.GoogleLimit = "0"
		if f, err := strconv.ParseFloat(strings.TrimSpace(req.MaxCost), 64); err == nil && f > 0 && !math.IsInf(f, 0) {
			out.GoogleLimit = strconv.FormatFloat(f, 'f', -1, 64)
		}
	}
	if !req.NoTranslate && !req.Google && req.Ollama {
		model := strings.TrimSpace(req.OllamaModel)
		if model == "" {
			model = DefaultOllamaModel
		}
		available, err := Probes.Ollama(ctx, model)
		if err != nil {
			add("ollamaService", "optional", "")
		} else if !available {
			add("ollamaModel", "optional", model)
		}
	}
	return out
}

// Packaged reports whether the program runs from inside an MSIX/Store package. Such installs live under
// ...\WindowsApps\... and virtualize HKCU writes, so the -register flow is a no-op there (associations come from the
// package manifest).
func Packaged() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return packagedPath(exe)
}

func packagedPath(exe string) bool {
	return strings.Contains(strings.ToLower(exe), `\windowsapps\`)
}
