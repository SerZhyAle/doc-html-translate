package iconart

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/vector"
)

// canvas paints filled shapes onto an RGBA image, antialiased, in pixel coordinates.
type canvas struct {
	img *image.RGBA
	z   *vector.Rasterizer
}

func newCanvas(w, h int) *canvas {
	return &canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), z: vector.NewRasterizer(w, h)}
}

// transform maps design units to pixels: p*scale + offset.
type transform struct{ scale, dx, dy float64 }

func (t transform) apply(p point) (float32, float32) {
	return float32(p.x*t.scale + t.dx), float32(p.y*t.scale + t.dy)
}

// fill paints the segments in c. Every subpath is filled by the nonzero rule, so a cut-out
// must wind against its outline (the Material glyphs do); overlapping parts that wind the
// same way simply stay covered.
func (cv *canvas) fill(segs []segment, t transform, c color.Color) {
	cv.paint(segs, t, image.NewUniform(c), draw.Over)
}

// erase clears the segments' coverage back to transparent - a real cut-out, so the one-tone
// forms show whatever the platform draws behind them.
// Neither the rasterizer's Src op nor image/draw's (with a uniform transparent source) keeps
// dst outside the mask, so the coverage is taken as an alpha mask and applied here as
// destination-out: RGBA is premultiplied, so every channel scales by 1 - coverage.
func (cv *canvas) erase(segs []segment, t transform) {
	mask := image.NewAlpha(cv.img.Bounds())
	cv.trace(segs, t)
	cv.z.DrawOp = draw.Over
	cv.z.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	for i, m := range mask.Pix {
		keep := 255 - uint32(m)
		for c := 0; c < 4; c++ {
			p := &cv.img.Pix[4*i+c]
			*p = uint8((uint32(*p)*keep + 127) / 255)
		}
	}
}

func (cv *canvas) paint(segs []segment, t transform, src image.Image, op draw.Op) {
	cv.trace(segs, t)
	cv.z.DrawOp = op
	cv.z.Draw(cv.img, cv.img.Bounds(), src, image.Point{})
}

// trace resets the rasterizer and feeds it the segments.
func (cv *canvas) trace(segs []segment, t transform) {
	b := cv.img.Bounds()
	cv.z.Reset(b.Dx(), b.Dy())
	open := false
	for _, s := range segs {
		switch s.op {
		case 'M':
			if open {
				cv.z.ClosePath()
			}
			x, y := t.apply(s.pts[0])
			cv.z.MoveTo(x, y)
			open = true
		case 'L':
			x, y := t.apply(s.pts[0])
			cv.z.LineTo(x, y)
		case 'C':
			ax, ay := t.apply(s.pts[0])
			bx, by := t.apply(s.pts[1])
			cx, cy := t.apply(s.pts[2])
			cv.z.CubeTo(ax, ay, bx, by, cx, cy)
		case 'Z':
			if open {
				cv.z.ClosePath()
				open = false
			}
		}
	}
	if open {
		cv.z.ClosePath()
	}
}

// polygon is a closed outline from points.
func polygon(pts ...point) []segment {
	out := make([]segment, 0, len(pts)+1)
	for i, p := range pts {
		op := byte('L')
		if i == 0 {
			op = 'M'
		}
		out = append(out, segment{op: op, pts: [3]point{p}})
	}
	return append(out, segment{op: 'Z'})
}

// roundRect is a rectangle with circular corners of radius r, clockwise.
func roundRect(x0, y0, x1, y1, r float64) []segment {
	const k = 0.5522847498 // cubic approximation of a quarter circle
	c := r * k
	return []segment{
		{op: 'M', pts: [3]point{{x0 + r, y0}}},
		{op: 'L', pts: [3]point{{x1 - r, y0}}},
		{op: 'C', pts: [3]point{{x1 - r + c, y0}, {x1, y0 + r - c}, {x1, y0 + r}}},
		{op: 'L', pts: [3]point{{x1, y1 - r}}},
		{op: 'C', pts: [3]point{{x1, y1 - r + c}, {x1 - r + c, y1}, {x1 - r, y1}}},
		{op: 'L', pts: [3]point{{x0 + r, y1}}},
		{op: 'C', pts: [3]point{{x0 + r - c, y1}, {x0, y1 - r + c}, {x0, y1 - r}}},
		{op: 'L', pts: [3]point{{x0, y0 + r}}},
		{op: 'C', pts: [3]point{{x0, y0 + r - c}, {x0 + r - c, y0}, {x0 + r, y0}}},
		{op: 'Z'},
	}
}

// stroke outlines an open polyline of width w with mitred joins and butt ends, as one
// clockwise polygon - so a chevron is a single shape with no overlap to double-cover.
func stroke(w float64, pts ...point) []segment {
	h := w / 2
	n := len(pts)
	left := make([]point, n)
	right := make([]point, n)
	normal := func(a, b point) point {
		dx, dy := b.x-a.x, b.y-a.y
		l := math.Hypot(dx, dy)
		return point{-dy / l, dx / l}
	}
	for i := range pts {
		var nv point
		switch {
		case i == 0:
			nv = normal(pts[0], pts[1])
		case i == n-1:
			nv = normal(pts[n-2], pts[n-1])
		default:
			a, b := normal(pts[i-1], pts[i]), normal(pts[i], pts[i+1])
			m := point{a.x + b.x, a.y + b.y}
			ml := math.Hypot(m.x, m.y)
			m = point{m.x / ml, m.y / ml}
			// The miter length that keeps both edges at distance h.
			cos := m.x*a.x + m.y*a.y
			nv = point{m.x / cos, m.y / cos}
		}
		left[i] = point{pts[i].x + nv.x*h, pts[i].y + nv.y*h}
		right[i] = point{pts[i].x - nv.x*h, pts[i].y - nv.y*h}
	}
	outline := append([]point{}, left...)
	for i := n - 1; i >= 0; i-- {
		outline = append(outline, right[i])
	}
	if signedArea(outline) < 0 {
		for i, j := 0, len(outline)-1; i < j; i, j = i+1, j-1 {
			outline[i], outline[j] = outline[j], outline[i]
		}
	}
	return polygon(outline...)
}

func signedArea(pts []point) float64 {
	var a float64
	for i := range pts {
		j := (i + 1) % len(pts)
		a += pts[i].x*pts[j].y - pts[j].x*pts[i].y
	}
	return a / 2
}
