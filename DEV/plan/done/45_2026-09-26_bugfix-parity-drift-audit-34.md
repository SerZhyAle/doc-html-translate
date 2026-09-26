# The Go and JS editions have drifted on eighteen behaviours

**Status:** BlockNeedUserTest - implemented 2026-09-26; left: `go test ./internal/epub/` on Windows; Chrome: Calibre EPUB anchors, windows-1251 chapter, self-extracting .cbz
**Priority:** 60
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).
> docs/PARITY.md is the authority on which side is right.

## 1. Problem

The same input gives different results in the desktop app and the extension, or docs/PARITY.md says
something the code does not:

- EPUB: **B46** (the extension halves of E6, E12, E14: XHTML parsed as HTML, UTF-8-only decoding, SVG
  cover text lost), **E36** (TOC href resolution), **E37** (root-relative links), **B48** (container
  fallback), **E39** (symlink entries - PARITY is wrong).
- FB2: **B34** (stanza title and subtitle dropped), **B35** (cover and missing-image placeholder).
- TXT: **X34** (form feed and Unicode line separators).
- Markdown: **E26** (nested heading split). HTML input: **E28** (where the language comes from).
- Comics: **B30** (symlink entries), **B31** (TAR name precedence), **B32** (no-signature fallback).
- MOBI: **B33** (a document-supplied link marker survives).
- OCR: **B64** (column-order confidence floor on rescue passes), **B38** (CJK class), **B56**
  (`ocr-screen.js` missing from `configs/parity-map.json`).
- Docs only: **B42**, **E29** (stale constants and names in PARITY.md).

## 2. Goals

For each item: make both editions behave the same, or record the divergence in docs/PARITY.md with its
consequence - never leave it unwritten. Pin every value that can be pinned in `tests/parity_test.go`.

## 3. Constraints

- One ticket, both editions, per the parity process. OCR items follow the `OCR-PIPELINE` contract:
  amend the catalog first if a stage changes.

## 4. Acceptance

- Each item is closed by a paired Go and JS test on one fixture, or by a PARITY.md divergence entry.

## Implementation (2026-09-26)

Every item is closed by a shared fixture under `tests/testdata/` that a Go package test and an
`extension/test` file both run, except where noted; each fails against the pre-change code of the
edition that was wrong (checked by swapping the old file back in) and passes after. docs/PARITY.md names
each fixture in the Guard line of its section. No output-format constant bumped: `internal/outputpath`
has none, and its completion record deliberately compares result-affecting options, not the tool
version (`CheckReuse`), so an existing output built before this change is reused until `-force`.

**EPUB**

- **B46** (extension halves of E6, E12, E14). New `extension/src/epub-normalize.js` (`xhtmlToHtmlSyntax`,
  the port of `xhtmlToHTMLSyntax`; `coverSvgImage` / `outermostSvg`, the port of `singleImageHref`) and
  `extension/src/charset.js` (`decodeChapter` = `decodeToUTF8`, `decodeHtml` = htmlconv `parseDocument`).
  `epub.js` rewrites XHTML chapters and the nav document before parsing, decodes every book file by its
  declaration, and replaces only a true cover SVG; any other SVG keeps its `<image>`, pointed at the blob.
  `html.js` decodes through `decodeHtml`. Go: an empty SVG `<title/>` put the tokenizer in raw-text mode
  and swallowed the rest of the chapter; `xhtmlToHTMLSyntax` now clears raw-text mode for every
  self-closing tag (`internal/epub/normalize.go`). Fixture `content_fidelity_cases.json`; tests
  `TestContentFidelitySharedCases` (internal/epub), `TestCharsetSharedCases` (internal/htmlconv),
  `content fidelity: shared Go/JS ..` and `renderChapter: an XHTML self-closing anchor ..`
  (extension/test/epub-parity.test.mjs), `parseHtml: a windows-1251 page ..` (html.test.mjs). The
  existing `convertSvgImage: multi-image svg ..` test now expects the `<image>` kept, since an HTML `<img>`
  inside an SVG never renders.
- **E36** Go `resolveTOCHref` resolves through `resolveBookPath` from the TOC folder (`toc.go`).
  Fixture `epub_target_cases.json`; `TestTOCTargetSharedCases`, `resolveTocAnchor: shared Go/JS target
  fixture` (`resolveTocAnchor` exported for it).
- **E37** Go `rewriteLinks` takes the OPF directory and rewrites a root-relative (or backslash) link
  relative to its file via `resolveBookPath` (`links.go` `rewriteRootLink`, `normalize.go`). Same fixture;
  `TestLinkTargetSharedCases`, `rewriteAnchor: shared Go/JS target fixture`.
- **B48** JS `parseContainer` no longer falls back to the first rootfile of any type (exported for the
  test); Go now matches `.opf` in any letter case and skips a rootfile with no path, as JS did. Fixture
  `epub_container_cases.json`; `TestContainerSharedCases`, `parseContainer: shared Go/JS fixture`.
