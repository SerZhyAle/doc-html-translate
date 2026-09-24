# The lab drops the DevTools connection on a detailed scan

**Status:** Implemented (2026-08-12)
**Priority:** 38
**Date:** 2026-08-12

> Cross-edition ticket. One defect = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

The file name records the symptom as it was first seen. The finding is different, and this is the
correction: **the shipped extension never crashed anything.** The browser stayed up throughout. What
died was the lab runner's own connection to it, and the 17 scenes recorded as extension crashes in the
first cross-edition baseline were never crashes - they were a measurement the instrument failed to
collect and then blamed on the thing it was measuring.

## What it looked like

The extension appeared to kill the whole browser on 17 of the lab corpus's 45 scenes - 38% - and on
exactly the material this product exists for: comic covers, halftoned newspaper pages, engraved
cartoons, photographed notices. The desktop app processed all 45 including a 17-megapixel newspaper
page, so it read as the extension's own defect.

Measured in [`DEV/research/ocrlab/2026-08-11__baseline.md`](../../research/ocrlab/2026-08-11__baseline.md) §5.
Run `temp/ocrlab/base06-ext`, Chrome 151.0.7922.109, tesseract.js 7.0.0.

## What it actually is

`Page.captureScreenshot` returns its PNG as base64 inside a single DevTools websocket message. Past
roughly four megabytes that message is dropped and the connection goes with it, close code 1006. The
runner asked for the clip rendered at the source image's natural size - `scale: naturalWidth / renderedWidth`,
often an upscale - so the reply's size tracked the source image's pixel count and its detail, and the
scenes that "crashed" were the ones whose PNG did not fit.

Measured, in this order, each step chosen because the previous one refuted something:

- **The browser survives.** Driving the worst scene for two minutes with the console, exceptions and
  target lifecycle all subscribed: the page answers, the browser answers, `child.exitCode` stays null,
  Windows logs no application error, no crash dump is written. The renderer never crashed either.
- **The call that kills the socket is `Page.captureScreenshot`**, in 583 ms, with the browser still
  answering afterwards on a fresh connection.
- **The ceiling is about 4 MB of base64 in one reply.** Walking a noise canvas: 4 121 988 bytes
  arrived, the next step up did not. Nothing was received at all on a failing call - the largest frame
  seen was about a kilobyte, so it is the reply, not a partial transfer.

## Three hypotheses tested and refuted along the way - do not re-run them

Six downscaled copies of `caricature-gillray-plumpudding-jpg` (11.3 Mpx) were run first: **1.0 and 1.5 Mpx
survived; 2.0, 2.5, 3.0 and 5.0 Mpx crashed** - while corpus scenes lived at 8.2 Mpx. Then five probes at
an identical 2.52 Mpx and varying detail: **flat, halftone and a resampled cartoon survived; noise and a
resampled engraving died.**

1. **"The 2x pre-OCR upscale blows the budget."** No. The 1.0 Mpx probe is upscaled to 4.7 Mpx before
   recognition and survives; the 2.0 Mpx probe is fed at native size and dies.
2. **"The grey rescue ladder's extra renditions blow it."** No. Both surviving probes returned a plate,
   so the ladder never ran; two corpus scenes that did run it to the end survived.
3. **"Cost scales with the number of connected components, not with area."** No. Counted after Otsu, the
   halftone probe has 70 200 components and survives while the resampled engraving has 7 083 and dies.
   The measure that does separate them is how large each one's PNG is.

Every one of the three is about the recognizer, and a fix drawn from any of them would have changed
shipped code to work around a defect that was not there.

## The fix

Both editions now capture at the viewport's own resolution and map into image space themselves - which
is what the Go runner already did in `CropToImage`, so this is the extension runner rejoining a
contract rather than a new rule. New [`extension/scripts/_ocrlab-image.mjs`](../../../extension/scripts/_ocrlab-image.mjs):

- `bandsFor` splits the capture into horizontal bands of at most `CaptureBandPx` = 1 500 000 pixels.
  Derived, not chosen: pure noise - the worst case a PNG can present - encodes to about 2.0 MB of base64
  per megapixel, so a band worst-cases to about 3.0 MB against a limit that took 4.12 MB. The extension
  needs bands and the Go runner does not, because the extension clips to the image, which can be
  arbitrarily tall, while the Go runner clips to the viewport. Recorded in `docs/PARITY.md` as an
  intentional difference.
- `assembleToNatural` stitches the bands and resamples to the source's pixel space. Band boundaries are
  computed from the whole rather than accumulated, so rounding cannot drift down the page and shift the
  bottom of the image against the source it is about to be compared with.

Guarded by [`extension/test/ocrlab-image.test.mjs`](../../../extension/test/ocrlab-image.test.mjs): bands
tile the region exactly, no band exceeds the ceiling, out-of-order bands land where they belong, and a
capture that caught nothing is refused instead of being measured.

**No shipped code changed.** `extension/src` is untouched by this ticket.

## What it cost the measurement

Both editions now measure a resampled render instead of one edition measuring a natively re-rastered
one, so the extension's pixel numbers move. Over the 28 scenes that produced a result in both runs:
mean residual ink 0.1170 -> 0.1185, mean cut-glyph ink unchanged at 0.0743, mean minimum contrast
91.7 -> 82.8 - softer edges, as resampling implies. Movements are in both directions and small, which
is what rules out a misalignment.

The concealment and contrast figures in the baseline's extension column are therefore superseded by
`temp/ocrlab/fix02`; recognition, grouping, position, damage and the geometry gates are not affected.
`DEV/ocrlab/thresholds.json` is unaffected - it was derived from the desktop run, which did not change.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[-]` | Never affected - the Go runner already captured at the device scale and resampled locally. |
| GUI (`doc-html-ui`) | `[-]` | Inherits the desktop pipeline. |
| MSIX Store app | `[-]` | Same. |
| Browser extension | `[-]` | **Never affected.** The defect was in `extension/scripts/ocrlab.mjs`, a developer tool that ships in nothing. |
| Website / docs | `[-]` | No user-visible limit exists, so nothing to document. |
| `docs/PARITY.md` | `[x]` | New "Screenshot space" row plus the reply-size rule and why the extension bands and the Go runner does not. |

## How it was verified

- `npm run ocrlab -- --split all` - **45 of 45 scenes produced a result, zero errored, zero browser
  relaunches** (`temp/ocrlab/fix02`). Was 28 of 45 with 17 errored.
- `go run ./tools/ocrlab gate temp/ocrlab/fix02` against `temp/ocrlab/base06-ext`: **no check moved from
  PASS to FAIL.** Every failing check was already failing on the baseline, on the hard gates strategic
  §6.1 names. `synth-caption-on-gradient`, the worst concealment scene in both runs, has essentially
  nothing concealed in either (0.877 -> 0.943 with no plates produced at all), so its movement is noise
  on a scene that already fails outright.
- `npm test` - 129 pass, 0 fail. `./scripts/lint.ps1` - passed.

## What this leaves open

The 17 scenes have never actually been measured before, and now they have been. Whether the overlay is
any *good* on them is a separate question this ticket does not answer - `samson-and-delilah-03-jpg` at
0.4583 residual ink and `ludwig-hohlwein-..-garmisch-partenkirchen-jpg` at 0.3897 are the two worst, and
they belong to Phase 07 of
[`2026-08-11_ocr-visual-fidelity-lab`](../2026-08-11_ocr-visual-fidelity-lab.md), which is where
concealment and grouping are dealt with.
