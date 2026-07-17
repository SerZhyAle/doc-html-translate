# Tactical plan: 2026-07-01_extension-format-parity - extension-format-parity

**Strategic spec:** [`../2026-07-01_extension-format-parity.md`](../2026-07-01_extension-format-parity.md)
**Research inputs:** [`DEV/research/extension_formats_feasibility_ru.md`](../../../research/extension_formats_feasibility_ru.md)
**Tier:** Strategic · **Priority:** 50
**Status:** Verified - closed 2026-07-17; production use accepted as the manual acceptance (see the strategic spec)
**Phases:** 5 / 5 done
**Last updated:** 2026-07-01

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview
| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | foundation | - | ✅ Done | 5/5 | [PHASE_01__foundation.md](PHASE_01__foundation.md) |
| 02 | text-rtf-html | 01 | ✅ Done | 5/5 | [PHASE_02__text-rtf-html.md](PHASE_02__text-rtf-html.md) |
| 03 | markdown-fb2 | 01 | ✅ Done | 5/5 | [PHASE_03__markdown-fb2.md](PHASE_03__markdown-fb2.md) |
| 04 | ebook-mobi-azw3 | 01 | ✅ Done | 5/5 | [PHASE_04__ebook-mobi-azw3.md](PHASE_04__ebook-mobi-azw3.md) |
| 05 | docs-cleanup | 01,02,03,04 | ✅ Done | 6/6 | [PHASE_05__docs-cleanup.md](PHASE_05__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

## Pre-implementation blockers
- None blocking Phase 01. The one open research item (minimal vendored foliate module set)
  is resolved by a spike **inside** Phase 04 (Step 04.1) and does not gate earlier phases.

## Completion gate
- [ ] All phases ✅ Done.
- [ ] User-facing docs updated (strategic §8 mandates it) - Phase 05.
- [ ] `DEV/CHANGELOG.md` has an entry per modified file.
- [ ] `/spec-check 2026-07-01_extension-format-parity` returns Verified.

## How to track progress
1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on
   intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Architecture seam (read once before Phase 01)
Every format parser returns the same render-ready **book** shape that
[`epub.js`](../../../../extension/src/epub.js) `loadEpub` already returns:

```
{ title, lang, sampleText, sections: [ { id, label, frag } ], toc, revoke }
```

- `frag` is a sanitized `DocumentFragment`; `toc` is `[ { title, anchor, children } ]`
  (anchor = an element id inside a section) or `[]`.
- [`viewer.js`](../../../../extension/src/viewer.js) `renderEpubDocument` already renders this
  shape generically. Phase 01 renames it `renderBook` and routes all formats through it.
- New parsers turn source HTML into a `frag` via the shared `src/sanitize.js` helper created
  in Phase 01 (they do **not** reuse epub.js `renderChapter`, which stays EPUB-specific).

## Change log
- 2026-07-01 - initial tactical plan authored by /spec-tech.
