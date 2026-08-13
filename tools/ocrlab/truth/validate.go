package truth

import (
	"fmt"
	"sort"

	"doc-html-translate/tools/ocrlab/corpus"
)

// Rule names a class of annotation problem, mirroring corpus.Rule so the verify command can
// print both lists in one vocabulary.
type Rule string

const (
	RuleUnknownScene      Rule = "annotation-unknown-scene"
	RuleSizeMismatch      Rule = "annotation-size-mismatch"
	RuleBadRegion         Rule = "annotation-bad-region"
	RuleRegionOutside     Rule = "annotation-region-outside-image"
	RuleReplaceMissesText Rule = "replace-area-misses-its-own-text"
	RuleReplaceProtect    Rule = "replace-area-hits-protected"
	RuleBadGroupType      Rule = "annotation-bad-group-type"
	RuleBadAmbiguity      Rule = "annotation-bad-ambiguity"
	RuleBadReadingOrder   Rule = "annotation-bad-reading-order"
	RuleNoLines           Rule = "annotation-group-without-lines"
	RuleDuplicateGroup    Rule = "annotation-duplicate-group-id"
	RuleUnreviewed        Rule = "holdout-annotation-unreviewed"
	RuleSelfChecked       Rule = "holdout-annotation-self-checked"
	RuleDraftInHoldout    Rule = "holdout-annotation-is-draft"
)

// replaceMissTolerance is how much of a group's own text (in percent) may fall outside its
// replacement area before the annotation is wrong. Not zero, because a line box includes the
// font's descender slack and an area drawn to a balloon's inner edge can clip a pixel of it;
// well above zero would let a genuinely misplaced area through.
const replaceMissTolerance = 2

// groupTextRegions is the group's line regions, falling back to its bounds when a group was
// annotated at region level only (the reviewed proxy the strategic spec allows when full glyph
// masks are too costly).
func groupTextRegions(g Group) []Region {
	if len(g.Lines) > 0 {
		return g.Lines
	}
	if g.Bounds.Empty() {
		return nil
	}
	return []Region{g.Bounds}
}

// Problem is one annotation rule violation.
type Problem struct {
	Rule    Rule
	SceneID string
	Detail  string
}

func (p Problem) String() string {
	return fmt.Sprintf("%-34s %-40s %s", p.Rule, p.SceneID, p.Detail)
}

// Validate reports every way an annotation contradicts its scene or itself.
//
// The geometric rules are checked by rasterizing, not by comparing bounding boxes: the whole
// point of a polygon replace-area is that its bounding box overlaps things it does not actually
// cover, and a validator that approximated would reject exactly the careful annotations the
// spec asks for.
func Validate(a *Annotation, s *corpus.Scene) []Problem {
	var ps []Problem
	add := func(r Rule, detail string) { ps = append(ps, Problem{Rule: r, SceneID: a.SceneID, Detail: detail}) }

	if s == nil {
		add(RuleUnknownScene, "no scene with this id in the manifest")
		return ps
	}
	if !a.Ambiguity.Valid() {
		add(RuleBadAmbiguity, fmt.Sprintf("ambiguity %q is not clear/partly-illegible/illegible", a.Ambiguity))
	}

	w, h := a.ImageWidth, a.ImageHeight
	if w <= 0 || h <= 0 {
		add(RuleSizeMismatch, "annotation declares no image size")
		return ps
	}
	if s.Width > 0 && s.Height > 0 && (s.Width != w || s.Height != h) {
		add(RuleSizeMismatch, fmt.Sprintf("annotation %dx%d, manifest %dx%d", w, h, s.Width, s.Height))
	}

	protected := RasterizeAll(a.Protected, w, h)
	for i, r := range a.Protected {
		checkRegion(add, fmt.Sprintf("protected[%d]", i), r, w, h)
	}

	seenGroup := map[string]bool{}
	orders := make([]int, 0, len(a.Groups))
	for _, g := range a.Groups {
		if seenGroup[g.ID] {
			add(RuleDuplicateGroup, "group id "+g.ID+" used more than once")
		}
		seenGroup[g.ID] = true

		if !g.Type.Valid() {
			add(RuleBadGroupType, fmt.Sprintf("group %s: type %q is not in the list", g.ID, g.Type))
		}
		if g.Ambiguity != "" && !g.Ambiguity.Valid() {
			add(RuleBadAmbiguity, fmt.Sprintf("group %s: ambiguity %q", g.ID, g.Ambiguity))
		}
		if g.Transcript != "" && len(g.Lines) == 0 {
			add(RuleNoLines, "group "+g.ID+" has a transcript but no line geometry")
		}
		orders = append(orders, g.ReadingOrder)

		checkRegion(add, "group "+g.ID+" bounds", g.Bounds, w, h)
		for i, l := range g.Lines {
			checkRegion(add, fmt.Sprintf("group %s line[%d]", g.ID, i), l, w, h)
		}
		if g.ReplaceArea.Empty() {
			continue
		}
		checkRegion(add, "group "+g.ID+" replaceArea", g.ReplaceArea, w, h)

		replace := g.ReplaceArea.Rasterize(w, h)
		// The replacement area is legitimately *larger* than the text it replaces - inside a
		// speech balloon it is the whole interior, which is what gives a longer translation
		// room to reflow. So it is not required to sit inside Bounds. What it must do is cover
		// the text it claims to replace: an area that misses its own lines is an annotation
		// that permits painting where the text is not.
		if lines := groupTextRegions(g); len(lines) > 0 {
			text := RasterizeAll(lines, w, h)
			if area := text.Area(); area > 0 {
				if uncovered := text.SubtractArea(replace); uncovered*100 > area*replaceMissTolerance {
					add(RuleReplaceMissesText, fmt.Sprintf(
						"group %s: replaceArea leaves %d of %d px of the group's own text outside it",
						g.ID, uncovered, area))
				}
			}
		}
		if hit := replace.IntersectArea(protected); hit > 0 {
			add(RuleReplaceProtect, fmt.Sprintf("group %s: replaceArea covers %d px of protected content", g.ID, hit))
		}
	}

	if len(orders) > 0 {
		sort.Ints(orders)
		for i, got := range orders {
			if got != i+1 {
				add(RuleBadReadingOrder, fmt.Sprintf("reading orders must be 1..%d with no gaps, got %v", len(orders), orders))
				break
			}
		}
	}

	ps = append(ps, validateReview(a, s)...)
	return ps
}

