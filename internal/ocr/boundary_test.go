package ocr

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"testing"
)

// paperPage is a blank page of the given luma; fill paints a rectangle of it.
func paperPage(w, h int, luma uint8) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, w, h))
	draw.Draw(g, g.Bounds(), image.NewUniform(color.Gray{Y: luma}), image.Point{}, draw.Src)
	return g
}

func fill(g *image.Gray, x0, y0, x1, y1 int, luma uint8) {
	draw.Draw(g, image.Rect(x0, y0, x1, y1), image.NewUniform(color.Gray{Y: luma}), image.Point{}, draw.Src)
}

// Two words 100 px tall on white paper, 200 px apart - a 2.0x gap, well under ocrMaxWordGapRatio,
// inside the band where balloon stitches and real lines overlap.
var (
	leftWord  = ocrWord{x0: 100, y0: 200, x1: 300, y1: 300, text: "BE"}
	rightWord = ocrWord{x0: 500, y0: 200, x1: 700, y1: 300, text: "WHO"}
)

func reachFor(med int) int { return int(float64(med) * ocrBoundaryReach) }

func TestStrokeBetweenCutsAtABalloonOutline(t *testing.T) {
	g := paperPage(800, 500, 255)
	// An outline drawn clean through the line and on past it, as between BE and WHO on
	// samson-and-delilah-15.
	fill(g, 395, 60, 399, 440, 0)
	if !strokeBetween(g, leftWord, rightWord, reachFor(100)) {
		t.Error("an outline crossing the gap and running on past the line was not a boundary")
	}
	line := lineFromWords([]ocrWord{leftWord, rightWord})
	if runs := line.splitWideGaps(g); len(runs) != 2 {
		t.Errorf("the stitched line stayed %d run(s), want 2", len(runs))
	}
	// Without pixels only the ratio rule is left, and 2.0x is under it.
	if runs := line.splitWideGaps(nil); len(runs) != 1 {
		t.Errorf("with no pixels the line was cut into %d runs", len(runs))
	}
}

// TestStrokeBetweenKeepsALetterOutsideItsBox is the false cut the rule has to avoid: the recognizer
// leaves a letter out of its word box (the T of "DON'T", the V of "IV. VINTER"), so there is ink in
// the gap - but a letter stays inside the line's band.
func TestStrokeBetweenKeepsALetterOutsideItsBox(t *testing.T) {
	g := paperPage(800, 500, 255)
	fill(g, 310, 200, 330, 300, 0) // a stem exactly as tall as the line
	fill(g, 300, 200, 360, 215, 0) // and its bar, touching the word box
	if strokeBetween(g, leftWord, rightWord, reachFor(100)) {
		t.Error("a letter inside the line's band was read as a boundary")
	}
}

// TestBoundaryReachBracketsTheMeasuredBands states the rule rather than the constant: word height
// is 100 px, so a stroke's overshoot in pixels is the reach in hundredths. The J of the "Petit
// Journal" masthead cut a real line at 0.07 and below; the first stitch was lost at 0.30. See
// ocrBoundaryReach and DEV/research/ocr_balloon_boundary_2026-09-25.md.
func TestBoundaryReachBracketsTheMeasuredBands(t *testing.T) {
	stroke := func(overshoot int) bool {
		g := paperPage(800, 500, 255)
		fill(g, 395, 200-overshoot, 399, 300+overshoot, 0)
		return strokeBetween(g, leftWord, rightWord, reachFor(100))
	}
	if stroke(7) {
		t.Error("a glyph overshooting the line by 7% of its height (the Petit Journal J) was a boundary")
	}
	if !stroke(30) {
		t.Error("an outline running 30% of the line height past it (the first stitch lost) was not a boundary")
	}
	// One side is not enough: a descender runs below the line and nowhere above it.
	g := paperPage(800, 500, 255)
	fill(g, 395, 200, 399, 400, 0)
	if strokeBetween(g, leftWord, rightWord, reachFor(100)) {
		t.Error("a stroke running past the line on one side only was a boundary")
	}
	if ocrBoundaryReach <= 0.07 || ocrBoundaryReach >= 0.30 {
		t.Errorf("ocrBoundaryReach = %v, want it strictly between the measured failures 0.07 and 0.30", ocrBoundaryReach)
	}
}

// TestStrokeBetweenFollowsASlantedOutline: a balloon's side is rarely vertical, and a path that
// had to stay in one column would miss every curved outline.
func TestStrokeBetweenFollowsASlantedOutline(t *testing.T) {
	g := paperPage(800, 500, 255)
	for y := 150; y < 350; y++ {
		x := 340 + (y-150)/2 // one pixel right every two rows
		fill(g, x, y, x+2, y+1, 0)
	}
	if !strokeBetween(g, leftWord, rightWord, reachFor(100)) {
		t.Error("a slanted outline was not followed across the gap")
	}
}

// TestStrokeBetweenReadsLightLetteringOnDarkGround: the paper is whatever surrounds the words, so a
// caption in white on black has black paper and a white rule is the ink.
func TestStrokeBetweenReadsLightLetteringOnDarkGround(t *testing.T) {
	g := paperPage(800, 500, 10)
	fill(g, 395, 60, 399, 440, 245)
	if !strokeBetween(g, leftWord, rightWord, reachFor(100)) {
		t.Error("a light rule on a dark ground was not a boundary")
	}
}

// TestStrokeBetweenReadsRightToLeft: the words arrive right to left, as synth-rtl-layout's do.
func TestStrokeBetweenReadsRightToLeft(t *testing.T) {
	g := paperPage(800, 500, 255)
	fill(g, 395, 60, 399, 440, 0)
	if !strokeBetween(g, rightWord, leftWord, reachFor(100)) {
		t.Error("a right-to-left pair missed the outline between its words")
	}
}

