# Phase 07 - Docs cleanup

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** all phases
**Steps done:** 8 / 8

## Objective

Every surface in [`DEV/DOCS_SURFACES.md`](../../../DOCS_SURFACES.md) that this feature touches says the
same true thing in every language it ships in, including the privacy wording and the parity record.

## Prerequisites

- [x] Phases 01-06 are ✅ Done.
- [~] The feature has been exercised once by hand, so the documented wording matches reality. **Not
      met as written.** The wording was instead derived from the shipped source - `report.Build`,
      `report.Environment`, `report.Redact`, `store.go`'s two bounds, `ui.html`'s three buttons and
      `diagnostics.js`'s field list - and every documented value was read out of the code rather than
      observed running. The four hands-on checks in INDEX.md's completion gate are what still closes
      this, and they are the owner's to run.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `README.md` · `README_RU.md` · `README_UK.md` | Modified | ≤ +25 each |
| `docs.html` · `docs.ru.html` · `docs.uk.html` | Modified | ≤ +30 each |
| `index.html` | Modified (132) | ≤ +25 |
| `{de,it,es,fr,pt,ar,hi,bn,ur,zh}/index.html` × 10 | Modified | ≤ +12 each |
| `extension.html` | Modified | ≤ +15 |
| `privacy.html` · `extension-privacy.html` · `extension/store/PRIVACY.md` | Modified | ≤ +20 each |
| `docs/PARITY.md` | Modified | ≤ +25 |
| `DEV/CHANGELOG.md` | Modified | ≤ +3 rows |

## Steps

### Step 07.1 - Document the flag and the button in the three READMEs

**Files:** `README.md`, `README_RU.md`, `README_UK.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Add a `-report` row to the flag table in all three READMEs and one feature bullet describing the
> About section's "Send logs to the author" button: what the archive contains (recent run logs, an
> environment summary, the last run's settings), that the user attaches it and presses Send, and
> that nothing is sent automatically. Keep the three files mirrored - same rows, same order, only
> the prose translated. House typography applies to all three.

**Verification:**
- `-report` appears in the flag table of each of the three files.
- `grep -c 'report' README_RU.md` and `README_UK.md` both return non-zero for the new bullet.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x]` done

---

### Step 07.2 - Document it in the docs trio

**Files:** `docs.html`, `docs.ru.html`, `docs.uk.html`
**Depends on:** Step 07.1

**Prompt for developer:**
> Add a "Sending logs to the author" section to all three docs pages, in lockstep: where the button
> is, what the archive contains field by field, where the archive is written, the size cap and that
> the oldest logs are dropped first, the clear-logs control, the `-report` flag for the command
> line, and the extension's copy-diagnostics button as the browser-edition equivalent. State
> plainly that the app never sends anything by itself.

**Verification:**
- The new section heading appears exactly once in each of the three files.
- Each of the three mentions both `-report` and the extension's button.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x]` done

---

### Step 07.3 - Add the site section in thirteen languages

**Files:** `index.html`, `{de,it,es,fr,pt,ar,hi,bn,ur,zh}/index.html`
**Depends on:** Step 07.2

**Prompt for developer:**
> Add a short "Something went wrong?" block to the root landing page in all three in-page languages
> and to each of the ten per-language pages: one sentence that the app can pack its own logs and
> open a pre-addressed mail to the author, and that nothing leaves the machine until the user sends
> it. Place it near the support/contact area, **not** in the hero - see the note in "Deliberate
> departure" below. On `ar` and `ur` keep the product name and any command inside `dir="ltr"`
> islands, as the existing pages do. Do not touch `sitemap.xml` - no page is added.

**Verification:**
- The new block's marker id or class appears exactly once in `index.html` and once in each of the
  ten per-language pages (11 files).
