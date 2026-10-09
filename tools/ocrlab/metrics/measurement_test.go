package metrics

import (
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/synth"
	"doc-html-translate/tools/ocrlab/truth"
)

// The ink-share condition is a statement about real line boxes, so keep it tied to the generated
// scenes: plain lettering on paper, panels and a locally smooth ramp must stay measurable, and the
// one line box sitting on a halftone screen must not. If a scene ever moves across that line, the
// condition is wrong for the corpus and this test says so before a run does.
func TestSupportConditionSeparatesGeneratedTextFromTexture(t *testing.T) {
	anns, scs, root := scenes(t)
	for id, a := range anns {
		src := loadPNG(t, filepath.Join(root, filepath.FromSlash(scs[id].File)))
		for _, g := range a.Groups {
			for _, r := range groupRegions(g) {
				why := unsupportedBackground(regionLumas(src, r, a.ImageWidth, a.ImageHeight))
				if textured := id == "synth-text-on-halftone"; textured && why == "" {
					t.Errorf("%s/%s: a halftone line box must be unsupported", id, g.ID)
				} else if !textured && why != "" {
					t.Errorf("%s/%s: plain lettering must stay measurable, got %q", id, g.ID, why)
				}
			}
		}
	}
}

// Residual is judged on the text-hidden capture against the lettering mask the drawing produced:
// replacement glyphs drawn on the very coordinates of the old lettering do not count as old ink.
func TestScoreResidualUsesTheHiddenCaptureAndTheDeclaredMask(t *testing.T) {
	anns, scs, root := scenes(t)
	const id = "synth-caption-on-gradient"
	a := anns[id]
	src := loadPNG(t, filepath.Join(root, filepath.FromSlash(scs[id].File)))
	maskDir := t.TempDir()
	if _, err := synth.GenerateLettering(maskDir); err != nil {
		t.Fatal(err)
	}
	lettering, err := truth.LoadLettering(maskDir, id, a.ImageWidth, a.ImageHeight)
	if err != nil || lettering == nil {
		t.Fatalf("lettering mask: %v, %v", lettering, err)
	}

	plates := perfectPlates(a, PrimaryStressCase)
	// The plate repaints the lettering area; the replacement glyphs are drawn on top in the normal
	// render at the very coordinates of the old lettering, in black.
	hidden := paintOver(src, plates, color.RGBA{200, 200, 200, 255})
	normal := paintOver(src, plates, color.RGBA{200, 200, 200, 255})
	for _, p := range plates {
		for y := p.Rect.Y0; y < p.Rect.Y0+8; y++ {
			for x := p.Rect.X0; x < p.Rect.X1; x += 2 {
				normal.(*image.RGBA).Set(x, y, color.Black)
			}
		}
	}
	open := func(rel string) image.Image {
		if rel == "hidden.png" {
			return hidden
		}
		return normal
	}
	sc := evScene(a, plates)
	sc.Observations = []evidence.Observation{{Viewport: testViewport, StressCase: PrimaryStressCase, Rendered: "normal.png", Concealed: "hidden.png"}}

	with, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs[id], Captures{Source: src, Open: open, Lettering: lettering})
	if err != nil {
		t.Fatal(err)
	}
	if !with.Residual.IsMeasured() || with.Residual.Basis != BasisKnownMask || with.Residual.Residual != 0 {
		t.Errorf("with the declared mask the removed lettering reads 0%% old ink: %+v", with.Residual)
	}
	if with.Readability.State != StateMeasured || with.Readability.GlyphPx == 0 {
		t.Errorf("the glyphs drawn on the plate must be found: %+v", with.Readability)
	}
	if !with.Damage.IsMeasured() {
		t.Errorf("both captures present, damage must be measured: %+v", with.Damage)
	}

}

