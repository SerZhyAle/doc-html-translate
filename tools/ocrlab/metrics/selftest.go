package metrics

import (
	"fmt"
	"image"
	"sort"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// Self-diagnosis: what can be measured about a scene with no ground truth at all.
//
// Annotating an image is slow and human, and a corpus grows faster than a person can annotate
// it. But the two questions this product actually has to answer - "is the original lettering
// hidden?" and "is the replacement legible and in the right place?" - do not need to know what
// the image says. They need the source pixels, the rendered pixels and the plate rectangles, all
// of which every run already records.
//
// So an unannotated scene is not skipped: it gets these measures instead. They cannot say
// whether the recognizer read the right words, and they never enter the quality gate - that is
// what the annotated holdout is for. What they do is let the measure-look-fix loop run on a
// hundred harvested images the afternoon they are harvested.

// rimBandOutside is how far outside a plate's edge the leftover-glyph check looks, as a fraction
// of the plate's shorter side. A plate that stops one stroke short of the lettering leaves
// exactly this: original ink hugging the patch.
const rimBandOutside = 0.12

// cutGlyphBound is how much escaped ink is worth telling a reader about, relative to the ink the
// plates cover. Descriptive only - nothing in this file gates anything.
const cutGlyphBound = 0.05

// SelfScore is one scene measured without truth.
type SelfScore struct {
	SceneID  string           `json:"sceneId"`
	Edition  evidence.Edition `json:"edition"`
	Viewport string           `json:"viewport"`
	Plates   int              `json:"plates"`

	// ResidualUnderPlates is the fraction of the source's ink inside the plate rectangles that
	// is still ink-like in the render. 0 is a clean cover; anything high means the plate is
	// transparent, mis-coloured or simply not there.
	ResidualUnderPlates float64 `json:"residualUnderPlates"`
	// CutGlyphInk is source ink in the rim just outside the plates that is joined, through ink,
	// to the ink the plates cover - the tops, tails and first letters a too-small patch leaves
	// behind - measured against the covered ink. 0 means every stroke the plates touch ends
	// inside them.
	//
	// It deliberately does not ask whether that rim ink survived the render. It always does: an
	// overlay is CSS over an untouched image, so no pixel outside a plate can change, and a
	// "did the rim survive" measure reads ~100% on every scene that has any artwork near its
	// lettering - including synthetic scenes built to be correct. That earlier measure was a
	// tautology and is replaced rather than kept alongside.
	CutGlyphInk float64 `json:"cutGlyphInk"`
	// InkPxUnderPlates is the sample size, so a tiny number is not read as a clean result.
	InkPxUnderPlates int `json:"inkPxUnderPlates"`

	// MinContrast is the smallest luma separation between drawn text and drawn background over
	// the scene's plates, read back from the render rather than from the sampled values.
	MinContrast float64 `json:"minContrast"`

	Clipped      int `json:"clipped"`
	PlateOverlap int `json:"plateOverlap"`
	OutOfBounds  int `json:"outOfBounds"`

	CoveredImageFraction float64 `json:"coveredImageFraction"`

	// Findings names, in words, what is wrong with this scene. This is what the loop reads.
	Findings []string `json:"findings,omitempty"`
}

// SelfDiagnose measures a scene without an annotation. source and rendered may be nil, in which
// case the pixel measures are absent and the DOM-derived ones still work.
func SelfDiagnose(run *evidence.Run, sc *evidence.Scene, source, rendered image.Image) *SelfScore {
	w, h := sc.ImageWidth, sc.ImageHeight
	out := &SelfScore{SceneID: sc.SceneID, Edition: run.Edition, Viewport: primaryViewport(run, sc)}

	plates := sc.PlatesFor(out.Viewport, PrimaryStressCase)
	out.Plates = len(plates)

	if len(plates) == 0 {
		out.Findings = append(out.Findings,
			"no plates at all - the recognizer found nothing here, so the reader sees the original text untouched")
		return out
	}
	if w <= 0 || h <= 0 {
		return out
	}

	overlay := PlateMask(plates, w, h)
	if total := w * h; total > 0 {
		out.CoveredImageFraction = float64(overlay.Area()) / float64(total)
	}

	if source != nil && rendered != nil {
		under, cut, inkPx := residualAroundPlates(source, rendered, plates, w, h)
		out.ResidualUnderPlates, out.CutGlyphInk, out.InkPxUnderPlates = under, cut, inkPx
		out.MinContrast = RenderedContrast(rendered, plates, w, h).MinLuma
	}

	// Layout facts, from the DOM rather than from a guess.
	for i, p := range plates {
		if p.Clipped() {
			out.Clipped++
		}
		if p.Rect.X0 < 0 || p.Rect.Y0 < 0 || p.Rect.X1 > w || p.Rect.Y1 > h {
			out.OutOfBounds++
		}
		pm := p.Rect.Region().Rasterize(w, h)
		for j := i + 1; j < len(plates); j++ {
			if pm.IntersectArea(plates[j].Rect.Region().Rasterize(w, h)) > 0 {
				out.PlateOverlap++
			}
		}
	}
	// Clipping across the stress cases too - the case a page passes in its own language and
	// fails in a longer one.
	for _, name := range sortedStress(sc) {
		if name == PrimaryStressCase {
			continue
		}
		for _, p := range sc.PlatesFor(out.Viewport, name) {
			if p.Clipped() {
				out.Findings = append(out.Findings,
					fmt.Sprintf("a plate clips under the %s replacement", name))
				break
			}
		}
	}

	out.Findings = append(out.Findings, selfFindings(out)...)
	return out
}

// selfFindings turns the numbers into sentences. The bounds here are descriptive - they decide
// what is worth a reader's attention, not what passes - and nothing in this file gates anything.
func selfFindings(s *SelfScore) []string {
	var out []string
	if s.InkPxUnderPlates > 200 {
		if s.ResidualUnderPlates > 0.25 {
			out = append(out, fmt.Sprintf(
				"%.0f%% of the original lettering under the plates is still visible - the cover is not covering",
				s.ResidualUnderPlates*100))
		}
		if s.CutGlyphInk > cutGlyphBound {
			out = append(out, fmt.Sprintf(
				"strokes the plates cover carry on past their edges, worth %.0f%% of the covered ink - the patches stop short of the glyphs",
				s.CutGlyphInk*100))
		}
	}
	if s.MinContrast > 0 && s.MinContrast < 40 {
		out = append(out, fmt.Sprintf(
			"a plate's text and background differ by only %.0f luma - the replacement is barely readable", s.MinContrast))
	}
	if s.Clipped > 0 {
		out = append(out, fmt.Sprintf("%d plate(s) clip their own recognized text", s.Clipped))
	}
	if s.PlateOverlap > 0 {
		out = append(out, fmt.Sprintf("%d plate pair(s) overlap each other", s.PlateOverlap))
	}
	if s.OutOfBounds > 0 {
		out = append(out, fmt.Sprintf("%d plate(s) extend past the image", s.OutOfBounds))
	}
	return out
}

// residualAroundPlates measures the original ink under each plate by comparing the source with
// the render at the same pixels, and the ink that escapes past the plate edges by following the
// source's own strokes across them.
func residualAroundPlates(source, rendered image.Image, plates []evidence.Plate, w, h int) (under, cut float64, inkPx int) {
	inner := truth.NewMask(w, h)
	outer := truth.NewMask(w, h)
	for _, p := range plates {
		r := p.Rect
		band := int(float64(min(r.Width(), r.Height())) * rimBandOutside)
		if band < 2 {
			band = 2
		}
		inner.Or(truth.Box("", r.X0, r.Y0, r.X1, r.Y1).Rasterize(w, h))
		outer.Or(truth.Box("", r.X0-band, r.Y0-band, r.X1+band, r.Y1+band).Rasterize(w, h))
	}
	// The rim is the grown box minus the plates themselves.
	rim := truth.NewMask(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if outer.At(x, y) && !inner.At(x, y) {
				rim.Set(x, y)
			}
		}
	}

	srcUnder := inkWithin(source, inner, w, h)
	renUnder := inkWithin(rendered, inner, w, h)
	inkPx = srcUnder.Area()
	if inkPx > 0 {
		under = float64(srcUnder.IntersectArea(renUnder)) / float64(inkPx)
	}

	// One ink threshold has to span the plate edge or a stroke would break at the seam, so the
	// escaped-ink measure takes its ink over the grown area as a whole rather than per side. The
	// render is not consulted: nothing outside a plate can change.
	all := inkWithin(source, outer, w, h)
	if base := all.IntersectArea(inner); base > 0 {
		cut = float64(cutGlyphPixels(all, inner, rim, w, h)) / float64(base)
	}
	return under, cut, inkPx
}

