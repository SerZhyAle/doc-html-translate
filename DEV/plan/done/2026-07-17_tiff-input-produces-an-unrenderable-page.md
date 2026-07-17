# TIFF input produces a page Chrome cannot render, and multi-page TIFF piles every page onto it

**Status:** Implemented - 2026-07-17; TIFF transcoded to PNG, verified rendering in Chrome
**Priority:** 5
**Date:** 2026-07-17

> **Owner decision (2026-07-17): transcode, don't refuse.** The Go app has a TIFF decoder, so it can
> honestly support the format rather than drop it. `internal/img.extractTIFF` decodes every frame and
> writes each as a PNG; a multi-page TIFF becomes **one PNG page per frame**.
>
> **The multi-frame decode, since `x/image/tiff` only decodes frame 0:** a TIFF's strip offsets are
> absolute from the file start, so `tiffFrameOffsets` walks the IFD chain for every frame's offset, and
> `decodeTIFFFrame` copies the file with the header's first-IFD offset repointed at frame N - `tiff.Decode`
> then returns frame N with its strip offsets still valid. This reuses the standard decoder for every
> frame instead of reimplementing TIFF.
>
> **Verified by Chrome itself** (the done criterion, not a plate count): headless Chrome, `naturalWidth`.
> The raw `.tif` returns `onerror` (the bug, reproduced); each transcoded PNG returns `W=800`. The 3-frame
> fixture produces 3 PNG pages, all three rendering, and OCR now reports `3 image(s) overlaid` with plates
> in 3 separate `ocr-fig` containers - the old pile-up (33 plates on one unshowable image, `1 overlaid`)
> is gone. Both halves closed by the one change, as the ticket predicted.
>
> **Parity is now an intentional divergence** (recorded in `docs/PARITY.md`): Go transcodes TIFF, the
> extension refuses it, because a browser tab has no TIFF decoder and the Go app does.

> Cross-edition ticket. As with
> [`2026-07-17_binary-input-becomes-a-garbage-document`](2026-07-17_binary-input-becomes-a-garbage-document.md),
> the **extension is already correct** and the Go app is the outlier.

## What / why

[`internal/img`](../../../internal/img) accepts `.tif` and `.tiff`, copies the source **untouched** into
the output, and points the page at it:

```html
<img src="img-tif_Nyoka-comic-page.tif" alt="img-tif_Nyoka-comic-page"/>
```

**Chrome cannot decode TIFF.** The whole premise of this app is "convert, open `index.html` in Chrome,
use Chrome's built-in translation" - so a TIFF input produces a page that is broken in the only place
it is meant to be opened. Asked directly (headless Chrome, `new Image()`, `naturalWidth`, 2026-07-17,
same 800x1091 page in six containers):

| Container | Chrome |
|---|---|
| `.png` `.jpeg` `.gif` `.bmp` `.webp` | render, `naturalWidth=800` |
| **`.tif`** | **`onerror` - refused to decode** |

TIFF is the only entry in `img.exts` the target browser cannot show.

Nothing in the run hints at it. Tesseract reads TIFF perfectly well, so the log is a clean success:

```
[1/4] Preparing image...
  Image: img-tiff-spelling_Nyoka.tiff
  OCR overlay: 1 image(s) overlaid
Done.
```

Exit 0, 8 OCR plates, 1,143 characters of correct text. Every mechanical signal says it worked. What
a human opening the file actually sees is a broken-image icon with eight text plates floating over
white space. This is the exact shape the corpus was built to catch: "silently emits nonsense" and
"says it cannot read this" are different products - and here it is a third thing, "reports success and
shows nothing".

### Multi-page TIFF is worse than losing pages

`Extract` writes exactly one `page_001.html` for one `<img>`, so a multi-page TIFF was expected to lose
pages silently. It does not - it does something messier. Measured on a purpose-built 3-frame TIFF
(same page x3, `GetFrameCount` = 3) against the 1-frame original:

| Input | OCR plates | Log |
|---|---|---|
| 1-frame `.tif` | 8 | `OCR overlay: 1 image(s) overlaid` |
| **3-frame `.tiff`** | **33** | `OCR overlay: 1 image(s) overlaid` |

