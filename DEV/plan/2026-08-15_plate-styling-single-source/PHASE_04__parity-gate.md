# Phase 04 — The gate that fails on an unnamed difference

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02, Phase 03
**Steps done:** 4 / 4

## Objective

A blocking test that compares both editions against the canonical source declaration by declaration,
is blind to naming, reads the divergence list rather than hard-coding exceptions, and fails when
either side declares a role outside its derived path.

## Prerequisites

- [x] Phases 02 and 03 are ✅ Done.
- [x] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `tests/appearance_parity_test.go` | New | ≤ 320 |
| `tests/parity_test.go` | Modified | ≤ 125 changed |

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

**Status:** `[x]` done

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

**Status:** `[x]` done

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
- `box-shadow` appears in `tests/appearance_parity_test.go` and in no plate role: no `box-shadow`
  entry in `roles.plate` of `appearance.json`, and no `box-shadow` declaration in a generated region.
- `go test ./tests/ -run TestAppearance` exits 0.

**Status:** `[x]` done

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

**Status:** `[x]` done

## Phase done criteria

- [x] Every `Step 04.*` is `[x] done`.
- [x] `go test ./tests/` exits 0 and `npm test` in `extension/` exits 0. `./scripts/test.ps1` itself
      was not run - no PowerShell in the Linux session; `go test ./...` is green apart from
      `internal/pdf` `TestPdfTitle`, which fails identically on the base commit (a Windows path).
- [x] Grep for `TODO(phase-04)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Establishes: the rule itself. What Step 04.3's first full run surfaced (strategic §6 item 3):

- **Unified:** the image role's reset guard (`margin:0; max-height:none`) - on the role on both sides
  now (done in Phase 01 / 03, so the first gate run was already green on it).
- **Named as a divergence:** selector names, the OCR toggle class, the palette prefix and theme
  attribute (naming - invisible to the comparator by construction), and the viewer's
  `#content .ocr-overlay { margin: 1em auto; }` - placement of the unit in the reading column, not a
  declaration of the role, so the outside-the-source scan (exact role selectors only) does not read it.
- Nothing else: container, image and plate were declaration-identical once the image guard moved.

Demonstrated on the real tree (then reverted): the ring re-added inside the extension's generated
region fails `TestAppearanceRolesMatchSource` with `role plate: box-shadow: missing on source,
desktop`; added as a hand rule after the region it fails `TestAppearanceNoRoleDeclaredOutsideSource`;
renaming the plate selector and toggle class in the generator only, then regenerating, stays green;
a dark-theme accent changed in `viewer.css` fails with `theme dark: accent: values differ`; renaming
the extension's palette prefix stays green.

`TestParityOCRPrintPlate` and the paper-carrier check in `TestParityOCRFontFit` were rewritten to
read the source rather than two CSS literals - the literal on the desktop side no longer exists.

## Rollback plan

Revert phase commit(s); `TestParityThemePalette` returns with them.
