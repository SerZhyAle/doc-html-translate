# Strategic spec: 2026-09-24_hotfix-epub-href-containment - Book-supplied paths stay inside the book

**Ticket:** 2026-09-24_hotfix-epub-href-containment
**Status:** Draft
**Priority:** 95
**Date:** 2026-09-24
**Tier:** Security/Compliance (urgent)
**Tactical plan:** `DEV/plan/2026-09-24_hotfix-epub-href-containment/` (created by /spec-tech)
**Findings:** E1 E5 E10 E20 B19 B20 (see `../README.md`)

> **Scope:** STRATEGIC. No class names, paths, line budgets, schema versions, or framework module details.

---

## 1. Problem
An EPUB names its own files in its package document: the manifest, the spine, and the container
pointer. The archive-extraction step blocks `..` in archive entry names, but the later steps trust
the package document's names completely. A crafted book whose spine points at `../something` makes
the default single-page mode read a file **next to the book** into the page and then **delete it**.
In multipage mode that outside file is rewritten instead. Separately, legitimate books with
percent-encoded names (`Chapter%201.xhtml`) or case-only name differences fail or lose chapters,
in both editions.

## 2. Goals
1. No book-supplied path can make either edition read, write or delete anything outside the book's own extracted tree.
2. Percent-encoded manifest names resolve to the right files in both editions.
3. One unresolvable manifest item skips that item with a warning, and the rest of the book still converts.
4. Generated file names never overwrite a book file that differs only in letter case.
5. The extension resolves root-relative and `%23`-containing hrefs correctly.

**Non-goals:**
- Content fidelity of the merge itself (ticket `bugfix-reader-layer-and-single-page`).

## 3. Wishes and constraints
### 3.1 Owner wishes
- One resolution rule, shared in spirit by both editions and pinned in the parity doc.

### 3.2 Hard constraints
- **Platform / versions:** Windows semantics (case-insensitive, backslashes, drive letters, UNC) must be covered even when the tests run on another OS.
- **Performance:** n/a.
- **Data compatibility:** none. Existing outputs are unaffected.
- **Localization:** the warning text goes through i18n.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `hotfix-output-dir-ownership` (the same containment principle for the output root).
- **Platform constraints:** cross-edition. Go and JS are fixed in the same ticket, and docs/PARITY.md is updated.
- **Validation level:** a malicious-fixture test per vector (spine, manifest, container pointer, absolute path, drive letter, UNC), plus the parity test.
- **Owner sign-off:** required for the security fix.

## 4. Current architecture context
Archive extraction has a containment guard. Every later consumer - XHTML normalization, the
single-page merge, navbar injection, splitting, the index generator - turns a manifest name into a
disk path by plain joining, with no decoding and no containment check. The single-page merge then
removes each spine file it absorbed. The extension edition keys its manifest by the raw name, while
its link and image code decodes names first.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Single resolution gate:** when the package document is parsed, each manifest, spine and container name is decoded once, normalized once, and checked to be inside the book root. An item that fails is dropped with a warning. Every later step uses only these pre-resolved names.
- **Case-aware reserved names:** comparisons against the names the converter generates (the index, the merged page) ignore case, and a case-only collision between archive entries is detected and reported.
- **Per-item tolerance:** normalization continues past a missing item instead of aborting the book.
- **Extension parity:** the same decode-then-resolve order, root-relative handling, and fragment and query split before decoding.

### 5.2 Data & event flows
Package document -> resolution gate (decode, normalize, contain) -> clean item list -> all later stages.

### 5.3 Extension points
- The gate is the only way a book name becomes a path, so future stages inherit the protection.

## 6. Open questions / research items
1. **Out-of-tree spine item policy**
   - **Question:** should dropping it fail the whole conversion or only that item?
   - **Options:** skip with a warning (preferred); refuse the book.
   - **Status:** Open.

## 7. Risks
- **Over-strict normalization drops valid items (for example `./` prefixes).** Likelihood: medium. Impact: missing chapters. Mitigation: a corpus sweep over the local test books before and after the change.
- **The editions diverge again.** Likelihood: medium. Impact: parity drift. Mitigation: a shared fixture list in the parity test.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
**ADR-1: resolve once at parse time.** Alternatives: guarding every consumer. Why: a single gate cannot be forgotten by the next new stage.

## 10. Links to other specs
`hotfix-output-dir-ownership`, `bugfix-reader-layer-and-single-page`.

## 11. Done criteria (strategic)
1. A crafted EPUB whose spine or manifest points outside the book leaves every file outside the output untouched, and its content does not appear in the page.
2. A book with `Chapter%201.xhtml` in its manifest converts with all chapters, in both editions.
3. A book with one missing manifest item converts, and a warning names that item.
4. A book containing `Index.xhtml` keeps that chapter.

## 12. Next step
`/spec-tech 2026-09-24_hotfix-epub-href-containment`
