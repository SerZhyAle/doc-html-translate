// Package appearance builds the CSS both editions share - the OCR overlay unit and the reader
// theme palette - from appearance.json, the one place those values are written. The extension
// derives its copy from the same file (extension/scripts/gen-appearance.mjs), and
// tests/appearance_parity_test.go holds the two together. Names are not in the source: each caller
// supplies its own selectors and custom-property prefix.
package appearance

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed appearance.json
var raw []byte

// Role names, in the order the overlay stylesheet emits them.
const (
	RoleContainer = "container"
	RoleImage     = "image"
	RolePlate     = "plate"
)

// Roles lists every overlay role the source defines, in emission order.
var Roles = []string{RoleContainer, RoleImage, RolePlate}

// HiddenPlateDecl is the one declaration of the hidden-plate rule. It is fixed by the schema
// rather than listed in the source because it is what the toggle means, not a styling choice.
const HiddenPlateDecl = "display:none"

// Decl is one declaration of a role. Note, when set, is the measurement or reason behind a
// load-bearing value and ships as a CSS comment in front of it.
type Decl struct {
	Property string `json:"property"`
	Value    string `json:"value"`
	Note     string `json:"note,omitempty"`
}

// Color is one palette token of a theme, e.g. {"barBg", "#ffffff"}.
type Color struct {
	Token string
	Value string
}

// Theme is one reader theme; its colours are in source order.
type Theme struct {
	Name   string
	Colors []Color
}

// Divergence is a difference between the editions that the parity gate treats as legal. Role,
// Property and Edition are set when the entry lets one declaration differ on one edition.
type Divergence struct {
	What     string `json:"what"`
	Reason   string `json:"reason"`
	Role     string `json:"role,omitempty"`
	Property string `json:"property,omitempty"`
	Edition  string `json:"edition,omitempty"`
}

// Source is the decoded appearance.json. Themes keep their source order because the first one
// is the default and is emitted on :root.
type Source struct {
	Roles       map[string][]Decl
	Themes      []Theme
	Divergences []Divergence
	Notes       []string
}

var src = mustDecode(raw)

// Load returns the decoded source. The data is shared; callers must not modify it.
func Load() Source { return src }

// OverlayNames are one edition's selectors for the overlay roles.
type OverlayNames struct {
	Container   string // e.g. ".ocr-fig"
	Image       string // e.g. ".ocr-fig>img"
	Plate       string // e.g. ".ocr-box"
	HiddenPlate string // e.g. "html.dht-ocr-off .ocr-box"
}

// PaletteNames are one edition's names for the palette: the custom-property prefix (without the
// leading "--") and the attribute on <html> that selects a non-default theme.
type PaletteNames struct {
	Prefix string // e.g. "dht-"
	Attr   string // e.g. "data-dht-theme"
}

// OverlayCSS returns the overlay stylesheet - container, image, plate and hidden-plate rules -
// minified, one rule per line, with each note as a comment in front of its declaration.
func OverlayCSS(n OverlayNames) string {
	var sb strings.Builder
	for _, r := range []struct{ sel, role string }{
		{n.Container, RoleContainer}, {n.Image, RoleImage}, {n.Plate, RolePlate},
	} {
		sb.WriteString(r.sel)
		sb.WriteByte('{')
		for i, d := range src.Roles[r.role] {
			if i > 0 {
				sb.WriteByte(';')
			}
			if d.Note != "" {
				sb.WriteString("/* " + d.Note + " */")
			}
			sb.WriteString(d.Property + ":" + minifyValue(d.Value))
		}
		sb.WriteString("}\n")
	}
	sb.WriteString(n.HiddenPlate + "{" + HiddenPlateDecl + "}")
	return sb.String()
}

// PaletteCSS returns the theme palette: the first theme on :root, every other one under
// html[<Attr>="<name>"], one rule per line.
func PaletteCSS(n PaletteNames) string {
	var sb strings.Builder
	for i, th := range src.Themes {
		if i == 0 {
			sb.WriteString(":root{")
		} else {
			fmt.Fprintf(&sb, "\nhtml[%s=%q]{", n.Attr, th.Name)
		}
		for j, c := range th.Colors {
			if j > 0 {
				sb.WriteByte(';')
			}
			sb.WriteString("--" + n.Prefix + TokenProperty(c.Token) + ":" + c.Value)
		}
		sb.WriteByte('}')
	}
	return sb.String()
}

// TokenProperty maps a camelCase palette token to its kebab-case CSS name: barBg -> bar-bg.
func TokenProperty(token string) string {
	var sb strings.Builder
	for _, r := range token {
		if r >= 'A' && r <= 'Z' {
			sb.WriteByte('-')
			r += 'a' - 'A'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// minifyValue drops the space after a comma outside quotes, which is how the desktop stylesheet
// has always been written (`"Segoe UI",system-ui`). Rendering is identical either way.
func minifyValue(v string) string {
	var sb strings.Builder
	inQuote := byte(0)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case inQuote != 0:
			if c == inQuote {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
		case c == ' ' && i > 0 && v[i-1] == ',':
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}

func mustDecode(b []byte) Source {
	s, err := decode(b)
	if err != nil {
		panic("appearance.json: " + err.Error())
	}
	return s
}

func decode(b []byte) (Source, error) {
	var doc struct {
		Roles       map[string][]Decl `json:"roles"`
		Themes      json.RawMessage   `json:"themes"`
		Divergences []Divergence      `json:"divergences"`
		Notes       []string          `json:"notes"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return Source{}, err
	}
	s := Source{Roles: doc.Roles, Divergences: doc.Divergences, Notes: doc.Notes}
	for _, r := range Roles {
		if len(s.Roles[r]) == 0 {
			return Source{}, fmt.Errorf("role %q is missing or empty", r)
		}
	}
	if len(s.Roles) != len(Roles) {
		return Source{}, fmt.Errorf("roles has %d entries, want exactly %v", len(s.Roles), Roles)
	}
	names, err := orderedKeys(doc.Themes)
	if err != nil {
		return Source{}, fmt.Errorf("themes: %w", err)
	}
	var byName map[string]json.RawMessage
	if err := json.Unmarshal(doc.Themes, &byName); err != nil {
		return Source{}, fmt.Errorf("themes: %w", err)
	}
	for _, name := range names {
		tokens, err := orderedKeys(byName[name])
		if err != nil {
			return Source{}, fmt.Errorf("theme %q: %w", name, err)
		}
		var values map[string]string
		if err := json.Unmarshal(byName[name], &values); err != nil {
			return Source{}, fmt.Errorf("theme %q: %w", name, err)
		}
		th := Theme{Name: name}
		for _, tok := range tokens {
			th.Colors = append(th.Colors, Color{Token: tok, Value: values[tok]})
		}
		s.Themes = append(s.Themes, th)
	}
	if len(s.Themes) == 0 {
		return Source{}, fmt.Errorf("themes is empty")
	}
	return s, nil
}

// orderedKeys returns the keys of a JSON object in document order; encoding/json's map decoding
// loses it, and theme order decides which theme is the default.
func orderedKeys(obj json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(obj))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("not a JSON object")
	}
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, t.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}
