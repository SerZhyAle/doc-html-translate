# Phase 03 - A sparse-text rung for input that is not a page

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02
**Steps done:** 3 / 3

## Objective

Make the page-segmentation mode a parameter of a pass instead of a fixed constant, and add a
sparse-text rescue rung that runs on the grey rendition.

## Prerequisites
- [x] Phase 02 is ✅ Done.
- [x] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/tesseract.go` | Modified | ≤ 400 |
| `internal/ocr/rescue_test.go` | Modified | ≤ 200 |

## Steps

### Step 03.1 - Carry the segmentation mode into the pass
**Files:** `internal/ocr/tesseract.go`
**Depends on:** - start of phase

**Prompt for developer:**
> `tesseractArgs` hardcodes `ocrPageSegMode`. Take the mode as an argument and thread it through
> `recognizePass`. Keep `ocrPageSegMode` as the value every existing caller passes, so this step
> changes no command line the app builds today - the ordinary pass, the grey rungs and the screen
> rung must produce byte-identical arguments to what they produce now.

**Verification:**
- `func tesseractArgs(` names a page-segmentation parameter.
- No literal `ocrPageSegMode` remains inside `tesseractArgs`.
- `go test ./internal/ocr/` exits 0, including the existing argument tests.

**Status:** `[x]` done

---

### Step 03.2 - Add the sparse rung to the ladder
**Files:** `internal/ocr/tesseract.go`
**Depends on:** Step 03.1

**Prompt for developer:**
> Add a rung that recognizes the grey rendition with the sparse-text segmentation mode, at
> `ocrRescueLineConf`. Place it after the existing grey thresholding rungs and before the halftone
> screen rung, and name the constant for the mode next to `ocrPageSegMode` so both are visible in one
> place. Measured basis: the reported scene gains three lines of text from this rung and nothing else
> in the ladder recovers them.

**Verification:**
- A named constant for the sparse mode is declared exactly once in `internal/ocr/tesseract.go`.
- The rung appears in the ladder between the thresholding rungs and the screen rung.
- `go test ./internal/ocr/` exits 0.

**Status:** `[x]` done

---

### Step 03.3 - Test the rung's position and its arguments
**Files:** `internal/ocr/rescue_test.go`
**Depends on:** Step 03.2

**Prompt for developer:**
> Assert the order of the ladder by expression rather than by value - the rungs the app tries, in the
> order it tries them - and assert that the ordinary pass still asks for the page mode while the
> sparse rung asks for the sparse mode. A test that pins only the integers would pass while the rungs
> were swapped.

**Verification:**
- `go test ./internal/ocr/ -run Rescue` exits 0.
- The test names both modes and the rung order.

**Status:** `[x]` done

## Phase done criteria
- [x] Every `Step 03.*` is `[x] done`.
- [x] `go test ./internal/ocr/` exits 0.
- [x] Grep for `TODO(phase-03)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

The ladder can now read sparse display type, but it is still reachable only for an image that
produced nothing at all. Phase 04 is what makes it reachable for the reported case.

## Rollback plan

Revert the phase commit. The ladder is additive here: removing the rung restores the previous order.

## Execution notes (2026-08-12)

- `greyRescuePasses` became `[]rescueRung` - a thresholding method paired with a segmentation mode -
  rather than two parallel lists, so a rung cannot half-exist. `TestGreyRescueLadderOrder` and
  `TestParityOCRGreyRescue` were both updated to the new shape.
- `TestScreenRungIsAfterTheGreyLadder` pinned the ladder's *length* at 2, which would break on every
  future rung. It now pins what it means: every rung in the ladder reads the unaltered grey
  rendition, and the screen rung is the one after them.
