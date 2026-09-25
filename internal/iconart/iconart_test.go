package iconart

import (
	"image/color"
	"strings"
	"testing"
)

func TestParsePathReadsMaterialShorthand(t *testing.T) {
	// "c0 1.1.9 2 2 2" is c 0,1.1 .9,2 2,2 - a second dot starts the next number.
	segs, err := parsePath("M4 4v16c0 1.1.9 2 2 2h12l-2-2z")
	if err != nil {
		t.Fatal(err)
	}
	var ops strings.Builder
	for _, s := range segs {
		ops.WriteByte(s.op)
	}
	if ops.String() != "MLCLLZ" {
		t.Fatalf("ops = %s", ops.String())
	}
	c := segs[2].pts
	if c[0] != (point{4, 21.1}) || c[1] != (point{4.9, 22}) || c[2] != (point{6, 22}) {
		t.Errorf("cubic = %v", c)
	}
	if got := segs[4].pts[0]; got != (point{16, 20}) {
		t.Errorf("relative lineto after h ended at %v", got)
	}
}

func TestParsePathRefusesArcs(t *testing.T) {
	if _, err := parsePath("M0 0a2 2 0 0 1 4 0z"); err == nil {
		t.Fatal("an arc was accepted")
	}
}

func TestICORoundTripKeepsEveryFrame(t *testing.T) {
	raw, err := Output{Path: "x.ico", Images: markFrames(appICOSizes)}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	frames, err := DecodeICO(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != len(appICOSizes) {
		t.Fatalf("%d frames, want %d", len(frames), len(appICOSizes))
	}
	for i, f := range frames {
		if f.Bounds().Dx() != appICOSizes[i] || f.Bounds().Dy() != appICOSizes[i] {
			t.Errorf("frame %d is %v, want %d px", i, f.Bounds(), appICOSizes[i])
		}
	}
}

// The one tone of ICON-RENDER rule 9 holds 3:1 on the light and the dark Explorer menu.
func TestOneToneHoldsOnBothMenus(t *testing.T) {
	for _, bg := range []color.NRGBA{{0xFF, 0xFF, 0xFF, 0xFF}, {0xF9, 0xF9, 0xF9, 0xFF}, {0x2B, 0x2B, 0x2B, 0xFF}, {0x2C, 0x2C, 0x2C, 0xFF}} {
		if r := Contrast(OneTon, bg); r < 3 {
			t.Errorf("#808080 on %v is %.2f:1", bg, r)
		}
	}
}

// At 16 px the brackets are still drawn: the tip row crosses sheet, bracket, sheet, bracket, sheet.
func TestSmallMarkKeepsTheBrackets(t *testing.T) {
	img := Mark(16, 16, 16, Plated)
	var runs []bool // true = navy-ish
	for x := 3; x < 13; x++ {
		c := img.RGBAAt(x, 10)
		navy := Contrast(c, Sheet) >= 3
		if len(runs) == 0 || runs[len(runs)-1] != navy {
			runs = append(runs, navy)
		}
	}
	if len(runs) < 5 {
		t.Errorf("row 10 of the 16 px mark alternates %d times, want sheet/bracket/sheet/bracket/sheet", len(runs))
	}
}

// The forms Windows plates itself cut the code out to transparency, so the plate shows through.
func TestOnPlatformFormCutsTheCodeOut(t *testing.T) {
	img := Mark(48, 48, 48, OnPlatform)
	if a := img.RGBAAt(0, 0).A; a != 0 {
		t.Errorf("corner alpha %d, want no plate", a)
	}
	if a := img.RGBAAt(15, 29).A; a != 0 {
		t.Errorf("bracket tip alpha %d, want a cut-out", a)
	}
	if c := img.RGBAAt(18, 12); c != (color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}) {
		t.Errorf("sheet pixel %v, want white", c)
	}
}
