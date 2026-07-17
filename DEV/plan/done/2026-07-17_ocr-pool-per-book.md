# OCR pool: batch recognition per book, not per content file

**Status:** Implemented
**Priority:** 11
**Date:** 2026-07-17

> Split out of [`2026-07-17_comic-archives`](2026-07-17_comic-archives.md) by owner decision
> (2026-07-17): comic support ships first, this perf fix follows. Desktop-only - the extension's OCR is
> lazy on scroll and has no equivalent defect.

## What / why

The desktop OCR overlay is advertised as running "across a process pool" of `tesseract` workers, and on a
single-page-mode book it does. On a **multi-page** book it silently degrades to serial recognition: the
worker pool is scoped to the jobs found *inside one HTML file*, while the overlay loop walks content
files one at a time. A page carrying exactly one image therefore hands the pool a one-item queue, and the
pool's width buys nothing.

The shape is the same wherever one page holds one image:

- **Scanned PDFs outside single-page mode** - the defect exists today, unnoticed because the pool
  comment was written around the single-page case.
- **Comic archives outside single-page mode** (the ticket that surfaced this) - one image per page by
  definition, and OCR there is forced.

The pool was designed against the single-page case, where the whole book is one content file, so the
per-file scoping looked equivalent to per-book. It is not, and the two formats where OCR matters most are
exactly the ones it fails.

**Scope check, measured 2026-07-17** - worth stating so this ticket is not read as bigger than it is.
Single-page mode is the **default** (`SinglePage: !*multiPage`), and there the pool is genuinely at full
width: `Aphrodite's Mirror (1).pdf` (2304 pages, no text layer, forced OCR) recognizes at ~40 images/s on
a 20-core machine (pool width 16), finishing in a few minutes. So the defect does not touch the default
desktop path for scanned PDFs.

It bites **`-multipage` only**, in either format. Aphrodite settles the comic case too, because it *is* the
comic case structurally - 2304 pages, no text layer, forced OCR, one image per page - and in the default
mode the whole book merges into one content file, so the pool sees all 2304 images at once. An earlier
draft of this ticket claimed comics would land as one-image-per-page content files and that the pool must
therefore be fixed *before* comics ship. That conflated the extractor's `page_%03d.html` shape with the
content-file count at OCR time, which single-page mode collapses to one. The owner's split
([`2026-07-17_comic-archives`](2026-07-17_comic-archives.md): comics first, this fix follows) is correct
and carries no measured cost on the default path. This is a `-multipage` perf ticket, not a blocker.

## Approach sketch

Lift the batching boundary from "one content file" to "the whole book": collect recognition jobs across
every content file first, run one pool over the full queue, then write each page back. The per-file
parse/rewrite stays where it is - only the recognition fan-out moves. Progress reporting is already
per-image (deliberately, so single-page mode does not sit at 0/1), so the ticker should survive the move
largely intact.

Watch: memory (holding N parsed pages vs one), and keeping the "best-effort, never fatal" contract - a
failed image or a missing tesseract must still never abort a conversion.

## Cross-references

- `internal/pipeline/pipeline.go` - `overlayImages`, the sequential per-file loop.
- `internal/ocr/overlay.go` - `OverlayFile`, `recognizeJobs`, `ocrWorkers`.
- `docs/PARITY.md` - the "OCR execution" divergence row states Go OCRs eagerly across a pool while the
  extension is lazy on one worker. The row stays true; only the Go pool's real width changes.

## What was done

`internal/ocr/overlay.go`: the batching boundary moved from one content file to the whole book.
`OverlayBook(bin, htmlPaths, ..)` runs three phases - **(1)** parse each file and collect the recognizable
image paths (deduped by absolute path, first-seen order; docs released); **(2)** `recognizePaths` runs one
`ocrWorkers()`-wide pool over *every* image at once; **(3)** re-parse each file one at a time and wrap its
images from the precomputed `map[path]recognition`. Only the small results map is held across the book -
never every parsed page or decoded image - so memory stays close to the old per-file cost (the ticket's
memory watch). `decodeImage` (plate colours) and the HTML rewrite stay in phase 3, one file at a time,
because the DOM is not safe for concurrent mutation. `OverlayFile` is retained as `OverlayBook` over a
one-file book. `internal/pipeline/pipeline.go`: `overlayImages` now builds the content-file path list and
makes a single `OverlayBook` call under one book-wide ticker.

## Measured (2026-07-17, real tesseract, `cbz-tiny_Nyoka-the-Jungle-Girl-071`, 37 pages, 20-core / pool 16)

A faithful before/after: the "before" loops the retained `OverlayFile` per page (its pool is that page's
one image, i.e. exactly the old serial path), the "after" is one `OverlayBook`. Same output both ways
(36 overlaid, 1 no-text, 0 failed).

| | Wall-clock | Rate |
|---|---|---|
| BEFORE (per-file pool = serial) | **67.2 s** | 0.55 img/s |
| AFTER (book-wide pool) | **11.0 s** | 3.35 img/s |

**6.1x faster** on the `-multipage` path, which is exactly where the defect lived; the default single-page
path was already at full width and is unchanged.

## Done criteria

- [x] A multi-page scanned PDF and a multi-page comic archive recognize at the pool's full width, not
      serially. (Batching boundary is now the book; measured 6.1x on a 37-page comic in `-multipage`.)
- [x] Wall-clock improvement measured and recorded on a real multi-page sample, before and after (above).
- [x] Best-effort contract preserved: missing tesseract, an unreadable page, and a single failed image
      still never abort (phases skip; `TestOverlayBookNoImagesNeverTouchesEngine` proves the no-op path
      touches no engine).
- [x] Progress reporting stays per-image and readable in both modes (one book-wide ticker over the unique
      image count; `recognizePaths` reports per completion).
- [x] Coverage for the batching boundary added: `collectBookImages` (cross-file collect + dedup + order,
      external/missing filtered) and `applyOverlays` (three outcomes + changed flag) in `internal/ocr/batch_test.go`.
- [x] Gate green: `scripts/test.ps1`.
