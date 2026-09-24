# A scanned PDF page comes out as a smear, because the layer we keep is not the page

**Status:** Implemented
**Priority:** 1
**Date:** 2026-08-15

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

Found in the 21-document sweep ([`DEV/research/ocr_sweep_2026-08-13.md`](../../research/ocr_sweep_2026-08-13.md),
"Found on the way") and carried as an unfiled package-1 item in
[`RELEASE_QUEUE.md`](../RELEASE_QUEUE.md) §1.6. `pdf-1page-blackletter_Plague-Proclamation-1625`
extracts a pink-and-brown raster with the lettering smeared into vertical streaks. OCR then
correctly finds nothing on it and the log blames the language data, which sent the first look at
this in the wrong direction. The same bytes come out **without** `-ocr` (sha256 identical), so the
overlay was never involved.

**It is not a corrupt file and not a broken decoder.** The PDF is a mixed raster content (MRC) scan,
which is what library and archive scanners produce: the page is stored as a low-resolution
*background* layer plus a high-resolution *foreground* layer painted through a stencil `/Mask`.
Measured with `pdfcpu`'s own image list plus the raw XObject dictionaries:

| Obj | Id | Size | Filter | Colour space | `/Mask` |
|---|---|---|---|---|---|
| 36 | `I1` | 1455x2065, 39 KB | JPXDecode | DeviceRGB | no |
| 37 | `Im001` | 4363x6193, 91 KB | JPXDecode | DeviceRGB | **yes (38 0 R)** |

`selectPageImages` collapses a same-shape duplicate group down to **the largest** raster, so it kept
`Im001`. Painted whole, without the stencil that selects the ink, that layer is undefined nearly
everywhere - which is exactly the smear. 91 KB for 27 megapixels is the other half of the tell.
Decoding `I1` instead gives the readable page: the crown block, the blackletter body, the fold
lines, correct colour.

Two things were ruled out along the way, and both are worth keeping because both looked likely:

- **ffmpeg is not at fault.** The extraction path converts a `.jpx` with ImageMagick if it is
  installed and ffmpeg otherwise; on this machine only ffmpeg is present, and ffmpeg's *native*
  JPEG 2000 decoder is the usual suspect for a smear like this. It decodes `I1` correctly and
  reports `0 decode errors` on `Im001` - it is faithfully rendering a layer that means nothing
  on its own.
- **The duplicate-collapse rule is not wrong in general.** Bigger is right for the case it was
  written for (the same picture embedded twice at two resolutions). It is wrong only when the
  bigger one is not a whole picture.

## How wide the class is

Measured over every PDF in `test_doc/` - **1 file of 21, 1 image XObject of 2 560** carries a
`/Mask`. So this is a narrow class today. It is fixed as a **rule** rather than as a special case
because the rule is one comparison and the input class (a scanned page from a library) is one this
product is squarely aimed at.

## What changed

[`internal/pdf/extract.go`](../../../internal/pdf/extract.go):

- `betterPageRaster` replaces the bare size comparison inside `selectPageImages`: within a
  same-shape duplicate group, a raster carrying `/Mask` loses to one without it however big it is;
  otherwise the larger picture still wins.
- `writePDFImages` carries `HasImgMask` / `HasSMask` across from pdfcpu's stub pass. Only the stub
  pass reads the dictionary's mask entries - the real extraction leaves them false - which is the
  same reason `Width`/`Height` were already copied there.
- **`/SMask` deliberately does not demote.** Soft-masked transparency leaves the base image a whole
  picture and is the ordinary shape of a PNG-with-alpha illustration. And the preference applies
  only *inside* a duplicate group, so a lone masked illustration - which has no unmasked twin to
  fall back to - is still kept and still reaches the reader.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/pdf/extract.go` |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no surface |
| MSIX Store app | `[x]` | inherits the GUI |
| Browser extension | `[x]` | **Not applicable, not declined.** The extension does not extract PDF rasters: it renders pages through pdf.js, which composites the MRC layers the way a viewer does. There is nothing to port and no divergence to record |
| Website / docs | `[x]` | no user-facing string changes; the rule is documented in the code |

## Shared invariants touched

None. Raster selection is desktop-only (see the parity row above), so `docs/PARITY.md` is unchanged.

## Done criteria

- [x] The cause is attributed with evidence rather than guessed - the XObject dictionaries are in
      the table above, and both plausible wrong answers (ffmpeg, the collapse rule) are ruled out
      with a measurement.
- [x] How wide the class is, is measured and stated: 1 of 21 files, 1 of 2 560 image XObjects.
- [x] `pdf-1page-blackletter_Plague-Proclamation-1625` converts to the readable page, verified by
      looking at the extracted raster and not only at a byte count.
- [x] `TestSelectPageImages` grows three cases: the masked foreground loses to the unmasked page,
      the outcome does not depend on which raster is seen first, and neither an `/SMask` nor a lone
      masked image is demoted.
- [x] `./scripts/test.ps1`, `./scripts/lint.ps1` green; `DEV/CHANGELOG.md` entry.
