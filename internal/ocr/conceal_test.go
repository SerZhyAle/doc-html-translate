package ocr

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	gohtml "golang.org/x/net/html"

	"doc-html-translate/tools/ocrlab/synth"
	"doc-html-translate/tools/ocrlab/truth"
)

// The concealment-mode decision is judged on the lab's synthetic scenes (Phase 02), whose text
// geometry is exact by construction: each annotated group becomes the Block the recognizer would
// hand the overlay, so these tests need no tesseract. Only the test links the lab; no shipped
// binary does.

var synthOnce struct {
	sync.Once
	dir  string
	anns map[string]*truth.Annotation
	err  error
}

func synthScene(t *testing.T, id string) (image.Image, map[string]Block) {
	t.Helper()
	synthOnce.Do(func() {
		synthOnce.dir, synthOnce.err = os.MkdirTemp("", "ocr-conceal-synth-")
		if synthOnce.err != nil {
			return
		}
		var anns []*truth.Annotation
		_, anns, synthOnce.err = synth.Generate(synthOnce.dir)
		synthOnce.anns = map[string]*truth.Annotation{}
		for _, a := range anns {
			synthOnce.anns[a.SceneID] = a
		}
	})
	if synthOnce.err != nil {
		t.Fatal(synthOnce.err)
	}
	ann := synthOnce.anns[id]
	if ann == nil {
		t.Fatalf("no synthetic scene %q", id)
	}
	img := decodeImage(filepath.Join(synthOnce.dir, synth.Dir, id+".png"))
	if img == nil {
		t.Fatalf("%s: image not decoded", id)
	}
	blocks := map[string]Block{}
	for _, g := range ann.Groups {
		var b Block
		var hs []int
		for i, l := range g.Lines {
			x0, y0, x1, y1 := l.Bounds()
			b.Lines = append(b.Lines, LineBox{x0, y0, x1, y1})
			if i == 0 {
				b.X0, b.Y0, b.X1, b.Y1 = x0, y0, x1, y1
			}
			b.X0, b.Y0, b.X1, b.Y1 = min(b.X0, x0), min(b.Y0, y0), max(b.X1, x1), max(b.Y1, y1)
			hs = append(hs, y1-y0)
		}
		sort.Ints(hs)
		b.LineH = hs[len(hs)/2]
		b.Text = g.Transcript
		blocks[g.ID] = b
	}
	return img, blocks
}

func TestConcealmentModeOnSyntheticScenes(t *testing.T) {
	for _, tt := range []struct {
		scene, group string
		want         Mode
	}{
		// Flat paper: the fill, as every plate always was.
		{"synth-uniform-paper", "para", ModeFill},
		{"synth-two-columns", "left-column", ModeFill},
		{"synth-rtl-layout", "rtl-block", ModeFill},
		// The balloon's interior is flat white all round its text, and the outline lies outside the
		// block rectangle, so the fill cannot reach it: fill is right here, not mask (Step 07.1's
		// verification expected a mask on this scene - see the phase file's deviation note).
		{"synth-balloon-on-panel", "balloon", ModeFill},
		{"synth-side-by-side-balloons", "balloon-left", ModeFill},
		// A caption on a sky: a ramp top to bottom, its middle on both sides.
		{"synth-caption-on-gradient", "caption", ModeReconstruct},
		// Halftone round the text: a block-wide patch would erase the screen.
		{"synth-text-on-halftone", "narration", ModeMask},
	} {
		img, blocks := synthScene(t, tt.scene)
		b, ok := blocks[tt.group]
		if !ok {
			t.Fatalf("%s: no group %q", tt.scene, tt.group)
		}
		r := measureRing(img, b)
		got, conf := decideMode(r)
		if got != tt.want {
			t.Errorf("%s/%s: mode %s (conf %.2f, busy %d/%d), want %s", tt.scene, tt.group, got, conf, r.busy, r.n, tt.want)
		}
		if conf < 0 || conf > 1 {
			t.Errorf("%s/%s: confidence %v outside [0,1]", tt.scene, tt.group, conf)
		}
	}
}

func TestReconstructPaintsTheRampOfTheRing(t *testing.T) {
	img, blocks := synthScene(t, "synth-caption-on-gradient")
	b := blocks["caption"]
	r := measureRing(img, b)
	bg := plateBackground(ModeReconstruct, r, b, "rgb(1,2,3)", img.Bounds().Dx())
	want := "linear-gradient(to bottom," + rgb(r.top.med) + "," + rgb(r.bottom.med) + ")"
	if bg != want {
		t.Errorf("background = %q, want %q", bg, want)
	}
	if r.top.med == r.bottom.med {
		t.Errorf("the gradient's two ends are the same colour %v", r.top.med)
	}
}

// canvasOf builds an image of the given size filled with bg, then applies each fill in turn.
func canvasOf(w, h int, bg color.RGBA, fills ...func(*image.RGBA)) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	for _, f := range fills {
		f(img)
	}
	return img
}

