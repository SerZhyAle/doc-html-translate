# Fix roadmap - the 2026-07-17 corpus sweep

The queue of open defects, in execution order. Built 2026-07-17 from the corpus sweep
([`test_doc/CORPUS.md`](../../test_doc/CORPUS.md),
[`DEV/research/format_verification_sweep.md`](../research/format_verification_sweep.md)), where every
finding below was reproduced against a real file and judged by reading the output, not by counting log
lines.

**How to read this.** `**Priority:** N` in each ticket is its position here - **lower runs first**.
Finished tickets move to [`done/`](done/). The status line inside each ticket is the truth; this file is
only the order.

## The queue

| P | Ticket | What a user sees today |
|---|---|---|
| ~~1~~ | ~~[cli-diagnostics-honesty](done/2026-07-17_cli-diagnostics-honesty.md)~~ | **Implemented 2026-07-17.** The log lied and every error path hung when piped |
| ~~2~~ | ~~[utf16-text-not-decoded](done/2026-07-17_utf16-text-not-decoded.md)~~ | **Implemented 2026-07-17.** A UTF-16 Cyrillic `.txt` came out as mojibake, both editions |
| ~~3~~ | ~~[txt-legacy-encoding-detection](done/2026-07-17_txt-legacy-encoding-detection.md)~~ | **Implemented 2026-07-17.** A CP866 / KOI8-R / CP1251 `.txt` came out as mojibake, both editions |
| ~~4~~ | ~~[binary-input-becomes-a-garbage-document](done/2026-07-17_binary-input-becomes-a-garbage-document.md)~~ | **Implemented 2026-07-17.** A `.docx` became 15 pages of garbage; 8 extensions, all 12 corpus binaries now refuse |
| ~~5~~ | ~~[tiff-input-produces-an-unrenderable-page](done/2026-07-17_tiff-input-produces-an-unrenderable-page.md)~~ | **Implemented 2026-07-17.** A TIFF was a broken-image icon in Chrome; now transcoded to PNG (Chrome-verified) |
| ~~6~~ | ~~[fb2-drops-every-image](done/2026-07-17_fb2-drops-every-image.md)~~ | **Implemented 2026-07-17.** An illustrated FB2 lost all 55 images; now 55/55 render in Chrome |
| ~~7~~ | ~~[html-input-drops-local-images](done/2026-07-17_html-input-drops-local-images.md)~~ | **Implemented 2026-07-17.** A saved web page lost all 28 images; now 28/28 render in Chrome |
| ~~8~~ | ~~[pdf-raster-extraction-takes-the-wrong-images](done/2026-07-17_pdf-raster-extraction-takes-the-wrong-images.md)~~ | **Implemented 2026-07-17.** A thumbnail shown as the page (now the real chart via embed); the same page twice (duplicate rasters collapsed) |
| ~~9~~ | ~~[ocr-upscale-threshold-misses-page-scans](2026-07-17_ocr-upscale-threshold-misses-page-scans.md)~~ | **Implemented 2026-07-17.** Gate keys on estimated DPI now, not pixels: declare resolution always (frees balloon separation), upscale only below a DPI floor; masked non-ASCII path bug closed too |
| ~~10~~ | ~~[ocr-plate-fit](2026-07-17_ocr-plate-fit.md)~~ | **Implemented 2026-07-17.** Runtime re-fit shrinks plate font to its box (and re-fits after the translator swaps text); Chrome-verified 0 clipped. Stays in `DEV/plan/` until the umbrella closes |
| ~~11~~ | ~~[ocr-pool-per-book](2026-07-17_ocr-pool-per-book.md)~~ | **Implemented 2026-07-17.** Batching moved from per-file to per-book (`OverlayBook`); `-multipage` OCR now runs at full pool width - measured 6.1x (67.2s -> 11.0s) on a 37-page comic |
| ~~12~~ | ~~[comic-archives](2026-07-17_comic-archives.md)~~ | **Implemented 2026-07-17.** All four open on desktop (`internal/comic`; CBR/CB7 via 7-Zip), CBZ/CBT in the extension (CBR/CB7 declined); forced OCR, natural page order, entry filter guarded. Manual browser smoke test still pending |
| ~~13~~ | ~~[output-typography](done/2026-07-17_output-typography.md)~~ | **Implemented 2026-07-18.** Swept ~42 `...`/long-dash violations in generated output (Go + extension `_locales`, en/ru/uk); the `<title>` em dash kept by decision; a source-scanning guard (`tests/typography_test.go`) stops re-drift |
| ~~14~~ | ~~[docs-site-reality-check](done/2026-07-17_docs-site-reality-check.md)~~ | **Verified 2026-07-18.** Read-together pass after P1-P13 landed: docs/site already correct and in sync (comics, TIFF, FB2/HTML images, extension CBZ/CBT-only) in en/ru/uk; fixed the P8 changelog row (was merged into P9) and documented legacy TXT encodings + binary refusal on README + the three-language docs |

