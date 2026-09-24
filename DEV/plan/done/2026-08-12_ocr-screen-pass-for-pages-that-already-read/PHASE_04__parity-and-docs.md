# Phase 04 - Parity, docs and the cost statement

**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 3 / 3

## Objective

Record the two new shared invariants where drift will be caught, say publicly-in-the-repo what the
pass costs, and leave the ticket's remaining criteria honestly open rather than ticked.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `docs/PARITY.md` | Modified | - |
| `tests/parity_test.go` | Modified | - |
| `DEV/CHANGELOG.md` | Modified | - |
| `DEV/plan/2026-08-12_ocr-screen-pass-for-pages-that-already-read.md` | Modified | - |

## Steps

### Step 04.1 - The drift guard

**Files:** `tests/parity_test.go`

**Prompt for developer:**
> Extend `TestParityOCRScreenRung` with the two new constants (`ocrScreenTileCoverMax`,
> `ocrScreenMergeMaxOverlap` against their `OCR_SCREEN_*` mirrors) and with the sweep's *position*,
> the same way the rung's position is already pinned: both editions must call the sweep on the branch
> where the ordinary pass found plates, not on the empty branch. Position is the invariant that
> matters most here - a sweep moved in front of the ordinary pass, or applied unconditionally, is the
> exact regression the strategic spec measured and rejected.

**Verification:**
- `go test ./tests/ -run TestParityOCRScreenRung` passes.
- It fails when either constant is changed on one side only (check by hand once).

**Status:** `[x]` done - the two constants plus **three** structural facts rather than the one the
prompt asked for: the sweep runs on the branch that already read, it measures only outside the plates,
and it merges rather than replaces. The trigger earned its own guard because a sweep wired to the
whole-image detector would still pass every other check while paying a recognition on every screened
page. Proven by setting `OCR_SCREEN_MERGE_MAX_OVERLAP` to 0.3 on the JS side alone: `merge overlap
bound drift: screen.go=0.2 ocr-screen.js=0.3`, then reverted.

---

### Step 04.2 - `docs/PARITY.md`

**Files:** `docs/PARITY.md`

**Prompt for developer:**
> Under the OCR section, beside the halftone rung, add the additive screen sweep: what triggers it,
> the two constants, that the merge is by union overlap, and the one intentional coordinate
> difference (the extension scales its covered rectangles back up because it downscales blocks
> earlier). Name the guard.

**Verification:**
- `docs/PARITY.md` names both constants and `TestParityOCRScreenRung`.

**Status:** `[x]` done - new "Additive screen sweep" bullet in the OCR section, split into trigger,
merge and confidence, with the coordinate difference recorded as intentional.

---

### Step 04.3 - Changelog and ticket status

**Files:** `DEV/CHANGELOG.md`, the strategic spec

**Prompt for developer:**
> One changelog row covering every modified file. In the strategic spec, tick the edition-parity rows
> the code closes, state the added cost measured from the code path (what runs, and on which images),
> and leave the annotation-dependent Done criteria unticked with the human-owned gate named. Set the
> status to `BlockNeedUserTest` - the remaining criteria are a corpus decision, not a code one.

**Verification:**
- The changelog's newest row lists every file this plan touched.
- The strategic spec's unticked criteria each say what would tick them.

**Status:** `[x]` done - one changelog row at 2026-08-12 05:20:00; all four edition rows ticked; three
Done criteria closed and three left open, each naming the human-owned gate. The first open question is
answered in place and struck through, the second left open on purpose - marking a merged plate is the
first place the overlay would distinguish one plate from another, which is a design question and not a
flag. Status `BlockNeedUserTest`.

## Phase done criteria

- [x] Every `Step 04.*` is `[x] done`.
- [x] `./scripts/test.ps1`, `./scripts/lint.ps1` and `npm test` green.
