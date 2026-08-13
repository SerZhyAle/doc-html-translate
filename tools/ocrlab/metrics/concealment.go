package metrics

import (
	"image"
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// Ink detection and contrast constants. They describe how the measurement looks at pixels, not
// what counts as good - a bound on the resulting number belongs in the thresholds file.
const (
	// inkLumaDelta is how far a pixel's luma must sit from its neighbourhood's median before it
	// is called ink. Low enough to catch grey newsprint lettering, high enough to ignore paper
	// grain and JPEG mottle.
	inkLumaDelta = 40
	// haloBandFraction is the band just inside a plate's edge, as a fraction of the plate's
	// shorter side, where a block fill characteristically leaves the tops and tails of the
	// original glyphs. Measured separately because a whole-region average dilutes it away.
	haloBandFraction = 0.18
)

// ResidualScore is how much of the original lettering a reader can still see.
type ResidualScore struct {
	Residual float64 `json:"residual"` // fraction of the source ink mask still showing ink
	Halo     float64 `json:"halo"`     // the same, measured only in the band inside the plate edge
	InkPx    int     `json:"inkPx"`    // size of the source ink mask, so a tiny sample is visible
}

// luma is the same weighting internal/ocr uses for its contrast floor, so the lab and the app
// agree on what "darker" means.
func luma(r, g, b uint32) int {
	return (299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000
}

// inkMask marks the pixels inside a region that stand out from the region's own median luma.
// Local by construction: the median is taken over the region, not the page, because a caption
// on a dark panel and a line on white paper have opposite polarity and a global threshold gets
// one of them wrong.
func inkMask(img image.Image, r truth.Region, w, h int) *truth.Mask {
	region := r.Rasterize(w, h)
	x0, y0, x1, y1 := r.Bounds()
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, w), min(y1, h)

	var lumas []int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !region.At(x, y) {
				continue
			}
			cr, cg, cb, _ := img.At(x, y).RGBA()
			lumas = append(lumas, luma(cr, cg, cb))
		}
	}
	m := truth.NewMask(w, h)
	if len(lumas) == 0 {
		return m
	}
	sort.Ints(lumas)
	med := lumas[len(lumas)/2]

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !region.At(x, y) {
				continue
			}
			cr, cg, cb, _ := img.At(x, y).RGBA()
			if absInt(luma(cr, cg, cb)-med) > inkLumaDelta {
				m.Set(x, y)
			}
		}
	}
	return m
}

// Covered is the fraction of the group's own text area that an opaque plate hides. Geometric,
// so it works without a rendered screenshot and is the cheap check; ResidualInk is the honest
// one and needs the render.
func Covered(plates []evidence.Plate, g truth.Group, w, h int) float64 {
	regions := g.Lines
	if len(regions) == 0 {
		regions = []truth.Region{g.Bounds}
	}
	text := truth.RasterizeAll(regions, w, h)
	area := text.Area()
	if area == 0 {
		return 0
	}
	return float64(text.IntersectArea(PlateMask(plates, w, h))) / float64(area)
}

// ResidualInk measures what a reader can actually still see.
//
// The source image gives the ink mask - which pixels were lettering. The rendered image is then
// asked, at exactly those pixels, whether something ink-like is still there. That catches the
// two failures a geometric check cannot: a plate that is slightly too small and leaves ascenders
// showing, and a plate whose sampled "paper" colour is transparent enough or wrong enough that
// the original shows through.
func ResidualInk(source, rendered image.Image, g truth.Group, w, h int) ResidualScore {
	regions := g.Lines
	if len(regions) == 0 {
		regions = []truth.Region{g.Bounds}
	}
	var s ResidualScore
	srcInk := truth.NewMask(w, h)
	for _, r := range regions {
		srcInk.Or(inkMask(source, r, w, h))
	}
	s.InkPx = srcInk.Area()
	if s.InkPx == 0 {
		return s
	}
	renderedInk := truth.NewMask(w, h)
	for _, r := range regions {
		renderedInk.Or(inkMask(rendered, r, w, h))
	}
	s.Residual = float64(srcInk.IntersectArea(renderedInk)) / float64(s.InkPx)

	// The halo band: a ring just inside the group's bounds.
	x0, y0, x1, y1 := g.Bounds.Bounds()
	band := int(float64(min(x1-x0, y1-y0)) * haloBandFraction)
	if band < 1 {
		return s
	}
	ring := truth.NewMask(w, h)
	outer := g.Bounds.Rasterize(w, h)
	inner := truth.Box("", x0+band, y0+band, x1-band, y1-band).Rasterize(w, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if outer.At(x, y) && !inner.At(x, y) {
				ring.Set(x, y)
			}
		}
	}
	ringInk := truth.NewMask(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ring.At(x, y) && srcInk.At(x, y) {
				ringInk.Set(x, y)
			}
		}
	}
	if n := ringInk.Area(); n > 0 {
		s.Halo = float64(ringInk.IntersectArea(renderedInk)) / float64(n)
	}
	return s
}

// ContrastScore is whether the replacement text is legible where it was drawn.
type ContrastScore struct {
	MinLuma  float64 `json:"minLumaSeparation"`
	MeanLuma float64 `json:"meanLumaSeparation"`
	Plates   int     `json:"plates"`
}

// RenderedContrast measures the luma separation between a plate's drawn text and its drawn
// background *in the rendered image*.
//
// The strategic spec is explicit that colour adaptation is a quality aid and not permission to
// draw unreadable text, and the sampled bg/ink values recorded in the evidence are what the code
// intended - not what landed. A plate whose ink was sampled from a shadow, or whose background
// was re-derived by the browser's own rendering, only shows up when the pixels are read back.
func RenderedContrast(rendered image.Image, plates []evidence.Plate, w, h int) ContrastScore {
	s := ContrastScore{MinLuma: -1}
	var seps []float64
	for _, p := range plates {
		region := p.Rect.Region()
		x0, y0, x1, y1 := region.Bounds()
		x0, y0 = max(x0, 0), max(y0, 0)
		x1, y1 = min(x1, w), min(y1, h)
		if x1-x0 < 2 || y1-y0 < 2 {
			continue
		}
		var lumas []int
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				cr, cg, cb, _ := rendered.At(x, y).RGBA()
				lumas = append(lumas, luma(cr, cg, cb))
			}
		}
		if len(lumas) < 4 {
			continue
		}
		sort.Ints(lumas)
		// Background is the plate's dominant colour (median); text is the minority that stands
		// away from it. The 5th/95th percentiles resist antialiasing at glyph edges.
		med := lumas[len(lumas)/2]
		lo := lumas[len(lumas)*5/100]
		hi := lumas[len(lumas)*95/100]
		sep := float64(max(absInt(med-lo), absInt(hi-med)))
		seps = append(seps, sep)
		if s.MinLuma < 0 || sep < s.MinLuma {
			s.MinLuma = sep
		}
		s.Plates++
	}
	if s.MinLuma < 0 {
		s.MinLuma = 0
	}
	s.MeanLuma = mean(seps)
	return s
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
