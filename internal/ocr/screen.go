package ocr

import (
	"image"
	"math"
	"sort"
)

// A halftone screen - the dot lattice a press lays down to print a tone - is the one thing the
// grey rescue ladder cannot see through. When the screen's own tone falls between the ink and the
// paper, a global thresholder either swallows the lettering with the dots or turns the whole
// picture into texture, and the mask that reaches recognition holds no text. Measured on the lab's
// diagnostic scene `synth-text-on-halftone` (a 50% grey dot screen under bold type): the colour
// pass, the grey pass and the tiled-Otsu pass all read nothing at every scale.
//
// What does recover it is a low-pass wide enough to dissolve the lattice and narrow enough to
// leave a letter stroke standing. That is a *screen-aware* transform rather than a blur applied
// blindly: the kernel is derived from the screen's own measured period, which is why this file
// measures before it filters.
//
// Both numbers below come from measurements recorded in
// DEV/research/ocr_halftone_2026-08-12.md, over 59 corpus and harvested images.
const (
	// ocrScreenSigmaDivisor turns the measured screen pitch into a Gaussian sigma. Chosen because
	// it is at or next to the optimum on every screened image measured, at two different pitches
	// and on real material as well as synthetic: on the diagnostic scene (pitch 12 after staging)
	// the whole transcript comes back for sigma 3.0-3.5 and nothing at all outside that band; on a
	// real screened comic caption (pitch 6) sigma 1.5 gives the best confident-word yield of any
	// kernel tried; on a real screened scroll panel (pitch 12) sigma 2.4-3.0 is the peak. A
	// divisor near 4 is the only single rule that lands inside all three windows.
	ocrScreenSigmaDivisor = 4.0

	// The screen detector. A dot lattice repeats at a fixed period, so after a high-pass the
	// residual correlates with itself at that lag - no frequency transform needed, which is what
	// makes the test cheap enough to run inside a rescue.
	ocrScreenTile      = 64   // side of the square the autocorrelation is taken over
	ocrScreenMinPitch  = 3    // below this a "period" is JPEG noise or the sensor, not a screen
	ocrScreenMaxPitch  = 24   // above this the lattice is coarser than any lettering it could hide
	ocrScreenMaxTiles  = 96   // cap the work so a 20-megapixel page costs the same as a panel
	ocrScreenMinEnergy = 3.0  // a tile flatter than this is paper or solid ink - nothing to find
	ocrScreenPeakFloor = 0.30 // autocorrelation at the winning lag, relative to lag 0
	ocrScreenTileFrac  = 0.25 // share of textured tiles that must agree on one pitch

	// ocrScreenTileCoverMax is how much of a tile an existing plate may cover before the tile stops
	// counting as evidence of screened area the reader is not served on. Half rather than "touches
	// at all": on a dense page a plate clipping a tile's corner would otherwise blind the detector
	// to a screened caption standing right beside a balloon, which is the case this whole trigger
	// exists for. Half rather than "covers it entirely" for the mirror reason - a plate that takes
	// most of a tile leaves too little unserved area for the remainder to mean anything.
	ocrScreenTileCoverMax = 0.5
)

// screenPitch returns the period in pixels of the halftone screen covering the image, or 0 when
// there is no lattice to find. It samples up to ocrScreenMaxTiles tiles spread over the picture,
// high-passes each one, and takes the lag whose autocorrelation peaks; the answer is the pitch the
// most tiles agree on, and only when at least ocrScreenTileFrac of the textured tiles agree.
//
// Requiring agreement across tiles is what separates a press screen from an accident: a screen
// covers an area with one period, whereas JPEG blocking, film grain and engraved hatching either
// vary tile to tile or fall outside the pitch bounds.
func screenPitch(g *image.Gray) int { return screenPitchOutside(g, nil) }

// screenPitchOutside is screenPitch restricted to the parts of the picture no rectangle in covered
// reaches: it answers "is there screened area the reader has no plate over" rather than "does this
// picture carry a screen". That is the cheaper of the two questions the additive sweep could ask -
// the alternative is running a second recognition to find out - and it is the one that matters,
// because a screen under lettering that is already plated has nothing left to give.
//
// The vote share ocrScreenTileFrac applies to the tiles that survive the skip, which is the intended
// reading: a quarter of the *unserved* textured tiles must agree on one pitch.
func screenPitchOutside(g *image.Gray, covered []image.Rectangle) int {
	w, h := g.Bounds().Dx(), g.Bounds().Dy()
	if w < ocrScreenTile || h < ocrScreenTile {
		return 0
	}
	// Spread the sampled tiles over the whole picture rather than taking the first few, so a page
	// whose top is a solid masthead is still judged by its body.
	step := tileStep(w, h)

	votes := map[int]int{}
	textured, agreed := 0, 0
	buf := make([]float64, ocrScreenTile*ocrScreenTile)
	for ty := 0; ty+ocrScreenTile <= h; ty += step {
		for tx := 0; tx+ocrScreenTile <= w; tx += step {
			if tileServed(tx, ty, covered) {
				continue
			}
			if !tileResidual(g, tx, ty, buf) {
				continue
			}
			textured++
			if lag, peak := tilePeriod(buf); lag > 0 && peak >= ocrScreenPeakFloor {
				votes[lag]++
			}
		}
	}
	if textured == 0 {
		return 0
	}
	best := 0
	for lag, n := range votes {
		if n > agreed || (n == agreed && lag < best) {
			best, agreed = lag, n
		}
	}
	if best < ocrScreenMinPitch || best > ocrScreenMaxPitch {
		return 0
	}
	if float64(agreed) < ocrScreenTileFrac*float64(textured) {
		return 0
	}
	return best
}

