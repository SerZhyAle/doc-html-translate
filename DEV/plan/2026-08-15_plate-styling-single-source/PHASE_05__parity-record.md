# Phase 05 — Record the rule and mark what it does not cover

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ⛔ Blocked - Step 05.3 only (catalog not reachable from the session that did 05.1-05.2)
**Depends on:** Phase 04
**Steps done:** 2 / 3

## Objective

`docs/PARITY.md` states that the shared appearance is derived from one source, carries the
divergence list, and marks every invariant in the document as gate-enforced or prose-only.

## Prerequisites

- [x] Phase 04 is ✅ Done - the marks must describe what actually runs.
- [x] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `docs/PARITY.md` | Modified | ≤ 130 changed |
| `OCR-PIPELINE.md` (shared contracts catalog) | Modified | ≤ 20 changed |

## Steps

### Step 05.1 — State the single source

**Files:** `docs/PARITY.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Replace the plate-appearance and theme-palette prose with a statement that both are derived from
> `internal/appearance/appearance.json`, that neither edition's copy may be edited by hand, and that
> the divergence list in that file is what the gate treats as legal. Keep the palette value table -
> it is the human-readable form of the source - and mark it as generated-from, not authoritative.

**Verification:**
- `internal/appearance/appearance.json` matches at least twice in `docs/PARITY.md`.
- The palette table still matches, with its four theme rows.

**Status:** `[x]` done

---

### Step 05.2 — Mark every invariant guarded or prose-only

**Files:** `docs/PARITY.md`
**Depends on:** Step 05.1

**Prompt for developer:**
> Give every `###` invariant section in the document an explicit mark stating whether a test enforces
> it and which one, or that only this prose does. Derive the marks by reading the test files, not by
> assumption. The eight sections currently unguarded - format detection by signature, plain-text
> decode order, reader fonts, PDF page-image selection, EPUB TOC rules, settings defaults, product
> URL and feedback address, interface language set - each get a one-line note naming what a future
> ticket would have to pin.

**Verification:**
- Every `### ` heading under "Shared invariants" is followed within 5 lines by a line containing
  either `Guarded by` or `Prose only`.
- At least 8 sections carry `Prose only`.

**Status:** `[x]` done

---

### Step 05.3 — Repoint the OCR pipeline doc

**Files:** `OCR-PIPELINE.md` (the contract lives in the shared catalog, not in this repo - see `docs/contracts/OCR-PIPELINE.md`)
**Depends on:** Step 05.1

**Prompt for developer:**
> Update the plate-rendering rows and §3.1 so they name the canonical source as the place the plate's
> appearance is defined, with the two generators as consumers, instead of pointing at `overlay.go`
> and `ocr-overlay.css` as two parallel definitions. The measurement prose about the paper carrier
> and the padding stays where it is - it explains a decision, not a value.

**Verification:**
- `internal/appearance` matches at least once in the catalog's `OCR-PIPELINE.md`.
- The §3.1 paragraph on the opaque paper carrier is unchanged in wording.

**Status:** ⛔ blocked - the catalog lives at the path named in `AGENTS.md`, a Windows drive that is
not mounted in the Linux session this ran in. The in-repo pointer `docs/contracts/APP-STYLE.md` was
updated (it named `TestParityThemePalette`); `docs/contracts/OCR-PIPELINE.md` names no file this ticket
moved. To do on the owner's machine: in the catalog's `ocr-overlay/ocr-pipeline.md`, repoint the
plate-rendering rows and §3.1 at `internal/appearance/appearance.json` as the single definition, with
`internal/appearance` (Go) and `extension/scripts/gen-appearance.mjs` (JS) as its consumers.

## Phase done criteria

- [ ] Every `Step 05.*` is `[x] done` - 05.3 blocked.
- [ ] `./scripts/typo.ps1` exits 0 - not run (no PowerShell); `tests/typography_test.go` green.
- [x] Grep for `TODO(phase-05)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Establishes: the document now distinguishes an invariant that is enforced from one that is merely
written down - the distinction this whole ticket came from.

## Rollback plan

Revert phase commit(s).
