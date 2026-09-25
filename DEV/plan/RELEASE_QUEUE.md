# Release Queue

What is still LEFT TO DO before the next release, in execution order. Everything that has not reached
`Implemented` lives here, including work blocked on another ticket, on an owner answer, or on a human
step - those need sorting too. Finished work is not here: the moment a ticket reaches `Implemented` or
`Verified` its file moves to [`done/`](done/) and its line leaves this table.

Rebuilt 2026-09-25 against the ticket files themselves: the 17 tickets of the 2026-09-24 code audit
(until then parked under `DEV/research/audit_2026-09-24/specs/`) joined the 13 that were already open, so
this is now the one queue for every open ticket. Finished lines and the release-1 narrative of
2026-08-15 .. 2026-09-12 moved to [`done/2026-08-15_release-1-worklog.md`](done/2026-08-15_release-1-worklog.md).

**File names carry the queue position.** An open ticket is `DEV/plan/NN_YYYY-MM-DD_<slug>.md` (its
tactical folder `NN_YYYY-MM-DD_<slug>/`), where `NN` is its line number in the tables below, so a
directory listing reads in execution order. Reordering the queue means renaming the files and fixing
every link to them in the same commit. A ticket that moves to `done/` drops its `NN_` prefix.

- `#` - the queue position, the same number as the file's `NN_` prefix. `--` = no ticket file yet.
- The release package is the `## release N` heading a line sits under. It is an **ordinal**, not a
  version: this product's version is derived mechanically from the build date (`26.MMDD.HHmm`) and is
  never hand-picked.
- `ticket` - spec file name in `DEV/plan/` without the extension, or `(no ticket)` for work that is
  real, evidenced and unfiled.
- `changed` - the date the STATUS last moved (from the ticket's own status line or its tactical
  `INDEX.md` "Last updated"), not the date the prose was last edited.
- `status` - copied from the ticket file itself. There is no catalog to mirror in this repo, so this
  column is maintained by hand and the **ticket wins** whenever the two disagree.

**This file decides what gets worked on next.** When it and a ticket disagree about order, this file
wins; when they disagree about a status, the ticket file wins. Deviating from the order below is
allowed with a stated reason, never silently.

The line order inside a package is execution order, built by these rules, top to bottom:

1. Work already in flight comes first - it is half-paid for, and here it is literally uncommitted.
2. Then whatever the rest of the package is validated against: the measuring instrument before the
   thing it measures.
3. Then ready-to-build work, closest to done first - Tactical before Approved before Draft.
4. A blocker always sits above the ticket that waits for it, so the chain reads downwards.
5. Work that only adds strings or docs goes last in its package, after every change that will add keys.
6. Anything waiting on an owner answer sinks to the bottom of the package.
7. Anything waiting on a human pass or a third party sinks below that. It is not schedulable, so it
   must not sit where it looks like the next task.

Packages, so a new ticket lands in the right one:

- `1` - **must ship before the next release.** A user-visible defect in shipped code that loses or
  exposes the user's data, or the instrument that certifies the release is honest. Nothing enters this
  package for tidiness.
- `2` - output truth: the converted book or its translation is wrong or incomplete, silently.
- `3` - robustness: a hostile or oversized input, or a hung external tool, takes the run down.
- `4` - real, evidenced, and survivable for one more release. OCR quality, the extension, contract
  sync, hygiene and gates whose absence costs no reader anything today.
- `--` - living backlogs, human-gated corpus work, and items with no code left, so no package.

current-next-release: 1 (rebuilt 2026-09-25)

## release 1 - must ship

```
#   ticket                                              changed     status
01  01_2026-09-24_hotfix-epub-href-containment          2026-09-25  BlockNeedUserTest - sign-off, Windows
02  02_2026-09-24_bugfix-shell-open-injection           2026-09-25  BlockNeedUserTest - Windows hands-on
03  03_2026-09-24_bugfix-gui-local-api-hardening        2026-09-24  Draft (P90)
05  05_2026-09-24_hotfix-output-dir-ownership           2026-09-25  BlockNeedUserTest - Windows hands-on
```

