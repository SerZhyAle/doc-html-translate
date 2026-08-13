// Package metrics measures. It does not judge.
//
// No threshold, bound or verdict lives here, and that separation is deliberate: Phase 06 sets
// acceptance numbers from a recorded baseline, and if those numbers lived next to the
// measurement then moving a bound and moving a measure would look the same in a diff. Every
// function takes ground truth and recorded evidence and returns numbers; deciding whether a
// number is acceptable is somebody else's job.
//
// Rasterization comes from package truth (see truth/region.go for why it lives there).
package metrics

import (
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// IoU is intersection over union of two regions, rasterized at image resolution. 0 when both
// are empty, so a degenerate pair never divides by zero.
func IoU(a, b truth.Region, w, h int) float64 {
	ma, mb := a.Rasterize(w, h), b.Rasterize(w, h)
	inter := ma.IntersectArea(mb)
	union := ma.Area() + mb.Area() - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// Contained is the fraction of a that lies inside b. Asymmetric on purpose: "how much of the
// text did the plate cover" and "how much of the plate was over the text" are different
// questions and several metrics need one without the other.
func Contained(a, b truth.Region, w, h int) float64 {
	ma := a.Rasterize(w, h)
	area := ma.Area()
	if area == 0 {
		return 0
	}
	return float64(ma.IntersectArea(b.Rasterize(w, h))) / float64(area)
}

// EdgeError is how far each edge of got sits from want, normalized by want's own size, so the
// number is comparable between a caption and a full page. Returned as the four signed errors
// plus the worst absolute one.
type EdgeError struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Worst  float64 `json:"worst"`
}

// Edges computes the normalized per-edge error of got against want.
func Edges(got, want truth.Region) EdgeError {
	gx0, gy0, gx1, gy1 := got.Bounds()
	wx0, wy0, wx1, wy1 := want.Bounds()
	ww, wh := wx1-wx0, wy1-wy0
	if ww <= 0 {
		ww = 1
	}
	if wh <= 0 {
		wh = 1
	}
	e := EdgeError{
		Left:   float64(gx0-wx0) / float64(ww),
		Top:    float64(gy0-wy0) / float64(wh),
		Right:  float64(gx1-wx1) / float64(ww),
		Bottom: float64(gy1-wy1) / float64(wh),
	}
	for _, v := range []float64{e.Left, e.Top, e.Right, e.Bottom} {
		if abs(v) > e.Worst {
			e.Worst = abs(v)
		}
	}
	return e
}

// PlateMask unions a set of plates into one coverage mask - what the overlay actually paints.
func PlateMask(plates []evidence.Plate, w, h int) *truth.Mask {
	m := truth.NewMask(w, h)
	for _, p := range plates {
		m.Or(p.Rect.Region().Rasterize(w, h))
	}
	return m
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// mean of a slice, 0 for empty.
func mean(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	var s float64
	for _, v := range vs {
		s += v
	}
	return s / float64(len(vs))
}
