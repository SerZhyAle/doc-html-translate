package ocr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepairPipeMisreadsSharedCases runs the fixture extension/test/ocr-text.test.mjs runs, so the
// two editions repair the same tokens (OCR-PIPELINE amendment 1.5). This is the token mechanic
// only - the guards live in the flush, where the word boxes are.
func TestRepairPipeMisreadsSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "ocr_pipe_repair_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Line  string `json:"line"`
			Want  string `json:"want"`
			About string `json:"about"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		if got := repairPipeMisreads(c.Line, nil); got != c.Want {
			t.Errorf("repairPipeMisreads(%q) = %q, want %q (%s)", c.Line, got, c.Want, c.About)
		}
	}
}

// pipeWords builds the only word boxes the repair reads - the bare pipe tokens', in order.
func pipeWords(boxes ...[4]int) []ocrWord {
	out := make([]ocrWord, 0, len(boxes))
	for _, b := range boxes {
		out = append(out, ocrWord{x0: b[0], y0: b[1], x1: b[2], y1: b[3], text: "|"})
	}
	return out
}

// TestClusterLinesRepairsThePipeMisread is the school-text page of 2026-09-28: a translatable
// paragraph in which the recognizer read the capital I's as pipes. The gate passes on the raw text,
// the flush rewrites every pipe that is neither an outline nor a column of a grid, and the plate
// carries the subjects back. The pipe boxes are the page's own: bare strokes at whatever x the
// sentences put them, none of them repeated across lines.
func TestClusterLinesRepairsThePipeMisread(t *testing.T) {
	lines := []*ocrLine{
		fixtureLineWithWords(100, "I am Andrew. | am a pupil of the", pipeWords([4]int{172, 200, 176, 224})),
		fixtureLineWithWords(156, "5th form. | get up at seven o'clock.", pipeWords([4]int{415, 256, 419, 280})),
		fixtureLineWithWords(212, "dress. | make a bed. My friend come", pipeWords([4]int{78, 312, 82, 336})),
	}
	blocks := clusterLines(lines, ocrMinLineConf, 0, 0)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	want := "I am Andrew. I am a pupil of the 5th form. I get up at seven o'clock. dress. I make a bed. My friend come"
	if blocks[0].Text != want {
		t.Errorf("block = %q, want %q", blocks[0].Text, want)
	}
}

// fixtureLineWithWords is a fixtureLine plus the line's own word boxes (the repair reads the pipe
// tokens' boxes; the other words' boxes never enter its decisions, so they stay unset).
func fixtureLineWithWords(y0 int, text string, words []ocrWord) *ocrLine {
	l := &ocrLine{x0: 21, y0: y0, x1: 560, y1: y0 + 24, confSum: 95, confN: 1}
	l.text.WriteString(text)
	l.words = words
	for _, w := range words {
		l.wordH = append(l.wordH, w.y1-w.y0)
	}
	return l
}

// TestClusterLinesKeepsATableColumnsBars: a table's separators survive recognition as a grid -
// each row's pipes repeat the column centres - and the flush spares exactly those, while a stray
// pipe off the grid is still repaired.
func TestClusterLinesKeepsATableColumnsBars(t *testing.T) {
	lines := []*ocrLine{
		fixtureLineWithWords(100, "Item | Qty | Price", pipeWords([4]int{118, 200, 122, 224}, [4]int{238, 200, 242, 224})),
		fixtureLineWithWords(156, "Bread | 2 | 3.50", pipeWords([4]int{119, 256, 123, 280}, [4]int{241, 256, 245, 280})),
		fixtureLineWithWords(212, "Milk | 1 | 2.80", pipeWords([4]int{121, 312, 125, 336}, [4]int{239, 312, 243, 336})),
	}
	blocks := clusterLines(lines, ocrMinLineConf, 0, 0)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	want := "Item | Qty | Price Bread | 2 | 3.50 Milk | 1 | 2.80"
	if blocks[0].Text != want {
		t.Errorf("block = %q, want %q", blocks[0].Text, want)
	}
}

// TestClusterLinesNeverRepairsAnUngatedCluster: pipe garbage - the balloon outlines and table rules
// the recognizer reads as bars - fails the translatability gate on its raw text, and the repair
// must not resurrect it as lettering.
func TestClusterLinesNeverRepairsAnUngatedCluster(t *testing.T) {
	lines := fixtureLines([]fixtureLine{
		{116, 123, 390, 151, 95.0, "|"},
		{116, 175, 362, 203, 95.0, "|"},
		{118, 259, 298, 287, 95.0, "|"},
		{117, 311, 308, 339, 95.0, "|"},
	})
	var dropped []DroppedLine
	blocks := clusterLinesRecording(lines, ocrMinLineConf, 0, 0, &dropped)
	if len(blocks) != 0 {
		t.Errorf("blocks = %d, want 0 (pipe garbage stays refused)", len(blocks))
	}
	if len(dropped) != len(lines) {
		t.Fatalf("dropped = %d, want %d", len(dropped), len(lines))
	}
	for _, d := range dropped {
		if d.Gate != gateTranslatable {
			t.Errorf("line %q dropped by gate %q, want %q", d.Text, d.Gate, gateTranslatable)
		}
		if strings.Contains(d.Text, "I") {
			t.Errorf("the record keeps the raw text, got %q", d.Text)
		}
	}
}

// TestClusterLinesGuardsThePipeRepairOnTokenHeight: the height guard is the trim's own comparison -
// a letter-sized pipe is a misread I and is repaired, a pipe taller than the text's own type is the
// outline the trim handles and keeps its bar.
func TestClusterLinesGuardsThePipeRepairOnTokenHeight(t *testing.T) {
	build := func(pipeH int) []*ocrLine {
		l := &ocrLine{x0: 64, y0: 100, x1: 400, y1: 124, confSum: 95, confN: 1}
		l.text.WriteString("he is here. | today")
		l.words = pipeWords([4]int{200, 100 - pipeH + 24, 202, 124})
		l.wordH = []int{24, 24, 24, pipeH, 24}
		return []*ocrLine{l}
	}
	if got := clusterLines(build(24), ocrMinLineConf, 0, 0); len(got) != 1 || got[0].Text != "he is here. I today" {
		t.Errorf("a letter-sized pipe survived: %v", got)
	}
	if got := clusterLines(build(74), ocrMinLineConf, 0, 0); len(got) != 1 || got[0].Text != "he is here. | today" {
		t.Errorf("a tall outline pipe was repaired: %v", got)
	}
}

// TestClusterLinesRepairsReleasedLines: the coverage release splits a gated cluster into one plate
// per line, and those lines share the repair. The geometry is accountsWindow's, which releases;
// the rows carry no word boxes, so the repair runs unguarded - the field repro's shape.
func TestClusterLinesRepairsReleasedLines(t *testing.T) {
	lines := fixtureLines([]fixtureLine{
		{15, 17, 290, 38, 90, "Your family group members"},
		{15, 51, 339, 66, 90, "View and manage your family group. Learn more"},
		{13, 99, 527, 151, 90, "Se) | Given Family Family manager"},
		{22, 182, 467, 231, 90, "i) | Second | Family Parent"},
		{15, 262, 479, 310, 90, "6 Third Family Member"},
		{13, 341, 479, 393, 90, "wi Fourth Family Member"},
		{13, 423, 555, 474, 90, "te Fifth Family Supervised member"},
		{15, 505, 479, 553, 90, "(c) Sixth Family Member"},
	})
	blocks := clusterLines(lines, ocrMinLineConf, 640, 563)
	if len(blocks) < 6 {
		t.Fatalf("blocks = %d, want the rows released", len(blocks))
	}
	var joined string
	for _, b := range blocks {
		joined += b.Text + " "
	}
	if !strings.Contains(joined, "Se) I Given Family Family manager") {
		t.Errorf("the released row's pipe was not repaired: %q", joined)
	}
	if !strings.Contains(joined, "i) I Second I Family Parent") {
		t.Errorf("the released row's other pipes were not repaired: %q", joined)
	}
}
