package metrics

import (
	"fmt"
	"image"
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
)

// Readability states. A plate set with no drawn glyph pixels is its own state: it has no
// readability to report, and its background contrast says nothing about that.
const StateNoGlyphs = "no-glyphs"

// ReadabilityScore describes the replacement glyphs that were actually drawn, found as the pixels
// that differ between the normal render and the text-hidden render inside a plate. It reports
// what the glyphs look like; it applies no limit, so a reader of the report decides what is
// readable.
type ReadabilityScore struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`

	Plates           int `json:"plates"`           // plates inspected
	PlatesWithGlyphs int `json:"platesWithGlyphs"` // plates with at least one glyph pixel
	GlyphPx          int `json:"glyphPx"`

	// MinGlyphHeightPx is the smallest plate-level text row height: per plate, the median height
	// of the contiguous pixel rows holding glyphs, so a descender or a lone accent does not move
	// it. Font metadata is not consulted - a plate that claims 16 px and draws 2 px is 2 px here.
	MinGlyphHeightPx int `json:"minGlyphHeightPx"`
	// MinSeparation and MeanSeparation are, per plate, the median luma difference between a glyph
	// pixel and the pixel it covers in the text-hidden render - the ink against the very
	// background it sits on, so a textured background cannot inflate it.
	MinSeparation  float64 `json:"minSeparation"`
	MeanSeparation float64 `json:"meanSeparation"`

	// Lines counts text rows; ClippedLines those touching the plate's own edge, where a glyph that
	// overflowed a clipping box is cut. ClippedFraction is their share.
	Lines           int     `json:"lines"`
	ClippedLines    int     `json:"clippedLines"`
	ClippedFraction float64 `json:"clippedFraction"`
}

func unmeasuredReadability(reason string) ReadabilityScore {
	return ReadabilityScore{State: StateUnmeasured, Reason: reason}
}

// plateGlyphs is what one plate's glyph pixels look like.
type plateGlyphs struct {
	px        int
	heights   []int // height in rows of each contiguous glyph row band
	clipped   int   // bands touching the plate's edge
	separated []int // |luma(normal) - luma(hidden)| per glyph pixel
}

func scanPlate(normal, hidden image.Image, r evidence.Rect, w, h int) plateGlyphs {
	x0, y0, x1, y1 := max(r.X0, 0), max(r.Y0, 0), min(r.X1, w), min(r.Y1, h)
	var out plateGlyphs
	if x1 <= x0 || y1 <= y0 {
		return out
	}
	rowPx := make([]int, y1-y0)
	rowTouchesSide := make([]bool, y1-y0)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !pixelsDiffer(normal, hidden, x, y) {
				continue
			}
			nr, ng, nb, _ := normal.At(x, y).RGBA()
			hr, hg, hb, _ := hidden.At(x, y).RGBA()
			out.px++
			rowPx[y-y0]++
			out.separated = append(out.separated, absInt(luma(nr, ng, nb)-luma(hr, hg, hb)))
			if x == x0 || x == x1-1 {
				rowTouchesSide[y-y0] = true
			}
		}
	}
	for i := 0; i < len(rowPx); {
		if rowPx[i] == 0 {
			i++
			continue
		}
		j, side := i, false
		for j < len(rowPx) && rowPx[j] > 0 {
			side = side || rowTouchesSide[j]
			j++
		}
		out.heights = append(out.heights, j-i)
		if side || i == 0 || j == len(rowPx) {
			out.clipped++
		}
		i = j
	}
	return out
}

func medianInt(vs []int) int {
	s := append([]int(nil), vs...)
	sort.Ints(s)
	return s[len(s)/2]
}

// ReplacementReadability measures the glyphs the plates drew. Glyph pixels are those that differ
// between the normal render and the text-hidden render inside a plate's rectangle, so the result
// follows the real drawn text and not the plate's box or its font metadata.
func ReplacementReadability(normal, hidden image.Image, plates []evidence.Plate, w, h int) ReadabilityScore {
	if normal == nil || hidden == nil {
		return unmeasuredReadability("normal or text-hidden capture unavailable")
	}
	if why := sizeMismatch("normal capture", normal, w, h); why != "" {
		return unmeasuredReadability(why)
	}
	if why := sizeMismatch("text-hidden capture", hidden, w, h); why != "" {
		return unmeasuredReadability(why)
	}
	s := ReadabilityScore{State: StateMeasured, Plates: len(plates)}
	var seps []float64
	for _, p := range plates {
		g := scanPlate(normal, hidden, p.Rect, w, h)
		if g.px == 0 {
			continue
		}
		s.PlatesWithGlyphs++
		s.GlyphPx += g.px
		s.Lines += len(g.heights)
		s.ClippedLines += g.clipped
		height := medianInt(g.heights)
		if s.PlatesWithGlyphs == 1 || height < s.MinGlyphHeightPx {
			s.MinGlyphHeightPx = height
		}
		sep := float64(medianInt(g.separated))
		seps = append(seps, sep)
		if s.PlatesWithGlyphs == 1 || sep < s.MinSeparation {
			s.MinSeparation = sep
		}
	}
	if s.GlyphPx == 0 {
		s.State = StateNoGlyphs
		s.Reason = fmt.Sprintf("no replacement glyph pixels inside %d plate(s)", len(plates))
		return s
	}
	s.MeanSeparation = mean(seps)
	s.ClippedFraction = float64(s.ClippedLines) / float64(s.Lines)
	return s
}
