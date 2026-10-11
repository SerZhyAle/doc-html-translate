// Package auditstand - command auditstand is the deep dynamic audit's measuring stand: the
// instrument the strategic spec DEV/plan/113_2026-10-11_dynamic-behaviour-audit.md requires
// before any dynamic property of the running desktop edition is accepted (responsiveness,
// wrapped errors, no system blocking, leaks, understandability).
//
// It is a developer tool. No build script compiles it, it ships in no package, and it is never
// linked into doc-html-translate or doc-html-ui - it starts those from outside and measures
// what they do to the machine. The instrument itself changes no shipped code (strategic
// section 2: no app changes for measurability).
package main

// matrix.go holds the scene catalog, the coverage matrix and the budget hypotheses of the
// strategic spec's sections 5 and 8, as data. The self-test re-derives what it can and fails on
// internal contradictions, so this file is the single source the runner and the summary share.

import (
	"fmt"
	"strings"
)

// sceneID is a stable scene identifier from the strategic spec's section 5.1 table.
type sceneID string

// The five audited properties, numbered as the spec numbers them.
const (
	propResponsiveness   = 1
	propWrappedErrors    = 2
	propNoSystemBlocking = 3
	propLeaks            = 4
	propUnderstand       = 5
)

// cellState is the honest state of one matrix cell (strategic section 5.2): open until a run
// record closes it, n/a only through a register row.
type cellState string

const (
	cellOpen     cellState = "open"
	cellMeasured cellState = "measured"
	cellNA       cellState = "n/a"
)

// scene describes one row of the scene catalog.
type scene struct {
	id    sceneID
	title string
	// touchesRegistry marks scenes whose steps can write HKCU. The runner refuses them (exit 2)
	// until the Level-B named-key backup of strategic section 7 is configured for the run.
	touchesRegistry bool
	// phase is the tactical phase that owns the scene's steps (DEV/plan/113_.../INDEX.md).
	phase int
}

// scenes is the catalog of strategic section 5.1, in table order.
var scenes = []scene{
	{"S01", "first-start-gui", true, 2},
	{"S02", "first-start-cli", true, 2},
	{"S03", "reopen-reuse", false, 2},
	{"S04", "convert-mainstream", false, 2},
	{"S05", "convert-scan-ocr", false, 2},
	{"S06", "convert-comic", false, 2},
	{"S07", "convert-fdsec", false, 1},
	{"S08", "translate-guards", false, 2},
	{"S09", "queue-batch", false, 2},
	{"S10", "exit-under-load", false, 1},
	{"S11", "registration-toggle", true, 3},
	{"S12", "sleep-lock", false, 3},
	{"S13", "rdp-session", false, 3},
	{"S14", "storm-inputs", true, 2},
	{"S15", "big-inputs", false, 1},
	{"S16", "failure-zoo", false, 1},
	{"S17", "idle-soak", false, 3},
	{"S18", "report-build", false, 2},
}

// budget is one hypothesis of strategic section 8. A hypothesis becomes a budget only after the
// Phase 0 control run and the owner's sign-off; until then every evaluation is advisory except
// the one hard bar (H1's watchdog tick, a shipped constant).
type budget struct {
	id       string
	property int
	limit    string
	hardBar  bool // a shipped constant, not a hypothesis: a host ping above one watchdog tick is P1 by definition
}

// budgets holds H1-H14 verbatim enough to print and to evaluate against recorded measurements.
var budgets = []budget{
	{"H1", propResponsiveness, "GUI host ping p95 <= 50 ms idle, <= 500 ms under conversion; hard bar: never above one watchdog tick (5 s)", true},
	{"H2", propResponsiveness, "browser window ping p95 <= 50 ms, <= 100 ms under the S14 storm", false},
	{"H3", propResponsiveness, "injected key to visible reaction p95 <= 250 ms idle, <= 700 ms under load", false},
	{"H4", propResponsiveness, "Cancel/Ctrl+C/close to a fully stopped process tree <= 2 s", false},
	{"H5", propResponsiveness, "cold start: GUI usable <= 3 s warm, CLI first output <= 1.5 s; reopen-reuse indistinguishable from the host ping round trip", false},
	{"H6", propResponsiveness, "file/folder dialog open to usable <= 1.5 s, focus restored", false},
	{"H7", propResponsiveness, "report build <= 5 s, clipboard held <= 100 ms", false},
	{"H8", propWrappedErrors, "100% of injected failures end in a named outcome, one boundary log line, zero canary hits, a message in the active locale, consistent state", false},
	{"H9", propNoSystemBlocking, "foreign clipboard and focus latency within 2x control p95; only the app's own scenes change the foreground window", false},
	{"H10", propNoSystemBlocking, "foreign keyboard, window-message, disk and network probes within 2x control p95", false},
	{"H11", propNoSystemBlocking, "association changes: no broadcast storm; foreign message latency within 2x control p95", false},
	{"H12", propNoSystemBlocking, "no focus trap outside an open dialog; every dialog closes and returns focus", false},
	{"H13", propLeaks, "resource slopes' 95% CI includes zero after documented caches; registry outside the named set unchanged; logs within MaxLogBytes; temp residue zero; idle wakeups within 2x control", false},
	{"H14", propUnderstand, "100% of GUI toggles observable, revert byte-identical, 13/13 locales, announced; keyboard-only flows without a focus trap", false},
}

