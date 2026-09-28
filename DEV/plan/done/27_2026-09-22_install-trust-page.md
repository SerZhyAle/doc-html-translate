# The user meets SmartScreen and nothing we ship answers it

**Status:** Implemented (2026-09-28 - the page is live on the site, the catalog row and its exception are closed)
**Priority:** 51
**Date:** 2026-09-22

> Documentation / user-facing ticket, every authored locale in one edit.
> Contract: `INSTALL-TRUST` 1.0, pointer [`docs/contracts/INSTALL-TRUST.md`](../../docs/contracts/INSTALL-TRUST.md).

> **Remote execution (2026-09-25):** every step ran remote except the catalog step; the contract snapshot
> this note spoke of is deleted - the catalog registry carries the adoption row and the closed exception
> as of 2026-09-28.

## What / why

The product offers three unsigned downloads - the per-user installer
`doc-html-translate-setup-<version>.exe`, the portable CLI and the portable GUI - plus the winget portable
package, and [`installer/doc-html-translate.iss`](../../installer/doc-html-translate.iss) configures no
code signing. Every one of them meets "Windows protected your PC". Searched on 2026-09-22 across every
README, every site page and the GUI strings: no surface contains the word SmartScreen, the quoted dialog,
or any instruction for getting past it. The user is left to guess, and the guess that loses us the user is
the safe one.

`INSTALL-TRUST` binds every product shipping an unsigned or unreputed artifact. This product is bound and
has not adopted it; the gap is declared with a dated exception in the catalog registry rather than left
silent, and this ticket is what closes it.

The Store (MSIX) build is signed by Microsoft at certification and is outside the contract's scope.

Re-verified 2026-09-25: still no trust page - a case-insensitive search for "SmartScreen" and "protected
your PC" over the tracked `*.md`, `*.html`, `*.go` and `*.json` outside `DEV/` hits only the pointer
`docs/contracts/INSTALL-TRUST.md`. `installer/doc-html-translate.iss` still has no signing line, and it sets
`PrivilegesRequired=lowest` (line 42, "no admin / no UAC"), so the installer raises no administrator prompt
and rule 5 below has no elevation to itemize - the page should say the install needs no administrator
rights rather than invent a prompt section.

## Done criteria

- [x] One trust page carrying the contract's four sections **in order**: what the warning is, why it
      appears, exactly what to click, and what the app never does. A reader who reads only the third
      section is unblocked.
- [x] The dialog is quoted verbatim ("Windows protected your PC"), so the user can match what is on their
      screen without reading the page.
- [x] The reason states the truth, cost included: the build is not signed, a certificate costs money, that
      was a choice, and reputation accrues so the warning fades. No implication that the warning is a bug.
- [x] No instruction that weakens a protection - no "turn off SmartScreen", no "disable antivirus". Only
      More info -> Run anyway.
- [x] "What the app never does" agrees with [`privacy.html`](../../privacy.html) and the Store data-safety
      answers.
- [x] Reachable from every place a download is offered: `README.md` + `README_RU.md` + `README_UK.md`,
      `index.html` and the localized site pages, in each authored locale.
- [x] **⛔ Local only - changes the contract catalog.** Done 2026-09-28: the registry row for
      `INSTALL-TRUST` reads adopted (rule by rule, verified against the live page) and the 2026-09-22
      exception is closed, following the FileDO and FastMediaSorter Android closure pattern.

## Implementation record (2026-09-25)

- **The page:** [`install-trust.html`](../../install-trust.html), en/ru/uk in-page like `privacy.html`
  (the three authored locales; the long-form tier of `DEV/DOCS_SURFACES.md`). A lead with the two clicks,
  then the four rule-1 sections in order. The dialog is quoted in English on all three, plus the Russian
  Windows wording on `ru`; the Ukrainian Windows wording was not verified, so `ua` quotes the English and
  says the labels are translated rather than guessing them. The browser "not commonly downloaded" step and
  the one-folder antivirus exception are the only other instructions (both per-file / per-folder, rule 4).
  Rule 5 has no elevation to itemize: the page says the installer asks for no administrator rights.
- **"Never does"** restates `privacy.html` (no personal data, no telemetry/ads/accounts/servers, documents
  leave only with Google translation chosen, logs never uploaded) and adds two code-checked facts: the GUI
  listens on `127.0.0.1` only (`cmd/doc-html-ui/main.go`), and the installer's "Open with" task never
  takes a default handler and is swept on uninstall (`installer/doc-html-translate.iss` `[Registry]`).
- **Links:** `index.html` `#get` note (`id="smartscreen"`), the same note in the ten `<code>/index.html`
  landing pages (machine-translated, pointing at the English page; the dialog labels quoted in English),
  the README trio under "Download", the `docs.*` trio next to "Download the latest release".
- **Registration:** `sitemap.xml` (three hreflang entries), `.sza-canon.json` `site.pages`,
  `tests/site_test.go` `authorPages`, `DEV/DOCS_SURFACES.md`.
- **Check:** `go test ./tests/ -run 'Site|Typography|Parity' -count=1` - `ok`, exit 0. Headless Chrome
  render of `?l=ru` read back.
- **Not done here:** the GUI has no first-run surface to link from (the user has already passed SmartScreen
  by the time it opens). The Partner Center data-safety form agreement is the owner's to confirm.

## Notes

The Store data-safety answers themselves live in Partner Center, not in this repository; the in-repo text a
remote session can check against is [`privacy.html`](../../privacy.html) and the Store feature list in
[`msix/README.md`](../../msix/README.md) ("Open source - no accounts, no telemetry, no ads, no data
collection"). Agreement with the Partner Center form proper is confirmed by the owner.

No code changes. If signing is ever bought, the page is updated, not deleted - users of older builds still
meet the warning and search engines already point at it.
