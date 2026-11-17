package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/outputpath"
)

// Exit codes of the converter that the convert group reads (OCR-INVOCATION section D); the table itself is the
// pipeline's and is copied into the manifest. 2 is the one this file returns on its own.
const (
	ExitIO       = 2
	ExitPartial  = 4
	ExitNotFound = 11 // APP-COMMANDS rule 11: the thing the verb names does not exist
)

// convertGroup is the set of verbs served by the process that does conversions (doc-html-translate.exe). A process with
// no conversion hook refuses them with one sentence (ticket E0145, assumption Y7).
var convertGroup = map[string]bool{"convert": true, "output status": true}

// secretExt is the extension of a FileDO secret file (internal/fdsec.Ext; a test holds the two equal). The package does
// not import fdsec, which would drag the dialog and process layers into the command front.
const secretExt = ".fd-sec"

// The values of `outcome` in the data of `convert --result`.
const (
	OutcomeConverted  = "converted"  // converted, no translation asked for
	OutcomeTranslated = "translated" // an engine was asked for and the record says full
	OutcomePartial    = "partial"    // the translation stopped part-way
	OutcomeSkipped    = "skipped"    // an engine was asked for and never ran: no key, estimate above the cap, declined
	OutcomeReused     = "reused"     // a finished, current result was opened as it was
)

// ConvertResult is the data `convert --result` writes (APP-COMMANDS rule 9): one JSON object, UTF-8.
type ConvertResult struct {
	Outcome     string                 `json:"outcome"`
	OutputDir   string                 `json:"outputDir"`
	EntryFile   string                 `json:"entryFile"`
	Translation string                 `json:"translation"`
	Complete    *outputpath.Completion `json:"complete"`
	ExitCode    int                    `json:"exitCode"`
}

// LegacyArgs spells the options of an invocation as the conversion parser (internal/config) reads them: `--name` for a
// bool that is true, `--name=value` for the rest, then `--` and the operand. The verb and the bare call therefore run the
// same code with the same defaults. skip names the options of the verb that are not the parser's (`result`, `json`, `in`).
func LegacyArgs(inv *Invocation, operand string, skip ...string) ([]string, *UsageError) {
	var out []string
	for _, a := range inv.Command.Args {
		if a.Positional || len(inv.Options[a.Name]) == 0 || contains(skip, a.Name) {
			continue
		}
		if a.Kind == "bool" {
			v, err := inv.Bool(a.Name)
			if err != nil {
				return nil, err
			}
			if v {
				out = append(out, "--"+a.Name)
			}
			continue
		}
		v, err := inv.String(a.Name)
		if err != nil {
			return nil, err
		}
		out = append(out, "--"+a.Name+"="+v)
	}
	return append(out, "--", operand), nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// convertPolicy is the one place the verb differs from the bare call (APP-COMMANDS rule 4: a command never waits on a
// person it did not ask for). The bare call opens a cost dialog and asks for a password; a program that runs the verb
// cannot answer either, so what the verb cannot decide it refuses, naming the missing option.
func convertPolicy(inv *Invocation) *UsageError {
	path, err := inv.String("path")
	if err != nil {
		return err
	}
	google, err := inv.Bool("google")
	if err != nil {
		return err
	}
	cost, err := inv.String("max-cost")
	if err != nil {
		return err
	}
	amount := 0.0
	if cost != "" {
		v, perr := strconv.ParseFloat(cost, 64)
		if perr != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return usage(ReasonBadArgument, msgUnexpected, cost, "--max-cost")
		}
		amount = v
	}
	if google && amount <= 0 {
		return usage(ReasonBadArgument, msgGoogleNeedsCap, "--google", "--max-cost")
	}
	if strings.EqualFold(filepath.Ext(path), secretExt) {
		env, err := inv.String("fdsec-password-env")
		if err != nil {
			return err
		}
		if env == "" {
			return usage(ReasonBadArgument, msgSecretNeedsEnv, "--fdsec-password-env")
		}
	}
	return nil
}

