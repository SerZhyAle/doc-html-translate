package ocr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCJKJoinSharedCases runs the fixture extension/test/ocr-text.test.mjs runs, so the two editions
// join the same words and the same plate lines (OCR-PIPELINE amendment 1.8).
func TestCJKJoinSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "ocr_cjk_join_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Words []struct {
			About string              `json:"about"`
			Words [][]json.RawMessage `json:"words"`
			Want  string              `json:"want"`
		} `json:"words"`
		Lines []struct {
			About string   `json:"about"`
			Lines []string `json:"lines"`
			Want  string   `json:"want"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Words) == 0 || len(fx.Lines) == 0 {
		t.Fatal("the fixture carries no cases")
	}
	for _, c := range fx.Words {
		words := make([]ocrWord, 0, len(c.Words))
		for _, raw := range c.Words {
			var w ocrWord
			if len(raw) != 5 {
				t.Fatalf("%s: a word is [text, x0, y0, x1, y1], got %d fields", c.About, len(raw))
			}
			for i, dst := range []any{&w.text, &w.x0, &w.y0, &w.x1, &w.y1} {
				if err := json.Unmarshal(raw[i], dst); err != nil {
					t.Fatalf("%s: %v", c.About, err)
				}
			}
			words = append(words, w)
		}
		if got := joinLineWords(words); got != c.Want {
			t.Errorf("joinLineWords = %q, want %q (%s)", got, c.Want, c.About)
		}
	}
	for _, c := range fx.Lines {
		if got := joinPlateLines(c.Lines); got != c.Want {
			t.Errorf("joinPlateLines(%q) = %q, want %q (%s)", c.Lines, got, c.Want, c.About)
		}
	}
}

// TestLetterRunReadsTheRecognizersWords pins what the rescue admission (1.6 A) reads once CJK lines
// are joined (1.8): the longest letter run of any one recognized word, not of the joined text. A
// Japanese line of one- and two-character tokens stays short of ocrRescueAnchorRun, as it was when
// the admission was measured; a line with no words is read from its text.
func TestLetterRunReadsTheRecognizersWords(t *testing.T) {
	words := []ocrWord{
		{x0: 0, y0: 0, x1: 40, y1: 30, text: "コモ"}, {x0: 42, y0: 0, x1: 60, y1: 30, text: "ナ"},
		{x0: 62, y0: 0, x1: 90, y1: 30, text: "市"}, {x0: 92, y0: 0, x1: 120, y1: 30, text: "長"},
	}
	l := lineFromWords(words)
	if got := l.text.String(); got != "コモナ市長" {
		t.Fatalf("text = %q, want the joined line", got)
	}
	if got := longestLetterRun(l.text.String()); got != 5 {
		t.Fatalf("the joined text's run = %d, want 5 (the case this guards against)", got)
	}
	if got := l.letterRun(); got != 2 || got >= ocrRescueAnchorRun {
		t.Errorf("letterRun = %d, want 2 - the longest recognized word, under the anchor run", got)
	}
	var bare ocrLine
	bare.text.WriteString("PROSTO WORD")
	if got := bare.letterRun(); got != 6 {
		t.Errorf("a line without words reads its text: letterRun = %d, want 6", got)
	}
}

// TestParseTSVJoinsCJKWords runs the join where it lives, in the TSV parse: the line text a plate is
// built from carries the Japanese line unspaced and the Korean word space, and a line the split cuts
// rebuilds each run's text by the same rule.
func TestParseTSVJoinsCJKWords(t *testing.T) {
	tsv := strings.Join([]string{
		"level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext",
		"1\t1\t0\t0\t0\t0\t0\t0\t2000\t1000\t-1\t",
		"4\t1\t1\t1\t1\t0\t100\t100\t190\t30\t-1\t",
		"5\t1\t1\t1\t1\t1\t100\t100\t40\t30\t90\tコモ",
		"5\t1\t1\t1\t1\t2\t142\t100\t18\t30\t90\tナ",
		"5\t1\t1\t1\t1\t3\t162\t100\t28\t30\t90\t市",
		"5\t1\t1\t1\t1\t4\t192\t100\t28\t30\t90\t長",
		"4\t1\t2\t1\t1\t0\t100\t400\t220\t30\t-1\t",
		"5\t1\t2\t1\t1\t1\t100\t400\t24\t30\t90\t코",
		"5\t1\t2\t1\t1\t2\t126\t400\t24\t30\t90\t모",
		"5\t1\t2\t1\t1\t3\t152\t400\t24\t30\t90\t나",
		"5\t1\t2\t1\t1\t4\t192\t400\t24\t30\t90\t시",
		"5\t1\t2\t1\t1\t5\t218\t400\t24\t30\t90\t장",
		// A stitched line: the Chinese run and a second one far past OCR_MAX_WORD_GAP_RATIO.
		"4\t1\t3\t1\t1\t0\t100\t700\t1700\t30\t-1\t",
		"5\t1\t3\t1\t1\t1\t100\t700\t30\t30\t90\t市",
		"5\t1\t3\t1\t1\t2\t132\t700\t30\t30\t90\t长",
		"5\t1\t3\t1\t1\t3\t1700\t700\t30\t30\t90\t我",
		"5\t1\t3\t1\t1\t4\t1732\t700\t30\t30\t90\t宣",
	}, "\n")
	res, err := parseTSV([]byte(tsv), ocrMinLineConf, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, b := range res.Blocks {
		got = append(got, b.Text)
	}
	for _, want := range []string{"コモナ市長", "코모나 시장", "市长", "我宣"} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
			}
		}
		if !found {
			t.Errorf("no plate reads %q; plates: %q", want, got)
		}
	}
}
