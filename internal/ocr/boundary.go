package ocr

import "image"

// strokeBetween reports whether a stroke crosses the gap between two consecutive words of one
// recognizer line - evidence that the two words were never one line, whatever the width of the gap.
//
// It is the pixel half of splitWideGaps. The ratio rule there cannot see the case this answers:
// comic balloons drawn side by side are stitched into one line with the words 1.00-3.46 word
// heights apart, while real lines reach 3.07x on the same corpus (a letter-spaced masthead), so the
// two populations overlap and no ratio separates them. What separates them is what a reader sees
// between the words: two balloon outlines, or a panel rule, drawn clean through the line and on past
// it. A letter the recognizer left out of its word box - the T of "DON'T", the V of "IV. VINTER",
// both measured - is ink in the gap too, but it stays inside the line's band.
//
// So the test is a path of ink, 8-connected, through the strip strictly between the two boxes, from
// reach pixels above the words' band to reach pixels below it. Ink is a pixel whose luma stands at
// least plateMinContrast from the paper - the same distance the overlay requires between a plate's
// paper and its ink - and the paper is the median luma of the rows just outside both words' boxes
// (see paperLuma). A window the picture cannot hold, a strip with no width, or a picture that could
// not be decoded is no evidence, and no evidence never cuts.
//
// Measured over the 46 lab scenes plus test_doc/1.png with the desktop engine, all 1043 word gaps
// of the lines that clear ocrMinLineConf (DEV/research/ocr_balloon_boundary_2026-09-25.md). Under
// ocrMaxWordGapRatio the test fires 21 times: 11 stitches between two regions (10 of them two
// balloons side by side, from 1.00x up), 8 balloon outlines or scraps of artwork the recognizer read as a token of the line
// ("|", "}", "gs") and 2 noise lines - and never on a real line. Of the 878 gaps under 1.00x, two
// are cut, both in front of an outline read as "|". See ocrBoundaryReach for the one number.
// Mirrors ocr-cluster.js strokeBetween (docs/PARITY.md).
func strokeBetween(ink *image.Gray, a, b ocrWord, reach int) bool {
	if ink == nil || reach < 1 {
		return false
	}
	x0, x1 := a.x1, b.x0
	if b.x1 <= a.x0 { // right to left: b stands to the left of a
		x0, x1 = b.x1, a.x0
	}
	y0, y1 := min(a.y0, b.y0)-reach, max(a.y1, b.y1)+reach
	bnds := ink.Bounds()
	if x1 <= x0 || x0 < bnds.Min.X || x1 > bnds.Max.X || y0 < bnds.Min.Y || y1 > bnds.Max.Y {
		return false
	}
	paper, ok := paperLuma(ink, a, b)
	if !ok {
		return false
	}
	w, h := x1-x0, y1-y0
	isInk := func(x, y int) bool {
		d := int(ink.GrayAt(x0+x, y0+y).Y) - paper
		return d >= plateMinContrast || -d >= plateMinContrast
	}
	seen := make([]bool, w*h)
	var stack []int
	for x := range w {
		if isInk(x, 0) {
			seen[x] = true
			stack = append(stack, x)
		}
	}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		py, px := p/w, p%w
		if py == h-1 {
			return true
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := px+dx, py+dy
				if nx < 0 || ny < 0 || nx >= w || ny >= h || seen[ny*w+nx] {
					continue
				}
				if isInk(nx, ny) {
					seen[ny*w+nx] = true
					stack = append(stack, ny*w+nx)
				}
			}
		}
	}
	return false
}

// paperLuma is the median luma of the paper the words sit on: two rows above and two rows below each
// word's box, across the word's own width, skipping the row that touches the box so a glyph's
// antialiased edge is not read as paper.
//
// Outside the boxes and not inside them, because inside is not reliably paper: bold comic lettering
// fills half of a tight word box, and the median inside one read 111 on a balloon whose paper is 228
// (samson-and-delilah-15, measured while building this). A band of rows above and below the words is
// the balloon's interior on a balloon and the page on a page, and it holds for light lettering on a
// dark ground as well, where the paper is the dark side. Mirrors ocr-cluster.js paperLuma.
func paperLuma(ink *image.Gray, words ...ocrWord) (int, bool) {
	bnds := ink.Bounds()
	var vals []int
	for _, w := range words {
		for _, y := range [...]int{w.y0 - 3, w.y0 - 2, w.y1 + 1, w.y1 + 2} {
			if y < bnds.Min.Y || y >= bnds.Max.Y {
				continue
			}
			for x := max(w.x0, bnds.Min.X); x < min(w.x1, bnds.Max.X); x++ {
				vals = append(vals, int(ink.GrayAt(x, y).Y))
			}
		}
	}
	if len(vals) == 0 {
		return 0, false
	}
	return median(vals, 0), true
}
