package ocr

// Regression tests for ticket 67 and ticket E0128: two speech balloons drawn close enough to touch
// are stitched into one recognizer line, and the touching outlines are read as tokens of that line,
// so the stroke test finds nothing between the words - the boundary runs INSIDE the boundary token's
// box. The split therefore also cuts before a word whose own box stands more than ocrTypeSizeRatio
// above the line's median word height and which carries no letter or digit: an outline glyph that
// tall reaches into a neighbouring row, and no lettering of one line does. A tall word that carries
// a letter is lettering and is never cut by its height (OCR-PIPELINE amendment 1.10 A); the Superman
// page's own boundary tokens carry letters, which makes that page a known limit of the rule (1.10 C).

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSplitWideGapsCutsWhereTheOutlineIsTheToken reads the shared fixture
// tests/testdata/ocr_balloon_stitch_cases.json: the word boxes the desktop engine returned for the
// touching balloons of a public-service Superman page (uncut, the known limit), the same row with
// the outline read as a bare pipe (cut), two honest lines with a tall word that carries a letter or
// only digits (uncut) and a real line of one size; the extension's ocr-cluster.test.mjs reads the
// same file.
func TestSplitWideGapsCutsWhereTheOutlineIsTheToken(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "ocr_balloon_stitch_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Words []struct {
				Text string  `json:"text"`
				Conf float64 `json:"conf"`
				BBox struct {
					X0, Y0, X1, Y1 int
				} `json:"bbox"`
			} `json:"words"`
			Want []string `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range fx.Cases {
		words := make([]ocrWord, 0, len(tc.Words))
		for _, w := range tc.Words {
			words = append(words, ocrWord{
				x0: w.BBox.X0, y0: w.BBox.Y0, x1: w.BBox.X1, y1: w.BBox.Y1,
				text: w.Text, conf: w.Conf, hasConf: true,
			})
		}
		var got []string
		for _, r := range lineFromWords(words).splitWideGaps(nil) {
			got = append(got, strings.TrimSpace(r.text.String()))
		}
		if len(got) != len(tc.Want) {
			t.Errorf("%s: runs = %q, want %q", tc.Name, got, tc.Want)
			continue
		}
		for i := range got {
			if got[i] != tc.Want[i] {
				t.Errorf("%s: run %d = %q, want %q", tc.Name, i, got[i], tc.Want[i])
			}
		}
	}
}

// TestParseTSVKeepsBalloonWordsWhereTheOutlinesAreLetters replays the desktop engine's own TSV for
// the whole Superman page (internal/ocr/testdata/superman_stitch.tsv) through the real parse. Before
// the tall-token cut the page's middle panel came back as one plate carrying both balloons' words
// interleaved ("GEE, TH-THANKS, “J OON'T DEPEND SUPERMAN: (T'S A J] ON LUCK! ..."); the cut
// separated it while it fired on every tall word. Since amendment 1.10 A it fires only on a token
// with no letter or digit, and this page's two boundary tokens (“J, A) carry letters, so the middle
// panel is the known limit of the rule (1.10 C) and its first row stays one stitched run. What holds
// either way - and is what this test pins - is that no word of either balloon is lost on the way to
// the plates; the page's title banner, the WHEW balloon and the caption's missing lines are the grey
// sweep's business - its floor and merge were measured on this page
// (DEV/research/ocr_comic_coverage_2026-09-29.md).
func TestParseTSVKeepsBalloonWordsWhereTheOutlinesAreLetters(t *testing.T) {
	tsv, err := os.ReadFile(filepath.Join("testdata", "superman_stitch.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	// Paper at the balloons' luma and no stroke drawn: on the real picture no stroke test finds
	// anything between the words of either row (ticket E0128 measured the strips), so the cut
	// points are the ratio rule and the tall-token rule alone.
	page := paperPage(1010, 1440, 220)

	res, err := parseTSV(tsv, ocrMinLineConf, page, false, false)
	if err != nil {
		t.Fatal(err)
	}
	var plates []string
	for _, b := range res.Blocks {
		plates = append(plates, strings.ToUpper(b.Text))
	}
	all := strings.Join(plates, "\n")
	for _, w := range []string{"GEE,", "SUPERMAN:", "LUCKY", "THING", "OON'T", "DEPEND", "LUCK!", "AROUND", "NEXT", "CAREFUL"} {
		if !strings.Contains(all, w) {
			t.Errorf("the balloons' word %q reached no plate: %q", w, plates)
		}
	}
}

// TestGreySweepTrigger pins the sweep's trigger: it spends its pass only when a region the layout
// analysis marked unread stands where the accepted plates leave it mostly uncovered. With nothing
// to look at it hands the plates back untouched - both short circuits run before any engine call,
// so a nil frame proves they are reachable.
func TestGreySweepTrigger(t *testing.T) {
	kept := []Block{{X0: 0, Y0: 0, X1: 500, Y1: 100}}
	rects := blockRects(kept)
	if unreadOutside(nil, rects) {
		t.Error("no unread regions, the sweep must not fire")
	}
	if unreadOutside([]image.Rectangle{image.Rect(0, 0, 100, 100)}, rects) {
		t.Error("a region the plate covers must not fire the sweep")
	}
	if !unreadOutside([]image.Rectangle{image.Rect(600, 0, 700, 100)}, rects) {
		t.Error("an unplated region must fire the sweep")
	}
	got, dropped := greySweep(context.Background(), "", nil, "", "", 0, kept, nil)
	if len(got) != 1 || len(dropped) != 0 || got[0].Text != kept[0].Text {
		t.Errorf("greySweep with no unread regions moved the plates: %v", got)
	}
}
