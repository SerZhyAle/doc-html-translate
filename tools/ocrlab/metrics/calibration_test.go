package metrics

import (
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// The controls below have exact known answers: the fixtures are drawn here, so the lettering,
// the replacement glyphs and the painted pixels are inputs and not estimates. They assert the
// meaning of each measurement; they set no acceptance bound.

const calW, calH = 100, 60

// The old lettering: a 20x10 block, 200 pixels, mid-image.
var calLettering = image.Rect(40, 20, 60, 30)

var calBackgrounds = []string{"flat", "gradient", "texture"}

// calBackground paints one of the three background classes the diagnostic was found wrong on.
func calBackground(kind string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, calW, calH))
	for y := range calH {
		for x := range calW {
			v := uint8(240)
			switch kind {
			case "gradient":
				v = uint8(60 + x*180/calW)
			case "texture":
				if (x/4+y/4)%2 == 0 {
					v = 100
				}
			}
			img.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return img
}

func calCopy(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, image.Point{}, draw.Src)
	return dst
}

func calFill(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, &image.Uniform{c}, image.Point{}, draw.Src)
}

func calLetteringMask() *truth.Mask {
	m := truth.NewMask(calW, calH)
	for y := calLettering.Min.Y; y < calLettering.Max.Y; y++ {
		for x := calLettering.Min.X; x < calLettering.Max.X; x++ {
			m.Set(x, y)
		}
	}
	return m
}

func calGroup() truth.Group {
	return truth.Group{ID: "text", Bounds: truth.Box("text", 0, 0, calW, calH)}
}

// A known mask measures the same on every background; that is the point of having one.
func TestResidualKnownMaskControls(t *testing.T) {
	mask := calLetteringMask()
	for _, background := range calBackgrounds {
		t.Run(background, func(t *testing.T) {
			clean := calBackground(background)
			source := calCopy(clean)
			calFill(source, calLettering, color.Black)

			// New text drawn exactly on the old coordinates, in the old colour: the normal
			// render is indistinguishable from untouched lettering. Only the text-hidden
			// capture, where the plate stays and the new glyphs are transparent, shows the truth.
			normal := calCopy(clean)
			calFill(normal, calLettering, color.Black)
			hidden := calCopy(clean)

			cases := []struct {
				name     string
				capture  image.Image
				residual float64
			}{
				{"untouched lettering", source, 1},
				{"exactly removed", hidden, 0},
				{"aligned replacement glyphs, text-hidden capture", hidden, 0},
				{"aligned replacement glyphs, normal render (the trap)", normal, 1},
			}
			for _, c := range cases {
				got := ResidualKnownMask(source, c.capture, mask, calGroup(), calW, calH)
				if !got.IsMeasured() || got.State != StateMeasured || got.Basis != BasisKnownMask {
					t.Fatalf("%s: %+v", c.name, got)
				}
				if got.Residual != c.residual || got.InkPx != 200 {
					t.Errorf("%s: residual %g over %d px, want %g over 200", c.name, got.Residual, got.InkPx, c.residual)
				}
			}
		})
	}
}

// Without a mask the single-tone estimate is valid on a flat background and is refused elsewhere.
// The refusal carries no percentage: the old answers were 90.86% on a gradient and 96.89% on a
// texture with every lettering pixel removed.
func TestResidualWithoutMaskIsUnmeasuredOnBackgroundStructure(t *testing.T) {
	for _, background := range calBackgrounds {
		t.Run(background, func(t *testing.T) {
			clean := calBackground(background)
			source := calCopy(clean)
			calFill(source, calLettering, color.Black)

			untouched := ResidualFor(source, source, nil, calGroup(), calW, calH)
			removed := ResidualFor(source, clean, nil, calGroup(), calW, calH)
			if background == "flat" {
				if !untouched.IsMeasured() || untouched.Residual != 1 || untouched.Basis != BasisFlatRegion {
					t.Errorf("flat untouched: %+v", untouched)
				}
				if !removed.IsMeasured() || removed.Residual != 0 {
					t.Errorf("flat removed: %+v", removed)
				}
				return
			}
			for name, got := range map[string]ResidualScore{"untouched": untouched, "removed": removed} {
				if got.IsMeasured() || got.State != StateUnmeasured || got.Reason == "" {
					t.Errorf("%s %s: want unmeasured with a reason, got %+v", background, name, got)
				}
				if got.Residual != 0 || got.Halo != 0 || got.InkPx != 0 || got.Basis != "" {
					t.Errorf("%s %s: an unmeasured residual must carry no number, got %+v", background, name, got)
				}
			}
			scene := ResidualAcross(source, clean, nil, []truth.Group{calGroup()}, calW, calH)
			if scene.IsMeasured() || !strings.Contains(scene.Reason, "text") {
				t.Errorf("%s: scene fold must stay unmeasured and say which group, got %+v", background, scene)
			}
		})
	}
}