// With no hidden capture the painted damage is unmeasured, never clean, and the rectangle figure
// stays available under its own name without failing the scene.
func TestScoreDamageIsUnmeasuredWithoutAHiddenCapture(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]
	sloppy := []evidence.Plate{{
		Rect:     evidence.Rect{X0: 295, Y0: 55, X1: 525, Y1: 205},
		Viewport: testViewport, StressCase: PrimaryStressCase, ScrollHeight: 20, ClientHeight: 20,
	}}
	sc := evScene(a, sloppy)
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], Captures{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Damage.IsMeasured() || got.Damage.Reason == "" {
		t.Errorf("painted damage without captures: %+v", got.Damage)
	}
	if got.RectangleIntrusion.ProtectedHit == 0 {
		t.Error("rectangle intrusion needs no capture and must still be reported")
	}
	for _, f := range got.Failures {
		if strings.Contains(f, "protected-area damage") {
			t.Errorf("rectangle geometry must not stand in for painted damage: %v", got.Failures)
		}
	}
	sum := Aggregate("t", evidence.EditionDesktop, []*SceneScore{got}, nil)
	if sum.Overall.UnmeasuredDamage != 1 {
		t.Errorf("UnmeasuredDamage = %d, want 1", sum.Overall.UnmeasuredDamage)
	}
}

// Pixels proven painted over protected content stay a hard failure even when another observation
// of the same scene has no capture.
func TestScoreKeepsProvenPaintedDamageWhenAnObservationIsMissing(t *testing.T) {
	anns, scs, root := scenes(t)
	const id = "synth-balloon-on-panel"
	a := anns[id]
	src := loadPNG(t, filepath.Join(root, filepath.FromSlash(scs[id].File)))
	sloppy := func(stress string) []evidence.Plate {
		return []evidence.Plate{{
			Rect:     evidence.Rect{X0: 295, Y0: 55, X1: 525, Y1: 205},
			Viewport: testViewport, StressCase: stress, ScrollHeight: 20, ClientHeight: 20,
		}}
	}
	sc := evScene(a, append(sloppy(PrimaryStressCase), sloppy("long-latin")...))
	sc.Observations = []evidence.Observation{
		{Viewport: testViewport, StressCase: PrimaryStressCase, Rendered: "p.png", Concealed: "p.png"},
		{Viewport: testViewport, StressCase: "long-latin", Rendered: "missing.png", Concealed: "missing.png"},
	}
	painted := paintOver(src, sloppy(PrimaryStressCase), color.RGBA{255, 255, 255, 255})
	open := func(rel string) image.Image {
		if rel == "p.png" {
			return painted
		}
		return nil
	}
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs[id], Captures{Source: src, Open: open})
	if err != nil {
		t.Fatal(err)
	}
	if got.Damage.IsMeasured() || got.Damage.ProtectedHit == 0 {
		t.Errorf("one missing observation: want unmeasured with the proven pixels kept, got %+v", got.Damage)
	}
	var named bool
	for _, f := range got.Failures {
		named = named || strings.Contains(f, "protected-area damage")
	}
	if !named {
		t.Errorf("proven painted damage must still fail the scene: %v", got.Failures)
	}
}

// ---- self-diagnosis guard ---------------------------------------------------

func selfFixture(background string) (source, rendered image.Image, sc evidence.Scene) {
	src := calBackground(background)
	calFill(src, image.Rect(30, 20, 60, 30), color.Black) // 300 px of lettering
	sc = evidence.Scene{
		SceneID: "self-" + background, ImageWidth: calW, ImageHeight: calH,
		Plates: []evidence.Plate{{
			Rect: evidence.Rect{X0: 20, Y0: 10, X1: 80, Y1: 40}, Viewport: testViewport, StressCase: PrimaryStressCase,
			ScrollHeight: 20, ClientHeight: 20,
		}},
	}
	// The plate is transparent: the render is the source, so the cover genuinely does not cover.
	return src, src, sc
}

