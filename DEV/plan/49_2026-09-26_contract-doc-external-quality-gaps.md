# Published documentation meets DOC-EXTERNAL-QUALITY

**Status:** Draft
**Priority:** 45
**Date:** 2026-09-26

> Contract sync ticket. Contract: `DOC-EXTERNAL-QUALITY` 0.9 draft (domain `documentation-quality/`, owner
> FastMediaSorter Android). Pointer: [`docs/contracts/DOC-QUALITY.md`](../../docs/contracts/DOC-QUALITY.md).
> Requires `DOC-INTERNAL-QUALITY` first - ticket [48](48_2026-09-26_contract-doc-internal-quality-gaps.md).
> Adopted 2026-09-26; the gaps below are the dated exception in the shared registry until this closes.

## What / why

This product publishes: the GitHub Pages site (landing, ten locale landings, the `docs*.html` trio,
extension, privacy and install-trust pages), the README trio and the store listings. The adoption run of
2026-09-26 read the seven rules against it.

Held or partly held:

- **Rule 2 - mirroring and build: partial.** The site pages are the authored source (static HTML, no
  markdown render step), so no rendered HTML is hand-edited; the rendered parts are gated -
  `sitemap.xml` from the registry (`scripts/doc-registry.ps1`), the privacy blocks from
  `docs/security-posture.json` (`scripts/security-posture.ps1`). There is no search index.
- **Rule 5 - SEO: partial.** `doc-registry.ps1` fails a page without title, description, canonical, Open
  Graph with an image, Twitter card, JSON-LD, one `h1`, its hreflang cluster, or its sitemap entry. It does
  not check length: **14 of 18 pages have a `<title>` over 60 characters** (the landing 75, the extension
  page 81, nine locale landings 64-80) and **2 descriptions are over 160** (`extension.html` 296,
  `extension-privacy.html` 214).
- **Rule 7 - link integrity: held, not gated.** 0 broken internal `href`/`src` targets or same-page
  anchors and 0 `http://` references across the 18 pages on 2026-09-26; nothing fails a new one.

Not met:

- **Rule 1 - task orientation.** `docs*.html` has task sections (recommended workflow, quick start,
  editions) but no subject index and no glossary.
- **Rule 3 - translation freshness.** en/ru/uk are authored together and the other ten landings are fanned
  out, but nothing records which source revision a translation was made from, so a stale locale is
  invisible.
- **Rule 4 - termbase.** No glossary of product terms per locale and no gate against forbidden synonyms.
- **Rule 6 - visual evidence.** The multi-step guides carry no screenshot on any page.

## Done when

- A check (with CHECK-VERDICT exit codes, run by `scripts/check.ps1`) resolves every internal link and
  anchor on every announced page, and the SEO check enforces the title and description lengths - each
  over-long page shortened or recorded with a reason.
- Every localized page or block carries a source fingerprint of the English text it was made from, and a
  check flags the stale ones (flagging is enough; the fan-out may stay at the release boundary).
- A per-locale termbase exists for the key terms (convert, translate, OCR overlay, edition names) with a
  check against forbidden variants.
- `docs*.html` gains a subject index and a glossary; its multi-step tasks carry current screenshots or
  language-neutral captures.
- Where a rule does not fit a single-page product site (a search index, a full portal), a proposal to the
  contract owner is filed beside the contract instead of a local waiver.
- The shared registry row is re-verified and the exception row closed.

## Evidence (2026-09-26)

- `pwsh -NoProfile -File scripts/doc-registry.ps1` - exit 0, 18 pages announced, SEO block and sitemap held.
- Title/description lengths, link and screenshot counts: a throwaway scan of the 18 pages in the adoption
  session (not committed).
