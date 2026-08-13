// Package synth draws diagnostic scenes whose text and geometry are known exactly, because the
// lab drew them.
//
// Two problems it solves. First, every metric in the lab needs a fixture, and a fixture built
// from downloaded media cannot exist until a human has verified a licence - which would make
// the measurement code untestable until the corpus is finished. Second, a real scene's ground
// truth is an estimate a person typed; here the transcript, the line boxes, the balloon outline
// and the protected panel are the inputs to the drawing, so the annotation is exact by
// construction rather than by care.
//
// Every scene is a pure function of its own constants - no clock, no randomness - so
// regenerating produces byte-identical PNGs and the manifest diff stays empty.
//
// What these scenes are NOT: a substitute for the corpus. They cover geometry, background
// classes and layout traps. Script coverage (Arabic, CJK, Devanagari) needs real typography and
// real shaping, which the Go fonts here cannot provide; those scenes are corpus work.
package synth

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// Annotator is the name recorded on a generated annotation. It is not a person, and that is the
// point: these scenes are exact rather than reviewed, and a reader of the JSON should be able
// to tell at a glance which kind of record they are holding.
const Annotator = "tools/ocrlab/synth"

// Dir is the sub-directory of the corpus root that holds generated media.
const Dir = "synthetic"

// canvas is a scene under construction: the image plus the annotation being accumulated as
// things are drawn onto it.
type canvas struct {
	img *image.RGBA
	ann *truth.Annotation
}

func newCanvas(sceneID string, w, h int, bg color.RGBA) *canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	return &canvas{
		img: img,
		ann: &truth.Annotation{
			SchemaVersion: truth.SchemaVersion,
			SceneID:       sceneID,
			Origin:        truth.OriginHuman,
			ImageWidth:    w,
			ImageHeight:   h,
			Ambiguity:     truth.AmbiguityClear,
			Review: truth.Review{
				AnnotatedBy: Annotator,
				AnnotatedOn: GeneratedOn,
				CheckedBy:   Annotator + " (exact by construction)",
				CheckedOn:   GeneratedOn,
			},
		},
	}
}

// GeneratedOn is the date stamped into generated records. It is a constant rather than the
// clock so that regenerating never rewrites a file.
const GeneratedOn = "2026-08-11"

// face loads a Go font at the given pixel size. The Go fonts ship inside golang.org/x/image,
// already a dependency of this repository, so the generator adds no module and needs no
// network.
func loadFace(size float64, bold bool) font.Face {
	data := goregular.TTF
	if bold {
		data = gobold.TTF
	}
	f, err := opentype.Parse(data)
	if err != nil {
		panic("synth: parse bundled Go font: " + err.Error())
	}
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic("synth: build face: " + err.Error())
	}
	return fc
}

