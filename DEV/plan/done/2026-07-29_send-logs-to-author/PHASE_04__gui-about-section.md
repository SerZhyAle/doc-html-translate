# Phase 04 - GUI About section

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 5 / 5

## Objective

The settings page carries an "About the program" section that states what is installed and turns
one press into: archive built, mail program open with the author's address and a prefilled
message, archive revealed with its path on the clipboard.

## Prerequisites

- [ ] Phase 03 is ✅ Done.
- [ ] `cmd/doc-html-ui/ui.html` (1064) and `i18n.js` (1201) are committed before editing - the
      tree is the backup for files this size.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `cmd/doc-html-ui/i18n.js` | Modified (1201) | ≤ 1420 |
| `cmd/doc-html-ui/ui.html` | Modified (1064) | ≤ 1200 |
| `cmd/doc-html-ui/i18n_test.go` | Modified | ≤ 200 |

## Steps

### Step 04.1 - Add the dictionary keys in all thirteen languages

**Files:** `cmd/doc-html-ui/i18n.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Add these keys to **every** language object in `I18N`, in the order `en ru uk de it es fr pt ar
> hi bn ur zh` the file already uses: `secAbout` ("About the program"), `aboutVersion`,
> `aboutEdition`, `aboutEditionPackaged`, `aboutEditionPortable`, `aboutEngine`, `aboutLogsHeld`
> (takes `{1}` files and `{2}`), `btnSendLogs` ("Send logs to the author"), `sendLogsBuilding`,
> `sendLogsReady`, `sendLogsAttachHint` ("The archive is selected in the folder and its path is on
> the clipboard - attach it to the message before sending."), `sendLogsFailed`, `sendLogsDropped`
> (takes `{1}`), `btnOpenArchive`, `btnClearLogs`, `logsCleared`, `mailBody` (the message
> template: one line stating this is a log report, one blank line, "What went wrong:", one blank
> line, and the attach reminder). English first, then the twelve translations; no key may be left
> empty. Follow the house typography rule for `en`/`ru`/`uk` only (`..` never `...`, plain
> hyphen, Russian `ё`); the other ten follow their own script.

**Verification:**
- `go test ./cmd/doc-html-ui -run TestGUIDictionaries` exits 0 (it fails on any language missing a
  key).
- `grep -c 'secAbout' cmd/doc-html-ui/i18n.js` returns 13.
- `grep -c 'btnSendLogs' cmd/doc-html-ui/i18n.js` returns 13.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x]` done - 2026-08-11. 19 keys × 13 languages = 247 lines added; `secAbout`,
`btnSendLogs`, `mailBody` and `sendLogsAttachHint` each match 13 times;
`TestGUIDictionariesCoverEveryLanguage` and `TestTypography` exit 0. **Two keys beyond the
prompt's list:** `aboutEngineFound` / `aboutEngineMissing` - step 04.3 renders the engine line
from a boolean, and without them that line would be hardcoded English, which step 04.2 forbids.
Placeholders use the plan's `{1}`/`{2}` convention; `t()` splits on `{k}` for any key, so this
works unchanged, though the rest of the file uses named placeholders.

---

### Step 04.2 - Add the About section markup

**Files:** `cmd/doc-html-ui/ui.html`
**Depends on:** Step 04.1

**Prompt for developer:**
> After the `integrationSection` `<details>` block, add `<details id="aboutSection">` with a
> `<summary data-i18n="secAbout">`, a definition-style block of version / edition / engine /
> logs-held lines (ids `aboutVer`, `aboutEd`, `aboutEng`, `aboutLogs`), the existing product-site
> and feedback links, and three buttons: `btnSendLogs`, `btnOpenArchive` (hidden until an archive
> exists) and `btnClearLogs`, plus a `<div class="hint" id="sendLogsMsg">` for the result text.
> Tag every visible string with `data-i18n`; do not hardcode English anywhere in the block.
> Accessibility (strategic §3.2): use real `<button>` elements so the block is keyboard reachable
> without a `tabindex`, carry every outcome as text in `sendLogsMsg` rather than by colour alone,
> and give that element `role="status"` so the result is announced.

**Verification:**
- `id="aboutSection"` appears exactly once in `cmd/doc-html-ui/ui.html`.
- All three controls are `<button` elements, none carries a `tabindex`, and `sendLogsMsg` carries
  `role="status"`.
- `data-i18n="secAbout"`, `data-i18n="btnSendLogs"`, `id="btnOpenArchive"`, `id="btnClearLogs"` and
  `id="sendLogsMsg"` each appear exactly once.
- `go test ./cmd/doc-html-ui -run TestGUIMarkupKeysExist` exits 0 (no markup key without a
  translation).

**Status:** `[x]` done - 2026-08-11. All ids and `data-i18n` attributes match once, all three
controls are real `<button>` elements, no `tabindex` anywhere in the file, `sendLogsMsg` carries
`role="status"`, and `TestGUI*` exit 0. Two more keys were needed to keep the "no hardcoded
English" rule: `aboutSite` / `aboutFeedback` for the two links (the header byline's own copies
stay untranslated - they are the existing chrome, not this block). The feedback link's `href` is
filled from `/api/env` at load, so `sza@ukr.net` still occurs exactly once in the file.

---

### Step 04.3 - Populate the section from /api/env

**Files:** `cmd/doc-html-ui/ui.html`
**Depends on:** Step 04.2, Phase 03 Step 03.3

