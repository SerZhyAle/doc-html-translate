# Phase 04 — The gate that fails on an unnamed difference

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ⬜ Not started
**Depends on:** Phase 02, Phase 03
**Steps done:** 0 / 4

## Objective

A blocking test that compares both editions against the canonical source declaration by declaration,
is blind to naming, reads the divergence list rather than hard-coding exceptions, and fails when
either side declares a role outside its derived path.

## Prerequisites

- [ ] Phases 02 and 03 are ✅ Done.
- [ ] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `tests/appearance_parity_test.go` | New | ≤ 320 |
| `tests/parity_test.go` | Modified | ≤ 40 changed |

> A new file rather than growing `tests/parity_test.go`, which is already 690 lines.

## Steps

### Step 04.1 — Declaration comparator

**Files:** `tests/appearance_parity_test.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Write a helper that parses a CSS fragment into `selector -> ordered [property, value]` and one that
> loads `internal/appearance/appearance.json`. Normalize whitespace inside values and lowercase hex
> colours, so `#222` and `#222222` and `Georgia, "Times New Roman"` versus `Georgia,"Times New Roman"`
> compare equal. Comparison keys on the role, never on the selector text.

**Verification:**
- `func parseDeclarations(` and `func loadAppearance(` each match exactly once.
- `go test ./tests/ -run TestAppearance` exits 0 (no assertions yet is acceptable at this step).

**Status:** `[ ]` not done

---

### Step 04.2 — Assert both editions against the source

**Files:** `tests/appearance_parity_test.go`
**Depends on:** Step 04.1

**Prompt for developer:**
> Add `TestAppearanceRolesMatchSource`: build the desktop overlay and palette CSS through the
> `appearance` package, read the generated regions out of `extension/src/ocr-overlay.css` and
> `extension/src/viewer.css`, and assert that each role's declaration set on each side equals the
> source exactly. A property present on one side and absent on the other fails with a message naming
> the property, the role and the side that lacks it. Add `TestAppearanceNoRoleDeclaredOutsideSource`:
> scan both stylesheets outside the generated markers, and the Go sources, for any rule whose
> selector is a role selector, and fail if one exists.

**Verification:**
- `func TestAppearanceRolesMatchSource(` and `func TestAppearanceNoRoleDeclaredOutsideSource(` each
  match exactly once.
- `go test ./tests/ -run TestAppearance` exits 0.

**Status:** `[ ]` not done

---

### Step 04.3 — Prove the gate bites, and triage what it finds

**Files:** `tests/appearance_parity_test.go`, `internal/appearance/appearance.json`
**Depends on:** Step 04.2

**Prompt for developer:**
> Add `TestAppearanceComparatorDetectsDrift`, a unit test feeding the comparator two synthetic
> fragments differing by one declaration - use `box-shadow: 0 0 0 1px rgba(0,0,0,0.06)`, the actual
> defect - and asserting the comparator reports exactly that property as missing on one side. Then
> run the full gate against the real tree and triage every failure it reports that is not already
> expected: either unify the two sides, or add the difference to `divergences` in the source with a
> reason. Record what the first run found in the phase handoff notes.

**Verification:**
- `func TestAppearanceComparatorDetectsDrift(` matches exactly once.
- `box-shadow` appears in `tests/appearance_parity_test.go` and in no stylesheet.
- `go test ./tests/ -run TestAppearance` exits 0.

**Status:** `[ ]` not done

---

### Step 04.4 — Retire the palette-only comparison

**Files:** `tests/parity_test.go`
**Depends on:** Step 04.2

**Prompt for developer:**
> Delete `TestParityThemePalette` - the palette is now held by the general rule, and two comparisons
> of the same values would drift from each other. Leave every other test in the file alone; in
> particular `TestParityOCRFontFit` keeps the runtime font-fit assertions and loses only whatever it
> asserted about padding, radius and the paper carrier, which the new gate now covers as
> declarations.

**Verification:**
- `func TestParityThemePalette(` no longer matches in `tests/parity_test.go`.
- `func TestParityOCRFontFit(` still matches exactly once.
- `go test ./tests/` exits 0.

**Status:** `[ ]` not done

## Phase done criteria

- [ ] Every `Step 04.*` is `[x] done`.
- [ ] `./scripts/test.ps1` exits 0 and `npm test` in `extension/` exits 0.
- [ ] Grep for `TODO(phase-04)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes

Establishes: the rule itself. Record here what Step 04.3's first full run surfaced, split into
"unified" and "named as a divergence" - strategic §6 item 3 is answered by that list.

## Rollback plan

Revert phase commit(s); `TestParityThemePalette` returns with them.
