# Pointer: DOC-INTERNAL-QUALITY, DOC-EXTERNAL-QUALITY

- **Id:** `DOC-INTERNAL-QUALITY`, `DOC-EXTERNAL-QUALITY`
- **Version:** 0.9 (both, draft)
- **Home:** the shared contracts catalog, `documentation-quality/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer of both - internal engineering docs, and a published site, README trio and store listings
- **Wire carrier:** none - a documentation tree and the checks that hold it

What this repo does to stay conformant:
- `docs/DOCUMENT_REGISTRY.jsonl` declares every maintained document (product areas, update triggers,
  `generated` for a render); `scripts/doc-registry.ps1` holds it both ways, checks each announced page's SEO
  block and hreflang cluster, and keeps `sitemap.xml` equal to its render. Runs in `scripts/check.ps1`.
- Rendered texts come from one source and are gated: `scripts/security-posture.ps1` (privacy blocks,
  `docs/SECURITY_POSTURE.md`), `tests/iconography_test.go` (`docs/GLYPH-MAP.md`), `tests/limits_parity_test.go`.
- House text style in code and UI strings: `tests/typography_test.go`; on site pages: `tests/site_test.go`.

**Adopted 2026-09-26 with gaps**, each a ticket and a dated exception in the shared registry:
internal rules 3-6 in [ticket 48](../../DEV/plan/48_2026-09-26_contract-doc-internal-quality-gaps.md),
external rules 1-7 in [ticket 49](../../DEV/plan/49_2026-09-26_contract-doc-external-quality-gaps.md).