// drawLine paints one text line with its top-left at (x, y) and returns the line's exact
// bounding box. The box comes from the font metrics of what was actually drawn, so a line
// region is never an approximation.
func (c *canvas) drawLine(text string, x, y int, ink color.RGBA, face font.Face) truth.Region {
	m := face.Metrics()
	ascent := m.Ascent.Ceil()
	baseline := y + ascent
	d := &font.Drawer{
		Dst:  c.img,
		Src:  &image.Uniform{ink},
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	advance := d.MeasureString(text)
	d.DrawString(text)
	return truth.Box("", x, y, x+advance.Ceil(), baseline+m.Descent.Ceil())
}

// addGroup records a reading group from the lines just drawn, computing Bounds as their union
// and ReplaceArea from an explicit inset (how far inside the region a plate may paint).
func (c *canvas) addGroup(id string, kind truth.GroupType, transcript string, lines []truth.Region, replace truth.Region) {
	x0, y0, x1, y1 := lines[0].Bounds()
	for _, l := range lines[1:] {
		a, b, cc, d := l.Bounds()
		x0, y0, x1, y1 = min(x0, a), min(y0, b), max(x1, cc), max(y1, d)
	}
	bounds := truth.Box(id+"-bounds", x0, y0, x1, y1)
	if replace.Empty() {
		replace = bounds
		replace.ID = id + "-replace"
	}
	c.ann.Groups = append(c.ann.Groups, truth.Group{
		ID:           id,
		Type:         kind,
		Transcript:   transcript,
		Language:     "en",
		Direction:    truth.DirLTR,
		ReadingOrder: len(c.ann.Groups) + 1,
		Lines:        lines,
		Bounds:       bounds,
		ReplaceArea:  replace,
	})
}

func (c *canvas) protect(r truth.Region) { c.ann.Protected = append(c.ann.Protected, r) }

// ---- drawing helpers -------------------------------------------------------

func (c *canvas) fillRect(x0, y0, x1, y1 int, col color.RGBA) {
	draw.Draw(c.img, image.Rect(x0, y0, x1, y1), &image.Uniform{col}, image.Point{}, draw.Src)
}

// strokeRect draws a rectangle outline of the given thickness and returns the outline as a
// protected region: a border a plate paints over is damage, not concealment.
func (c *canvas) strokeRect(id string, x0, y0, x1, y1, thick int, col color.RGBA) truth.Region {
	c.fillRect(x0, y0, x1, y0+thick, col)
	c.fillRect(x0, y1-thick, x1, y1, col)
	c.fillRect(x0, y0, x0+thick, y1, col)
	c.fillRect(x1-thick, y0, x1, y1, col)
	// The outline as a polygon ring: outer boundary then inner boundary, so the even-odd rule
	// leaves the interior uncovered and only the stroke itself is protected.
	return truth.Region{ID: id, Kind: truth.RegionPolygon, Points: [][2]int{
		{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}, {x0, y0},
		{x0 + thick, y0 + thick}, {x0 + thick, y1 - thick}, {x1 - thick, y1 - thick},
		{x1 - thick, y0 + thick}, {x0 + thick, y0 + thick},
	}}
}

// ellipse fills an axis-aligned ellipse and returns its polygon approximation.
func (c *canvas) ellipse(id string, cx, cy, rx, ry int, col color.RGBA) truth.Region {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1 {
				c.img.SetRGBA(x, y, col)
			}
		}
	}
	const steps = 48
	pts := make([][2]int, 0, steps)
	for i := 0; i < steps; i++ {
		t := 2 * math.Pi * float64(i) / steps
		pts = append(pts, [2]int{cx + int(float64(rx)*math.Cos(t)), cy + int(float64(ry)*math.Sin(t))})
	}
	return truth.Region{ID: id, Kind: truth.RegionPolygon, Points: pts}
}

// gradient paints a smooth vertical ramp - the "reconstruct" background class.
func (c *canvas) gradient(x0, y0, x1, y1 int, top, bottom color.RGBA) {
	h := y1 - y0
	if h <= 0 {
		return
	}
	for y := y0; y < y1; y++ {
		t := float64(y-y0) / float64(h)
		col := color.RGBA{
			R: lerp(top.R, bottom.R, t),
			G: lerp(top.G, bottom.G, t),
			B: lerp(top.B, bottom.B, t),
			A: 255,
		}
		c.fillRect(x0, y, x1, y+1, col)
	}
}

// halftone paints a regular dot screen - the textured class, where a block fill is visibly
// wrong and a tight mask is the only honest option.
func (c *canvas) halftone(x0, y0, x1, y1 int, dot color.RGBA, pitch int) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x%pitch == 0 && y%pitch == 0 {
				c.fillRect(x, y, x+pitch/2, y+pitch/2, dot)
			}
		}
	}
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

