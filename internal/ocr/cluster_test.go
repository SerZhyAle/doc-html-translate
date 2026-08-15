package ocr

import (
	"strings"
	"testing"
)

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

// displayHeadlineOverBody: poster-display-type-on-flat-colour as the app actually reads it - the
// grey rendition at PSM 11 with rus data, in the 2x staged space, rows in the order tesseract
// returned them (PSM 11 is "sparse text, in no particular order", and it did return one line out of
// reading order). ЗАЧЕМ and ОБ ЗЛОМ sit under the rescue floor and are dropped by it, which is why
// three of the poster's nine lines never reach a plate.
//
// The step from the headline to the next recognized line is 335 px; the step between two body lines
// a reader takes as one sentence is 381 px. No pitch bound cuts in the right place here. The type
// sizes do: 281 px of ink against a 155 px median.
var displayHeadlineOverBody = []fixtureLine{
	{76, 79, 780, 366, 69.2, "ЗАЧЕМ"},
	{74, 415, 1328, 696, 80.7, "ТРАХАТЬСЯ:"},
	{79, 750, 521, 905, 96.1, "МЫ ЖЕ"},
	{76, 1131, 456, 1302, 92.6, "ЛЮДИ,"},
	{79, 1319, 538, 1472, 95.9, "МОЖЕМ"},
	{79, 1694, 498, 1836, 73.9, "ОБ ЗЛОМ"},
	{80, 1509, 477, 1658, 87.2, "ПРОСТО"},
	{80, 1872, 644, 2009, 95.0, "ПОГОВОРИТЬ"},
}

// scrollCaption: the 19 hand-drawn line boxes of samson-and-delilah-03-scroll, the corpus's widest
// within-group spread (ink 23-34 px, worst line 1.42x the group's own median). One caption, one
// sentence, one plate - and the scene that decides how far ocrTypeSizeRatio may be tightened.
var scrollCaption = []fixtureLine{
	{103, 40, 469, 74, 95, "LONG AGO IN ISRAEL"},
	{130, 74, 441, 102, 95, "THE SMALL TOWN OF"},
	{130, 109, 469, 134, 95, "ASHKELON WAS RULED"},
	{130, 142, 469, 165, 95, "BY A PHILISTINE KING,"},
	{132, 172, 469, 196, 95, "WHO WAS CRUEL AND"},
	{135, 203, 469, 227, 95, "HEARTLESS. ALL THE"},
	{132, 235, 469, 258, 95, "DANITES HATED HIM."},
	{130, 265, 469, 289, 95, "HIS TAXES WERE HEAVY,"},
	{130, 297, 469, 320, 95, "HIS PUNISHMENTS SEVERE!"},
	{130, 328, 469, 353, 95, "SAMSON, JUDGE OF THE"},
	{130, 362, 463, 385, 95, "DANITES, WAS CHOSEN"},
	{130, 392, 469, 415, 95, "TO PROTECT HIS"},
	{130, 423, 469, 446, 95, "DOWNTRODDEN PEOPLE."},
	{130, 453, 469, 477, 95, "IN ISRAEL WAS DELILAH,"},
	{130, 484, 447, 510, 95, "IN ISRAEL WAS THEIR"},
	{130, 516, 381, 540, 95, "GREAT LOVE, IN ISRAEL"},
	{134, 547, 381, 570, 95, "WAS THE DEATH OF----"},
	{130, 574, 457, 607, 95, "SAMSON AND"},
	{164, 613, 402, 645, 95, "DELILAH!"},
}

