# The user meets SmartScreen and nothing we ship answers it

**Status:** Draft
**Priority:** 51
**Date:** 2026-09-22

> Documentation / user-facing ticket, every authored locale in one edit.
> Contract: `INSTALL-TRUST` 1.0, pointer [`docs/contracts/INSTALL-TRUST.md`](../../docs/contracts/INSTALL-TRUST.md).

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

## Done criteria

- [ ] One trust page carrying the contract's four sections **in order**: what the warning is, why it
      appears, exactly what to click, and what the app never does. A reader who reads only the third
      section is unblocked.
- [ ] The dialog is quoted verbatim ("Windows protected your PC"), so the user can match what is on their
      screen without reading the page.
- [ ] The reason states the truth, cost included: the build is not signed, a certificate costs money, that
      was a choice, and reputation accrues so the warning fades. No implication that the warning is a bug.
- [ ] No instruction that weakens a protection - no "turn off SmartScreen", no "disable antivirus". Only
      More info -> Run anyway.
- [ ] "What the app never does" agrees with [`privacy.html`](../../privacy.html) and the Store data-safety
      answers.
- [ ] Reachable from every place a download is offered: `README.md` + `README_RU.md` + `README_UK.md`,
      `index.html` and the localized site pages, in each authored locale.
- [ ] The catalog registry row for `INSTALL-TRUST` moves from "not adopted" to adopted, and the exception
      is closed rather than re-dated.

## Notes

No code changes. If signing is ever bought, the page is updated, not deleted - users of older builds still
meet the warning and search engines already point at it.