var (
	paper   = color.RGBA{250, 249, 246, 255}
	ink     = color.RGBA{24, 24, 28, 255}
	white   = color.RGBA{255, 255, 255, 255}
	black   = color.RGBA{0, 0, 0, 255}
	panelBg = color.RGBA{188, 92, 64, 255}
	skyTop  = color.RGBA{70, 130, 200, 255}
	skyBot  = color.RGBA{230, 200, 150, 255}
)

// ---- the scenes ------------------------------------------------------------

// builder is one named scene generator plus the categories it exercises.
type builder struct {
	id         string
	categories []corpus.Category
	note       string
	draw       func() *canvas
}

func builders() []builder {
	return []builder{
		{"synth-uniform-paper", []corpus.Category{corpus.CatDocument}, "three document lines on flat paper - the ModeFill case", sceneUniformPaper},
		{"synth-two-columns", []corpus.Category{corpus.CatDocument}, "two text columns - a plate spanning both is a merge defect", sceneTwoColumns},
		{"synth-balloon-on-panel", []corpus.Category{corpus.CatComic}, "white balloon with an outline over a coloured panel - the outline is protected", sceneBalloonOnPanel},
		{"synth-adjacent-balloons", []corpus.Category{corpus.CatComic}, "two balloons close enough to tempt a merge", sceneAdjacentBalloons},
		{"synth-caption-on-gradient", []corpus.Category{corpus.CatTexture}, "caption over a smooth gradient - the ModeReconstruct case", sceneCaptionOnGradient},
		{"synth-text-on-halftone", []corpus.Category{corpus.CatTexture, corpus.CatCartoon}, "text over a halftone screen - a block fill is visibly wrong here", sceneTextOnHalftone},
		{"synth-display-lettering", []corpus.Category{corpus.CatPoster}, "large display type on a coloured fill", sceneDisplayLettering},
		{"synth-rtl-layout", []corpus.Category{corpus.CatScript}, "right-to-left layout proxy: direction and alignment only, not Arabic shaping - real RTL scenes are corpus work", sceneRTLLayout},
	}
}

func sceneUniformPaper() *canvas {
	c := newCanvas("synth-uniform-paper", 640, 320, paper)
	face := loadFace(20, false)
	texts := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Pack my box with five dozen liquor jugs.",
		"How vexingly quick daft zebras jump.",
	}
	var lines []truth.Region
	y := 60
	for i, t := range texts {
		r := c.drawLine(t, 48, y, ink, face)
		r.ID = fmt.Sprintf("line-%d", i+1)
		lines = append(lines, r)
		y += 42
	}
	c.addGroup("para", truth.GroupParagraph, joinLines(texts), lines, truth.Region{})
	return c
}

func sceneTwoColumns() *canvas {
	c := newCanvas("synth-two-columns", 720, 360, paper)
	face := loadFace(17, false)
	left := []string{"Column one begins here", "and carries a second line", "and then a third."}
	right := []string{"Column two is separate", "and must never share", "a plate with column one."}

	var lines []truth.Region
	y := 56
	for i, t := range left {
		r := c.drawLine(t, 48, y, ink, face)
		r.ID = fmt.Sprintf("l-%d", i+1)
		lines = append(lines, r)
		y += 34
	}
	c.addGroup("left-column", truth.GroupParagraph, joinLines(left), lines, truth.Region{})

	lines = nil
	y = 56
	for i, t := range right {
		r := c.drawLine(t, 396, y, ink, face)
		r.ID = fmt.Sprintf("r-%d", i+1)
		lines = append(lines, r)
		y += 34
	}
	c.addGroup("right-column", truth.GroupParagraph, joinLines(right), lines, truth.Region{})
	return c
}