// accountsWindow: the eight rows of test_doc/accounts.jpg as the app reads them (640x563, the image
// is under the upscale floor so these are its own pixels). One type size, one column, an even
// pitch - nothing in the typography separates a row from the next - so before the coverage rule the
// six list rows landed in one plate spanning [13,99,555,553]: 0.6829 of the picture, with its own
// line boxes accounting for only 0.6608 of the height it spanned. That plate is what the sweep
// photographed as a wall of words with the screenshot no longer visible behind it.
//
// The **boxes are the recognizer's, unaltered** - they are what the rule is measured on. The row
// *texts* are not: the screenshot is a private account list, and its rows name real people who did
// not put their names in this repository. Each one keeps its shape (a leading glyph the recognizer
// made of the row's avatar, a name, a role) so the "nothing is lost" assertion still means what it
// says, and the geometry is untouched.
var accountsWindow = []fixtureLine{
	{15, 17, 290, 38, 90, "Your family group members"},
	{15, 51, 339, 66, 90, "View and manage your family group. Learn more ©"},
	{13, 99, 527, 151, 90, "Se) Given Family Family manager"},
	{22, 182, 467, 231, 90, "i) Second Family Parent"},
	{15, 262, 479, 310, 90, "6 Third Family Member"},
	{13, 341, 479, 393, 90, "wi Fourth Family Member"},
	{13, 423, 555, 474, 90, "te Fifth Family Supervised member"},
	{15, 505, 479, 553, 90, "© Sixth Family Member"},
}

// TestClusterLinesReleasesAPlateThatCoversItsPage: a cluster that is both too big and too loose is
// released into its own lines, so the reader gets the rows rather than a slab over the window. The
// assertion is on what a reader loses, not on a plate count: every recognized word still reaches a
// plate, and no plate covers more than a row.
func TestClusterLinesReleasesAPlateThatCoversItsPage(t *testing.T) {
	blocks := clusterLines(fixtureLines(accountsWindow), ocrMinLineConf, 640, 563)
	if len(blocks) < 6 {
		t.Fatalf("blocks = %d, want the six rows released (plus the header)", len(blocks))
	}
	img := float64(640 * 563)
	for _, b := range blocks {
		if cover := float64((b.X1 - b.X0) * (b.Y1 - b.Y0)); cover/img > ocrMaxPlateCoverage {
			t.Errorf("plate %q covers %.4f of the image, want <= %.2f", b.Text, cover/img, ocrMaxPlateCoverage)
		}
	}
	// Nothing recognized may be lost on the way: the released lines carry their own text, and the
	// rows are short enough that re-filtering them individually would have dropped some.
	var joined string
	for _, b := range blocks {
		joined += b.Text + " "
	}
	for _, want := range []string{"Given Family", "Second Family", "Sixth Family", "Supervised member"} {
		if !strings.Contains(joined, want) {
			t.Errorf("released plates lost %q", want)
		}
	}
}

// TestReleaseOversizedNeedsBothConditions is the price side: each half of the rule alone would
// release a scene the corpus says is one plate. The numbers are the measured ones - see
// ocrMaxPlateCoverage / ocrMinPlateLineFill.
func TestReleaseOversizedNeedsBothConditions(t *testing.T) {
	// Too big but tightly packed: samson-and-delilah-03-scroll's caption covers 0.6087 of its crop
	// in the hand-drawn annotation and its lines fill 0.7921 of that height.
	tight := []LineBox{{0, 0, 100, 79}, {0, 80, 100, 158}}
	if got := releaseOversized(0, 0, 100, 200, []string{"a", "b"}, tight, 120, 170); got != nil {
		t.Errorf("a tightly packed caption was released into %d lines", len(got))
	}
	// Loose but small: synth-uniform-paper's three body lines fill 0.6667 of their box and cover
	// 0.2141 of the page.
	loose := []LineBox{{0, 0, 100, 20}, {0, 40, 100, 60}, {0, 80, 100, 100}}
	if got := releaseOversized(0, 0, 100, 100, []string{"a", "b", "c"}, loose, 500, 500); got != nil {
		t.Errorf("an ordinary paragraph was released into %d lines", len(got))
	}
	// Both, which is the defect.
	if got := releaseOversized(0, 0, 100, 100, []string{"a", "b", "c"}, loose, 120, 120); len(got) != 3 {
		t.Errorf("released %d plates, want 3", len(got))
	}
}

