package metrics

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/synth"
	"doc-html-translate/tools/ocrlab/truth"
)

// ---- fixtures ---------------------------------------------------------------

const testViewport = "desktop"

// scenes generates the diagnostic scenes once per test binary and indexes them by ID. Every
// metric test binds to these rather than to hand-typed geometry: their truth is exact because
// the lab drew them, so a disagreement is the metric's fault and not the fixture's.
func scenes(t *testing.T) (map[string]*truth.Annotation, map[string]*corpus.Scene, string) {
	t.Helper()
	root := t.TempDir()
	sc, anns, err := synth.Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*truth.Annotation{}
	for _, a := range anns {
		byID[a.SceneID] = a
	}
	scByID := map[string]*corpus.Scene{}
	for i := range sc {
		scByID[sc[i].ID] = &sc[i]
	}
	return byID, scByID, root
}

// perfectPlates builds the plate set a flawless overlay would produce: one plate exactly over
// each group's own text, carrying the transcript.
func perfectPlates(a *truth.Annotation, stress string) []evidence.Plate {
	var out []evidence.Plate
	for _, g := range a.Groups {
		x0, y0, x1, y1 := g.Bounds.Bounds()
		out = append(out, evidence.Plate{
			Text:         g.Transcript,
			Rect:         evidence.Rect{X0: x0, Y0: y0, X1: x1, Y1: y1},
			Viewport:     testViewport,
			StressCase:   stress,
			Mode:         evidence.ModeFill,
			ScrollHeight: 20,
			ClientHeight: 20,
		})
	}
	return out
}

func run(edition evidence.Edition, sc evidence.Scene) *evidence.Run {
	return &evidence.Run{
		SchemaVersion: evidence.SchemaVersion,
		RunID:         "test",
		Edition:       edition,
		Viewports:     []evidence.Viewport{{Name: testViewport, Width: 1280, Height: 800, DeviceScaleFactor: 1}},
		Scenes:        []evidence.Scene{sc},
	}
}

func evScene(a *truth.Annotation, plates []evidence.Plate) evidence.Scene {
	return evidence.Scene{
		SceneID:     a.SceneID,
		ImageWidth:  a.ImageWidth,
		ImageHeight: a.ImageHeight,
		Plates:      plates,
	}
}

func loadPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// paintOver returns a copy of src with every plate rect filled flat - a perfectly concealing
// overlay, which is what "residual = 0" has to mean.
func paintOver(src image.Image, plates []evidence.Plate, fill color.RGBA) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	for _, p := range plates {
		draw.Draw(dst, image.Rect(p.Rect.X0, p.Rect.Y0, p.Rect.X1, p.Rect.Y1),
			&image.Uniform{fill}, image.Point{}, draw.Src)
	}
	return dst
}

// ---- geometry ---------------------------------------------------------------

func TestIoUAndContainment(t *testing.T) {
	a := truth.Box("a", 10, 10, 50, 50)
	same := truth.Box("b", 10, 10, 50, 50)
	far := truth.Box("c", 60, 60, 90, 90)
	half := truth.Box("d", 10, 10, 50, 30)

	if got := IoU(a, same, 100, 100); math.Abs(got-1) > 1e-9 {
		t.Errorf("identical boxes: IoU = %v, want 1", got)
	}
	if got := IoU(a, far, 100, 100); got != 0 {
		t.Errorf("disjoint boxes: IoU = %v, want 0", got)
	}
	if got := Contained(half, a, 100, 100); math.Abs(got-1) > 1e-9 {
		t.Errorf("a box inside another: containment = %v, want 1", got)
	}
	if got := Contained(a, half, 100, 100); math.Abs(got-0.5) > 0.01 {
		t.Errorf("half-covered box: containment = %v, want ~0.5", got)
	}
}

func TestEdgeErrorIsNormalized(t *testing.T) {
	want := truth.Box("w", 0, 0, 100, 50)
	got := truth.Box("g", 10, 5, 100, 50) // 10 px right on a 100-wide box, 5 down on a 50-tall one
	e := Edges(got, want)
	if math.Abs(e.Left-0.1) > 1e-9 {
		t.Errorf("left = %v, want 0.1", e.Left)
	}
	if math.Abs(e.Top-0.1) > 1e-9 {
		t.Errorf("top = %v, want 0.1", e.Top)
	}
	if math.Abs(e.Worst-0.1) > 1e-9 {
		t.Errorf("worst = %v, want 0.1", e.Worst)
	}
}