Tesseract OCRs **every frame** and returns all of it; the overlay then stacks all three pages' plates
onto the single `<img>`, positioned in the first frame's coordinate space. So the text is not lost -
it is piled up, three deep, on one image the browser will not draw anyway. The log reports one image
either way and says nothing about frames 2 and 3.

## The fix probably closes both halves at once

Transcoding TIFF to PNG on the way in (`golang.org/x/image/tiff` decodes; `image/png` encodes) makes
the page renderable **and** gives multi-page TIFF a natural shape: one frame -> one page -> one
`<img>` -> one OCR call, which is what the rest of the pipeline already expects. Worth confirming in
the tactical pass rather than assuming: `internal/img.Extract` currently returns a one-spine book, so
multi-frame means returning N spine items instead.

Note `.bmp` is a related-but-separate case, and already known: Chrome renders it, but the overlay's
colour sampling has no Go decoder for BMP, so its plates fall back to white (0 of 13 sampled on the
BMP fixture). Different defect, do not merge the two.

## The extension already refuses TIFF

`extension/src/viewer.js` - `imageMime` recognizes PNG, JPEG, GIF, BMP and WebP by signature, and its
extension fallback table is `{ png, jpg, jpeg, gif, bmp, webp }`. No TIFF, in either half. A `.tif`
therefore falls through `detectFormat` to `"unknown"` -> the PDF path -> a clear "cannot read this".
That is the right answer for a browser that cannot display the format, and it is the second place the
Go app's extension-only trust diverges from the extension's byte-signature routing.

Decide deliberately which way parity closes: transcode on the Go side (the app has a real decoder
available and can honestly support TIFF), or drop `.tif`/`.tiff` from `img.exts` and refuse. Either is
defensible; shipping "accept and show nothing" is not. If the Go app gains TIFF support the extension
cannot match, that is an **intentional divergence** and belongs in `docs/PARITY.md`.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` Done | `internal/img.extractTIFF` transcodes every frame to PNG; one page per frame |
| GUI (`doc-html-ui`) | `[x]` Done | inherits; the file-picker's TIFF entry is now correct because TIFF genuinely works |
| MSIX Store app | `[x]` Done | inherits the GUI; TIFF stays in the accepted set, no association change |
| Browser extension | `[x]` Declined (intentional divergence) | refuses TIFF - no browser decoder; recorded in `docs/PARITY.md` |
| Website / docs | `[x]` Done | the supported-image list already includes TIFF and is now accurate (it works, transcoded) |

## Cross-references

- Go: [`internal/img/extract.go`](../../../internal/img/extract.go) - `exts`, `IsImage`, `Extract`,
  `copyFile`, `buildPageHTML`.
- JS: `extension/src/viewer.js` - `imageMime`, `detectFormat`.
- Fixtures: `test_doc/img-tif_Nyoka-comic-page.tif`. The `.tiff` spelling and the multi-frame file are
  built by the repro, not stored - see `test_doc/CORPUS.md`.

## Done criteria

- [x] A `.tif` / `.tiff` input renders in Chrome - transcoded to PNG (`W=800` in headless Chrome, vs
      `onerror` for the raw `.tif`).
- [x] Verified by opening the output in Chrome and looking at it, not by the plate count.
- [x] A multi-page TIFF produces one page per frame (3 frames -> 3 PNG pages, 3 `ocr-fig` containers);
      plates from frame N land on frame N, not on frame 1.
- [x] Every edition row is `Done` or `Declined (reason)`; `docs/PARITY.md` records the Go-transcodes /
      extension-refuses divergence.
- [x] Tests / gate green: `internal/img` (unit, incl. the real 3-frame fixture) + `tests` (integration,
      187s) + lint.
- [x] Changelog entry in `DEV/CHANGELOG.md`.

## Note for a possible follow-up

`.bmp` is the remaining "Chrome shows it but the overlay can't colour-sample it" case (no Go BMP decoder
for `blockColors`, plates fall back to white). Different defect, unchanged here - flagged in the original
ticket body above, worth its own small ticket if it bothers anyone.