- `ar/index.html` and `ur/index.html` keep every new Latin token inside a `dir="ltr"` element.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x]` done

---

### Step 07.4 - Mention the extension's button on the extension page

**Files:** `extension.html`
**Depends on:** Step 07.3

**Prompt for developer:**
> Add one sentence in each of the page's three in-page languages: the options page can copy a short
> technical summary for a bug report, and it contains no document text.

**Verification:**
- The new sentence appears three times (en/ru/uk) in `extension.html`.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x]` done

---

### Step 07.5 - Keep the privacy statements true

**Files:** `privacy.html`, `extension-privacy.html`, `extension/store/PRIVACY.md`
**Depends on:** Step 07.4

**Prompt for developer:**
> Add a paragraph to each: the app can build an archive of its own logs **on the user's request**,
> the archive stays on the user's disk, the app never uploads it, and the mail is composed in the
> user's own mail program and sent by the user. Name what the archive contains and that the
> translation API key is excluded. For the extension: the copy-diagnostics button writes to the
> clipboard only, adds no permission, and collects no document content. Keep the existing
> "no telemetry" claim and make it precise rather than removing it.

**Verification:**
- Each of the three files contains the new paragraph and still contains its existing no-telemetry
  statement.
- No file claims the app sends anything automatically (read the new paragraphs).

**Status:** `[x]` done

---

### Step 07.6 - Record the parity invariant and the divergence

**Files:** `docs/PARITY.md`
**Depends on:** Step 07.5

**Prompt for developer:**
> Add a shared-invariant row for the report's field list (the fields common to the desktop
> `environment.txt` and the extension's `reportText`, so the two cannot drift), with the Go side
> pointing at `internal/report/environment.go` and the JS side at `extension/src/diagnostics.js`.
> Add an "Intentional divergences" entry: the desktop editions build a log archive and hand off to a
> mail program, the extension copies a text summary, because the browser sandbox hosts no run-log
> store (strategic ADR-4). Leave the existing "Feedback" row alone - the address is unchanged.

**Verification:**
- `internal/report/environment.go` and `extension/src/diagnostics.js` both appear in
  `docs/PARITY.md`.
- The divergences table has one new row naming this feature.

**Status:** `[x]` done

---

### Step 07.7 - Add a guard against a silent drift of the two reports

**Files:** `tests/parity_test.go`
**Depends on:** Step 07.6

**Prompt for developer:**
> Add `TestParityReportFields` in the style of the existing `TestParityReflowConstants`: read
> `internal/report/environment.go` and `extension/src/diagnostics.js` and assert the shared field
> labels documented in step 07.6 appear in both. Fail with a message pointing at
> `docs/PARITY.md`.

**Verification:**
- `go test ./tests -run TestParityReportFields` exits 0.
- `TestParityReportFields` matches exactly once.

**Status:** `[x]` done

---

### Step 07.8 - Changelog

**Files:** `DEV/CHANGELOG.md`
**Depends on:** Step 07.7

**Prompt for developer:**
> Append changelog rows via `/changelog` covering every file touched by phases 01-07, grouped by
> phase, in the table's existing shape (timestamp, paths, target, description). Record the two
> facts a reader would otherwise have to re-derive: that no attachment is made programmatically and
> why, and that the extension's shape is different by decision.

**Verification:**
- `DEV/CHANGELOG.md` contains a row naming `internal/report/archive.go` and a row naming
  `extension/src/diagnostics.js`.
- `go test ./tests -run TestTypography` exits 0 (the changelog is English prose under the house
  rule).

**Status:** `[x]` done

## Deliberate departure - confirmed by the owner 2026-08-11

`DEV/DOCS_SURFACES.md` and the `feature-prominence-3-lang` memory say a new user-facing feature
**leads the landing hero**. This feature is deliberately placed in a support block instead: the hero
sells what the product does to documents, and "send us your logs" in that slot reads as an admission
of unreliability. **The owner confirmed the support-block placement on 2026-08-11**, so step 07.3
landed the block near the documentation/contact area of all eleven pages and the hero is untouched.
This is the one standing exception to the hero rule for this ticket; it does not generalize.

