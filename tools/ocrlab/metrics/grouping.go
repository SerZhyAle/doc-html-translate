package metrics

import (
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// coverageForMerge is how much of a group a plate must cover before the plate is said to have
// swallowed it. Well below 1 because a plate that covers most of a second balloon has already
// committed the defect; well above 0 so a few pixels of overlap at a shared edge is not a merge.
const coverageForMerge = 0.5

// Match is one plate paired with the group it was judged to represent.
type Match struct {
	Plate   evidence.Plate
	Group   truth.Group
	Overlap int     // intersecting pixels
	IoU     float64 // for the placement metric
}

// GroupingScore is whether the overlay respected the page's reading structure.
//
// Merges and Splits are the two failure modes that matter to a reader and that an aggregate
// recall figure cannot see: recall is happy when one enormous plate covers both balloons, and
// happy again when a caption is torn into four plates.
type GroupingScore struct {
	Matched         int `json:"matched"`
	Merges          int `json:"merges"`
	Splits          int `json:"splits"`
	UnmatchedGroups int `json:"unmatchedGroups"`
	UnmatchedPlates int `json:"unmatchedPlates"`
	OrderInversions int `json:"orderInversions"`
}

// MatchPlates pairs plates with groups by greatest overlap, one to one.
//
// Greedy by descending overlap rather than an optimal assignment: it is deterministic, it is
// obvious in a report ("this plate went to that group because it covered it most"), and the
// cases where an optimal matching would differ are cases where the overlay is already wrong
// enough that the metric's exact attribution is not the interesting part.
func MatchPlates(plates []evidence.Plate, groups []truth.Group, w, h int) (matches []Match, freePlates []int, freeGroups []int) {
	type cand struct {
		pi, gi  int
		overlap int
		iou     float64
	}
	var cands []cand
	groupMasks := make([]*truth.Mask, len(groups))
	for gi, g := range groups {
		groupMasks[gi] = g.Bounds.Rasterize(w, h)
	}
	for pi, p := range plates {
		pm := p.Rect.Region().Rasterize(w, h)
		for gi := range groups {
			ov := pm.IntersectArea(groupMasks[gi])
			if ov == 0 {
				continue
			}
			union := pm.Area() + groupMasks[gi].Area() - ov
			iou := 0.0
			if union > 0 {
				iou = float64(ov) / float64(union)
			}
			cands = append(cands, cand{pi, gi, ov, iou})
		}
	}
	// Deterministic order: biggest overlap first, ties broken by index so two runs agree.
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].overlap != cands[j].overlap {
			return cands[i].overlap > cands[j].overlap
		}
		if cands[i].pi != cands[j].pi {
			return cands[i].pi < cands[j].pi
		}
		return cands[i].gi < cands[j].gi
	})

	usedP := make([]bool, len(plates))
	usedG := make([]bool, len(groups))
	for _, c := range cands {
		if usedP[c.pi] || usedG[c.gi] {
			continue
		}
		usedP[c.pi], usedG[c.gi] = true, true
		matches = append(matches, Match{Plate: plates[c.pi], Group: groups[c.gi], Overlap: c.overlap, IoU: c.iou})
	}
	for i, u := range usedP {
		if !u {
			freePlates = append(freePlates, i)
		}
	}
	for i, u := range usedG {
		if !u {
			freeGroups = append(freeGroups, i)
		}
	}
	return matches, freePlates, freeGroups
}

// Grouping counts merges, splits and reading-order inversions.
func Grouping(plates []evidence.Plate, groups []truth.Group, w, h int) (GroupingScore, []Match) {
	matches, freePlates, freeGroups := MatchPlates(plates, groups, w, h)
	s := GroupingScore{
		Matched:         len(matches),
		UnmatchedPlates: len(freePlates),
		UnmatchedGroups: len(freeGroups),
	}

	groupMasks := make([]*truth.Mask, len(groups))
	groupArea := make([]int, len(groups))
	for gi, g := range groups {
		groupMasks[gi] = g.Bounds.Rasterize(w, h)
		groupArea[gi] = groupMasks[gi].Area()
	}

	// A merge: one plate substantially covering two or more groups.
	// A split: one group substantially covered by two or more plates.
	coveredBy := make([]int, len(groups))
	for _, p := range plates {
		pm := p.Rect.Region().Rasterize(w, h)
		covers := 0
		for gi := range groups {
			if groupArea[gi] == 0 {
				continue
			}
			if float64(pm.IntersectArea(groupMasks[gi]))/float64(groupArea[gi]) >= coverageForMerge {
				covers++
				coveredBy[gi]++
			}
		}
		if covers >= 2 {
			s.Merges++
		}
	}
	for _, n := range coveredBy {
		if n >= 2 {
			s.Splits++
		}
	}

	s.OrderInversions = orderInversions(matches)
	return s, matches
}

// orderInversions counts pairs whose plates read in a different order than the annotation says
// they should. Plate order is top-to-bottom then left-to-right, the order a reader scans a
// page and the order the DOM will hand to a translator.
func orderInversions(matches []Match) int {
	ordered := append([]Match(nil), matches...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Plate.Rect.Y0 != ordered[j].Plate.Rect.Y0 {
			return ordered[i].Plate.Rect.Y0 < ordered[j].Plate.Rect.Y0
		}
		return ordered[i].Plate.Rect.X0 < ordered[j].Plate.Rect.X0
	})
	n := 0
	for i := 0; i < len(ordered); i++ {
		for j := i + 1; j < len(ordered); j++ {
			if ordered[i].Group.ReadingOrder > ordered[j].Group.ReadingOrder {
				n++
			}
		}
	}
	return n
}
