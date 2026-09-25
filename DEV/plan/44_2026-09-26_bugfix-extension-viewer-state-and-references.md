# Extension viewer and popup: stale state, broken references, wrong site

**Status:** Draft
**Priority:** 65
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **B39** - a superseded PDF load still sets the new document's `<html lang>` and banner (the load token
  is checked once in `renderDocument`), so Chrome can offer translation from the wrong language.
- **B40** - the HTML export re-encodes every image as JPEG; a transparent PNG exports as a black box.
- **B41** - toolbar state carries over between documents (the TOC button stays hidden, the OCR group
  stays shown).
- **B51** - ids are prefixed but `url(#id)`, `usemap`, `label for` and `aria-*` references are not, so
  SVG gradients, clip paths and image maps break.
- **B58** - the popup never shows which site its switch applies to (the host span is detached by i18n).
- **B59** - the per-site switch stores the tab's host while interception is decided by the request's
  host: it is a no-op on the viewer tab and misses cross-host PDF links.

## 2. Goals

1. Every await in a load re-checks that the load is still current.
2. The export keeps transparency.
3. Toolbar state is reset per document.
4. Every id reference is rewritten together with its id.
5. The popup names the site it acts on, and the switch excludes what the user means.

## 3. Constraints

- No new permissions. Strings only through the existing locales, every locale in one edit.

## 4. Acceptance

- One `extension/test` case per item, failing before and passing after.
