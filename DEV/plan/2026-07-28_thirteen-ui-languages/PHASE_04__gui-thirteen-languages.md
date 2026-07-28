# Phase 04 - GUI in thirteen languages

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 7 / 7

## Objective

Take `doc-html-ui` from 3 languages to 13, including right-to-left layout and the system fonts the Indic
and CJK scripts need, without letting `ui.html` grow past a maintainable size.

## Prerequisites

- [ ] Phase 03 ✅ Done (the GUI already forwards `-ui-lang`).
- [ ] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `cmd/doc-html-ui/ui.html` | Modified | ≤ 1150 (shrinks by ~180: the dictionary leaves) |
| `cmd/doc-html-ui/i18n.js` | New | ≤ 1400 (13 × 87 keys) |
| `cmd/doc-html-ui/main.go` | Modified | ≤ 1060 |
| `cmd/doc-html-ui/main_test.go` | Modified | ≤ 420 |

> `ui.html` is 1318 lines and the dictionary would add ~900 more. Step 04.1 extracts it **before** any
> language is added - do not add languages in place.

## Steps

### Step 04.1 - Extract the dictionary into its own served file

**Files:** `cmd/doc-html-ui/i18n.js`, `cmd/doc-html-ui/ui.html`, `cmd/doc-html-ui/main.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Move the `const I18N = { en, ru, uk }` block out of `ui.html` into `cmd/doc-html-ui/i18n.js` verbatim,
> `//go:embed i18n.js` it next to the existing `ui.html` embed, serve it at `/i18n.js`, and load it from
> `ui.html` with a `<script src="/i18n.js"></script>` placed before the code that calls `t()`. Behaviour
> must be unchanged in this step - no new languages, no key edits.

**Verification:**
- `//go:embed i18n.js` matches exactly once in `main.go` and a handler serves `/i18n.js`.
- `const I18N` returns zero grep hits in `ui.html` and exactly one in `i18n.js`.
- The GUI starts and the language buttons still switch between en/ru/uk.

**Status:** `[x] done`

---

### Step 04.2 - Replace the three buttons with an endonym selector

**Files:** `cmd/doc-html-ui/ui.html`
**Depends on:** Step 04.1

**Prompt for developer:**
> Replace the `#langSwitch` three-button group with a `<select id="langSwitch">` populated at load from
> `I18N` (value = code, label = endonym from a new `ENDONYMS` map in `i18n.js`). Keep the existing
> behaviour: a change saves to settings, calls `applyLang`, and the saved value wins over the system
> language.

**Verification:**
- `<select id="langSwitch"` matches exactly once in `ui.html`.
- `data-lang="ru"` button markup returns zero grep hits.
- `ENDONYMS` is declared in `i18n.js` with 13 entries.

**Status:** `[x] done`

---

### Step 04.3 - Add the five Latin-script languages

**Files:** `cmd/doc-html-ui/i18n.js`
**Depends on:** Step 04.2

**Prompt for developer:**
> Add complete `de`, `it`, `es`, `fr`, `pt` dictionaries - all 87 keys, no empty values, `{placeholder}`
> tokens preserved exactly. Portuguese is the single neutral translation from strategic decision D1.
> Latin first, deliberately: it surfaces layout problems (German is the longest) before the harder scripts.

**Verification:**
- `i18n.js` declares 8 language objects.
- Each new object has the same key count as `en` (87).
- No value in the new objects is the empty string.

**Status:** `[x] done`

---

### Step 04.4 - Add Hindi, Bengali and Chinese with their font stacks

**Files:** `cmd/doc-html-ui/i18n.js`, `cmd/doc-html-ui/ui.html`, `cmd/doc-html-ui/main.go`
**Depends on:** Step 04.3

**Prompt for developer:**
> Add complete `hi`, `bn`, `zh` dictionaries. Extend `/api/env` to return the resolved language and its
> font family from `i18n.FontFamily`, and have `applyLang` set `document.body.style.fontFamily` to
> `Nirmala UI` for hi/bn and `Microsoft YaHei UI` for zh, falling back to the current stack otherwise.
> These fonts ship with Windows; do not add a webfont - the GUI runs offline.

**Verification:**
- `i18n.js` declares 11 language objects, each with 87 non-empty values.
- `Nirmala UI` and `Microsoft YaHei UI` both appear in the GUI sources.
- `/api/env` response includes a `font` (or equivalently named) field.

