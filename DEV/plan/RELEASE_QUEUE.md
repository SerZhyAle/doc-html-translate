# Release Queue

What is still LEFT TO DO before the next release, in execution order. Everything that has not reached
`Implemented` lives here, including work blocked on another ticket, on an owner answer, or on a human
step - those need sorting too. Finished work is not here: the moment a ticket reaches `Implemented` or
`Verified` its file moves to [`done/`](done/) and its line leaves this table.

Rebuilt 2026-09-25 against the ticket files themselves: the 17 tickets of the 2026-09-24 code audit
(until then parked under `DEV/research/audit_2026-09-24/specs/`) joined the 13 that were already open, so
this is now the one queue for every open ticket. Finished lines and the release-1 narrative of
2026-08-15 .. 2026-09-12 moved to [`done/2026-08-15_release-1-worklog.md`](done/2026-08-15_release-1-worklog.md).

**The number is the ticket's id, not its place in the queue.** A ticket is
`DEV/plan/NN_YYYY-MM-DD_<slug>.md` (its tactical folder `NN_YYYY-MM-DD_<slug>/`). `NN` is given once,
when the ticket is filed - the next unused number, never one a ticket has had before - and it never
changes: not on a reorder, not on the move to `done/`, which keeps the prefix. Execution order is the
order of the lines below, so reordering the queue moves lines and renames nothing.

next-ticket-number: 50

- `#` - the ticket's id, the same number as the file's `NN_` prefix. `--` = no ticket file yet.
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

- `1` - **fixes.** A defect in shipped code, in either edition, and the small hygiene that makes the
  next fix safe. A user-visible defect that loses or exposes the user's data goes to the top of it.
- `2` - OCR quality: the lab that measures the overlay and the defects it has evidenced.
- `3` - contract sync and docs: conforming to the shared catalog, the product site and user docs.
- `4` - waiting on the owner's machine or a human: built or half-built work no cloud session can finish.
- `--` - living backlogs, human-gated corpus work, and items with no code left, so no package.

Implemented tickets leave the tables for [`done/`](done/); the ones still owed a hands-on check are
listed once, under "implemented, waiting on a hands-on check", so the check is not forgotten.

current-next-release: 1 (reordered 2026-09-25: fixes first)

## release 1 - fixes

```
#   ticket                                              changed     status
35  35_2026-09-25_pdftotext-missing-from-ci-builds       2026-09-26  In Progress - exe tracked + release guard; awaits commit and a CI build
36  36_2026-09-26_bugfix-release-gate-evidence-integrity  2026-09-26  In Progress - implemented; #43 (X25) landed, green gate waits on a Windows check.ps1
--  (no ticket) installer x86 build does not link     2026-09-26  Evidenced, unfiled - blocks the next setup.exe
```

15, the last line before, moved to [`done/`](done/) on 2026-09-25 with its catalog step closed. 33 (the bundled
pdftotext set shipped without its licences) joined and was implemented the same day and moved to [`done/`](done/).

35 was found by 33: `pdftotext.exe` was never tracked in git, so the CI-built release assets and the winget zip embed
the runtime DLLs without the executable. It sits first because it is a known defect in a shipped channel.

