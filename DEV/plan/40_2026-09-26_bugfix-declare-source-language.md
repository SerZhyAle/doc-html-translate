# Converted pages claim English when the source language is unknown or ignored

**Status:** Implemented
**Priority:** 65
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

A wrong `<html lang>` can stop Chrome offering "Translate page", which is the product's free flow.

- **X30** - FB2 pages hardcode `lang="en"` and ignore `<title-info><lang>` (the extension reads it); the
  same literal is in the txt, rtf, pdf, img and comic extractors (**X33** for image input).
- **E30** - the merged page and the TOC index fall back to `lang="en"` when the first page declares
  none; Markdown declares none on purpose, and EPUB `dc:language` is never read.
- **E33** - the index reads `dir` only from `<html>` while the merge also reads `<body dir>`.

## 2. Goals

1. Every extractor declares the source language when the document states it (FB2 `<lang>`, EPUB
   `dc:language`), and declares none rather than a guess when it does not.
2. The merge and the index never invent a language, and read `dir` the same way.

## 3. Constraints

- The interface language dresses the chrome only; the page keeps the document's language
  (`TestConvertedChromeLanguage`).
- Output-format change: covered by the completion record's rebuild rule.

## 4. Acceptance

- A Russian FB2 converts to `<html lang="ru">`, a Markdown book to a page with no `lang`; tests pin both.

## Implementation (2026-09-26)

- **X30** - FB2 reads `<title-info><lang>` (never `<src-title-info>`) into `fb2Doc.lang`
  ([`internal/fb2/content.go`](../../internal/fb2/content.go)); `Extract` sets the new `epub.Book.Language`
  from it and each page opens `<html lang="..">`, or plain `<html>` when the book states none
  ([`internal/fb2/extract.go`](../../internal/fb2/extract.go)). The declared value is cleaned by the new
  `textutil.NormalizeLangTag` ([`internal/textutil/lang.go`](../../internal/textutil/lang.go)), the twin of
  `lang.js` `normalizeLangTag`. TXT, RTF, PDF (both page builders) and comic stop writing `lang="en"`.
  The PDF "nothing to convert" fallback page keeps `lang="en"` on purpose: its only text is the app's own
  English note. Tests: `internal/fb2` `TestExtract_FB2DeclaresTitleInfoLang`, `internal/textutil`
  `TestNormalizeLangTag`.
- **X33** - image input ([`internal/img/extract.go`](../../internal/img/extract.go)) declares no `lang`,
  the same one-line change as TXT/RTF/comic; the end-to-end rule "no stated language -> `<html>`" is
  pinned through the Markdown and TXT paths (`TestConvertedSourceLanguage`, `TestConvertedChromeLanguage`),
  not by an image-specific test.
- **E30** - EPUB `dc:language` is read into `Book.Language` (`parseOPF`), and a content page whose
  `<html>` declares neither `lang` nor `xml:lang` is given it during normalization (`declareBookLang` in
  [`internal/epub/normalize.go`](../../internal/epub/normalize.go)). The merged page and the TOC index take
  the first page's language, else `Book.Language`, else none - the `"en"` fallback is gone
  (`rootLangDir` / `rootAttrs` in [`internal/htmlgen/singlepage.go`](../../internal/htmlgen/singlepage.go),
  `indexRootAttrs` in [`htmlgen.go`](../../internal/htmlgen/htmlgen.go)). Tests: `internal/epub`
  `TestExtractDeclaresPackageLanguage`, `internal/htmlgen` `TestGenerateSinglePageDeclaresOnlyAStatedLanguage`
  and `TestGenerateIndexCarriesDocumentLang`, and the acceptance test
  [`tests/source_lang_test.go`](../../tests/source_lang_test.go) `TestConvertedSourceLanguage` (Russian FB2
  -> `<html lang="ru">`, Markdown -> `<html>`, single-page and multi-page each; on the old code it fails
  with `<html lang="en">` in all four cases).
- **E33** - the index now reads `dir` through the same `htmlDir` the merge uses (`<html>`, then `<body>`);
  pinned by the `<body dir="RTL">` case of `TestGenerateIndexCarriesDocumentLang`.
- **Extension side** - FB2, EPUB and HTML already read the stated language. `normalizeLangTag` gained the
  word-boundary rule its Go twin has (`russian` no longer reads as `rus`, `zh-Hans` no longer as `zh-HA`),
  with the same cases in `extension/test/reflow.test.mjs`. The extension still fills an unstated language
  from PDF `/Lang` and the script heuristic; recorded as an intentional difference in
  [`docs/PARITY.md`](../../docs/PARITY.md) "Declared source language", and the new pair is in
  `configs/parity-map.json`.
- **Rebuild rule** - no format-version bump: the completion record deliberately does not compare the tool
  version, so existing output is reused until `-force` or an option change, as with earlier
  output-format fixes.
- **Left open** - Go does not read PDF `/Lang` (the extension does); a follow-up if wanted. Nothing here
  needs Windows-only proof.
