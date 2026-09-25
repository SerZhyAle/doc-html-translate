# Converted pages claim English when the source language is unknown or ignored

**Status:** Draft
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