// tileStep spaces the sampled tiles so no more than ocrScreenMaxTiles of them fit, whatever the
// image size. The step is never below the tile itself, so tiles never overlap.
func tileStep(w, h int) int {
	tiles := (w / ocrScreenTile) * (h / ocrScreenTile)
	if tiles <= ocrScreenMaxTiles {
		return ocrScreenTile
	}
	return ocrScreenTile * int(math.Ceil(math.Sqrt(float64(tiles)/ocrScreenMaxTiles)))
}

// tileServed reports whether one of the covered rectangles already takes more than
// ocrScreenTileCoverMax of this tile. Each rectangle is tested on its own rather than as a union:
// two plates that between them cover a tile are two separate regions with a gap down the middle,
// and that gap is exactly where an unserved caption would sit.
func tileServed(tx, ty int, covered []image.Rectangle) bool {
	tile := image.Rect(tx, ty, tx+ocrScreenTile, ty+ocrScreenTile)
	limit := int(ocrScreenTileCoverMax * float64(ocrScreenTile*ocrScreenTile))
	for _, c := range covered {
		in := c.Intersect(tile)
		if in.Dx()*in.Dy() > limit {
			return true
		}
	}
	return false
}

// tileResidual fills buf with the tile's high-pass residual (the pixel minus its 3x3 mean, which
// is where a screen lives) and reports whether the tile carries enough of it to be worth testing.
func tileResidual(g *image.Gray, tx, ty int, buf []float64) bool {
	w, h := g.Bounds().Dx(), g.Bounds().Dy()
	at := func(x, y int) float64 {
		return float64(g.Pix[min(max(y, 0), h-1)*g.Stride+min(max(x, 0), w-1)])
	}
	energy := 0.0
	for j := 0; j < ocrScreenTile; j++ {
		for i := 0; i < ocrScreenTile; i++ {
			x, y := tx+i, ty+j
			mean := 0.0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					mean += at(x+dx, y+dy)
				}
			}
			v := at(x, y) - mean/9
			buf[j*ocrScreenTile+i] = v
			energy += v * v
		}
	}
	return energy/float64(ocrScreenTile*ocrScreenTile) >= ocrScreenMinEnergy
}

// tilePeriod returns the lag whose normalized autocorrelation is highest in the tile, and that
// value. Both axes are summed into one sequence because a press screen is normally rotated (45
// degrees for black, other angles per colour) - it still repeats along x and along y, but neither
// axis alone carries the whole signal.
//
// Only a local maximum counts. A smooth gradient decorrelates monotonically, so without that rule
// every gradient would report the smallest lag as its "period".
func tilePeriod(buf []float64) (int, float64) {
	acf := make([]float64, ocrScreenMaxPitch+1)
	for lag := 0; lag <= ocrScreenMaxPitch; lag++ {
		sum := 0.0
		for y := 0; y < ocrScreenTile; y++ {
			row := y * ocrScreenTile
			for x := 0; x+lag < ocrScreenTile; x++ {
				sum += buf[row+x] * buf[row+x+lag]
			}
		}
		for x := 0; x < ocrScreenTile; x++ {
			for y := 0; y+lag < ocrScreenTile; y++ {
				sum += buf[y*ocrScreenTile+x] * buf[(y+lag)*ocrScreenTile+x]
			}
		}
		acf[lag] = sum
	}
	if acf[0] <= 0 {
		return 0, 0
	}
	bestLag, bestVal := 0, 0.0
	for lag := 2; lag <= ocrScreenMaxPitch; lag++ {
		v := acf[lag] / acf[0]
		if v <= bestVal || v < acf[lag-1]/acf[0] {
			continue
		}
		if lag < ocrScreenMaxPitch && v < acf[lag+1]/acf[0] {
			continue
		}
		bestLag, bestVal = lag, v
	}
	return bestLag, bestVal
}

