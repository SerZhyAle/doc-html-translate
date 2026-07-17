# HTML input drops every local image, leaving the output full of broken pictures

**Status:** Implemented - 2026-07-17; 28/28 images render in Chrome, verified
**Priority:** 7
**Date:** 2026-07-17

> **What landed.** `internal/htmlconv.copyLocalImages` walks the parsed body, copies each local image
> from the source's directory subtree into the output, and rewrites the `src`. Confined against `../`
> traversal (a test proves a `../secret.png` is not copied out); remote (`http(s):`, `//`) and inline
> (`data:`) and absolute paths are left untouched; a dangling ref is left as a visibly broken image, not
> dropped; the same file referenced twice is copied once, colliding basenames get a numeric suffix.
>
> **Verified by Chrome:** the illustrated Alice (`html + images/`) now reports `total=28 render=28
> broken=0` (was 0/28). The dangling fixture `html-danglingimages_Pride-and-Prejudice.html` still
> converts (Pages: 1) and shows its 164 missing images as broken icons - honest visible degradation,
> since those files genuinely are not on disk.
>
> **Extension: no port, by browser model** (recorded in `docs/PARITY.md`). A URL-loaded page lets the
> browser resolve relative images against the origin; a file picked through the picker grants no
> directory access to reach its siblings. So copying sibling files is intentionally Go-only.

> Same family as [`2026-07-17_fb2-drops-every-image`](2026-07-17_fb2-drops-every-image.md) - images
> lost, silently, by one format's reader. Different cause, so a separate ticket; worth doing together.

## What / why

Converting an HTML file whose images sit next to it on disk produces a document where **every picture
is broken**. The images are never copied into the output, the `src` attributes are carried over
unchanged, and the run reports success.

Measured 2026-07-17 on `test_doc/html-images_Alice-in-Wonderland/pg19033-images.html` (Gutenberg's
illustrated Alice - the HTML plus an `images/` folder, exactly the shape a browser's "Save page as"
produces):

| | |
|---|---|
| `<img>` refs in the source | 28 |
| ..that resolve next to the source HTML | **28 of 28** |
| Image files carried into the output | **0** |
| Output refs that resolve from the output folder | **0 of 28** |

The log:

```
[1/4] Extracting HTML...
  Title: Alice's Adventures in Wonderland | Project Gutenberg
  Pages: 1
Done.
```

Exit 0. The reader opens `index.html` and gets the text with 28 broken-image placeholders.

The cause is simply that the feature does not exist. Grepping `internal/htmlconv` for
`img|image|copy|src` returns **exactly one line** in the whole package:

```go
sb.WriteString("    img { max-width: 100%; }\n")
```

The extractor writes CSS to style images it never carries. Nothing reads `src`, nothing copies a file,
nothing rewrites a path.

Every sibling format already does this correctly, so the machinery and the conventions exist:
`epub-huge_Treasure-Island.epub` -> 157 images on disk, 199 refs, **0 broken**;
`mobi_Alice-in-Wonderland.mobi` -> 55 images. HTML is the outlier.

This is not an exotic input. "Save page as" in any browser writes an HTML file plus an assets folder,
and a saved article or a downloaded Gutenberg HTML edition is exactly the thing a user drags onto this
app.

## Worth settling in the tactical pass

- **Which refs to follow.** Relative paths next to the source are the case above. Absolute local paths,
  `file:` URLs, `data:` URIs (already inline, nothing to do) and `http(s):` remote images are all
  different decisions - remote fetching in particular is a policy question, not an obvious yes.
- **Path traversal.** `src="../../../windows/system32/.."` must not copy arbitrary files out of the
  user's disk into an output folder. Confine to the source's directory subtree.
- **Name collisions and case.** Two refs resolving to the same file, or differing only by case on a
  case-insensitive filesystem, need a stable mapping - `internal/epub` already solved this.
- **A ref that does not resolve** must degrade visibly, not silently. The corpus's other HTML fixture,
  `html-danglingimages_Pride-and-Prejudice.html`, carries 164 refs to images that are not there and is
  the fixture for exactly that path - see
  [`2026-07-17_cli-diagnostics-honesty`](2026-07-17_cli-diagnostics-honesty.md).

## Cross-references

- Go: [`internal/htmlconv/extract.go`](../../../internal/htmlconv/extract.go) - the whole package; there is
  no image path to point at yet.
- Compare: `internal/epub`, `internal/mobi` - both land images on disk and rewrite refs correctly.
- Extension: it loads HTML from fetched bytes over a URL, where the browser resolves relative images
  against the origin - a different model, so a Go-side fix does not obviously apply. Check what a local
  file picked through the extension's file picker does before recording a parity verdict.

## Done criteria

- [x] An HTML file with local images converts to a document whose images display.
- [x] Verified by opening the output in Chrome (28/28 render), not by counting `<img>` tags.
- [x] Refs outside the source's directory subtree are not copied (`TestExtract_HTMLPathTraversalRefused`).
- [x] A dangling ref degrades visibly; `html-danglingimages_Pride-and-Prejudice.html` still converts.
- [x] Extension checked, not assumed: no port needed by browser model (see the status note + PARITY.md).
- [x] Tests / gate green: `internal/htmlconv` (unit) + `tests` (215s) + lint; extension `npm test` 45/45.
- [x] Changelog entry in `DEV/CHANGELOG.md`.
