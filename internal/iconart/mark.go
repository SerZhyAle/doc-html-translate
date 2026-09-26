package iconart

import (
	"image"
	"image/color"
)

// The product mark: a white sheet with a folded corner and "</>" cut into it,
// on a navy plate (#1E3A8A). It is product artwork (ICON-SET rule 7) - never a vocabulary
// glyph and never drawn for a meaning. Three drawings of it:
//  1. Small (16 and 20 px): drops the slash and thickens the brackets so the sheet and "<>"
//     read with razor-sharp 1px stroke clarity on high-DPI and standard-DPI taskbars.
//  2. Medium (24 to 64 px): the full "</>" code transformation symbol on the folded sheet.
//  3. Large (96 px and up: 128, 150, 256, 310 px): adds the "DOC" header and "HTML" footer
//     product branding typography framing the prominent "</>" transformation center.
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

var mediumMark = markDrawing{
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

var (
	docSVG = "M 30,20 H 37.5 C 40.5,20 43,22.2 43,27 C 43,31.8 40.5,34 37.5,34 H 30 Z M 33.5,23 V 31 H 37 C 38.8,31 39.8,29.5 39.8,27 C 39.8,24.5 38.8,23 37,23 Z " +
		"M 52,20 C 56.2,20 59.5,23 59.5,27 C 59.5,31 56.2,34 52,34 C 47.8,34 44.5,31 44.5,27 C 44.5,23 47.8,20 52,20 Z M 52,23 C 49.2,23 47.8,24.6 47.8,27 C 47.8,29.4 49.2,31 52,31 C 54.8,31 56.2,29.4 56.2,27 C 56.2,24.6 54.8,23 52,23 Z " +
		"M 76.5,23.5 C 75,21.3 72.5,20 69.5,20 C 65.5,20 62.5,23.2 62.5,27 C 62.5,30.8 65.5,34 69.5,34 C 72.5,34 75,32.7 76.5,30.5 L 73.9,28.5 C 72.9,29.8 71.3,30.8 69.5,30.8 C 67.2,30.8 65.8,29 65.8,27 C 65.8,25 67.2,23.2 69.5,23.2 C 71.3,23.2 72.9,24.2 73.9,25.5 Z"

	htmlSVG = "M 34,96 H 37.5 V 101.5 H 42.5 V 96 H 46 V 110 H 42.5 V 104.5 H 37.5 V 110 H 34 Z " +
		"M 49,96 H 61 V 99 H 56.8 V 110 H 53.2 V 99 H 49 Z " +
		"M 64,96 H 67.5 L 71,102.5 L 74.5,96 H 78 V 110 H 74.5 V 101 L 71.8,105.8 H 70.2 L 67.5,101 V 110 H 64 Z " +
		"M 81,96 H 84.5 V 107 H 93 V 110 H 81 Z"
)

var largeMark = func() markDrawing {
	docSegs, err := parsePath(docSVG)
	if err != nil {
		panic(err)
	}
	htmlSegs, err := parsePath(htmlSVG)
	if err != nil {
		panic(err)
	}
	code := [][]segment{
		docSegs,
		// Hero code symbol "</>"
		polygon(point{49, 48}, point{31, 66}, point{49, 84}, point{54.5, 78.5}, point{42, 66}, point{54.5, 53.5}),
		polygon(point{79, 48}, point{73.5, 53.5}, point{86, 66}, point{73.5, 78.5}, point{79, 84}, point{97, 66}),
		polygon(point{69, 45}, point{74.5, 47}, point{59, 87}, point{53.5, 85}),
		// Bottom "HTML" branding lettering
		htmlSegs,
	}
	return markDrawing{
		grid:   128,
		plate:  func(in float64) []segment { return roundRect(in, in, 128-in, 128-in, 24-in/2) },
		sheet:  polygon(point{24, 10}, point{82, 10}, point{104, 32}, point{104, 118}, point{24, 118}),
		fold:   polygon(point{82, 10}, point{82, 32}, point{104, 32}),
		code:   code,
		insetR: 4.0,
	}
}()

func drawingFor(px float64) markDrawing {
	if px < 24 {
		return smallMark
	}
	if px < 96 {
		return mediumMark
	}
	return largeMark
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