// naRow is one row of the n/a register (strategic section 5.2): the reason a cell (or a named
// sub-probe of a cell) is not measured.
type naRow struct {
	id     string
	cell   string // "S13x5" or, for sub-probe rows, the narrowed form the register uses
	reason string
}

// naRegister holds N1-N6 verbatim.
var naRegister = []naRow{
	{"N1", "S02/S03x3", "the CLI conversion path has no hooks/clipboard/broadcast mechanism (statically verified, register 2026-09-24 areas G/P); the cell scores the sub-probes that do apply: disk, network, locks"},
	{"N2", "S08x2", "the audit never spends money; the fake-key and cost-guard refusal paths are the measurable surface; the paid success path is n/a for the whole campaign"},
	{"N3", "S13x5", "an RDP reconnect carries no interface decision to understand or revert; S12 and S11 cover the language and revert cells"},
	{"N4", "S14x5", "a storm deliberately loads the UI past readability; scoring understandability under a storm would find the storm, not the product"},
	{"N5", "S15x5", "big inputs drive no interface decision; the refusal message readability is scored under property 2 of the same scene and property 5 of S16"},
	{"N6", "S17x5", "an unattended idle has no user present; the update-check understanding cells are covered by S11/S18 property 5 probes"},
}

// matrixCell is one scene x property cell with its budget references, as strategic section 5.2
// pins them at Draft time.
type matrixCell struct {
	scene   sceneID
	prop    int
	state   cellState
	budgets []string
	na      string // register row id for n/a cells or sub-probe narrowing
}

// matrix is the 18x5 table of strategic section 5.2. Every applicable cell is open at Draft
// time - the honest state until a run record closes it.
var matrix = buildMatrix()

func buildMatrix() []matrixCell {
	type row struct {
		id     sceneID
		byProp [5]string // budget refs per property, "" prefixed with na row id when n/a
	}
	rows := []row{
		{"S01", [5]string{"H3,H5", "H8", "H9,H10,H11,H12", "H13", "H14"}},
		{"S02", [5]string{"H4,H5", "H8", "H9,H10 " + naRegister[0].id, "H13", "H14"}},
		{"S03", [5]string{"H5", "H8", "H9,H10 " + naRegister[0].id, "H13", "H14"}},
		{"S04", [5]string{"H1,H2,H3", "H8", "H9,H10,H11,H12", "H13", "H14"}},
		{"S05", [5]string{"H1,H2,H3", "H8", "H9,H10,H11", "H13", "H14"}},
		{"S06", [5]string{"H1,H2,H3", "H8", "H9,H10", "H13", "H14"}},
		{"S07", [5]string{"H3,H6", "H8", "H9,H12", "H13", "H14"}},
		{"S08", [5]string{"H1,H2,H3", "H8 " + naRegister[1].id, "H9,H10", "H13", "H14"}},
		{"S09", [5]string{"H1,H2,H3", "H8", "H9,H10", "H13", "H14"}},
		{"S10", [5]string{"H4", "H8", "H10", "H13", "H14"}},
		{"S11", [5]string{"H3", "H8", "H9,H10,H11,H12", "H13", "H14"}},
		{"S12", [5]string{"H1,H2,H3", "H8", "H9,H10", "H13", "H14"}},
		{"S13", [5]string{"H1,H2,H3", "H8", "H9,H12", "H13", naRegister[2].id}},
		{"S14", [5]string{"H1,H2,H3", "H8", "H9,H10,H11,H12", "H13", naRegister[3].id}},
		{"S15", [5]string{"H1,H2,H3", "H8", "H9,H10", "H13", naRegister[4].id}},
		{"S16", [5]string{"H1,H2,H3", "H8", "H9,H10", "H13", "H14"}},
		{"S17", [5]string{"H1,H3", "H8", "H9,H10", "H13", naRegister[5].id}},
		{"S18", [5]string{"H7", "H8", "H9", "H13", "H14"}},
	}
	var cells []matrixCell
	for _, r := range rows {
		for p := 1; p <= 5; p++ {
			ref := r.byProp[p-1]
			c := matrixCell{scene: r.id, prop: p, state: cellOpen}
			if na, ok := cutNA(ref); ok {
				if exactNA(ref) {
					c.state = cellNA
					c.na = na
					c.budgets = nil
				} else {
					// A sub-probe narrowing (N1, N2): the cell stays open on the remaining
					// budgets and records the register row it narrows itself with.
					c.na = na
					c.budgets = budgetRefs(stripNA(ref))
				}
			} else {
				c.budgets = budgetRefs(ref)
			}
			cells = append(cells, c)
		}
	}
	return cells
}

