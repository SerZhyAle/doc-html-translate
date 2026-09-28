package ocr

import (
	"fmt"
	"strings"
	"testing"
)

// The fixtures and the assertions here are the anchored rescue admission of OCR-PIPELINE amendment
// 1.6 (ticket 29): a rescue rung may keep a line under ocrRescueLineConf when its pass holds at
// least two floor-clearing 4-letter-run anchors, one at the candidate's own type size, and the
// clustering tolerates a late row of the unordered sparse rung. The numbers are the lab scenes'
// own - displayHeadlineOverBody and adjacentBalloons above are the poster and the balloon scene as
// the app reads them.

// rescueTSV renders fixture lines as a TSV page: one line each, one word each, so a line's mean
// confidence is its word's.
func rescueTSV(f []fixtureLine) string {
	var b strings.Builder
	b.WriteString("level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n")
	b.WriteString("1\t1\t0\t0\t0\t0\t0\t0\t2000\t2000\t-1\t\n")
	for _, l := range f {
		b.WriteString("4\t1\t1\t1\t1\t0\t")
		// box: x0 y0 x1 y1 -> left top width height
		b.WriteString(fmt.Sprintf("%d\t%d\t%d\t%d\t-1\t\n", l.x0, l.y0, l.x1-l.x0, l.y1-l.y0))
		b.WriteString(fmt.Sprintf("5\t1\t1\t1\t1\t1\t%d\t%d\t%d\t%d\t%g\t%s\n",
			l.x0, l.y0, l.x1-l.x0, l.y1-l.y0, l.conf, l.text))
	}
	return b.String()
}

// The unanchored admission of 2026-09-25 split the poster's body into two overlapping plates the
// moment it admitted the out-of-order ОБ ЗЛОМ. With the corroboration guard and the unordered late
// row joined into the open cluster, the whole poster reads as two plates: the headline the ticket
// is about, and the body whole.
func TestRescueAdmissionKeepsThePosterWhole(t *testing.T) {
	res, err := parseTSV([]byte(rescueTSV(displayHeadlineOverBody)), ocrRescueLineConf, nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, b := range res.Blocks {
		texts = append(texts, b.Text)
	}
	want := []string{"ЗАЧЕМ ТРАХАТЬСЯ:", "МЫ ЖЕ ЛЮДИ, МОЖЕМ ОБ ЗЛОМ ПРОСТО ПОГОВОРИТЬ"}
	if len(texts) != len(want) {
		t.Fatalf("plates = %q, want %q", texts, want)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Errorf("plate %d = %q, want %q", i, texts[i], want[i])
		}
	}
	if n := len(res.Dropped); n != 0 {
		t.Errorf("dropped = %d entries, want 0 (the admitted lines leave the record)", n)
	}
}

// The same scene without the admission is the baseline: the sub-floor lines are dropped and the
// headline plate never forms.
func TestOrdinaryRescueFloorKeepsTheOldBehaviour(t *testing.T) {
	res, err := parseTSV([]byte(rescueTSV(displayHeadlineOverBody)), ocrRescueLineConf, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, b := range res.Blocks {
		texts = append(texts, b.Text)
	}
	// The body's out-of-order row is dropped by the floor, so the baseline walk never sees a late
	// row here: the body stays one plate and the headline plate never forms. The record names the
	// two losses.
	want := []string{"ТРАХАТЬСЯ:", "МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ"}
	if len(texts) != len(want) {
		t.Fatalf("plates = %q, want %q", texts, want)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Errorf("plate %d = %q, want %q", i, texts[i], want[i])
		}
	}
	dropped := map[string]bool{}
	for _, d := range res.Dropped {
		if d.Floor == ocrRescueLineConf && d.Gate == gateConfidence {
			dropped[d.Text] = true
		}
	}
	for _, lost := range []string{"ЗАЧЕМ", "ОБ ЗЛОМ"} {
		if !dropped[lost] {
			t.Errorf("dropped record misses %q (%v)", lost, dropped)
		}
	}
}