// cutGlyphPixels counts rim ink joined, through ink, to ink under a plate. That is the shape a
// patch leaves when it stops one stroke short: the covered half of a letter ends at the plate
// edge and the same stroke continues outside it. Artwork merely standing next to the lettering
// is not connected to it and is not counted, which is the whole point of measuring it this way.
func cutGlyphPixels(ink, inner, rim *truth.Mask, w, h int) int {
	seen := truth.NewMask(w, h)
	var stack []int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ink.At(x, y) && inner.At(x, y) {
				seen.Set(x, y)
				stack = append(stack, y*w+x)
			}
		}
	}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		px, py := p%w, p/w
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := px+dx, py+dy
				if nx < 0 || ny < 0 || nx >= w || ny >= h || seen.At(nx, ny) || !ink.At(nx, ny) {
					continue
				}
				seen.Set(nx, ny)
				stack = append(stack, ny*w+nx)
			}
		}
	}
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if seen.At(x, y) && rim.At(x, y) {
				n++
			}
		}
	}
	return n
}

// inkWithin marks pixels inside a mask that stand out from that region's own median luma - the
// same local rule the annotated concealment metric uses, so the two numbers mean the same thing.
func inkWithin(img image.Image, area *truth.Mask, w, h int) *truth.Mask {
	out := truth.NewMask(w, h)
	if img == nil {
		return out
	}
	var lumas []int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !area.At(x, y) {
				continue
			}
			cr, cg, cb, _ := img.At(x, y).RGBA()
			lumas = append(lumas, luma(cr, cg, cb))
		}
	}
	if len(lumas) == 0 {
		return out
	}
	sort.Ints(lumas)
	med := lumas[len(lumas)/2]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !area.At(x, y) {
				continue
			}
			cr, cg, cb, _ := img.At(x, y).RGBA()
			if absInt(luma(cr, cg, cb)-med) > inkLumaDelta {
				out.Set(x, y)
			}
		}
	}
	return out
}