// TestStrokeBetweenNeedsEvidence: no picture, no strip, a window the picture cannot hold, or a
// reach that rounds to nothing - each is the absence of evidence, and that never cuts.
func TestStrokeBetweenNeedsEvidence(t *testing.T) {
	g := paperPage(800, 500, 255)
	fill(g, 395, 0, 399, 500, 0)
	if strokeBetween(nil, leftWord, rightWord, reachFor(100)) {
		t.Error("a missing picture was a boundary")
	}
	touching := ocrWord{x0: 300, y0: 200, x1: 500, y1: 300}
	if strokeBetween(g, leftWord, touching, reachFor(100)) {
		t.Error("two touching boxes have no gap, and were cut")
	}
	atTheTop := func(w ocrWord) ocrWord { w.y0, w.y1 = 0, 100; return w }
	if strokeBetween(g, atTheTop(leftWord), atTheTop(rightWord), reachFor(100)) {
		t.Error("a line against the picture's top edge cannot show a stroke above it, and was cut")
	}
	if strokeBetween(g, leftWord, rightWord, reachFor(6)) {
		t.Error("a reach that rounds to zero pixels decided anything")
	}
}

// TestParseTSVSplitsBalloonsStitchedAtANarrowGap is the end-to-end shape of samson-and-delilah-15:
// two balloons side by side, every line of the pair stitched into one recognizer line with the two
// texts 2.5 word heights apart - under ocrMaxWordGapRatio, so the ratio alone reads one plate across
// both balloons and both outlines. The pixels between the words decide it.
func TestParseTSVSplitsBalloonsStitchedAtANarrowGap(t *testing.T) {
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t800\t400\t-1\t\n"
	for i, row := range [][2]string{{"one", "one"}, {"two", "two"}, {"three", "three"}} {
		y := strconv.Itoa(100 + 30*i)
		n := strconv.Itoa(i + 1)
		tsv += "4\t1\t1\t1\t" + n + "\t0\t100\t" + y + "\t450\t20\t-1\t\n" +
			"5\t1\t1\t1\t" + n + "\t1\t100\t" + y + "\t90\t20\t95\tLeft\n" +
			"5\t1\t1\t1\t" + n + "\t2\t200\t" + y + "\t100\t20\t95\t" + row[0] + "\n" +
			"5\t1\t1\t1\t" + n + "\t3\t350\t" + y + "\t90\t20\t95\tRight\n" +
			"5\t1\t1\t1\t" + n + "\t4\t450\t" + y + "\t100\t20\t95\t" + row[1] + "\n"
	}
	page := paperPage(800, 400, 255)
	fill(page, 312, 70, 315, 200, 0) // the left balloon's right side
	fill(page, 335, 70, 338, 200, 0) // the right balloon's left side

	merged, err := parseTSV([]byte(tsv), ocrMinLineConf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Blocks) != 1 {
		t.Fatalf("without pixels: %d block(s), want the one merged plate this test exists to fix", len(merged.Blocks))
	}
	res, err := parseTSV([]byte(tsv), ocrMinLineConf, page)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, b := range res.Blocks {
		got = append(got, b.Text)
	}
	want := []string{"Left one Left two Left three", "Right one Right two Right three"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("blocks = %q, want %q - one plate per balloon", got, want)
	}
}

// TestParseTSVParksWhatAStrokeCutOff is atomicwar0401: a speck of artwork beside a balloon is read as
// the word "A" at the head of the balloon's last line, the outline between them cuts it off, and
// the page's headline spans every column, so the speck lands in the balloon's column at the height
// of the line it was cut from. Handed to the clustering there it broke the balloon into two plates;
// it cannot be a plate itself, so it is parked and the translatability gate records it.
func TestParseTSVParksWhatAStrokeCutOff(t *testing.T) {
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t1000\t1000\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t27\t19\t861\t31\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t27\t19\t400\t31\t95\tONLY A STRONG AMERICA\n" +
		"5\t1\t1\t1\t1\t2\t500\t19\t388\t31\t95\tCAN PREVENT\n" +
		"4\t1\t2\t1\t1\t0\t660\t785\t234\t17\t-1\t\n" +
		"5\t1\t2\t1\t1\t1\t660\t785\t42\t17\t95\tTHE\n" +
		"5\t1\t2\t1\t1\t2\t712\t785\t104\t17\t95\tKREMLIN,\n" +
		"5\t1\t2\t1\t1\t3\t824\t785\t70\t17\t95\tTHOSE\n" +
		"4\t1\t2\t1\t2\t0\t618\t818\t197\t19\t-1\t\n" +
		"5\t1\t2\t1\t2\t1\t618\t818\t2\t4\t90\tA\n" +
		"5\t1\t2\t1\t2\t2\t652\t820\t103\t17\t95\tRUSSKIES\n" +
		"5\t1\t2\t1\t2\t3\t765\t820\t50\t16\t95\tWILL\n"
	page := paperPage(1000, 1000, 255)
	fill(page, 630, 700, 636, 950, 0) // the balloon's outline, between the speck and the lettering

	res, err := parseTSV([]byte(tsv), ocrMinLineConf, page)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, b := range res.Blocks {
		got = append(got, b.Text)
	}
	want := []string{"ONLY A STRONG AMERICA CAN PREVENT", "THE KREMLIN, THOSE RUSSKIES WILL"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("blocks = %q, want %q - the balloon is one plate", got, want)
	}
	var recorded bool
	for _, d := range res.Dropped {
		recorded = recorded || (d.Text == "A" && d.Gate == gateTranslatable)
	}
	if !recorded {
		t.Errorf("the parked speck left no record in %+v", res.Dropped)
	}
}