### Wave 1 - fix the instrument (P1) - **done 2026-07-17**

Done first because everything after it is verified by reading the CLI's output, and the CLI's output
lied. What it actually cost, recorded because two of the three items were bigger than written:

- **The hang was four sites, not one.** Not "the flag package's error path" but every unconditional
  `Scanln` - and the worst was invisible to the sweep, because the sweep ran `-notranslate`:
  `internal/pipeline` pauses identically after a *translation failure*, hanging any scripted conversion
  whose translation broke.
- **Fixing the hang did not fix the reported command.** `-h` also exited 1 with the help on stderr, so
  `-h | Select-Object -First 60` returned empty. Fixed by owner decision: `-version` already exits 0 in
  the same binary, so `-h` was the outlier.
- **The page counter now diagnoses rather than merely not-lying**: `Pages: 6 (with text: 0,
  image-only: 6)` says "this is a scan" at a glance.
- **The overlay gap is explained where it happens.** The sweep spent most of its time on this one:
  Aphrodite reported `1711 image(s) overlaid` of 2304, and establishing that the 593 silent pages were
  **correct** (art panels with no dialogue) took per-image plate counts, file-size distributions, and
  finally opening the images. An identically-shaped gap on `Kupní smlouva ..pdf` was the whole bug. The
  same run now says `5 image(s) overlaid, 1 with no text found`, and a real failure is named with
  tesseract's own reason.

One process note worth keeping: the first harness built to prove the hang **reported a false pass on
the pre-fix binary**. A background job hands the child a closed stdin, so `Scanln` returns EOF and the
bug hides. Reproducing it needs a redirected stdout *and* a stdin pipe held open and never written.

### Wave 2 - intake: stop producing confident garbage (P2-P5)

The worst class in the sweep: the app accepts a file it cannot read, emits garbage, and exits 0. Measured
across eight extensions - `.docx` `.odt` `.pptx` `.xlsx` `.djvu` `.cbz` `.cbr` `.cb7` `.cbt` - all exit 0,
`.cbr` producing a 17 MB `index.html` of 14,301,891 characters at 18.4% control bytes, presented to the
reader as a 363-page book.

One root cause underneath P4 and P5: [`internal/pipeline/pipeline.go`](../../internal/pipeline/pipeline.go)
dispatches on the **file extension alone**, and everything unknown falls through to `internal/txt`.

**P2 + P3 done (2026-07-17).** `internal/txt.decodeText` (+ its JS twin) decides encoding from the
leading bytes: BOM, then valid-UTF-8, then a legacy Cyrillic code page by detection. UTF-16 (P2) was
mojibake in *both* editions (measured, not assumed); the UTF-8-BOM leak was Go-only, because
`TextDecoder` strips a BOM by itself. P3 filled the legacy seam with a two-metric detector - a
frequency-weighted fit picks the code page (cp1251 vs koi8-r yield the same *count* of Cyrillic letters;
only frequency separates them), a Cyrillic-fraction floor of 0.30 is the confidence gate (rejects French
Latin-1, which tops the weight score). `test_doc/Лицензионное Соглашение.txt` - which P3 had wrongly
claimed did not exist - is real cp1251 and now reads as Russian, logging `Decoded from windows-1251`.
The whole decode order, table, and floor are a `docs/PARITY.md` invariant.

**P4 done (2026-07-17).** `internal/txt.LooksBinary` guards the `default:` arm: BOM first (so UTF-16 is
not refused for its NULs), then named signatures (ZIP/RAR/7z/tar-at-257/DjVu, plus PDF/MOBI/image
defensively), then a NUL byte in the first 4 KB as the catch-all. All 12 corpus binaries now refuse with
a named format, a non-zero exit, and no leftover output dir; all five text fixtures still convert. The
extension was confirmed already-correct *by tracing the routing*, not assumed. Signature-first detection
is now a `docs/PARITY.md` invariant. The misleading `treating as plain text...` line (which P13 flagged
for its long dash + ellipsis) is gone as a side effect. Remaining wave-2 work: **P5, TIFF.**

One nuance left open in P4, deliberately: the sniff sits only on the `default:` arm, so a binary renamed
to `.txt` still reaches the text reader unguarded. That is a wider change than the measured problem and
is recorded in the ticket as a possible future sibling.