40-47 were implemented on 2026-09-26 and moved to [`done/`](done/) (40 fully; 41-47 wait on the checks listed
below). 36-47 come from the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md), which
closed on 2026-09-26 (22 slices, 138 findings: 0 crit, 6 high, 31 med, 101 low; register in its
[`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md)). 42 sits first: it is the one
finding that can expose a user's local or intranet images to a web page. The five other highs follow 35 -
the two that let a release ship something other than the tested tag (36, 37), then the three that give a
visibly wrong result in the default flow (39, 41, 43). The meds follow, parity drift (45) last because each of
its items may end as a PARITY.md entry rather than code. 34 re-checked every implemented id of the 2026-09-24
audit: none regressed.

37 reached `Implemented` on 2026-09-26 and moved to [`done/`](done/37_2026-09-26_bugfix-release-workflow-provenance.md):
the release workflow builds only the tag it names, the local installer and MSIX builds take `-Tag` and refuse
any other tree, the MSIX identity defaults to the frozen `SZA.Doc-HTML-Translate`, and every action is pinned by
commit. Its scratch-clone run found the unfiled line above: `build-installer.ps1` fails in the x86 link
(`resource.syso: unknown relocation type 3` with `goversioninfo` v1.7.0; the v1.4.1 on this machine's `PATH`
cannot read the three-ICO `IconPath` at all), so no setup.exe can be built for the next release until it is fixed.

Reordered 2026-09-25: every remaining defect ticket comes before any instrument, contract or docs work.
25 (`hygiene`) went first and is done (waiting on a Windows pass, listed below); the pipeline sandbox
harness it added (`internal/pipeline/harness_test.go`) is there for the tickets after it to extend. The two
extension tickets lead now, content security (19) first - an untrusted document that stays live is the more serious defect, and
`lifecycle-leaks` (18) touches the same viewer code right after it. 20 (`registration`) is Windows-only and
independent. 15 is the remaining OCR defect from the 2026-09-22 alignment run (17, its sibling, is done); it is the
instrument 29 is measured with (rule 4).

[`15_2026-09-22_ocr-discard-record-missing-for-blank-images`](done/15_2026-09-22_ocr-discard-record-missing-for-blank-images.md)
(done 2026-09-25) was first among the OCR lines by rule 2: it is the instrument the rest is measured with. Opened by the contract
alignment run of 2026-09-22 against `OCR-OVERLAY rule 12`, and it corrects §1.3 of this file - the record
was preserved as far as `applyOverlays` and is then not written, so an image that produced **no** plates
still leaves nothing behind, which is the one case the record exists for. Proven with a throwaway probe in
package `ocr` (`applyOverlays: changed=false NoText=1`, no diagnostics file). Nothing a reader sees changes
when it lands; what changes is that a blank scene can be told from a discarded one.

## release 2 - OCR quality

```
#   ticket                                              changed     status
16  16_2026-08-11_ocr-visual-fidelity-lab               2026-09-25  In Progress (6/8; 07: 07.3 done, 07.1-07.2 ⛔ annotation)
--  (no ticket) plate box rides over the logo           2026-08-13  Evidenced, unfiled
--  (no ticket) tesseract.js misses a caption on        2026-08-15  Evidenced, unfiled
    a gradient
```

`ocr-visual-fidelity-lab` is 6 of 8 phases. Phase 07 (concealment and grouping) is 1 of 7: Step 07.3
landed on 2026-09-25 on the owner's machine - the corpus re-measure the reconciliation asked for found
the balloon merge band still there (1.00-3.46x), and balloons stitched side by side are now cut on a
stroke crossing the gap between two words, in both editions (`OCR-PIPELINE` 1.1 in the catalog first).
Its concealment-mode steps 07.1 / 07.2 still need strategic §9.1 / §9.2, which wait on annotated
`texture` scenes and protected polygons - human-owned work in package `--`. Measured and left open by
07.3: a balloon pair whose two outlines the recognizer reads as one token (`ff`, `fj`, `|`) stays
stitched - two plates on `samson-and-delilah-15` - which needs a test through the token, not a
threshold. Phase 08 is 5 of 6 with its changelog step waiting only on 07's files.

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

## release 3 - contract sync and docs

```
#   ticket                                              changed     status
21  21_2026-09-23_contract-ocr-pipeline-sync            2026-09-25  In Progress (catalog done 2026-09-25; left: A3 plate font + lab, A5 overflow rule, A7, A11)
23  23_2026-09-23_contract-desktop-app-ux-sync          2026-09-23  In Progress
32  32_2026-09-25_icon-system-surfaces                  2026-09-25  In Progress (built 2026-09-25; left: the owner's on-device look - taskbar light/dark, Chrome/Edge toolbar)
26  26_2026-09-23_contract-product-web-pages-sync       2026-09-25  In Progress - Direction A done in the repo; rendered 360/768/1280 check, catalog row + exceptions and B1-B12 local only
27  27_2026-09-22_install-trust-page                    2026-09-25  In Progress - page + links done in the repo; left: publish, then the local catalog row
48  48_2026-09-26_contract-doc-internal-quality-gaps    2026-09-26  Partial - gates and fixes landed; left: shared registry row
49  49_2026-09-26_contract-doc-external-quality-gaps    2026-09-26  Partial - gates and pages landed; left: file the proposal, shared registry row
```

48 and 49 come from the `documentation-quality` adoption run of 2026-09-26: both contracts adopted as a
consumer, the rules not met recorded here and as dated exceptions in the shared registry. They sit last in
the package by rule 5 (docs-only work), 49 below 48 because the external contract requires the internal
one first.

**The six `2026-09-23_contract-*` tickets** (three left - `automated-checks` reached Implemented on 2026-09-24, `rule-adoption` and `iconography` on 2026-09-25, all moved to `done/`; iconography's system surfaces continue as 32) come out of one contract-sync pass over every catalog domain
that touches this product. Each has two halves: what the repo changes to conform, and what the catalog
lacks - written as a dated amendment where this product owns the contract (`OCR-PIPELINE`,
`OCR-INVOCATION`) and as a proposal beside the contract everywhere else. `OCR` leads because this product
owns the document another product ports from, and it was registered stale. `rule-adoption` closed on
2026-09-25: the stamp is re-synced to canon `2026.09.24.1` and its catalog half is filed. The web-pages
ticket carries one user-visible bug that should not wait for the rest of it: the shared `sza-lang` value is
`ua` on the landing page and `uk` on the extension page, so a language chosen on one shows all three on the
other - a `/fix` candidate on its own. `WAVE-PARTICLES` was read and does not apply (no canvas backdrop).

[`27_2026-09-22_install-trust-page`](27_2026-09-22_install-trust-page.md) is docs-only work, so rule 5 puts it
last among the schedulable lines. Three of the four download channels are unsigned - the setup exe and both
portable exes - and nothing we ship says the word SmartScreen, so a user who meets "Windows protected your
PC" reads nothing from us. No shipped code is wrong, but it is not `--` either: every
unanswered warning is a user who does not come back. The contract it closes is
`INSTALL-TRUST` 1.0, and until it lands the gap is a dated exception in the shared registry.

[`31_2026-09-25_canon-resync-new-duties`](done/31_2026-09-25_canon-resync-new-duties.md) moved to
[`done/`](done/) on 2026-09-25 with all four items built on the owner's instruction: the documentation
registry (`scripts/doc-registry.ps1`, which now generates `sitemap.xml`), the permission and
network-surface inventories every privacy text renders from (`scripts/security-posture.ps1`), the contract
gate in `scripts/release.ps1`, and the `RESEARCH_` naming. Both new checks run in `scripts/check.ps1`, so
26 and 27 now edit `sitemap.xml` through `-Generate` and the privacy facts through
`docs/security-posture.json`. Its first contract-gate run names five rows of 23, 26 and 27 to re-verify.
One owner step is left, listed below.

## release 4 - waiting on the owner's machine or a human

```
#   ticket                                              changed     status
28  28_2026-08-15_plate-styling-single-source           2026-09-25  BlockNeedUserTest (4 manual, owner machine)
29  29_2026-09-25_ocr-rescue-third-axis                 2026-09-25  Partial - size anchor measured + rejected 2026-09-25, next: sparse row order
30  30_2026-08-13_ocr-sweep-plate-composition           2026-08-13  Partial (7/8 criteria) - human corpus entry
```

Rule 7 puts all three last: none can be scheduled from a cloud session. 28 has its code built and waits
on the catalog step and four gates on the owner's machine; 29 needs every candidate run through the lab
corpus, which exists only there, and the discard record that 15 adds; 30 has no code left.

`plate-styling-single-source` is built (2026-09-24): `internal/appearance/appearance.json` is the one
description of the OCR overlay and the reader palette, both editions derive from it, and
`tests/appearance_parity_test.go` fails on any declaration one side has and the other lacks. What is left
needs the owner's machine: the catalog's `ocr-pipeline.md` repoint (Step 05.3 - the catalog was not
reachable), the four PowerShell gates, and the corpus comic render (Step 06.2); a Chromium render of old
against new CSS was pixel-identical meanwhile.

`ocr-sweep-plate-composition` is `Partial` with 7 of 8 done-criteria met and measured. **No code work is
left in it.** The open criterion is corpus entry for `Soldiers-Creed` and the `First-Earthman` cover,
which the lab's own contributor rules put outside what an agent may do: an annotation is truth only when
a person has corrected it, and `licenceVerifiedBy` is filled by hand after opening the asset's licence
page. It stays out of `done/` because `Partial` is not `Implemented`, and it stays out of package 1
because nothing in it ships. Moving it is the owner's call.

## implemented, waiting on a hands-on check

Moved to [`done/`](done/) on 2026-09-25, keeping their numbers, because their code is implemented and merged; each keeps its
honest `BlockNeedUserTest` status line until the named check passes, and then becomes `Verified`.

```
ticket (in done/)                                   check left
01_2026-09-24_hotfix-epub-href-containment             owner sign-off, Windows pass
02_2026-09-24_bugfix-shell-open-injection              Windows open check
03_2026-09-24_bugfix-gui-local-api-hardening           owner sign-off, Windows pass
05_2026-09-24_hotfix-output-dir-ownership              hidden marker, lock, GUI drop/delete on Windows
07_2026-09-24_bugfix-output-completeness               Ctrl+C in a real console, rebuild while Chrome holds files
08_2026-09-24_bugfix-translation-engine-correctness    real Google API and Ollama model, -max-cost run
11_2026-09-24_bugfix-external-process-bounds           process-tree kill, upgraded pdftotext cache
12_2026-09-24_bugfix-resource-budgets                  2 GB CBZ on the 386 build, real 7-Zip
13_2026-09-24_bugfix-ocr-language-data-and-detection   language download in the Store build
25_2026-09-24_chore-hygiene-and-test-gaps              scripts/check.ps1 on Windows (Windows-only tests)
20_2026-09-24_bugfix-windows-registration-honesty      Windows 11 with an existing .epub user choice
31_2026-09-25_canon-resync-new-duties                  owner reads the rendered privacy texts before main is pushed
33_2026-09-25_bundled-binaries-notices                 next release: THIRD-PARTY-NOTICES.txt in the zip and as an asset
37_2026-09-26_bugfix-release-workflow-provenance       next release: the first tag run of release.yml with the pinned actions
38_2026-09-26_bugfix-bundled-ocr-data-per-channel      next release: tessdata/eng.traineddata inside the zip, MSIX and setup.exe
41_2026-09-26_bugfix-translation-scope-and-partial-results  `-google` without a key exits 4 on Windows
42_2026-09-26_bugfix-page-ocr-image-origin             real Chrome: public fallback, same-origin pixels, file: image refused
43_2026-09-26_bugfix-pdf-text-and-image-fidelity       go test ./internal/pdf/ on Windows with the vendored pdftotext
44_2026-09-26_bugfix-extension-viewer-state-and-references  real Chrome: cross-host PDF, popup host, export alpha
45_2026-09-26_bugfix-parity-drift-audit-34             go test ./internal/epub/ on Windows; Chrome: anchors, windows-1251, sfx .cbz
46_2026-09-26_bugfix-ocr-desktop-paths-and-language    non-ANSI profile, 8.3 names on and off, OCR after -ocr-download rus
47_2026-09-26_bugfix-cli-robustness-audit-34           `n yy` at the first-run prompt; JPEG2000 PDF with -noopen opens no box
```

[`2026-09-19_page-ocr-overlay`](done/2026-09-19_page-ocr-overlay.md) is the first **new user-facing feature**
in this queue rather than a defect or an instrument: recognize every picture on an ordinary live web
page and lay the plates over them in place, so the words can be copied and the browser's own page
translation reaches them. **The code is written and the gates are green**; what gates it now is a human
pass, not a decision. The reach question that blocked it - the obvious host for the recognition engine
is newer than the extension's declared minimum Chrome version, and raising that minimum is forbidden -
was answered in code: the newer host is used where it exists and an extension-origin frame in the page
is used where it does not, and `minimum_chrome_version` stays at 105, pinned by a test. What is left is
a hands-on pass on real sites (which is also how research items 2, 3 and 4 get their measurements) and
the store-listing permission text, which needs the owner's sign-off first. It sits in this list rather than a package
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