**Status:** `[x] done`

---

### Step 04.5 - Add Arabic and Urdu with right-to-left layout

**Files:** `cmd/doc-html-ui/i18n.js`, `cmd/doc-html-ui/ui.html`
**Depends on:** Step 04.4

**Prompt for developer:**
> Add complete `ar` and `ur` dictionaries and make `applyLang` set `document.documentElement.dir`. Check
> and fix under mirroring: the header row, the `.langs` row, the drop zone, the copy boxes, and the swap
> button - its `⇄` glyph and its meaning must still agree once the row is mirrored.

**Verification:**
- `i18n.js` declares 13 language objects, each with 87 non-empty values.
- `documentElement.dir` is assigned inside `applyLang`.
- Manual: the GUI opened in `ar` shows the header controls mirrored and the swap arrow pointing sensibly
  (evidence: the screenshot produced in Phase 07).

**Status:** `[x] done`

---

### Step 04.6 - Make the GUI language reach the converter

**Files:** `cmd/doc-html-ui/ui.html`, `cmd/doc-html-ui/main.go`
**Depends on:** Step 04.5

**Prompt for developer:**
> Send the current interface language with the run request and pass it through to the CLI as `-ui-lang`
> (the argument plumbing landed in Step 03.8; this step makes the value follow the switcher rather than the
> OS). The command preview shown in the GUI must include the flag so what is shown is what runs.

**Verification:**
- The run-request JSON shape includes the interface language field, and `assembleArgs` reads it.
- `updateCmdPreview` output contains `-ui-lang` when the language differs from the system default.
- `go test ./cmd/doc-html-ui -count=1` exits 0.

**Status:** `[x] done`

---

### Step 04.7 - Guard the dictionaries with a test

**Files:** `cmd/doc-html-ui/main_test.go`
**Depends on:** Step 04.6

**Prompt for developer:**
> Add a test that reads the embedded `i18n.js`, extracts the top-level language object names and each
> object's keys, and asserts: exactly the 13 codes of `i18n.Codes`, identical key sets across all of them,
> no empty values, and every `data-i18n*` attribute used in `ui.html` present in the `en` object. A missing
> key silently falls back to English at runtime, which is why this has to be static.

**Verification:**
- `func TestGUIDictionariesCoverEveryLanguage` matches exactly once.
- `go test ./cmd/doc-html-ui -count=1` exits 0.

**Status:** `[x] done`

## Phase done criteria

- [x] Every `Step 04.*` is `[x] done`.
- [x] `go test ./cmd/doc-html-ui ./tests -count=1` exits 0.
- [x] `./scripts/build-ui.ps1` succeeds (this phase changes embedded resources).
- [x] Grep for `TODO(phase-04)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Step log

- 2026-07-28 - 04.1 dictionary extracted verbatim to `cmd/doc-html-ui/i18n.js` + `/i18n.js` handler; ui.html 1318 -> ~1050 lines.
- 2026-07-28 - 04.2 `<select id="langSwitch">` built from `I18N` + `ENDONYMS`; the three `data-lang` buttons are gone.
- 2026-07-28 - 04.3 de/it/es/fr/pt added, 88 keys each, verified with a node key-set diff.
- 2026-07-28 - 04.4 hi/bn/zh added; `FONT_STACKS` applied in `applyLang`; `/api/env` reports `lang` + `font`.
- 2026-07-28 - 04.5 ar/ur added; `documentElement.dir` follows the language. Visual RTL confirmation is deferred to the phase 07 screenshots, as the step states.
- 2026-07-28 - 04.6 `uiLang` rides in the run request and reaches the CLI as `-ui-lang`; preview uses the same assembleArgs.
- 2026-07-28 - 04.7 `i18n_test.go`: 13 identical key sets, every markup key defined, no empty value. All PASS.
- 2026-07-28 - phase check: `go test ./cmd/doc-html-ui ./tests` exit 0, `./scripts/build-ui.ps1` OK.

## Handoff notes

The GUI is now the reference for what a fully localized surface looks like: dictionary out of the markup,
endonym selector, `dir` from the language, system font per script. Phase 05 mirrors this in the extension
with `chrome.i18n` instead of a hand-rolled dictionary.

## Rollback plan

Revert the phase commit(s). If only the RTL work misbehaves, dropping the `ar`/`ur` objects from `i18n.js`
returns the GUI to 11 languages without touching the mechanism.