The three data-safety Drafts lead because each one lets a crafted book or a local page reach outside
the output folder. `href-containment` goes first: it reuses the DOM link rewriter that ticket 06 already
landed ([`done/2026-09-24_bugfix-epub-html-content-fidelity`](done/2026-09-24_bugfix-epub-html-content-fidelity.md)), so it is the cheapest of the three.

**01 is implemented** and covered by tests on Linux; it waits on the owner's sign-off and a Windows pass.
Rule 7 would sink it below 05. 02 is in the same state (a Windows open check). Both keep their numbers for now, a stated deviation:
renumbering 01-05 is one commit of its own, so the implementation commits stay reviewable.

**04 left the package on 2026-09-25.** `ocr-rescue-floor-drops-genuine-lettering` closed on its
measurement and moved to [`done/`](done/2026-08-13_ocr-rescue-floor-drops-genuine-lettering.md); the recall
it did not reach is ticket 29 in package 4. It was never a data-safety item, so package 4 is where the
rest belongs. Its number stays free rather than renumbering 05, the same as 06.
Ticket 15 corrects that section's claim about the no-plate record.

**05 is implemented** (58c9caa) and covered by tests on Linux. It stays out of `done/` because
`BlockNeedUserTest` is not `Implemented`: the hidden marker attribute, the lock against a real second
process and the GUI drop/delete need a pass on Windows. Rule 7 puts it at the bottom.

## release 2 - output truth

```
#   ticket                                              changed     status
07  07_2026-09-24_bugfix-output-completeness            2026-09-24  Draft (P90)
08  08_2026-09-24_bugfix-translation-engine-correctness 2026-09-24  Draft (P85)
09  09_2026-09-24_bugfix-reader-layer-and-single-page   2026-09-24  Draft (P80)
10  10_2026-09-24_bugfix-legacy-text-decoding           2026-09-24  Draft (P80)
```

06 was in the audit's fourth package and led here by rule 1; it reached Implemented on 2026-09-25 (c1e9ec7,
274fdb8, 8810164) and moved to [`done/`](done/2026-09-24_bugfix-epub-html-content-fidelity.md). 09 builds on its
link-rewrite machinery.

## release 3 - robustness

```
#   ticket                                              changed     status
11  11_2026-09-24_bugfix-external-process-bounds        2026-09-24  Draft (P75)
12  12_2026-09-24_bugfix-resource-budgets               2026-09-24  Draft (P75)
13  13_2026-09-24_bugfix-ocr-language-data-and-         2026-09-24  Draft (P70)
    detection
14  14_2026-09-24_bugfix-pdf-extraction-accuracy        2026-09-24  Draft (P65)
```

## release 4 - can slip one release, with the reason stated

```
#   ticket                                              changed     status
15  15_2026-09-22_ocr-discard-record-missing-for-       2026-09-22  Draft
    blank-images
16  16_2026-08-11_ocr-visual-fidelity-lab               2026-08-15  In Progress (6/8 phases)
17  17_2026-09-22_tsv-columns-read-by-position          2026-09-22  Draft
18  18_2026-09-24_bugfix-extension-lifecycle-leaks      2026-09-24  Draft (P60)
19  19_2026-09-24_bugfix-extension-content-security     2026-09-24  Draft (P60)
20  20_2026-09-24_bugfix-windows-registration-honesty   2026-09-24  Draft (P50)
21  21_2026-09-23_contract-ocr-pipeline-sync            2026-09-23  Draft - catalog amendment first, then code
22  22_2026-09-23_contract-rule-adoption-sync           2026-09-23  Draft - no product code
23  23_2026-09-23_contract-desktop-app-ux-sync          2026-09-23  Draft
24  24_2026-09-23_contract-iconography-sync             2026-09-23  Draft - proposals before code
25  25_2026-09-24_chore-hygiene-and-test-gaps           2026-09-24  Draft (P40)
26  26_2026-09-23_contract-product-web-pages-sync       2026-09-23  Draft - site, every authored locale
27  27_2026-09-22_install-trust-page                    2026-09-22  Draft - docs only, every authored locale
28  28_2026-08-15_plate-styling-single-source           2026-09-25  BlockNeedUserTest (4 manual, owner machine)
29  29_2026-09-25_ocr-rescue-third-axis                 2026-09-25  Draft - lab corpus, owner machine
30  30_2026-08-13_ocr-sweep-plate-composition           2026-08-13  Partial (7/8 criteria) - human corpus entry
--  (no ticket) plate box rides over the logo           2026-08-13  Evidenced, unfiled
--  (no ticket) tesseract.js misses a caption on        2026-08-15  Evidenced, unfiled
    a gradient
```

