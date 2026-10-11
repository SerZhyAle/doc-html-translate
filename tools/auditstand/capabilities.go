// capabilities.go implements the capability inventory and the self-test's two-direction check
// (strategic section 5.3): the pinned inventory of strategic section 5.3 must cover every
// capability CLAUDE.md and AGENTS.md announce, and every pin must still exist in the file it
// cites - a vanished phrase is a stale pin, a new capability phrase with no covering scene is a
// coverage gap.

package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// pin is a literal phrase that must exist in a named repo file. The file path is repo-relative.
type pin struct {
	file   string
	phrase string
}

// capabilityRow is one row of the pinned inventory (strategic section 5.3), with its covering
// scenes and the claims that tie it to the derived capability list.
type capabilityRow struct {
	id     string
	text   string   // the capability as the spec's section 5.3 table states it
	scenes []string // covering scene ids from that table
	pins   []pin    // phrases that must exist now; a row may carry none (recorded as informational)
	covers []string // derived capability bullets this row claims
}

// The derivation sources the spec names plus the contracts pointer roster (the strategic
// spec's section 13 cites it as the pointer roster; cap-update's phrase lives only there).
const (
	claudemd        = "CLAUDE.md"
	agentsmd        = "AGENTS.md"
	contractsRoster = "docs/contracts/README.md"
)

// capabilityInventory is the 17-row pinned inventory of strategic section 5.3.
var capabilityInventory = []capabilityRow{
	{
		id:     "cap-formats",
		text:   "Convert EPUB / PDF / MOBI / AZW3 / FB2 / RTF / TXT / Markdown / HTML",
		scenes: []string{"S04", "S15"},
		pins:   []pin{{claudemd, "converts EPUB / PDF / MOBI"}},
		covers: []string{"Pipeline orchestration", "Format extractors", "internal/pipeline",
			"internal/epub", "internal/pdf", "internal/mobi", "internal/fb2", "internal/rtf",
			"internal/txt", "internal/md", "internal/htmlconv"},
	},
	{
		id:     "cap-comics",
		text:   "Comic archives CBZ/CBR/CB7/CBT, one page per spine entry, forced OCR",
		scenes: []string{"S06", "S15"},
		pins:   []pin{{agentsmd, "comic archives (CBZ/CBR/CB7/CBT)"}},
		covers: []string{"comic"},
	},
	{
		id:     "cap-image",
		text:   "Standalone image input wrapped in a one-page doc, forced OCR",
		scenes: []string{"S05"},
		pins:   []pin{{agentsmd, "standalone image input"}},
		covers: []string{"internal/img"},
	},
	{
		id:     "cap-ocr",
		text:   "Optional OCR overlay, tessdata download on demand, best-effort",
		scenes: []string{"S05", "S16"},
		pins:   []pin{{agentsmd, "Optional OCR overlay"}},
		covers: []string{"internal/ocr"},
	},
	{
		id:     "cap-translation",
		text:   "Optional translation, Google Cloud or local Ollama, cost guard",
		scenes: []string{"S08"},
		pins:   []pin{{claudemd, "Google Cloud or local Ollama"}},
		covers: []string{"Translation", "internal/translator"},
	},
	{
		id:     "cap-cli",
		text:   "CLI binary; no-arg run enters the registration flow",
		scenes: []string{"S02"},
		pins:   []pin{{claudemd, "No-arg CLI enters the"}},
		covers: []string{"cmd/doc-html-translate", "Entry points", "No-arg CLI enters the"},
	},
	{
		id:     "cap-gui",
		text:   "GUI binary: queue, dialogs, readiness, stages, final status",
		scenes: []string{"S01", "S09", "S10"},
		pins:   []pin{{claudemd, "doc-html-ui"}, {agentsmd, "doc-html-ui (GUI wrapper for CLI)"}},
		covers: []string{"cmd/doc-html-ui", "App wiring", "internal/app", "internal/config"},
	},
	{
		id:     "cap-registration",
		text:   "Windows integration ask-first, register/unregister, user choice wins",
		scenes: []string{"S11"},
		pins:   []pin{{agentsmd, "internal/windowsreg"}},
		covers: nil, // windowsreg is named outside the scoped derivation sources; the pin guards it
	},
	{
		id:     "cap-reuse",
		text:   "Output reuse via the completion record; -force rebuilds",
		scenes: []string{"S03"},
		pins:   []pin{{claudemd, "completion record says it finished"}},
		covers: []string{"Existing output is reused"},
	},
	{
		id:     "cap-report",
		text:   "Send-logs report, clipboard path, nothing automatic",
		scenes: []string{"S18"},
		pins:   []pin{{agentsmd, "DIAGNOSTIC-REPORT"}},
		covers: nil, // internal/report is named by no scoped derivation source
	},
	{
		id:     "cap-i18n",
		text:   "13 interface languages, resolve order, document language preserved",
		scenes: []string{"S01", "S04"},
		pins:   []pin{{agentsmd, "Interface language (13 languages"}},
		covers: []string{"Interface language", "internal/i18n"},
	},
	{
		id:     "cap-fdsec",
		text:   "FileDO .fd-sec secret files, password once per queue, private copy removed",
		scenes: []string{"S07"},
		pins:   []pin{{agentsmd, "FileDO secret files"}},
		covers: []string{"internal/fdsec"},
	},
	{
		id:     "cap-procrun",
		text:   "External helpers through the one runner: deadlines, tree kill, one error type",
		scenes: []string{"S06", "S10", "S16"},
		pins:   []pin{{agentsmd, "one way to run a helper"}},
		covers: []string{"External helpers", "internal/procrun"},
	},
	{
		id:     "cap-freeflow",
		text:   "The free flow: open in browser, browser page translation",
		scenes: []string{"S04"},
		pins:   []pin{{claudemd, "built-in page translation"}},
		covers: nil,
	},
	{
		id:     "cap-update",
		text:   "Update check off by default, once a day, public info only",
		scenes: []string{"S17"},
		pins:   []pin{{contractsRoster, "UPDATE-MANIFEST"}},
		covers: nil,
	},
	{
		id:     "cap-reader",
		text:   "Reader themes and position remembered across sessions",
		scenes: []string{"S04", "S12"},
		pins:   []pin{{agentsmd, "reading themes and reading position use localStorage"}},
		covers: []string{"HTML processing/generation", "internal/htmlproc", "internal/htmlsplit",
			"internal/htmlgen", "internal/assets"},
	},
	{
		id:     "cap-bundled",
		text:   "Bundled pdftotext unpacked into a content-hash cache, nothing installed",
		scenes: []string{"S04", "S15"},
		pins:   []pin{{agentsmd, "internal/bundledtools"}},
		covers: []string{"internal/bundledtools"},
	},
}

