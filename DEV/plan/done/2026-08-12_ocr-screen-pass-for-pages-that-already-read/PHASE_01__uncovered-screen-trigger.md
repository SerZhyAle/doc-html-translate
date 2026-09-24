# Phase 01 - The uncovered-screen trigger

**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** -
**Steps done:** 3 / 3

## Objective

Let the screen detector answer a narrower question than "does this picture carry a screen": *is there
screened area the reader has no plate over*. That is the trigger the strategic spec asks for, and it
reuses the sweep that already exists rather than paying for a second recognition to find out.

No behaviour changes in this phase - the existing callers pass no covered rectangles and must get
byte-identical answers.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/screen.go` | Modified | ≤ 300 |
| `internal/ocr/screen_test.go` | Modified | ≤ 280 |
| `extension/src/ocr-screen.js` | Modified | ≤ 190 |
| `extension/test/ocr-screen.test.mjs` | Modified | - |

## Steps

### Step 01.1 - Go: measure the screen outside the plates

**Files:** `internal/ocr/screen.go`

**Prompt for developer:**
> Split `screenPitch` into `screenPitch(g)` and `screenPitchOutside(g, covered []image.Rectangle)`,
> the first calling the second with no rectangles so every existing caller is unchanged. In the tile
> loop, skip a tile that an entry of `covered` overlaps by more than a new constant
> `ocrScreenTileCoverMax` of the tile's area - a tile the reader already has a plate over is not
> evidence of unserved screened area, and counting it would let a screened balloon trigger a sweep
> that can only re-find what is there. Half, not "touches at all": on a dense page a plate clipping a
> tile corner would otherwise blind the detector to a caption standing right beside it. Say in the
> comment that the vote share `ocrScreenTileFrac` now applies to the surviving tiles, which is the
> intended reading - a quarter of the *unserved* textured tiles must agree.

**Verification:**
- `go build ./internal/ocr/` succeeds and `func screenPitchOutside(` matches exactly once.
- `go test ./internal/ocr/ -run TestScreen` passes - the existing detector tests call `screenPitch`
  and must be unaffected.

**Status:** `[x]` done - `screenPitchOutside` matches once, `screenPitch` delegates to it with no
rectangles, and the four existing detector tests are unchanged. The skip is `tileServed`, which tests
each covered rectangle on its own rather than as a union: two plates that between them cover a tile
are two regions with a gap down the middle, and that gap is where an unserved caption sits.

---

### Step 01.2 - Go: prove the trigger both fires and holds off

**Files:** `internal/ocr/screen_test.go`

**Prompt for developer:**
> Add one test that builds a synthetic screened image and asserts three things about
> `screenPitchOutside`: with no covered rectangles it returns the same pitch `screenPitch` does; with
> a rectangle over the screened part it returns 0; and with a rectangle over a *different* part it
> still returns the pitch. The third assertion is the one that matters - it is the caption-beside-a-
> balloon case the whole ticket is about, and a trigger that fails it silently does nothing.

**Verification:**
- `go test ./internal/ocr/` passes.
- The test fails when the tile-skip is removed from `screenPitchOutside` (check by hand once).

**Status:** `[x]` done - `TestScreenPitchOutsideAsksTheNarrowerQuestion` passes. Removing the
`tileServed` skip leaves `covered` unread, so the second case would return the pitch instead of 0.

---

### Step 01.3 - Extension: the same trigger

**Files:** `extension/src/ocr-screen.js`, `extension/test/ocr-screen.test.mjs`

**Prompt for developer:**
> Give `screenPitch(grey, width, height, covered = [])` the same fourth argument and the same skip
> rule, exporting `OCR_SCREEN_TILE_COVER_MAX` next to the other detector constants. Rectangles are
> `{x0, y0, x1, y1}`, the shape the rest of the extension's OCR code already uses. Mirror the Go test
> in `ocr-screen.test.mjs`.

**Verification:**
- `npm test` in `extension/` passes.
- `OCR_SCREEN_TILE_COVER_MAX` is exported and equals `ocrScreenTileCoverMax`.

**Status:** `[x]` done - `npm test` 116/116; `OCR_SCREEN_TILE_COVER_MAX` = 0.5 =
`ocrScreenTileCoverMax`. The fourth argument defaults to `[]`, so every existing call is unchanged.

## Phase done criteria

- [x] Every `Step 01.*` is `[x] done`.
- [x] `go test ./internal/ocr/` and `npm test` green.
- [x] No caller's behaviour changed: the detector still answers the whole-image question when asked
      the whole-image question.
