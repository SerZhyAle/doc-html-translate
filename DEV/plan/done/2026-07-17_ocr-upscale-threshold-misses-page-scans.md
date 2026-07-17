# Pre-OCR upscale never fires for real page scans

**Status:** Implemented 2026-07-17 - gate now keys on **estimated DPI**, not pixel count: always declare
the resolution to tesseract (frees balloon separation) and upscale `2 x` only below a ~120-DPI floor
(so a ~150-DPI scan is not over-segmented and does not pay 4x). Also closed the masked non-ASCII path
bug it flagged (`prepareForOCR` ASCII-stages the path). Measured on the corpus with the real tesseract,
not guessed. Both editions + parity guard + docs updated.
**Priority:** 9
**Date:** 2026-07-17

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.
> Follow-up to [`2026-07-01_ocr-pre-upscale`](done/2026-07-01_ocr-pre-upscale.md) (**Implemented**) - the
> mechanism landed and works; the threshold that gates it does not select the images that need it.

## What / why

Low-resolution scanned pages come out of OCR as unreadable word salad, when the exact same page
upscaled twice over reads cleanly. The pre-OCR upscale that exists to prevent this is gated on the
image's **long side being under 1000 px** - but a scanned page is over 1000 px tall even at a poor
~100 DPI, so the gate rejects essentially every page scan. It only ever fires for thumbnails and
small in-line figures, which is close to the inverse of the intent.

Measured on `test_doc/comic-scan-tiny_First-Earthman-on-Mars-1944.pdf` (6 pages, ~100 DPI). Four of
its six pages are 1001 px tall and miss the gate **by one pixel**; the two that happen to be 971 and
911 px pass it. The two that pass are the only two whose plates are readable. Same page, same
tesseract, same language, upscale the only variable:

| | Recognized text |
|---|---|
| As shipped (gate rejects) | `fad, spreading i thinly, for he ould not have had much left titer his ight through the tack less void, set it aflame by using a ane Fa` |
| Upscaled 2x (gate blocks this) | `a great area of reddish sand, spreading it thinly, for he could not have had much left after his flight through the track- less void, and then set it aflame by using a long fuse.` |

The second is essentially correct against the page. The first is unusable - and worse than useless
downstream, because a translator will happily render the salad into confident nonsense.

For reference, archive.org's own OCR layer for the same book (`comic-textlayer-tiny_..pdf`, produced
from the pre-downsample scans) lands much closer to the upscaled result. We are not hitting a limit
of Tesseract or of the source; we are declining to use the fix we already wrote.

### It also merges and loses speech balloons

The gate is not only about legibility of prose. On a real comic page
(`test_doc/comic-scan-mid_Plastic-Man-v1-017.pdf` p.14, 1200x1674, ~182 DPI, ~12 balloons) the
shipped path produces **5 plates for 12 balloons**, several of them a line-by-line interleave of two
or three *different* balloons that happen to sit side by side, and most balloons missing entirely.
This is the owner-reported "plate missed the bubble" (shape 2 in
[`2026-07-17_ocr-plate-fit`](2026-07-17_ocr-plate-fit.md)): the plate's box is the bounding box of
several balloons at once.

The merging happens **inside Tesseract**, at block level, before any of our clustering runs - so it
is not a clustering bug. Resolution is what drives it. Measured on that page:

| Image | PSM | Words | Blocks | Mean conf | Words under conf 50 |
|---|---|---|---|---|---|
| as-is | 3 (shipped) | 121 | 10 | 68.6 | 38 |
| as-is | 11 | 286 | 163 | 59.7 | 123 |
| as-is | 12 | 174 | 108 | 57.6 | 79 |
| **2x** | **3** | **158** | **14** | **81.6** | **17** |
| 2x | 11 | 344 | 207 | 57.1 | 160 |
| 2x | 12 | 334 | 202 | 58.4 | 147 |

At 2x, PSM 3's own layout analysis separates the balloons on its own: blocks go 10 -> 14 against ~12
real balloons, mean confidence 68.6 -> 81.6, and low-confidence words halve.

