# Phase 05 — Docs, parity map, and store listing

**Strategic spec:** [`../2026-07-01_extension-format-parity.md`](../2026-07-01_extension-format-parity.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done (store name change flagged for owner sign-off at release)
**Depends on:** Phases 01, 02, 03, 04
**Steps done:** 6 / 6

## Objective
Bring every source of truth for the extension's supported formats up to date - parity map,
READMEs, website, store listing, privacy, localized strings, and changelog - so the next
publication advertises the true format range. This phase is the user's "update all
documentation" step; no parser code changes here.

## Prerequisites
- [ ] Phases 01-04 are ✅ Done (all parsers exist and pass their smoke tests).
- [ ] Owner is available to sign off the store rename (strategic §3.3).

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `docs/PARITY.md` | Modified | ≤ 30 delta |
| `extension/README.md` | Modified | ≤ 20 delta |
| `extension/store/LISTING.md` | Modified | ≤ 60 delta |
| `extension/store/PRIVACY.md` | Modified | ≤ 15 delta |
| `extension/_locales/en/messages.json` | Modified | ≤ 8 delta |
| `extension/_locales/ru/messages.json` | Modified | ≤ 8 delta |
| `extension/_locales/uk/messages.json` | Modified | ≤ 8 delta |
| `README.md` | Modified | ≤ 15 delta |
| `docs.html`, `docs.ru.html`, `docs.uk.html`, `extension.html`, `index.html` | Modified | ≤ 20 delta total |
| `DEV/CHANGELOG.md` | Modified | ≤ 15 delta |

## Steps

### Step 05.1 — Update the Go<->JS parity map
**Files:** `docs/PARITY.md`
**Depends on:** - start of phase

**Prompt for developer:**
> In [`docs/PARITY.md`](../../../docs/PARITY.md) "The port map (Go <-> JS)" table, add one row
> per new format mapping the Go extractor to the JS module: `internal/txt` <-> `src/txt.js`,
> `internal/rtf` <-> `src/rtf.js`, `internal/md` <-> `src/md.js`, `internal/fb2` <->
> `src/fb2.js`, `internal/htmlconv` <-> `src/html.js`, `internal/mobi` (Calibre) <->
> `src/ebook.js` (foliate-js). Add a short note under "Intentional divergences" that the
> extension uses `src/sanitize.js` for new formats (EPUB keeps its own `renderChapter`) and
> vendors `marked` and `foliate-js` where the desktop uses `goldmark` and Calibre.

**Verification:**
- `docs/PARITY.md` contains `src/txt.js`, `src/rtf.js`, `src/md.js`, `src/fb2.js`,
  `src/html.js`, and `src/ebook.js`.
- `foliate-js` appears in `docs/PARITY.md`.

**Status:** `[x]` done

---

### Step 05.2 — Update the extension README
**Files:** `extension/README.md`
**Depends on:** - start of phase

**Prompt for developer:**
> In [`extension/README.md`](../../../extension/README.md), replace the "PDF and EPUB" framing
> with the full supported set (PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, local HTML) and
> list the new `src/*.js` parser modules plus the vendored `marked` and `foliate-js`
> dependencies in the layout/architecture section.

**Verification:**
- `extension/README.md` contains `MOBI`, `AZW3`, `FB2`, `RTF`, and `Markdown`.
- `extension/README.md` contains `foliate-js`.

**Status:** `[x]` done

---

### Step 05.3 — Update the store listing + privacy copy
**Files:** `extension/store/LISTING.md`, `extension/store/PRIVACY.md`
**Depends on:** - start of phase

**Prompt for developer:**
> In [`extension/store/LISTING.md`](../../../extension/store/LISTING.md): change the item name
> away from "PDF & EPUB Page Translator" to a wider-format name (draft, e.g. "Document Page
> Translator - PDF, EPUB, MOBI & more"), update the short and long descriptions to enumerate
> the full format set, and refresh the EN/RU/UK caption blocks. Mark the final name/wording
> **owner-curated at release** (strategic §3.3). In
> [`extension/store/PRIVACY.md`](../../../extension/store/PRIVACY.md), extend any "what the
> extension processes" wording so it covers the new formats and reaffirms that all parsing is
> local (no upload).

**Verification:**
- `extension/store/LISTING.md` no longer describes the format scope as only PDF/EPUB (it
  contains `MOBI` and `FB2`).
- `extension/store/PRIVACY.md` still asserts local-only processing and mentions the wider set.

**Status:** `[x]` done

---

### Step 05.4 — Update localized name/description strings
**Files:** `extension/_locales/en/messages.json`, `extension/_locales/ru/messages.json`, `extension/_locales/uk/messages.json`
**Depends on:** Step 05.3

**Prompt for developer:**
> Update `appName`, `appShortName`, and `appDesc` (the `__MSG_*` keys the manifest references)
> in all three locale files so the extension name/description reflect the wider format range,
> consistent with the LISTING.md name chosen in Step 05.3. Keep RU/UK translations aligned with
> the EN wording. Do not change message key names.

**Verification:**
- All three `messages.json` files contain updated `appDesc` mentioning more than PDF/EPUB
  (e.g. contains `MOBI` or the localized equivalent).
- The set of message keys is unchanged (only values edited).

**Status:** `[x]` done

---

### Step 05.5 — Update root README and website format lists
**Files:** `README.md`, `docs.html`, `docs.ru.html`, `docs.uk.html`, `extension.html`, `index.html`
**Depends on:** - start of phase

**Prompt for developer:**
> Grep the repo for where the extension is described as PDF/EPUB-only (root
> [`README.md`](../../../README.md) editions section; the website pages `docs.html`,
> `docs.ru.html`, `docs.uk.html`, `extension.html`, `index.html`). Update each place that
> enumerates the extension's formats to the full set so the desktop app and the extension read
> as format-equivalent. Do not touch the desktop-only format notes (they are already correct).

**Verification:**
- Root `README.md` extension mention lists the wider set (contains `MOBI` near the extension
  description).
- Each website page that previously said the extension is PDF/EPUB-only now lists the wider set
  (grep for the old phrasing returns zero, or is updated).

**Status:** `[x]` done

---

### Step 05.6 — Changelog headline entry
**Files:** `DEV/CHANGELOG.md`
**Depends on:** Step 05.1, Step 05.2, Step 05.3, Step 05.4, Step 05.5

**Prompt for developer:**
> Add a single headline entry to [`DEV/CHANGELOG.md`](../../../DEV/CHANGELOG.md) summarizing the
> feature: the extension now opens TXT, Markdown, FB2, RTF, local HTML, MOBI and AZW3 in
> addition to PDF and EPUB (client-side, offline; MOBI/AZW3 via vendored foliate-js). Follow the
> existing changelog format and typography (short hyphens, `..` not `...`).

**Verification:**
- `DEV/CHANGELOG.md` has a new entry naming the new formats and `foliate-js`.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 05.*` is `[x] done`.
- [ ] Grep the repo for "PDF & EPUB" / "PDF and EPUB" extension descriptions returns only
      intentional/historical mentions - no stale "extension supports only PDF and EPUB".
- [ ] Grep for `TODO(phase-05)` returns zero hits.
- [ ] See [`INDEX.md`](INDEX.md) Completion gate.
- [ ] Version bump and publishing are deferred to the **release** flow (per `CLAUDE.md` build vs
      release) - do NOT bump `manifest.json`/`package.json` version or push an `ext-v*` tag here.

## Handoff notes
- Store rename is public-facing: the owner signs off the final name/wording during the release
  flow. This phase only stages the copy in-repo.
- After this phase, run `/spec-check 2026-07-01_extension-format-parity` to set the ticket to
  Verified.

## Rollback plan
Revert the phase commit(s). Docs-only - no runtime impact.
