package outputpath

import (
	"errors"
	"os"
	"path/filepath"

	"doc-html-translate/internal/config"
)

// ErrNoPreviousResult says that no output the converter owns exists for an input.
var ErrNoPreviousResult = errors.New("no previous result found")

// PreviousResult returns the output directory of an earlier conversion of input, if one exists that the converter
// owns. A folder that merely contains an index.html - a saved website, a user's own folder - is never reported,
// opened or deleted as a result. The app's window and the `output status` command both ask this, so they cannot
// disagree about what a previous result is.
func PreviousResult(input, folder string) (string, error) {
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	target, err := Resolve(abs, folder)
	if err != nil {
		return "", err
	}
	if !target.State.Ours() {
		return "", ErrNoPreviousResult
	}
	if _, err := os.Stat(filepath.Join(target.Dir, "index.html")); err != nil {
		return "", ErrNoPreviousResult
	}
	return target.Dir, nil
}

// OptionsChanged reports whether the finished output in outputDir was built with other result-affecting options than
// the run described by cfg asks for. The options come from the very configuration the run would use, so the CLI's
// defaults apply exactly as they will in the run. An unfinished output is not "changed": the CLI rebuilds it on its
// own, without asking. A source that changed on disk is not an option change either.
func OptionsChanged(cfg config.Config, outputDir string) bool {
	abs, err := filepath.Abs(cfg.InputFile)
	if err != nil {
		return false
	}
	reason, _ := CheckReuse(outputDir, abs, OptionsFor(cfg))
	return reason == ReuseOptionsChanged
}

// Token names a reuse reason as one lower-case word, the form the `reason` of `output status` carries. ReuseOK has no
// token: a current result needs no reason.
func (r Reason) Token() string {
	switch r {
	case ReuseNoRecord:
		return "no-record"
	case ReuseSourceChanged:
		return "source-changed"
	case ReuseOptionsChanged:
		return "options-changed"
	case ReusePartial:
		return "partial"
	case ReuseUntranslated:
		return "untranslated"
	}
	return ""
}

// Status is what is known about the result of one input: the data `output status` answers with.
type Status struct {
	// Exists: an output the converter owns exists for this input (PreviousResult).
	Exists bool `json:"exists"`
	// Complete: its record carries the completion, so the run that made it finished.
	Complete bool `json:"complete"`
	// Stale: a run with the options asked for now would rebuild it, for any reason CheckReuse knows.
	Stale bool `json:"stale"`
	// Reason is the Token of that reason; empty when the result is current or does not exist.
	Reason      string `json:"reason,omitempty"`
	OutputDir   string `json:"outputDir,omitempty"`
	EntryFile   string `json:"entryFile,omitempty"`
	Translation string `json:"translation,omitempty"`
	ToolVersion string `json:"toolVersion,omitempty"`
}

// StatusFor answers whether a finished result exists for input (placed under folder when it is not empty), whether it
// is complete, and whether a run with the options want would reuse it - the same CheckReuse the app's window asks. It
// only reads. An input with no result of the converter's own is not an error: Exists is false.
func StatusFor(input, folder string, want Options) (Status, error) {
	dir, err := PreviousResult(input, folder)
	if errors.Is(err, ErrNoPreviousResult) {
		return Status{}, nil
	}
	if err != nil {
		return Status{}, err
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return Status{}, err
	}
	st := Status{Exists: true, OutputDir: dir, EntryFile: filepath.Join(dir, "index.html")}
	if m, merr := ReadMarker(dir); merr == nil && m.Complete != nil {
		st.Complete = true
		st.Translation = m.Complete.Translation
		st.ToolVersion = m.Complete.ToolVersion
	}
	if reason, _ := CheckReuse(dir, abs, want); reason != ReuseOK {
		st.Stale = true
		st.Reason = reason.Token()
	}
	return st, nil
}
