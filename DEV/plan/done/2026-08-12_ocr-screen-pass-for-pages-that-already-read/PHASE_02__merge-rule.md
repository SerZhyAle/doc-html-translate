# Phase 02 - The merge rule

**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 3 / 3

## Objective

Decide which screen-pass plates may join a page that already read, without ever moving or dropping
one the ordinary pass produced. A duplicate plate over lettering that is already covered is visible
damage - worse than the caption staying untranslated - so the rule errs towards dropping.

Still no behaviour change: this phase writes a pure function and its tests, and wires nothing.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/screen.go` | Modified | ≤ 380 |
| `internal/ocr/screen_test.go` | Modified | ≤ 280 |
| `extension/src/ocr-screen.js` | Modified | ≤ 240 |
| `extension/test/ocr-screen.test.mjs` | Modified | - |

**Deviation:** the budgets for `screen.go` and `ocr-screen.js` were raised from 300 / 190 while the
step ran. They were guessed before the union-overlap helper was written, and the alternative - a
separate `merge.go` against one JS file - would have split a pair the parity guard reads as a pair
for no gain. Final: 358 and 224.

## Steps

### Step 02.1 - Go: the union-overlap merge

**Files:** `internal/ocr/screen.go`

**Prompt for developer:**
> Add `mergeScreenBlocks(kept, found []Block) []Block`: return `kept` followed by those of `found`
> whose box is covered by at most `ocrScreenMergeMaxOverlap` of its own area by the boxes already
> accepted - the ordinary plates *and* the screen plates taken so far, so two screen plates cannot
> stack on each other either. Measure the coverage as the **union** of the overlaps, in a helper
> `coveredFraction(r image.Rectangle, rects []image.Rectangle) float64` that compresses the
> rectangles' own edge coordinates into a grid and sums the cells lying inside any of them. The union
> rather than a sum is what answers the strategic spec's "what happens to a new plate that overlaps
> two": adding two overlapping contributions double-counts, and a candidate two thirds covered by two
> neighbours would pass a sum-based rule that reported over 100%.
>
> State in the comment why the bound is a fraction of the *new* plate rather than of the existing
> one: the question is whether this lettering is already plated, and a small candidate sitting inside
> a large existing plate is a duplicate however little of that plate it occupies.

**Verification:**
- `go build ./internal/ocr/` succeeds; `func mergeScreenBlocks(` and `func coveredFraction(` each
  match exactly once.

**Status:** `[x]` done - both match once, plus a small `blockRects` helper the trigger also wants.
The accepted-so-far rectangles are carried in a slice rather than rebuilt from the output on each
candidate.

---

### Step 02.2 - Go: prove the three cases

**Files:** `internal/ocr/screen_test.go`

**Prompt for developer:**
> One test with three cases: a candidate clear of every existing plate is kept; a candidate on top of
> one is dropped; a candidate whose halves lie under two different existing plates is dropped, and
> would be kept by a rule that looked at each existing plate separately. Assert as well that the
> returned slice starts with `kept` unchanged - the "every existing plate is still there" invariant
> is the one a future refactor is most likely to break quietly.

**Verification:**
- `go test ./internal/ocr/` passes.
- The straddling case fails if `coveredFraction` is replaced by a per-rectangle maximum.

**Status:** `[x]` done - `TestMergeScreenBlocksDropsWhatIsAlreadyPlated` passes. The test asserts the
straddling property itself rather than only the outcome: each existing plate covers 0.175 of the
candidate, under the 0.2 bound, and the two together cover 0.35. A per-rectangle maximum would
therefore keep it. The first draft of the case was built to be *fully* covered by two plates, which
is impossible while each stays under the bound - corrected rather than left as a weaker assertion.

---

### Step 02.3 - Extension: the same rule

**Files:** `extension/src/ocr-screen.js`, `extension/test/ocr-screen.test.mjs`

**Prompt for developer:**
> Port `mergeScreenBlocks` and `coveredFraction` over the extension's `{bbox: {x0, y0, x1, y1}}`
> blocks, exporting `OCR_SCREEN_MERGE_MAX_OVERLAP`. Mirror the three cases in the test file.

**Verification:**
- `npm test` in `extension/` passes.
- `OCR_SCREEN_MERGE_MAX_OVERLAP` equals `ocrScreenMergeMaxOverlap`.

**Status:** `[x]` done - `npm test` 117/117, same three cases, same numbers.

## Phase done criteria

- [x] Every `Step 02.*` is `[x] done`.
- [x] `go test ./internal/ocr/` and `npm test` green.
- [x] Nothing calls the merge yet - phase 03 wires it.
