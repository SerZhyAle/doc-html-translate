# Phase 03 - Wire the additive sweep

**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01, Phase 02
**Steps done:** 2 / 2

## Objective

The behaviour change. A page the ordinary pass read gets one extra recognition - and only one, and
only when the trigger says there is screened area no plate covers - whose plates are merged in
beside the existing ones.

This is the first phase that can regress a page working today, which is why the trigger and the merge
were proven first.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/tesseract.go` | Modified | ≤ 800 |
| `extension/src/ocr-overlay.js` | Modified | ≤ 700 |

## Steps

### Step 03.1 - Go: `screenSweep` beside the rescue ladder

**Files:** `internal/ocr/tesseract.go`

**Prompt for developer:**
> In `Recognize`, keep the empty-result branch exactly as it is and add its opposite: when the
> ordinary pass produced plates, pass them through `screenSweep(bin, ocrPath, lang, dataDir, dpi,
> res.Blocks)`. `screenSweep` builds the grey rendition, calls `screenPitchOutside` with the existing
> plates' boxes, and returns the input untouched when there is no pitch, when the copy cannot be
> built or when recognition errors - a sweep that cannot run leaves the page exactly as it was.
> Otherwise it recognizes the low-passed copy at `ocrRescueLineConf` with the engine-default
> thresholder, the same staging `screenRescue` uses, and returns `mergeScreenBlocks`.
>
> Both this and the ladder run before `scaleDown`, so every rectangle here is in prepared-image
> coordinates and nothing needs rescaling.
>
> Write down why the floor is inherited rather than derived: the local prior is the rescue prior -
> nothing was found *in this region* - while a wrong plate costs more here, because it lands on a page
> the reader is otherwise happy with. That makes 80 a lower bound, and re-deriving it needs the
> annotated whole pages the plan's human-owned gate names.

**Verification:**
- `func screenSweep(` matches exactly once; `go build ./...` and `go vet ./internal/ocr/` clean.
- `go test ./internal/ocr/` passes - in particular the existing rescue tests, which must still see
  the empty-result path unchanged.
- Reading `Recognize`: the empty branch is untouched and the new branch cannot reach it.

**Status:** `[x]` done - `screenSweep` matches once; the new branch is the `else` of the existing
`len(res.Blocks) == 0`, so the two are exclusive by construction. `go build ./...`, `go vet` and
`./scripts/test.ps1` clean.

---

### Step 03.2 - Extension: the same sweep

**Files:** `extension/src/ocr-overlay.js`

**Prompt for developer:**
> Mirror it in `recognize`: when the ordinary pass returned blocks, `blocks = await
> screenSweep(worker, image, scale, blocks)`. The extension's blocks are already divided by `scale`
> while the prepared image the detector reads is not, so the covered rectangles must be multiplied
> back up before `screenPitch` sees them - note that as the one coordinate difference between the
> editions, since it follows from where each edition puts its downscale. Everything else matches:
> the same blur filter, the same rescue floor, the same merge.

**Verification:**
- `npm test` in `extension/` passes.
- `screenSweep` appears in `ocr-overlay.js` and is called only on the non-empty branch.

**Status:** `[x]` done - wired as the `else` of `if (!blocks.length)`; `npm test` 117/117. The module
loads far enough to prove it parses (it then stops on `self is not defined`, the vendored worker
bundle expecting a browser).

## Phase done criteria

- [x] Every `Step 03.*` is `[x] done`.
- [x] `./scripts/test.ps1` green, `./scripts/lint.ps1` green, `npm test` green.
- [x] A picture with no screen outside its plates issues no second `recognize` in either edition -
      both return `kept` from the `pitch == 0` branch, before any blur or recognition.

## What the sweep costs, from the code path

The second recognition is the expensive part and it is the part the trigger gates: it runs only on an
image where the detector finds a lattice in the area no plate covers. What is spent unconditionally,
on every image that reads, is the grey rendition (one decode plus a luminance pass) and the detector
sweep - bounded at `ocrScreenMaxTiles` = 96 tiles whatever the page size, which is why a 600-DPI
magazine page costs the same measurement as a panel. On material with no screen that is the whole
bill, and it is far below the recognition it decides against.

How often the trigger fires on material that gains nothing is a corpus question, not a code one, and
it is one of the numbers the human-owned gate would produce.