- **E39** PARITY "Input limits" rewritten to what both editions now do: a symlink is never unpacked but
  counts toward the entry budget and total. JS `unzip` skips non-regular entries through the new
  `limits.js` `zipEntryIsRegular` (the port of `archive/zip` `FileHeader.Mode`). Fixture
  `archive-parity/symlink.epub`; `TestArchiveParityEPUB`, `unzip: shared Go/JS archive fixture ..`.

**FB2** - **B34** a stanza's `<title>` and `<subtitle>` render before its verses; **B35** the first
`<coverpage>` image opens the book, and an `<image>` with no binary leaves `[image not found: <id>]`
(`fb2.js`). Fixture `fb2_parity_cases.json`; `TestFB2SharedCase` (internal/fb2), `parseFb2: shared Go/JS
fixture`.

**TXT** - **X34** `txt.js` `splitParagraphs` maps NEL, LS, PS, VT and FF to line breaks. Fixture
`txt_paragraph_cases.json`; `TestParagraphsSharedCases` (internal/txt), `splitParagraphs: shared Go/JS
fixture`.

**Markdown** - **E26** Go `splitBySections` splits only at top-level `<h1>`/`<h2>` (depth-tracking
tokenizer, `internal/md/extract.go`). Fixture `md_section_cases.json`; `TestSectionsSharedCases`,
`parseMarkdown: sections, shared Go/JS fixture` (new extension/test/md.test.mjs).

**HTML input** - **E28** `html.js` `declaredLang` reads `<html>` then `<body>`, `lang` then `xml:lang`, as
`rootAttrs` does. Kept minimal for ticket 40: the remaining difference (Go copies the tag verbatim, the
extension normalizes it) is recorded in PARITY "HTML input language" with its consequence. Fixture
`html_lang_cases.json`; `TestLangSharedCases`, `parseHtml: declared language, shared Go/JS fixture`.

**Comics** - `comic.js`: **B30** the ZIP lister skips non-regular entries (`zipEntryIsRegular`);
**B31** a GNU long name wins over PAX `path=`; **B32** `detectContainer(u8, name)` recognizes `ustar` at
257 and falls back to the file extension as Go's `containerKind` does, and `zipEntries` finds a ZIP behind
a stub (`zipBaseOffset`, ported from `archive/zip`); `viewer.js` passes the file name through. Fixtures
`archive-parity/{symlink.cbz,names.cbt,prefixed.cbz}` built by `archive-parity/gen.go`;
`TestArchiveParityComics` (internal/comic), `archive fixtures: shared Go/JS page lists` and
`detectContainer falls back to the file extension ..` (comic.test.mjs).

**MOBI** - **B33** `ebook.js` `retargetBookLinks` strips a document-supplied `data-dht-target` from every
section and re-serializes it. No Go counterpart (Calibre converts on the desktop, there is no marker), so
it is a JS test only - `retargetBookLinks: a document-supplied target marker never becomes a link`
(ebook.test.mjs) - with the rule recorded in PARITY under "New-format parsing stacks".

**OCR** - both items bring the extension to the desktop mechanism the `OCR-PIPELINE` catalog documents;
no stage or constant changes, so no catalog amendment is needed (docs/contracts/OCR-PIPELINE.md: "the
document describes this product's own mechanism"). **B64** `ocr-overlay.js` `collectLines` takes the pass
floor and hands it to `orderColumns`; the rescue, screen-rescue and screen-sweep passes pass
`OCR_RESCUE_LINE_CONF`. Fixture `ocr_column_order_cases.json` (`TestOrderColumnsSharedCases`,
`orderColumns: shared Go/JS fixture, per pass floor`) shows what the floor changes, and
`TestParityOCRColumnOrderFloor` (tests/ocr_floor_parity_test.go) pins that every pass orders by the floor
it drops by. **B38** `ocr-text.js` CJK class is `\p{Script=Han|Hiragana|Katakana|Hangul}` with the `u`
flag. Fixture `ocr_translatable_cases.json`; `TestIsTranslatableSharedCases`, `isTranslatable: shared
Go/JS fixture`. **B56** `ocr-screen.js` added to the OCR pair in `configs/parity-map.json` and a port-map
row for `screen.go`; `TestParityMapWatchesEveryOCRPort` (tests/parity_map_test.go) now fails for any
`ocr-*.js` that names the Go file it ports and is not watched. The two new modules, `charset.js` and
`epub-normalize.js`, are in the port map and the EPUB / HTML pairs too.

**Docs** - **B42** PARITY `PAGE_CHUNK` / `CHUNK_LEAD` corrected to 100 / 5 and the stale viewer line links
dropped; `TestParityReaderFonts` and `TestParityDocChunkConstants` (tests/reader_parity_test.go) pin the
three font stacks across both editions and the doc, and the two chunk numbers against the code. **E29**
`copyLocalImages` replaced by the `internal/assets` `Copier`.

**For the owner to verify on Windows:** `go test ./internal/epub/` (the root-link rewrite goes through
`filepath.Rel` and back to slashes); in Chrome, a Calibre EPUB with `<a id/>` anchors and a titled SVG
cover, a windows-1251 EPUB chapter and HTML page, and a self-extracting `.cbz`, since the extension tests
run on linkedom rather than the browser's parser.
