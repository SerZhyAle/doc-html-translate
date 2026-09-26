# Extension viewer and popup: stale state, broken references, wrong site

**Status:** BlockNeedUserTest - implemented 2026-09-26; left: real Chrome: switched-off site cross-host PDF, popup host on the viewer tab, EPUB export keeps PNG alpha
**Priority:** 65
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

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

## Implementation (2026-09-26)

Every test below was run against the pre-change source and failed, then passed after. Extension suite
279 pass / 0 fail (baseline 271); `go build ./... && go vet ./... && go test ./...` green.

- **B39** - `renderDocument` (`extension/src/viewer.js`) re-checks the load token after the sample, after
  the metadata read and after the TOC, and releases the superseded `pdf` at whichever await notices;
  `setDocumentLang` became `pdfDocumentLang`, which only decides the language, so `applyLang` runs only
  for a current load. `warnIfNoText` and the Export unhide are skipped once a newer load owns the page.
  The other loaders already checked after every await. Test: `viewer.test.mjs` "a PDF superseded while
  its language is being read leaves the newer document's language alone" (was `de`, now `en`).
- **B40** - `exportImageEncoding` (`extension/src/export-html.js`) scans the drawn image's alpha in
  256-row bands: any pixel short of opaque exports as PNG, an opaque image stays JPEG 0.85 (scans and
  comic pages stay compact). `buildImageDataMap` in `viewer.js` uses it. Test: `export-html.test.mjs`
  (real `@napi-rs/canvas`: a transparent pixel below the first band, a blank canvas, an opaque fill).
- **B41** - `resetToolbar` in `viewer.js`, called from `teardownCurrent`, puts the TOC button, TOC
  entries, OCR layer group, Export button and the "Save file" source bytes back to their page-load
  state; `renderToc` now shows the button for a document with a TOC (and closes the panel for one
  without); the image loader calls `renderToc(null)` like the comic loader. Test: `viewer.test.mjs`
  "toolbar state is reset for each document" (TXT, then FB2 with a TOC, then TXT).
- **B51** - `scrubTree` (`extension/src/url-policy.js`) rewrites every same-section reference with the
  id: `url(#id)` in any attribute (quoted or not; inline `style` is still dropped whole), `#id` hrefs on
  SVG `<use>`/`<image>` and `<area>`, `usemap`, and the id-list attributes `for`, `headers`, `list`,
  `form`, `popovertarget`, `commandfor` and the `aria-*` relationships. A `<map name>` is kept,
  namespaced, because `usemap` finds a map by name and a map's name is not exposed on `window`/`document`
  (no clobbering). EPUB's `rewriteFragments: false` now only spares the `<a>` links `rewriteAnchor`
  already retargeted; before, it also left SVG references in EPUB chapters unprefixed. Tests:
  `sanitize.test.mjs` "every same-section id reference follows the namespaced id" and `epub-dom.test.mjs`
  "renderChapter: SVG and image-map references follow the namespaced ids". `docs/PARITY.md` (EPUB output
  model divergence) updated.
- **B58** - `popup.html`: `data-i18n` moved to an inner `<span>`, so localization no longer replaces the
  label's content and detaches `#host`. The "(not a website)" literal now goes through the new
  `popupNoSite` key, added to all 13 `extension/_locales` in one edit. Test: `popup.test.mjs` "the popup
  names the site its switch acts on".
- **B59** - decision: the switch means "leave this site's documents alone". The popup keys it on the
  site the reader is looking at - the tab's page host, and on the viewer's own tab the host of the
  shown document (`?file=`; a local document is on no site). Interception excludes a stored host both
  as the request's domain (a document on the site) and as the initiator's domain (a document on another
  host that a page of the site links to), via `excludedInitiatorDomains` - no new permission. A URL typed
  into the address bar has no initiator, so only its own host counts. New `extension/src/site-host.js`
  holds `siteHost` and `parseFileParam` (moved out of `viewer.js`); the decision is written in its header,
  in `background.js` and in `extension/README.md` (Known limitations). Tests: `popup.test.mjs` "on the
  viewer tab the switch acts on the shown document's site" and `background.test.mjs` "a switched-off
  site's documents are left alone wherever they are served from".

Open / for the owner:
- Check in a real Chrome: with a site switched off, a PDF link from it to another host opens in Chrome's
  own viewer; the popup on a viewer tab names the document's host; "Export HTML" of an EPUB with a
  transparent PNG keeps the transparency.
- Not in this ticket's findings, noticed on the way: a remote `url(https://..)` in an SVG presentation
  attribute is neither parked nor dropped by `scrubTree` (only `url(#id)` is handled). Whether Chrome
  fetches it for `fill`/`mask`/`filter` was not verified; worth a content-security follow-up.
- `viewer.js` was already past the file-size budget (about 1770 lines); this ticket did not split it.
