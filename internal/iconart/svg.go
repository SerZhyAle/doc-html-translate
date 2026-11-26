package iconart

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// The vector forms of the mark (PRODUCT-IDENTITY section 3.1 rule 1). The drawing is defined
// once, in mark.go; these SVGs are that drawing written out path for path, plated, so the
// master a person or another product opens is the same picture the rasterizer draws.
const (
	// MasterSVGPath is the master: the large drawing, the one every store and site picture shows.
	MasterSVGPath = "assets/identity/master/mark.svg"
	// MediumSVGPath is the reduced form the product draws from 24 to 95 px.
	MediumSVGPath = "assets/identity/master/mark-medium.svg"
	// SmallSVGPath is the small form, below 24 px.
	SmallSVGPath = "assets/identity/master/mark-small.svg"
	// HelpHeadSVGPath is the picture beside the title of every help page (rule 8): the help
	// generator inlines it, so the page makes no request for it.
	HelpHeadSVGPath = "assets/identity/out/help-head.svg"
)

// SVGOutput is one vector file the generator writes, relative to the repository root.
type SVGOutput struct {
	Path string
	Data []byte
}

// SVGOutputs are the vector forms and the help-page head, each the drawing Mark uses at the
// size named.
func SVGOutputs() []SVGOutput {
	return []SVGOutput{
		{MasterSVGPath, MarkSVG(256)},
		{MediumSVGPath, MarkSVG(48)},
		{SmallSVGPath, MarkSVG(16)},
		{HelpHeadSVGPath, MarkSVG(48)},
	}
}

// MarkSVG writes the plated mark as the drawing Mark picks for a box of px pixels, on that
// drawing's own grid. Every shape is a path filled by the nonzero rule, as the rasterizer
// fills it; the code shapes stay separate paths so none can cancel another.
func MarkSVG(px float64) []byte {
	d := drawingFor(px)
	var b strings.Builder
	g := num(d.grid)
	fmt.Fprintf(&b, "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %s %s\">\n", g, g)
	path := func(segs []segment, c color.NRGBA) {
		fmt.Fprintf(&b, "<path fill=\"#%02X%02X%02X\" d=\"%s\"/>\n", c.R, c.G, c.B, pathData(segs))
	}
	path(d.plate(plateInset(d, px)), Navy)
	path(d.sheet, Sheet)
	path(d.fold, Fold)
	for _, c := range d.code {
		path(c, Navy)
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func pathData(segs []segment) string {
	var b strings.Builder
	for i, s := range segs {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(s.op)
		n := 0
		switch s.op {
		case 'M', 'L':
			n = 1
		case 'C':
			n = 3
		}
		for _, p := range s.pts[:n] {
			b.WriteByte(' ')
			b.WriteString(num(p.x))
			b.WriteByte(',')
			b.WriteString(num(p.y))
		}
	}
	return b.String()
}

// num prints a coordinate to three decimals, without trailing zeros: the same drawing always
// writes the same bytes.
func num(v float64) string {
	r := math.Round(v*1000) / 1000
	if r == 0 {
		r = 0 // no "-0"
	}
	return strconv.FormatFloat(r, 'f', -1, 64)
}
