package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Verdicts, as the ticket names them. COULD NOT VERIFY is never a pass: it is the answer when the
// input, the environment or a human gate is missing.
const (
	Pass            = "PASS"
	PassAdvisories  = "PASS WITH ADVISORIES"
	Fail            = "FAIL"
	CouldNotVerify  = "COULD NOT VERIFY"
	grossTextLoss   = 0.5  // below this share of the source's text layer a document has lost its text
	partialTextLoss = 0.85 // below this share the loss is reported as an advisory
)

// Result is one edition's record of one case, as tools/doccorpus/run.mjs writes it.
type Result struct {
	Schema                int         `json:"schema"`
	CaseID                string      `json:"caseId"`
	Edition               string      `json:"edition"`
	OCRMode               string      `json:"ocrMode"`
	Source                string      `json:"source"`
	Error                 string      `json:"error"`
	EnvironmentGap        string      `json:"environmentGap"`
	IntentionalDifference string      `json:"intentionalDifference"`
	Convert               *Convert    `json:"convert"`
	RenderMs              int         `json:"renderMs"`
	Truncated             bool        `json:"truncated"`
	Collection            *Collection `json:"collection"`
	Probe                 *Probe      `json:"probe"`
	ReusedFrom            *ReusedFrom `json:"reusedFrom,omitempty"`

	dir string // where the result's evidence lives
}

// Collection is how the collector's pass over the page ended (tools/doccorpus/collect.mjs). Any
// outcome but "settled" means the reader was still producing the document when the pass stopped,
// so what the probe holds is incomplete evidence about the collector, not a finding about the
// product. The count fields are null when the page never showed that counter.
type Collection struct {
	Outcome           string `json:"outcome"`
	Stage             string `json:"stage"`
	Rendered          int    `json:"rendered"`
	Total             *int   `json:"total"`
	OCRDone           *int   `json:"ocrDone"`
	OCRTotal          *int   `json:"ocrTotal"`
	ExtractionPending int    `json:"extractionPending"`
	Polls             int    `json:"polls"`
}

// Unsettled reports whether the pass ended before the reader finished.
func (c *Collection) Unsettled() bool { return c != nil && c.Outcome != "" && c.Outcome != "settled" }

// Progress is the "a/b" the stage is counting: recognised of queued images while OCR runs,
// extracted of rendered pages while page images are pulled, otherwise rendered of total pages.
func (c *Collection) Progress() string {
	of := func(a int, b *int) string {
		if b == nil {
			return fmt.Sprintf("%d/?", a)
		}
		return fmt.Sprintf("%d/%d", a, *b)
	}
	switch c.Stage {
	case "ocr-active":
		if c.OCRDone != nil {
			return of(*c.OCRDone, c.OCRTotal)
		}
	case "extracting":
		total := c.Rendered
		return of(max(total-c.ExtractionPending, 0), &total)
	}
	return of(c.Rendered, c.Total)
}

// Convert is the desktop conversion step.
type Convert struct {
	ExitCode *int     `json:"exitCode"`
	Ms       int      `json:"ms"`
	Argv     []string `json:"argv"`
	TimedOut bool     `json:"timedOut"`
	Index    string   `json:"index"`
}

// Probe is what the page showed the reader - the same probe for both editions.
type Probe struct {
	Title                string   `json:"title"`
	Lang                 string   `json:"lang"`
	TextChars            int      `json:"textChars"`
	Images               int      `json:"images"`
	ImagesLoaded         int      `json:"imagesLoaded"`
	ImagesBroken         int      `json:"imagesBroken"`
	Canvases             int      `json:"canvases"`
	Headings             int      `json:"headings"`
	TocEntries           int      `json:"tocEntries"`
	InternalLinks        int      `json:"internalLinks"`
	InternalLinksMissing int      `json:"internalLinksMissing"`
	MissingSample        []string `json:"missingSample"`
	Overlays             int      `json:"overlays"`
	Plates               int      `json:"plates"`
	PlateChars           int      `json:"plateChars"`
	PageUnits            int      `json:"pageUnits"`
	Status               string   `json:"status"`
	Notice               string   `json:"notice"`
	VerticalWriting      bool     `json:"verticalWriting"`
	RubyElements         int      `json:"rubyElements"`
}