// cutNA splits a trailing " N#" register reference off a budget-reference list.
func cutNA(ref string) (string, bool) {
	for _, n := range naRegister {
		for i := 0; i+len(n.id) <= len(ref); i++ {
			if ref[i:i+len(n.id)] == n.id {
				return n.id, true
			}
		}
	}
	return "", false
}

// exactNA reports whether the whole reference is the register row (the cell is n/a) rather than
// a budget list narrowed by one (the cell stays open on its remaining budgets).
func exactNA(ref string) bool {
	for _, n := range naRegister {
		if ref == n.id {
			return true
		}
	}
	return false
}

// budgetRefs splits "H1,H2,H3" into ids. The caller has already stripped any register row.
func budgetRefs(ref string) []string {
	var out []string
	for _, part := range splitCompact(ref, ',') {
		if len(part) >= 2 && part[0] == 'H' {
			out = append(out, part)
		}
	}
	return out
}

// stripNA removes the trailing register-row reference (" N1") from a cell's reference list.
func stripNA(ref string) string {
	for _, n := range naRegister {
		for i := 0; i+len(n.id) <= len(ref); i++ {
			if ref[i:i+len(n.id)] == n.id {
				return ref[:i] + ref[i+len(n.id):]
			}
		}
	}
	return ref
}

// splitCompact splits on sep and drops empty and space-only parts.
func splitCompact(s string, sep byte) []string {
	var out []string
	for _, part := range strings.Split(s, string(sep)) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// cellFor returns the matrix cell for a scene x property pair.
func cellFor(id sceneID, prop int) (matrixCell, bool) {
	for _, c := range matrix {
		if c.scene == id && c.prop == prop {
			return c, true
		}
	}
	return matrixCell{}, false
}

// validateMatrix checks the internal consistency the self-test demands (strategic section 5.4):
// every cell has a state and, when open, budgets; every budget id resolves; every n/a cell and
// every sub-probe narrowing names a register row; every register row is referenced; every scene
// of the catalog appears in the matrix and vice versa.
func validateMatrix() []error {
	var errs []error
	byID := map[sceneID]scene{}
	for _, s := range scenes {
		byID[s.id] = s
	}
	seen := map[sceneID]bool{}
	for _, c := range matrix {
		if _, ok := byID[c.scene]; !ok {
			errs = append(errs, fmt.Errorf("matrix cell names unknown scene %s", c.scene))
			continue
		}
		seen[c.scene] = true
		switch c.state {
		case cellOpen, cellMeasured:
			if c.state == cellOpen && len(c.budgets) == 0 {
				errs = append(errs, fmt.Errorf("open cell %sx%d has no budget reference", c.scene, c.prop))
			}
			for _, b := range c.budgets {
				if !budgetExists(b) {
					errs = append(errs, fmt.Errorf("cell %sx%d names unknown budget %s", c.scene, c.prop, b))
				}
			}
			if c.na != "" && !naExists(c.na) {
				errs = append(errs, fmt.Errorf("cell %sx%d names unknown n/a row %s", c.scene, c.prop, c.na))
			}
		case cellNA:
			if c.na == "" || !naExists(c.na) {
				errs = append(errs, fmt.Errorf("n/a cell %sx%d has no register row", c.scene, c.prop))
			}
			if len(c.budgets) != 0 {
				errs = append(errs, fmt.Errorf("n/a cell %sx%d still names budgets %v", c.scene, c.prop, c.budgets))
			}
		default:
			errs = append(errs, fmt.Errorf("cell %sx%d has unknown state %q", c.scene, c.prop, c.state))
		}
	}
	for _, s := range scenes {
		if !seen[s.id] {
			errs = append(errs, fmt.Errorf("scene %s has no matrix cells", s.id))
		}
		if len(matrixCellsFor(s.id)) != 5 {
			errs = append(errs, fmt.Errorf("scene %s does not have exactly 5 cells", s.id))
		}
	}
	referenced := map[string]bool{}
	for _, c := range matrix {
		if c.na != "" {
			referenced[c.na] = true
		}
	}
	for _, n := range naRegister {
		if !referenced[n.id] {
			errs = append(errs, fmt.Errorf("n/a register row %s is referenced by no cell", n.id))
		}
		if n.reason == "" {
			errs = append(errs, fmt.Errorf("n/a register row %s has no reason", n.id))
		}
	}
	if len(matrix) != len(scenes)*5 {
		errs = append(errs, fmt.Errorf("matrix holds %d cells, want %d", len(matrix), len(scenes)*5))
	}
	return errs
}

func budgetExists(id string) bool {
	for _, b := range budgets {
		if b.id == id {
			return true
		}
	}
	return false
}

func naExists(id string) bool {
	for _, n := range naRegister {
		if n.id == id {
			return true
		}
	}
	return false
}

func matrixCellsFor(id sceneID) []matrixCell {
	var out []matrixCell
	for _, c := range matrix {
		if c.scene == id {
			out = append(out, c)
		}
	}
	return out
}

func sceneByID(id sceneID) (scene, bool) {
	for _, s := range scenes {
		if s.id == id {
			return s, true
		}
	}
	return scene{}, false
}
