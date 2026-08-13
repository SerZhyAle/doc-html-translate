package ocr

import (
	"image"
	"math"
	"math/rand/v2"
	"testing"
)

// halftoneField draws a dot lattice of the given pitch over a light ground - the thing a press
// lays down to print a tone, and the thing the screen rung has to recognize.
func halftoneField(w, h, pitch int) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, w, h))
	for i := range g.Pix {
		g.Pix[i] = 240
	}
	r := float64(pitch) / 3
	for cy := 0; cy < h; cy += pitch {
		for cx := 0; cx < w; cx += pitch {
			for y := cy - pitch; y <= cy+pitch; y++ {
				for x := cx - pitch; x <= cx+pitch; x++ {
					if x < 0 || y < 0 || x >= w || y >= h {
						continue
					}
					dx, dy := float64(x-cx), float64(y-cy)
					if math.Hypot(dx, dy) <= r {
						g.Pix[y*g.Stride+x] = 90
					}
				}
			}
		}
	}
	return g
}

// TestScreenPitchFindsTheLattice: the sigma the rung filters with is derived from this number, so
// reading the period wrong is the same as choosing the wrong kernel. Several pitches, because a
// screen's period depends on the press and on the scan resolution - a detector that only works at
// one pitch is a fix for one pitch.
func TestScreenPitchFindsTheLattice(t *testing.T) {
	for _, pitch := range []int{4, 6, 8, 12} {
		if got := screenPitch(halftoneField(320, 320, pitch)); got != pitch {
			t.Errorf("screenPitch on a pitch-%d screen = %d", pitch, got)
		}
	}
}

// TestScreenPitchIgnoresWhatIsNotAScreen: the rung costs a recognition pass, and on a picture with
// no lattice that pass is spent for nothing. Flat paper has no residual to measure at all; noise
// has residual everywhere but no period the tiles agree on.
func TestScreenPitchIgnoresWhatIsNotAScreen(t *testing.T) {
	flat := image.NewGray(image.Rect(0, 0, 320, 320))
	for i := range flat.Pix {
		flat.Pix[i] = 236
	}
	if got := screenPitch(flat); got != 0 {
		t.Errorf("screenPitch on flat paper = %d, want 0", got)
	}

	noise := image.NewGray(image.Rect(0, 0, 320, 320))
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range noise.Pix {
		noise.Pix[i] = uint8(rng.IntN(256))
	}
	if got := screenPitch(noise); got != 0 {
		t.Errorf("screenPitch on white noise = %d, want 0", got)
	}
}

// TestScreenPitchRejectsPeriodsOutsideTheBounds: a two-pixel "period" is JPEG blocking or the
// scanner's own grain, and blurring for it would soften an image that has no screen at all.
func TestScreenPitchRejectsPeriodsOutsideTheBounds(t *testing.T) {
	if got := screenPitch(halftoneField(320, 320, 2)); got != 0 {
		t.Errorf("screenPitch on a 2 px lattice = %d, want 0 (below ocrScreenMinPitch)", got)
	}
	if ocrScreenMinPitch < 3 {
		t.Errorf("ocrScreenMinPitch = %d: a period this short is noise, not a press screen", ocrScreenMinPitch)
	}
}

// TestScreenPitchOutsideAsksTheNarrowerQuestion pins the additive sweep's trigger. It is not "does
// this picture carry a screen" but "is there screened area the reader has no plate over", because a
// screen under lettering that is already plated has nothing left to give and the sweep it would
// trigger costs a whole recognition.
//
// The third case is the one the ticket exists for: a screened caption standing beside a balloon the
// ordinary pass read. A trigger that fails it does nothing at all, and does it silently.
func TestScreenPitchOutsideAsksTheNarrowerQuestion(t *testing.T) {
	const pitch = 8
	// A screen over the right half of the picture, flat paper over the left.
	img := halftoneField(320, 320, pitch)
	for y := 0; y < 320; y++ {
		for x := 0; x < 160; x++ {
			img.Pix[y*img.Stride+x] = 240
		}
	}

	if got := screenPitchOutside(img, nil); got != pitch {
		t.Fatalf("no plates: screenPitchOutside = %d, want %d", got, pitch)
	}
	overTheScreen := []image.Rectangle{image.Rect(128, 0, 320, 320)}
	if got := screenPitchOutside(img, overTheScreen); got != 0 {
		t.Errorf("a plate over the screened half: screenPitchOutside = %d, want 0 - nothing is left to sweep for", got)
	}
	elsewhere := []image.Rectangle{image.Rect(0, 0, 128, 320)}
	if got := screenPitchOutside(img, elsewhere); got != pitch {
		t.Errorf("a plate over the plain half: screenPitchOutside = %d, want %d - the screened caption beside it is still unserved", got, pitch)
	}
}