// TestClusterLinesKeepsOneBalloonWhole is the splitting half of the pair: all-caps lettering must
// not be split by measuring its leading against the height of its own ink.
func TestClusterLinesKeepsOneBalloonWhole(t *testing.T) {
	blocks := clusterLines(fixtureLines(balloonOnPanel), ocrRescueLineConf, 1120, 840)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1 (one balloon is one plate)", len(blocks))
	}
	if want := "WELL, THAT IS ONE WAY TO SOLVE IT!"; blocks[0].Text != want {
		t.Errorf("text = %q, want %q", blocks[0].Text, want)
	}
}

// TestClusterLinesSurvivesAnOutlineArtefact: a balloon outline recognized as a tall "|" inside a
// line must not end that balloon. The line box is the union of its words, so the artefact sets it
// for the whole line - 74 px beside two 26 px words - and ocrTypeSizeRatio then reads two type
// sizes where a reader sees one. Found in the extension edition, whose engine produces the
// artefact this one does not (DEV/research/ocrlab/2026-08-15__extension-parity-run.md); the rule
// is shared, so both sides carry the fixture.
func TestClusterLinesSurvivesAnOutlineArtefact(t *testing.T) {
	build := func(withWords bool) []*ocrLine {
		type row struct {
			fixtureLine
			wordH []int
		}
		rows := []row{
			{fixtureLine{116, 123, 390, 151, 96.4, "ARE YOU SURE"}, []int{28, 28, 28}},
			{fixtureLine{116, 175, 362, 203, 95.7, "ABOUT THIS?"}, []int{28, 28}},
			{fixtureLine{78, 259, 298, 333, 90.1, "| NOT EVEN"}, []int{74, 26, 26}},
			{fixtureLine{117, 311, 308, 337, 95.9, "SLIGHTLY."}, []int{26}},
		}
		out := make([]*ocrLine, 0, len(rows))
		for _, r := range rows {
			o := &ocrLine{x0: r.x0, y0: r.y0, x1: r.x1, y1: r.y1, confSum: r.conf, confN: 1}
			o.text.WriteString(r.text)
			if withWords {
				o.wordH = r.wordH
			}
			out = append(out, o)
		}
		return out
	}
	blocks := clusterLines(build(true), ocrRescueLineConf, 1240, 600)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (one plate per balloon)", len(blocks))
	}
	if got := blocks[1].Text; got != "| NOT EVEN SLIGHTLY." {
		t.Errorf("second plate = %q, want the whole balloon", got)
	}

	// With no word heights the line box is all there is, so the old, stricter behaviour stands -
	// the fix must not loosen a case it has no evidence about.
	if blocks := clusterLines(build(false), ocrRescueLineConf, 1240, 600); len(blocks) != 3 {
		t.Errorf("without word heights: blocks = %d, want 3 (the line box splits it)", len(blocks))
	}
}

// TestTrimOutlierWords: grouping the balloon back together is not enough - the artefact still pulls
// the line box, and so the plate, out over the protected outline. Measured on the extension edition
// of synth-adjacent-balloons: 148 px of damage before the type-size fix, 160 px after it.
func TestTrimOutlierWords(t *testing.T) {
	line := func(words ...ocrWord) *ocrLine {
		l := &ocrLine{x0: words[0].x0, y0: words[0].y0, x1: words[0].x1, y1: words[0].y1}
		for _, w := range words[1:] {
			l.x0, l.y0 = min(l.x0, w.x0), min(l.y0, w.y0)
			l.x1, l.y1 = max(l.x1, w.x1), max(l.y1, w.y1)
		}
		for _, w := range words {
			l.words = append(l.words, w)
			l.wordH = append(l.wordH, w.y1-w.y0)
		}
		return l
	}

	l := line(
		ocrWord{78, 259, 88, 333, "|"},
		ocrWord{116, 265, 210, 291, "NOT"},
		ocrWord{220, 265, 298, 291, "EVEN"},
	)
	l.trimOutlierWords()
	if l.inkX0 != 116 || l.inkY0 != 265 || l.inkX1 != 298 || l.inkY1 != 291 {
		t.Errorf("drawn box = [%d,%d %d,%d], want the lettering's own [116,265 298,291]", l.inkX0, l.inkY0, l.inkX1, l.inkY1)
	}
	// The line's own box is untouched: every clustering decision still reads it, so trimming can
	// never change what reaches the page.
	if l.x0 != 78 || l.y1 != 333 {
		t.Errorf("the untrimmed box moved: [%d,%d %d,%d]", l.x0, l.y0, l.x1, l.y1)
	}

	// Ordinary punctuation is short, so it is not an artefact and the box is untouched.
	c := line(
		ocrWord{116, 265, 210, 291, "NOT"},
		ocrWord{212, 281, 220, 293, ","},
	)
	c.trimOutlierWords()
	if c.inkX1 != 220 || c.inkY1 != 293 {
		t.Errorf("punctuation was trimmed: box = [%d,%d %d,%d]", c.inkX0, c.inkY0, c.inkX1, c.inkY1)
	}

	// A line that is nothing but the artefact keeps its box - no lettering to shrink towards.
	r := line(ocrWord{78, 259, 88, 333, "|"})
	r.trimOutlierWords()
	if r.inkX0 != 78 || r.inkY1 != 333 {
		t.Errorf("a rule-only line lost its box: [%d,%d %d,%d]", r.inkX0, r.inkY0, r.inkX1, r.inkY1)
	}
}