// ---- recognition ------------------------------------------------------------

func TestCERAndWER(t *testing.T) {
	if got := CER("abc", "abc"); got != 0 {
		t.Errorf("identical strings: CER = %v, want 0", got)
	}
	if got := WER("one two", "one two"); got != 0 {
		t.Errorf("identical strings: WER = %v, want 0", got)
	}
	// Normalization must absorb case, spacing and invented punctuation, or every score would be
	// about punctuation rather than reading.
	if got := CER("Hello,  World!", "hello world"); got != 0 {
		t.Errorf("case/space/punctuation must normalize away, CER = %v", got)
	}
	if got := CER("cat", "car"); math.Abs(got-1.0/3) > 1e-9 {
		t.Errorf("one substitution in three: CER = %v, want 1/3", got)
	}
	if got := CER("something", ""); got != 1 {
		t.Errorf("output against an empty reference: CER = %v, want 1", got)
	}
	if got := CER("", ""); got != 0 {
		t.Errorf("both empty: CER = %v, want 0", got)
	}
}

// An illegible group must leave the denominator entirely rather than count as a miss.
func TestDetectionSkipsAmbiguousGroups(t *testing.T) {
	a := &truth.Annotation{
		ImageWidth: 100, ImageHeight: 100, Ambiguity: truth.AmbiguityClear,
		Groups: []truth.Group{
			{ID: "clear", Bounds: truth.Box("", 0, 0, 40, 20), Ambiguity: truth.AmbiguityClear},
			{ID: "gone", Bounds: truth.Box("", 0, 40, 40, 60), Ambiguity: truth.AmbiguityIllegible},
		},
	}
	plates := []evidence.Plate{{Rect: evidence.Rect{X0: 0, Y0: 0, X1: 40, Y1: 20}}}
	d := Detection(plates, a, a.Groups, 100, 100, DefaultDetectionIoU)

	if d.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", d.Skipped)
	}
	if d.FN != 0 {
		t.Errorf("an illegible group must not count as a miss, FN = %d", d.FN)
	}
	if d.Recall != 1 {
		t.Errorf("recall over the legible group = %v, want 1", d.Recall)
	}
}

// ---- grouping ---------------------------------------------------------------

// The merge trap: one plate spanning both balloons of the adjacent-balloon scene.
func TestGroupingCountsAMerge(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-adjacent-balloons"]
	if a == nil {
		t.Fatal("scene missing")
	}
	x0, y0, _, _ := a.Groups[0].Bounds.Bounds()
	_, _, x1, y1 := a.Groups[1].Bounds.Bounds()
	one := []evidence.Plate{{
		Rect: evidence.Rect{X0: x0, Y0: y0, X1: x1, Y1: y1}, Viewport: testViewport,
	}}
	g, _ := Grouping(one, a.Groups, a.ImageWidth, a.ImageHeight)
	if g.Merges != 1 {
		t.Errorf("one plate over both balloons: Merges = %d, want 1", g.Merges)
	}

	// The correct overlay, two plates, must report no merge.
	g2, _ := Grouping(perfectPlates(a, PrimaryStressCase), a.Groups, a.ImageWidth, a.ImageHeight)
	if g2.Merges != 0 {
		t.Errorf("one plate per balloon: Merges = %d, want 0", g2.Merges)
	}
	if g2.Matched != 2 {
		t.Errorf("Matched = %d, want 2", g2.Matched)
	}
}

func TestGroupingCountsASplit(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-uniform-paper"]
	x0, y0, x1, y1 := a.Groups[0].Bounds.Bounds()
	mid := (y0 + y1) / 2
	torn := []evidence.Plate{
		{Rect: evidence.Rect{X0: x0, Y0: y0, X1: x1, Y1: mid + 5}, Viewport: testViewport},
		{Rect: evidence.Rect{X0: x0, Y0: mid - 5, X1: x1, Y1: y1}, Viewport: testViewport},
	}
	g, _ := Grouping(torn, a.Groups, a.ImageWidth, a.ImageHeight)
	if g.Splits != 1 {
		t.Errorf("a paragraph torn into two plates: Splits = %d, want 1", g.Splits)
	}
}

