package metrics

import (
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
type DamageScore struct {
	OverlayPx            int     `json:"overlayPx"`
	OutsideReplaceArea   int     `json:"outsideReplaceArea"`
	OutsideFraction      float64 `json:"outsideFraction"`
	ProtectedHit         int     `json:"protectedHit"`
	ProtectedFraction    float64 `json:"protectedFraction"`
	WorstProtectedRegion string  `json:"worstProtectedRegion,omitempty"`
	WorstProtectedPx     int     `json:"worstProtectedPx"`
}

// Damage measures one viewport's plates against the annotation's permitted and protected areas.
func Damage(plates []evidence.Plate, a *truth.Annotation, w, h int) DamageScore {
	var s DamageScore
	overlay := PlateMask(plates, w, h)
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
				s.WorstProtectedRegion = "protected[" + itoa(i) + "]"
			}
		}
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
