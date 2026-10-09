package metrics

import (
	"fmt"
	"image"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/truth"
)

// Support conditions of the single-tone ink estimate. They describe where the estimate is
// defined, not what a good result is: outside them the answer is "unmeasured", never a number.
const (
	// maxBackgroundSpread is how far the putative background tone may vary - its 10th to 90th
	// percentile range among the pixels within one ink step of the median. A background that
	// varies by half an ink step or more cannot be told apart from lettering by luma alone.
	// Derived from the ink step rather than tuned.
	maxBackgroundSpread = inkLumaDelta / 2
	// maxInkShare is the largest share of a region that may stand out from its median. The median
	// is the background only while the background clearly outnumbers the ink; a checker texture
	// splits the region about evenly and the "ink" is then half the background. Measured on the
	// generated scenes (TestSupportConditionSeparatesGeneratedTextFromTexture): line boxes of plain
	// lettering carry 14-33% ink, the one line box on a halftone screen 48%. A tightly cropped real
	// line can exceed this and is then reported unmeasured, which is the safe side.
	maxInkShare = 0.4
	// unchangedChannelTolerance is how far a colour channel may move between two captures of
	// the same pixel and still be the same pixel. A quarter of the ink step is paper grain and
	// codec noise, not paint.
	unchangedChannelTolerance = inkLumaDelta / 4
)

// unsupportedBackground returns why the single-tone ink estimate is undefined for a region with
// these lumas, or "" when it is defined.
func unsupportedBackground(lumas []int) string {
	if len(lumas) == 0 {
		return ""
	}
	sorted := append([]int(nil), lumas...)
	sort.Ints(sorted)
	med := sorted[len(sorted)/2]
	var band []int
	ink := 0
	for _, l := range sorted {
		if absInt(l-med) <= inkLumaDelta {
			band = append(band, l)
		} else {
			ink++
		}
	}
	lo, hi := band[len(band)/10], band[min(len(band)-1, len(band)*9/10)]
	if spread := hi - lo; spread > maxBackgroundSpread {
		return fmt.Sprintf("the background varies by %d luma, so it cannot be told from lettering (gradient or texture)", spread)
	}
	if share := float64(ink) / float64(len(sorted)); share > maxInkShare {
		return fmt.Sprintf("%.0f%% of the region stands out from its median, so the median is not a single background tone (texture)", share*100)
	}
	return ""
}

// pixelsDiffer reports whether two captures disagree about a pixel by more than codec noise.
func pixelsDiffer(a, b image.Image, x, y int) bool {
	ar, ag, ab, _ := a.At(x, y).RGBA()
	br, bg, bb, _ := b.At(x, y).RGBA()
	return absInt(int(ar>>8)-int(br>>8)) > unchangedChannelTolerance ||
		absInt(int(ag>>8)-int(bg>>8)) > unchangedChannelTolerance ||
		absInt(int(ab>>8)-int(bb>>8)) > unchangedChannelTolerance
}

// sizeMismatch names a capture whose size is not the annotation's, or returns "".
func sizeMismatch(label string, img image.Image, w, h int) string {
	if b := img.Bounds(); b.Dx() != w || b.Dy() != h {
		return fmt.Sprintf("%s is %dx%d, expected %dx%d", label, b.Dx(), b.Dy(), w, h)
	}
	return ""
}