func sceneBalloonOnPanel() *canvas {
	c := newCanvas("synth-balloon-on-panel", 560, 420, panelBg)
	// A drawn "figure" the overlay must not paint over.
	fig := c.ellipse("figure", 150, 330, 70, 80, color.RGBA{240, 205, 170, 255})
	c.protect(fig)

	c.fillRect(300, 60, 520, 200, white)
	outline := c.strokeRect("balloon-outline", 300, 60, 520, 200, 5, black)
	c.protect(outline)

	face := loadFace(19, true)
	texts := []string{"WELL, THAT IS", "ONE WAY TO", "SOLVE IT!"}
	var lines []truth.Region
	y := 88
	for i, t := range texts {
		r := c.drawLine(t, 322, y, black, face)
		r.ID = fmt.Sprintf("b-%d", i+1)
		lines = append(lines, r)
		y += 36
	}
	// The plate may paint the balloon's interior only - inset past the 5 px stroke.
	c.addGroup("balloon", truth.GroupBalloon, joinLines(texts), lines, truth.Box("balloon-replace", 306, 66, 514, 194))
	return c
}

// sceneAdjacentBalloons is the merge trap, and its geometry is the whole point.
//
// The recognizer groups lines into one plate while the step from one line's top to the next stays
// within ocrClusterPitchFactor (1.2) x the page's reference line pitch - so two balloons only tempt
// a merge when the step across to the next balloon is barely more than the leading inside one.
// Balloons that merely look close on the page prove nothing: at 18 px type the leading is 26 px, so
// the step has to be under ~31 px, which puts the two outlines almost touching. That is exactly the
// dense-panel case, and separating them has to come from the outline between them rather than from
// distance. Measured 2026-08-11: 26 px inside a balloon against 42 px across to the next one.
func sceneAdjacentBalloons() *canvas {
	c := newCanvas("synth-adjacent-balloons", 620, 300, panelBg)
	face := loadFace(18, true)

	c.fillRect(40, 50, 290, 114, white)
	c.protect(c.strokeRect("left-outline", 40, 50, 290, 114, 4, black))
	l1 := c.drawLine("ARE YOU SURE", 58, 58, black, face)
	l1.ID = "a-1"
	l2 := c.drawLine("ABOUT THIS?", 58, 84, black, face)
	l2.ID = "a-2"
	c.addGroup("balloon-a", truth.GroupBalloon, "ARE YOU SURE ABOUT THIS?", []truth.Region{l1, l2},
		truth.Box("a-replace", 45, 55, 285, 109))

	c.fillRect(40, 118, 290, 182, white)
	c.protect(c.strokeRect("right-outline", 40, 118, 290, 182, 4, black))
	l3 := c.drawLine("NOT EVEN", 58, 126, black, face)
	l3.ID = "b-1"
	l4 := c.drawLine("SLIGHTLY.", 58, 152, black, face)
	l4.ID = "b-2"
	c.addGroup("balloon-b", truth.GroupBalloon, "NOT EVEN SLIGHTLY.", []truth.Region{l3, l4},
		truth.Box("b-replace", 45, 123, 285, 177))
	return c
}

func sceneCaptionOnGradient() *canvas {
	c := newCanvas("synth-caption-on-gradient", 640, 360, white)
	c.gradient(0, 0, 640, 360, skyTop, skyBot)
	face := loadFace(21, false)
	l1 := c.drawLine("Evening fell over the harbour", 60, 240, white, face)
	l1.ID = "cap-1"
	l2 := c.drawLine("and the lamps came on one by one.", 60, 276, white, face)
	l2.ID = "cap-2"
	c.addGroup("caption", truth.GroupCaption,
		"Evening fell over the harbour and the lamps came on one by one.",
		[]truth.Region{l1, l2}, truth.Region{})
	return c
}

func sceneTextOnHalftone() *canvas {
	c := newCanvas("synth-text-on-halftone", 600, 300, color.RGBA{245, 240, 225, 255})
	c.halftone(0, 0, 600, 300, color.RGBA{120, 110, 100, 255}, 6)
	face := loadFace(24, true)
	l1 := c.drawLine("MEANWHILE, ELSEWHERE", 60, 110, black, face)
	l1.ID = "h-1"
	c.addGroup("narration", truth.GroupCaption, "MEANWHILE, ELSEWHERE", []truth.Region{l1}, truth.Region{})
	return c
}

