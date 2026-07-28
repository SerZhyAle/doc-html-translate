# Phase 07 - Release screenshots per language

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** 🚧 In Progress
**Depends on:** Phase 03, Phase 04, Phase 05
**Steps done:** 4 / 5

## Objective

Turn `tools/store/make-screenshot.ps1` from a two-frame English capture into a reproducible generator of
13 × 3 app frames plus the extension frames, driven by `-ui-lang` and the extension's language override.

## Prerequisites

- [x] Phases 03, 04, 05 ✅ Done.
- [x] A CLI build available at `msix/staging/` or `build/` (the script's existing discovery).
- [x] Output goes to the repo's gitignored `temp/` while iterating - never a deep scratchpad path, which
      breaks OCR through MAX_PATH and fakes a quality bug.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `tools/store/make-screenshot.ps1` | Modified | ≤ 400 |
| `tools/store/make-gui-screenshot.ps1` | New | ≤ 220 |
| `extension/scripts/make-screenshots.mjs` | New | ≤ 260 |
| `tools/store/*.png` | Regenerated | - |
| `extension/store/screenshots/*.png` | Regenerated | - |

## Steps

### Step 07.1 - Add the `-Language` parameter and per-locale file names

**Files:** `tools/store/make-screenshot.ps1`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `[string[]]$Language` defaulting to all 13 codes. For each language, run the CLI conversion with
> `-ui-lang <code>` and capture the reading view and the table of contents, writing
> `reading-view-<store-locale>.png` and `table-of-contents-<store-locale>.png` (Partner Center locale
> codes: `en-us`, `ru`, `uk`, `de`, `it`, `es`, `fr`, `pt-br`, `ar`, `hi`, `bn`, `ur`, `zh-hans`). Keep the
> existing sample-document and headless-Edge machinery; only the loop and the naming are new.

**Verification:**
- `param(` block contains `[string[]]$Language`.
- `-ui-lang` appears in the CLI invocation.
- A run with `-Language en,ar` produces exactly 4 PNGs named with `en-us` and `ar`.

**Status:** `[x] done`

---

### Step 07.2 - Capture the GUI window per language

**Files:** `tools/store/make-gui-screenshot.ps1`
**Depends on:** Step 07.1

**Prompt for developer:**
> Add a script that starts `doc-html-ui`, drives its language selector (or seeds the saved setting) to each
> of the 13 languages, and captures the window at 1366×768 into `gui-<store-locale>.png`. Restore the
> developer's own saved GUI settings at the end whatever happens - this walks over real settings, and
> leaving the app in Bengali after a crash is not a fair trade for a screenshot.

**Verification:**
- The script exists and accepts the same `-Language` parameter shape as Step 07.1.
- A restore path runs in a `finally` block.
- A run with `-Language de` produces `gui-de.png` at 1366×768 or larger.

**Status:** `[x] done`

---

### Step 07.3 - Assert the right-to-left frames are correct

**Files:** `tools/store/make-screenshot.ps1`, `tools/store/make-gui-screenshot.ps1`
**Depends on:** Step 07.2

**Prompt for developer:**
> For `ar` and `ur`, assert before writing the PNG that the captured DOM had `dir="rtl"` on the chrome and
> not on the content container, and fail the run with a named error otherwise. We capture a browser page,
> not a `WS_EX_LAYOUTRTL` window - do not port CyrFlip's flip-the-bitmap workaround; a mirrored *image* here
> would be a bug, not a fix.

**Verification:**
- Both scripts contain an explicit `ar`/`ur` assertion path that can `throw`.
- Deliberately breaking the chrome `dir` makes the run fail rather than silently produce a shot.

**Status:** `[x] done`

---

### Step 07.4 - Regenerate the app screenshots

**Files:** `tools/store/*.png`
**Depends on:** Step 07.3

**Prompt for developer:**
> Run both scripts for all 13 languages and commit the resulting PNGs. Delete the two obsolete
> language-less files (`reading-view.png`, `table-of-contents.png`) once every listing reference points at
> a locale-suffixed name.

**Verification:**
- `tools/store/` contains 39 PNGs (13 × 3) with locale-suffixed names.
- `reading-view.png` and `table-of-contents.png` no longer exist.
- Each PNG is at least 1366×768.

**Status:** `[x] done`

---

### Step 07.5 - Regenerate the extension screenshots

**Files:** `extension/scripts/make-screenshots.mjs`, `extension/store/screenshots/*.png`
**Depends on:** Step 07.4

**Prompt for developer:**
> Add a script that loads the built extension in headless Chrome, sets the interface-language override from
> Step 05.6, opens the sample PDF and the sample EPUB, and captures `shot1-<code>.png` and
> `shot2-<code>.png` for all 13 languages. Replace the four hand-made en/ru files.

**Verification:**
- `extension/store/screenshots/` contains 26 PNGs.
- The script is referenced from `extension/package.json` scripts.
- `shot1-en.png` and `shot1-ru.png` are regenerated (newer mtime than the commit before this phase).

**Status:** `[ ]` not done - see the step log; the four hand-made en/ru PNGs are still in place and the
listing still has images, so nothing is broken, but the ten new languages have no extension frames.

## Step log

- 2026-07-28 - 07.1 `-Language` (13 codes, all by default) drives `-ui-lang` per run; output is named by
  Partner Center locale (`en-us`, `pt-br`, `zh-hans`, the rest plain). Each language converts into its own
  work directory so a failed run cannot mix frames.
- 2026-07-28 - 07.3 `Assert-ChromeDirection` runs **before** each capture: the chrome must carry
  `lang="<code>"`, must mirror exactly for `ar`/`ur`, and the document root must never be `dir="rtl"`.
  Split content pages carry no chrome, so the function reports that instead of failing, and the loop
  throws if a language ended up with nothing verified. No bitmap flipping - a mirrored *image* would be
  the bug, not the fix.
- 2026-07-28 - 07.2 `make-gui-screenshot.ps1`: `doc-html-ui` is a local HTTP app, so the capture starts it,
  finds its ephemeral port through the OS connection table (the app never prints it), and shoots
  `?lang=<code>`. That override is new in `ui.html`, reads only from the URL and is never persisted -
  which is what keeps a 13-language run from overwriting the developer's own saved language. Settings are
  still backed up and restored in a `finally`.
- 2026-07-28 - 07.4 39 PNGs in `tools/store/` (13 × reading view + TOC + GUI), all 1366×768; the two
  language-less files are deleted and nothing outside `DEV/plan/` referenced them.
- 2026-07-28 - **07.5 not done.** Capturing the extension needs the viewer running under a real
  `chrome-extension://` origin - `chrome.i18n` and `chrome.storage` do not exist on a `file://` page - and
  that means either a Puppeteer dependency the extension does not have, or a hand-rolled CDP client. It was
  left out rather than half-built. The four existing `shot1/shot2-{en,ru}.png` stay valid, so the listing
  keeps its images; the ten new languages simply have no extension frames yet.

## Phase done criteria

- [ ] Every `Step 07.*` is `[x] done`. - 4 / 5; 07.5 open.
- [ ] Both capture scripts run end-to-end from a clean `temp/` with no manual step.
- [ ] Grep for `TODO(phase-07)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes

Every listing locale now has an image to point at - which is what makes a Partner Center listing count as
complete. Phase 08 consumes these filenames directly in the import folder.

## Rollback plan

Revert the phase commit(s); the previous PNGs return with them. The scripts are additive and safe to keep
even if the images are rolled back.
