package outputpath

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fsutil"
)

// Translation states recorded on completion. Only "full" makes a translated output reusable:
// a partial one reads as translated in the TOC while later pages are still in the source
// language, which is exactly the result a re-run must not reopen.
const (
	TranslationNone    = "none"
	TranslationFull    = "full"
	TranslationPartial = "partial"
)

// Engine names as the record stores them.
const (
	EngineNone   = "none"
	EngineGoogle = "google"
	EngineOllama = "ollama"
)

// Options are the settings that change what a conversion writes. Execution-only switches
// (-force, -noopen, -v, -ollama-parallel, -ollama-ctx, -max-cost, -ui-lang) are left out on
// purpose: rebuilding a book because the log was verbose, or because the interface language of
// the chrome changed, would throw away a paid translation for nothing.
type Options struct {
	Engine      string `json:"engine"`
	SourceLang  string `json:"sourceLang,omitempty"`
	TargetLang  string `json:"targetLang,omitempty"`
	OllamaModel string `json:"ollamaModel,omitempty"`
	OCR         bool   `json:"ocr"`
	OCRLang     string `json:"ocrLang,omitempty"`
	SinglePage  bool   `json:"singlePage"`
	SplitSize   int    `json:"splitSize"`
	TOCDepth    int    `json:"tocDepth"`
}

// OptionsFor returns the result-affecting options a run with cfg asks for. The CLI stores them
// and the GUI compares against them, so both derive them from the same parsed config.
func OptionsFor(cfg config.Config) Options {
	o := Options{
		Engine:      EngineNone,
		SourceLang:  cfg.SourceLang,
		TargetLang:  cfg.TargetLang,
		OllamaModel: cfg.OllamaModel,
		OCR:         cfg.OCR,
		OCRLang:     cfg.OCRLang,
		SinglePage:  cfg.SinglePage,
		SplitSize:   cfg.SplitSize,
		TOCDepth:    cfg.TOCDepth,
	}
	// The same precedence the pipeline's engine switch applies.
	switch {
	case cfg.NoTranslate:
	case cfg.UseGoogle:
		o.Engine = EngineGoogle
	case cfg.UseOllama:
		o.Engine = EngineOllama
	}
	return o
}

// normalized drops the fields that cannot influence the output under the rest of the options,
// so "-dst de" without an engine, or "-split 3000" in single-page mode, does not count as a
// change. ocrForced is set for inputs (an image, a comic) that are OCRed whatever -ocr says.
func (o Options) normalized(ocrForced bool) Options {
	o.OCR = o.OCR || ocrForced
	if o.Engine == "" {
		o.Engine = EngineNone
	}
	if o.Engine == EngineNone {
		o.TargetLang = ""
	}
	if o.Engine != EngineOllama {
		o.OllamaModel = ""
	}
	// With no -ocr-lang the OCR language follows -src, so that is the value compared.
	switch {
	case !o.OCR:
		o.OCRLang = ""
	case o.OCRLang == "":
		o.OCRLang = o.SourceLang
	}
	if o.Engine == EngineNone {
		o.SourceLang = ""
	}
	if o.SinglePage || o.SplitSize < 0 {
		o.SplitSize = 0
	}
	if o.SinglePage || o.TOCDepth < 0 {
		o.TOCDepth = 0
	}
	return o
}

// Completion is written as the very last step of a successful run. Its absence means the run
// never finished: it was interrupted, it crashed, or it predates this record.
type Completion struct {
	Completed   time.Time `json:"completed"`
	ToolVersion string    `json:"toolVersion"`
	Options     Options   `json:"options"`
	OCRForced   bool      `json:"ocrForced,omitempty"`
	Translation string    `json:"translation"`
}

// Reason says why an existing output is not reused. ReuseOK means it is.
type Reason int

const (
	ReuseOK             Reason = iota
	ReuseNoRecord              // interrupted, crashed, or made before the completion record existed
	ReuseSourceChanged         // the document on disk is not the one that was converted
	ReuseOptionsChanged        // built with different result-affecting options
	ReusePartial               // translation stopped part-way
	ReuseUntranslated          // translation was asked for but never ran (no key, over the cost limit)
)

// ReadMarker loads dir's ownership record.
func ReadMarker(dir string) (Marker, error) {
	data, err := os.ReadFile(filepath.Join(dir, MarkerName))
	if err != nil {
		return Marker{}, err
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		return Marker{}, err
	}
	return m, nil
}

// CheckReuse decides whether the output in dir may be opened as-is for a run of source with
// the options want, and names the reason when it may not. The tool version is recorded but not
// compared: an upgrade alone is no reason to throw away a paid translation.
func CheckReuse(dir, source string, want Options) (Reason, []string) {
	m, err := ReadMarker(dir)
	if err != nil || m.Complete == nil || m.Tool != markerTool || !samePath(m.Source, source) {
		return ReuseNoRecord, nil
	}
	fi, err := os.Stat(source)
	if err != nil || fi.Size() != m.SourceSize || !fi.ModTime().UTC().Equal(m.SourceModTime) {
		return ReuseSourceChanged, nil
	}
	c := m.Complete
	have, wantN := c.Options.normalized(c.OCRForced), want.normalized(c.OCRForced)
	if have != wantN {
		return ReuseOptionsChanged, diffOptions(have, wantN)
	}
	if have.Engine != EngineNone {
		switch c.Translation {
		case TranslationFull:
		case TranslationPartial:
			return ReusePartial, nil
		default:
			return ReuseUntranslated, nil
		}
	}
	return ReuseOK, nil
}

// diffOptions names what changed by the CLI flag a user would type, which reads the same in
// every interface language. a and b are normalized and differ.
func diffOptions(a, b Options) []string {
	// A value that matters on one side only is explained by the switch that made it matter
	// (-ocr, the engine), so it is listed only when nothing else explains the difference.
	for _, strict := range []bool{false, true} {
		differ := func(x, y string) bool { return x != y && (strict || (x != "" && y != "")) }
		var d []string
		add := func(changed bool, flag string) {
			if changed {
				d = append(d, flag)
			}
		}
		add(a.Engine != b.Engine, "-google/-ollama")
		add(differ(a.SourceLang, b.SourceLang), "-src")
		add(differ(a.TargetLang, b.TargetLang), "-dst")
		add(differ(a.OllamaModel, b.OllamaModel), "-ollama-model")
		add(a.OCR != b.OCR, "-ocr")
		add(differ(a.OCRLang, b.OCRLang), "-ocr-lang")
		add(a.SinglePage != b.SinglePage, "-multipage")
		add(a.SplitSize != b.SplitSize, "-split")
		add(a.TOCDepth != b.TOCDepth, "-toc-depth")
		if len(d) > 0 {
			return d
		}
	}
	return nil
}

// MarkComplete rewrites dir's ownership record with the completion, atomically, so a crash in
// the middle of it leaves the incomplete record rather than a broken one.
func MarkComplete(dir, source string, c Completion) error {
	m, err := ReadMarker(dir)
	if err != nil || m.Tool != markerTool {
		m = newMarker(source)
	}
	if c.Completed.IsZero() {
		c.Completed = time.Now().UTC()
	}
	m.Complete = &c
	return writeMarker(dir, m)
}

func writeMarker(dir string, m Marker) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(dir, MarkerName)
	if err := fsutil.WriteFile(p, data, 0o644); err != nil {
		return err
	}
	hideFile(p)
	return nil
}
