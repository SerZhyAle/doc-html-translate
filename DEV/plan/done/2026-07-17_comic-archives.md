# Comic archives (CBZ / CBR / CB7 / CBT) as an input format

**Status:** Implemented
**Priority:** 12
**Date:** 2026-07-17

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

A comic archive is a container of page images with no text layer at all - every word lives inside the
artwork, in speech bubbles and captions. Today a reader with comics or manga has no working path: the
desktop app silently treats `.cbz` as plain text and emits garbage, and the extension routes it to the
EPUB reader, which fails with a misleading "not an EPUB" error. Supporting comic archives means a user
opens the file and reads it page by page in the browser, with the text recognized and laid over each
page as translatable plates - so the browser's built-in "Translate page" works on the bubbles. Without
that recognition step the output would be a picture gallery with nothing to translate, which is why OCR
is forced here rather than left to a flag (same reasoning as a standalone image input).

This is roadmap item [23] from [`competitor_feature_research_ru.md`](../research/competitor_feature_research_ru.md#L113-L116)
and the comic half of the "separate roadmap items" deferred by
[`2026-07-01_extension-format-parity`](done/2026-07-01_extension-format-parity.md#L52).

## Owner decisions (2026-07-17)

Recorded because two of the three set the shape of the work and one is a knowingly accepted regression risk.

1. **OCR is forced for comic archives**, independent of `-ocr` / the "Use OCR for images" toggle - the
   same rationale as `internal/img`: opening a comic *is* the request to read its text.
2. **Scope is all four formats**: CBZ, CBR, CB7, CBT (not CBZ alone).
3. **The OCR pool perf defect is out of scope** - tracked separately in
   [`2026-07-17_ocr-pool-per-book`](2026-07-17_ocr-pool-per-book.md).

**Accepted risk from (1) + (3) together - measured 2026-07-17, and it is smaller than first written.**
This ticket originally recorded the risk as "strictly serial recognition, roughly 200 s for a 200-page
comic instead of ~15 s". That is **wrong for the default path**, and the correction matters because it is
what makes decision (3) cheap rather than costly.

Single-page mode is the **default** (`SinglePage: !*multiPage`), and there the whole book is one content
file, so the pool sees every image at once and runs at full width. The empirical case is already in the
corpus: `Aphrodite's Mirror (1).pdf` is structurally a comic - 2304 pages, no text layer, forced OCR, one
image per page - and it recognizes at **~40 images/s** (54.2 s total, 20-core machine, pool width 16). By
that rate a 200-page comic costs about **5 s**, not 200 s.

So the serial degradation bites `-multipage` only. Decision (3) stands and the feature does **not** ship in
a slow form on the default path; [`2026-07-17_ocr-pool-per-book`](2026-07-17_ocr-pool-per-book.md) is a
`-multipage` fix, not a precondition for comics. The extension is unaffected either way: its OCR is lazy on
scroll, so the cost is spread across reading.

## Format / container map

The four extensions are **three different containers**, which is what decides feasibility per edition.
Only CBR and CB7 need an external tool; the 7-Zip dependency is not "all comics", it is half of them.

| Ext | Container | Go | Extension (browser) |
|---|---|---|---|
| `.cbz` | ZIP | stdlib `archive/zip` | existing hand-rolled `unzip` |
| `.cbt` | TAR | stdlib `archive/tar` | small hand-rolled reader (TAR is trivial) |
| `.cbr` | RAR | shell out to 7-Zip | **infeasible** - no RAR decoder |
| `.cb7` | 7z | shell out to 7-Zip | **infeasible** - no 7z decoder |

The 7-Zip shell-out follows the established MOBI/Calibre precedent: detect on PATH, fail with a clear
"install 7-Zip" notice, never bundle. Note this makes CBR/CB7 the second runtime-dependency format on
desktop, which the docs and store copy must state as plainly as they state Calibre for MOBI.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` Done | All four via `internal/comic`. CBZ/CBT pure stdlib; CBR/CB7 via `find7Zip` (LookPath + probe paths). Forced OCR wired in `pipeline.go`. Unit tests for natural sort, entry filter, CBZ+CBT extraction |
| GUI (`doc-html-ui`) | `[x]` Done | Picker gains a "Comics" filter group + the combined filter; drop-zone `dropSupported` copy updated in en/ru/uk. No new flag |
| MSIX Store app | `[x]` Done | Four `<uap:FileType>` rows added to `AppxManifest.xml`; inherits the GUI. Store listing copy in `msix/README.md` updated. 7-Zip not bundled, so CBR/CB7 surface the same notice |
| Browser extension | `[x]` Done **(Partial by design)** | CBZ + CBT implemented (`comic.js`, lazy inflate, forced OCR, ZIP disambiguation); **CBR/CB7 declined** via `DesktopOnlyError`. Recorded under "Intentional divergences" in `docs/PARITY.md` |
| Website / docs | `[x]` Done | en/ru/uk across README, docs.*.html, index/extension pages, store listing, `_locales`, DEV/AGENTS docs. States the 7-Zip requirement for CBR/CB7 and that the extension covers CBZ/CBT only |

## Shared invariants touched

Two **new** shared invariants are introduced; both must be added to the tables in `docs/PARITY.md` and
guarded by `tests/parity_test.go` when this lands, because both sides implement them independently and
would drift silently.

- **Page-ordering rule.** Comic page order *is* archive entry order by filename, so the sort is
  load-bearing for correctness, not cosmetics: naive lexicographic sort puts `page10.jpg` before
  `page2.jpg`. Both editions must use the same natural (numeric-aware) sort. No natural-sort helper
  exists on either side today.
- **Page-entry filter.** Which archive entries count as pages, and what is ignored (`ComicInfo.xml`,
  `Thumbs.db`, `__MACOSX/`, directory entries, non-image files). Must match, or the two editions
  disagree on page count and numbering for the same file.

Existing invariants **not** changed: theme palette, PDF reflow constants, EPUB TOC rules, OCR
host/catalog/plate geometry, settings defaults.

## Cross-references

Code that must stay in sync (mirror these rows into the `docs/PARITY.md` port map when implementing):

| Capability | Go | JS (extension) |
|---|---|---|
| Comic archive -> page book | `internal/comic/` (new) | `extension/src/comic.js` (new) |
| Natural page ordering + entry filter | `internal/comic/` | `extension/src/comic.js` |
| Forced-OCR decision | `internal/pipeline/pipeline.go` (`forceOCR`, today gated on `img.IsImage`) | `extension/src/viewer.js` (the `loadImageData` unconditional-OCR policy, extended to comics) |

Closest existing models to follow: [`internal/img/extract.go`](../../internal/img/extract.go) (wrap an
image into a `*epub.Book`, force OCR) and the multi-page `page_%03d.html` shape in
[`internal/pdf/extract.go`](../../internal/pdf/extract.go). On the JS side,
[`unzip`](../../extension/src/epub.js) is reusable as-is for CBZ.

## Known hazards to resolve in the tactical plan

Recorded from the 2026-07-17 research pass so the tactical split does not rediscover them.

- **"Detect on PATH" is not enough for 7-Zip.** Measured on the dev machine while building the test
  corpus: 7-Zip 26.01 is installed at `C:\Program Files\7-Zip\7z.exe` and its directory is in
  **neither the machine nor the user `Path`** - its installer simply does not register there. A bare
  `exec.LookPath("7z")` therefore fails on a machine that *has* 7-Zip, and CBR/CB7 would show the
  "install 7-Zip" notice to a user who already installed it. The MOBI/Calibre precedent this ticket
  cites is `LookPath` **plus** a probe list of known install paths
  ([`internal/mobi/extract.go`](../../internal/mobi/extract.go), `findEbookConvert`) - follow it
  fully, not just its first half. Note the contrast that makes this easy to get wrong: Calibre
  *does* add itself to `Path`, so the fallback probe looks redundant right up until the dependency
  is one that behaves like 7-Zip.
- **ZIP-vs-ZIP ambiguity.** EPUB and CBZ are both `PK\x03\x04`. The extension's `detectFormat` returns
  `"epub"` for *any* ZIP before the filename is consulted, and its contract says byte signatures are
  authoritative. Disambiguation must not regress the EPUB hot path.
- **Whole-archive eager inflate (extension).** The shared `unzip` inflates every entry into memory before
  returning. That is fine for a few MB of EPUB XHTML and a plausible tab-crash for a 500 MB comic; blob
  URLs then hold a second copy. A lazy/on-demand entry path is likely a precondition, and it touches the
  ZIP layer EPUB also depends on.
- **No chunking on the shared renderer (extension).** `renderBook` renders every section in one
  synchronous pass; only the PDF path chunks, and its chunker is bound to PDF module state.
- **Six unsynchronised format lists (extension) and 13 registration points (Go)**, with no shared
  constant and no test asserting they agree. Drift is already real: `.mobi`/`.azw3` are missing from the
  file-picker `accept` list. Adding four extensions by copying the existing pattern inherits the miss.
- **Archive hardening.** The existing per-entry 100 MB cap has no aggregate or entry-count companion;
  comic archives are large and untrusted by nature.
- **Entry-name encoding.** Legacy comics zipped with CP866 / Shift-JIS names mojibake under
  unconditional UTF-8 decoding - and since page order is filename order, this is a correctness bug.
- **HTML export (extension).** The "&#8595; HTML" download re-encodes every live image to a base64 JPEG
  and silently drops images that are not yet decoded - a lazily-rendered comic would export with pages
  missing and no warning.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] A CBZ, CBT, CBR and CB7 open from the CLI, the GUI picker and drag-drop; CBR/CB7 without 7-Zip on
      PATH fail with an actionable notice, never a crash or a garbage conversion. (`readSevenZip` returns
      the "install 7-Zip" error; `find7Zip` probes known paths so an installed-but-unregistered 7-Zip is found.)
- [x] A CBZ and CBT open in the extension via the picker and the context menu; a CBR/CB7 there fails with
      a clear "desktop app only" notice rather than a misleading parse error. (`comic.js` `DesktopOnlyError`;
      `CONVERT_EXTS` includes all four so the context menu reaches CBR/CB7 for the notice.)
- [x] Pages appear in natural filename order on both editions for the same archive, and non-page entries
      are ignored identically. (`naturalLess`/`naturalCompare`; `isPageEntry` both sides; unit-tested.)
- [x] Text in bubbles is recognized and translatable via the browser's "Translate page" on both editions,
      without the user touching an OCR setting. (Go: `forceOCR` arm; extension: `registerImagesForOcr(.., true)`.)
- [x] `docs/PARITY.md` records the two new invariants, the new port-map rows, and the CBR/CB7 extension
      decline under "Intentional divergences".
- [x] Guard test added in `tests/parity_test.go` (`TestParityComicPageFilter`) for the entry filter (page-ext set);
      the ordering rule is unit-tested on each side (a string-scrape guard cannot compare an algorithm).
- [x] Gate green: `scripts/test.ps1` (Go) and `npm test` (extension, 84/84 incl. 9 comic tests).
- [x] Changelog entry in `DEV/CHANGELOG.md`; site + docs updated in en/ru/uk.

## Not covered by automation (needs a manual smoke test before release)

- Browser-side comic **rendering** (lazy inflate on scroll, forced OCR, partial-save warning) and
  real-comic **OCR quality** are not exercised by `npm test` (no DOM/engine there) - run the extension's
  manual gate (`extension/README.md`) on a real CBZ/CBT before publishing.
- CBR/CB7 desktop extraction runs only when 7-Zip is present; the corpus sweep skips them otherwise.

## Next step

`/spec-tech 2026-07-17_comic-archives` - phased tactical plan. Suggested phasing, cheapest and
highest-value first: **(1)** CBZ end-to-end on Go, **(2)** CBZ on the extension (includes the ZIP
disambiguation + lazy-inflate work), **(3)** CBT both sides, **(4)** CBR/CB7 via 7-Zip on Go,
**(5)** docs/site/store copy across every surface.
