# The single-page merge drops every page's own styles

**Status:** Draft
**Priority:** 85
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

**E31 (high)** - single-page output is the default mode, and the merge keeps only each page's
`<body>` children. Every `<head><style>` and every `<body>` attribute is dropped:

- PDF pages lose `.pdf-flip-y { transform: scaleY(-1) }`, so images stored bottom-up render mirrored,
  and they lose their side-by-side float layout;
- FB2 loses its stanza, subtitle and author styles;
- EPUB chapters lose their inline head CSS.

Proved with a throwaway merge test: the class survived, the rule did not
(`internal/htmlgen/singlepage.go:72-79,98-103`; the rule comes from `internal/pdf/extract.go:426,778`).

## 2. Goals

1. The merged page keeps the styling each source page carried, scoped so one chapter's rules cannot
   restyle another's (the merge already rebases body `<style>` blocks and renames colliding ids).
2. A converted PDF with a flipped image looks the same in single-page and multi-page mode.

## 3. Constraints

- This changes the output format: existing converted books keep being recognised by their completion
  record, and a result-affecting change is a rebuild reason, not a silent difference.
- The extension renders pages itself; record in docs/PARITY.md whether it has the same class of loss.

## 4. Acceptance

- A merge test with head styles on two pages keeps both rules, scoped.
- The PDF fixture with a flipped image renders upright in single-page mode (headless check,
  `/verify-view`).