// validateReview enforces the strategic spec's two-pairs-of-eyes rule, and only for the
// holdout: a dev scene may be annotated by one person, because a dev scene is not a gate.
func validateReview(a *Annotation, s *corpus.Scene) []Problem {
	if s.Split != corpus.SplitHoldout || !s.Scored() {
		return nil
	}
	var ps []Problem
	add := func(r Rule, detail string) { ps = append(ps, Problem{Rule: r, SceneID: a.SceneID, Detail: detail}) }

	if a.Origin == OriginOCRSeed {
		add(RuleDraftInHoldout, "a holdout scene cannot be gated by an OCR-seeded draft")
	}
	if a.Review.CheckedBy == "" {
		add(RuleUnreviewed, "holdout annotation has no independent check")
		return ps
	}
	if a.Review.CheckedBy == a.Review.AnnotatedBy {
		add(RuleSelfChecked, "the independent check must not be the annotator ("+a.Review.CheckedBy+")")
	}
	return ps
}

func checkRegion(add func(Rule, string), label string, r Region, w, h int) {
	if r.Empty() {
		return
	}
	if err := r.Valid(); err != nil {
		add(RuleBadRegion, label+": "+err.Error())
		return
	}
	if !r.Inside(w, h) {
		x0, y0, x1, y1 := r.Bounds()
		add(RuleRegionOutside, fmt.Sprintf("%s: (%d,%d)-(%d,%d) is outside the %dx%d image", label, x0, y0, x1, y1, w, h))
	}
}

// ValidateAll checks every annotation against the manifest, and reports a scored scene that has
// no annotation at all. That last rule is the one that keeps a report honest: a corpus with 200
// scenes and 12 annotations would otherwise show excellent numbers over 12 scenes.
func ValidateAll(anns map[string]*Annotation, m *corpus.Manifest) []Problem {
	var ps []Problem
	for id, a := range anns {
		ps = append(ps, Validate(a, m.Find(id))...)
	}
	for i := range m.Scenes {
		s := &m.Scenes[i]
		if !s.Scored() {
			continue
		}
		if _, ok := anns[s.ID]; !ok {
			ps = append(ps, Problem{
				Rule:    Rule(corpus.RuleAnnotationMissing),
				SceneID: s.ID,
				Detail:  "scored scene has no annotation",
			})
		}
	}
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Rule != ps[j].Rule {
			return ps[i].Rule < ps[j].Rule
		}
		return ps[i].SceneID < ps[j].SceneID
	})
	return ps
}