**Prompt for developer:**
> In the page script, extend the existing `/api/env` consumer to fill `aboutVer` from
> `/api/version`, `aboutEd` from `packaged` (`aboutEditionPackaged` / `aboutEditionPortable`),
> `aboutEng` from `cli`, and `aboutLogs` from the new `logs` field via `t('aboutLogsHeld', n,
> size)`. Store the `author` field in a page-level variable - the mail address must come from
> `/api/env`, not from a second literal in the markup.

**Verification:**
- `aboutLogsHeld` and `aboutEditionPackaged` both appear in the script section of `ui.html`.
- The string `sza@ukr.net` appears in `ui.html` **exactly once** (the existing byline link) - the
  new code reads the address from `/api/env`.

**Status:** `[x]` done - 2026-08-11. Both keys appear once in the script section and `sza@ukr.net`
still occurs exactly once. The values are kept as raw facts in `aboutFacts` and rendered by
`renderAbout()`, which `applyLang` now calls: written once imperatively, the edition and engine
lines would stay in the language that happened to be current when `/api/env` answered.

---

### Step 04.4 - Wire the button: build, mail, reveal, clipboard

**Files:** `cmd/doc-html-ui/ui.html`
**Depends on:** Step 04.3

**Prompt for developer:**
> On `btnSendLogs`: disable the button and show `sendLogsBuilding`; `POST /api/report`; on failure
> show `sendLogsFailed` with the error and stop. On success, in this order: copy the archive path
> with the page's existing clipboard helper (the one with the hidden-textarea fallback), `POST
> /api/report-reveal` with the path, then set `location.href` to a `mailto:` built from the
> `author` address, subject `doc-html-translate <version> - log report`, and body
> `t('mailBody')`, all through `encodeURIComponent`. Finally show `sendLogsReady` plus
> `sendLogsAttachHint`, and append `sendLogsDropped` when `dropped > 0`. Reveal the
> `btnOpenArchive` button and point it at `/api/report-open`. Wire `btnClearLogs` to
> `/api/logs-clear` and show `logsCleared`. **Never claim the archive was attached** - the only
> wording after a success is the attach hint.

**Verification:**
- `'/api/report'`, `'/api/report-reveal'`, `'/api/report-open'` and `'/api/logs-clear'` each appear
  exactly once in `ui.html`.
- `mailto:` appears in the new handler and the body is built from `t('mailBody')`.
- `encodeURIComponent` appears at least three times in the new handler (address, subject, body).
- No string containing `attached` (past tense) exists in `cmd/doc-html-ui/i18n.js`.

**Status:** `[x]` done - 2026-08-11. The four routes match once each, `mailto:` is built in the new
handler from `t('mailBody')`, `encodeURIComponent` wraps address, subject and body, and `attached`
occurs nowhere in the dictionary. The page's clipboard code was extracted into
`copyToClipboard(text)` (the prompt calls it "the existing helper"; it was inline in the
copy-command handler) so both callers share one path including the hidden-textarea fallback. The
success line is stored as `lastReport` and re-rendered by `renderAbout`, so a language switch does
not leave it frozen.

---

### Step 04.5 - Guard the new strings

**Files:** `cmd/doc-html-ui/i18n_test.go`
**Depends on:** Step 04.4

**Prompt for developer:**
> Add `TestGUIAboutKeysPresent`: assert `secAbout`, `btnSendLogs`, `sendLogsAttachHint` and
> `mailBody` exist in all thirteen dictionaries and that no dictionary's `sendLogsAttachHint` is
> empty. Add `TestGUIMailAddressComesFromEnv`: read `ui.html` and fail if `sza@ukr.net` occurs
> more than once, so the address cannot be re-hardcoded in the new handler.

**Verification:**
- `go test ./cmd/doc-html-ui` exits 0.
- `TestGUIAboutKeysPresent` and `TestGUIMailAddressComesFromEnv` each match exactly once.

**Status:** `[x]` done - 2026-08-11. `go test ./cmd/doc-html-ui` exit 0; both names match once. The
attach-hint check had to tolerate a trailing `\r` - the working tree is CRLF and a bare `$` anchor
matched nothing, which is exactly how a guard passes for the wrong reason.

## Phase done criteria

- [x] Every `Step 04.*` is `[x] done`.
- [x] `./scripts/build-ui.ps1` exits 0 - this phase changes embedded resources, so a build is the
      lowest sufficient rung.
- [x] `go test ./cmd/doc-html-ui ./tests -run 'TestGUI|TestTypography'` exits 0.
- [x] RTL check: **measured, not asserted.** The built exe was started against a temp
      `%LOCALAPPDATA%` and its page dumped with headless Chrome at `?lang=ar`: the document opens
      as `<html lang="ar" dir="rtl">`, the whole About block is Arabic, it carries no `dir` of its
      own (so it mirrors with the chrome), and `aboutMail` had `href="mailto:sza@ukr.net"` - proof
      the address arrives from `/api/env` rather than the markup. `POST /api/report` against the
      same live server returned `ok` with a readable 341-byte archive whose `environment.txt` held
      the expected eight fields.
- [x] Grep for `TODO(phase-04)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Established: the About section, thirteen-language strings, and the rule that the author's address
reaches the page only through `/api/env`. Phase 07 documents this UI; phase 05 must not duplicate
the mail composition on the CLI side - the CLI prints a path, it does not open mail.

## Rollback plan

Revert phase commit(s). The endpoints from phase 03 keep working unused.
