package corpus

import (
	"fmt"
	"os"
	"sort"
)

// Corpus shape rules from the strategic spec, section 4.1. They are named constants rather
// than inline literals because Phase 06 must be able to cite them, and because a number that
// moves has to move visibly in a diff.
const (
	// MinScenes is the total the corpus must reach before it can gate anything.
	MinScenes = 200
	// MinHoldoutPercent is the share of scenes reserved from tuning.
	MinHoldoutPercent = 30
	// MaxHoldoutCategoryPercent stops one easy class from carrying the holdout: a set that is
	// 70% clean documents would pass on aggregate while saying nothing about balloons.
	MaxHoldoutCategoryPercent = 35
)

// categoryMinimum is the per-category floor from the section 4.1 table. Counts overlap by
// label - a Cyrillic comic page counts under both comic and script - so the sum exceeds
// MinScenes and that is correct.
var categoryMinimum = map[Category]int{
	CatDocument: 35,
	CatPoster:   25,
	CatComic:    60,
	CatCartoon:  20,
	CatTexture:  20,
	CatDegraded: 20,
	CatScript:   20,
}

// CategoryMinimum returns the floor for a category, 0 for an unknown one.
func CategoryMinimum(c Category) int { return categoryMinimum[c] }

// Rule names a class of problem. Reporting the rule separately from the detail lets the caller
// group ten missing files into one heading instead of ten unrelated lines.
type Rule string

const (
	RuleDuplicateID       Rule = "duplicate-id"
	RuleBadLicence        Rule = "bad-licence"
	RuleNoAttribution     Rule = "missing-attribution"
	RuleNotVerified       Rule = "licence-not-human-verified"
	RuleNoSource          Rule = "missing-source-url"
	RuleBadDerivedFrom    Rule = "unknown-derived-from"
	RuleBadCategory       Rule = "bad-category"
	RuleBadSplit          Rule = "bad-split"
	RuleNoCategory        Rule = "no-category"
	RuleHashUnmeasured    Rule = "hash-not-measured"
	RuleHashMismatch      Rule = "hash-mismatch"
	RuleMediaMissing      Rule = "media-missing"
	RuleCategoryShort     Rule = "category-below-minimum"
	RuleCorpusShort       Rule = "corpus-below-minimum"
	RuleHoldoutShort      Rule = "holdout-below-minimum"
	RuleHoldoutSkewed     Rule = "holdout-category-over-share"
	RuleAnnotationMissing Rule = "annotation-missing"
)

// Problem is one rule violation. SceneID is empty for a corpus-wide rule.
type Problem struct {
	Rule    Rule
	SceneID string
	Detail  string
}

// Coverage reports whether this is a shortfall in how much corpus exists rather than something
// wrong with the corpus that does.
//
// The distinction matters because the two need opposite responses. A bad licence, a hash that
// does not match or a missing file is a defect: it must stop a run, because acting on it would
// produce a wrong or an unlawful result. "Only 11 comics so far, the target is 60" is progress
// against a target - it should be visible on every run and it should block nothing, or the loop
// cannot start until the corpus is finished, which is backwards.
func (p Problem) Coverage() bool {
	switch p.Rule {
	case RuleCategoryShort, RuleCorpusShort, RuleHoldoutShort, RuleHoldoutSkewed:
		return true
	default:
		return false
	}
}

// SplitProblems separates the defects from the coverage shortfalls.
func SplitProblems(ps []Problem) (defects, coverage []Problem) {
	for _, p := range ps {
		if p.Coverage() {
			coverage = append(coverage, p)
		} else {
			defects = append(defects, p)
		}
	}
	return defects, coverage
}

func (p Problem) String() string {
	if p.SceneID == "" {
		return fmt.Sprintf("%-28s %s", p.Rule, p.Detail)
	}
	return fmt.Sprintf("%-28s %-40s %s", p.Rule, p.SceneID, p.Detail)
}

// SortProblems orders by rule then scene, so two runs of the same corpus diff cleanly.
func SortProblems(ps []Problem) {
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Rule != ps[j].Rule {
			return ps[i].Rule < ps[j].Rule
		}
		return ps[i].SceneID < ps[j].SceneID
	})
}

