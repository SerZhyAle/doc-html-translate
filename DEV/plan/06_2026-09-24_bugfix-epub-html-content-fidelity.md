# Strategic spec: 06_2026-09-24_bugfix-epub-html-content-fidelity - EPUB, HTML and Markdown content survives normalization and splitting

**Ticket:** 06_2026-09-24_bugfix-epub-html-content-fidelity
**Status:** In Progress - E6, E7, E8, E9, E11, E12 and the EPUB half of E14 landed (c1e9ec7, 274fdb8); the HTML/Markdown half of E14, E15, E21 and E22 are open.
**Priority:** 65
**Date:** 2026-09-24
**Tier:** Strategic
**Tactical plan:** `DEV/plan/06_2026-09-24_bugfix-epub-html-content-fidelity/` (created by /spec-tech)
**Findings:** E6 E7 E8 E9 E11 E12 E14 E15 E21 E22 (see the [findings register](../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
Several text-level rewrites damage books.
- **Cover-image rewrite:** the pattern can span two drawings and replace the text between them with one picture.
- **Link and extension rewrites:** they are blind substring replacements, so they change unrelated URLs and even prose.
- **XML self-closing tags:** XHTML is renamed to HTML without re-serializing, so tags like `<script/>` or `<a id=".."/>` swallow the rest of the chapter in the browser.
- **Splitting of long chapters:**
  - It never happens for the common single-wrapper layout.
  - It counts bytes instead of characters.
  - It drops the language and direction attributes, which breaks RTL books and Chrome's translate offer.
  - It never updates the TOC, or links pointing into the moved parts.
- **HTML and Markdown inputs:**
  - Non-UTF-8 pages come out as mojibake.
  - Responsive images still point at the source folder.
  - Image names can collide, and a symlink can copy files from outside the source tree.
  - The language is hard-coded to English.
  - Page CSS is dropped, and Markdown copies no local images.

## 2. Goals
1. No normalization step can remove or alter book text.
2. Only real link attributes are rewritten, resolved relative to their own file.
3. Every converted page parses in the browser as the author's XHTML did.
4. Long chapters split in every common layout, measured in characters, with language, direction and body attributes preserved on every part.
5. TOC entries and in-book links still land on the right heading after a split.
6. HTML inputs in any declared or BOM-indicated charset are decoded correctly.
7. All local images of HTML and Markdown inputs display, collision-free, and never from outside the source tree.
8. HTML and Markdown outputs carry the source language when known, and keep the source's own styles where safe.

**Non-goals:**
- Single-page merge rebasing (ticket `bugfix-reader-layer-and-single-page`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** Chrome `file://` rendering; the free Chrome-translate flow.
- **Performance:** DOM-based rewrites must stay linear on 5 MB chapters.
- **Data compatibility:** n/a.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `hotfix-epub-href-containment`, `bugfix-reader-layer-and-single-page` (the same link rewrite machinery; build it once).
- **Platform constraints:** cross-edition. Check the extension's EPUB and HTML paths for the same regex and charset behaviour, and record the result in docs/PARITY.md.
- **Validation level:** fixtures: two SVGs with text between, a Calibre `<a id/>` book, a single-wrapper long chapter, an RTL book, a windows-1251 HTML page, a "Save page as" page with `srcset`, and a Markdown file with local images. Checked headless via `/verify-view`.

## 4. Current architecture context
EPUB normalization works on raw text with regular expressions and global string replacement. The
splitter considers only the body's direct children and builds each part from a fixed shell. The
HTML input reader parses bytes as UTF-8, rewrites only `img src`, and drops the head. The Markdown
reader emits a fixed shell.

## 5. Proposed approach
### 5.1 Pillars / modules
- **DOM-based normalization:** the cover-image rewrite and link rewrites run on the parsed tree, not on text.
- **XHTML to HTML serialization:** the chapter is parsed as XML and serialized as HTML, so non-void self-closing elements are expanded.
- **Structure-aware splitting:** the split descends through a sole wrapper, counts characters, clones root and body attributes, and records an id-to-part map used to rewrite the TOC and links.
- **Charset-aware HTML input:** the encoding comes from the BOM, then the meta tag, then detection.
- **Complete asset copying:** `src`, `srcset`, `picture` sources and CSS `url()`; case-insensitive unique naming, reserved generated names avoided, symlinks refused.
- **Source metadata carry-over:** language, and safe head styles.

## 6. Open questions / research items
1. **Source head CSS**
   - **Question:** keep all of the source's styles, or only inline ones?
   - **Options:** keep linked and inline CSS copied locally; keep none (current).
   - **Status:** Open.

## 7. Risks
- **Moving from regexes to DOM rewrites changes output for many books.** Likelihood: high. Impact: visual diffs. Mitigation: a corpus before/after sweep (`/verify-view`), accepting only intended diffs.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
**ADR-1: rewrite on the tree, never on text.** Why: every text-level rewrite here has a proven false-positive case.

## 10. Links to other specs
`hotfix-epub-href-containment`, `bugfix-reader-layer-and-single-page`.

## 11. Done criteria (strategic)
1. The two-SVG fixture keeps its middle paragraphs.
2. The Calibre `<a id/>` fixture renders all its text.
3. A 50 000-character single-wrapper chapter splits, and a TOC entry to its last section opens the right part.
4. An RTL book stays RTL on every part.
5. A windows-1251 HTML page reads correctly.

## 12. Next step
`/spec-tech 06_2026-09-24_bugfix-epub-html-content-fidelity`
