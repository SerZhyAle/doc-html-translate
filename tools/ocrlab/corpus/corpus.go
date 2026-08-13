// Package corpus describes the visual-fidelity corpus: what a scene is, where it came from,
// under which licence, and whether it may be tuned against.
//
// The manifest is versioned metadata only - it never holds media. The media itself lives under
// the gitignored corpus root (test_doc/ocrlab by default) and ships in nothing. That split is
// what makes the corpus reproducible without committing megabytes of other people's work, and
// it is why every scene carries a hash: the manifest is the claim, the file on disk is the
// thing, and Validate is what confronts one with the other.
//
// See DEV/plan/2026-08-11_ocr-visual-fidelity-lab.md sections 4.1 and 4.2 for the rules this
// package encodes.
package corpus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// SchemaVersion is the manifest contract. Changing the meaning of a field, removing one, or
// tightening a rule that would invalidate existing entries means bumping this - never a silent
// reinterpretation of data a human already verified.
const SchemaVersion = 1

// Licence is a closed set on purpose. The strategic spec forbids non-commercial,
// no-derivatives, unclear, revoked and user-private material; making those unspellable means a
// bad entry fails to parse rather than passing review by looking plausible.
type Licence string

const (
	LicencePD        Licence = "PD"        // public domain by age or by declaration
	LicencePDM       Licence = "PDM"       // Public Domain Mark 1.0
	LicenceCC0       Licence = "CC0"       // CC0 1.0 dedication
	LicenceCCBY30    Licence = "CC-BY-3.0" // attribution required
	LicenceCCBY40    Licence = "CC-BY-4.0" // attribution required
	LicenceOwn       Licence = "OWN"       // the author's own material, kept local
	LicenceSynthetic Licence = "SYNTHETIC" // drawn by tools/ocrlab/synth, no third party involved
)

var allLicences = []Licence{
	LicencePD, LicencePDM, LicenceCC0, LicenceCCBY30, LicenceCCBY40, LicenceOwn, LicenceSynthetic,
}

// Valid reports whether l is one of the seven accepted licences.
func (l Licence) Valid() bool {
	for _, k := range allLicences {
		if l == k {
			return true
		}
	}
	return false
}

// NeedsAttribution is true for the CC BY variants only. PD/PDM/CC0 need none, and OWN/SYNTHETIC
// have no third party to credit.
func (l Licence) NeedsAttribution() bool {
	return l == LicenceCCBY30 || l == LicenceCCBY40
}

// NeedsDownload is true when the media comes from somewhere else. OWN material is placed by
// hand and SYNTHETIC material is generated, so neither has a URL to fetch.
func (l Licence) NeedsDownload() bool {
	return l != LicenceOwn && l != LicenceSynthetic
}

// Category is a scene label from the strategic spec's coverage table. A scene carries one or
// more; the counts in that table overlap by design, which is why Categories is a slice.
type Category string

const (
	CatDocument Category = "document" // clean printed or scanned documents
	CatPoster   Category = "poster"   // posters, labels and signs
	CatComic    Category = "comic"    // comics and illustrated dialogue
	CatCartoon  Category = "cartoon"  // caricatures, cartoons and memes
	CatTexture  Category = "texture"  // text over imagery or texture
	CatDegraded Category = "degraded" // skew, blur, scan noise, low DPI, compression, crop
	CatScript   Category = "script"   // script and direction variants
)

var allCategories = []Category{
	CatDocument, CatPoster, CatComic, CatCartoon, CatTexture, CatDegraded, CatScript,
}

// Categories returns every known category in table order, for reporting.
func Categories() []Category { return append([]Category(nil), allCategories...) }

// Valid reports whether c is one of the seven table categories.
func (c Category) Valid() bool {
	for _, k := range allCategories {
		if c == k {
			return true
		}
	}
	return false
}

// Split decides what a scene may be used for. Parameters and algorithms may be tuned on dev
// scenes only; once a scene enters the holdout it does not move back after a failure.
type Split string

const (
	SplitDev     Split = "dev"
	SplitHoldout Split = "holdout"
)

// Valid reports whether s is a known split.
func (s Split) Valid() bool { return s == SplitDev || s == SplitHoldout }