func TestOrderInversions(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-two-columns"]
	perfect := perfectPlates(a, PrimaryStressCase)
	g, _ := Grouping(perfect, a.Groups, a.ImageWidth, a.ImageHeight)
	if g.OrderInversions != 0 {
		t.Errorf("plates in the annotated order: inversions = %d, want 0", g.OrderInversions)
	}
}

// ---- placement --------------------------------------------------------------

func TestPlacementPerfectAndShifted(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-uniform-paper"]

	matches, _, _ := MatchPlates(perfectPlates(a, PrimaryStressCase), a.Groups, a.ImageWidth, a.ImageHeight)
	p := Placement(matches)
	if math.Abs(p.MeanIoU-1) > 1e-9 {
		t.Errorf("plates exactly over their groups: mean IoU = %v, want 1", p.MeanIoU)
	}
	if p.WorstEdgeError != 0 {
		t.Errorf("worst edge error = %v, want 0", p.WorstEdgeError)
	}

	shifted := perfectPlates(a, PrimaryStressCase)
	for i := range shifted {
		shifted[i].Rect.X0 += 20
		shifted[i].Rect.X1 += 20
	}
	m2, _, _ := MatchPlates(shifted, a.Groups, a.ImageWidth, a.ImageHeight)
	if p2 := Placement(m2); p2.MeanIoU >= p.MeanIoU {
		t.Errorf("a 20 px shift must lower IoU: %v vs %v", p2.MeanIoU, p.MeanIoU)
	}
}

func TestDriftIsZeroWhenGeometryAgrees(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-uniform-paper"]
	matches, _, _ := MatchPlates(perfectPlates(a, PrimaryStressCase), a.Groups, a.ImageWidth, a.ImageHeight)

	same := map[string][]Match{"desktop": matches, "tablet": matches}
	if d, _ := Drift(same, a.ImageWidth, a.ImageHeight); d != 0 {
		t.Errorf("identical geometry at two viewports: drift = %v, want 0", d)
	}

	moved := make([]Match, len(matches))
	copy(moved, matches)
	for i := range moved {
		moved[i].Plate.Rect.Y0 += 30
		moved[i].Plate.Rect.Y1 += 30
	}
	d, group := Drift(map[string][]Match{"desktop": matches, "tablet": moved}, a.ImageWidth, a.ImageHeight)
	if d == 0 {
		t.Error("a plate that moves between viewports must register drift")
	}
	if group == "" {
		t.Error("drift must name the group that moved")
	}
}

// ---- concealment ------------------------------------------------------------

func TestResidualInkAndCoverage(t *testing.T) {
	anns, scs, root := scenes(t)
	a := anns["synth-uniform-paper"]
	src := loadPNG(t, filepath.Join(root, filepath.FromSlash(scs["synth-uniform-paper"].File)))
	plates := perfectPlates(a, PrimaryStressCase)
	g := a.Groups[0]

	// Against the untouched source, every original glyph is still there.
	untouched := ResidualInk(src, src, g, a.ImageWidth, a.ImageHeight)
	if untouched.InkPx == 0 {
		t.Fatal("the ink mask found no lettering - the measurement cannot mean anything")
	}
	if untouched.Residual < 0.5 {
		t.Errorf("nothing was painted, residual = %v, want > 0.5", untouched.Residual)
	}

	// Against a copy with the plates painted flat, none of it is.
	painted := paintOver(src, plates, color.RGBA{250, 249, 246, 255})
	concealed := ResidualInk(src, painted, g, a.ImageWidth, a.ImageHeight)
	if concealed.Residual != 0 {
		t.Errorf("fully painted over, residual = %v, want 0", concealed.Residual)
	}

	if c := Covered(plates, g, a.ImageWidth, a.ImageHeight); math.Abs(c-1) > 1e-9 {
		t.Errorf("a plate exactly over its text: covered = %v, want 1", c)
	}
	if c := Covered(nil, g, a.ImageWidth, a.ImageHeight); c != 0 {
		t.Errorf("no plates: covered = %v, want 0", c)
	}
}

