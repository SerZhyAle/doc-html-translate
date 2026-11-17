package readiness

import (
	"context"
	"strings"
	"sync"
)

// The names of the helper tools in a Report; Format.Tool and Format.OptionalTool use the same words.
const (
	ToolTesseract = "tesseract"
	ToolPDFToText = "pdftotext"
	ToolCalibre   = "calibre"
	ToolSevenZip  = "sevenzip"
	ToolFileDO    = "filedo"
)

// ToolInfo is one helper tool: whether it is found and, when it reports one, its version number. Never a path.
type ToolInfo struct {
	Found   bool   `json:"found"`
	Version string `json:"version"`
}

// Tools is the helper tools the converter can use.
type Tools struct {
	Tesseract ToolInfo `json:"tesseract"`
	PDFToText ToolInfo `json:"pdftotext"`
	Calibre   ToolInfo `json:"calibre"`
	SevenZip  ToolInfo `json:"sevenzip"`
	FileDO    ToolInfo `json:"filedo"`
}

// FormatInfo is one input format. Tool is the helper the format cannot be read without ("" when it needs none) and Ready
// says whether that helper is found; OptionalTool is the helper that only improves the result (the text of a scanned
// page, a PDF's text).
type FormatInfo struct {
	Ext          string `json:"ext"`
	Tool         string `json:"tool"`
	Ready        bool   `json:"ready"`
	OptionalTool string `json:"optionalTool,omitempty"`
}

// OllamaInfo is what the local Ollama server answered for the model asked about.
type OllamaInfo struct {
	Model        string `json:"model"`
	Reachable    bool   `json:"reachable"`
	ModelPresent bool   `json:"modelPresent"`
}

// Report is what `doctor` answers: this machine's capabilities as data. It holds booleans, version numbers, extensions
// and language codes - never a path, a document name or the Google key.
type Report struct {
	Version          string       `json:"version"`
	Packaged         bool         `json:"packaged"`
	Tools            Tools        `json:"tools"`
	Formats          []FormatInfo `json:"formats"`
	OCRLangs         []string     `json:"ocrLangs"`
	GoogleKeyPresent bool         `json:"googleKeyPresent"`
	Ollama           *OllamaInfo  `json:"ollama,omitempty"`
}

// DoctorOptions are the options of `doctor`. Ollama asks the local server; without it nothing is requested from any
// service.
type DoctorOptions struct {
	Ollama      bool
	OllamaModel string
}

// formatTable is every input the converter accepts, with the helper each needs. The converter's own predicates are not
// lists (the readers' extension sets are private), so a test holds this table to pipeline.Convertible both ways.
var formatTable = func() []formatRow {
	rows := []formatRow{
		{".epub", "", ""},
		{".pdf", "", ToolPDFToText},
		{".txt", "", ""}, {".md", "", ""}, {".fb2", "", ""}, {".rtf", "", ""}, {".html", "", ""}, {".htm", "", ""},
		{".mobi", ToolCalibre, ""}, {".azw3", ToolCalibre, ""},
		{".cbz", "", ToolTesseract}, {".cbt", "", ToolTesseract},
		{".cbr", ToolSevenZip, ToolTesseract}, {".cb7", ToolSevenZip, ToolTesseract},
	}
	// A standalone image is always read through OCR: it reads without Tesseract and then carries no text.
	for _, e := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tif", ".tiff", ".jxl"} {
		rows = append(rows, formatRow{e, "", ToolTesseract})
	}
	return append(rows, formatRow{".fd-sec", ToolFileDO, ""})
}()

type formatRow struct{ ext, tool, optional string }

// Diagnose answers what this machine can do. version is the build's version. It reads only: a tool is looked up, never
// started except to read its own version number (at most 3 seconds each, all at once); the bundled pdftotext is not
// unpacked; nothing is written. Only opt.Ollama makes a request, and only to the local server.
func Diagnose(ctx context.Context, version string, opt DoctorOptions) Report {
	rep := Report{Version: version, Packaged: Probes.Packaged()}

	_, ocrErr := Probes.OCRLocate()
	rep.Tools = Tools{
		Tesseract: ToolInfo{Found: ocrErr == nil},
		PDFToText: ToolInfo{Found: Probes.PDFToText()},
		Calibre:   ToolInfo{Found: Probes.Calibre()},
		SevenZip:  ToolInfo{Found: Probes.SevenZip()},
		FileDO:    ToolInfo{Found: Probes.FileDO()},
	}
	// FileDO has no version query, and the bundled pdftotext would have to be unpacked to be asked: both stay "". The
	// others answer through Probes.Version, in parallel, each bounded by its own deadline.
	var wg sync.WaitGroup
	for _, t := range []struct {
		name string
		info *ToolInfo
	}{
		{ToolTesseract, &rep.Tools.Tesseract}, {ToolPDFToText, &rep.Tools.PDFToText},
		{ToolCalibre, &rep.Tools.Calibre}, {ToolSevenZip, &rep.Tools.SevenZip},
	} {
		if !t.info.Found {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.info.Version = Probes.Version(ctx, t.name)
		}()
	}
	wg.Wait()

	for _, row := range formatTable {
		rep.Formats = append(rep.Formats, FormatInfo{
			Ext: row.ext, Tool: row.tool, OptionalTool: row.optional,
			Ready: row.tool == "" || rep.Tools.found(row.tool),
		})
	}

	rep.OCRLangs = append([]string{}, Probes.OCRLangs()...)
	_, keyErr := Probes.GoogleKey()
	rep.GoogleKeyPresent = keyErr == nil

	if opt.Ollama {
		model := strings.TrimSpace(opt.OllamaModel)
		if model == "" {
			model = DefaultOllamaModel
		}
		present, err := Probes.Ollama(ctx, model)
		rep.Ollama = &OllamaInfo{Model: model, Reachable: err == nil, ModelPresent: err == nil && present}
	}
	return rep
}

// found reports whether the named tool is found; an unknown name is not.
func (t Tools) found(name string) bool {
	switch name {
	case ToolTesseract:
		return t.Tesseract.Found
	case ToolPDFToText:
		return t.PDFToText.Found
	case ToolCalibre:
		return t.Calibre.Found
	case ToolSevenZip:
		return t.SevenZip.Found
	case ToolFileDO:
		return t.FileDO.Found
	}
	return false
}
