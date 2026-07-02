package ocr

import (
	"strings"
	"testing"
)

func TestParseTSV(t *testing.T) {
	// One line, two words -> one plate boxed to the line.
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t200\t100\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t10\t20\t120\t18\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t10\t20\t50\t18\t90\tHello\n" +
		"5\t1\t1\t1\t1\t2\t65\t20\t60\t18\t88\tworld\n"
	res, err := parseTSV([]byte(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 200 || res.Height != 100 {
		t.Fatalf("dims = %dx%d, want 200x100", res.Width, res.Height)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(res.Blocks))
	}
	b := res.Blocks[0]
	if b.Text != "Hello world" {
		t.Errorf("text = %q, want %q", b.Text, "Hello world")
	}
	if b.X0 != 10 || b.Y0 != 20 || b.X1 != 130 || b.Y1 != 38 {
		t.Errorf("bbox = (%d,%d,%d,%d), want (10,20,130,38)", b.X0, b.Y0, b.X1, b.Y1)
	}
	if b.LineH != 18 {
		t.Errorf("lineH = %d, want 18", b.LineH)
	}
}

func TestParseTSVSkipsBlankLine(t *testing.T) {
	// A line whose only word is blank must not produce a plate.
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t100\t100\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t0\t0\t50\t50\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t0\t0\t50\t50\t0\t \n"
	res, err := parseTSV([]byte(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 0 {
		t.Fatalf("blocks = %d, want 0 (blank text)", len(res.Blocks))
	}
}

func TestParseTSVDropsLowConfNoise(t *testing.T) {
	// A heading and a body line separated by a low-confidence "line" the engine hallucinated
	// from a figure. The noise line (mean conf 20 < ocrMinLineConf) must be dropped, so no plate
	// covers its band and the heading font is not inflated by its oversized box.
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t200\t300\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t20\t10\t100\t20\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t20\t10\t100\t20\t95\tHeading\n" +
		"4\t1\t2\t1\t1\t0\t40\t120\t120\t40\t-1\t\n" +
		"5\t1\t2\t1\t1\t1\t40\t120\t60\t40\t20\thello\n" +
		"5\t1\t2\t1\t1\t2\t105\t120\t55\t40\t20\tworld\n" +
		"4\t1\t3\t1\t1\t0\t10\t240\t180\t20\t-1\t\n" +
		"5\t1\t3\t1\t1\t1\t10\t240\t90\t20\t90\tHello\n" +
		"5\t1\t3\t1\t1\t2\t105\t240\t80\t20\t90\tthere\n"
	res, err := parseTSV([]byte(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (heading + body, noise dropped)", len(res.Blocks))
	}
	for _, b := range res.Blocks {
		if strings.Contains(b.Text, "hello world") {
			t.Errorf("low-conf noise leaked into a plate: %q", b.Text)
		}
		if b.Y0 <= 140 && b.Y1 >= 140 { // 140 is inside the noise band [120,160]
			t.Errorf("a plate covers the figure band: bbox y=%d..%d", b.Y0, b.Y1)
		}
	}
	if res.Blocks[0].Text != "Heading" || res.Blocks[0].LineH != 20 {
		t.Errorf("heading = %q lineH=%d, want %q lineH=20", res.Blocks[0].Text, res.Blocks[0].LineH, "Heading")
	}
	if res.Blocks[1].Text != "Hello there" {
		t.Errorf("body = %q, want %q", res.Blocks[1].Text, "Hello there")
	}
}

func TestParseTSVMergesContiguousLines(t *testing.T) {
	// Three vertically-adjacent lines (small gaps) - even across paragraph boundaries the engine
	// invents - must merge into one plate rather than splinter into several.
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t200\t300\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t10\t10\t180\t20\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t10\t10\t80\t20\t95\tAlpha\n" +
		"5\t1\t1\t1\t1\t2\t95\t10\t90\t20\t95\tbeta\n" +
		"4\t1\t1\t2\t1\t0\t10\t34\t180\t20\t-1\t\n" +
		"5\t1\t1\t2\t1\t1\t10\t34\t80\t20\t95\tgamma\n" +
		"5\t1\t1\t2\t1\t2\t95\t34\t90\t20\t95\tdelta\n" +
		"4\t1\t1\t2\t2\t0\t10\t58\t180\t20\t-1\t\n" +
		"5\t1\t1\t2\t2\t1\t10\t58\t150\t20\t95\tepsilon\n"
	res, err := parseTSV([]byte(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1 (contiguous lines merged)", len(res.Blocks))
	}
	b := res.Blocks[0]
	if b.Text != "Alpha beta gamma delta epsilon" {
		t.Errorf("text = %q, want %q", b.Text, "Alpha beta gamma delta epsilon")
	}
	if b.Y0 != 10 || b.Y1 != 78 {
		t.Errorf("bbox y = %d..%d, want 10..78", b.Y0, b.Y1)
	}
}

func TestParseTSVSplitsOnGap(t *testing.T) {
	// A heading and a body far apart (a figure / whitespace between them) must become two plates,
	// each boxed to its own line - so the gap between them is left uncovered.
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t200\t300\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t20\t10\t100\t20\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t20\t10\t100\t20\t90\tHeading\n" +
		"4\t1\t1\t2\t1\t0\t10\t200\t180\t20\t-1\t\n" +
		"5\t1\t1\t2\t1\t1\t10\t200\t60\t20\t88\tThis\n" +
		"5\t1\t1\t2\t1\t2\t80\t200\t50\t20\t88\tbody\n"
	res, err := parseTSV([]byte(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (heading and body split by the gap)", len(res.Blocks))
	}
	if res.Blocks[0].Text != "Heading" || res.Blocks[0].Y1 != 30 {
		t.Errorf("para1 = %q y1=%d, want %q y1=30", res.Blocks[0].Text, res.Blocks[0].Y1, "Heading")
	}
	if res.Blocks[1].Text != "This body" || res.Blocks[1].Y0 != 200 {
		t.Errorf("para2 = %q y0=%d, want %q y0=200", res.Blocks[1].Text, res.Blocks[1].Y0, "This body")
	}
}

func TestPercentStyle(t *testing.T) {
	// font-size = pct(LineH, w) * fontFitFactor = (20/200*100) * 0.92 = 9.20cqw.
	got := percentStyle(Block{X0: 10, Y0: 20, X1: 110, Y1: 60, LineH: 20}, 200, 100)
	want := "left:5.00%;top:20.00%;width:50.00%;min-height:40.00%;font-size:9.20cqw"
	if got != want {
		t.Errorf("percentStyle = %q, want %q", got, want)
	}
}

func TestPercentStyleZeroDims(t *testing.T) {
	if got := percentStyle(Block{X1: 10, Y1: 10}, 0, 0); got != "" {
		t.Errorf("percentStyle with zero dims = %q, want empty", got)
	}
}

func TestIsTranslatable(t *testing.T) {
	cases := map[string]bool{
		// keep: real translatable text
		"Get your food, drink & duty free delivered before the trolley.": true,
		"AUTO MIETEN. BIS ZU 25% SPAREN.":                                true,
		"CHAPTER ONE":                                                    true,
		"Section 12.3 - Results and Summary":                             true,
		"Привет мир, это тест":                                           true,
		"日本語":                                                            true, // short CJK phrase
		// drop: nothing to translate
		"":                             false,
		"   ":                          false,
		"XS":                           false, // < 5 letters
		"Se of":                        false, // < 5 letters
		"25":                           false, // digits only
		"25%":                          false,
		"1234 5678":                    false,
		"————— ——— ~~ :":               false, // symbols only
		"BCDFG":                        false, // no vowels
		"https://example.com/x":        false, // address
		"www.sixt.de":                  false,
		"user@example.com":             false,
		"sixt.de":                      false,
		"C:\\Users\\serzh":             false, // path
		"gr [u : &o Se A JETZT MIETEN": false, // mishmash
	}
	for in, want := range cases {
		if got := isTranslatable(in); got != want {
			t.Errorf("isTranslatable(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestIsExternal(t *testing.T) {
	cases := map[string]bool{
		"images/a.png":            false,
		"./pdf_images/p1.jpg":     false,
		"http://x/a.png":          true,
		"https://x/a.png":         true,
		"//x/a.png":               true,
		"data:image/png;base64,x": true,
	}
	for src, want := range cases {
		if got := isExternal(src); got != want {
			t.Errorf("isExternal(%q) = %v, want %v", src, got, want)
		}
	}
}