// A plate whose text and background end up the same colour is unreadable however well it
// conceals, so the contrast check has to read the rendered pixels rather than trust the
// sampled values in the evidence.
func TestRenderedContrastCatchesInvisibleText(t *testing.T) {
	w, h := 100, 60
	plates := []evidence.Plate{{Rect: evidence.Rect{X0: 10, Y0: 10, X1: 90, Y1: 50}}}

	flat := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(flat, flat.Bounds(), &image.Uniform{color.RGBA{200, 200, 200, 255}}, image.Point{}, draw.Src)
	if got := RenderedContrast(flat, plates, w, h); got.MinLuma > 5 {
		t.Errorf("a uniform plate has no readable text: min separation = %v, want ~0", got.MinLuma)
	}

	legible := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(legible, legible.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	draw.Draw(legible, image.Rect(20, 20, 80, 40), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)
	if got := RenderedContrast(legible, plates, w, h); got.MinLuma < 100 {
		t.Errorf("black on white: min separation = %v, want a large number", got.MinLuma)
	}
}

// ---- damage -----------------------------------------------------------------

func TestDamageNamesTheProtectedRegionItHit(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]
	if len(a.Protected) == 0 {
		t.Fatal("the balloon scene must protect something")
	}

	// The correct overlay paints only inside the permitted area.
	good := perfectPlates(a, PrimaryStressCase)
	if d := Damage(good, a, a.ImageWidth, a.ImageHeight); d.ProtectedHit != 0 {
		t.Errorf("a plate inside the balloon must not damage its outline, hit = %d px in %s",
			d.ProtectedHit, d.WorstProtectedRegion)
	}

	// A block fill over the whole balloon, outline included: the classic defect.
	sloppy := []evidence.Plate{{
		Rect: evidence.Rect{X0: 295, Y0: 55, X1: 525, Y1: 205}, Viewport: testViewport,
	}}
	d := Damage(sloppy, a, a.ImageWidth, a.ImageHeight)
	if d.ProtectedHit == 0 {
		t.Fatal("a plate covering the balloon outline must be reported as damage")
	}
	if d.WorstProtectedRegion != "balloon-outline" {
		t.Errorf("damage must name the region it hit, got %q", d.WorstProtectedRegion)
	}
	if d.OutsideReplaceArea == 0 {
		t.Error("paint outside the permitted area must be counted")
	}
}

// ---- replacement ------------------------------------------------------------

func TestReplacementReadsClippingFromTheDOM(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-uniform-paper"]
	plates := perfectPlates(a, PrimaryStressCase)
	matches, _, _ := MatchPlates(plates, a.Groups, a.ImageWidth, a.ImageHeight)

	if r := Replacement(plates, a.Groups, matches, a.ImageWidth, a.ImageHeight); r.Clipped != 0 {
		t.Errorf("scrollHeight == clientHeight: Clipped = %d, want 0", r.Clipped)
	}

	plates[0].ScrollHeight = plates[0].ClientHeight + 40
	if r := Replacement(plates, a.Groups, matches, a.ImageWidth, a.ImageHeight); r.Clipped != 1 {
		t.Errorf("scrollHeight > clientHeight: Clipped = %d, want 1", r.Clipped)
	}
}