func sortedStress(sc *evidence.Scene) []string {
	names := sc.StressCases()
	sort.Strings(names)
	return names
}

// SelfSummary aggregates the annotation-free measures, so a harvest of a hundred images produces
// one line per failure class instead of a hundred records to read.
type SelfSummary struct {
	Scenes            int            `json:"scenes"`
	ScenesWithNoPlate int            `json:"scenesWithNoPlate"`
	ScenesWithFinding int            `json:"scenesWithFinding"`
	MeanResidual      float64        `json:"meanResidual"`
	WorstResidual     float64        `json:"worstResidual"`
	WorstResidualID   string         `json:"worstResidualScene,omitempty"`
	MeanCutGlyphInk   float64        `json:"meanCutGlyphInk"`
	TotalClipped      int            `json:"totalClipped"`
	FindingCounts     map[string]int `json:"findingCounts"`
}

// SummarizeSelf folds the per-scene self scores into one picture of the batch.
func SummarizeSelf(scores []*SelfScore) *SelfSummary {
	s := &SelfSummary{Scenes: len(scores), FindingCounts: map[string]int{}}
	var res, cuts []float64
	for _, sc := range scores {
		if sc.Plates == 0 {
			s.ScenesWithNoPlate++
		}
		if len(sc.Findings) > 0 {
			s.ScenesWithFinding++
		}
		if sc.InkPxUnderPlates > 200 {
			res = append(res, sc.ResidualUnderPlates)
			cuts = append(cuts, sc.CutGlyphInk)
			if sc.ResidualUnderPlates > s.WorstResidual {
				s.WorstResidual, s.WorstResidualID = sc.ResidualUnderPlates, sc.SceneID
			}
		}
		s.TotalClipped += sc.Clipped
		for _, f := range sc.Findings {
			s.FindingCounts[findingClass(f)]++
		}
	}
	s.MeanResidual, s.MeanCutGlyphInk = mean(res), mean(cuts)
	return s
}

// findingClass strips the numbers out of a finding so the counts group by cause.
func findingClass(f string) string {
	switch {
	case contains(f, "no plates at all"):
		return "no plates at all"
	case contains(f, "still visible"):
		return "original lettering still visible under the plates"
	case contains(f, "stop short"):
		return "patches stop short of the glyphs"
	case contains(f, "barely readable"):
		return "replacement text barely readable"
	case contains(f, "clip their own"):
		return "plate clips its own text"
	case contains(f, "clips under the"):
		return "plate clips under a longer translation"
	case contains(f, "overlap each other"):
		return "plates overlap each other"
	case contains(f, "past the image"):
		return "plate extends past the image"
	default:
		return f
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
