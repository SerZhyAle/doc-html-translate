package ocr

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

// paint fills a rectangle of an image with one colour.
func paint(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// A plate must never come back as the photographic negative of what it covers. The block box
// tesseract returns is tight around the lettering, and heavy display capitals fill more of that
// box than the paper between them does - so the median over the box lands on the ink, and the
// pair used to be assigned the wrong way round. Reported from use on a poster: the word МОЖЕМ,
// black on cream, was overlaid as cream on near-black.
//
// The two cases are one test because the fix must not buy the poster at the cost of the balloon:
// body text inside a balloon is the minority of its own box and has to keep the reading it
// already had.
func TestBlockColorsOrientByWhatSurroundsTheBlock(t *testing.T) {
	paper := color.RGBA{235, 226, 205, 255}
	inkc := color.RGBA{40, 35, 30, 255}

	tests := []struct {
		name       string
		build      func() *image.RGBA
		block      Block
		wantBg     color.RGBA
		wantInk    color.RGBA
		wantReason string
	}{
		{
			name: "display capitals cover most of their own box",
			build: func() *image.RGBA {
				img := image.NewRGBA(image.Rect(0, 0, 300, 300))
				paint(img, 0, 0, 300, 300, paper)
				// One heavy word: 70% of the block's area is stroke.
				paint(img, 100, 130, 200, 170, inkc)
				paint(img, 105, 125, 195, 175, inkc)
				return img
			},
			block:      Block{X0: 100, Y0: 125, X1: 200, Y1: 175, LineH: 50},
			wantBg:     paper,
			wantInk:    inkc,
			wantReason: "the cream around the word is the paper, whatever the box's own majority is",
		},
		{
			name: "body text is the minority of its own box",
			build: func() *image.RGBA {
				img := image.NewRGBA(image.Rect(0, 0, 300, 300))
				paint(img, 0, 0, 300, 300, color.RGBA{30, 30, 30, 255}) // dark artwork
				paint(img, 60, 60, 240, 240, paper)                     // a balloon on it
				for y := 100; y < 108; y++ {                            // three thin lines of type
					paint(img, 90, y, 210, y+1, inkc)
				}
				for y := 130; y < 138; y++ {
					paint(img, 90, y, 210, y+1, inkc)
				}
				for y := 160; y < 168; y++ {
					paint(img, 90, y, 210, y+1, inkc)
				}
				return img
			},
			block:      Block{X0: 90, Y0: 100, X1: 210, Y1: 168, LineH: 8},
			wantBg:     paper,
			wantInk:    inkc,
			wantReason: "the balloon's interior is still the paper, and the reading must not flip",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bg, ink, ok := blockColors(tc.build(), tc.block)
			if !ok {
				t.Fatalf("blockColors returned ok=false")
			}
			if wantBg := rgbString(tc.wantBg); bg != wantBg {
				t.Errorf("background = %s, want %s - %s", bg, wantBg, tc.wantReason)
			}
			if wantInk := rgbString(tc.wantInk); ink != wantInk {
				t.Errorf("ink = %s, want %s - %s", ink, wantInk, tc.wantReason)
			}
		})
	}
}

// The ink colour must be the colour of the strokes, not the average of the strokes and the ramp of
// antialiased pixels around them. Reported from use on a phone screenshot whose caption is
// rgb(17,17,17) on rgb(253,253,253): the plate came back rgb(61,61,61), a visible grey next to the
// black lettering it covers. The deviation test that selects ink pixels admits most of a glyph's
// edge ramp, so the mean of the selection sits between the two colours by construction, and the
// wider the ramp the further from the ink it lands.
func TestBlockColorsInkIsNotDilutedByGlyphEdges(t *testing.T) {
	paper := color.RGBA{253, 253, 253, 255}
	inkc := color.RGBA{17, 17, 17, 255}

	img := image.NewRGBA(image.Rect(0, 0, 400, 200))
	paint(img, 0, 0, 400, 200, paper)
	// Three lines of type, each stroke wrapped in a two-pixel ramp between ink and paper - the
	// antialiasing a real glyph carries, and about as much of the block's area as the stroke itself.
	for _, y := range []int{40, 90, 140} {
		paint(img, 60, y-2, 340, y+14, color.RGBA{135, 135, 135, 255})
		paint(img, 60, y, 340, y+12, inkc)
	}

	_, ink, ok := blockColors(img, Block{X0: 60, Y0: 38, X1: 340, Y1: 154, LineH: 16})
	if !ok {
		t.Fatal("blockColors returned ok=false")
	}
	if want := rgbString(inkc); ink != want {
		t.Errorf("ink = %s, want %s - the edge ramp must not pull the ink toward the paper", ink, want)
	}
}

// sameBlock compares two blocks by text and geometry. Block carries a slice of line boxes and so
// is no longer comparable with ==; nothing in these tests depends on the line list itself.
func sameBlock(a, b Block) bool {
	return a.Text == b.Text && a.X0 == b.X0 && a.Y0 == b.Y0 && a.X1 == b.X1 && a.Y1 == b.Y1 && a.LineH == b.LineH
}

func rgbString(c color.RGBA) string {
	return fmt.Sprintf("rgb(%d,%d,%d)", c.R, c.G, c.B)
}