// The balloon scene is the case the corroboration guard exists for: the pass's only confident line
// (МОТ ЕУЕМ, 81.4) read an English scene with Russian data and is itself debris. One anchor vouches
// for nothing, so АВОЧТ ТН1$? stays dropped and the plate count does not move.
func TestRescueAdmissionNeedsTwoAnchors(t *testing.T) {
	rus := []fixtureLine{
		{116, 123, 390, 151, 67.2, "АКЕ УОЧ УВЕ"},
		{116, 175, 362, 203, 62.0, "АВОЧТ ТН1$?"},
		{118, 259, 298, 287, 81.4, "МОТ ЕУЕМ"},
		{117, 311, 308, 339, 0.0, "$ЫСНТЕУ."},
	}
	res, err := parseTSV([]byte(rescueTSV(rus)), ocrRescueLineConf, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 1 || res.Blocks[0].Text != "МОТ ЕУЕМ" {
		t.Fatalf("blocks = %v, want the single МОТ ЕУЕМ plate", res.Blocks)
	}
	found := false
	for _, d := range res.Dropped {
		if d.Text == "АВОЧТ ТН1$?" && d.Floor == ocrRescueLineConf && d.Gate == gateConfidence {
			found = true
		}
	}
	if !found {
		t.Error("АВОЧТ ТН1$? is not in the discard record; the admission ran on a single anchor")
	}
}

// Each condition of the admission refuses its own near-miss: a candidate under ocrRescueAnchorConf,
// a candidate whose letter run is short, and a candidate no anchor's type size covers. The anchors
// themselves always clear the floor with the run, so every refusal below is the candidate's alone.
func TestRescueAdmissionConditions(t *testing.T) {
	cases := []struct {
		name string
		cand string
		f    []fixtureLine
	}{
		{
			name: "confidence under 47",
			cand: "CANDIDATE",
			f: []fixtureLine{
				{0, 0, 300, 100, 92.0, "ANCHORED"},
				{0, 200, 300, 300, 90.0, "SECOND ANCHOR"},
				{0, 400, 300, 500, 46.9, "CANDIDATE"},
			},
		},
		{
			name: "letter run under four",
			cand: "АВ",
			f: []fixtureLine{
				{0, 0, 300, 100, 92.0, "ANCHORED"},
				{0, 200, 300, 300, 90.0, "SECOND ANCHOR"},
				{0, 400, 300, 500, 60.0, "АВ"},
			},
		},
		{
			name: "type size no anchor covers",
			cand: "CANDIDATE",
			f: []fixtureLine{
				{0, 0, 300, 100, 92.0, "ANCHORED"},
				{0, 200, 300, 300, 90.0, "SECOND ANCHOR"},
				{0, 400, 300, 700, 60.0, "CANDIDATE"},
			},
		},
	}
	for _, c := range cases {
		res, err := parseTSV([]byte(rescueTSV(c.f)), ocrRescueLineConf, nil, true, false)
		if err != nil {
			t.Fatal(err)
		}
		// The two anchors may cluster together - they are same-size, same-column lines - but no
		// plate may carry the candidate, and the record must name it.
		for _, b := range res.Blocks {
			if strings.Contains(b.Text, c.cand) {
				t.Errorf("%s: a plate carries the candidate: %q", c.name, b.Text)
			}
		}
		found := false
		for _, d := range res.Dropped {
			if d.Text == c.cand && d.Floor == ocrRescueLineConf && d.Gate == gateConfidence {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: candidate %q not in the discard record", c.name, c.cand)
		}
	}
}

// A late row of the unordered sparse rung that fits inside the open cluster's band joins it instead
// of splitting it: the poster's body must stay one plate (OCR-PIPELINE amendment 1.6 B).
func TestUnorderedLateRowJoinsTheOpenCluster(t *testing.T) {
	lines := fixtureLines([]fixtureLine{
		{0, 0, 300, 100, 95, "FIRSTLY"},
		{0, 200, 300, 300, 95, "SECONDLY"},
		{0, 120, 300, 200, 90, "LATELY"},
	})
	// Arrival order: the late row comes after the second line and lands inside the band the two of
	// them span - its pitch is negative, its gap deeper than the ordinary negative tolerance, its
	// type 80 px against the cluster's 100 px median ink.
	for _, l := range lines {
		l.unordered = true
	}
	blocks := clusterLinesRecording(lines, ocrMinLineConf, 0, 0, nil)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1 (the late row joins)", len(blocks))
	}
	if blocks[0].Text != "FIRSTLY SECONDLY LATELY" {
		t.Errorf("text = %q, want the walk order with the late row appended", blocks[0].Text)
	}
	if blocks[0].Y0 != 0 || blocks[0].Y1 != 300 {
		t.Errorf("box = (%d,%d), want (0,300)", blocks[0].Y0, blocks[0].Y1)
	}
}

// The same late row without the unordered mark splits, exactly as the ordered passes always behaved.
func TestOrderedLateRowStillSplitsTheCluster(t *testing.T) {
	lines := fixtureLines([]fixtureLine{
		{0, 0, 300, 100, 95, "FIRSTLY"},
		{0, 200, 300, 300, 95, "SECONDLY"},
		{0, 120, 300, 200, 90, "LATELY"},
	})
	blocks := clusterLinesRecording(lines, ocrMinLineConf, 0, 0, nil)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (the walk is unchanged for ordered passes)", len(blocks))
	}
}

// The late-row tolerance is not a blanket merge: a row below the band, in another column, or at
// another type size still starts a new plate.
func TestUnorderedLateRowOutsideTheBandStillSplits(t *testing.T) {
	cases := []struct {
		name string
		f    []fixtureLine
	}{
		{
			name: "below the band",
			f: []fixtureLine{
				{0, 0, 300, 100, 95, "FIRSTLY"},
				{0, 200, 300, 300, 95, "SECONDLY"},
				{0, 700, 300, 780, 90, "LATELY"},
			},
		},
		{
			name: "another column",
			f: []fixtureLine{
				{0, 0, 300, 100, 95, "FIRSTLY"},
				{0, 200, 300, 300, 95, "SECONDLY"},
				{600, 120, 900, 200, 90, "LATELY"},
			},
		},
		{
			name: "another type size",
			f: []fixtureLine{
				{0, 0, 300, 100, 95, "FIRSTLY"},
				{0, 200, 300, 300, 95, "SECONDLY"},
				{0, 120, 300, 320, 90, "LATELY"},
			},
		},
	}
	for _, c := range cases {
		lines := fixtureLines(c.f)
		for _, l := range lines {
			l.unordered = true
		}
		blocks := clusterLinesRecording(lines, ocrMinLineConf, 0, 0, nil)
		if len(blocks) != 2 {
			t.Errorf("%s: blocks = %d, want 2", c.name, len(blocks))
		}
	}
}
