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

// Measurement states shared by every pixel metric. Anything that was not measured says so and
// carries no number: a zero would be read as "perfect".
const (
	StateMeasured   = "measured"
	StateUnmeasured = "unmeasured"
)

// How a residual was obtained. The known-mask basis is the only one that is independent of the
// image content; the flat-region basis is an estimate that is valid on a single-tone background.
const (
	BasisKnownMask  = "known-mask"
	BasisFlatRegion = "flat-region"
	BasisMixed      = "mixed"
)

// ResidualScore is how much of the original lettering a reader can still see.
//
// State separates "the reader can still see nothing" from "this was never measured". They are
// the same zero in JSON and the aggregates used to take both, so a run that stored no render
// scored as the best possible concealment - see the 2026-08-15 parity run. Anything reading
// Residual or Halo for a gate, a worst-of or a mean must check IsMeasured first.
//
// Scores written before State existed carry only Measured; IsMeasured reads those as before.
type ResidualScore struct {
	Residual float64 `json:"residual"` // fraction of the old lettering still showing
	Halo     float64 `json:"halo"`     // the same, measured only in the band inside the plate edge
	InkPx    int     `json:"inkPx"`    // size of the old-lettering sample, so a tiny sample is visible
	Measured bool    `json:"measured"` // false when there was nothing to compare against
	State    string  `json:"state,omitempty"`
	Reason   string  `json:"reason,omitempty"` // why a measurement is unmeasured
	Basis    string  `json:"basis,omitempty"`  // how the lettering was identified
}

// IsMeasured reports whether Residual and Halo mean anything.
func (r ResidualScore) IsMeasured() bool {
	switch r.State {
	case StateMeasured:
		return true
	case StateUnmeasured:
		return false
	}
	return r.Measured
}

func unmeasuredResidual(reason string) ResidualScore {
	return ResidualScore{State: StateUnmeasured, Reason: reason}
}

func measuredResidual(basis string) ResidualScore {
	return ResidualScore{State: StateMeasured, Measured: true, Basis: basis}
}

// luma is the same weighting internal/ocr uses for its contrast floor, so the lab and the app
// agree on what "darker" means.
func luma(r, g, b uint32) int {
	return (299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000
}

// regionLumas collects the luma of every pixel inside a region.
func regionLumas(img image.Image, r truth.Region, w, h int) []int {
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
	return lumas
}

// inkMask marks the pixels inside a region that stand out from the region's own median luma.
// Local by construction: the median is taken over the region, not the page, because a caption
// on a dark panel and a line on white paper have opposite polarity and a global threshold gets
// one of them wrong.
//
// Only meaningful where the background is a single tone; see unsupportedBackground.
func inkMask(img image.Image, r truth.Region, w, h int) *truth.Mask {
	region := r.Rasterize(w, h)
	x0, y0, x1, y1 := r.Bounds()
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, w), min(y1, h)

	lumas := regionLumas(img, r, w, h)
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
// so it works without a rendered screenshot and is the cheap check; the residual is the honest
// one and needs a text-hidden capture.
func Covered(plates []evidence.Plate, g truth.Group, w, h int) float64 {
	regions := groupRegions(g)
	text := truth.RasterizeAll(regions, w, h)
	area := text.Area()
	if area == 0 {
		return 0
	}
	return float64(text.IntersectArea(PlateMask(plates, w, h))) / float64(area)
}

// groupRegions is the group's line boxes, or its bounds when it has none.
func groupRegions(g truth.Group) []truth.Region {
	if len(g.Lines) == 0 {
		return []truth.Region{g.Bounds}
	}
	return g.Lines
}

// ResidualInk estimates what a reader can still see when no independent lettering mask exists.
//
// The source gives an ink mask - pixels far from the region's median luma. The capture is then
// asked, at exactly those pixels, whether something ink-like is still there. That catches the
// two failures a geometric check cannot: a plate that is slightly too small and leaves ascenders
// showing, and a plate whose sampled "paper" colour is transparent enough or wrong enough that
// the original shows through.
//
// The estimate rests on the background being one tone. On a gradient or a texture the background
// itself lands in the ink mask on both sides, so removing the real lettering barely moves the
// ratio (measured 90.86% and 96.89% with the lettering exactly removed). Such a region returns
// unmeasured with no percentage; ResidualKnownMask is the measurement that works there.
//
// Pass the text-hidden capture, not the normal render: replacement glyphs drawn on the old
// coordinates would otherwise count as old ink.
func ResidualInk(source, hidden image.Image, g truth.Group, w, h int) ResidualScore {
	if source == nil || hidden == nil {
		return unmeasuredResidual("source or text-hidden capture unavailable")
	}
	regions := groupRegions(g)
	for _, r := range regions {
		if reason := unsupportedBackground(regionLumas(source, r, w, h)); reason != "" {
			return unmeasuredResidual(reason + "; no independent lettering mask declared")
		}
	}
	s := measuredResidual(BasisFlatRegion)
	srcInk := truth.NewMask(w, h)
	for _, r := range regions {
		srcInk.Or(inkMask(source, r, w, h))
	}
	s.InkPx = srcInk.Area()
	if s.InkPx == 0 {
		return s
	}
	hiddenInk := truth.NewMask(w, h)
	for _, r := range regions {
		hiddenInk.Or(inkMask(hidden, r, w, h))
	}
	s.Residual = float64(srcInk.IntersectArea(hiddenInk)) / float64(s.InkPx)

	ring := haloRing(g, w, h)
	if ring == nil {
		return s
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
		s.Halo = float64(ringInk.IntersectArea(hiddenInk)) / float64(n)
	}
	return s
}

// haloRing is the ring just inside the group's bounds, or nil when the group is too small to
// have one.
func haloRing(g truth.Group, w, h int) *truth.Mask {
	x0, y0, x1, y1 := g.Bounds.Bounds()
	band := int(float64(min(x1-x0, y1-y0)) * haloBandFraction)
	if band < 1 {
		return nil
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
	return ring
}

// ContrastScore is the luma separation of a plate's rectangle in a rendered image. It is a
// property of the background and the plate fill, not a verdict on the replacement text: it reads
// the same with no text drawn and with any font size (see ReplacementReadability for the glyphs).
type ContrastScore struct {
	MinLuma  float64 `json:"minLumaSeparation"`
	MeanLuma float64 `json:"meanLumaSeparation"`
	Plates   int     `json:"plates"`
}

// BackgroundContrast measures the luma separation inside each plate's rectangle in a rendered
// image: the dominant tone against the 5th and 95th percentile tails.
//
// It deliberately does not know where the glyphs are. A texture with no replacement text at all
// reports 140 here, and changing the plate's font from 1 px to 16 px changes nothing.
func BackgroundContrast(rendered image.Image, plates []evidence.Plate, w, h int) ContrastScore {
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
		// Background is the plate's dominant colour (median); the tails stand away from it. The
		// 5th/95th percentiles resist antialiasing at glyph edges.
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