Changes against the previous edition of this package, each for a rule: 28 (`plate-styling`) and 30
(`ocr-sweep`) moved to the bottom by rule 7 - the first waits on four manual checks on the owner's
machine, the second on human corpus entry with no code left. The audit's extension and registration
tickets (18-20) sit with the other ready Drafts; its hygiene ticket (25) sits just above the docs-only
lines by rule 5. The two unfiled items carry no number until they get a ticket file. 29 (`ocr-rescue-third-axis`,
split out of the closed 04 on 2026-09-25) joins them by rule 7: every candidate it names has to be run
through the lab corpus, which exists only on the owner's machine. It sits below 15 by rule 4 - it
needs the discard record for a no-plate image to compare what a relaxed rule newly reads.

**The six `2026-09-23_contract-*` tickets** (five left - `automated-checks` reached Implemented on 2026-09-24 and moved to `done/`) come out of one contract-sync pass over every catalog domain
that touches this product. Each has two halves: what the repo changes to conform, and what the catalog
lacks - written as a dated amendment where this product owns the contract (`OCR-PIPELINE`,
`OCR-INVOCATION`) and as a proposal beside the contract everywhere else. `OCR` leads because this product
owns the document another product ports from, and it was registered stale. `rule-adoption` is next because
its first item - committing `docs/contracts/` - is what every other pointer depends on. The web-pages
ticket carries one user-visible bug that should not wait for the rest of it: the shared `sza-lang` value is
`ua` on the landing page and `uk` on the extension page, so a language chosen on one shows all three on the
other - a `/fix` candidate on its own. `WAVE-PARTICLES` was read and does not apply (no canvas backdrop).

[`15_2026-09-22_ocr-discard-record-missing-for-blank-images`](15_2026-09-22_ocr-discard-record-missing-for-blank-images.md)
is first among the OCR lines by rule 2: it is the instrument the rest is measured with. Opened by the contract
alignment run of 2026-09-22 against `OCR-OVERLAY rule 12`, and it corrects §1.3 of this file - the record
was preserved as far as `applyOverlays` and is then not written, so an image that produced **no** plates
still leaves nothing behind, which is the one case the record exists for. Proven with a throwaway probe in
package `ocr` (`applyOverlays: changed=false NoText=1`, no diagnostics file). Nothing a reader sees changes
when it lands; what changes is that a blank scene can be told from a discarded one.

`plate-styling-single-source` is built (2026-09-24): `internal/appearance/appearance.json` is the one
description of the OCR overlay and the reader palette, both editions derive from it, and
`tests/appearance_parity_test.go` fails on any declaration one side has and the other lacks. What is left
needs the owner's machine: the catalog's `ocr-pipeline.md` repoint (Step 05.3 - the catalog was not
reachable), the four PowerShell gates, and the corpus comic render (Step 06.2); a Chromium render of old
against new CSS was pixel-identical meanwhile.

**The `First-Earthman` cover plate rides over the `PLANET COMICS` logo.** The plate's text is the
cover's own top banner line and it sits on that banner, but its box is taller than its line, so it
reaches up over the logo. Coverage 0.2774, inside the bound - a plate-box height question, not a merge.
Needs its own ticket. Package 2 because comics are the core case and this is visible, but it damages
one logo rather than the reading.

**`synth-caption-on-gradient` is read by native tesseract and missed by tesseract.js.** An engine
difference, which the lab spec anticipates in §6, but it is still a scene the reader loses in one
edition and keeps in the other. The cheap resolution is to record it as a divergence in
[`docs/PARITY.md`](../../docs/PARITY.md) with the consequence stated; the expensive one is a rescue
rung. Decide, do not leave it unwritten.

