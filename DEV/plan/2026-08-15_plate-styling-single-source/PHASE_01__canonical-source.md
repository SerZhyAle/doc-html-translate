# Phase 01 — Canonical appearance source

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ⬜ Not started
**Depends on:** none - foundation phase
**Steps done:** 0 / 3

## Objective

One machine-readable description of the shared appearance - the OCR overlay's three roles and the
four reader themes - that both editions will derive from, plus the named-divergence list the gate
reads.

## Prerequisites

- [ ] Strategic §6 items blocking this phase are Resolved - item 1 is.
- [ ] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/appearance/appearance.json` | New | ≤ 220 |
| `internal/appearance/README.md` | New | ≤ 60 |

> The file lives under `internal/` to follow the repo's Go layout; it is not desktop-owned. The
> extension reads the same path at generation time (Phase 03), never at run time.

## Steps

### Step 01.1 — Define the source schema

**Files:** `internal/appearance/appearance.json`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `internal/appearance/appearance.json` with four top-level keys: `roles`, `themes`,
> `divergences`, `notes`. `roles` maps a role name (`container`, `image`, `plate`) to an ordered
> array of `{ "property": ..., "value": ..., "note": ... }` objects, where `note` is optional and
> carries the measurement or reason behind a load-bearing value. `themes` maps a theme name
> (`light`, `sepia`, `dark`, `night`) to the eight colour tokens `bg`, `fg`, `muted`, `barBg`,
> `barFg`, `border`, `accent`, `link`. `divergences` is an array of
> `{ "what": ..., "reason": ... }`. Do not put selector names, custom-property prefixes or toggle
> class names anywhere in this file - naming is per-edition.

**Verification:**
- File `internal/appearance/appearance.json` exists and parses as JSON.
- Top-level keys are exactly `roles`, `themes`, `divergences`, `notes`.
- `roles` has exactly the keys `container`, `image`, `plate`.
- No value anywhere in the file matches `ocr-box`, `ocr-plate`, `ocr-fig`, `ocr-overlay`, `dht-` or
  `data-theme`.

**Status:** `[ ]` not done

---

### Step 01.2 — Fill it from both editions, and reconcile the image role

**Files:** `internal/appearance/appearance.json`
**Depends on:** Step 01.1

**Prompt for developer:**
> Fill `roles` from the values shipping today: `container` and `plate` from
> `internal/ocr/overlay.go` `ocrCSS` (7 and 14 declarations, byte-identical to
> `extension/src/ocr-overlay.css` as of 2026-08-15). For `image`, take the desktop side's five
> declarations - `display`, `width`, `height`, `margin`, `max-height` - because `margin:0` and
> `max-height:none` are the reset guard the extension currently applies from its reader stylesheet
> instead; the role is the correct home for them. Carry the existing explanatory comments across as
> `note` fields on `padding`, `border-radius` and `background` so the measurements behind them are
> not lost. Fill `themes` from the palette table in `docs/PARITY.md`, normalizing every colour to
> 6-digit lowercase hex.

**Verification:**
- `roles.container` has 7 entries; `roles.plate` has 14 entries; `roles.image` has 5 entries.
- `roles.image` contains entries with `property` values `margin` and `max-height`.
- `roles.plate` contains no entry with `property` equal to `box-shadow`.
- Every entry in `themes` has exactly the eight keys `bg`, `fg`, `muted`, `barBg`, `barFg`,
  `border`, `accent`, `link`; `themes` has exactly 4 entries.
- Every colour value matches `^#[0-9a-f]{6}$`.

**Status:** `[ ]` not done

---

### Step 01.3 — Seed the divergence list and document the format

**Files:** `internal/appearance/appearance.json`, `internal/appearance/README.md`
**Depends on:** Step 01.2

**Prompt for developer:**
> Seed `divergences` with the differences already recorded in `docs/PARITY.md`: the plate selector
> names, the OCR toggle class names, and the custom-property prefix on the desktop palette. Each
> entry states what differs and why it is allowed to. Then write `internal/appearance/README.md`:
> what the file is, that it is the only place these values may be edited, which two generators read
> it, and that adding a declaration here is how a change reaches both editions.

**Verification:**
- `divergences` has at least 3 entries, each with non-empty `what` and `reason`.
- File `internal/appearance/README.md` exists and names both `internal/appearance` (Go) and
  `extension/scripts/gen-appearance.mjs` (JS) as its consumers.

**Status:** `[ ]` not done

## Phase done criteria

- [ ] Every `Step 01.*` is `[x] done`.
- [ ] `go run ./tools/... ` is not required; the narrowest check is `./scripts/lint.ps1` plus a JSON
      parse - run `node -e "JSON.parse(require('fs').readFileSync('internal/appearance/appearance.json','utf8'))"`.
- [ ] Grep for `TODO(phase-01)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes

Establishes: the source of truth, its schema, and the rule that naming is never in it. Phases 02 and
03 both consume this file and nothing else from each other.

## Rollback plan

Revert phase commit(s). Nothing else reads the file yet.