// TestClusterLinesKeepsAdjacentBalloonsApart is the merging half: the fix for the split above must
// not be paid for by a plate that spans two balloons.
func TestClusterLinesKeepsAdjacentBalloonsApart(t *testing.T) {
	blocks := clusterLines(fixtureLines(adjacentBalloons), ocrRescueLineConf, 1240, 600)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (one plate per balloon)", len(blocks))
	}
	for i, want := range []string{"ARE YOU SURE ABOUT THIS?", "NOT EVEN SLIGHTLY."} {
		if blocks[i].Text != want {
			t.Errorf("block %d = %q, want %q", i, blocks[i].Text, want)
		}
	}
}

// TestClusterLinesSeparatesDisplayTypeFromBody: a headline and the body under it are two texts even
// when they sit within a page's own line pitch of each other. Pitch cannot see it - on this poster
// the headline's step is smaller than one inside the body - so the type size has to.
// TestClusterLinesSeparatesDisplayTypeFromBody: a headline and the body under it are two texts even
// when they sit within a page's own line pitch of each other. Pitch cannot see it - on this poster
// the headline's step is smaller than one inside the body - so the type size has to.
//
// ЗАЧЕМ is under the rescue floor and stays there: the 2026-08-15 re-measurement of that floor
// (DEV/research/ocr_rescue_floor_2026-08-15.md) found no value and no length rule that recovers it
// without putting a plate of transliterated debris on a Cyrillic poster read with English data.
func TestClusterLinesSeparatesDisplayTypeFromBody(t *testing.T) {
	blocks := clusterLines(fixtureLines(displayHeadlineOverBody), ocrRescueLineConf, 1920, 2560)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (headline and body are two plates)", len(blocks))
	}
	if want := "ТРАХАТЬСЯ:"; blocks[0].Text != want {
		t.Errorf("headline = %q, want %q", blocks[0].Text, want)
	}
	if want := "МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ"; blocks[1].Text != want {
		t.Errorf("body = %q, want %q", blocks[1].Text, want)
	}
	// The body's box is what the lab scores against the annotation's body group, [38,375,344,1005]
	// once scaled back down. Pinned in the staged space this fixture is measured in.
	if b := blocks[1]; b.X0 != 76 || b.Y0 != 750 || b.X1 != 644 || b.Y1 != 2009 {
		t.Errorf("body box = [%d,%d,%d,%d], want [76,750,644,2009]", b.X0, b.Y0, b.X1, b.Y1)
	}
}

