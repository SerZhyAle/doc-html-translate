package ocr

import "testing"

// The line boxes below are not invented: they are what tesseract returned for the lab's two
// grouping scenes on 2026-08-11, in the upscaled space clusterLines works in (the scenes are
// small enough that prepareForOCR doubles them). Both scenes are read by the grey rescue pass -
// the colour pass finds nothing on saturated artwork - so the confidences are that pass's.
//
// Regenerate by dumping the level-4/5 rows of
// `tesseract <scene>.png stdout --psm 3 -l eng -c tessedit_create_tsv=1` on a greyscale copy.

type fixtureLine struct {
	x0, y0, x1, y1 int
	conf           float64
	text           string
}

func fixtureLines(f []fixtureLine) []*ocrLine {
	out := make([]*ocrLine, 0, len(f))
	for _, l := range f {
		o := &ocrLine{x0: l.x0, y0: l.y0, x1: l.x1, y1: l.y1, confSum: l.conf, confN: 1}
		o.text.WriteString(l.text)
		out = append(out, o)
	}
	return out
}

// balloonOnPanel: one speech balloon, three lines of all-caps lettering drawn with 36 px leading
// (72 px here). The ink boxes are 29-34 px because caps have no descenders, and the gaps between
// them are 38-43 px - so the pre-2026-08-11 test, which compared the gap against 1.2 x the median
// ink height (34.8 px), tore one balloon into three plates.
var balloonOnPanel = []fixtureLine{
	{644, 184, 930, 218, 96.0, "WELL, THAT IS"},
	{645, 256, 890, 285, 96.7, "ONE WAY TO"},
	{646, 328, 838, 357, 93.6, "SOLVE IT!"},
	{156, 126, 1034, 826, 0.0, ""}, // the panel itself: no text, dropped before clustering
}

// adjacentBalloons: two balloons, two lines each. The pitch inside a balloon is 52 px and the step
// across to the next balloon is 84 px - the margin that has to survive any loosening of the gate,
// or this scene's two plates collapse into one that crosses an unrelated reading group.
var adjacentBalloons = []fixtureLine{
	{116, 123, 390, 151, 96.4, "ARE YOU SURE"},
	{116, 175, 362, 203, 95.7, "ABOUT THIS?"},
	{118, 259, 298, 287, 96.4, "NOT EVEN"},
	{117, 311, 308, 339, 95.9, "SLIGHTLY."},
}

// TestClusterLinesKeepsOneBalloonWhole is the splitting half of the pair: all-caps lettering must
// not be split by measuring its leading against the height of its own ink.
func TestClusterLinesKeepsOneBalloonWhole(t *testing.T) {
	blocks := clusterLines(fixtureLines(balloonOnPanel), ocrRescueLineConf)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1 (one balloon is one plate)", len(blocks))
	}
	if want := "WELL, THAT IS ONE WAY TO SOLVE IT!"; blocks[0].Text != want {
		t.Errorf("text = %q, want %q", blocks[0].Text, want)
	}
}

// TestClusterLinesKeepsAdjacentBalloonsApart is the merging half: the fix for the split above must
// not be paid for by a plate that spans two balloons.
func TestClusterLinesKeepsAdjacentBalloonsApart(t *testing.T) {
	blocks := clusterLines(fixtureLines(adjacentBalloons), ocrRescueLineConf)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (one plate per balloon)", len(blocks))
	}
	for i, want := range []string{"ARE YOU SURE ABOUT THIS?", "NOT EVEN SLIGHTLY."} {
		if blocks[i].Text != want {
			t.Errorf("block %d = %q, want %q", i, blocks[i].Text, want)
		}
	}
}

// TestMedianLinePitchIgnoresSectionBreaks: the estimate must reject a step that is a section break
// rather than leading, or a page holding a heading and one distant paragraph makes that single
// jump its "typical" pitch and the two merge into one plate.
func TestMedianLinePitchIgnoresSectionBreaks(t *testing.T) {
	heading := fixtureLines([]fixtureLine{
		{20, 10, 120, 30, 90, "Heading"},
		{10, 200, 190, 220, 88, "This body"},
	})
	if _, ok := medianLinePitch(heading, 20); ok {
		t.Error("a 190 px step over 20 px ink counted as a line pitch")
	}

	body := fixtureLines([]fixtureLine{
		{10, 10, 190, 30, 95, "Alpha beta"},
		{10, 34, 190, 54, 95, "gamma delta"},
		{10, 58, 160, 78, 95, "epsilon"},
	})
	got, ok := medianLinePitch(body, 20)
	if !ok || got != 24 {
		t.Errorf("pitch = %d (ok=%v), want 24", got, ok)
	}
}

// TestMedianLinePitchNeedsASharedColumn: two columns side by side never measure a pitch off each
// other, so the estimate stays the leading of the text rather than the distance between columns.
func TestMedianLinePitchNeedsASharedColumn(t *testing.T) {
	columns := fixtureLines([]fixtureLine{
		{10, 10, 100, 30, 95, "left one"},
		{300, 20, 390, 40, 95, "right one"},
	})
	if _, ok := medianLinePitch(columns, 20); ok {
		t.Error("lines that share no column contributed a pitch")
	}
}
