# The user meets SmartScreen and nothing we ship answers it

**Status:** In Progress (repo half done 2026-09-25; left: publish the site, then the ⛔ local catalog step)
**Priority:** 51
**Date:** 2026-09-22

> Documentation / user-facing ticket, every authored locale in one edit.
> Contract: `INSTALL-TRUST` 1.0, pointer [`docs/contracts/INSTALL-TRUST.md`](../../docs/contracts/INSTALL-TRUST.md).

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

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
- [ ] **⛔ Local only - changes the contract catalog.** The catalog registry row for `INSTALL-TRUST` moves
      from "not adopted" to adopted, and the exception is closed rather than re-dated (both rows quoted
      below). Runs after the page is merged and published.

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

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - INSTALL-TRUST 1.0. A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

From the shared contracts catalog, `install-trust/README.md`:

> ## 1. The rules
>
> 1. **Four sections, in this order**: what the warning is, why it appears, exactly what to click, and what
>    the app never does. A user who reads only the third section must still be unblocked.
> 2. **Quote the dialog.** The heading is the text the user is looking at right now - "Windows protected your
>    PC", "Do you want to allow this app to make changes?" - so they can match it without reading the page.
> 3. **State the real reason, including the cost.** The build is not signed, a certificate costs money, and
>    that was a choice. Never imply the warning is a mistake, a bug, or something the user should not be
>    seeing. Reputation accrues; say that too, because it is true and it explains why the warning fades.
> 4. **Never tell a user to weaken their protection.** No "turn off SmartScreen", no "disable your
>    antivirus", no "run as administrator to make it go away". The only instructions permitted are
>    per-app and reversible: More info -> Run anyway, or an exception for one install folder.
> 5. **Every elevation is itemized and undoable.** What the administrator prompt is for, one line per thing
>    it does, and the single action that removes all of them together. A product that asks for elevation and
>    cannot say exactly what it did with it has already lost the argument.
> 6. **"What the app never does" is a factual list**, and it says the same thing as the privacy page, the
>    permission list and the store data-safety form. Contradicting them here is worse than silence.
>    (canon invariant 14)
> 7. **The page ships with the product and is reachable from where the download is** - the site page, the
>    README, and the first-run surface where there is one.
> 8. **When the product becomes signed, the page is updated, not deleted.** Users of older builds still meet
>    the warning, and the page is what search engines already point at.
>
> ## 2. Localization
>
> This is user-facing text, so it follows the canon's localization rules: it is authored in English and
> translated wherever the product's other user-facing surfaces are. The contract page you are reading stays
> English only.
>
> ## 3. Conformance
>
> A product conforms when its shipped trust page carries all four sections of rule 1 and contains no
> instruction of the kind rule 4 forbids. Both are readable from the document itself, which is the point of
> a contract at this level.

The reference rendering - a model for shape and tone, not text to copy (it describes another product,
whose installer does elevate) - from the shared contracts catalog, `install-trust/TRUST_GUIDE.md`, in full:

> # Why Windows warns about this app - and what it needs
>
> FastMediaSorter Companion is a small free tool from an independent developer. This
> build is not code-signed - a signing certificate costs hundreds of dollars per year,
> and we chose to skip it for now. Windows treats unsigned apps with extra caution,
> so you will see a few prompts. Here is what they mean.
>
> ## SmartScreen: "Windows protected your PC"
>
> This appears the first time you run an unsigned app that Windows has not seen often.
>
> 1. Click **More info**.
> 2. Click **Run anyway**.
>
> That is all. The warning does not mean the app is harmful - only that it is new
> and unsigned. As more people use it, the warning fades on its own.
>
> ## "Do you want to allow this app to make changes?" (administrator prompt)
>
> The companion asks for administrator rights once, during setup, to do two things:
>
> - **Install the background service** - so your folders stay shared after a reboot
>   without you starting the app by hand.
> - **Add a firewall rule** - so your phone can reach this PC. Without it, Windows
>   silently blocks incoming connections and the share never works.
>
> Both are standard steps any server-like app performs. You can remove them at any
> time with the "Uninstall service" button - it deletes the service and the firewall
> rule together.
>
> ## Antivirus flags
>
> Some antivirus tools score unsigned new binaries cautiously. If yours quarantines
> the app, add an exception for the install folder. Reputation accrues slowly for
> unsigned software - this improves over time and disappears once builds are signed.
>
> ## What the app never does
>
> - It never sends your files anywhere by itself - your phone connects directly to
>   your PC over an encrypted SFTP connection.
> - It never opens your folders to the internet without telling you - the status
>   screen always shows exactly what is reachable and from where.

The two rows the local step changes, from the shared contracts catalog, `_meta/REGISTRY.md`. Section 2
"Adoption" (columns: Contract | Product | Role | Implements | Reads | Verified | Notes):

> | `INSTALL-TRUST` | doc-html-translate | P | - | - | 2026-09-22 | **not adopted, declared rather than left silent.** Three of the four download channels are unsigned - the per-user installer `doc-html-translate-setup-<version>.exe`, the portable CLI and the portable GUI, from the GitHub release and from winget - and `installer/doc-html-translate.iss` configures no signing, so the user meets SmartScreen. Searched this date across every README (en/ru/uk), every site page and the GUI strings: no surface contains "SmartScreen", the quoted dialog or any instruction, so not one of the eight rules is met. The MSIX build is signed by Microsoft at certification and is outside this. Ticket `2026-09-22_install-trust-page`; exception below |

Section 3 "Exceptions" (columns: Contract | Product | Deviation | Reason | Until):

> | `INSTALL-TRUST` | doc-html-translate | the whole contract: no trust page ships with the unsigned installer or either portable exe | the product adopted the contract on 2026-09-22 by declaring the gap rather than by conforming; the page has to be written in three authored locales and agreed against the privacy page and the Store data-safety answers. Ticket `2026-09-22_install-trust-page` | 2026-12-31 |

Re-verified 2026-09-25: both rows unchanged since 2026-09-22, the exception still open.

## Notes

The Store data-safety answers themselves live in Partner Center, not in this repository; the in-repo text a
remote session can check against is [`privacy.html`](../../privacy.html) and the Store feature list in
[`msix/README.md`](../../msix/README.md) ("Open source - no accounts, no telemetry, no ads, no data
collection"). Agreement with the Partner Center form proper is confirmed by the owner.

No code changes. If signing is ever bought, the page is updated, not deleted - users of older builds still
meet the warning and search engines already point at it.
