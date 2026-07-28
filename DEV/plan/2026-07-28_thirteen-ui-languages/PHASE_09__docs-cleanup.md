# Phase 09 - Docs cleanup

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** all phases
**Steps done:** 5 / 5

## Objective

Record the new invariants where the next change will trip over them, describe the feature on the
user-facing docs pages, and close the ticket honestly.

## Prerequisites

- [ ] Phases 01-08 ✅ Done.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `docs/PARITY.md` | Modified | ≤ 460 |
| `DEV/DOCS_SURFACES.md` | Modified | ≤ 90 |
| `docs.html` · `docs.ru.html` · `docs.uk.html` | Modified | ≤ 420 / ≤ 240 / ≤ 240 |
| `index.html` | Modified | ≤ 210 |
| `DEV/CHANGELOG.md` | Modified | - |
| `DEV/plan/2026-07-28_thirteen-ui-languages.md` · this plan | Modified | - |

## Steps

### Step 09.1 - Record the invariants in PARITY.md

**Files:** `docs/PARITY.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Add: (1) **UI language set = 13**, one list on both sides - `internal/i18n.Codes` ↔ the
> `extension/_locales` directories - alongside the existing OCR-catalog invariant; (2) `<html lang>` is the
> **content** language, never the UI language, and chrome carries its own `lang`/`dir`; (3) RTL applies to
> chrome only, never to the converted document body; (4) the UI-language resolution order (explicit flag /
> stored override → saved choice → OS → English) in the settings-defaults table. Replace the line calling
> Russian localization a Go-only reader feature - it is no longer true.

**Verification:**
- `internal/i18n` and `_locales` both appear in the invariants section.
- `syslocale.IsRussian` returns zero hits in `docs/PARITY.md`.
- The `<html lang>` rule is stated in one sentence with the reason.

**Status:** `[x] done`

---

### Step 09.2 - Rewrite the docs-surfaces manifest

**Files:** `DEV/DOCS_SURFACES.md`
**Depends on:** Step 09.1

**Prompt for developer:**
> The manifest currently states "3-language parity is mandatory" and "en/ru/uk is one atomic edit". Rewrite
> the table and the invariants to say which surfaces are 13-language (GUI dictionary, extension `_locales`,
> landing pages, store listing sources, screenshots) and which stay 3-language by decision (full docs trio,
> release notes, READMEs), citing D3 and D4. Add the new files: `<code>/index.html`, `README_RU/UK.md`,
> `tools/store/listing/<code>.txt`, `sitemap.xml`.

**Verification:**
- The table contains rows for the per-language landing pages and the listing sources.
- The phrase "3-language parity is mandatory" no longer appears unqualified.
- Every file created in Phases 06 and 08 appears in the manifest.

**Status:** `[x] done`

---

### Step 09.3 - Document the feature for users

**Files:** `docs.html`, `docs.ru.html`, `docs.uk.html`, `index.html`
**Depends on:** Step 09.2

**Prompt for developer:**
> Add a section to the docs trio describing the 13 interface languages, how the language is chosen (OS by
> default, `-ui-lang` on the CLI, the selector in the GUI and in the extension), which surfaces are
> translated and which stay English, and the machine-translation disclosure. Lead the landing hero with the
> feature in all three in-page languages - a new user-facing feature belongs in the hero, not in a collapsed
> section.

**Verification:**
- All three docs files contain a heading naming the 13 languages, and the three are mirrored in content.
- `-ui-lang` is documented in the CLI flag table of all three.
- The hero of `index.html` mentions the feature in ru, en and ua copy.

**Status:** `[x] done`

---

### Step 09.4 - Changelog

**Files:** `DEV/CHANGELOG.md`
**Depends on:** Step 09.3

**Prompt for developer:**
> Add one entry covering the feature with a line per modified area: CLI + converted page, GUI, extension,
> installer, site, listings, screenshots, guards. Use `/changelog` rather than writing the format by hand.

**Verification:**
- The changelog contains an entry mentioning `-ui-lang`, `internal/i18n` and `_locales`.
- Every file listed in Phases 01-08 "Files touched" is covered by some line of that entry.

**Status:** `[x] done`

---

### Step 09.5 - Close the plan

**Files:** `DEV/plan/2026-07-28_thirteen-ui-languages.md`, `INDEX.md`, `DEV/plan/ROADMAP.md`
**Depends on:** Step 09.4

**Prompt for developer:**
> Flip the strategic spec's `Status:` to `Implemented` (not `Verified` - that is `/spec-check`'s call),
> mark the INDEX phases table complete, and update the ROADMAP row. State plainly in the spec what was not
> done if any edition row ended as `Declined`.

**Verification:**
- The strategic spec's first `**Status:**` line reads `Implemented` with the date.
- `INDEX.md` shows `Phases: 9 / 9 done`.
- The ROADMAP row for this ticket matches the spec's status.

**Status:** `[x] done` - closed as **Partial**, not `Implemented`: step 07.5 and the extension long-description half of 08.7 are open, and a status has to come from the tree rather than from this checklist.

## Step log

- 2026-07-28 - 09.1 `docs/PARITY.md` gains the language-set invariant, the resolution order for both
  editions, and the one-sentence `<html lang>` rule with its reason. `syslocale.IsRussian` no longer
  appears anywhere in the file, and the "Russian localization is Go-only" line is gone - it stopped being
  true in phase 05.
- 2026-07-28 - 09.2 `DEV/DOCS_SURFACES.md` rewritten around two explicit tiers instead of "3-language
  parity is mandatory": which surfaces take all 13 languages, which stay en/ru/uk and why (D3, D4). Every
  file created in phases 06 and 08 is in the table, including the ones that came out partial.
- 2026-07-28 - 09.3 the docs trio gains an "interface languages" section (how the language is chosen, what
  it does *not* touch) and the `-ui-lang` row that was missing from all three flag tables. The landing hero
  leads with the feature in ru/en/ua - verified by a headless render at 1366 px.
- 2026-07-28 - 09.4 one summary changelog entry over the listings and docs work, on top of the per-phase
  entries.
- 2026-07-28 - 09.5 closed as **Partial**. Two things are open and the status says so rather than reading
  `Implemented` off a checklist: the extension screenshot generator (07.5) and the extension's long store
  description (08.7, a recorded decision rather than an omission).
- 2026-07-28 - 07.5 was attempted once more here before closing: deriving the unpacked extension ID from
  the SHA-256 of its absolute path (both UTF-16LE and UTF-8 forms) and loading it with
  `--load-extension --headless=new`. Chrome returned an empty DOM for every `chrome-extension://` URL, so
  the extension is not loading in that mode at all. The route needs a real CDP session; leaving it open.

## Phase done criteria

- [ ] Every `Step 09.*` is `[x] done`.
- [ ] `./scripts/test.ps1` and `npm test` both exit 0 on the final tree.
- [ ] Grep for `TODO(phase-0*)` across the repo returns zero hits.
- [ ] See INDEX.md Completion gate.

## Handoff notes

See INDEX.md Completion gate. Next action after this phase is `/spec-check 2026-07-28_thirteen-ui-languages`,
which sets `Verified` from reality rather than from this checklist.

## Rollback plan

Documentation-only phase - revert the phase commit.
