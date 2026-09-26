# Phase 05 - Port both changes to the extension and pin them

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 04
**Steps done:** 4 / 4

## Objective

Give the browser edition the same strength floor, comparator and sparse rung, and make the two
editions unable to drift on any of them.

## Prerequisites
- [x] Phase 04 is ✅ Done.
- [x] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/ocr-overlay.js` | Modified | ≤ 400 |
| `extension/test/ocr-overlay.test.mjs` | Modified | ≤ 250 |
| `docs/PARITY.md` | Modified | ≤ 60 |
| `tests/parity_test.go` | Modified | ≤ 150 |

## Steps

### Step 05.1 - Port the strength measure and comparator
**Files:** `extension/src/ocr-overlay.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Hand-port `resultStrength` and `strictlyBetter` with the same floor and the same tie rule. Port,
> do not re-derive: a second derivation is how the two editions drift.

**Verification:**
- Both function names appear in `extension/src/ocr-overlay.js`.
- The floor constant's value equals the Go constant's value.

**Status:** `[x]` done

---

### Step 05.2 - Port the trigger and the sparse rung
**Files:** `extension/src/ocr-overlay.js`
**Depends on:** Step 05.1

**Prompt for developer:**
> Replace the zero-plate trigger with the floor, adopt under the comparator, and add the sparse rung
> in the same ladder position the Go side uses. Check how tesseract.js takes a page-segmentation mode
> before writing the rung - it is a worker parameter there, not a command-line flag.

**Verification:**
- The ladder in `ocr-overlay.js` lists the rungs in the Go order.
- `npm test --prefix extension` exits 0.

**Status:** `[x]` done

---

### Step 05.3 - Test the ported behaviour on the JS side
**Files:** `extension/test/ocr-overlay.test.mjs`
**Depends on:** Step 05.2

**Prompt for developer:**
> Mirror the Go tests: weak result retried, strong result untouched, worse rescue rejected. Same
> cases, same names where the language allows, so a reader can diff the two suites.

**Verification:**
- `npm test --prefix extension` exits 0 and the three cases are named in the file.

**Status:** `[x]` done

---

### Step 05.4 - Record and pin the shared invariants
**Files:** `docs/PARITY.md`, `tests/parity_test.go`
**Depends on:** Step 05.3

**Prompt for developer:**
> Add a section for the weakness floor, the comparator rule, the rung order and the two segmentation
> modes, then extend the parity test to pin them across editions by expression and not only by value -
> the pattern `TestParityOCRScreenRung` already uses.

**Verification:**
- `docs/PARITY.md` contains the new section naming all four items.
- `go test ./tests/ -run Parity` exits 0.

**Status:** `[x]` done

## Phase done criteria
- [x] Every `Step 05.*` is `[x] done`.
- [x] `./scripts/test.ps1` green and `npm test --prefix extension` green.
- [x] Grep for `TODO(phase-05)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Both editions behave the same and a drift now fails a test rather than reaching a user.

## Rollback plan

Revert the phase commit. The Go edition keeps Phase 04's behaviour; the parity test reverts with it.

## Execution notes (2026-08-12)

- `resultStrength` / `strictlyBetter` live in `ocr-cluster.js`, not `ocr-overlay.js`. They are pure
  functions over clustered blocks, and `ocr-overlay.js` imports the vendored tesseract.js at module
  load - putting them there would have meant loading the whole recognizer to unit-test two lines.
  `docs/PARITY.md` and the parity test name the file they are actually in.
- The port carries a fourth item the plan did not have: the plate-colour orientation fix
  (`ringNearerInk` / `RING_MIN_SAMPLES`), pinned by `TestParityOCRPlateColourOrientation`.
- tesseract.js takes the segmentation mode as a worker parameter, so the rung sets
  `tessedit_pageseg_mode` alongside `thresholding_method` and both are restored in the `finally`.