// An unmeasured concealment must not be taken for a clean one, and a scene made of one measurable
// and one unmeasurable group is unmeasured as a whole.
func TestResidualAcrossRefusesToAverageAwayAnUnmeasuredGroup(t *testing.T) {
	flat := calBackground("flat")
	source := calCopy(flat)
	calFill(source, calLettering, color.Black)
	hidden := calCopy(flat)
	good := truth.Group{ID: "flat", Bounds: truth.Box("flat", 0, 0, calW, calH)}
	mask := calLetteringMask()
	if got := ResidualAcross(source, hidden, mask, []truth.Group{good}, calW, calH); !got.IsMeasured() || got.Residual != 0 {
		t.Fatalf("one measurable group: %+v", got)
	}
	empty := truth.Group{ID: "elsewhere", Bounds: truth.Box("elsewhere", 0, 40, 30, 60)}
	got := ResidualAcross(source, hidden, mask, []truth.Group{good, empty}, calW, calH)
	if got.IsMeasured() || !strings.Contains(got.Reason, "elsewhere") {
		t.Errorf("a group with no known lettering must make the scene unmeasured, got %+v", got)
	}
}

// Scores written before State existed carry only Measured; they must keep meaning what they meant.
func TestResidualScoreDecodesLegacyJSON(t *testing.T) {
	var measured, unmeasured ResidualScore
	if err := json.Unmarshal([]byte(`{"residual":0.5,"halo":0.25,"inkPx":120,"measured":true}`), &measured); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"residual":0,"halo":0,"inkPx":0,"measured":false}`), &unmeasured); err != nil {
		t.Fatal(err)
	}
	if !measured.IsMeasured() || measured.Residual != 0.5 {
		t.Errorf("legacy measured score: %+v", measured)
	}
	if unmeasured.IsMeasured() {
		t.Errorf("legacy unmeasured score: %+v", unmeasured)
	}
	var damage DamageScore
	if err := json.Unmarshal([]byte(`{"protectedHit":600}`), &damage); err != nil {
		t.Fatal(err)
	}
	if !damage.IsMeasured() {
		t.Error("a legacy damage score is rectangle geometry and was always measured")
	}
}

// ---- replacement readability ------------------------------------------------

// glyphRows draws a row of glyph-like blocks of the given height starting at the given y.
func glyphRows(img *image.RGBA, y, height int) {
	for x := 14; x < 80; x += 8 {
		calFill(img, image.Rect(x, y, x+5, y+height), color.Black)
	}
}

func readabilityPlate(r evidence.Rect, fontPx float64) []evidence.Plate {
	return []evidence.Plate{{Rect: r, FontPx: fontPx, Viewport: testViewport, StressCase: PrimaryStressCase}}
}

func TestReplacementReadabilityDistinguishesWhatContrastCannot(t *testing.T) {
	texture := calBackground("texture")
	plateRect := evidence.Rect{X0: 10, Y0: 10, X1: 90, Y1: 40}

	t.Run("no glyphs drawn", func(t *testing.T) {
		got := ReplacementReadability(texture, texture, readabilityPlate(plateRect, 16), calW, calH)
		if got.State != StateNoGlyphs || got.GlyphPx != 0 || got.Reason == "" {
			t.Fatalf("a texture with no replacement text: %+v", got)
		}
		// The rectangle still has luma separation. That is the old figure which was read as
		// readability, and it is unchanged by the font metadata.
		contrast := BackgroundContrast(texture, readabilityPlate(plateRect, 1), calW, calH)
		if contrast.MinLuma <= 0 {
			t.Fatal("the texture control has no background contrast")
		}
		if other := BackgroundContrast(texture, readabilityPlate(plateRect, 16), calW, calH); other != contrast {
			t.Fatal("background contrast must not depend on font metadata")
		}
	})

	draw3 := func(y, height int, r evidence.Rect) ReadabilityScore {
		normal := calCopy(texture)
		glyphRows(normal, y, height)
		return ReplacementReadability(normal, texture, readabilityPlate(r, 12), calW, calH)
	}
	readable := draw3(14, 12, plateRect)
	undersized := draw3(14, 2, plateRect)
	// The plate is only 10 px tall, so a 12 px row is cut by its lower edge.
	clipped := draw3(14, 12, evidence.Rect{X0: 10, Y0: 10, X1: 90, Y1: 20})

	for name, got := range map[string]ReadabilityScore{"readable": readable, "undersized": undersized, "clipped": clipped} {
		if got.State != StateMeasured || got.GlyphPx == 0 || got.MinSeparation < 100 {
			t.Fatalf("%s: %+v", name, got)
		}
	}
	if readable.MinGlyphHeightPx != 12 || readable.ClippedFraction != 0 {
		t.Errorf("readable glyphs: %+v", readable)
	}
	if undersized.MinGlyphHeightPx != 2 || undersized.ClippedFraction != 0 {
		t.Errorf("undersized glyphs: %+v", undersized)
	}
	if clipped.MinGlyphHeightPx != 6 || clipped.ClippedFraction != 1 || clipped.ClippedLines != clipped.Lines {
		t.Errorf("clipped glyphs: %+v", clipped)
	}
	if undersized.MinGlyphHeightPx >= readable.MinGlyphHeightPx {
		t.Error("undersized glyphs must be shorter than readable ones on the same background")
	}
	// Same pixels, different claimed font: the result follows what was drawn.
	normal := calCopy(texture)
	glyphRows(normal, 14, 2)
	if a, b := ReplacementReadability(normal, texture, readabilityPlate(plateRect, 1), calW, calH),
		ReplacementReadability(normal, texture, readabilityPlate(plateRect, 16), calW, calH); a != b {
		t.Errorf("font metadata changed the readability: %+v vs %+v", a, b)
	}
}

func TestReplacementReadabilityNeedsBothCaptures(t *testing.T) {
	img := calBackground("flat")
	for name, got := range map[string]ReadabilityScore{
		"no normal":  ReplacementReadability(nil, img, nil, calW, calH),
		"no hidden":  ReplacementReadability(img, nil, nil, calW, calH),
		"wrong size": ReplacementReadability(img, image.NewRGBA(image.Rect(0, 0, 5, 5)), nil, calW, calH),
	} {
		if got.State != StateUnmeasured || got.Reason == "" {
			t.Errorf("%s: want unmeasured with a reason, got %+v", name, got)
		}
	}
	if got := ReplacementReadability(img, img, nil, calW, calH); got.State != StateNoGlyphs {
		t.Errorf("no plates draw no glyphs: %+v", got)
	}
}

// ---- painted damage ---------------------------------------------------------

// protectedArt marks a 10x60 strip, 600 pixels, as protected content.
func calArt() (*truth.Annotation, image.Rectangle) {
	art := image.Rect(0, 0, 10, calH)
	return &truth.Annotation{
		ImageWidth: calW, ImageHeight: calH,
		Groups:    []truth.Group{calGroup()},
		Protected: []truth.Region{truth.Box("art", art.Min.X, art.Min.Y, art.Max.X, art.Max.Y)},
	}, art
}

func TestPaintedDamageSeparatesPaintFromRectangle(t *testing.T) {
	ann, art := calArt()
	source := calBackground("flat")
	calFill(source, art, color.RGBA{30, 90, 160, 255})

	t.Run("central mask paints no art", func(t *testing.T) {
		// The plate's box covers the whole image, but the mask changed only the central glyphs.
		plates := readabilityPlate(evidence.Rect{X0: 0, Y0: 0, X1: calW, Y1: calH}, 12)
		hidden := calCopy(source)
		calFill(hidden, calLettering, color.Black)

		if rect := RectangleIntrusion(plates, ann, calW, calH); rect.ProtectedHit != 600 {
			t.Fatalf("rectangle intrusion: %+v", rect)
		}
		painted := PaintedDamage(source, hidden, plates, ann, calW, calH)
		if !painted.IsMeasured() || painted.State != StateMeasured {
			t.Fatalf("painted damage: %+v", painted)
		}
		if painted.ProtectedHit != 0 || painted.OverlayPx != calLettering.Dx()*calLettering.Dy() {
			t.Errorf("a central mask paints 200 px and none of them art: %+v", painted)
		}
	})

	t.Run("rectangle fill paints its known count", func(t *testing.T) {
		// Fill x 5..50: five columns of art fall inside, 5 x 60 = 300 pixels.
		plates := readabilityPlate(evidence.Rect{X0: 5, Y0: 0, X1: 50, Y1: calH}, 12)
		hidden := calCopy(source)
		calFill(hidden, image.Rect(5, 0, 50, calH), color.RGBA{255, 255, 255, 255})

		painted := PaintedDamage(source, hidden, plates, ann, calW, calH)
		rect := RectangleIntrusion(plates, ann, calW, calH)
		if painted.ProtectedHit != 300 || rect.ProtectedHit != 300 {
			t.Errorf("painted %d, rectangle %d, want 300 each", painted.ProtectedHit, rect.ProtectedHit)
		}
		if painted.WorstProtectedRegion != "art" || painted.WorstProtectedPx != 300 {
			t.Errorf("the damage must name the region it hit: %+v", painted)
		}
	})

	t.Run("a difference outside every plate is not the overlay's", func(t *testing.T) {
		plates := readabilityPlate(evidence.Rect{X0: 40, Y0: 0, X1: 60, Y1: 10}, 12)
		hidden := calCopy(source)
		calFill(hidden, image.Rect(0, 0, 10, 10), color.White) // capture noise on the art, no plate there
		if painted := PaintedDamage(source, hidden, plates, ann, calW, calH); painted.ProtectedHit != 0 {
			t.Errorf("paint must be attributed to plates only: %+v", painted)
		}
	})

	t.Run("without a hidden capture it is unmeasured", func(t *testing.T) {
		plates := readabilityPlate(evidence.Rect{X0: 0, Y0: 0, X1: calW, Y1: calH}, 12)
		got := PaintedDamage(source, nil, plates, ann, calW, calH)
		if got.IsMeasured() || got.State != StateUnmeasured || got.Reason == "" || got.ProtectedHit != 0 {
			t.Errorf("no hidden capture: %+v", got)
		}
	})
}
