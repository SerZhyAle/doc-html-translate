package ocr

import "testing"

func TestParseTSV(t *testing.T) {
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t200\t100\t-1\t\n" +
		"2\t1\t1\t0\t0\t0\t10\t20\t120\t40\t-1\t\n" +
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
	if b.X0 != 10 || b.Y0 != 20 || b.X1 != 130 || b.Y1 != 60 {
		t.Errorf("bbox = (%d,%d,%d,%d), want (10,20,130,60)", b.X0, b.Y0, b.X1, b.Y1)
	}
	if b.LineH != 18 {
		t.Errorf("lineH = %d, want 18", b.LineH)
	}
}

func TestParseTSVSkipsEmptyBlocks(t *testing.T) {
	// A block with only blank words must not produce a plate.
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t100\t100\t-1\t\n" +
		"2\t1\t1\t0\t0\t0\t0\t0\t50\t50\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t0\t0\t50\t50\t0\t \n"
	res, err := parseTSV([]byte(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 0 {
		t.Fatalf("blocks = %d, want 0 (blank text)", len(res.Blocks))
	}
}

func TestPercentStyle(t *testing.T) {
	got := percentStyle(Block{X0: 10, Y0: 20, X1: 110, Y1: 60, LineH: 20}, 200, 100)
	want := "left:5.00%;top:20.00%;width:50.00%;min-height:40.00%;font-size:10.00cqw"
	if got != want {
		t.Errorf("percentStyle = %q, want %q", got, want)
	}
}

func TestPercentStyleZeroDims(t *testing.T) {
	if got := percentStyle(Block{X1: 10, Y1: 10}, 0, 0); got != "" {
		t.Errorf("percentStyle with zero dims = %q, want empty", got)
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
