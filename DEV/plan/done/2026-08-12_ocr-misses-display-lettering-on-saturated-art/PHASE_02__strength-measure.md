# Phase 02 - A measure of how much a result actually found

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 4 / 4

## Objective

Add a pure, testable measure of a recognition result's strength and a comparator between two
results, with the weakness floor chosen from the recorded corpus rather than from the reported scene.

## Prerequisites
- [x] Phase 01 is ✅ Done.
- [x] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/strength.go` | New | ≤ 120 |
| `internal/ocr/strength_test.go` | New | ≤ 200 |
| `DEV/research/ocr_display_lettering_2026-08-12.md` | Modified | ≤ 120 |

## Steps

### Step 02.1 - Implement the strength measure
**Files:** `internal/ocr/strength.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `resultStrength(res Result, imgW, imgH int) strength`, returning the three quantities a floor
> could key on: the number of words at or above `ocrMinLineConf`, the fraction of image area the
> plates cover, and the number of lines. Pure - no engine call, no file access. Do not wire it into
> `Recognize` in this phase.

**Verification:**
- File `internal/ocr/strength.go` exists.
- `func resultStrength(` matches exactly once.
- `internal/ocr` compiles: `go build ./internal/ocr/` exits 0.

**Status:** `[x]` done

---

### Step 02.2 - Implement the comparator
**Files:** `internal/ocr/strength.go`
**Depends on:** Step 02.1

**Prompt for developer:**
> Add `strictlyBetter(candidate, current strength) bool`, the only rule by which a retry may replace
> a result that already exists. It must be conservative: a candidate that is not clearly ahead on
> confident words loses, and ties keep the incumbent. Document in one sentence why the rule is not
> symmetric.

**Verification:**
- `func strictlyBetter(` matches exactly once in `internal/ocr/strength.go`.
- `go build ./internal/ocr/` exits 0.

**Status:** `[x]` done

---

### Step 02.3 - Unit-test both against fabricated results
**Files:** `internal/ocr/strength_test.go`
**Depends on:** Step 02.2

**Prompt for developer:**
> Cover: an empty result, a single-word result over a large image (the reported failure shape), a
> sparse but correct result (few words, high confidence, small area - must not be called weak on word
> count alone), and comparator ties. Build the `Result` values in the test; no engine.

**Verification:**
- `go test ./internal/ocr/ -run 'Strength|StrictlyBetter'` exits 0.
- The test file names all four cases.

**Status:** `[x]` done

---

### Step 02.4 - Choose the weakness floor from the recorded corpus
**Files:** `DEV/research/ocr_display_lettering_2026-08-12.md`
**Depends on:** Step 02.3

**Prompt for developer:**
> Compute the strength measure over every scene of the baseline run `temp/ocrlab/fix02` and tabulate
> it against whether the scene actually read. Choose the floor where the two populations separate, and
> record the table, the chosen value and the scenes that sit closest to it on each side. If they do
> not separate, say so and change the signal rather than the number.

**Verification:**
- The research file contains a section naming the chosen floor and the run directory it was derived from.
- The chosen constant appears in `internal/ocr/strength.go` with a comment pointing at that section.
- `go test ./internal/ocr/` exits 0.

**Status:** `[x]` done

## Phase done criteria
- [x] Every `Step 02.*` is `[x] done`.
- [x] `go test ./internal/ocr/` exits 0.
- [x] Grep for `TODO(phase-02)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

`resultStrength` and `strictlyBetter` exist and are defended by measurement, but nothing calls them.
Phase 03 and Phase 04 are their only consumers.

## Rollback plan

Revert the phase commit - no shipped code path changes in this phase.

## Execution notes (2026-08-12)

- **`resultStrength` returns a word count, not a struct of three quantities.** The three were
  candidate signals for a weakness floor; no floor was needed in the end (see Phase 04), so area and
  line count would have been dead code.
- **Step 02.4's floor was not chosen, and the reason is recorded rather than worked around.** The
  strength table over `temp/ocrlab/base06-desktop` is in the research note: the scenes that read carry
  2 to 23 words and 3.9% to 20.9% plated area, while the reported poster's shipped result is 2 words
  and 1.7% area. Word count does not separate the two populations at all, and area separates them by
  a single synthetic scene - which is a difference in image size, not in how much was found. Rule 2 of
  the lab ("no threshold outside a dated report") says that is a reason to change the signal, and the
  signal that needed no threshold is the comparator.
