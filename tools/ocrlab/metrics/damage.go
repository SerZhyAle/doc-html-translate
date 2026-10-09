package metrics

import (
	"fmt"
	"image"
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// DamageScore is overlay paint where it was not permitted.
//
// Two numbers rather than one, because they mean different things to a reader.
// OutsideReplaceArea is untidiness - a plate spilling onto blank paper is ugly but harmless.
// ProtectedHit is the blocker: a bubble outline, a panel rule or a face that the overlay
// painted over. Both are reported in absolute pixels *and* as a fraction, because "0.4% of the
// panel" and "the whole balloon outline" need to be distinguishable and a percentage alone
// flattens them.
//
// The same shape carries two different measurements. RectangleIntrusion counts the plates'
// bounding rectangles; PaintedDamage counts the pixels the overlay actually changed. State is
// set by PaintedDamage only - a score written without it is geometry and was always measured.
type DamageScore struct {
	OverlayPx            int     `json:"overlayPx"`
	OutsideReplaceArea   int     `json:"outsideReplaceArea"`
	OutsideFraction      float64 `json:"outsideFraction"`
	ProtectedHit         int     `json:"protectedHit"`
	ProtectedFraction    float64 `json:"protectedFraction"`
	WorstProtectedRegion string  `json:"worstProtectedRegion,omitempty"`
	WorstProtectedPx     int     `json:"worstProtectedPx"`
	State                string  `json:"state,omitempty"`
	Reason               string  `json:"reason,omitempty"`
}

// IsMeasured reports whether the counts cover the whole observed matrix. An unmeasured score may
// still carry the pixels that were proven damaged, which are a floor and never a clean result.
func (d DamageScore) IsMeasured() bool { return d.State != StateUnmeasured }

// RectangleIntrusion measures one viewport's plate rectangles against the annotation's permitted
// and protected areas. It is geometry: a plate that paints only its glyphs still "intrudes" on
// everything inside its box. PaintedDamage is the measurement of what was actually painted.
func RectangleIntrusion(plates []evidence.Plate, a *truth.Annotation, w, h int) DamageScore {
	return damageOf(PlateMask(plates, w, h), a, w, h)
}

// PaintedDamage measures the pixels the overlay actually changed. A pixel inside a plate counts
// as painted when the text-hidden render differs from the source there by more than codec noise;
// the plates stay painted in that capture and only the replacement glyphs are hidden, so the
// result covers fills, reconstructions and masks alike. Plates bound where paint may be
// attributed: a difference outside every plate is the capture's, not the overlay's.
//
// Without a source or a text-hidden capture there is nothing to count, and the answer is
// unmeasured rather than clean.
func PaintedDamage(source, hidden image.Image, plates []evidence.Plate, a *truth.Annotation, w, h int) DamageScore {
	unmeasured := func(reason string) DamageScore { return DamageScore{State: StateUnmeasured, Reason: reason} }
	if source == nil || hidden == nil {
		return unmeasured("source or text-hidden capture unavailable")
	}
	if why := sizeMismatch("source", source, w, h); why != "" {
		return unmeasured(why)
	}
	if why := sizeMismatch("text-hidden capture", hidden, w, h); why != "" {
		return unmeasured(why)
	}
	rects := PlateMask(plates, w, h)
	painted := truth.NewMask(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if rects.At(x, y) && pixelsDiffer(source, hidden, x, y) {
				painted.Set(x, y)
			}
		}
	}
	d := damageOf(painted, a, w, h)
	d.State = StateMeasured
	return d
}

// damageOf scores a set of overlay pixels against the annotation.
func damageOf(overlay *truth.Mask, a *truth.Annotation, w, h int) DamageScore {
	var s DamageScore
	s.OverlayPx = overlay.Area()
	if s.OverlayPx == 0 {
		return s
	}

	permitted := truth.NewMask(w, h)
	for _, g := range a.Groups {
		area := g.ReplaceArea
		if area.Empty() {
			area = g.Bounds
		}
		if !area.Empty() {
			permitted.Or(area.Rasterize(w, h))
		}
	}
	s.OutsideReplaceArea = overlay.SubtractArea(permitted)
	s.OutsideFraction = float64(s.OutsideReplaceArea) / float64(s.OverlayPx)

	protectedAll := truth.RasterizeAll(a.Protected, w, h)
	s.ProtectedHit = overlay.IntersectArea(protectedAll)
	if area := protectedAll.Area(); area > 0 {
		s.ProtectedFraction = float64(s.ProtectedHit) / float64(area)
	}

	// Name the worst offender: "damage happened" is not actionable, "the left balloon's outline
	// lost 340 px" is.
	regions := append([]truth.Region(nil), a.Protected...)
	sort.SliceStable(regions, func(i, j int) bool { return regions[i].ID < regions[j].ID })
	for i, r := range regions {
		hit := overlay.IntersectArea(r.Rasterize(w, h))
		if hit > s.WorstProtectedPx {
			s.WorstProtectedPx = hit
			s.WorstProtectedRegion = r.ID
			if s.WorstProtectedRegion == "" {
				s.WorstProtectedRegion = fmt.Sprintf("protected[%d]", i)
			}
		}
	}
	return s
}