**P5 done (2026-07-17), wave 2 complete.** Owner chose **transcode over refuse**: the Go app has a TIFF
decoder, so `internal/img.extractTIFF` converts each frame to PNG (Chrome cannot show TIFF) and a
multi-page TIFF becomes one PNG page per frame. Multi-frame decode uses the fact that TIFF strip offsets
are absolute - walk the IFD chain for each frame's offset, copy the file with the header repointed at
frame N, let `tiff.Decode` read it - so the standard decoder handles every frame. Verified by headless
Chrome (`naturalWidth`): raw `.tif` -> `onerror`, transcoded PNG -> `W=800`; the 3-frame fixture yields
3 rendering pages with OCR plates in 3 separate containers, killing the old 33-plates-on-one-image
pile-up. Extension refuses TIFF (no browser decoder) - now an intentional divergence in `docs/PARITY.md`.

### Wave 3 - nothing is lost silently (P6-P8) - **done 2026-07-17**

P6 and P7 were the same defect wearing two hats: one format's reader drops every image and says nothing.

**P6 + P7 done (2026-07-17), verified in Chrome.** The "same machinery" framing turned out half-right -
the shape is shared (land images on disk, reference them, degrade a miss visibly) but the *readers*
differ enough that each got its own code:

- **FB2** was rewritten from struct-unmarshal (which captured only `<p>`) to a **token walk** that keeps
  content in document order and decodes every base64 `<binary>` to a sibling file. The catch the ticket
  missed: FB2 wraps most illustrations *inside* a `<p>` (`<p><image/></p>`), so the first pass found only
  the cover - pulling inline images out of each paragraph took Alice from 1 to **55/55 rendering**.
- **HTML** gained `copyLocalImages`: copy each referenced local image from the source's directory subtree
  into the output, rewrite the `src`, **refuse `../` traversal**, leave remote/`data:`/absolute and
  dangling refs alone. Alice went **0/28 -> 28/28 rendering**; the dangling fixture still converts and
  shows its missing images as honest broken icons.

Extension needed no change for either: it inlines FB2 `<binary>` as `data:` URLs already, and can't reach
an HTML file's sibling images from the picker (browser sandbox) - both recorded as intentional divergences
in `docs/PARITY.md`.

P8 (done) was upstream of wave 4 and had to precede it - see the constraints below.

### Wave 4 - OCR you can actually read (P9-P10)

Both are follow-ups to shipped work, not new features: the upscale mechanism landed
([`done/2026-07-01_ocr-pre-upscale`](done/2026-07-01_ocr-pre-upscale.md)) but its gate selects close to the
inverse of what it was meant for, and the plate clustering landed
([`done/2026-07-01_ocr-overlay-line-clustering`](done/2026-07-01_ocr-overlay-line-clustering.md)) but the
boxes do not fit the text poured into them.

Worth carrying into both: PSM 3 is **not** the lever. The sweep confirmed it twice - the same PSM that
merges balloons on newsprint gives near-perfect results on Aphrodite's clean 1600x1200 renders (4 balloons
-> 4 plates). Resolution and lettering quality decide the outcome, which is why P9 (what OCR sees) sits
above P10 (how the result is displayed), and both sit below P8 (which image OCR gets at all).

### Wave 5 - perf, then the new format (P11-P12)

P11 is narrower than it first looked and P12 is the payoff - the corpus already holds all four comic
fixtures.

### Wave 6 - close the loop on what we claim (P13-P14)

