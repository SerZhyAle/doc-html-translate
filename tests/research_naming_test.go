package tests

// Research notes carry the RESEARCH_ type prefix (REPO-LAYOUT rule 3; the naming is declared in
// CLAUDE.md, "Research notes"). The notes written before the rule keep their names, because tickets
// and the frozen DEV/plan/done/ archive link to them; this list is that set, and it only shrinks - an
// entry whose note is gone fails too, so the list never describes a folder that no longer exists.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// namedBeforeTheRule are the entries of DEV/research/ that predate the RESEARCH_ prefix
// (2026-09-25). Never add to it: a new note takes the prefix.
var namedBeforeTheRule = map[string]bool{
	"audit_2026-09-24":                       true,
	"competitor_feature_research_ru.md":      true,
	"epub2html_research_ru.md":               true,
	"extension_formats_feasibility_ru.md":    true,
	"format_verification_sweep.md":           true,
	"ocr_balloon_boundary_2026-09-25.md":     true,
	"ocr_display_lettering_2026-08-12.md":    true,
	"ocr_grey_rescue_2026-08-11.md":          true,
	"ocr_halftone_2026-08-12.md":             true,
	"ocr_plate_coverage_2026-08-13.md":       true,
	"ocr_positioning_exchange_2026-08-12.md": true,
	"ocr_rescue_floor_2026-08-15.md":         true,
	"ocr_rescue_third_axis_2026-09-25.md":    true,
	"ocr_sweep_2026-08-13.md":                true,
	"ocr_word_gap_2026-09-12.md":             true,
	"ocrlab":                                 true,
	"page_ocr_placement_2026-09-25":          true,
	"pdf_translate_extension_spec.md":        true,
}

// methodReferences sit beside the notes but are the method the agent rules link to, not findings.
var methodReferences = map[string]bool{
	"CODE_QUALITY.md":   true,
	"VALIDATION.md":     true,
	"RESEARCH_INDEX.md": true,
}

// A note is RESEARCH_<topic-slug>_<YYYY-MM-DD>.md, or a folder of that name.
var researchNoteName = regexp.MustCompile(`^RESEARCH_[A-Za-z0-9][A-Za-z0-9_-]*_\d{4}-\d{2}-\d{2}(\.md)?$`)

func TestResearchNoteNaming(t *testing.T) {
	dir := filepath.Join("..", "DEV", "research")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		present[name] = true
		switch {
		case namedBeforeTheRule[name], methodReferences[name]:
		case researchNoteName.MatchString(name):
		default:
			t.Errorf("DEV/research/%s: a new research note is named RESEARCH_<topic-slug>_<YYYY-MM-DD>.md (CLAUDE.md, \"Research notes\")", name)
		}
	}
	var gone []string
	for name := range namedBeforeTheRule {
		if !present[name] {
			gone = append(gone, name)
		}
	}
	for name := range methodReferences {
		if !present[name] {
			gone = append(gone, name)
		}
	}
	sort.Strings(gone)
	for _, name := range gone {
		t.Errorf("DEV/research/%s is listed in this test but no longer exists - drop it from the list", name)
	}
}
