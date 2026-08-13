// Package truth is the corpus's ground truth: what an image actually says, where each reading
// group sits, and which parts of the picture a replacement may never paint over.
//
// The one rule the whole lab rests on is enforced here. OCR output may *seed* an annotation to
// save a human's time, but a seeded record is marked as such and IsTruth reports false for it,
// so the engine can never be scored against its own output. Everything else in this package
// exists to make that rule mechanical rather than a matter of discipline.
package truth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaVersion is the annotation contract. Same rule as the manifest: a change in meaning is a
// version bump, never a silent reinterpretation of records a human already reviewed.
const SchemaVersion = 1

// Origin says who produced the record. It is deliberately not omitempty: an unset origin must
// be visible in the file rather than defaulting to the trusted value.
type Origin string

const (
	OriginHuman   Origin = "human"
	OriginOCRSeed Origin = "ocr-seed"
)

// GroupType is what kind of reading unit this is. The distinction matters to the metrics: a
// merged pair of balloons is a defect, whereas two document lines joining into a paragraph is
// the intended behaviour.
type GroupType string

const (
	GroupDocumentLine GroupType = "document-line"
	GroupParagraph    GroupType = "paragraph"
	GroupBalloon      GroupType = "balloon"
	GroupCaption      GroupType = "caption"
	GroupLabel        GroupType = "label"
	GroupTitle        GroupType = "title"
	GroupIncidental   GroupType = "incidental"
)

var allGroupTypes = []GroupType{
	GroupDocumentLine, GroupParagraph, GroupBalloon, GroupCaption,
	GroupLabel, GroupTitle, GroupIncidental,
}

// Valid reports whether t is a known group type.
func (t GroupType) Valid() bool {
	for _, k := range allGroupTypes {
		if t == k {
			return true
		}
	}
	return false
}

// Direction is the reading direction of a group's script.
type Direction string

const (
	DirLTR Direction = "ltr"
	DirRTL Direction = "rtl"
)

// Ambiguity records how legible the source really is. Historical lettering that a careful human
// cannot read must not be counted as an OCR regression, so a scored dimension can exclude
// anything that is not clear.
type Ambiguity string

const (
	AmbiguityClear     Ambiguity = "clear"
	AmbiguityPartly    Ambiguity = "partly-illegible"
	AmbiguityIllegible Ambiguity = "illegible"
)

// Valid reports whether a is a known ambiguity label.
func (a Ambiguity) Valid() bool {
	return a == AmbiguityClear || a == AmbiguityPartly || a == AmbiguityIllegible
}

// Group is one reading unit: a balloon, a caption, a document paragraph.
//
// Bounds is where the text sits. ReplaceArea is where a plate is *permitted* to paint, and is
// often tighter - inside a balloon it stops short of the outline, over artwork it may be only
// the line boxes. Keeping the two apart is what lets the damage metric distinguish "covered its
// own text" from "painted over the drawing".
type Group struct {
	ID           string    `json:"id"`
	Type         GroupType `json:"type"`
	Transcript   string    `json:"transcript"`
	Language     string    `json:"language,omitempty"`
	Direction    Direction `json:"direction,omitempty"`
	ReadingOrder int       `json:"readingOrder"`
	Lines        []Region  `json:"lines"`
	Bounds       Region    `json:"bounds"`
	ReplaceArea  Region    `json:"replaceArea"`
	Ambiguity    Ambiguity `json:"ambiguity,omitempty"` // falls back to the annotation's label
}

// Review is the two-pairs-of-eyes record from the strategic spec. A holdout scene is not a gate
// until CheckedBy is filled by someone other than the annotator.
type Review struct {
	AnnotatedBy   string   `json:"annotatedBy"`
	AnnotatedOn   string   `json:"annotatedOn"`
	CheckedBy     string   `json:"checkedBy"`
	CheckedOn     string   `json:"checkedOn"`
	Disagreements []string `json:"disagreements,omitempty"`
}

// Annotation is everything known about one scene independently of any OCR engine.
type Annotation struct {
	SchemaVersion int    `json:"schemaVersion"`
	SceneID       string `json:"sceneId"`
	Origin        Origin `json:"origin"`

	ImageWidth  int `json:"imageWidth"`
	ImageHeight int `json:"imageHeight"`

	Groups    []Group  `json:"groups"`
	Protected []Region `json:"protected"`

	Ambiguity    Ambiguity `json:"ambiguity"`
	StressCases  []string  `json:"stressCases,omitempty"`
	ReviewerNote string    `json:"reviewerNote,omitempty"`
	Review       Review    `json:"review"`
}

// IsTruth reports whether this record may be scored against. An OCR-seeded draft never can - it
// is the engine's own output - and neither can a record nobody has signed. The holdout's second
// pair of eyes is checked by Validate, which knows the scene's split.
func (a *Annotation) IsTruth() bool {
	return a != nil && a.Origin == OriginHuman && strings.TrimSpace(a.Review.AnnotatedBy) != ""
}

// NotTruthReason explains a false IsTruth in words fit for a report, so a skipped scene is never
// just a smaller denominator.
func (a *Annotation) NotTruthReason() string {
	switch {
	case a == nil:
		return "no annotation"
	case a.Origin == OriginOCRSeed:
		return "OCR-seeded draft - the engine cannot be scored against its own output"
	case a.Origin != OriginHuman:
		return fmt.Sprintf("unknown origin %q", a.Origin)
	case strings.TrimSpace(a.Review.AnnotatedBy) == "":
		return "no annotator recorded"
	default:
		return ""
	}
}

// GroupAmbiguity is the group's own label, falling back to the scene's.
func (a *Annotation) GroupAmbiguity(g Group) Ambiguity {
	if g.Ambiguity != "" {
		return g.Ambiguity
	}
	return a.Ambiguity
}

// Load reads one annotation file.
func Load(path string) (*Annotation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a Annotation
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if a.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%s: schemaVersion %d, this build understands %d",
			path, a.SchemaVersion, SchemaVersion)
	}
	return &a, nil
}

// Save writes one annotation with a stable layout.
func Save(path string, a *Annotation) error {
	a.SchemaVersion = SchemaVersion
	buf, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

// LoadDir reads every <scene-id>.json in dir, keyed by scene ID. Draft files (*.draft.json) are
// loaded too - they are how a reviewer picks up where the seeder left off - but they carry
// OriginOCRSeed and so never score. A missing directory is not an error: a corpus with no
// annotations yet is a legitimate early state, and Validate is what reports the gap.
func LoadDir(dir string) (map[string]*Annotation, error) {
	out := map[string]*Annotation{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // "x.draft.json" sorts before "x.json", so a reviewed file wins
	for _, name := range names {
		a, err := Load(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if a.SceneID == "" {
			return nil, fmt.Errorf("%s: no sceneId", name)
		}
		out[a.SceneID] = a
	}
	return out, nil
}

// DraftPath and FinalPath are the two file names a scene's annotation can have.
func DraftPath(dir, sceneID string) string { return filepath.Join(dir, sceneID+".draft.json") }
func FinalPath(dir, sceneID string) string { return filepath.Join(dir, sceneID+".json") }