// Scene is one image with text baked into its pixels, plus everything needed to fetch it
// again, prove it is what it claims to be, and know what it may be used for.
//
// LicenceVerifiedBy is the human gate. An entry is not valid until a person has opened the
// asset's own licence page - a search-result snippet or a site's category label is not licence
// proof - and recorded who did so. No code path may fill this field.
type Scene struct {
	ID   string `json:"id"`
	File string `json:"file"` // path relative to the corpus root, forward slashes

	// Provenance
	SourceURL   string  `json:"sourceUrl"`
	SourceItem  string  `json:"sourceItem,omitempty"` // archive.org item, Commons file name, ..
	RetrievedOn string  `json:"retrievedOn"`          // YYYY-MM-DD
	Licence     Licence `json:"licence"`
	LicenceURL  string  `json:"licenceUrl"`
	Attribution string  `json:"attribution,omitempty"` // required text for CC BY

	// The human gate (section 4.2)
	LicenceVerifiedBy string `json:"licenceVerifiedBy"`
	LicenceVerifiedOn string `json:"licenceVerifiedOn"` // YYYY-MM-DD

	// Identity of the bytes
	SHA256 string `json:"sha256"` // empty = not yet measured, and Validate says so
	Bytes  int64  `json:"bytes,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`

	// Labels
	Languages  []string   `json:"languages,omitempty"` // BCP-47-ish, e.g. en, ru, ar
	Scripts    []string   `json:"scripts,omitempty"`   // ISO 15924: Latn, Cyrl, Arab, Deva, Hans
	Categories []Category `json:"categories"`
	Split      Split      `json:"split"`

	// A crop, blur, downscale or synthetic-translation variant stays tied to its source entry
	// and never overwrites the downloaded original.
	DerivedFrom string `json:"derivedFrom,omitempty"`

	Note string `json:"note,omitempty"` // "unscored" excuses a scene from needing an annotation
}

// Scored reports whether the scene participates in the quality gate. A scene explicitly noted
// as unscored is corpus context - a page kept for a perf measurement, say - and is not expected
// to carry an annotation.
func (s *Scene) Scored() bool { return s.Note != NoteUnscored }

// NoteUnscored is the one Note value with a meaning to the tooling.
const NoteUnscored = "unscored"

// Manifest is the whole corpus description. Scenes are kept sorted by ID on Save so a
// regenerated manifest produces a readable diff instead of a reordering.
type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	Scenes        []Scene `json:"scenes"`
}

// Load reads a manifest. A missing file is an error, not an empty corpus: silently scoring
// zero scenes is exactly the failure mode the strategic spec forbids.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	if m.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("manifest %s: schemaVersion %d, this build understands %d",
			path, m.SchemaVersion, SchemaVersion)
	}
	return &m, nil
}

// Save writes the manifest with a stable scene order and two-space indent, so re-running a
// generator that changes nothing leaves the file byte-identical.
func Save(path string, m *Manifest) error {
	m.SchemaVersion = SchemaVersion
	sort.Slice(m.Scenes, func(i, j int) bool { return m.Scenes[i].ID < m.Scenes[j].ID })
	buf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

// Find returns the scene with the given ID, or nil.
func (m *Manifest) Find(id string) *Scene {
	for i := range m.Scenes {
		if m.Scenes[i].ID == id {
			return &m.Scenes[i]
		}
	}
	return nil
}

// Upsert replaces the entry with s.ID or appends it. Generators use this so re-running one is
// idempotent rather than duplicating its scenes.
func (m *Manifest) Upsert(s Scene) {
	for i := range m.Scenes {
		if m.Scenes[i].ID == s.ID {
			m.Scenes[i] = s
			return
		}
	}
	m.Scenes = append(m.Scenes, s)
}

// Select returns the scenes matching a split ("all" for every scene) and, when ids is
// non-empty, restricted to those IDs. An ID that matches nothing is returned as missing rather
// than dropped, because a typo in a scene name must not look like a clean run.
func (m *Manifest) Select(split string, ids []string) (sel []*Scene, missing []string) {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	seen := make(map[string]bool, len(ids))
	for i := range m.Scenes {
		s := &m.Scenes[i]
		if len(want) > 0 {
			if !want[s.ID] {
				continue
			}
			seen[s.ID] = true
		} else if split != "all" && string(s.Split) != split {
			continue
		}
		sel = append(sel, s)
	}
	for _, id := range ids {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	return sel, missing
}

// Path is the scene's media file under the corpus root.
func (s *Scene) Path(root string) string {
	return filepath.Join(root, filepath.FromSlash(s.File))
}

// HashFile returns the lower-case hex SHA-256 of a file.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
