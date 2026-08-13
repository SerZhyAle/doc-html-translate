# A poster the app reads as one word - reproduction sweep, 2026-08-12

**Ticket:** [`2026-08-12_ocr-misses-display-lettering-on-saturated-art`](../plan/2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Question:** a user-reported standalone poster produces a single plate holding one word. Which part
of the shipped path loses the other ten, and does anything the code already contains recover them?
**Scope:** one image. This is a **reproduction**, not a corpus measurement - nothing here chooses a
threshold, and no number below may be used to justify landing a change on its own. The tuning
evidence is the corpus run the ticket requires.

## Material

A 960x1280 JPEG of a Soviet-style poster: eleven words of Cyrillic display type, red and black over
flat cream, with a two-figure illustration occupying the right half. Estimated DPI 116 by
`estimateDPI`, which is under `ocrUpscaleDPIFloor` - so the app stages it at 2x and declares 232 DPI.
The sweep reproduces that staging rather than feeding the original.

Ground truth, read from the image: `ЗАЧЕМ ТРАХАТЬСЯ: МЫ ЖЕ ВЗРОСЛЫЕ ЛЮДИ, МОЖЕМ ПРОСТО ОБ ЭТОМ
ПОГОВОРИТЬ` - 11 words.

## What the app does today

```
OCR overlay: 1 image(s) overlaid
```

One plate, transcript `| МОЖЕМ`. The log is not wrong and that is the problem: the run reports a
success, and the reader gets a stray box.

## Sweep

`tesseract 5.4.0.20240606`, `-l rus` from the app's own tessdata, `tessedit_create_tsv=1`,
`user_defined_dpi=232`, the 2x staged image unless noted. Metric: words at the app's two line-gates,
`ocrMinLineConf` = 50 and `ocrRescueLineConf` = 80.

| Configuration | words at 50 | words at 80 |
|---|---:|---:|
| colour + PSM 3 - **ships today** | 0 | 0 |
| colour + PSM 4 | 6 | - |
| colour + PSM 6 | 1 | - |
| colour + PSM 11 (sparse) | 16 | 6 |
| colour + PSM 12 (sparse + OSD) | 15 | - |
| grey + PSM 3 - **rescue rung 1** | 10 | 5 |
| **grey + PSM 11** | **10** | 6 |
| colour + PSM 11, `thresholding_method=1` | 37 | - |
| colour + PSM 11, `thresholding_method=2` | 26 | - |
| colour + PSM 3, `thresholding_method=1` | 0 | - |
| colour + PSM 11, 1x | 25 | - |
| colour + PSM 11, 3x | 22 | - |

Transcripts matter more than counts here, because two configurations reach 10 words and only one of
them is usable:

- **grey + PSM 11:** `ЗАЧЕМ ТРАХАТЬСЯ: МЫ ЖЕ ВЗРОСЛЫЕ ЛЮДИ, МОЖЕМ ПРОСТО ОБ ПОГОВОРИТЬ` - 10 of the
  11 words, in reading order, no debris. Only `ЭТОМ` is missing.
- **grey + PSM 3:** ``ЗАЧЕМ ВЗРОСЛЫЕ ‹ ЛЮДИ, ` | МОЖЕМ ПРОСТО ОБ ЭТОМ`` - four real words fewer, and
  three pieces of punctuation debris (a guillemet, a backtick and a pipe) counted among the ten.
- **The high-count rows are noise, not recall.** `thresholding_method=1` scores 37 words because it
  counts `2%`, `и:`, `2$`, `+;`, `№` and `у` alongside the real ones, and it splits `ТРАХАТЬСЯ` into
  two. A stricter gate would be needed to use it at all.

Per-word confidence for the candidate, at the rescue floor: `МЫ`(96) `ЖЕ`(96) `ЛЮДИ,`(91)
`МОЖЕМ`(94) `ПРОСТО`(90) `ПОГОВОРИТЬ`(95). These sit in the band the halftone cycle recorded for
genuine lettering (93-97), not in the hallucination band.

## Findings

1. **The grey rendition is the lever, and it is already written.** `greyRescue` was built for exactly
   this material - per-channel Otsu disagreeing over saturated colour - and it takes the image from 0
   confident words to 10. It never ran, because the ladder fires on `len(res.Blocks) == 0` and the
   ordinary pass produced one junk plate. One weak plate suppresses every rescue the app has.
2. **Segmentation is the second lever, worth about three lines of text.** On the grey rendition, PSM
   11 recovers `ТРАХАТЬСЯ`, `МЫ ЖЕ` and `ПОГОВОРИТЬ`, which PSM 3 loses. This does not contradict the
   wave-4 roadmap note that "PSM 3 is not the lever" - that was measured on comic balloons and clean
   page renders, where layout analysis has a layout to analyse. A poster has none.
3. **Resolution is not a lever here.** 1x, 2x and 3x all land in the same band; the staging is not
   what loses the text.
4. **Neither lever is corpus-proven.** Both sweeps that shipped before this one recorded a transform
   winning on its target scene and destroying results elsewhere. Nothing here has been run against
   anything but this poster, which is why the ticket's first done-criterion is annotating the scene
   into the corpus rather than changing code.

## Reproducing

The sweep was a throwaway PowerShell harness (staging by bicubic scale, luminance by the standard
coefficients, TSV parsed for level-5 rows) run outside the repo. It is not kept: like the
`temp/ocrdiag` and `temp/ocrscreen` harnesses before it, re-deriving it from this table is cheaper
than maintaining it. What has to survive is the scene itself, in the corpus, with annotations.
