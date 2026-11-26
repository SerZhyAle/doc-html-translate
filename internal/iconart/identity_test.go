package iconart

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

// PRODUCT-IDENTITY rule 1 and section 6 rung 3: the vector forms are well-formed SVG, carry
// the brand colours, and the same drawing always writes the same bytes.
func TestMarkSVGIsWellFormedAndStable(t *testing.T) {
	for _, o := range SVGOutputs() {
		if err := xml.Unmarshal(o.Data, new(struct{})); err != nil {
			t.Errorf("%s is not well-formed XML: %v", o.Path, err)
		}
		if !bytes.Contains(o.Data, []byte(`fill="#1E3A8A"`)) || !bytes.Contains(o.Data, []byte(`fill="#FFFFFF"`)) {
			t.Errorf("%s lacks the navy plate or the white sheet", o.Path)
		}
	}
	if !bytes.Equal(MarkSVG(256), MarkSVG(256)) {
		t.Error("two writes of the master differ")
	}
	if !strings.Contains(string(MarkSVG(256)), `viewBox="0 0 128 128"`) || !strings.Contains(string(MarkSVG(16)), `viewBox="0 0 16 16"`) {
		t.Error("the master is not the large drawing, or the small form not the small one")
	}
}

// The composed form fills the canvas edge to edge with the navy (rule 6: the declared background).
func TestOnBackgroundFillsTheCanvas(t *testing.T) {
	img := Mark(300, 200, 160, OnBackground)
	for _, p := range [][2]int{{0, 0}, {299, 0}, {0, 199}, {299, 199}} {
		if c := img.RGBAAt(p[0], p[1]); c.R != Navy.R || c.G != Navy.G || c.B != Navy.B || c.A != 255 {
			t.Errorf("corner %v is %v, want the navy", p, c)
		}
	}
}