func TestSelfDiagnosisDoesNotAssertACoverThatIsNotCoveringOnGradientOrTexture(t *testing.T) {
	findings := func(background string) (*SelfScore, string) {
		source, rendered, sc := selfFixture(background)
		got := SelfDiagnose(run(evidence.EditionDesktop, sc), &sc, source, rendered)
		return got, strings.Join(got.Findings, "\n")
	}
	flat, flatText := findings("flat")
	if !strings.Contains(flatText, "still visible") || flat.ResidualUnmeasured != "" {
		t.Fatalf("control: an uncovered flat plate must be reported, got %q (%+v)", flatText, flat)
	}
	for _, background := range []string{"gradient", "texture"} {
		got, text := findings(background)
		if strings.Contains(text, "still visible") || strings.Contains(text, "stop short") {
			t.Errorf("%s: ink findings asserted on background structure: %q", background, text)
		}
		if got.ResidualUnmeasured == "" || got.InkPxUnderPlates != 0 || got.ResidualUnderPlates != 0 {
			t.Errorf("%s: the residual must be marked unmeasured and carry no number: %+v", background, got)
		}
	}
}

func TestSelfFindingNamesBackgroundContrastNotReadability(t *testing.T) {
	out := selfFindings(&SelfScore{MinBackgroundContrast: 12})
	if len(out) != 1 || !strings.Contains(out[0], "low background contrast") {
		t.Fatalf("findings: %v", out)
	}
	if strings.Contains(out[0], "barely readable") {
		t.Error("a plate rectangle's luma separation is not a readability verdict")
	}
	if got := findingClass(out[0]); got != "plate rectangle has low background contrast" {
		t.Errorf("finding class: %q", got)
	}
}

// Across a whole line the ramp is a steep gradient: the background itself lands in the ink mask on
// both sides and the single-tone estimate answered 90.86% with the lettering exactly removed. With
// the declared mask the same capture measures 0%; without it, no percentage is invented.
func TestScoreResidualOnASteepGradientNeedsTheMask(t *testing.T) {
	anns, scs, _ := scenes(t)
	const id = "synth-uniform-paper"
	a := anns[id]
	w, h := a.ImageWidth, a.ImageHeight
	ramp := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			v := uint8(40 + x*200/w)
			ramp.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	src := calCopy(ramp)
	lettering := truth.NewMask(w, h)
	for _, g := range a.Groups {
		for _, line := range g.Lines {
			x0, y0, x1, y1 := line.Bounds()
			core := image.Rect(x0+4, y0+6, x1-4, y1-6)
			calFill(src, core, color.Black)
			for y := core.Min.Y; y < core.Max.Y; y++ {
				for x := core.Min.X; x < core.Max.X; x++ {
					lettering.Set(x, y)
				}
			}
		}
	}
	sc := evScene(a, perfectPlates(a, PrimaryStressCase))
	sc.Observations = []evidence.Observation{{Viewport: testViewport, StressCase: PrimaryStressCase, Rendered: "hidden.png", Concealed: "hidden.png"}}
	open := func(string) image.Image { return ramp }

	with, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs[id], Captures{Source: src, Open: open, Lettering: lettering})
	if err != nil {
		t.Fatal(err)
	}
	if !with.Residual.IsMeasured() || with.Residual.Residual != 0 || with.Residual.Basis != BasisKnownMask {
		t.Errorf("with the mask: %+v", with.Residual)
	}
	without, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs[id], Captures{Source: src, Open: open})
	if err != nil {
		t.Fatal(err)
	}
	if without.Residual.IsMeasured() || without.Residual.Residual != 0 || without.Residual.InkPx != 0 || !strings.Contains(without.Residual.Reason, "varies") {
		t.Errorf("a steep gradient with no mask must be unmeasured with no number: %+v", without.Residual)
	}
	sum := Aggregate("t", evidence.EditionDesktop, []*SceneScore{without}, nil)
	if sum.Overall.UnmeasuredConcealment != 1 || sum.Overall.WorstResidual != 0 {
		t.Errorf("the unmeasured scene must be counted and must not move the worst-of: %+v", sum.Overall)
	}
}
