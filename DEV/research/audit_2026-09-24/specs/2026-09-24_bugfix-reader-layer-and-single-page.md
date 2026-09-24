# Strategic spec: 2026-09-24_bugfix-reader-layer-and-single-page - Links, images and reading position survive conversion

**Ticket:** 2026-09-24_bugfix-reader-layer-and-single-page
**Status:** Draft
**Priority:** 80
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/2026-09-24_bugfix-reader-layer-and-single-page/` (created by /spec-tech)
**Findings:** E2 E3 E4 E13 E17 E18 E19 E23 E24 X23 (see `../README.md`)

> **Scope:** STRATEGIC.

---

## 1. Problem
The default single-page mode merges every chapter into one page, then deletes the chapter files.
- **Subfolder books:** books that keep chapters in a subfolder, the common Sigil layout, lose every image and relative link.
- **Footnotes:** every footnote or cross-reference link points to a deleted file.
- **Reading position:** the stored position is keyed differently on chapter pages and on the index after translation, so "Continue reading" never appears. Two books with the same title and page count share one saved position.
- **Restore vs fragment:** restoring the position overrides a TOC jump to a fragment.
- **Encoding:** file names with `#`, `%` or `?`, and external TOC links, produce broken hrefs. The page language attribute is emitted in a way that allows attribute injection.

## 2. Goals
1. In single-page mode every image and link that worked in the source book still works.
2. In-book links such as `chapter.xhtml#note12` land on the right spot in the merged page, and merged ids stay unique.
3. The reading position is stored and read under one key per book, stable across translation, and distinct between different books.
4. Opening a link with a fragment goes to the fragment, not to the saved position.
5. Every generated href and script string is correctly URL-encoded and HTML-escaped.
6. External TOC links are kept as-is. Only web and mail schemes are clickable, and script schemes are dropped.
7. Navbar and reader injection is idempotent per file, and `xml:lang` is honoured.

**Non-goals:**
- Splitting behaviour (ticket `bugfix-epub-html-content-fidelity`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** pages are opened as `file://` in Chrome; the free Chrome-translate flow must keep working (the document's `lang` stays on `<html>`).
- **Data compatibility:** saved positions under the old key are lost once. That is acceptable, but should be decided explicitly (§6.1).
- **Localization:** n/a (no new strings).

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `hotfix-epub-href-containment` (resolution gate), `bugfix-epub-html-content-fidelity` (split rewrites the same links).
- **Platform constraints:** cross-edition. Check whether the extension viewer has the same key or fragment behaviour and record the result in docs/PARITY.md.
- **Data compatibility:** migrating old positions, or dropping them.
- **Validation level:** a headless check (`/verify-view`) on a Sigil-layout fixture with footnotes, before and after translation.
- **Owner sign-off:** not required beyond review.

## 4. Current architecture context
The merge concatenates chapter bodies into one page placed in the book's base folder, then removes
the originals. Relative references are not rebased, and in-book links are not rewritten. The
reader script receives a storage key that is computed from the title at two different moments.
Hrefs are HTML-escaped, but not URL-encoded.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Reference rebasing during merge:** every relative `src`/`href` is rebased from its chapter's folder to the merged page's folder.
- **Link and id mapping:** each chapter's ids get a chapter-unique prefix, and `file#id` and `file` links are rewritten to in-page anchors.
- **Stable book identity:** one key computed once per conversion from immutable inputs (the source identity plus the original title), used by every page.
- **Fragment-first restore:** a restore happens only when the URL has no fragment.
- **Encoding discipline:** one helper for "path to href" (percent-encode each segment) and one for "string into script" (a JS-safe literal); attributes are HTML-escaped inside quotes.
- **External link policy:** schemes are allow-listed, and base prefixing is skipped for external links.
- **Idempotent injection:** a marker check prevents a second navbar or script.

### 5.2 Data & event flows
Chapters -> merge (rebase, remap ids and links) -> merged page -> reader layer (stable key).

## 6. Open questions / research items
1. **Old saved positions**
   - **Question:** migrate them, or accept a one-time loss?
   - **Status:** Open.
2. **Keep the chapter files?**
   - **Question:** after proper link rewriting, is deleting the originals still wanted?
   - **Options:** delete (smaller output); keep (safer for external deep links).
   - **Status:** Open.

## 7. Risks
- **The id prefix breaks book CSS that targets ids.** Likelihood: medium. Impact: styling lost. Mitigation: rewrite `#id` selectors in inline styles too, or prefix only colliding ids.
- **Rebasing misses CSS `url()` references.** Likelihood: medium. Impact: background images break. Mitigation: include linked and inline CSS in the rebase.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
**ADR-1: compute the book identity once.** Why: two computations drift by construction.

## 10. Links to other specs
`hotfix-epub-href-containment`, `bugfix-epub-html-content-fidelity`.

## 11. Done criteria (strategic)
1. A Sigil-layout EPUB in default mode shows all its images.
2. Clicking a footnote marker scrolls to the note in the merged page.
3. After a translated multipage conversion, the index offers "Continue reading" for the chapter last read.
4. A TOC click to `#sec7` lands on section 7 even when a saved position exists.
5. An image named `scan#1.png` displays.

## 12. Next step
`/spec-tech 2026-09-24_bugfix-reader-layer-and-single-page`