func TestReplacementCatchesCrossGroupAndOutOfBounds(t *testing.T) {
	anns, _, _ := scenes(t)
	a := anns["synth-two-columns"]
	plates := perfectPlates(a, PrimaryStressCase)
	matches, _, _ := MatchPlates(plates, a.Groups, a.ImageWidth, a.ImageHeight)
	if r := Replacement(plates, a.Groups, matches, a.ImageWidth, a.ImageHeight); r.CrossGroupOverlap != 0 {
		t.Errorf("two tidy columns: CrossGroupOverlap = %d, want 0", r.CrossGroupOverlap)
	}

	// Grow the left column's plate until it sits on the right column.
	_, _, rx1, _ := a.Groups[1].Bounds.Bounds()
	plates[0].Rect.X1 = rx1
	m2, _, _ := MatchPlates(plates, a.Groups, a.ImageWidth, a.ImageHeight)
	if r := Replacement(plates, a.Groups, m2, a.ImageWidth, a.ImageHeight); r.CrossGroupOverlap == 0 {
		t.Error("a plate reaching into the other column must be reported")
	}

	off := perfectPlates(a, PrimaryStressCase)
	off[0].Rect.X1 = a.ImageWidth + 50
	m3, _, _ := MatchPlates(off, a.Groups, a.ImageWidth, a.ImageHeight)
	r := Replacement(off, a.Groups, m3, a.ImageWidth, a.ImageHeight)
	if r.OutOfBounds != 1 || r.OutOfBoundsPx == 0 {
		t.Errorf("a plate past the image edge: OutOfBounds = %d, px = %d", r.OutOfBounds, r.OutOfBoundsPx)
	}
}

// ---- score ------------------------------------------------------------------

// The rule the lab rests on, enforced at the one place it must be: a non-truth annotation
// produces an error, never a zero that lands in an aggregate.
func TestScoreRefusesNonTruth(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-uniform-paper"]
	draft := *a
	draft.Origin = truth.OriginOCRSeed

	sc := evScene(a, perfectPlates(a, PrimaryStressCase))
	_, err := Score(run(evidence.EditionDesktop, sc), &sc, &draft, scs["synth-uniform-paper"], nil, nil)
	if err == nil {
		t.Fatal("an OCR-seeded annotation must not be scorable")
	}
	if !isNotTruth(err) {
		t.Errorf("want ErrNotTruth, got %v", err)
	}
}

