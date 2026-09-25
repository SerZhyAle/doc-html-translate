package iconart

import (
	"image"
	"image/color"
)

// The product mark (ticket 32, decided by the owner on 2026-09-25): a white sheet with a
// folded corner and "</>" cut into it, on a navy plate. It is product artwork (ICON-SET
// rule 7) - never a vocabulary glyph and never drawn for a meaning. Two drawings of it:
// the full one from 24 px up, and a small one for 16 and 20 px that drops the slash and
// thickens the brackets so the sheet and "<>" still read at one pixel per stroke.
var (
	Navy   = color.NRGBA{0x1E, 0x3A, 0x8A, 0xFF} // the brand colour, AppxManifest BackgroundColor
	Sheet  = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	Fold   = color.NRGBA{0xA9, 0xB8, 0xE0, 0xFF} // the turned-down corner, a tint of the navy
	OneTon = color.NRGBA{0x80, 0x80, 0x80, 0xFF} // ICON-RENDER rule 9: the one tone of a surface the product cannot see
)

// markDrawing is one drawing of the mark on a square design grid.
type markDrawing struct {
	grid   float64
	plate  func(inset float64) []segment
	sheet  []segment
	fold   []segment
	code   [][]segment
	insetR float64 // plate inset, in grid units, at sizes that leave a margin
}

var smallMark = markDrawing{
	grid:  16,
	plate: func(in float64) []segment { return roundRect(in, in, 16-in, 16-in, 3) },
	sheet: polygon(point{3, 1}, point{10, 1}, point{13, 4}, point{13, 15}, point{3, 15}),
	fold:  polygon(point{10, 1}, point{10, 4}, point{13, 4}),
	code: [][]segment{
		stroke(1.7, point{7, 7.2}, point{4.6, 10}, point{7, 12.8}),
		stroke(1.7, point{9, 7.2}, point{11.4, 10}, point{9, 12.8}),
	},
}

var fullMark = markDrawing{
	grid:  48,
	plate: func(in float64) []segment { return roundRect(in, in, 48-in, 48-in, 9-in/2) },
	sheet: polygon(point{11, 5}, point{28, 5}, point{37, 14}, point{37, 43}, point{11, 43}),
	fold:  polygon(point{28, 5}, point{28, 14}, point{37, 14}),
	code: [][]segment{
		stroke(3.4, point{20, 22.5}, point{14.8, 29}, point{20, 35.5}),
		stroke(3.4, point{28, 22.5}, point{33.2, 29}, point{28, 35.5}),
		stroke(2.8, point{25.6, 21.5}, point{22.4, 36.5}),
	},
	insetR: 1.5,
}

func drawingFor(px float64) markDrawing {
	if px < 24 {
		return smallMark
	}
	return fullMark
}

// MarkForm says what the mark is drawn on.
type MarkForm int

const (
	// Plated: the mark on its own navy plate - the ICO, the extension's action icon, the
	// MSIX unplated forms and StoreLogo. It holds on a light and a dark background alike.
	Plated MarkForm = iota
	// OnPlatform: the sheet alone on transparency, for the MSIX plated forms and tiles
	// that Windows draws on the manifest BackgroundColor (the same navy).
	OnPlatform
)

// Mark draws the mark in a w x h image, the mark's square box of side box centred in it.
func Mark(w, h int, box float64, form MarkForm) *image.RGBA {
	cv := newCanvas(w, h)
	d := drawingFor(box)
	t := transform{scale: box / d.grid, dx: (float64(w) - box) / 2, dy: (float64(h) - box) / 2}
	if form == Plated {
		inset := 0.0
		if box >= 32 {
			inset = d.insetR
		}
		cv.fill(d.plate(inset), t, Navy)
	}
	cv.fill(d.sheet, t, Sheet)
	cv.fill(d.fold, t, Fold)
	for _, c := range d.code {
		if form == Plated {
			cv.fill(c, t, Navy)
		} else {
			cv.erase(c, t)
		}
	}
	return cv.img
}

// Glyph draws a vocabulary glyph (24-unit path data) in one colour at size px, transparent
// around it - the mono look of ICON-RENDER.
func Glyph(pathData string, px int, c color.Color) (*image.RGBA, error) {
	segs, err := parsePath(pathData)
	if err != nil {
		return nil, err
	}
	cv := newCanvas(px, px)
	cv.fill(segs, transform{scale: float64(px) / 24}, c)
	return cv.img, nil
}
