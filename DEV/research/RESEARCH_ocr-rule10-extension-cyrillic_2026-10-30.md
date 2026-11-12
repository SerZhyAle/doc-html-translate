# Rule 10 in the extension: what tesseract.js with the default `eng` does to the five Cyrillic images

2026-10-30. Phase B (R10) of [`DEV/plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md`](../plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md),
the extension half the owner chose on 2026-10-30 ("measure tesseract.js on the five Cyrillic images first"). Follows
[`RESEARCH_ocr-rule10-script-check_2026-10-21.md`](RESEARCH_ocr-rule10-script-check_2026-10-21.md), which measured the
desktop edition on the same images. No code, constant or threshold was changed.

## Question

The extension reads every picture with the language the reader picked in the popup, `eng` when they picked none
(`extension/src/ocr.js`, `options.ocrLang || "eng"`), and it runs no script check. The `OCR-OVERLAY` rule 10 exception for
the extension says a Cyrillic page then gets transliterated debris; nobody had measured whether it does, nor whether the
rule 12 discard record (the evidence route that needs no 10 MB OSD pack) separates such a page from an honest one in this
engine.

## Answer

- **Three of the five Cyrillic images get debris plates** under the default `eng`, one more than on the desktop: the
  Russian UI screenshot (6 plates), the 18y poster (1 plate) and the marriage-certificate scan (1 plate; the desktop drew
  none there). The other two draw nothing: the floor refuses everything read.
- **The discard record does not separate them.** The share of read letters the pipeline refused is 0.69, 0.94 and 0.96 on
  the three Cyrillic images with plates, and 0.00 to 0.97 (median 0.08) on the 37 other images that draw plates. Any cut
  that catches all three Cyrillic images refuses 3 honest ones; with n=3 positives there is no bracket to derive a
  threshold from (rule 13).
- **Android's chosen guard would catch two of three and refuse three honest images.** `OcrLanguageGuard`'s thresholds
  (at least 40 letters read, 0.75 or more of them refused; marked chosen, not derived) applied to these records refuse the
  18y poster and the certificate, miss the UI screenshot (0.69), and refuse `crowdstrike-outage` (0.94), the Diaz cartoon
  (0.82) and the Brandeis cartoon (0.97), each of which is honest English.
- **The exception's text holds as written.** The debris case it names is real in this engine and more frequent than on the
  desktop; no evidence route in reach of the extension closes it on this corpus.

## How it was measured

The shipped `recognize()` of `extension/src/ocr-overlay.js` - ordinary pass, grey rescue ladder, screen and grey sweeps,
the confidence floor and the translatability gate, `clusterLines` - run in Node 26.13 on tesseract.js 7.0.0 (the version
vendored into `extension/vendor/tesseract`) with the vendored `eng.traineddata` (tessdata_fast 4.0.0). Two module hooks
replace the browser-only imports and nothing else: the vendored ESM bundle by the Node build of the same tesseract.js, and
`ocr-lang.js` by a stub that names the vendored language folder. `createImageBitmap` and `OffscreenCanvas` are backed by
`@napi-rs/canvas` 1.0.10, whose `grayscale(1)` filter was checked to give the luminance. Two runs of the five images
gave identical counts.

"Kept" counts the letters of the plates drawn; "refused" counts the letters of the discard record's lines (every gate).

## The five Cyrillic images

| Image | Plates | Kept letters | Refused letters (confidence / translatable) | Share refused | Plate text (first plate) |
|---|---:|---:|---|---:|---|
| `image.png` (Russian UI) | **6** | 207 | 451 (445 / 6) | 0.69 | `Katanorm-nonyyarenn I Npocmorp I Bugeo ..` |
| `polish-soviet-propaganda-poster-18y` | **1** | 11 | 286 (285 / 1) | 0.96 | `CAEAATb TEBA` |
| Russian marriage-certificate scan | **1** | 13 | 206 (202 / 4) | 0.94 | `aka Hapoauaach «` |
| `a-propaganda-poster-from-the-soviet-union` | 0 | 0 | 108 (108 / 0) | 1.00 | - |
| `poster-display-type-on-flat-colour` | 0 | 0 | 81 (80 / 1) | 1.00 | - |

Against the desktop (2026-10-21): 5 / 1 / 0 / 0 / 0 plates. The desktop's script check catches the first two and leaves
the certificate at 2.92 under its floor; the desktop draws nothing on the certificate, the extension draws one debris plate.

## The other images

The 44 other lab scenes plus `test_doc/1.png`, `accounts.jpg` and `img-png_Nyoka-comic-page.png` (47 images, none
of them Cyrillic), same harness:

| Set | Images | Share refused |
|---|---:|---|
| Cyrillic under `eng`, with plates | 3 | 0.69, 0.94, 0.96 |
| everything else, with plates | 37 | 0.00 to 0.97, median 0.08 |
| everything else, no plates | 10 | - |

At or over 0.69: `louis-brandeis-nomination` 0.97 (1 plate), `crowdstrike-outage-at-asb` 0.94 (1 plate), `diaz-political-cartoon`
0.82 (8 plates). A cut at 0.95 catches one Cyrillic image (the 18y poster) and refuses one honest image; a cut at 0.69 catches
all three and refuses three.

## What this does not say

- Node, not a browser: the canvas, the JPEG decoder and the blur of the screen rung are Skia's through `@napi-rs/canvas`, not
  Chrome's. The recognizer, its data and every decision after it are the shipped code. A run in the real extension
  (`npm run ocrlab`) would differ at most in decoded pixels; it was not made.
- Five Cyrillic images, as in the desktop note; no Greek, Arabic, Hebrew, Han or Hangul page.
- The share is one candidate signature; kept confidence or letter run were not tried here either.

## Reproducing

`temp/E0121/ext-cyr/` (gitignored): `run.mjs` (the harness; `--rest` for the other images), `register.mjs` and `hooks.mjs`
(the two import swaps), `shim-tesseract.mjs`, `shim-lang.mjs`, `analyze.mjs`; raw records `ext-eng.jsonl`,
`ext-eng-2.jsonl` (the repeat) and `ext-eng-rest.jsonl`. Run from the PowerShell tool:
`node --import ./register.mjs run.mjs out.jsonl [--rest]`.
