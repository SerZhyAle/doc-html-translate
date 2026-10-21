# Pointer: DOC-INTERNAL-QUALITY, DOC-EXTERNAL-QUALITY

- **Id:** `DOC-INTERNAL-QUALITY`, `DOC-EXTERNAL-QUALITY`
- **Version:** 0.10 / 0.12 (draft)
- **Home:** the shared contracts catalog, `documentation-quality/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer of both - internal engineering docs, and a published site, README trio and store listings; the internal rules are `partial` at 0.10 (read 2026-10-20, ticket 120)
- **Wire carrier:** none - a documentation tree and the checks that hold it

What this repo does to stay conformant:
- `docs/DOCUMENT_REGISTRY.jsonl` declares every maintained document (product areas, update triggers,
  `generated` for a render); `scripts/doc-registry.ps1` holds it both ways, checks each announced page's SEO
  block and hreflang cluster, and keeps `sitemap.xml` equal to its render. Runs in `scripts/check.ps1`.
- Every repository `.md` (internal rules 3, 5-7), in the Go suite that `scripts/check.ps1` runs:
  `tests/doc_links_test.go` (relative links, heading anchors, image paths, no `http://`, no remote
  script), `tests/typography_test.go` (house style in Markdown prose), `tests/readme_flags_test.go`
  (the README trio's flag tables against `internal/config/flags.go`, rule 4).
- Rendered texts come from one source and are gated: `scripts/security-posture.ps1` (privacy blocks,
  `docs/SECURITY_POSTURE.md`), `tests/iconography_test.go` (`docs/GLYPH-MAP.md`), `tests/limits_parity_test.go`.
- House text style in code and UI strings: `tests/typography_test.go`; on site pages: `tests/site_test.go`.
- External rules on the announced pages, as Go tests (run by `scripts/test.ps1` inside `scripts/check.ps1`):
  internal links and anchors resolve, no `http://` (`tests/site_links_test.go`, rule 7); title <= 60 and
  description <= 160 code points, script-switched strings included (`tests/site_seo_test.go`, rule 5);
  a source fingerprint on every localized page, stale = advisory (exit 3), missing = fail
  (`tests/site_l10n_test.go`, rule 3); the termbase `configs/termbase.json` with forbidden variants
  (`tests/site_termbase_test.go`, rule 4); subject index, glossary and workflow captures on the docs trio
  (`tests/site_docs_test.go`, rules 1 and 6).
- The portal-shaped rule questions are filed with the owner in the shared catalog as
  `documentation-quality/PROPOSAL-2026-09-26-doc-html-translate.md`.

**Read at 0.10 / 0.12 on 2026-10-20 (ticket 120).**
- External 0.12: the single-URL page language query is `lang`, `l` is read as legacy, and hreflang lists only translated
  addressable URLs - done in the resolvers, the hreflang sets, the sitemap and the registry's `localized_urls`.
  Item F (a page's embedded medium has its language disclosed) holds as a permanent variant: the demo video has
  burned-in English captions on the RU and UK pages, disclosed in the figcaption (design record
  [`../PARITY.md`](../PARITY.md) "Contract variants (ticket 120)"). Items H and I do not apply.
- Internal 0.10 (a local, git-ignored corpus under its own registry): **closed 2026-10-21, ticket 122 item 7.** The
  tracked registry names nothing in `DEV/plan/` or `DEV/private/` (`scripts/doc-registry.ps1` fails on a path there; the
  old `spec-method` record is now `spec-lifecycle` and holds `docs/SPEC_LIFECYCLE.md` only). The two trees have a
  git-ignored registry beside them, `DEV/DOCUMENT_REGISTRY.jsonl`, and `scripts/doc-registry-local.ps1` holds it in
  both directions (every record resolves, every `.md` / `.html` of the trees is claimed) and runs the same link scanner
  over the local documents (`tests/doc_links_local_test.go`, opt-in). It is a hand-run check: the corpus exists only on
  the owner's machine, so on a clone it ends in COULD NOT VERIFY.

**Adopted 2026-09-26; re-verified 2026-09-28.** Internal rules 3-6 were closed in
[ticket 48](../../DEV/plan/done/48_2026-09-26_contract-doc-internal-quality-gaps.md). External rules
1-7 were implemented in [ticket 49](../../DEV/plan/done/49_2026-09-26_contract-doc-external-quality-gaps.md).
Both dated exceptions in the shared registry are closed; the contract owner has the proposal above for
the single-page site questions.