## Out of scope, with reason

- **Store listing sources** (`tools/store/listing/<code>.txt` × 13), **winget locale manifests** and
  the **Partner Center CSV**: the feature list and "what's new" copy in those files is curated at
  release time by the `release` skill, against the version actually shipping. Adding a line here
  would be overwritten by that flow.
- **App and extension screenshots**: the About section is not part of the captured frames
  (`make-gui-screenshot.ps1` shoots the conversion view), so no regeneration is due.
- **`DEV/DOCS_SURFACES.md`**: no surface file is added or renamed by this ticket.

## Phase done criteria

- [x] Every `Step 07.*` is `[x] done`.
- [x] `./scripts/check.ps1` exits 0 (lint + tests + typo gate over the new prose). 2026-08-11: "All
      checks passed / Tests passed / Lint passed / Typo check passed / parity-check: no cross-edition
      drift in the change set", exit 0.
- [x] `npm test` in `extension/` exits 0. 2026-08-11: 93 pass, 0 fail.
- [x] Every surface listed in "Files touched" greps for the content it was supposed to gain (the
      doc-only rung of `DEV/research/VALIDATION.md`). 11/11 landings carry `id="support"`, the three
      privacy files carry the new paragraph and still carry their no-telemetry claim.
- [x] Grep for `TODO(phase-07)` returns zero hits (only the plan files' own checkbox text matches).

## Step log

- 2026-08-11 - **07.1 pre-resolved.** All three READMEs already carried the `-report` row and the
  feature bullet, uncommitted in the working tree from an earlier session that did not update this
  file. Verified rather than rewritten: `-report` present in each flag table, the bullet present in
  RU and UK, `go test ./tests -run TestTypography` exit 0.
- 2026-08-11 - **07.2 pre-resolved**, same origin. Each of the three docs pages carries the section
  heading exactly once and names both `-report` and the extension's copy-diagnostics button.
- 2026-08-11 - **07.3 done.** `id="support"` note added to `index.html` (en/ru/ua in-page) and to each
  of the ten per-language pages: 11 files, one marker each. The `ar` and `ur` blocks contain no Latin
  token in visible text, so no `dir="ltr"` island was required; verified by stripping tags and
  scanning for `[A-Za-z]`. `sitemap.xml` untouched.
- 2026-08-11 - **07.4 done.** One sentence per in-page language appended to `extension.html`'s privacy
  note: `Скопировать диагностику` / `Copy diagnostics` / `Скопіювати діагностику`, one occurrence each.
- 2026-08-11 - **07.5 done.** `privacy.html` gains "Sending logs to the author"; `extension/store/PRIVACY.md`
  and its rendered twin `extension-privacy.html` gain "Diagnostics you copy yourself", written into the
  Markdown source first and mirrored. All three keep their no-telemetry claim and none of the new prose
  claims anything is sent by the app. "Last updated" moved to 2026-08-11 on all three.
- 2026-08-11 - **07.6 done.** PARITY.md gains the "Report field labels" invariant (producer, the five
  shared labels, what is edition-specific) and an "Intentional divergences" entry for the archive-vs-clipboard
  split. Both `internal/report/environment.go` and `extension/src/diagnostics.js` are linked.
- 2026-08-11 - **07.7 done.** `TestParityReportFields` added to `tests/parity_test.go`, one definition,
  `go test ./tests -run TestParityReportFields` exit 0. It matches each label with its opening delimiter,
  so `ocr` cannot pass on the strength of `ocr languages`.
- 2026-08-11 - **07.8 done.** One changelog row for the whole phase; phases 01-06 already had theirs. It
  records the two facts the plan asked for: why no attachment is made programmatically, and that the
  extension's shape differs by decision (ADR-4).

## Handoff notes

See INDEX.md Completion gate - the four hands-on checks are what remain after this phase.

## Rollback plan

Revert phase commit(s); documentation-only, no runtime effect.