// convert serves `convert`: the policy above, then the same conversion the bare call runs, then - when --result names a
// file - the data of the run, written whole.
func (f *front) convert(inv *Invocation) int {
	// The person typed the language they want to read: a refusal of this run speaks it.
	if lang, _ := inv.String("ui-lang"); lang != "" {
		f.env.Lang = lang
	}
	if e := convertPolicy(inv); e != nil {
		return f.refuse(e)
	}
	result, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	path, _ := inv.String("path")
	args, uerr := LegacyArgs(inv, path, "result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	cfg, err := config.ParseArgs(args)
	if err != nil {
		// The parser's own sentence for a value the grammar cannot judge (a negative --split): bad arguments, exit 1.
		return f.failText(err.Error())
	}
	// A paid run is never spent on a result file that cannot land: the folder is tried before the run starts.
	if result != "" && !resultWritable(result) {
		return f.fail(msgResultFailed)
	}
	before := completedAt(cfg.InputFile, cfg.OutputFolder)
	code := f.env.Convert(cfg)
	if result == "" || (code != ExitDone && code != ExitPartial) {
		return code // a run that failed writes no file (rule 9)
	}
	res, ok := BuildConvertResult(cfg, before, code)
	if ok {
		ok = fsutil.WriteFile(result, encode(res), 0o644) == nil
	}
	if !ok {
		_, stderr := f.streams()
		fmt.Fprintf(stderr, "Error: %s\n", i18n.T(f.lang(), msgResultFailed))
		if code == ExitDone {
			return ExitIO // the book is there, the data the caller asked for is not: not proven, so not done
		}
	}
	return code
}

// BuildConvertResult reads what a finished run left and names it. before is the completion time of the output that was
// there when the run began (zero when none): a build always rewrites the record with the time of the write, so a record
// that still carries that time is a result the run reused. ok is false when the run left no finished record.
func BuildConvertResult(cfg config.Config, before time.Time, exit int) (ConvertResult, bool) {
	abs, err := filepath.Abs(cfg.InputFile)
	if err != nil {
		return ConvertResult{}, false
	}
	dir, err := outputpath.PreviousResult(abs, cfg.OutputFolder)
	if err != nil {
		return ConvertResult{}, false
	}
	m, err := outputpath.ReadMarker(dir)
	if err != nil || m.Complete == nil {
		return ConvertResult{}, false
	}
	c := m.Complete
	res := ConvertResult{
		OutputDir:   dir,
		EntryFile:   filepath.Join(dir, "index.html"),
		Translation: c.Translation,
		Complete:    c,
		ExitCode:    exit,
	}
	switch {
	case !before.IsZero() && c.Completed.Equal(before):
		res.Outcome = OutcomeReused
	case outputpath.OptionsFor(cfg).Engine == outputpath.EngineNone:
		res.Outcome = OutcomeConverted
	case c.Translation == outputpath.TranslationFull:
		res.Outcome = OutcomeTranslated
	case c.Translation == outputpath.TranslationPartial:
		res.Outcome = OutcomePartial
	default:
		res.Outcome = OutcomeSkipped
	}
	return res, true
}

// completedAt is the completion time of the finished output that exists for input, or the zero time.
func completedAt(input, folder string) time.Time {
	abs, err := filepath.Abs(input)
	if err != nil {
		return time.Time{}
	}
	dir, err := outputpath.PreviousResult(abs, folder)
	if err != nil {
		return time.Time{}
	}
	m, err := outputpath.ReadMarker(dir)
	if err != nil || m.Complete == nil {
		return time.Time{}
	}
	return m.Complete.Completed
}

// resultWritable reports whether a file could be written at path: the folder exists and takes a file, and the path is
// not a folder itself. It creates and removes one temporary file and leaves nothing else.
func resultWritable(path string) bool {
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		return false
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".result-probe.*.tmp")
	if err != nil {
		return false
	}
	name := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(name)
	return true
}

// outputStatus serves `output status`: whether a finished result exists for a document, whether it is complete and
// whether a run with the options asked for now would reuse it. It reads and prints; it writes only what --result names.
func (f *front) outputStatus(inv *Invocation) int {
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
	args, uerr := LegacyArgs(inv, abs, "in", "json", "result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	cfg, err := config.ParseArgs(args)
	if err != nil {
		// A value the grammar accepted and the parser refuses (a negative --split): usage, this verb has no exit 1 for it.
		_, stderr := f.streams()
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitUsage
	}
	st, err := outputpath.StatusFor(abs, cfg.OutputFolder, outputpath.OptionsFor(cfg))
	if err != nil {
		return f.failText(err.Error())
	}
	data := encode(st)
	if code := f.answer(inv, data); code != ExitDone {
		return code
	}
	stdout, _ := f.streams()
	if asJSON {
		_, _ = stdout.Write(data)
		return ExitDone
	}
	_, _ = io.WriteString(stdout, ReadableStatus(st))
	return ExitDone
}

// ReadableStatus is `output status` without --json: the data as `key: value` lines. The keys are the data names, so
// they stay as they are in every interface language (ticket E0145, assumption Y8).
func ReadableStatus(st outputpath.Status) string {
	var b strings.Builder
	line := func(key string, v any) { fmt.Fprintf(&b, "%s: %v\n", key, v) }
	line("exists", st.Exists)
	line("complete", st.Complete)
	line("stale", st.Stale)
	for _, kv := range [][2]string{
		{"reason", st.Reason}, {"outputDir", st.OutputDir}, {"entryFile", st.EntryFile},
		{"translation", st.Translation}, {"toolVersion", st.ToolVersion},
	} {
		if kv[1] != "" {
			line(kv[0], kv[1])
		}
	}
	return b.String()
}

// failText prints one sentence on standard error and returns exit 1 (the command ran and the operation failed).
func (f *front) failText(msg string) int {
	_, stderr := f.streams()
	fmt.Fprintf(stderr, "Error: %s\n", msg)
	return ExitFailed
}

// encode is the one JSON object of an answer: compact, UTF-8, no HTML escaping, one line.
func encode(v any) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return b.Bytes()
}