**PSM is not the lever** - a hypothesis worth recording as refuted, since it is the obvious suspect.
The sparse modes (11/12) find more words but shatter the page into 163-207 blocks against ~12
balloons and drop mean confidence, i.e. they would trade merged plates for a swarm of them. PSM 3 is
the right choice and gets *better* with resolution; leave
[`2026-07-02_ocr-psm-parity`](done/2026-07-02_ocr-psm-parity.md) alone.

The gate should key on something that tracks **text legibility** rather than raw pixel count.
Tesseract's own guidance is ~300 DPI; a page whose long side is ~1000 px is nowhere near it. Options
worth weighing in the tactical pass: estimate DPI from the image against an assumed page size and
upscale below a DPI floor; gate on the **short** side; or gate on measured median line height from a
cheap first pass (the value the plate font-size already derives). Whatever is picked must stay a
single shared rule - this is a parity invariant, and the extension has the identical gate.

Note the interaction with [`2026-07-17_comic-archives`](2026-07-17_comic-archives.md): comic archives
are pure page scans, so every page there flows through this gate. Landing comics on top of the
current threshold would ship the salad as the format's first impression.

### There is a second, nearly free lever: tell Tesseract the resolution

Upscaling 2x costs 4x the pixels per page, which lands straight on the OCR wall-clock. Tesseract also
accepts a **declared DPI** and uses it for layout decisions, at no pixel cost. Both levers were
measured on the same balloon page (`Plastic-Man-v1-017` p.14, 1200x1674, physically ~182 DPI):

| Case | Words | Blocks | Mean conf | Words under conf 50 |
|---|---|---|---|---|
| as-is, no DPI declared (shipped) | 121 | 10 | 68.6 | 38 |
| as-is, `--dpi 300` (== `-c user_defined_dpi=300`) | 155 | 13 | 75.4 | 29 |
| as-is, `--dpi 70` | 121 | 10 | 68.6 | 38 |
| **2x, no DPI** | **158** | **14** | **81.6** | **17** |
| 2x, `--dpi 300` | 189 | 16 | 78.6 | 32 |

Three things fall out, and they should shape the tactical pass rather than be re-derived:

- **Declaring 300 DPI alone buys most of the word/block gain for free** - 121 -> 155 words, 10 -> 13
  blocks against ~12 real balloons.
- **`--dpi 70` reproduces the shipped baseline exactly**, which means Tesseract is already estimating
  this page at ~70 DPI when it is physically ~182. Its own estimate is less than half of reality, and
  that estimate is what drives the layout analysis that merges the balloons.
- **The two levers do not compose.** 2x + `--dpi 300` finds more words but drops mean confidence
  (81.6 -> 78.6) and doubles the low-confidence tail (17 -> 32): after a 2x upscale the true
  resolution is ~365 DPI for this page, so declaring 300 is a smaller lie but still a lie. Whatever
  is picked, the declared value must track the *post-upscale* image, not be pasted on top.

Upscaling still wins on confidence (17 low-confidence words vs 29) and is the stronger lever; DPI is
the cheaper one. Which to ship - or what blend - is an owner call about quality against OCR time, and
wants the numbers above plus a wall-clock measurement on a real book, not a guess.

### It is currently masking the non-ASCII path bug

`upscaleForOCR` writes its enlarged copy to `os.CreateTemp("", "docht-ocr-*.png")` - an **ASCII**
path. So an upscaled page hands Tesseract a clean filename, while a non-upscaled page hands it the
original, which the open non-ANSI-codepage bug mangles (see `test_doc/CORPUS.md`).

Measured on `комикс-скан_První-pozemšťan-na-Marsu.pdf`, a byte-identical copy of the tiny comic under
a name that mangles: **2 of 6 images overlaid**, against 5 of 6 for the ASCII-named original. The two
that survived are exactly the two the gate happens to upscale (971 px and 911 px); the four at 1001 px
went in with the mangled path and failed silently.

