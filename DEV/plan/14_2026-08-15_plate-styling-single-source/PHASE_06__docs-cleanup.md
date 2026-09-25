# Phase 06 — Docs cleanup

**Strategic spec:** [`../14_2026-08-15_plate-styling-single-source.md`](../14_2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** 🚧 In Progress
**Depends on:** all
**Steps done:** 1 / 2

## Objective

The changelog records every file this ticket touched, and the whole gate runs green end to end.

## Prerequisites

- [ ] Phases 01-05 are ✅ Done.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `DEV/CHANGELOG.md` | Modified | ≤ 40 changed |

> No user-facing docs. Strategic §8 mandates none - the reader sees no change, which is criterion 5.

## Steps

### Step 06.1 — Changelog entries

**Files:** `DEV/CHANGELOG.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Add one row per file created or modified across Phases 01-05, in the table's existing format and
> ordering. The description for the stylesheets says they became generated regions; the description
> for the new test says what it fails on.

**Verification:**
- `internal/appearance/appearance.json`, `tests/appearance_parity_test.go`,
  `extension/scripts/gen-appearance.mjs` each match at least once in `DEV/CHANGELOG.md`.

**Status:** `[x]` done

---

### Step 06.2 — Full gate

**Files:** none - verification step against the finished tree
**Depends on:** Step 06.1

**Prompt for developer:**
> Run `./scripts/test.ps1`, `./scripts/lint.ps1`, `./scripts/typo.ps1`, `./scripts/check.ps1` and
> `npm test` in `extension/`, and record the exit codes in the INDEX change log. Then convert one
> comic page from the corpus with OCR on and compare the rendered result against the 2026-08-15
> output - no ring, no other visible change, which is strategic criterion 5.

**Verification:**
- All five commands exit 0, with their output cited in the INDEX change log.
- The rendered comparison is recorded with the file name of the page used.

**Status:** `[~]` partial - the four PowerShell gates were not run (no PowerShell in the Linux
session); their Go / Node equivalents were, and the corpus comic render needs tesseract and the corpus,
neither present. What ran is in the INDEX change log. Left for the owner's machine: `./scripts/test.ps1`,
`./scripts/lint.ps1`, `./scripts/typo.ps1`, `./scripts/check.ps1`, and one corpus comic page with OCR on
compared against its 2026-08-15 output.

## Phase done criteria

- [ ] Every `Step 06.*` is `[x] done`.
- [ ] See INDEX.md Completion gate.

## Handoff notes

See INDEX.md Completion gate.

## Rollback plan

Revert phase commit(s).