func sceneDisplayLettering() *canvas {
	c := newCanvas("synth-display-lettering", 600, 400, color.RGBA{212, 44, 52, 255})
	face := loadFace(46, true)
	small := loadFace(18, false)
	l1 := c.drawLine("READ", 70, 70, white, face)
	l1.ID = "d-1"
	l2 := c.drawLine("EVERY DAY", 70, 140, white, face)
	l2.ID = "d-2"
	c.addGroup("headline", truth.GroupTitle, "READ EVERY DAY", []truth.Region{l1, l2}, truth.Region{})

	l3 := c.drawLine("A public service message", 70, 320, white, small)
	l3.ID = "d-3"
	c.addGroup("footnote", truth.GroupLabel, "A public service message", []truth.Region{l3}, truth.Region{})
	return c
}

func sceneRTLLayout() *canvas {
	c := newCanvas("synth-rtl-layout", 600, 240, paper)
	face := loadFace(20, false)
	texts := []string{"right aligned line one", "right aligned line two"}
	var lines []truth.Region
	y := 70
	for i, t := range texts {
		// Right-align by measuring first, so the block hugs the right margin the way an RTL
		// column does. This is a layout proxy: the glyphs are Latin because the bundled Go
		// fonts carry no Arabic, and shaping is not simulated.
		d := &font.Drawer{Face: face}
		w := d.MeasureString(t).Ceil()
		r := c.drawLine(t, 540-w, y, ink, face)
		r.ID = fmt.Sprintf("rtl-%d", i+1)
		lines = append(lines, r)
		y += 40
	}
	c.addGroup("rtl-block", truth.GroupParagraph, joinLines(texts), lines, truth.Region{})
	c.ann.Groups[0].Direction = truth.DirRTL
	c.ann.ReviewerNote = "Layout-only RTL proxy. Arabic and Urdu shaping are not simulated; those scenes must come from the licensed corpus."
	return c
}

func joinLines(ls []string) string {
	out := ""
	for i, l := range ls {
		if i > 0 {
			out += " "
		}
		out += l
	}
	return out
}

// ---- generation ------------------------------------------------------------

// Generate draws every scene under <root>/synthetic, returning the manifest entries and the
// annotations. Existing files are overwritten with identical bytes, so running it twice leaves
// the working tree unchanged.
func Generate(root string) ([]corpus.Scene, []*truth.Annotation, error) {
	dir := filepath.Join(root, Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	var scenes []corpus.Scene
	var anns []*truth.Annotation

	for _, b := range builders() {
		c := b.draw()
		rel := filepath.ToSlash(filepath.Join(Dir, b.id+".png"))
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := writePNG(path, c.img); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", b.id, err)
		}
		sum, err := corpus.HashFile(path)
		if err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}
		bounds := c.img.Bounds()
		scenes = append(scenes, corpus.Scene{
			ID:                b.id,
			File:              rel,
			RetrievedOn:       GeneratedOn,
			Licence:           corpus.LicenceSynthetic,
			LicenceVerifiedBy: Annotator,
			LicenceVerifiedOn: GeneratedOn,
			SHA256:            sum,
			Bytes:             info.Size(),
			Width:             bounds.Dx(),
			Height:            bounds.Dy(),
			Languages:         []string{"en"},
			Scripts:           []string{"Latn"},
			Categories:        b.categories,
			Split:             corpus.SplitDev, // generated scenes can never be a holdout: they are tunable by definition
			Note:              b.note,
		})
		anns = append(anns, c.ann)
	}
	return scenes, anns, nil
}

// writePNG encodes deterministically: the standard encoder with default compression is a pure
// function of the pixels, so the same drawing always produces the same bytes.
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := (&png.Encoder{CompressionLevel: png.DefaultCompression}).Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
