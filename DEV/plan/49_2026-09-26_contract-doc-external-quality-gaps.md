# Published documentation meets DOC-EXTERNAL-QUALITY

**Status:** Partial
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

## Implementation (2026-09-26)

Done as Go tests under `tests/`, so `scripts/test.ps1` runs them inside `scripts/check.ps1`; no new
PowerShell check and no new `configs/check-placement.jsonl` row. Shared parsing is
`tests/site_html_test.go` (the `golang.org/x/net/html` tree, not regex over markup).

- **Rule 7 - link integrity: gated.** `tests/site_links_test.go` resolves every `href`, `src`,
  `srcset`, `poster` and `data-href` (the docs language switch) plus `og:url`, `og:image` and
  `twitter:image` on the 18 pages. An address on this site written in full is mapped back to its file, so
  the canonical, hreflang and share-card addresses are held too. Fragments are checked against the ids of
  the page they name, same-page and cross-page. `TestSiteNoPlainHTTP` fails any `http://`. External hosts
  are not fetched (offline suite). Mutation-checked: a renamed page and a missing anchor both fail.
- **Rule 5 - SEO length: gated, every page shortened, nothing allowlisted.** `tests/site_seo_test.go`
  holds `<title>` <= 60 and meta description <= 160 code points after entity decoding, og:title equal to
  `<title>` (as it was on all 18), twitter:title and the og/twitter descriptions within the same limits,
  and the `window.SITE` strings that `assets/site.js` switches in by language (the trio pages' ru/en/ua,
  the landings' `page`). Shortened: index (en/ru/ua), extension (en/ru/ua titles; the static description
  296 -> 158, the three script descriptions), extension-privacy (description 214 -> 159, ru/ua titles and
  descriptions), nine landings (all but zh). Each keeps its language and meaning. Counting note: the
  adoption scan's "14 of 18" does not reproduce in code points - 11 static titles were over, plus 6
  script-switched ones; the unit question went to the proposal (item 4).
- **Rule 3 - translation freshness: gated.** `tests/site_l10n_test.go`. Every localized page carries
  `<!-- l10n-source: <english page> sha256:<16 hex> -->` after its charset meta: the ten landings name
  `index.html`, `docs.ru.html`/`docs.uk.html` name `docs.html`, the five in-page trio pages name
  themselves (their ru/ua blocks sit beside the English). The hash covers the source's `<title>`, meta
  description and the English of `<main>`; `TestEnglishFingerprintScope` pins that an English edit moves
  it and a Russian one does not. Verdict decision: a missing, doubled or wrongly-sourced stamp **fails**;
  a stale one is an **advisory** (the fan-out stays at the release boundary): the test logs
  `advisory: ..`, and `scripts/test.ps1` now collects those lines, names them and ends in
  `test: PASS WITH ADVISORIES` (exit 3), so `check.ps1` does too and a release, which wants a bare PASS,
  stops on a stale locale. Re-stamp: `go test ./tests -run TestSiteTranslationFreshness -update-l10n`.
  The first stamps were written against today's English; this is honest because git shows every target
  was last written after (landings, 2026-09-26 01:53) or with (docs trio, `26605b9`) its source's last
  English change, and this change edits sources and targets together.
- **Rule 4 - termbase: gated.** `configs/termbase.json` (13 locales): convert, translate, OCR overlay,
  browser extension, desktop app, and the proper names Microsoft Store, Chrome Web Store, Edge Add-ons
  (locale `*`). `tests/site_termbase_test.go` checks the shape, that each `use` stem occurs in its locale's
  pages (no dead entries) and that no `avoid` variant does - word-start match for Latin and Cyrillic,
  substring for scripts that attach particles or have no spaces (`TestContainsTerm`). It found and this
  change fixed: "оверлей" in docs.ru/docs.uk ("слой текста" / "шар тексту"), "десктоп" in docs.ru,
  docs.uk and extension.html ("настольное приложение" / "настільний застосунок").
- **Rule 1 - subject index and glossary: done in all three languages.** `docs*.html` gain
  `#subject-index` (32 entries) and `#glossary` (12 terms), linked from the hero line; styling in
  `assets/site.css`. `tests/site_docs_test.go`: every section of `<main>` is reachable from the index, each
  glossary term has a definition, and every `glossary: true` termbase term heads a `<dt>` in that
  language.
- **Rule 6 - visual evidence: done for the workflow, recorded for the rest.** The recommended workflow
  shows steps 1 and 2 with the product's own captures, per language: `tools/store/gui-<loc>.png` (the
  app window) and `tools/store/table-of-contents-<loc>.png` (the converted book), regenerated by
  `tools/store/make-*.ps1` and already on the site as og:image. `TestDocsWorkflowCarriesCaptures` pins
  image, alt, size, caption and language. Not captured, and why: step 3 is the browser's own Translate
  menu, which a page capture cannot show (proposal item 3); the "sending logs" task's About section shows
  the machine's own state (edition, engine found, OCR languages), so a capture from this Linux container
  would show a state no Windows user sees - the owner can add one on Windows with
  `tools/store/make-gui-screenshot.ps1`; Quick Start is commands, already text. No new binary was added.
- **Portal-shaped rules: proposal drafted, not filed.**
  `docs/contracts/proposals/PROPOSAL-2026-09-26-doc-html-translate-doc-external-quality.md` (search
  index, task pages, third-party UI steps, fingerprint form and length unit). It sits in a subfolder
  because `scripts/contract-gate.ps1` reads every `docs/contracts/*.md` as a pointer. The pointer
  `docs/contracts/DOC-QUALITY.md` names the new checks and the draft.
- **Registry.** New records `contract-proposals` and `termbase` in `docs/DOCUMENT_REGISTRY.jsonl`. The set
  of pages did not change, so `sitemap.xml` needs no regeneration.

Open, for the owner:
- Run `scripts/check.ps1` on Windows once. Already run on Linux (pwsh 7.4.6, tree `9038e08`):
  `doc-registry.ps1` with the two new records - `PASS (41 record(s), 247 document file(s) covered,
  18 page(s) announced)`; `scripts/test.ps1` with `GOOS` left native (the Windows pin cannot execute
  here) - clean tree `PASS (run=1018 skipped=3)` exit 0, an English edit in `index.html` `PASS WITH
  ADVISORIES (.. advisories=11)` exit 3. That run found the advisories dropped unnamed when `test_doc/`
  is absent (exit 2 outranks 3); `scripts/test.ps1` now names them before any non-FAIL verdict.
- File the proposal in the catalog (`documentation-quality/PROPOSAL-2026-09-26-doc-html-translate.md`)
  and replace the draft with a line in the pointer.
- The shared registry row cannot be re-verified from this session (catalog not reachable); re-verify it
  and close the exception row once the checks above pass. That item keeps this ticket at Partial.