// TestClusterLinesKeepsACaptionWhoseLinesVary is the price side of the rule above: a real caption's
// own lines are not all one height, and the corpus's widest spread must survive it untouched.
func TestClusterLinesKeepsACaptionWhoseLinesVary(t *testing.T) {
	blocks := clusterLines(fixtureLines(scrollCaption), ocrRescueLineConf, 520, 720)
	if len(blocks) == 0 {
		t.Fatal("no plates")
	}
	// The assertion is about the size rule, so it is stated as "no break inside the caption's
	// varying lines" rather than as a plate count: the caption's last line stands 39 px from the one
	// above it against a 31 px page pitch, and the pitch bound separates it here with or without the
	// size rule. That break is this fixture's, not the rule's - it is human line geometry standing in
	// for a recognizer's, and the scene's own lab result is what governs.
	first := blocks[0]
	if first.Y0 != 40 || first.Y1 != 607 {
		t.Errorf("first plate spans y %d-%d, want 40-607 (lines 1-18 in one plate)", first.Y0, first.Y1)
	}
	if !strings.HasPrefix(first.Text, "LONG AGO IN ISRAEL") || !strings.HasSuffix(first.Text, "SAMSON AND") {
		t.Errorf("first plate = %q, want the caption from its 34 px first line to its 33 px last but one", first.Text)
	}
}

// TestSameTypeSizeBracketsTheMeasuredBands: the ratio has to admit the widest spread one text shows
// on its own and reject the narrowest step between two texts. Both numbers come from the corpus's
// hand-drawn line boxes - see ocrTypeSizeRatio for where each was measured.
func TestSameTypeSizeBracketsTheMeasuredBands(t *testing.T) {
	// samson-and-delilah-03-scroll: a 34 px line inside a caption whose median is 24 px.
	if !sameTypeSize(34, 24) {
		t.Error("the corpus's widest within-caption spread (1.42x) counts as a different type size")
	}
	// poster-display-type-on-flat-colour, as recognized: 281 px of display ink over a 155 px body.
	if sameTypeSize(281, 155) {
		t.Error("display type over body text (1.81x) counts as the same type size")
	}
	if !sameTypeSize(0, 155) || !sameTypeSize(155, 0) {
		t.Error("an unmeasured height ended a plate")
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

// TestDroppedLinesRecordTheFloor: the confidence floor is the one place the overlay silently
// decides a reader does not get words the engine read, and it has to be visible. Mirrors the
// extension's ocr-cluster.test.mjs case; the numbers are the measured staging of
// poster-display-type-on-flat-colour with rus data on the sparse rung.
func TestDroppedLinesRecordTheFloor(t *testing.T) {
	lines := fixtureLines([]fixtureLine{
		{38, 40, 390, 183, 69.2, "ЗАЧЕМ"},
		{37, 207, 664, 348, 80.7, "ТРАХАТЬСЯ:"},
		{39, 375, 261, 453, 73.9, "ОБ ЗЛОМ"},
		{0, 0, 10, 10, 12.0, ""}, // no text: not a loss, and not a record
	})

	var dropped []DroppedLine
	for _, l := range lines {
		if l.text.Len() > 0 && !keepLine(l, ocrRescueLineConf) {
			dropped = append(dropped, DroppedLine{
				Text: strings.TrimSpace(l.text.String()), Conf: l.meanConf(),
				X0: l.x0, Y0: l.y0, X1: l.x1, Y1: l.y1,
			})
		}
	}
	if len(dropped) != 2 || dropped[0].Text != "ЗАЧЕМ" || dropped[1].Text != "ОБ ЗЛОМ" {
		t.Fatalf("dropped = %+v, want the two lines under the floor", dropped)
	}
	if dropped[0].Conf != 69.2 || dropped[0].X1 != 390 {
		t.Errorf("the record must carry the confidence and the box: %+v", dropped[0])
	}

	// Every line is kept, dropped or textless - never both and never neither, because both sides
	// ask keepLine. A record derived from a second copy of the predicate could drift silently.
	kept, empty := 0, 0
	for _, l := range lines {
		switch {
		case l.text.Len() == 0:
			empty++
		case keepLine(l, ocrRescueLineConf):
			kept++
		}
	}
	if kept != 1 || empty != 1 || kept+empty+len(dropped) != len(lines) {
		t.Errorf("kept=%d empty=%d dropped=%d over %d lines", kept, empty, len(dropped), len(lines))
	}
}