// outOfScopeBullets names derived Architecture Map bullets that are deliberately not capabilities
// of the audited (shipped) program, each with the reason. The self-test reports them as matched.
var outOfScopeBullets = map[string]string{
	"Developer tooling, not shipped":         "the audit measures the shipped program only; its own instrument rides this exclusion too",
	"OCR visual-fidelity lab":                "the ocrlab precedent is developer tooling (same exclusion)",
	"Don't change public CLI flag semantics": "an editing constraint, not a runtime capability - strategic section 2 excludes a static re-read",
	"tools/ocrlab":                           "the ocrlab precedent is developer tooling (same exclusion)",
}

// deriveBullets extracts the capability list exactly where the spec names it: AGENTS.md's
// "Architecture Map" bullets (top-level and their internal/ sub-bullets), CLAUDE.md's
// "High-value invariants" bullets, and the cmd/ + internal/ tokens inside those scoped regions
// plus CLAUDE.md's "What this is" paragraph. Everything outside those regions is dev flow and
// documentation, not capability of the shipped program.
func deriveBullets(claude, agents string) []string {
	archMap := sectionBetween(agents, "## Architecture Map", "\n## ")
	invariants := sectionBetween(claude, "## High-value invariants", "\n## ")
	whatIs := sectionBetween(claude, "## What this is", "\n## ")

	var bullets []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		bullets = append(bullets, s)
	}
	// Bullet headlines: "- internal/comic: comic archives ..." -> "internal/comic".
	for _, line := range strings.Split(archMap+"\n"+invariants, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "- ") {
			continue
		}
		t = strings.TrimPrefix(t, "- ")
		if i := strings.Index(t, ":"); i > 0 && i < 120 {
			t = t[:i]
		}
		add(t)
	}
	// Code-surface tokens in the scoped regions: the shipped body's packages and binaries.
	for _, m := range regexp.MustCompile(`cmd/doc-html-[a-z-]+`).FindAllString(archMap+"\n"+invariants+"\n"+whatIs, -1) {
		add(m)
	}
	for _, m := range regexp.MustCompile(`internal/[a-z]+`).FindAllString(archMap+"\n"+invariants, -1) {
		add(m)
	}
	sort.Strings(bullets)
	return bullets
}

