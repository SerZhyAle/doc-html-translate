package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/readiness"
)

// ReasonTargetMissing is the closed reason token of APP-COMMANDS rule 12 for a verb whose named thing does not exist; it
// goes with exit 11.
const ReasonTargetMissing = "target-missing"

// The inspection group (ticket E0146) is stateless, in-process and read-only: it needs neither the console's run log nor
// Ctrl+C, so both executables serve it and no hook of Env is involved.

// targetMissing is the answer for a document that does not exist: one sentence on standard error, exit 11, reason
// target-missing. Nothing else was done.
func (f *front) targetMissing() int {
	_, stderr := f.streams()
	fmt.Fprintf(stderr, "Error: %s\n", i18n.T(f.lang(), msgNoSuchDocument))
	return ExitNotFound
}

// respond writes the data to --result when it was given (whole or not at all, rule 9), then prints it: as the JSON object
// with --json, as the readable form otherwise. A result file that cannot be written leaves standard output empty.
func (f *front) respond(inv *Invocation, asJSON bool, data []byte, readable string) int {
	if code := f.answer(inv, data); code != ExitDone {
		return code
	}
	stdout, _ := f.streams()
	if asJSON {
		_, _ = stdout.Write(data)
	} else {
		_, _ = io.WriteString(stdout, readable)
	}
	return ExitDone
}

// doctor serves `doctor`: what this machine can do, as data. Without --ollama no service is asked.
func (f *front) doctor(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	withOllama, uerr := inv.Bool("ollama")
	if uerr != nil {
		return f.refuse(uerr)
	}
	model, uerr := inv.String("ollama-model")
	if uerr != nil {
		return f.refuse(uerr)
	}
	rep := readiness.Diagnose(context.Background(), f.env.Version, readiness.DoctorOptions{Ollama: withOllama, OllamaModel: model})
	return f.respond(inv, asJSON, encode(rep), ReadableDoctor(rep))
}

// ReadableDoctor is `doctor` without --json: the data as `key: value` lines. The keys are the data names, so they stay
// as they are in every interface language (ticket E0145 assumption Y8, kept by E0146).
func ReadableDoctor(rep readiness.Report) string {
	var b strings.Builder
	line := func(key string, v any) { fmt.Fprintf(&b, "%s: %v\n", key, v) }
	tool := func(name string, t readiness.ToolInfo) {
		switch {
		case !t.Found:
			line("tool "+name, "missing")
		case t.Version == "":
			line("tool "+name, "found")
		default:
			line("tool "+name, "found "+t.Version)
		}
	}
	line("version", rep.Version)
	line("packaged", rep.Packaged)
	tool(readiness.ToolTesseract, rep.Tools.Tesseract)
	tool(readiness.ToolPDFToText, rep.Tools.PDFToText)
	tool(readiness.ToolCalibre, rep.Tools.Calibre)
	tool(readiness.ToolSevenZip, rep.Tools.SevenZip)
	tool(readiness.ToolFileDO, rep.Tools.FileDO)
	if len(rep.OCRLangs) == 0 {
		line("ocrLangs", "none")
	} else {
		line("ocrLangs", strings.Join(rep.OCRLangs, " "))
	}
	line("googleKeyPresent", rep.GoogleKeyPresent)
	if o := rep.Ollama; o != nil {
		line("ollama", fmt.Sprintf("model %s, reachable %v, modelPresent %v", o.Model, o.Reachable, o.ModelPresent))
	}
	for _, f := range rep.Formats {
		state := "ready"
		if !f.Ready {
			state = "not ready, needs " + f.Tool
		} else if f.Tool != "" {
			state = "ready, needs " + f.Tool
		}
		if f.OptionalTool != "" {
			state += ", optional " + f.OptionalTool
		}
		line("format "+f.Ext, state)
	}
	return b.String()
}

// InputCheckData is the data of `input check`: the window's answer without its display-only members (the Google cost
// limit belongs to the cost dialog).
type InputCheckData struct {
	State  string            `json:"state"`
	Issues []readiness.Issue `json:"issues"`
}

// inputCheck serves `input check`: the window's pre-flight for one document, as data.
func (f *front) inputCheck(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	in, uerr := inv.String("in")
	if uerr != nil {
		return f.refuse(uerr)
	}
	abs, err := filepath.Abs(in)
	if err != nil {
		return f.failText(err.Error())
	}
	if _, err := os.Stat(abs); errors.Is(err, fs.ErrNotExist) {
		return f.targetMissing()
	}
	// The converter's own parser reads the options, so the defaults (--src en, the default Ollama model) and the alias
	// --free are the converter's; the verb has no second set.
	args, uerr := LegacyArgs(inv, abs, "in", "json", "result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	cfg, err := config.ParseArgs(args)
	if err != nil {
		// A value the grammar accepted and the parser refuses: usage, this verb has no exit 1 for it.
		_, stderr := f.streams()
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitUsage
	}
	res := readiness.Check(context.Background(), readiness.Request{
		Input:       abs,
		Google:      cfg.UseGoogle,
		Ollama:      cfg.UseOllama,
		OllamaModel: cfg.OllamaModel,
		SrcLang:     cfg.SourceLang,
		OCR:         cfg.OCR,
		OCRLang:     cfg.OCRLang,
	})
	data := InputCheckData{State: res.State, Issues: res.Issues}
	return f.respond(inv, asJSON, encode(data), ReadableInputCheck(data))
}

// ReadableInputCheck is `input check` without --json: the state, then one line per issue.
func ReadableInputCheck(d InputCheckData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "state: %s\n", d.State)
	for _, i := range d.Issues {
		fmt.Fprintf(&b, "issue: %s %s", i.Code, i.Level)
		if i.Detail != "" {
			fmt.Fprintf(&b, " %s", i.Detail)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ocrLangs serves `ocr langs`: the catalogue with the installed languages marked. Without --json it prints the very
// listing the frozen alias -ocr-langs prints.
func (f *front) ocrLangs(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	data := encode(struct {
		Languages []ocr.LangRow `json:"languages"`
	}{ocr.LangRows()})
	return f.respond(inv, asJSON, data, ocr.LangsText())
}