// Validate reports every way the manifest and the media on disk disagree with the rules. It
// reports and never repairs: a caller can use it as a gate without wondering what it changed,
// and a hash mismatch stays visible instead of being quietly re-measured into agreement.
//
// root is the corpus root the File paths resolve against. Passing an empty root skips the
// on-disk checks (useful when validating a manifest without the media).
func Validate(m *Manifest, root string) []Problem {
	var ps []Problem
	add := func(r Rule, id, detail string) { ps = append(ps, Problem{Rule: r, SceneID: id, Detail: detail}) }

	seen := make(map[string]bool, len(m.Scenes))
	ids := make(map[string]bool, len(m.Scenes))
	for i := range m.Scenes {
		ids[m.Scenes[i].ID] = true
	}

	for i := range m.Scenes {
		s := &m.Scenes[i]
		if seen[s.ID] {
			add(RuleDuplicateID, s.ID, "declared more than once")
		}
		seen[s.ID] = true

		if !s.Licence.Valid() {
			add(RuleBadLicence, s.ID, fmt.Sprintf("licence %q is not one of the seven accepted values", s.Licence))
		}
		if s.Licence.NeedsAttribution() && s.Attribution == "" {
			add(RuleNoAttribution, s.ID, string(s.Licence)+" requires attribution text")
		}
		if s.LicenceVerifiedBy == "" {
			add(RuleNotVerified, s.ID, "no human has recorded reading the asset's own licence page")
		}
		if s.Licence.NeedsDownload() && s.SourceURL == "" {
			add(RuleNoSource, s.ID, "no sourceUrl, so the scene cannot be re-fetched")
		}
		if s.DerivedFrom != "" && !ids[s.DerivedFrom] {
			add(RuleBadDerivedFrom, s.ID, "derivedFrom points at unknown scene "+s.DerivedFrom)
		}
		if len(s.Categories) == 0 {
			add(RuleNoCategory, s.ID, "no category, so it counts towards no coverage minimum")
		}
		for _, c := range s.Categories {
			if !c.Valid() {
				add(RuleBadCategory, s.ID, fmt.Sprintf("category %q is not in the coverage table", c))
			}
		}
		if !s.Split.Valid() {
			add(RuleBadSplit, s.ID, fmt.Sprintf("split %q is neither dev nor holdout", s.Split))
		}

		if root == "" {
			continue
		}
		path := s.Path(root)
		info, err := os.Stat(path)
		if err != nil {
			add(RuleMediaMissing, s.ID, "no file at "+path)
			continue
		}
		if s.SHA256 == "" {
			add(RuleHashUnmeasured, s.ID, "sha256 is empty, so the bytes on disk prove nothing")
			continue
		}
		got, err := HashFile(path)
		if err != nil {
			add(RuleMediaMissing, s.ID, "cannot read "+path+": "+err.Error())
			continue
		}
		if got != s.SHA256 {
			add(RuleHashMismatch, s.ID, fmt.Sprintf("on disk %s, manifest %s", short(got), short(s.SHA256)))
		}
		if s.Bytes != 0 && s.Bytes != info.Size() {
			add(RuleHashMismatch, s.ID, fmt.Sprintf("size on disk %d, manifest %d", info.Size(), s.Bytes))
		}
	}

	ps = append(ps, validateShape(m)...)
	SortProblems(ps)
	return ps
}

// validateShape checks the corpus-wide coverage rules: the total, the per-category floors, the
// holdout size and the holdout's category balance.
func validateShape(m *Manifest) []Problem {
	var ps []Problem
	add := func(r Rule, detail string) { ps = append(ps, Problem{Rule: r, Detail: detail}) }

	counts := Counts(m)
	if len(m.Scenes) < MinScenes {
		add(RuleCorpusShort, fmt.Sprintf("%s, need %d", plural(len(m.Scenes), "scene"), MinScenes))
	}
	for _, c := range allCategories {
		if got, want := counts.ByCategory[c], categoryMinimum[c]; got < want {
			add(RuleCategoryShort, fmt.Sprintf("%s: %s, need %d", c, plural(got, "scene"), want))
		}
	}

	total := len(m.Scenes)
	if total == 0 {
		return ps
	}
	holdout := counts.BySplit[SplitHoldout]
	if holdout*100 < total*MinHoldoutPercent {
		add(RuleHoldoutShort, fmt.Sprintf("holdout is %d of %s (%.0f%%), need %d%%",
			holdout, plural(total, "scene"), float64(holdout)*100/float64(total), MinHoldoutPercent))
	}
	if holdout > 0 {
		for _, c := range allCategories {
			n := counts.HoldoutByCategory[c]
			if n*100 > holdout*MaxHoldoutCategoryPercent {
				add(RuleHoldoutSkewed, fmt.Sprintf("%s is %d of %d holdout scenes (%.0f%%), max %d%%",
					c, n, holdout, float64(n)*100/float64(holdout), MaxHoldoutCategoryPercent))
			}
		}
	}
	return ps
}

// Count is the coverage tally the verify command prints, so a gap is visible without reading
// the manifest.
type Count struct {
	Total             int
	Scored            int
	ByCategory        map[Category]int
	BySplit           map[Split]int
	HoldoutByCategory map[Category]int
	ByLicence         map[Licence]int
}

// Counts tallies the manifest.
func Counts(m *Manifest) Count {
	c := Count{
		ByCategory:        map[Category]int{},
		BySplit:           map[Split]int{},
		HoldoutByCategory: map[Category]int{},
		ByLicence:         map[Licence]int{},
	}
	for i := range m.Scenes {
		s := &m.Scenes[i]
		c.Total++
		if s.Scored() {
			c.Scored++
		}
		c.BySplit[s.Split]++
		c.ByLicence[s.Licence]++
		for _, cat := range s.Categories {
			c.ByCategory[cat]++
			if s.Split == SplitHoldout {
				c.HoldoutByCategory[cat]++
			}
		}
	}
	return c
}

// plural renders "1 scene" / "2 scenes", because a coverage report is read by people and
// "1 scenes" makes a careful reader distrust the rest of the numbers.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func short(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}