// TestMergeScreenBlocksDropsWhatIsAlreadyPlated pins the additive sweep's other half. The sweep is
// only safe if the page it adds to comes through untouched, and only useful if what it adds is not a
// second plate over lettering that already has one - which is visible damage, worse for the reader
// than the caption staying untranslated.
//
// The straddling case is the one the design turns on: a candidate whose halves lie under two
// different existing plates is entirely covered, and a rule that took each existing plate on its own
// would see two halves under the bound and let it through.
func TestMergeScreenBlocksDropsWhatIsAlreadyPlated(t *testing.T) {
	left := Block{Text: "already read", X0: 200, Y0: 0, X1: 335, Y1: 100}
	right := Block{Text: "also already read", X0: 465, Y0: 0, X1: 600, Y1: 100}
	kept := []Block{left, right}

	clear := Block{Text: "a screened caption", X0: 0, Y0: 200, X1: 100, Y1: 260}
	duplicate := Block{Text: "already read", X0: 210, Y0: 10, X1: 330, Y1: 90}
	// Each existing plate takes 35 columns of this one's 200, under the bound on its own; together
	// they take 70, over it.
	straddling := Block{Text: "already read twice", X0: 300, Y0: 0, X1: 500, Y1: 100}

	got := mergeScreenBlocks(kept, []Block{clear, duplicate, straddling})

	if len(got) < len(kept) || !sameBlock(got[0], kept[0]) || !sameBlock(got[1], kept[1]) {
		t.Fatalf("the ordinary pass's plates must come through unchanged and in order, got %+v", got)
	}
	if len(got) != len(kept)+1 || !sameBlock(got[2], clear) {
		t.Errorf("merged %d plates (%+v), want the two kept ones plus %q only", len(got), got, clear.Text)
	}

	// The straddling case again, as the arithmetic rather than the outcome: neither existing plate
	// puts it over the bound alone, the two together do.
	r := image.Rect(straddling.X0, straddling.Y0, straddling.X1, straddling.Y1)
	for _, k := range kept {
		one := coveredFraction(r, []image.Rectangle{image.Rect(k.X0, k.Y0, k.X1, k.Y1)})
		if one > ocrScreenMergeMaxOverlap {
			t.Fatalf("the straddling case is not straddling: %q alone covers %.3f of it", k.Text, one)
		}
	}
	if both := coveredFraction(r, blockRects(kept)); both <= ocrScreenMergeMaxOverlap {
		t.Errorf("the two plates together cover %.3f of the straddling candidate, want more than %.2f",
			both, ocrScreenMergeMaxOverlap)
	}
}