P13 came out of P1: sweeping for the `..`/long-dash rule turned up ~42 violations in user-visible
strings across 11 files, including `<title>%s — Page %d</title>` in five extractors - a long dash in
every page title of every converted book. It carries one product decision (does the rule mean the book
titles, or only the app's chatter?), which is why it is a ticket rather than a drive-by fix.

Every ticket above carries its own narrowly-scoped "Website / docs" row, but no single one of them owns
the *cumulative* result: twelve independent edits to the same handful of files (`README.md`, the
three-language site, `docs/PARITY.md`) need one final read-together pass to confirm they still tell one
coherent, accurate story once they've all landed. Deliberately last and blocked on P1-P12 closing - doing
it earlier would just get redone.

## Ordering constraints

These are not preferences; each is load-bearing and each is recorded in the tickets themselves.

- **P2 before P4 - satisfied.** P4's NUL sniff refuses any input with NUL bytes in the first 4 KB
  (measured: 10/10 binaries rejected, 6/6 texts accepted). UTF-16 text is full of NULs - 75 in a
  two-line ASCII file - so shipping the sniff first would turn UTF-16 from *mangled* into *refused*. P2
  landed the BOM check (`internal/txt.bomUTF16LE`/`bomUTF16BE`), so P4 can now reuse it and sniff after
  the BOM, not before.
- **P8 before P9 - satisfied.** Stated in P8: a `/Thumb` preview extracted as page content is 128x104,
  which is under P9's upscale gate and *would* be enlarged 2x - to 256x208, still unreadable. P8 now drops
  the thumbnail outright (it never reaches the gate), so P9 tunes the gate against real page scans only.
- **P4 before P12.** Both decide what happens to `.cbz` / `.cbr` / `.cb7` / `.cbt`. P4 refuses them, P12
  moves them to supported. Compatible only if P4's refusal is table-driven rather than hardcoded.
- **P1 before P6-P8.** The done criteria of P6, P7 and P8 each require a dangling ref or an unreadable page
  to "degrade visibly", and each points at P1 for what that means.
- **P2 and P3 are one implementation pass.** Two files, one function: how `internal/txt` picks an encoding.
  P3's own owner decision already makes the BOM authoritative, which is P2's fix.
- **P11 is *not* a precondition for P12.** It reads like one and an earlier draft said so; the measurement
  says otherwise. See below.

## Two claims this sort corrected

Both are the same error I already made once on Aphrodite - reasoning from a ticket instead of from a
measurement - so they are recorded rather than quietly fixed.

1. **P9 was numbered above P8, contradicting its own ticket.** The old numbers (upscale 20, raster 30) put
   the upscale gate first, while P8's text says plainly that raster selection sits upstream of it. The
   prose was right and the numbers were wrong; they now agree.
2. **P12 carried an "accepted risk" that does not exist on the default path.** It recorded that a 200-page
   comic would OCR serially - "roughly 200 s instead of ~15 s". But single-page mode is the **default**
   (`SinglePage: !*multiPage`), and there the whole book is one content file, so the pool sees every image
   at once. Aphrodite settles it empirically because it *is* structurally a comic - 2304 pages, no text
   layer, forced OCR, one image per page - and it runs at ~40 images/s. A 200-page comic costs about 5 s.
   The risk applies to `-multipage` only, which is why P11 is a perf ticket rather than a blocker, and why
   the owner's "comics first, pool later" split is both correct and free.

## The extension is the reference, not the follower

Three of the wave-2 defects (P4, P5, and format routing generally) have the same shape: the Go app checks
the file extension, the extension checks the **bytes**, and the extension is right. Its `detectFormat`
([`extension/src/viewer.js`](../../extension/src/viewer.js)) tests `%PDF`, `PK..`, `{\rtf` and `BOOKMOBI`
before it ever consults a filename, and it declines TIFF outright - which is correct, because Chrome cannot
decode TIFF (proven headless: PNG/JPEG/GIF/BMP/WebP all render, `.tif` fires `onerror`).

So close this parity by pulling Go **up to** the extension. Port the byte-signature check; do not design it
twice.

**One exception, so this is not over-applied:** CBZ. Both EPUB and CBZ are `PK\x03\x04`, and the extension
returns `"epub"` for any ZIP before the filename is consulted - so it hands a comic to the EPUB reader and
fails with a misleading "not an EPUB". On comics the extension is wrong too, and P12 has to disambiguate on
both sides without regressing the EPUB hot path.

## Not in the queue

| Ticket | Status | Why it is not queued |
|---|---|---|
| [app-ocr-image-overlay](2026-07-01_app-ocr-image-overlay.md) | `Partial` (P20) | The umbrella the OCR work hangs from. Closes when P9 + P10 land and its GUI / translation criteria are checked by hand. |
| [cross-edition-parity](2026-07-01_cross-edition-parity.md) | `In Progress` (P90) | A living backlog referenced from `CLAUDE.md`, not a task. Closes items as other tickets land. |

Two extension tickets sat at `BlockNeedUserTest` since 2026-07-01 and were **closed to `done/` on
2026-07-17 by owner decision** - [`ocr-image-overlay`](done/2026-07-01_ocr-image-overlay.md) and
[`extension-format-parity`](done/2026-07-01_extension-format-parity.md). Both had landed their code, docs
and `npm test`, and both have been live in the Chrome Web Store and Edge Add-ons for weeks across several
releases. The formal browser checklist was never run as a checklist; production use was accepted in its
place. That is recorded in each ticket as what it is, so nobody later reads it as a ticked-off pass.

## What the sweep found working

Recorded so the roadmap is not read as "everything is broken". EPUB (48 MB Treasure Island in 0.1 s, 262
anchors, 0 broken refs, 0 control chars), MOBI, AZW3, RTF, five image formats, the `.tiff` four-letter
spelling, leftover-output reuse (announced honestly, not silently), and OCR itself on decent input -
Aphrodite is the corpus's best evidence that the engine works when it is fed a clean image.
