# The single-page merge drops every page's own styles

**Status:** Implemented (2026-09-26)
**Priority:** 85
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

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

## 5. Outcome (2026-09-26)

**Scoped CSS extraction & prefixing.** Implemented in `internal/htmlgen`:
- `internal/htmlgen/scope.go`: `ScopeCSS` parses CSS rules and scopes all selectors to target a per-chapter class (e.g. `.dht-ch-N`), mapping `body`, `html`, and `:root` selectors to the scope container, prefixing inner selectors within `@media` / `@supports` blocks, and updating renamed element IDs (from collision handling).
- `internal/htmlgen/merge.go`: `rewriteRefs` now walks the entire document tree (`ch.doc`) to collect all `<style>` elements (both head and body), rebases relative URLs, scopes all rules to `.dht-ch-N`, and removes `<style>` nodes from the body DOM.
- `internal/htmlgen/singlepage.go`: `chapterInnerHTML` wraps each chapter in `<div class="dht-chapter dht-ch-N ...">` while preserving original `<body>` attributes (`class`, `style`, `dir`, `lang`, `data-*`). All chapter scoped stylesheets are gathered into `<style id="dht-scoped-css">` in the document `<head>`.
- `docs/PARITY.md`: Recorded the intentional divergence where the Go edition merges files into a single disk page using container classes and scoped rules, while the extension renders parsed fragments directly into its shared viewer DOM with `viewer.css` rules.

**Evidence.**
- `internal/htmlgen/scope_test.go`: `TestScopeCSS` verifies element, class, body, html, compound, comma-separated selectors, `@media` at-rules, keyframes/font-face preservation, and ID renaming.
- `internal/htmlgen/singlepage_test.go`: `TestGenerateSinglePagePreservesScopedStylesAndBodyAttrs` verifies that a single-page merge across multiple pages with head styles (`.pdf-flip-y`, `font-family`, `p`, `.stanza`) and `<body>` attributes (`class`, `style`, `dir`) keeps all rules scoped to `.dht-ch-1` and `.dht-ch-2`, and containers preserve original body attributes.
