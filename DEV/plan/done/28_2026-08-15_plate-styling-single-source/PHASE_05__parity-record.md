# Phase 05 - Record the rule and mark what it does not cover

**Strategic spec:** [`../28_2026-08-15_plate-styling-single-source.md`](../28_2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ⛔ Blocked - Step 05.3 only (catalog not reachable from the session that did 05.1-05.2)
**Depends on:** Phase 04
**Steps done:** 2 / 3

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

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
| `ocr-overlay/ocr-pipeline.md` (`OCR-PIPELINE`, shared contracts catalog) - ⛔ Local only | Modified | ≤ 20 changed |

## Steps

### Step 05.1 - State the single source

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

### Step 05.2 - Mark every invariant guarded or prose-only

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

### Step 05.3 - Repoint the OCR pipeline doc

**⛔ Local only - changes the contract catalog.**

**Files:** the shared contracts catalog, `ocr-overlay/ocr-pipeline.md` (contract `OCR-PIPELINE` 1.0; the
contract lives in the catalog, not in this repo - see `docs/contracts/OCR-PIPELINE.md`). Current target
text quoted in "Contract snapshot (2026-09-25)" below.
**Depends on:** Step 05.1

**Prompt for developer:**
> Update the plate-rendering rows and §3.1 so they name the canonical source as the place the plate's
> appearance is defined, with the two generators as consumers, instead of pointing at `overlay.go`
> and `ocr-overlay.css` as two parallel definitions. The measurement prose about the paper carrier
> and the padding stays where it is - it explains a decision, not a value.

Concretely, against the snapshot: the source-pointer quote closing §3.1 (`overlay.go`: `wrapImage`,
`percentStyle`, `ocrCSS` - `ocr-overlay.js`: `buildOverlay` + `ocr-overlay.css`) and the section 6
"Plate rendering, colours, re-fit" row are the two places that name the parallel definitions. Per the
pointer file's rule ("a breaking change gets a new dated section and a version bump"), this is not
breaking - no value moves - so it is an in-place edit plus a `correction` row in the document log.

**Verification:**
- `internal/appearance` matches at least once in the catalog's `ocr-overlay/ocr-pipeline.md`
  (re-verified 2026-09-25: 0 matches today).
- The §3.1 paragraph on the opaque paper carrier is unchanged in wording.
- The document log gains one row for the edit.

**Status:** ⛔ blocked - the catalog lives at the path named in `AGENTS.md`, a Windows drive that is
not mounted in the Linux session this ran in. The in-repo pointer `docs/contracts/APP-STYLE.md` was
updated (it named `TestParityThemePalette`); `docs/contracts/OCR-PIPELINE.md` names no file this ticket
moved. To do on the owner's machine: in the catalog's `ocr-overlay/ocr-pipeline.md`, repoint the
plate-rendering rows and §3.1 at `internal/appearance/appearance.json` as the single definition, with
`internal/appearance` (Go) and `extension/scripts/gen-appearance.mjs` (JS) as its consumers.

## Phase done criteria

- [ ] **⛔ Waits on Step 05.3 (local).** Every `Step 05.*` is `[x] done` - 05.3 edits the catalog, which
      only the owner's machine mounts.
- [ ] `./scripts/typo.ps1` exits 0 - not run (no PowerShell); `tests/typography_test.go` green.
- [x] Grep for `TODO(phase-05)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - OCR-PIPELINE 1.0. A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

The in-repo facts the new text must name, re-verified 2026-09-25: `internal/appearance/appearance.json` is
the one description; `internal/ocr/overlay.go` derives `ocrCSS` from it at run time
(`var ocrCSS = appearance.OverlayCSS(OverlayStyleNames)`); `extension/scripts/gen-appearance.mjs` writes the
marked region of `extension/src/ocr-overlay.css` ("generated from internal/appearance - do not edit by
hand"); `tests/appearance_parity_test.go` is the gate.

From the shared contracts catalog, `ocr-overlay/ocr-pipeline.md`, section 3.1, in full:

> ### 3.1 Geometry
>
> The image is wrapped in a **block** container carrying `aspect-ratio: W / H` and
> `container-type: inline-size`. Block with an explicit width, not inline-block: a shrink-to-fit box
> with size containment collapses to zero inline size and hides the image and every plate. The image
> is `width:100%` with `margin:0` and `max-height:none` so a host page's `img` reset cannot offset or
> shrink it and drift the plates.
>
> Each plate is positioned in percent of natural image size - `left`, `top`, `width`, and
> `min-height` (not `height`, so the plate may grow) - and its font is set in `cqw`, percent of
> container width, derived from the block's median line height times a **0.92** fit factor. Percent
> plus `cqw` is what makes the overlay survive responsive scaling with no JS at all: measured at ten
> browser states - device scale and page zoom at 100/125/150/200 %, plus tablet and phone - the worst
> plate-edge movement is 0 px of the source image.
>
> That whole property rests on the image filling the container, because the percentages are of the
> container and the measurement divides by the image. Anything that sizes the picture independently
> breaks it silently - the page's own navbar script used to write an inline `width` on every image,
> which beats `.ocr-fig>img{width:100%}`, and a 640 px scene in a 1216 px column rendered every plate
> 1.9x too large and off its text while still measuring 0 drift, because it was equally wrong
> everywhere. The guard now skips images inside `.ocr-fig`.
>
> **The opaque paper is on the plate box**, and the text sits directly in it. This is a decision taken
> against the corpus, not a default: the paper spent one day (2026-08-13) on an inline span hugging the
> string, so that it took the shape of the rendered words rather than of the block rectangle. That shape
> is genuinely better where the plate is much wider than its last line - the rectangle put 91 px of paper
> over a photograph on either side of a 984 px caption''s 759 px last line - but a plate exists to
> *conceal* what it replaces, and over the 46 lab scenes the string carrier left a mean **93%** of the
> source lettering still showing against **17%** for the box, against a recorded bound of 0.28. Both
> layers then read at once and the page is harder to read than the untouched scan. The box''s over-cover
> is what the coverage rule in 2.5 and the type-size rule bound from the other side.
>
> > `overlay.go`: `wrapImage`, `percentStyle`, `ocrCSS` - `ocr-overlay.js`: `buildOverlay` +
> > `ocr-overlay.css`

From the same file, section 6 "Module map" - the header and the plate-rendering row:

> | Responsibility | Go | JavaScript |
> |----------------|----|------------|
> | Plate rendering, colours, re-fit | [`internal/ocr/overlay.go`](https://github.com/SerZhyAle/doc-html-translate/blob/main/internal/ocr/overlay.go) | `ocr-overlay.js` + `ocr-overlay.css` |

From the same file, the rule for editing it (the "Document log" preamble):

> Append-only. A row is never rewritten: a consumer may have shipped against what it said.

## Handoff notes

Establishes: the document now distinguishes an invariant that is enforced from one that is merely
written down - the distinction this whole ticket came from.

## Rollback plan

Revert phase commit(s).