func filled(x0, y0, x1, y1 int, c color.RGBA) func(*image.RGBA) {
	return func(img *image.RGBA) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// A white caption with a dark panel rule just beyond its right edge: the two sides of the ring
// differ by far more than a gradient would, and a ramp painted from white to black across the
// caption would be the worst plate possible. The sides across the axis are white, not the grey of
// the midpoint, so this is an edge and the block keeps the fill - which stays inside the block.
func TestAnEdgeBesideTheBlockIsNotAGradient(t *testing.T) {
	white := color.RGBA{250, 250, 245, 255}
	img := canvasOf(400, 120, white, filled(330, 0, 345, 120, color.RGBA{10, 10, 10, 255}))
	b := Block{X0: 40, Y0: 40, X1: 328, Y1: 70, LineH: 30, Lines: []LineBox{{40, 40, 328, 70}}}
	r := measureRing(img, b)
	if spread(r.left, r.right) <= modeFlatSpread {
		t.Fatalf("setup: left/right spread %d is not above the flat bound", spread(r.left, r.right))
	}
	if got, _ := decideMode(r); got != ModeFill {
		t.Errorf("mode = %s, want fill: an edge beside the block is not a ramp", got)
	}
}

// A block with no ring worth the name - it fills the picture - has no evidence about the surface,
// and the mode that paints least is the one that cannot erase artwork.
func TestAThinRingIsAMaskAtZero(t *testing.T) {
	img := canvasOf(60, 20, color.RGBA{255, 255, 255, 255})
	b := Block{X0: 0, Y0: 0, X1: 60, Y1: 20, LineH: 20}
	if got, conf := decideMode(measureRing(img, b)); got != ModeMask || conf != 0 {
		t.Errorf("mode = %s at %.2f, want mask at 0", got, conf)
	}
}

// A masked plate paints only its line boxes, so over two lines with leading between them and a
// short last line it paints strictly less than the block rectangle - which is the whole point.
func TestMaskPaintsLessThanTheBlock(t *testing.T) {
	const w = 1000
	b := Block{X0: 100, Y0: 100, X1: 700, Y1: 190, LineH: 30,
		Lines: []LineBox{{100, 100, 700, 130}, {100, 160, 400, 190}}}
	bg := plateBackground(ModeMask, ringStats{}, b, "rgb(9,9,9)", w)
	// The same string ocr-conceal.test.mjs expects for the same block: both editions paint alike.
	if want := "linear-gradient(rgb(9,9,9),rgb(9,9,9)) -0.500cqw -0.500cqw/61.000cqw 4.000cqw no-repeat," +
		"linear-gradient(rgb(9,9,9),rgb(9,9,9)) -0.500cqw 5.500cqw/31.000cqw 4.000cqw no-repeat"; bg != want {
		t.Errorf("mask background = %q, want %q", bg, want)
	}
	layer := regexp.MustCompile(`linear-gradient\(rgb\(9,9,9\),rgb\(9,9,9\)\) (-?[\d.]+)cqw (-?[\d.]+)cqw/([\d.]+)cqw ([\d.]+)cqw no-repeat`)
	ms := layer.FindAllStringSubmatch(bg, -1)
	if len(ms) != len(b.Lines) || strings.Count(bg, "linear-gradient") != len(b.Lines) {
		t.Fatalf("want one layer per line, got %q", bg)
	}
	px := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v / 100 * w }
	painted := 0.0
	for i, m := range ms {
		x, y, lw, lh := px(m[1]), px(m[2]), px(m[3]), px(m[4])
		l := b.Lines[i]
		pad := float64(max(b.LineH/modeMaskPadDivisor, ringMinPad))
		// Each stripe covers its own line box plus the pad, measured from the plate's corner.
		if x > float64(l.X0-b.X0)-pad+0.01 || y > float64(l.Y0-b.Y0)-pad+0.01 ||
			x+lw < float64(l.X1-b.X0)+pad-0.01 || y+lh < float64(l.Y1-b.Y0)+pad-0.01 {
			t.Errorf("stripe %d (%v,%v %vx%v) does not cover line %v with its pad", i, x, y, lw, lh, l)
		}
		// Clipped to the plate, as the browser does.
		cw := min(x+lw, float64(b.X1-b.X0)) - max(x, 0)
		ch := min(y+lh, float64(b.Y1-b.Y0)) - max(y, 0)
		painted += cw * ch
	}
	if block := float64((b.X1 - b.X0) * (b.Y1 - b.Y0)); painted >= block {
		t.Errorf("mask paints %.0f px, not strictly less than the block's %.0f", painted, block)
	}
}

// wrapImage records the mode on every plate and paints the mode's background, and the picture
// under the overlay is never touched: the file's bytes are identical after the overlay.
func TestWrapImageEmitsTheModeAndLeavesTheSourceAlone(t *testing.T) {
	img, blocks := synthScene(t, "synth-text-on-halftone")
	path := filepath.Join(t.TempDir(), "src.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), buf.Bytes()...)

	doc, err := gohtml.Parse(strings.NewReader(`<html><body><p><img src="src.png"></p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	wrapImage(collectImgs(doc)[0], Result{Width: b.Dx(), Height: b.Dy(), Blocks: []Block{blocks["narration"]}}, decodeImage(path))

	var out bytes.Buffer
	if err := gohtml.Render(&out, doc); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if !strings.Contains(html, `data-ocr-mode="mask"`) || !regexp.MustCompile(`data-ocr-mode-conf="[01]\.\d\d"`).MatchString(html) {
		t.Errorf("plate does not record its mode: %s", html)
	}
	if !strings.Contains(html, "cqw no-repeat") {
		t.Errorf("a mask plate carries no line stripes: %s", html)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the source image changed under the overlay")
	}
}
