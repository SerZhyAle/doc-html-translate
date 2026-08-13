package metrics

import (
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// PlacementScore is whether each plate sits over the text it replaces, and whether it stays
// there.
//
// Drift is the one that needs the multi-viewport run to exist at all. A plate positioned in
// percent of the image looks correct at the viewport it was checked at and can still slide at
// another, because the container, the font and the re-fit all respond to width. Measuring the
// same plate at three viewports and reporting the worst movement is the only way that shows up
// before a reader finds it.
type PlacementScore struct {
	MeanIoU        float64 `json:"meanIou"`
	MedianIoU      float64 `json:"medianIou"`
	WorstIoU       float64 `json:"worstIou"`
	WorstIoUGroup  string  `json:"worstIouGroup,omitempty"`
	MeanEdgeError  float64 `json:"meanEdgeError"`
	WorstEdgeError float64 `json:"worstEdgeError"`
	Drift          float64 `json:"drift"`
	DriftGroup     string  `json:"driftGroup,omitempty"`
	Compared       int     `json:"compared"`
}

// Placement scores one viewport's matched pairs.
func Placement(matches []Match) PlacementScore {
	s := PlacementScore{WorstIoU: 1}
	if len(matches) == 0 {
		s.WorstIoU = 0
		return s
	}
	ious := make([]float64, 0, len(matches))
	edges := make([]float64, 0, len(matches))
	for _, m := range matches {
		ious = append(ious, m.IoU)
		if m.IoU < s.WorstIoU {
			s.WorstIoU, s.WorstIoUGroup = m.IoU, m.Group.ID
		}
		e := Edges(m.Plate.Rect.Region(), m.Group.Bounds).Worst
		edges = append(edges, e)
		if e > s.WorstEdgeError {
			s.WorstEdgeError = e
		}
	}
	s.MeanIoU, s.MeanEdgeError = mean(ious), mean(edges)
	sort.Float64s(ious)
	s.MedianIoU = ious[len(ious)/2]
	s.Compared = len(matches)
	return s
}

// Drift is the largest movement of any one group's plate across viewports, normalized by the
// group's own size. Takes the per-viewport match sets so it compares like with like: the same
// group, the same stress case, a different browser geometry.
func Drift(byViewport map[string][]Match, w, h int) (worst float64, worstGroup string) {
	type pos struct{ x, y float64 }
	seen := map[string][]pos{}
	sizes := map[string][2]float64{}

	for _, matches := range byViewport {
		for _, m := range matches {
			gx0, gy0, gx1, gy1 := m.Group.Bounds.Bounds()
			gw, gh := float64(gx1-gx0), float64(gy1-gy0)
			if gw <= 0 {
				gw = 1
			}
			if gh <= 0 {
				gh = 1
			}
			sizes[m.Group.ID] = [2]float64{gw, gh}
			seen[m.Group.ID] = append(seen[m.Group.ID], pos{
				x: float64(m.Plate.Rect.X0-gx0) / gw,
				y: float64(m.Plate.Rect.Y0-gy0) / gh,
			})
		}
	}

	// Deterministic iteration: a report that names a different group each run is not evidence.
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		ps := seen[id]
		for i := 0; i < len(ps); i++ {
			for j := i + 1; j < len(ps); j++ {
				d := max(abs(ps[i].x-ps[j].x), abs(ps[i].y-ps[j].y))
				if d > worst {
					worst, worstGroup = d, id
				}
			}
		}
	}
	return worst, worstGroup
}

// MatchesByViewport runs the matcher once per viewport for a single stress case, which is what
// Drift needs and what the report shows per row.
func MatchesByViewport(sc *evidence.Scene, groups []truth.Group, viewports []evidence.Viewport, stressCase string, w, h int) map[string][]Match {
	out := map[string][]Match{}
	for _, v := range viewports {
		plates := sc.PlatesFor(v.Name, stressCase)
		if len(plates) == 0 {
			continue
		}
		m, _, _ := MatchPlates(plates, groups, w, h)
		out[v.Name] = m
	}
	return out
}
