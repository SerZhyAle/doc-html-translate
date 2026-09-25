# Phase 03 — The extension derives its CSS

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 5 / 5

## Objective

The extension's overlay declarations and reader palette are written by a generator from the canonical
source into marked blocks inside the stylesheets that already ship, and the image reset guard moves
from the reader stylesheet onto the image role.

## Prerequisites

- [x] Phase 01 is ✅ Done.
- [x] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/scripts/gen-appearance.mjs` | New | ≤ 200 |
| `extension/src/ocr-overlay.css` | Modified | ≤ 105 |
| `extension/src/viewer.css` | Modified | ≤ 60 changed |
| `extension/package.json` | Modified | ≤ 5 changed |
| `extension/build.mjs` | Modified | ≤ 10 changed |

> Generated content goes into marked blocks **inside the existing stylesheets**, not into a new file:
> the manifest's `web_accessible_resources`, `ocr.html`'s link and `collectExportCss` in `viewer.js`
> all name these two files, and the export path inlines them verbatim. A third file would have to be
> added to each of those, for nothing.

## Steps

### Step 03.1 — Write the generator

**Files:** `extension/scripts/gen-appearance.mjs`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/scripts/gen-appearance.mjs`. It reads `../internal/appearance/appearance.json`
> relative to the repo root, and rewrites the region between
> `/* >>> generated from internal/appearance - do not edit by hand */` and
> `/* <<< generated */` in a target stylesheet. It takes the target and the naming per call, so the
> overlay call emits container `.ocr-overlay`, image `.ocr-overlay-img`, plate `.ocr-plate`, hidden
> plate `html.ocr-layer-off .ocr-plate`, and the palette call emits an unprefixed custom-property set
> under `:root` plus `html[data-theme=..]` blocks. Emit each `note` as a CSS comment. Exit non-zero
> if a marker pair is missing.

**Verification:**
- File `extension/scripts/gen-appearance.mjs` exists.
- It contains the literal marker strings `>>> generated from internal/appearance` and
  `<<< generated`.
- `node extension/scripts/gen-appearance.mjs --check` exits non-zero when a marker is absent.

**Status:** `[x]` done

---

### Step 03.2 — Generate the overlay block

**Files:** `extension/src/ocr-overlay.css`
**Depends on:** Step 03.1

**Prompt for developer:**
> Add the marker pair to `ocr-overlay.css` around the three role rules and run the generator. The
> hand-written declarations for `.ocr-overlay`, `.ocr-overlay-img`, `.ocr-plate` and the hidden-plate
> rule are replaced by generated output. Keep the file's own prose comments that explain *why* the
> unit exists - the per-declaration measurements now arrive as generated comments, so delete the
> hand copies that would otherwise say the same thing twice. `.ocr-badge` and `.ocr-empty` are not
> roles in the source and stay hand-written.

**Verification:**
- `extension/src/ocr-overlay.css` contains exactly one marker pair.
- `.ocr-badge` still matches exactly once outside the generated region.
- Re-running the generator leaves the file byte-identical.

**Status:** `[x]` done

---

### Step 03.3 — Move the image reset guard onto the role

**Files:** `extension/src/viewer.css`
**Depends on:** Step 03.2

**Prompt for developer:**
> Delete the `#content .ocr-overlay-img { margin: 0; max-height: none; }` override and its comment
> from `viewer.css`. The generated `.ocr-overlay-img` rule now carries `margin` and `max-height`
> from the source, so the guard applies in the standalone OCR page too, where the reader stylesheet
> is not loaded. Verify by eye that a converted page in the viewer still positions plates on the
> image - the `#content img` reset it was neutralizing is still there and must lose to the role rule
> on specificity or order.

**Verification:**
- `#content .ocr-overlay-img` no longer matches in `extension/src/viewer.css`.
- `#content img` still matches exactly once in `extension/src/viewer.css`.
- The generated region of `ocr-overlay.css` contains `max-height` and `margin` under the image
  selector.

**Status:** `[x]` done

---

### Step 03.4 — Generate the palette block

**Files:** `extension/src/viewer.css`
**Depends on:** Step 03.1

**Prompt for developer:**
> Add a marker pair to `viewer.css` around the `:root` colour tokens and the three
> `html[data-theme=..]` blocks, and run the palette generator into it. The reader font and size
> custom properties in `:root` are not palette and stay hand-written outside the markers.

**Verification:**
- `extension/src/viewer.css` contains exactly one marker pair.
- `--reader-size` still matches exactly once, outside the generated region.
- The generated region contains 4 selectors and 32 colour declarations.
- Re-running the generator leaves the file byte-identical.

**Status:** `[x]` done

---

### Step 03.5 — Wire the generator into the build

**Files:** `extension/package.json`, `extension/build.mjs`
**Depends on:** Step 03.2, Step 03.4

**Prompt for developer:**
> Add an `appearance` npm script running the generator, and call it from `build.mjs` before `zip` so
> a package can never ship a stale block. Add a `--check` mode invocation to the same build path
> that fails when regeneration would change a file.

**Verification:**
- `"appearance"` matches exactly once in `extension/package.json` scripts.
- `gen-appearance` matches in `extension/build.mjs`.
- `npm run appearance` in `extension/` exits 0 and leaves the tree unchanged.

**Status:** `[x]` done

## Phase done criteria

- [x] Every `Step 03.*` is `[x] done`.
- [x] `npm test` in `extension/` exits 0 (149 tests on 2026-09-24, with `vendor/` populated).
- [x] Grep for `TODO(phase-03)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Establishes: the marker convention and the generator, so the extension's copy of these declarations
is a render target. Phase 04 asserts that nothing outside the markers declares a role.

## Execution notes (2026-09-24)

- Step 03.3: the `#content img` reset is an id selector and outranks the `.ocr-overlay-img` role rule
  whatever the order, so deleting the override alone would have re-grown the container. The reset is
  now `#content img:not(.ocr-overlay-img)`; the role rule carries `margin` / `max-height` everywhere.
  Checked by a headless-Chromium render, old CSS against new, pixel-identical (see INDEX change log);
  the same harness does see the bug when the `:not()` is removed.
- Step 03.5: `npm run build` runs `appearance` after `stamp`; `node build.mjs zip` runs the generator's
  `--check` and refuses to package a stale region.

## Rollback plan

Revert phase commit(s). The stylesheets return to hand-written form, including the `#content`
override.