// Judgement is the verdict of one case in one edition and mode.
type Judgement struct {
	CaseID     string             `json:"caseId"`
	Edition    string             `json:"edition"`
	OCRMode    string             `json:"ocrMode"`
	Auto       string             `json:"auto"`
	Campaign   string             `json:"campaign"`
	Fails      []string           `json:"fails,omitempty"`
	Advisories []string           `json:"advisories,omitempty"`
	Gaps       []string           `json:"gaps,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
	Evidence   string             `json:"evidence"`
}

func (j *Judgement) fail(format string, a ...any) {
	j.Fails = append(j.Fails, fmt.Sprintf(format, a...))
}
func (j *Judgement) advise(format string, a ...any) {
	j.Advisories = append(j.Advisories, fmt.Sprintf(format, a...))
}
func (j *Judgement) gap(format string, a ...any) { j.Gaps = append(j.Gaps, fmt.Sprintf(format, a...)) }

// LoadResult reads one result.json.
func LoadResult(p string) (*Result, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	r.dir = filepath.Dir(p)
	return &r, nil
}

func (r *Result) readFile(name string) string {
	if r.dir == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		return ""
	}
	return string(raw)
}

// Judge grades one result against the case and its expectation. The automated verdict judges only
// what the bytes and the page show; the campaign verdict additionally requires the human gates -
// a rights review and a reviewed expectation - and stays COULD NOT VERIFY until both are closed.
// A failure is a failure either way: a defect seen on a candidate document is still a defect.
func Judge(c Case, e *Expectation, r *Result) (j Judgement) {
	j = Judgement{CaseID: c.ID, Edition: r.Edition, OCRMode: r.OCRMode, Metrics: map[string]float64{}, Evidence: r.dir}
	defer func() { finish(&j, c, e) }()

	truth := e != nil && e.IsTruth()
	soft := func(format string, a ...any) { // a finding that only a reviewed expectation can make a failure
		if truth {
			j.fail(format, a...)
		} else {
			j.advise(format+" (draft expectation)", a...)
		}
	}

	switch {
	case r.Source == "missing":
		j.gap("source file missing on this machine")
		return j
	case r.Source == "hash-changed":
		j.gap("source bytes differ from the recorded SHA-256")
		return j
	case r.Error != "":
		j.gap("runner error: %s", r.Error)
		return j
	}
	if r.EnvironmentGap != "" {
		j.gap("%s", r.EnvironmentGap)
	}
	if r.Convert != nil && (r.Convert.ExitCode == nil || *r.Convert.ExitCode != 0 || r.Convert.Index == "") {
		code := "none"
		if r.Convert.ExitCode != nil {
			code = fmt.Sprint(*r.Convert.ExitCode)
		}
		j.fail("desktop conversion failed (exit %s): %s", code, lastErrorLine(r.readFile("convert.log")))
		return j
	}
	p := r.Probe
	if p == nil {
		j.gap("no probe recorded")
		return j
	}
	j.Metrics["textChars"] = float64(p.TextChars)
	j.Metrics["plates"] = float64(p.Plates)
	j.Metrics["images"] = float64(p.ImagesLoaded)

	if r.IntentionalDifference != "" {
		if strings.Contains(strings.ToLower(p.Notice), "desktop app") {
			j.advise("intentional difference honoured: %s", r.IntentionalDifference)
		} else {
			j.fail("expected the intentional %q notice, the viewer showed %q", r.IntentionalDifference, p.Notice)
		}
		return j
	}

	noTextNotice := strings.HasPrefix(p.Notice, "Little or no text")
	switch {
	case p.Notice == "":
	case noTextNotice && r.OCRMode == "default" && (c.Class == "scanned-pdf" || e != nil && e.TextLayer != "full"):
		j.advise("the default path shows the %q notice (OCR is off by default)", p.Notice)
	default:
		j.fail("the viewer showed a notice instead of the document: %q", p.Notice)
	}

	if p.ImagesBroken > 0 {
		j.fail("%d of %d images failed to load", p.ImagesBroken, p.Images)
	}
	if p.InternalLinksMissing > 0 {
		j.fail("%d of %d in-page links point at no element (e.g. %s)", p.InternalLinksMissing, p.InternalLinks, strings.Join(p.MissingSample, " "))
	}
	incomplete := r.incomplete()
	switch {
	case r.Collection.Unsettled():
		j.gap("incomplete evidence: stage %s, %s", r.Collection.Stage, r.Collection.Progress())
	case incomplete:
		j.gap("the page did not settle within the probe budget - coverage is partial")
	}

	// Page survival: every page of a page document must reach the reader, as an image, a canvas or
	// a page wrapper.
	if e != nil && e.Pages > 0 && (c.Class == "comic-archive" || c.Class == "comic-image" || c.Class == "scanned-pdf") && !noTextNotice {
		got := max(p.PageUnits, p.ImagesLoaded+p.Canvases)
		j.Metrics["pages"] = float64(got)
		// An unsettled pass has already said so above, with its own counts: a page count taken
		// while pages were still arriving is not a page loss.
		if got < e.Pages && !incomplete {
			j.fail("%d of %d pages reached the reader", got, e.Pages)
		}
	}

	// Text survival against the source's own text layer.
	if e != nil && e.TextLayer == "full" && e.SourceTextChars > 0 {
		ratio := float64(p.TextChars) / float64(e.SourceTextChars)
		j.Metrics["textRatio"] = round2(ratio)
		switch {
		case ratio < grossTextLoss:
			j.fail("only %.0f%% of the source text layer reached the page (%d of %d chars)", ratio*100, p.TextChars, e.SourceTextChars)
		case ratio < partialTextLoss:
			j.advise("%.0f%% of the source text layer reached the page", ratio*100)
		}
	}

	text := r.readFile("text.txt")
	if e != nil && len(e.Snippets) > 0 {
		found, ordered, missing := snippetsIn(text, e.Snippets)
		j.Metrics["snippetsFound"] = float64(found)
		j.Metrics["snippets"] = float64(len(e.Snippets))
		if found < len(e.Snippets) {
			soft("%d of %d source snippets missing, first: %q", len(e.Snippets)-found, len(e.Snippets), missing)
		}
		if !ordered {
			soft("source snippets appear out of reading order")
		}
	}

	if e != nil && len(e.Lettering) > 0 {
		rec := charRecall(strings.Join(e.Lettering, " "), r.readFile("plates.txt"))
		j.Metrics["letteringRecall"] = round2(rec)
		if p.Plates == 0 && !envBlocksOCR(r) {
			j.fail("no OCR plate on a page the author lettered (%d lettering lines)", len(e.Lettering))
		}
	} else if (c.Class == "comic-image" || c.Class == "comic-archive") && p.Plates == 0 && !envBlocksOCR(r) {
		j.advise("no OCR plate; whether the lettering is legible is not yet judged by a person")
	}
	if r.OCRMode == "on" && c.Class == "scanned-pdf" && e != nil && e.TextLayer == "none" && p.Plates == 0 && !envBlocksOCR(r) && !incomplete {
		j.fail("OCR enabled on an image-only scan and no plate was produced")
	}

	// Document language, distinct from the interface language. A picture's page takes its language
	// from the OCR language, so when the case's pack is missing the label says nothing about the
	// product.
	got := primaryTag(p.Lang)
	switch {
	case envBlocksOCR(r) && (c.Class == "comic-image" || c.Class == "comic-archive"):
	case got == "":
		j.advise("no document language on <html> (the desktop leaves it off for undeclared sources - docs/PARITY.md)")
	case got != c.Language:
		j.fail("document language %q, the source is %q", p.Lang, c.Language)
	}

	if c.Class == "epub" && p.TocEntries == 0 {
		j.advise("no table of contents entries")
	}
	if hasFeature(c, "vertical") && c.Class == "epub" && !p.VerticalWriting {
		j.advise("vertical writing is not preserved (the source is vertical)")
	}
	if hasFeature(c, "ruby") && p.RubyElements == 0 {
		j.advise("ruby annotations are not preserved")
	}
	return j
}

// incomplete reports that the collector stopped before the reader finished. The collection record
// is authoritative when present; the bare Truncated flag covers results written before it existed.
func (r *Result) incomplete() bool {
	if r.Collection != nil && r.Collection.Outcome != "" {
		return r.Collection.Unsettled()
	}
	return r.Truncated
}

func envBlocksOCR(r *Result) bool { return strings.Contains(r.EnvironmentGap, "OCR language") }

func finish(j *Judgement, c Case, e *Expectation) {
	switch {
	case len(j.Fails) > 0:
		j.Auto = Fail
	case len(j.Gaps) > 0:
		j.Auto = CouldNotVerify
	case len(j.Advisories) > 0:
		j.Auto = PassAdvisories
	default:
		j.Auto = Pass
	}
	j.Campaign = j.Auto
	if j.Auto == Fail || j.Auto == CouldNotVerify {
		return
	}
	var open []string
	if !c.Rights.HumanReviewed() {
		open = append(open, "rights review")
	}
	if e == nil || !e.IsTruth() {
		open = append(open, "expectation review")
	}
	if len(open) > 0 {
		j.Campaign = CouldNotVerify
		j.Gaps = append(j.Gaps, "human gate open: "+strings.Join(open, ", "))
	}
}

// snippetsIn looks for each snippet in the page text with every space removed on both sides, so a
// line break, a justified gap or a CJK line join cannot hide a match; case and Unicode forms are
// folded. It reports how many were found, whether the found ones are in order, and the first miss.
func snippetsIn(text string, snippets []string) (found int, ordered bool, firstMissing string) {
	hay := squash(text)
	ordered = true
	last := -1
	for _, s := range snippets {
		i := strings.Index(hay, squash(s))
		if i < 0 {
			if firstMissing == "" {
				firstMissing = s
			}
			continue
		}
		found++
		if i < last {
			ordered = false
		}
		last = i
	}
	return found, ordered, firstMissing
}

func squash(s string) string {
	s = norm.NFKC.String(s)
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) || r == '­' || r == '​' {
			continue
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
}

// charRecall is the share of the reference's letters and digits (as a multiset) that the candidate
// also holds. It is script-agnostic and order-free: a measurement, not a threshold.
func charRecall(ref, got string) float64 {
	count := func(s string) map[rune]int {
		m := map[rune]int{}
		for _, r := range norm.NFKC.String(s) {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				m[unicode.ToLower(r)]++
			}
		}
		return m
	}
	want, have := count(ref), count(got)
	total, hit := 0, 0
	for r, n := range want {
		total += n
		hit += min(n, have[r])
	}
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total)
}

func primaryTag(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if i := strings.IndexAny(tag, "-_"); i >= 0 {
		tag = tag[:i]
	}
	return tag
}

func hasFeature(c Case, f string) bool { return contains(c.Features, f) }

func lastErrorLine(log string) string {
	lines := strings.Split(strings.TrimSpace(log), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if strings.Contains(strings.ToLower(l), "error") {
			return l
		}
	}
	if len(lines) > 0 {
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return ""
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