func isNotTruth(err error) bool {
	for e := err; e != nil; {
		if e == ErrNotTruth {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

func TestScoreOnAPerfectOverlay(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]

	plates := append(perfectPlates(a, PrimaryStressCase), perfectPlates(a, "long-latin")...)
	sc := evScene(a, plates)
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Detection.Recall != 1 {
		t.Errorf("recall = %v, want 1", got.Detection.Recall)
	}
	if got.Text.MeanCER != 0 {
		t.Errorf("plates carrying the transcript: CER = %v, want 0", got.Text.MeanCER)
	}
	if got.Damage.ProtectedHit != 0 {
		t.Errorf("protected damage = %d, want 0", got.Damage.ProtectedHit)
	}
	if len(got.Failures) != 0 {
		t.Errorf("a flawless overlay must have no hard failures, got %v", got.Failures)
	}
	if _, ok := got.Stress["long-latin"]; !ok {
		t.Error("every stress case in the evidence must appear in the breakdown")
	}
}

// A failure must be named in words, because "the number went down" is not actionable.
func TestScoreNamesItsFailures(t *testing.T) {
	anns, scs, _ := scenes(t)
	a := anns["synth-balloon-on-panel"]
	sloppy := []evidence.Plate{{
		Text:     a.Groups[0].Transcript,
		Rect:     evidence.Rect{X0: 295, Y0: 55, X1: 525, Y1: 205},
		Viewport: testViewport, StressCase: PrimaryStressCase,
		ScrollHeight: 200, ClientHeight: 100,
	}}
	sc := evScene(a, sloppy)
	got, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs["synth-balloon-on-panel"], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Failures) == 0 {
		t.Fatal("a plate over the outline and a clipped translation must both be named")
	}
	var sawDamage, sawClip bool
	for _, f := range got.Failures {
		if strings.Contains(f, "protected-area damage") {
			sawDamage = true
		}
		if strings.Contains(f, "clipped") {
			sawClip = true
		}
	}
	if !sawDamage {
		t.Errorf("failures must name the protected damage, got %v", got.Failures)
	}
	if !sawClip {
		t.Errorf("failures must name the clipping, got %v", got.Failures)
	}
}

func TestAggregateKeepsSkippedVisible(t *testing.T) {
	anns, scs, _ := scenes(t)
	var got []*SceneScore
	for _, id := range []string{"synth-uniform-paper", "synth-two-columns"} {
		a := anns[id]
		sc := evScene(a, perfectPlates(a, PrimaryStressCase))
		s, err := Score(run(evidence.EditionDesktop, sc), &sc, a, scs[id], nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	skipped := []Skipped{{SceneID: "commons-blue-speech-bubble", Reason: "no annotation"}}
	sum := Aggregate("test", evidence.EditionDesktop, got, skipped)

	if sum.Overall.Scenes != 2 {
		t.Errorf("overall scenes = %d, want 2", sum.Overall.Scenes)
	}
	if b := sum.ByCategory[corpus.CatDocument]; b == nil || b.Scenes != 2 {
		t.Errorf("both scenes are documents, got %+v", b)
	}
	if b := sum.BySplit[corpus.SplitDev]; b == nil || b.Scenes != 2 {
		t.Errorf("both scenes are dev, got %+v", b)
	}
	if len(sum.Skipped) != 1 {
		t.Error("a skipped scene must survive into the summary rather than shrink the denominator")
	}
}

// The escaped-ink measure has to tell two situations apart that look identical to a "did the ink
// outside the plate survive" check: a patch that cut a letter in half, and a patch that simply
// has artwork standing next to it. The second is the common case on every comic and poster in the
// corpus, and it must score zero - an overlay is CSS over an untouched image, so ink outside a
// plate always survives and a survival test reads ~100% on a scene that is perfectly correct.
func TestCutGlyphInkFollowsStrokesNotNeighbours(t *testing.T) {
	const w, h = 200, 120
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	black := image.NewUniform(color.Black)
	// One long stroke of lettering, and an unrelated mark close enough to fall in the rim.
	draw.Draw(src, image.Rect(40, 50, 100, 70), black, image.Point{}, draw.Src)
	draw.Draw(src, image.Rect(105, 50, 118, 70), black, image.Point{}, draw.Src)

	covered := func(r evidence.Rect) image.Image {
		out := image.NewRGBA(src.Bounds())
		draw.Draw(out, out.Bounds(), src, image.Point{}, draw.Src)
		draw.Draw(out, image.Rect(r.X0, r.Y0, r.X1, r.Y1), image.NewUniform(color.White), image.Point{}, draw.Src)
		return out
	}

	whole := evidence.Rect{X0: 36, Y0: 46, X1: 104, Y1: 74}
	_, cut, ink := residualAroundPlates(src, covered(whole), []evidence.Plate{{Rect: whole}}, w, h)
	if ink == 0 {
		t.Fatal("the covered stroke must register as ink, or the measure has no sample")
	}
	if cut != 0 {
		t.Errorf("a plate that covers its whole stroke: cut = %.3f, want 0 - the neighbouring mark is not joined to it", cut)
	}

	short := evidence.Rect{X0: 36, Y0: 46, X1: 80, Y1: 74}
	_, cutShort, _ := residualAroundPlates(src, covered(short), []evidence.Plate{{Rect: short}}, w, h)
	if cutShort <= 0 {
		t.Errorf("a plate that stops mid-stroke: cut = %.3f, want above 0", cutShort)
	}
}

// No threshold may live in this package. Phase 06 sets acceptance bounds from a recorded
// baseline, and if a bound lived next to a measurement then moving the bound and moving the
// measure would look identical in a diff.
//
// The check parses each file rather than grepping it: comments here legitimately explain *why*
// a threshold does not belong, and a text scan would fail on the explanation. Only declared
// identifiers and string literals count - which is exactly where a smuggled bound would live.
func TestNoThresholdsInMetrics(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0) // 0: comments are not attached
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.Ident:
				if strings.Contains(strings.ToLower(v.Name), "threshold") {
					t.Errorf("%s declares %s; bounds belong in DEV/ocrlab/thresholds.json", f, v.Name)
				}
			case *ast.BasicLit:
				if v.Kind == token.STRING && strings.Contains(strings.ToLower(v.Value), "threshold") {
					t.Errorf("%s has the string %s; bounds belong in DEV/ocrlab/thresholds.json", f, v.Value)
				}
			}
			return true
		})
	}
}
