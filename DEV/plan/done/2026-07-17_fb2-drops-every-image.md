# FB2 conversion silently drops every image

**Status:** Implemented - 2026-07-17; 55/55 images render in Chrome, verified
**Priority:** 6
**Date:** 2026-07-17

> **What landed.** `internal/fb2` rewritten from struct-unmarshal (which captured only `<p>`) to a
> single token walk (`collectContent`) that keeps body content in document order and decodes every
> `<binary>`. Referenced binaries become sibling files (`imageFileName`, deduped, written once); the
> coverpage image opens the book; a reference with no binary shows a visible `[image not found: id]`
> placeholder rather than vanishing.
>
> **The catch the ticket did not predict:** FB2 wraps most illustrations *inside* a `<p>`
> (`<p><image l:href="#id"/></p>`), not only between paragraphs. The first pass found 1 image (the cover)
> because `readParagraph` swallowed the inline `<image>`; it now pulls inline images out of each `<p>`.
> After that, Alice went 1 -> 55 images.
>
> **Verified by Chrome, not a tag count:** headless Chrome reports `total=55 render=55 broken=0` on the
> illustrated Alice. `fb2-russian_Moskoviya.fb2`: 47 refs -> 41 files (6 refs reuse an image, deduped),
> 0 broken. Extension already inlines FB2 `<binary>` as `data:` URLs, so it was correct already - the
> file-vs-inline split is recorded in `docs/PARITY.md` (same pattern as EPUB).

## What / why

An illustrated FB2 converts to a **text-only** book. Every image is dropped, and nothing in the log
says so - the conversion reports success.

FB2 embeds its images as base64 `<binary>` elements at the end of the file, referenced from the body
by `<image l:href="#id"/>`. Both halves are ignored today. Measured on
`test_doc/fb2-illustrated_Alice-in-Wonderland.fb2`:

| | Count |
|---|---|
| `<binary>` elements in the source | 55 |
| `<image>` refs in the source body | 55 |
| Image files written by the conversion | **0** |
| `<img>` tags across all 22 output HTML files | **0** |

Chapters survive - the same run produces 21 anchors and the correct title - so this is images
specifically, not a broken extractor.

For contrast, the sibling formats carrying the same book do it correctly:
`mobi_Alice-in-Wonderland.mobi` -> 55 image files, `azw3_Alice-in-Wonderland.azw3` -> 45. So the
pipeline downstream of extraction handles images fine; the gap is in the FB2 reader.

This matters more than the fixture count suggests. FB2 is a mainstay of Russian-language ebooks - it
is the format's native habitat, and it is where illustrated FB2 actually turns up
(`test_doc/fb2-russian_Moskoviya.fb2` carries 41 `<binary>`, also all dropped). A reader converting an
illustrated FB2 gets a book with the pictures quietly missing and no hint that anything was lost.

Worth deciding in the tactical pass: images become sibling files next to the HTML (as the PDF and
EPUB paths do) rather than `data:` URIs - `internal/htmlgen/favicon.go` already made that call once,
for the same reason (a 21 KB payload times N pages).

## Where to look

- [`internal/fb2`](../../../internal/fb2) - the reader. `<binary id=".." content-type="image/jpeg">` holds
  base64; body refs use the XLink namespace (`l:href="#id"`), so both the namespace and the leading
  `#` need handling.
- Compare against `internal/mobi` / `internal/epub`, which already land images on disk correctly.

## Done criteria

- [x] An illustrated FB2 produces its images and references them from the HTML.
- [x] Both fixtures convert with their images present - checked by opening in Chrome (Alice 55/55 render),
      not by counting log lines.
- [x] A malformed `<binary>` or a dangling `<image>` ref degrades visibly (`[image not found: id]` +
      a WARNING line), not silently (see [`2026-07-17_cli-diagnostics-honesty`](2026-07-17_cli-diagnostics-honesty.md))
      - covered by `TestExtract_FB2Images`.
- [x] Extension checked: it inlines `<binary>` as `data:` URLs, already correct; recorded in
      `docs/PARITY.md` as the file-vs-inline divergence.
- [x] Tests / gate green: `internal/fb2` + `tests` (215s) + lint; extension `npm test` 45/45.
- [x] Changelog entry in `DEV/CHANGELOG.md`.
