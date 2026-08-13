package truth

import "fmt"

// RegionKind distinguishes the two shapes an annotation may draw. A box is two points and
// covers the common case cheaply; a polygon is what a speech balloon, a tail or a torn scan
// edge actually needs.
type RegionKind string

const (
	RegionBox     RegionKind = "box"
	RegionPolygon RegionKind = "polygon"
)

// Region is an area of the source image in natural image pixels. Never in rendered pixels,
// never in percent: the strategic spec requires geometry that survives responsive scaling, and
// the only coordinate space that does is the image's own.
type Region struct {
	ID     string     `json:"id,omitempty"`
	Kind   RegionKind `json:"kind"`
	Points [][2]int   `json:"points"`
}

// Box is a convenience constructor for the common rectangular region.
func Box(id string, x0, y0, x1, y1 int) Region {
	return Region{ID: id, Kind: RegionBox, Points: [][2]int{{x0, y0}, {x1, y1}}}
}

// Empty reports whether the region has no points at all (an unset field, not a degenerate one).
func (r Region) Empty() bool { return len(r.Points) == 0 }

// Valid returns why the region is malformed, or nil.
func (r Region) Valid() error {
	switch r.Kind {
	case RegionBox:
		if len(r.Points) != 2 {
			return fmt.Errorf("box needs exactly 2 points, has %d", len(r.Points))
		}
		if r.Points[0][0] >= r.Points[1][0] || r.Points[0][1] >= r.Points[1][1] {
			return fmt.Errorf("box is empty or inverted: %v", r.Points)
		}
	case RegionPolygon:
		if len(r.Points) < 3 {
			return fmt.Errorf("polygon needs at least 3 points, has %d", len(r.Points))
		}
	default:
		return fmt.Errorf("kind %q is neither box nor polygon", r.Kind)
	}
	return nil
}

// Bounds is the axis-aligned bounding box, for both kinds.
func (r Region) Bounds() (x0, y0, x1, y1 int) {
	if len(r.Points) == 0 {
		return 0, 0, 0, 0
	}
	x0, y0 = r.Points[0][0], r.Points[0][1]
	x1, y1 = x0, y0
	for _, p := range r.Points[1:] {
		x0 = min(x0, p[0])
		y0 = min(y0, p[1])
		x1 = max(x1, p[0])
		y1 = max(y1, p[1])
	}
	return x0, y0, x1, y1
}

// Inside reports whether every point of r lies within a w x h image.
func (r Region) Inside(w, h int) bool {
	x0, y0, x1, y1 := r.Bounds()
	return x0 >= 0 && y0 >= 0 && x1 <= w && y1 <= h
}

// Mask is a per-pixel coverage bitmap at image resolution. Rasterizing is how a polygon
// annotation, a rectangular plate and a screenshot are compared in the same terms; every area
// figure downstream is a pixel count over one of these, never an approximation from bounding
// boxes.
//
// Geometry lives here rather than in the metrics package because the annotation validator needs
// exact overlap too - a replacement area that clips a protected balloon outline must be caught
// when the annotation is written, not only when a plate lands on it - and metrics already
// imports truth, so putting it the other way round would be an import cycle.
type Mask struct {
	W, H int
	bits []bool
}

// NewMask allocates an empty mask.
func NewMask(w, h int) *Mask {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Mask{W: w, H: h, bits: make([]bool, w*h)}
}

// Set marks one pixel, ignoring out-of-bounds coordinates.
func (m *Mask) Set(x, y int) {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return
	}
	m.bits[y*m.W+x] = true
}

// At reports whether a pixel is covered. Out-of-bounds reads as false.
func (m *Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return false
	}
	return m.bits[y*m.W+x]
}

// Area is the number of covered pixels.
func (m *Mask) Area() int {
	n := 0
	for _, b := range m.bits {
		if b {
			n++
		}
	}
	return n
}

// Or folds another mask of the same size into this one.
func (m *Mask) Or(o *Mask) *Mask {
	if o == nil || o.W != m.W || o.H != m.H {
		return m
	}
	for i, b := range o.bits {
		if b {
			m.bits[i] = true
		}
	}
	return m
}

// IntersectArea counts pixels covered by both masks.
func (m *Mask) IntersectArea(o *Mask) int {
	if o == nil || o.W != m.W || o.H != m.H {
		return 0
	}
	n := 0
	for i, b := range m.bits {
		if b && o.bits[i] {
			n++
		}
	}
	return n
}

// SubtractArea counts pixels covered by m but not by o.
func (m *Mask) SubtractArea(o *Mask) int {
	if o == nil || o.W != m.W || o.H != m.H {
		return m.Area()
	}
	n := 0
	for i, b := range m.bits {
		if b && !o.bits[i] {
			n++
		}
	}
	return n
}

// Rasterize renders the region into a fresh w x h mask. A box fills its rectangle; a polygon is
// filled by the even-odd rule with a scanline test at each pixel centre, which is the same rule
// SVG and canvas use, so an annotation drawn in an editor rasterizes here the way it looked
// there.
func (r Region) Rasterize(w, h int) *Mask {
	m := NewMask(w, h)
	if len(r.Points) == 0 {
		return m
	}
	x0, y0, x1, y1 := r.Bounds()
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, w), min(y1, h)

	if r.Kind == RegionBox {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				m.Set(x, y)
			}
		}
		return m
	}

	for y := y0; y < y1; y++ {
		cy := float64(y) + 0.5
		for x := x0; x < x1; x++ {
			if pointInPolygon(float64(x)+0.5, cy, r.Points) {
				m.Set(x, y)
			}
		}
	}
	return m
}

// pointInPolygon is the standard even-odd crossing test.
func pointInPolygon(px, py float64, pts [][2]int) bool {
	inside := false
	n := len(pts)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi := float64(pts[i][0]), float64(pts[i][1])
		xj, yj := float64(pts[j][0]), float64(pts[j][1])
		if (yi > py) == (yj > py) {
			continue
		}
		if px < (xj-xi)*(py-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

// RasterizeAll unions a list of regions into one mask.
func RasterizeAll(rs []Region, w, h int) *Mask {
	m := NewMask(w, h)
	for _, r := range rs {
		m.Or(r.Rasterize(w, h))
	}
	return m
}