This inverts the obvious reading of the fix. Widening the gate so page scans get upscaled would route
every page through an ASCII temp path and make the non-ASCII failure **disappear from view without
being fixed** - dormant until someone tunes the threshold again, and still live for any image the new
rule leaves un-upscaled. `internal/ocr` needs its own staging regardless, the way `internal/pdf`
already does for `pdftotext` via `stagePDFForPDFToText`. Do not let this ticket close that one.

## Edition parity checklist

For each edition: **Done**, or **Declined** with a one-line rationale (record lasting declines in
`docs/PARITY.md` under "Intentional divergences"). "Not applicable" needs a reason too.

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `prepareForOCR` / `estimateDPI` replace the `upscaleForOCR` pixel gate; ASCII staging added |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no new flag, confirmed by the corpus runs |
| MSIX Store app | `[x]` | inherits the GUI; upscale + ASCII stage both write to `os.TempDir` (`docht-ocr-*`), not the install dir |
| Browser extension | `[x]` | `upscaleForOcr` / `estimateDpi` ported; DPI declared via `worker.setParameters({user_defined_dpi})`; parity guard green |
| Website / docs | Declined | OCR fidelity is not claimed with numbers on the landing page; nothing to correct |

## Shared invariants touched

- **Pre-OCR upscale** (`docs/PARITY.md`): `OCR_UPSCALE_BELOW = 1000` / `OCR_UPSCALE_FACTOR = 2`.
  Both the rule and possibly the factor change here; the invariant table and the parity test that
  guards the pair must move with it.
- Cost is not free: upscaling every page scan 2x quadruples the pixels Tesseract chews per page.
  Measure against [`2026-07-17_ocr-pool-per-book`](2026-07-17_ocr-pool-per-book.md) before choosing a
  factor - a correct-but-slow OCR is a different complaint, not a fix.

## Cross-references

- Go: [`internal/ocr/tesseract.go`](../../internal/ocr/tesseract.go) - `upscaleForOCR`, `scaleDown`,
  `ocrUpscaleBelow`, `ocrUpscaleFactor`.
- JS: `extension/src/ocr-overlay.js` - `upscaleForOcr`, `OCR_UPSCALE_BELOW`, `OCR_UPSCALE_FACTOR`.
- Parity test guarding the constant pair: `tests/parity_test.go`.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] `docs/PARITY.md` updated - the "Pre-OCR resolution handling" row and the constants list now describe
      the DPI gate + declared-DPI rule (`OCR_UPSCALE_DPI_FLOOR` / `OCR_ASSUMED_PAGE_INCHES` /
      `OCR_MIN_DECLARED_DPI`), guarded by `TestParityOCRClustering`.
- [x] The 1001-px pages of `comic-scan-tiny` now recognize at the 2x quality: a formerly-salad page reads
      as clean prose (*"a great area of reddish sand.. his flight through the track-less void.."*), conf
      76.8 -> 91.7. Measured with the real tesseract on the extracted page images.
- [x] Per-page OCR wall-clock recorded: the tiny scan upscales (6 pages ~1.4s -> 4.5s), the mid-DPI
      Plastic-Man does **not** (37 images, 8.7/s, unchanged) - the cost lands only where it pays.
- [x] `scripts/test.ps1` green (Go incl. the 226s integration suite), `npm test` 75/75.
- [x] Changelog entry in `DEV/CHANGELOG.md` (2026-07-17 23:05).

**Note on the mechanism, recorded because it refines the ticket's framing:** the ticket floated three
gate options (DPI floor, short side, median line height from a first pass). The measurement settled it -
declaring DPI is a free win on **every** scan (it fixes the balloon-merge the ticket attributed to
resolution), and the 2x upscale helps only genuinely low-res pages, so the gate is "declare always,
upscale below a DPI floor". A confidence-from-a-first-pass gate was **refuted** by the data: native mean
confidence does not separate the pages that need upscaling from those that do not (a fine page at conf
84 sits below a needs-upscale page at conf 85). The non-ASCII staging (`stageASCIIPath`) is its own
concern the ticket insisted not be masked - done, not hidden behind the wider gate.
