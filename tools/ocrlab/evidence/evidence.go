// Package evidence is the contract between the two editions and the scorer.
//
// The desktop runner and the browser-extension runner are separate programs in separate
// languages driving separate OCR engines, and the strategic spec accepts that their recognized
// text will differ. What may not differ is the shape of what they report: both are graded
// against the same annotations by the same code, so both must describe a run in the same terms.
// That is this file. A parity test reads the field names out of both implementations and fails
// when they drift.
//
// Everything geometric is in natural image pixels. A plate's rect is read back from the browser
// in CSS pixels and mapped through the rendered image's scale before it is written here, so a
// run at three viewports produces three plate sets that are directly comparable to one another
// and to the annotation.
package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"doc-html-translate/tools/ocrlab/truth"
)

// SchemaVersion is the cross-edition contract version. Guarded by TestParityOCRLabEvidenceSchema.
const SchemaVersion = 1

// Edition names which implementation produced the run.
type Edition string

const (
	EditionDesktop   Edition = "desktop"
	EditionExtension Edition = "extension"
)

// Mode is how a plate concealed the source lettering. Phase 04 writes ModeFill for every plate
// because that is what both editions do today; Phase 07 is what makes the field interesting.
// It is recorded from the first run so a mode change shows up as a diff in the evidence rather
// than as an unexplained movement in the scores.
type Mode string

const (
	ModeFill        Mode = "fill"        // opaque block of the sampled background colour
	ModeReconstruct Mode = "reconstruct" // background rebuilt from the local surroundings
	ModeMask        Mode = "mask"        // painted only where the glyphs are
)

// Rect is an axis-aligned box in natural image pixels.
type Rect struct {
	X0 int `json:"x0"`
	Y0 int `json:"y0"`
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
}

// Region converts the rect for the geometry helpers in package truth.
func (r Rect) Region() truth.Region { return truth.Box("", r.X0, r.Y0, r.X1, r.Y1) }

// Width and Height in natural image pixels.
func (r Rect) Width() int  { return r.X1 - r.X0 }
func (r Rect) Height() int { return r.Y1 - r.Y0 }

// Viewport is one pinned browser geometry. Recording the device scale factor matters because a
// plate that is correct at 1x and drifts at 2x is a real defect and an aggregate over mixed
// scales would hide it.
type Viewport struct {
	Name              string  `json:"name"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
}

// Plate is one rendered text plate as the browser actually laid it out.
//
// ScrollHeight and ClientHeight are read back from the DOM rather than computed: whether a
// translation clipped is a fact about the rendered box, and the runtime re-fit means a
// compile-time estimate would be wrong. A plate is recorded once per (viewport, stress case).
type Plate struct {
	Text       string `json:"text"`
	Rect       Rect   `json:"rect"`
	Viewport   string `json:"viewport"`
	StressCase string `json:"stressCase"`

	FontPx     float64 `json:"fontPx"`
	Background string  `json:"background"`
	Ink        string  `json:"ink"`

	Mode           Mode    `json:"mode"`
	ModeConfidence float64 `json:"modeConfidence"`

	ScrollHeight int `json:"scrollHeight"`
	ClientHeight int `json:"clientHeight"`
}

// ClipSlackPx is how much overflow is layout rounding rather than hidden text.
//
// The shipped re-fit uses one pixel, and the lab needs more: scrollHeight and clientHeight are
// each rounded to an integer from a fractional layout, so a box that fits exactly can still
// report a two or three pixel difference. Measured over the 45-scene corpus on 2026-08-12, every
// plate the old one-pixel rule called clipped overshot by exactly 2 or 3 px and none by more -
// the re-fit's height:auto escape had fired on all of them and nothing was hidden. Four pixels
// clears that residue and cannot hide a real clip, because hiding text costs at least one line
// and a line is at least a font size tall.
const ClipSlackPx = 4

// Clipped reports whether the browser said this plate's content overflows its box.
func (p Plate) Clipped() bool { return p.ScrollHeight > p.ClientHeight+ClipSlackPx }

// Screenshots are paths relative to the run directory, so a run folder can be moved or zipped
// and the report still opens.
type Screenshots struct {
	Source   string            `json:"source"`
	Rendered string            `json:"rendered"`
	Stress   map[string]string `json:"stress,omitempty"`
}

// Scene is one corpus scene's outcome.
//
// Error is not an afterthought: a scene the runner could not process must appear in the
// evidence with its reason, because a run that silently dropped it would report better numbers
// over fewer scenes - the exact dishonesty the strategic spec is written against.
type Scene struct {
	SceneID     string      `json:"sceneId"`
	ImageWidth  int         `json:"imageWidth"`
	ImageHeight int         `json:"imageHeight"`
	Plates      []Plate     `json:"plates"`
	Screenshots Screenshots `json:"screenshots"`

	OcrMs        int64 `json:"ocrMs"`
	RenderMs     int64 `json:"renderMs"`
	PeakRSSBytes int64 `json:"peakRssBytes"`

	Error string `json:"error,omitempty"`
}

// PlatesFor returns the plates recorded for one viewport and stress case.
func (s *Scene) PlatesFor(viewport, stressCase string) []Plate {
	var out []Plate
	for _, p := range s.Plates {
		if p.Viewport == viewport && p.StressCase == stressCase {
			out = append(out, p)
		}
	}
	return out
}

// StressCases lists the stress cases present in this scene's plates, in first-seen order.
func (s *Scene) StressCases() []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range s.Plates {
		if !seen[p.StressCase] {
			seen[p.StressCase] = true
			out = append(out, p.StressCase)
		}
	}
	return out
}

// Engine records what did the recognizing, so a score can be attributed to a version rather
// than to a mood.
type Engine struct {
	Tesseract       string `json:"tesseract"`
	TessdataVersion string `json:"tessdataVersion"`
	Lang            string `json:"lang"`
}

// Browser records what did the rendering.
type Browser struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Run is one execution of one edition over a set of scenes.
type Run struct {
	SchemaVersion int        `json:"schemaVersion"`
	RunID         string     `json:"runId"`
	StartedAt     string     `json:"startedAt"`
	Edition       Edition    `json:"edition"`
	Engine        Engine     `json:"engine"`
	Browser       Browser    `json:"browser"`
	Viewports     []Viewport `json:"viewports"`
	Scenes        []Scene    `json:"scenes"`
}

// Find returns the scene with the given ID, or nil.
func (r *Run) Find(sceneID string) *Scene {
	for i := range r.Scenes {
		if r.Scenes[i].SceneID == sceneID {
			return &r.Scenes[i]
		}
	}
	return nil
}

// Save writes the run as indented JSON.
func (r *Run) Save(path string) error {
	r.SchemaVersion = SchemaVersion
	buf, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

// LoadRun reads a run written by either edition.
func LoadRun(path string) (*Run, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Run
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if r.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%s: schemaVersion %d, this build understands %d",
			path, r.SchemaVersion, SchemaVersion)
	}
	return &r, nil
}