// ResidualKnownMask measures how much old lettering is still showing against an independently
// known mask of the source lettering - pixels some other source than the image itself says are
// lettering (a synthetic scene's own drawing, a declared annotation mask).
//
// hidden is the text-hidden capture: the plates stay painted and every replacement glyph is
// transparent, so new text aligned on the old coordinates cannot count as old ink. A lettering
// pixel is still showing when the capture still holds the source's colour at it; any paint over
// it, or removal of the lettering, changes the pixel. Background structure plays no part, which
// is what makes this valid on gradients and textures where ResidualInk is not.
//
// Limit: a translucent plate that moves a pixel by more than codec noise counts as concealing
// even if the old stroke is faintly visible. The mask lists lettering core pixels; antialiased
// fringes that differ from the background by less than codec noise are not distinguishable and
// must not be in the mask.
func ResidualKnownMask(source, hidden image.Image, lettering *truth.Mask, g truth.Group, w, h int) ResidualScore {
	switch {
	case source == nil || hidden == nil:
		return unmeasuredResidual("source or text-hidden capture unavailable")
	case lettering == nil:
		return unmeasuredResidual("no independently known lettering mask")
	case lettering.W != w || lettering.H != h:
		return unmeasuredResidual(fmt.Sprintf("lettering mask is %dx%d, expected %dx%d", lettering.W, lettering.H, w, h))
	}
	if why := sizeMismatch("source", source, w, h); why != "" {
		return unmeasuredResidual(why)
	}
	if why := sizeMismatch("text-hidden capture", hidden, w, h); why != "" {
		return unmeasuredResidual(why)
	}

	regions := groupRegions(g)
	area := truth.RasterizeAll(regions, w, h)
	x0, y0, x1, y1 := regionsBounds(regions, w, h)
	known := truth.NewMask(w, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if area.At(x, y) && lettering.At(x, y) {
				known.Set(x, y)
			}
		}
	}
	s := measuredResidual(BasisKnownMask)
	s.InkPx = known.Area()
	if s.InkPx == 0 {
		return unmeasuredResidual("the lettering mask has no pixels inside this group")
	}
	still := truth.NewMask(w, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if known.At(x, y) && !pixelsDiffer(source, hidden, x, y) {
				still.Set(x, y)
			}
		}
	}
	s.Residual = float64(still.Area()) / float64(s.InkPx)

	if ring := haloRing(g, w, h); ring != nil {
		if n := known.IntersectArea(ring); n > 0 {
			s.Halo = float64(still.IntersectArea(ring)) / float64(n)
		}
	}
	return s
}

// ResidualFor measures one group: against the known lettering mask when there is one, otherwise
// with the single-tone estimate where that is defined.
func ResidualFor(source, hidden image.Image, lettering *truth.Mask, g truth.Group, w, h int) ResidualScore {
	if lettering != nil {
		return ResidualKnownMask(source, hidden, lettering, g, w, h)
	}
	return ResidualInk(source, hidden, g, w, h)
}

// ResidualAcross folds every group of a scene: the worst residual and halo, summed sample size.
// One group that cannot be measured makes the scene unmeasured - the worst of the others would
// otherwise stand in for it and read as the scene's concealment.
func ResidualAcross(source, hidden image.Image, lettering *truth.Mask, groups []truth.Group, w, h int) ResidualScore {
	if source == nil || hidden == nil {
		return unmeasuredResidual("source or text-hidden capture unavailable")
	}
	out := measuredResidual("")
	var unmeasured []string
	for _, g := range groups {
		r := ResidualFor(source, hidden, lettering, g, w, h)
		if !r.IsMeasured() {
			unmeasured = append(unmeasured, g.ID+": "+r.Reason)
			continue
		}
		out.InkPx += r.InkPx
		out.Residual = max(out.Residual, r.Residual)
		out.Halo = max(out.Halo, r.Halo)
		switch {
		case out.Basis == "":
			out.Basis = r.Basis
		case out.Basis != r.Basis:
			out.Basis = BasisMixed
		}
	}
	if len(unmeasured) > 0 {
		shown := unmeasured[:min(len(unmeasured), 3)]
		reason := fmt.Sprintf("%d of %d group(s) unmeasured - %s", len(unmeasured), len(groups), strings.Join(shown, "; "))
		return unmeasuredResidual(reason)
	}
	return out
}

// regionsBounds is the union bounding box of some regions, clamped to the image.
func regionsBounds(regions []truth.Region, w, h int) (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = w, h, 0, 0
	for _, r := range regions {
		a, b, c, d := r.Bounds()
		x0, y0, x1, y1 = min(x0, a), min(y0, b), max(x1, c), max(y1, d)
	}
	return max(x0, 0), max(y0, 0), min(x1, w), min(y1, h)
}