`ocr-visual-fidelity-lab` is 6 of 8 phases. Phase 07 (concealment and grouping) reads "Not started"
while most of its subject matter was in fact delivered out of band by the P46 and P47 tickets - that
mismatch needs reconciling before the phase is planned, not after. Phase 08 is 5 of 6 with its last step
waiting on 07. Its two genuinely open pieces are human-owned and sit in package `--`.

`ocr-sweep-plate-composition` is `Partial` with 7 of 8 done-criteria met and measured. **No code work is
left in it.** The open criterion is corpus entry for `Soldiers-Creed` and the `First-Earthman` cover,
which the lab's own contributor rules put outside what an agent may do: an annotation is truth only when
a person has corrected it, and `licenceVerifiedBy` is filled by hand after opening the asset's licence
page. It stays out of `done/` because `Partial` is not `Implemented`, and it stays out of package 1
because nothing in it ships. Moving it is the owner's call.

[`17_2026-09-22_tsv-columns-read-by-position`](17_2026-09-22_tsv-columns-read-by-position.md) is the other
finding of the alignment run: `parseTSV` skips the header row that names the columns and then reads fixed
indices, so a Tesseract build that inserts a column would not fail - it would put plates in the wrong place
with the wrong confidences. No build in use today does, which is why it is package 2 and not 1.

[`27_2026-09-22_install-trust-page`](27_2026-09-22_install-trust-page.md) is docs-only work, so rule 5 puts it
last among the schedulable lines. Three of the four download channels are unsigned - the setup exe and both
portable exes - and nothing we ship says the word SmartScreen, so a user who meets "Windows protected your
PC" reads nothing from us. It is package 2 rather than 1 because no shipped code is wrong; it is not `--`
because every unanswered warning is a user who does not come back. The contract it closes is
`INSTALL-TRUST` 1.0, and until it lands the gap is a dated exception in the shared registry.

[`2026-09-19_page-ocr-overlay`](done/2026-09-19_page-ocr-overlay.md) is the first **new user-facing feature**
in this queue rather than a defect or an instrument: recognize every picture on an ordinary live web
page and lay the plates over them in place, so the words can be copied and the browser's own page
translation reaches them. **The code is written and the gates are green**; what gates it now is a human
pass, not a decision. The reach question that blocked it - the obvious host for the recognition engine
is newer than the extension's declared minimum Chrome version, and raising that minimum is forbidden -
was answered in code: the newer host is used where it exists and an extension-origin frame in the page
is used where it does not, and `minimum_chrome_version` stays at 105, pinned by a test. What is left is
a hands-on pass on real sites (which is also how research items 2, 3 and 4 get their measurements) and
the store-listing permission text, which needs the owner's sign-off first. It sits at the bottom of package 4
for the reason rule 7 gives: it is waiting on a human pass, so it must not sit where it looks like the
next task.

## not scheduled - no package

```
--   lab corpus growth + holdout annotation          2026-08-11  Human-owned
--   thresholds.json is stale against its own        2026-08-15  Unblocked (worklog 1.2), next baseline
     corpus
--   the halo metric penalises a correct plate on    2026-08-13  Evidenced, no gate
     a screened ground
```

The lab's remaining human phases - acquiring 200+ licence-verified scenes, each licence page read by a
person, and the two-reviewer holdout annotation - are the real gate on the whole visual-fidelity
package and cannot be scheduled against a release date.

`thresholds.json` is now the **first thing after the release**: 1.2 landed, so the blocker (worklog 1.2) is gone and the concealment bound is knowingly derived against a blind scorer. It was derived from 11 annotated scenes; there are 13, and the two added since are the
hardest in the corpus, so `recall` and `review` fail on arithmetic rather than on a regression, and
`cost` is stale because the rescue ladder now runs every rung. Re-deriving belongs to a dated baseline
run; the baseline it should be derived against is `temp/ocrlab/20260815-190756`, the first run whose
concealment numbers cover every scene.
