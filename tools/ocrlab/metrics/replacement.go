package metrics

import (
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// crossGroupCoverage is how much of an *unrelated* group a plate must cover before it counts as
// having crossed into it. Lower than the merge threshold: a plate does not have to swallow a
// neighbouring balloon to ruin it, only to sit visibly on top of a fifth of it.
const crossGroupCoverage = 0.2

// ReplacementScore is what happens to the layout once real text is in the plates.
//
// Clipped is read from the DOM, not computed. The shipped re-fit shrinks a plate's font and then
// lets the box grow rather than clip, so whether a translation fitted is a fact about the
// rendered box that only the browser knows; recomputing it here from font metrics would measure
// a different program.
type ReplacementScore struct {
	Plates            int `json:"plates"`
	Clipped           int `json:"clipped"`
	CrossGroupOverlap int `json:"crossGroupOverlap"`
	PlateOverlap      int `json:"plateOverlap"`
	OutOfBounds       int `json:"outOfBounds"`
	OutOfBoundsPx     int `json:"outOfBoundsPx"`
}

// Worst returns true when any hard-failure counter is non-zero. Not a threshold: these four are
// the strategic spec's own zero-tolerance list, and a caller still decides what to do about it.
func (s ReplacementScore) Worst() bool {
	return s.Clipped > 0 || s.CrossGroupOverlap > 0 || s.OutOfBounds > 0
}

// Replacement scores one (viewport, stress case) plate set.
func Replacement(plates []evidence.Plate, groups []truth.Group, matches []Match, w, h int) ReplacementScore {
	s := ReplacementScore{Plates: len(plates)}

	groupMasks := make([]*truth.Mask, len(groups))
	groupArea := make([]int, len(groups))
	for gi, g := range groups {
		groupMasks[gi] = g.Bounds.Rasterize(w, h)
		groupArea[gi] = groupMasks[gi].Area()
	}

	// Which group each plate belongs to, so "crossing into another group" excludes its own.
	own := make([]string, len(plates))
	for _, m := range matches {
		for i, p := range plates {
			if p.Rect == m.Plate.Rect && p.Text == m.Plate.Text {
				own[i] = m.Group.ID
				break
			}
		}
	}

	for i, p := range plates {
		if p.Clipped() {
			s.Clipped++
		}
		if p.Rect.X0 < 0 || p.Rect.Y0 < 0 || p.Rect.X1 > w || p.Rect.Y1 > h {
			s.OutOfBounds++
			s.OutOfBoundsPx += outsidePixels(p, w, h)
		}
		pm := p.Rect.Region().Rasterize(w, h)
		for gi, g := range groups {
			if g.ID == own[i] || groupArea[gi] == 0 {
				continue
			}
			if float64(pm.IntersectArea(groupMasks[gi]))/float64(groupArea[gi]) >= crossGroupCoverage {
				s.CrossGroupOverlap++
			}
		}
		for j := i + 1; j < len(plates); j++ {
			if pm.IntersectArea(plates[j].Rect.Region().Rasterize(w, h)) > 0 {
				s.PlateOverlap++
			}
		}
	}
	return s
}

// outsidePixels is how much of the plate lies off the image.
func outsidePixels(p evidence.Plate, w, h int) int {
	total := p.Rect.Width() * p.Rect.Height()
	if total <= 0 {
		return 0
	}
	cx0, cy0 := max(p.Rect.X0, 0), max(p.Rect.Y0, 0)
	cx1, cy1 := min(p.Rect.X1, w), min(p.Rect.Y1, h)
	inside := 0
	if cx1 > cx0 && cy1 > cy0 {
		inside = (cx1 - cx0) * (cy1 - cy0)
	}
	return total - inside
}

// StressBreakdown is the replacement score per stress case, which is how the report shows that
// a page survives its own language and fails a longer one.
type StressBreakdown map[string]ReplacementScore

// ReplacementByStress scores every stress case present in the scene at one viewport.
func ReplacementByStress(sc *evidence.Scene, groups []truth.Group, viewport string, w, h int) StressBreakdown {
	out := StressBreakdown{}
	cases := sc.StressCases()
	sort.Strings(cases)
	for _, name := range cases {
		plates := sc.PlatesFor(viewport, name)
		if len(plates) == 0 {
			continue
		}
		matches, _, _ := MatchPlates(plates, groups, w, h)
		out[name] = Replacement(plates, groups, matches, w, h)
	}
	return out
}