// gaussBlurGray returns a Gaussian-blurred copy of g. Separable, kernel truncated at three sigma.
// A Gaussian rather than a box mean because the screen has to be suppressed at its own frequency
// *and* at its harmonics: a box kernel's stop band has holes, and a screen that lands in one of
// them survives the blur that was supposed to remove it.
func gaussBlurGray(g *image.Gray, sigma float64) *image.Gray {
	w, h := g.Bounds().Dx(), g.Bounds().Dy()
	if sigma <= 0 || w == 0 || h == 0 {
		return g
	}
	r := int(math.Ceil(3 * sigma))
	kernel := make([]float64, 2*r+1)
	sum := 0.0
	for i := -r; i <= r; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		kernel[i+r] = v
		sum += v
	}
	for i := range kernel {
		kernel[i] /= sum
	}

	// float32 for the intermediate, not float64: a book scanned at 600 DPI is a 20-megapixel page
	// and this buffer is the pass's whole footprint. The app builds for 386, so the process has a
	// 2 GB ceiling to stay well under, and eight bytes a pixel buys no accuracy that survives the
	// uint8 the row below rounds back into.
	tmp := make([]float32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			acc := 0.0
			for i := -r; i <= r; i++ {
				acc += kernel[i+r] * float64(g.Pix[y*g.Stride+min(max(x+i, 0), w-1)])
			}
			tmp[y*w+x] = float32(acc)
		}
	}
	out := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			acc := 0.0
			for j := -r; j <= r; j++ {
				acc += kernel[j+r] * float64(tmp[min(max(y+j, 0), h-1)*w+x])
			}
			out.Pix[y*out.Stride+x] = clampByte(acc)
		}
	}
	return out
}

func clampByte(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	default:
		return uint8(v + 0.5)
	}
}

// ocrScreenMergeMaxOverlap is how much of a screen-pass plate may already be covered by the plates
// the ordinary pass produced before it is dropped as a duplicate.
//
// The bound is a fraction of the *new* plate rather than of the existing one because the question
// the merge asks is "is this lettering already plated", and a small candidate sitting wholly inside
// a large existing plate is a duplicate however little of that plate it occupies. It is small
// because the two outcomes are not symmetric: a caption that stays untranslated is the complaint
// this feature answers, but a second plate painted over lettering that already has one is visible
// damage on a page the reader was happy with. A fifth leaves room for the boxes to disagree at their
// edges - two passes cluster the same lines slightly differently - without letting a real overlap
// through.
const ocrScreenMergeMaxOverlap = 0.2

// mergeScreenBlocks returns kept, unchanged and in order, followed by those of found that the plates
// already accepted leave mostly uncovered. "Already accepted" includes the screen plates taken
// earlier in the same call, so two of them cannot stack on each other either.
//
// Every plate the ordinary pass produced survives: the sweep is additive, and a pass that could move
// or drop an existing plate would put a page that reads fine today at risk to help one that does not.
func mergeScreenBlocks(kept, found []Block) []Block {
	out := make([]Block, len(kept), len(kept)+len(found))
	copy(out, kept)
	taken := blockRects(kept)
	for _, b := range found {
		r := image.Rect(b.X0, b.Y0, b.X1, b.Y1)
		if coveredFraction(r, taken) > ocrScreenMergeMaxOverlap {
			continue
		}
		out = append(out, b)
		taken = append(taken, r)
	}
	return out
}

// blockRects is the blocks' boxes as rectangles, which is the form both the merge and the trigger
// want.
func blockRects(blocks []Block) []image.Rectangle {
	out := make([]image.Rectangle, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, image.Rect(b.X0, b.Y0, b.X1, b.Y1))
	}
	return out
}

// coveredFraction returns how much of r lies inside the union of rects, as a fraction of r's area.
//
// The union rather than the sum of the overlaps, and that is the whole point: a candidate whose two
// halves lie under two different existing plates is entirely covered, but a rule that took each
// existing plate on its own would see two halves and let it through. Summing instead double-counts
// wherever the existing plates overlap each other and can report more than the whole. The union is
// computed exactly, by compressing the rectangles' own edge coordinates into a grid - every cell of
// it is either wholly inside a given overlap or wholly outside it - which costs nothing at the
// handful of plates a page carries.
func coveredFraction(r image.Rectangle, rects []image.Rectangle) float64 {
	area := r.Dx() * r.Dy()
	if area <= 0 {
		return 0
	}
	parts := make([]image.Rectangle, 0, len(rects))
	xs := []int{r.Min.X, r.Max.X}
	ys := []int{r.Min.Y, r.Max.Y}
	for _, o := range rects {
		in := o.Intersect(r)
		if in.Empty() {
			continue
		}
		parts = append(parts, in)
		xs = append(xs, in.Min.X, in.Max.X)
		ys = append(ys, in.Min.Y, in.Max.Y)
	}
	if len(parts) == 0 {
		return 0
	}
	sort.Ints(xs)
	sort.Ints(ys)

	covered := 0
	for i := 0; i+1 < len(xs); i++ {
		for j := 0; j+1 < len(ys); j++ {
			cell := image.Rect(xs[i], ys[j], xs[i+1], ys[j+1])
			if cell.Empty() {
				continue
			}
			for _, p := range parts {
				if cell.In(p) {
					covered += cell.Dx() * cell.Dy()
					break
				}
			}
		}
	}
	return float64(covered) / float64(area)
}
