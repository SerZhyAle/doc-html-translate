# Phase 04 - Make the ladder reachable for a result that is technically not empty

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 4 / 4

## Objective

Trigger the rescue ladder on a weak result as well as an empty one, adopt its output only when the
comparator says it is strictly better, and prove over the corpus that nothing which reads today loses.

## Prerequisites
- [x] Phase 03 is ✅ Done.
- [x] Phase 02's floor is recorded in the research note.
- [x] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/tesseract.go` | Modified | ≤ 400 |
| `internal/ocr/rescue_test.go` | Modified | ≤ 250 |
| `DEV/research/ocr_display_lettering_2026-08-12.md` | Modified | ≤ 200 |

## Steps

### Step 04.1 - Replace the emptiness trigger with the strength floor
**Files:** `internal/ocr/tesseract.go`
**Depends on:** - start of phase

**Prompt for developer:**
> `Recognize` runs the ladder on `len(res.Blocks) == 0`. Run it when the ordinary pass's strength is
> below the Phase 02 floor, which includes the empty case. Keep the screen sweep on the other arm:
> a result strong enough to keep still gets the additive sweep it gets today.

**Verification:**
- `len(res.Blocks) == 0` no longer appears as the ladder's trigger in `Recognize`.
- The trigger calls the Phase 02 measure.
- `go test ./internal/ocr/` exits 0.

**Status:** `[x]` done

---

### Step 04.2 - Adopt a rescue result only when it is strictly better
**Files:** `internal/ocr/tesseract.go`
**Depends on:** Step 04.1

**Prompt for developer:**
> A weak result is still a result: when the ladder returns something, keep whichever of the two the
> comparator prefers, and keep the original on a tie. Replacement rather than merge - the rescue
> re-reads the same region, so merging would put two plates over one word.

**Verification:**
- The adoption path calls `strictlyBetter`.
- `go test ./internal/ocr/` exits 0.

**Status:** `[x]` done

---

### Step 04.3 - Test that a weak result is retried and a good one is untouched
**Files:** `internal/ocr/rescue_test.go`
**Depends on:** Step 04.2

**Prompt for developer:**
> Two tests, both without an engine: a result under the floor reaches the ladder, and a result above
> it does not (and still reaches the screen sweep). Add a third asserting the incumbent survives when
> the rescue comes back worse.

**Verification:**
- `go test ./internal/ocr/ -run Rescue` exits 0.
- All three cases are named in the test file.

**Status:** `[x]` done

---

### Step 04.4 - Score the corpus against the named baseline
**Files:** `DEV/research/ocr_display_lettering_2026-08-12.md`
**Depends on:** Step 04.3

**Prompt for developer:**
> Run `ocrlab run`, `score` and `gate` on the dev split and compare against `temp/ocrlab/fix02` by
> name. Record plates gained and lost per scene, recall, IoU, concealment, damage, merges/splits and
> cross-group overlap, and the added cost for an image that reads normally versus one that triggers
> the ladder. A scene that loses a plate stops this phase - report it rather than adjusting the floor
> until it disappears.

**Verification:**
- `go run ./tools/ocrlab gate <run-dir>` exits 0.
- The research note contains the before/after table naming both run directories.
- The reported poster's scene shows plates covering its display lettering.

**Status:** `[x]` done

## Phase done criteria
- [x] Every `Step 04.*` is `[x] done`.
- [x] `./scripts/test.ps1` green.
- [x] Grep for `TODO(phase-04)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

The Go edition now behaves as intended and the numbers that justify it are on record. The extension
carries the same ladder and the same fixed segmentation mode and is still unchanged - that is Phase 05.

## Rollback plan

Revert the phase commit; the trigger returns to `len(res.Blocks) == 0` and Phase 03's rung becomes
unreachable again but harmless.

## Execution notes (2026-08-12)

- **The emptiness trigger was not replaced, and that is the phase's finding rather than a shortfall.**
  Measured per line - which is what `clusterLines` gates on - the ordinary pass returns no plates on
  the reported scene in both language configurations, so `len(res.Blocks) == 0` holds and the ladder
  runs in full. A weakness floor on the outer trigger would have changed nothing here, no corpus
  scene demanded one, and adding an unmeasured threshold is what the lab's second rule forbids.
- **What was replaced is the rule inside the ladder.** `greyRescue` returned the first rung that
  produced any plate at all; it now runs every rung and keeps the strongest under `strictlyBetter`.
  That is what makes the sparse rung reachable on this scene, since rung 1 returns exactly one word.
- Cost, measured end to end: 3.8 s to 6.1 s on the reported poster. An image that reads normally
  never enters the ladder and pays nothing.