// TestScreenSigmaMovesTheThresholdOffTheScreen pins what ocrScreenSigmaDivisor is for, in the terms
// that actually decide whether recognition sees anything: where a global thresholder puts its
// cut-off.
//
// On a screened picture the dots are a population of their own, and Otsu splits *them* from the
// paper rather than splitting the lettering from everything - so the mask that reaches recognition
// is a field of dots with the words lost inside it. The low-pass has to average the dots into a
// flat tone so the cut-off falls back onto the ink, and it has to do that without dissolving the
// lettering, which a wider kernel would. Either half alone is easy; the divisor is what gets both.
func TestScreenSigmaMovesTheThresholdOffTheScreen(t *testing.T) {
	const pitch = 12
	const strokeX0, strokeX1 = 100, 140 // a bold stroke of the width this lettering has
	g := halftoneField(240, 240, pitch)
	for y := 0; y < 240; y++ {
		for x := strokeX0; x < strokeX1; x++ {
			g.Pix[y*g.Stride+x] = 0
		}
	}
	blurred := gaussBlurGray(g, pitch/ocrScreenSigmaDivisor)

	// The darkest screen pixel away from the stroke: everything at or above it must read as paper.
	darkestScreen := func(im *image.Gray) int {
		lo := 255
		for y := 20; y < 200; y++ {
			for x := 170; x < 230; x++ {
				lo = min(lo, int(im.Pix[y*im.Stride+x]))
			}
		}
		return lo
	}
	strokeLevel := func(im *image.Gray) int {
		return int(im.Pix[120*im.Stride+(strokeX0+strokeX1)/2])
	}

	cut := otsu(blurred)
	if lo := darkestScreen(blurred); cut >= lo {
		t.Errorf("after the low-pass the cut-off is %d and the darkest screen pixel %d: the screen "+
			"still reads as ink", cut, lo)
	}
	if ink := strokeLevel(blurred); ink >= cut {
		t.Errorf("after the low-pass the stroke sits at %d against a cut-off of %d: the kernel took "+
			"the lettering with the screen", ink, cut)
	}

	// The lower side of the window, so a divisor that drifts upward (a kernel too narrow to reach
	// the screen) fails here rather than only in a measurement nobody re-runs. The upper side
	// cannot be asserted on a scene this simple - a solid bar survives any blur, whereas a real
	// glyph with counters does not - and its evidence stays where it was measured:
	// DEV/research/ocr_halftone_2026-08-12.md records the whole transcript coming back at sigma
	// 3.0-3.5 on this pitch and nothing at all at 4.0.
	tooNarrow := gaussBlurGray(g, pitch/16.0)
	if lo, c := darkestScreen(tooNarrow), otsu(tooNarrow); lo > c {
		t.Errorf("a kernel far too narrow (sigma %.2f) already clears the screen (darkest %d, cut %d): "+
			"the test cannot tell a working sigma from a useless one", pitch/16.0, lo, c)
	}
}

// otsu is the textbook between-class-variance threshold - the same decision Tesseract's default
// thresholder makes, reproduced here so the test can say where the cut-off lands.
func otsu(im *image.Gray) int {
	var hist [256]int
	for _, v := range im.Pix {
		hist[v]++
	}
	total := len(im.Pix)
	sum := 0.0
	for i, n := range hist {
		sum += float64(i * n)
	}
	sumB, wB, best, bestVar := 0.0, 0, 0, -1.0
	for i, n := range hist {
		wB += n
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += float64(i * n)
		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)
		if v := float64(wB) * float64(wF) * (mB - mF) * (mB - mF); v > bestVar {
			best, bestVar = i, v
		}
	}
	return best
}

// TestScreenRungSkipsAnImageWithoutAScreen: the rung is the last thing tried on a picture nothing
// could read, so its cost lands on the images least likely to repay it. Skipping when there is no
// lattice is what keeps that cost off every other failure - a Cyrillic poster read with English
// data, say, which fails for a reason no filter can fix.
func TestScreenRungSkipsAnImageWithoutAScreen(t *testing.T) {
	flat := image.NewGray(image.Rect(0, 0, 200, 200))
	for i := range flat.Pix {
		flat.Pix[i] = 200
	}
	// A bin that would fail loudly if it were ever executed: reaching it means the rung did not
	// skip. Nothing runs, so no tesseract is needed for this test.
	if _, ok := screenRescue("tesseract-that-does-not-exist", flat, "eng", "", 150); ok {
		t.Error("screenRescue reported a result for an image with no screen")
	}
}

// TestScreenRungIsAfterTheGreyLadder: order is the invariant, not the presence of the rungs. The
// screen pass is the only one that changes the picture rather than the reading of it, and it costs
// accuracy on lettering the cheaper rungs can already read, so it has to come last.
func TestScreenRungIsAfterTheGreyLadder(t *testing.T) {
	// The grey rungs read the picture as it is; the screen rung is the one that alters it, and it
	// is reached only when every rung here came back empty. Counting them would break each time a
	// rung is added, so what is pinned is that they all read the unaltered grey rendition.
	if len(greyRescuePasses) == 0 {
		t.Fatalf("greyRescuePasses is empty: the screen rung is defined as the one after these")
	}
	for _, rung := range greyRescuePasses {
		if rung.psm != ocrPageSegMode && rung.psm != ocrSparsePageSegMode {
			t.Errorf("rung %v asks for a segmentation mode this ladder does not define", rung)
		}
	}
	if ocrScreenSigmaDivisor <= 0 {
		t.Errorf("ocrScreenSigmaDivisor = %v, want a positive divisor of the measured pitch", ocrScreenSigmaDivisor)
	}
}