// sectionBetween returns one markdown section's body (header line excluded), or "" when absent.
func sectionBetween(text, header, nextHeader string) string {
	i := strings.Index(text, header)
	if i < 0 {
		return ""
	}
	rest := text[i+len(header):]
	if j := strings.Index(rest, nextHeader); j >= 0 {
		return rest[:j]
	}
	return rest
}

// outOfScopeMatches collects the by-design matches of the last checkCapabilities run.
var outOfScopeMatches []string

// checkCapabilities runs both directions and returns defects (stale pins, claims of vanished
// bullets, derived capabilities no row claims) plus informational notes (rows without pins).
func checkCapabilities(claude, agents, roster string) (defects []error, notes []string) {
	text := map[string]string{claudemd: claude, agentsmd: agents, contractsRoster: roster}
	outOfScopeMatches = nil
	bullets := deriveBullets(claude, agents)

	// Direction 2: every pin must exist in the file it cites (a vanished phrase = stale pin),
	// every covering scene must exist, every claimed derived bullet must still be derived.
	for _, row := range capabilityInventory {
		for _, p := range row.pins {
			if !strings.Contains(text[p.file], p.phrase) {
				defects = append(defects, fmt.Errorf("stale pin: %q no longer in %s (row %s)", p.phrase, p.file, row.id))
			}
		}
		if len(row.pins) == 0 {
			notes = append(notes, fmt.Sprintf("row %s carries no pin (its phrase is named by no derivation source); coverage rests on its scenes %s",
				row.id, strings.Join(row.scenes, ", ")))
		}
		for _, s := range row.scenes {
			if _, ok := sceneByID(sceneID(s)); !ok {
				defects = append(defects, fmt.Errorf("row %s names unknown scene %s", row.id, s))
			}
		}
		for _, c := range row.covers {
			if !derivedHas(bullets, c) {
				defects = append(defects, fmt.Errorf("row %s claims derived capability %q, which the derivation no longer finds", row.id, c))
			}
		}
	}

	// Direction 1: every derived capability must be claimed by a row or named out of scope.
	claimed := map[string]bool{}
	for _, row := range capabilityInventory {
		for _, c := range row.covers {
			claimed[c] = true
		}
	}
	for _, b := range bullets {
		if derivedMatch(b, claimed) {
			continue
		}
		if reason, ok := outOfScopeFor(b); ok {
			// Matched by design: the spec's own derivation excludes these. Recorded for the
			// -Plan print, never verdict-affecting.
			outOfScopeMatches = append(outOfScopeMatches, fmt.Sprintf("%q is out of scope: %s", b, reason))
			continue
		}
		defects = append(defects, fmt.Errorf("derived capability %q has no covering inventory row", b))
	}
	return defects, notes
}

// derivedMatch reports whether the bullet is covered by any claim (prefix containment in either
// direction handles "App wiring" vs the bullet's longer text).
func derivedMatch(bullet string, claimed map[string]bool) bool {
	for c := range claimed {
		if matchClaim(bullet, c) {
			return true
		}
	}
	return false
}

// matchClaim is the one matching rule: a claim matches a bullet when either is a prefix of the
// other after trimming, or when the bullet mentions the claim as a token.
func matchClaim(bullet, claim string) bool {
	bullet = strings.TrimSpace(bullet)
	claim = strings.TrimSpace(claim)
	if bullet == "" || claim == "" {
		return false
	}
	if strings.HasPrefix(bullet, claim) || strings.HasPrefix(claim, bullet) {
		return true
	}
	return strings.Contains(bullet, claim)
}

func derivedHas(bullets []string, claim string) bool {
	for _, b := range bullets {
		if matchClaim(b, claim) {
			return true
		}
	}
	return false
}

func outOfScopeFor(b string) (string, bool) {
	for prefix, reason := range outOfScopeBullets {
		if matchClaim(b, prefix) {
			return reason, true
		}
	}
	return "", false
}

// loadCapabilitySources reads the derivation sources. A missing source is a could-not-verify
// condition for -SelfTest, never a pass.
func loadCapabilitySources() (claude, agents, roster string, err error) {
	b1, err := os.ReadFile(claudemd)
	if err != nil {
		return "", "", "", fmt.Errorf("read %s: %w", claudemd, err)
	}
	b2, err := os.ReadFile(agentsmd)
	if err != nil {
		return "", "", "", fmt.Errorf("read %s: %w", agentsmd, err)
	}
	b3, err := os.ReadFile(contractsRoster)
	if err != nil {
		return "", "", "", fmt.Errorf("read %s: %w", contractsRoster, err)
	}
	return string(b1), string(b2), string(b3), nil
}
