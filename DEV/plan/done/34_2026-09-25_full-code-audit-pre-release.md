# Full code audit before the next release, split into slices

**Status:** Implemented (22/22 slices, 2026-09-26)
**Priority:** 50
**Date:** 2026-09-25
**Tactical plan:** [`34_2026-09-25_full-code-audit-pre-release/INDEX.md`](34_2026-09-25_full-code-audit-pre-release/INDEX.md) - scaffolded by `scripts/audit-slices.ps1 -Write` on 2026-09-26

> Modelled on FastMediaSorter's `S3556_full-code-audit-pre-release` (same owner ask, same day), adapted to
> this repo: no ticket catalog CLI, no queue runner, a hand-maintained queue, and a code base two orders of
> magnitude smaller. The previous whole-repo read is
> [`DEV/research/audit_2026-09-24/`](../../research/audit_2026-09-24/README.md); this ticket is its successor,
> not a repeat.

---

## 0. Captured request

**Captured:** 2026-09-25

> Run a full code audit before the release. The code base is large, so the audit has to be split into many
> parts.

---

## 1. Problem

The owner wants a full audit before the next release, and wants it split into parts that are each done and
closed on their own.

The last whole-repo read was the 2026-09-24 audit at commit `41fbc1b`: six areas, 17 tickets, every one
of them now implemented (twelve still wait on a Windows pass, listed in the queue). That read is already
stale in two ways, measured 2026-09-25 with `git diff --numstat 41fbc1b HEAD`:

- **The fixes rewrote a large share of the code they audited.** Since `41fbc1b` there are 78 commits, and
  about 13 100 lines were added to shipped Go and 2 600 to `extension/src`, against a shipped base of
  roughly 22 700 lines of application Go (`cmd/`, `internal/`) and 9 000 of `extension/src`. Code written
  to close a finding has not itself been read by anyone but its author.
- **The 2026-09-24 scope left the release path out.** It read `cmd/`, `internal/` and `extension/src`.
  The scripts that build, sign, stamp and publish the product - `scripts/*.ps1` (30 files, 3 100 lines),
  `msix/`, `installer/doc-html-translate.iss`, `extension/build.mjs`, `extension/scripts/publish-*.mjs`,
  `extension/scripts/bump-version.mjs` - were never audited, and they are the outward-facing, one-way part
  of the product (a wrong tag, a wrong manifest, a wrong Store package is not undone by a fix).

No single agent session reads 35 000 lines carefully: the 2026-09-24 audit already had to be split into
six areas, and its own register shows the density drops where an area was large. The audit has to be cut
into slices small enough to read line by line, with coverage proven by a script rather than asserted.

---

## 2. Goals

1. Every file of **class A** - shipped code and release-path code, defined in §3.2 - belongs to exactly one
   slice, and every slice is one phase of this ticket's tactical plan, closed on its own.
2. Every slice is read through the audit layers of §5.1, and every finding gets a severity (crit / high /
   med / low) and a confidence (`conf` / `plaus`) - the same taxonomy as the 2026-09-24 register, so the
   two registers read as one history.
3. Every **crit** and **high** finding is either fixed inside the slice with proof that needs no owner
   machine (§3.2), or filed as its own ticket in release package 1 of the queue.
4. Every finding id of the 2026-09-24 register whose ticket is implemented is re-checked in the slice that
   now holds its code: still fixed, regressed, or fixed only in one edition.
5. The campaign state is measured by a script, not read off the tickets: file coverage, slices open and
   closed, finding totals by severity, tickets filed.
6. Slices run in the order of measured risk, not alphabetically.

**Non-goals:**

- **Class B**: tests (`*_test.go`, `tests/`, `extension/test/`), the OCR lab (`tools/ocrlab`,
  `extension/scripts/ocrlab*.mjs`, `_ocrlab-*.mjs`), screenshot and fixture scripts, docs and the product
  site pages. The slicing script can include them behind a flag; this release does not wait for them. The
  lab has its own measuring discipline (ticket 16), the tests have their own gates.
- Refactoring for size. A long function is not a finding; a function with two responsibilities that hides
  a defect is.
- Re-auditing what 2026-09-24 listed under "Checked and found sound" in code that has not changed since
  `41fbc1b`.
- Visual design, UI strings and translations (package 3 of the queue owns them).
- Anything that needs a real release, a Store upload or a winget PR to prove. The audit reads the release
  scripts; it never runs their publishing steps.

---

## 3. Wishes and constraints

### 3.1 Owner wishes

1. Split the audit into many parts - the owner's words.
2. Each part closes on its own, so the audit can pause between parts without losing state.
3. Progress is visible in the file the owner reads: [`RELEASE_QUEUE.md`](../RELEASE_QUEUE.md) carries this
   ticket's line with `k/N slices`, and the findings it files join package 1.

### 3.2 Hard constraints

- **Class A** is exactly: non-test Go under `cmd/` and `internal/`; `extension/src/` (JS, HTML, CSS);
  `extension/manifest.json`, `extension/build.mjs`, `extension/scripts/publish-*.mjs`,
  `extension/scripts/bump-version.mjs`, `extension/scripts/gen-appearance.mjs`; `scripts/*.ps1` and
  `scripts/lib/`; `msix/` build scripts and manifest template; `installer/`; the winget manifest source.
  Generated files (`*_gen.go`, vendored third-party code, `internal/appearance` outputs) are listed in the
  manifest as generated and read only at their generator.
- **Platform split stays whole.** A `*_windows.go` file and its `*_nonwindows.go` twin are always in one
  slice, so the two sides are read against each other (the repo invariant that keeps them in sync).
- **Parity pairs stay whole where the port map has them.** A Go package and its hand-ported JS module named
  in [`docs/PARITY.md`](../../../docs/PARITY.md) are read in one slice when their combined size fits the
  limit; when it does not, the two slices are adjacent and each names the other. Drift between the two is
  a finding class of its own.
- **Inline-fix rule.** A slice fixes a finding itself only when the proof needs no owner machine:
  (a) a Go change with a test that runs on this machine, run after the fix with its exit code cited, or a
  JS change covered by `extension/test`; or (b) a change that preserves behaviour by construction - dead
  code, a missing `defer Close`, an unchecked error that is now returned, a comment. Anything that touches
  the registry, the Windows shell, the MSIX or installer, a real browser, or a publishing script is filed
  as a ticket. A fix in one edition that has a parity twin is filed, never made on one side only.
- **Dedupe rule.** Before filing, search `DEV/plan/`, `DEV/plan/done/` and the 2026-09-24 register for the
  symptom; a match gets a link, not a duplicate. A regression of a 2026-09-24 finding reopens nothing in
  `done/` - it is a new ticket citing the old id.
- **Evidence.** A finding says `conf` only when the code was read at the cited line or the behaviour was
  run; OS- or third-party-dependent claims are `plaus`. Line numbers cite the commit the slice was read at.
- **Toolchain.** Go runs from PowerShell, and the toolchain is 386: a `go test ./tests/` out-of-memory
  death is rerun, not reported as a finding.
- **Data compatibility.** No output-format or completion-record change inside a slice. A finding that needs
  one is a ticket, because existing converted books must keep being recognised.
- **Localisation.** No new UI strings inside a slice.

### 3.3 Owner inputs (approval gate)

- **Scope:** class A as in §3.2 - confirm the release path is in.
- **Slice size:** 25 files and 3 000 lines by default, both parameters of the slicing script.
- **Validation level:** partition and coverage proven by the script (0 uncovered, 0 doubly covered class-A
  files); each slice closed with its prescan output and its findings table; campaign closed by the
  summary script's exit code.
- **Owner sign-off:** pending.
- **Related tickets:** the 17 tickets of the 2026-09-24 audit (re-checked, not reopened);
  [33 bundled-binaries-notices](33_2026-09-25_bundled-binaries-notices.md) (a release-path finding already
  filed - the slice that holds `internal/bundledtools` links it instead of re-finding it).

---

## 4. Current state

The method exists: the 2026-09-24 audit read every file in six areas, ran `go vet` for linux and
`GOOS=windows`, `go test -race` and `staticcheck`, and kept one register with ids per area (P, G, E, X, T,
O, B, Q) that every ticket cites. [`DEV/research/CODE_QUALITY.md`](../../research/CODE_QUALITY.md) holds the
anti-slop patterns and [`DEV/research/VALIDATION.md`](../../research/VALIDATION.md) the evidence rules. What
it lacked was a partition that can be re-run - its six areas were chosen by hand, so there is no way to
tell which files a later commit added outside them - and it had no release-path area.

Execution is by hand here: no catalog CLI, no queue runner, no ticket lease. That settles the shape. The
slices are **phases of this one ticket**, not child tickets: thirty queue lines for thirty slices would
bury the queue the owner reads, while one ticket with `INDEX.md` as the authority on phase state is how
tickets 16 and 28 already work. Tickets appear in the queue only for what the audit finds.

Measured size, 2026-09-25 (`git ls-files`, lines counted per file):

| Area | Files | Lines |
| --- | ---: | ---: |
| `internal/` + `cmd/` (non-test Go) | 133 | about 22 700 |
| `extension/src` | 43 | about 9 000 |
| `scripts/*.ps1` | 30 | about 3 100 |
| other release path (`msix/`, `installer/`, `extension/*.mjs` publish/build) | about 10 | not yet counted |

The largest single packages are `internal/ocr` (4 000), `internal/htmlgen` (2 100), `cmd/doc-html-ui`
(2 100), `internal/epub` and `internal/pdf` (1 500 each); the largest single file is
`extension/src/viewer.js` (1 600). At 3 000 lines per slice that is on the order of 12 to 16 slices; the
number is whatever the script prints on the live tree, and this spec does not predict it.

---

## 5. Approach

This ticket does not audit code by itself. It delivers one script and one procedure; the audit is done by
the slice phases they produce.

### 5.1 Pillars

**Slicing.** A deterministic script (`scripts/audit-slices.ps1`) walks class A, groups files by package
(Go) or by module family (JS: `ocr-*`, the format readers, the viewer and its helpers, the service
worker and page agent), keeps platform twins and parity pairs together (§3.2), cuts a package over the
limit into the fewest balanced parts by file, and merges small neighbours under a common parent up to the
limit. For every slice the manifest records: files, lines, lines changed since `41fbc1b`, the 2026-09-24
finding ids whose cited files it holds, and a risk score. The risk score is counted from the text:
filesystem deletes and writes (`RemoveAll`, `os.Rename`, `WriteFile`), process spawns, archive and path
joins, HTTP handlers, goroutines and shared state, `innerHTML` and message listeners on the JS side,
`Invoke-Expression`, `git push` and store calls on the PowerShell side - per thousand lines, with the
divisor floored at a quarter of the line limit so a tiny slice does not outrank a large one on one hit,
and weighted up by the share of the slice changed since `41fbc1b`. The same tree gives the same manifest.
The manifest is written to the ticket folder as `slices.json`, and `INDEX.md` gets one phase per slice in
risk order.

**Slice procedure.** One procedure, written into the phase template, readable without opening this
ticket:

1. *Prescan* - the mechanical pass over the slice's own files: `go vet` for linux and `GOOS=windows`,
   `staticcheck`, `go test -race` on the slice's packages, the `extension/test` files of its modules,
   `scripts/lint.ps1` where it applies, `scripts/parity-check.ps1` for a parity pair. Its output is the
   starting point, not something the read repeats.
2. *Re-check* - every 2026-09-24 finding id the manifest lists for the slice: still fixed, regressed, or
   fixed in one edition only.
3. *Read* - line by line, through the layers the gates cannot see:
   - data safety: deletes, overwrites and renames of anything the run did not create; path containment;
     reuse of stale output;
   - untrusted input: archives, HTML, RTF, XML, filenames reaching a shell, the GUI's local API, messages
     into the extension;
   - resource bounds: process lifetime and timeouts, memory and size budgets on the 386 build, handles;
   - concurrency: shared state, goroutine and listener lifetimes, cancellation;
   - honesty: exit codes, messages that say done when it was not, errors swallowed (`CODE_QUALITY.md`
     pattern 2);
   - platform twins and Go/JS parity against `docs/PARITY.md`;
   - release path only: frozen anchors, version derivation, the pre-flight verdict gate, secrets, and
     every step that publishes - can it run without the owner having asked for that exact release.
4. *Triage* - crit and high: a ticket in package 1 (after dedupe) or an inline fix under §3.2; med: inline
   under §3.2, else a ticket in the package its area belongs to; low: inline or a register line with no
   ticket.
5. *Close* - the inline fixes land with their test run cited; the phase ends with its findings table,
   severity counts and links to the tickets it filed.

**Register.** Findings go to one register in the ticket folder, `FINDINGS.md`, in the 2026-09-24 format
(`id - severity - confidence - finding - evidence - ticket`) with ids that continue per area, so a later
reader has one list per campaign and can tell a new finding from a regression of an old one.

**Summary.** `scripts/audit-slices.ps1 -Summary` reads `slices.json`, `INDEX.md` and `FINDINGS.md`:
class-A files in the tree that no slice holds, files held twice, slices by state, finding totals by
severity, tickets filed and their status lines. Its exit code tells "campaign closed" from "open slices or
uncovered files". A file added to the tree after slicing shows up as uncovered and goes into a tail slice
made by the same script.

### 5.2 Flow

Tree at the audit commit -> slicing -> `slices.json` + phases in `INDEX.md` -> each slice: prescan,
re-check, read, triage, close -> `FINDINGS.md` + new tickets in the queue -> summary -> this ticket's
status.

While slices are open this ticket is `In Progress (k/N slices)` in its own status line and in the queue.
It becomes `Implemented` when the summary exits clean and moves to `done/` like any other ticket; the
tickets it filed stay in the queue on their own.

### 5.3 Extension points

- A flag that adds class B (tests, the OCR lab) for a later campaign, without changing the slicing.
- Slice limits are parameters, not literals.
- The next pre-release audit is a new ticket with its own manifest and a new base commit; this one is not
  reopened.
- The risk score is one function over a list of signals; a new signal does not touch the partition.

---

## 6. Open questions

1. **Base commit for "changed since"**
   - **Question:** `41fbc1b` (the commit the 2026-09-24 audit cites) or the commit that closed its last
     ticket.
   - **Status:** Resolved - `41fbc1b`: the fixes themselves are the unread code.
2. **Where the release-path scripts sit in the order**
   - **Question:** they carry little code but the most irreversible actions; the risk score may rank them
     low.
   - **Options:** trust the score; pin one release-path slice first.
   - **Status:** Open - owner input. Default: the score decides, and the release-path slice is pinned
     first only if the score puts it below the median.
3. **Whether a regression of a 2026-09-24 finding blocks the release**
   - **Status:** Resolved - a regressed `crit` or `high` is a new package-1 ticket at priority 90, the same
     as a new one.

---

## 7. Risks

- Files added after slicing fall outside every slice. Mitigation: the summary counts uncovered files, and a
  tail slice is cut before closing.
- One defect is found from two slices and filed twice. Mitigation: the dedupe step is part of the
  procedure, not a wish.
- A slice "repairs" instead of auditing and breaks behaviour, most likely in the Windows-only code this
  machine can only partly exercise. Mitigation: the inline-fix rule; anything Windows-shell, registry,
  installer or browser-runtime is a ticket.
- A fix made in Go only, leaving the JS edition behind. Mitigation: parity pairs are one slice, and a fix
  with a parity twin is always filed.
- The release-path read tempts a dry run of a publishing step. Mitigation: the audit reads those scripts
  and runs only their `-WhatIf` / local gates; invariant 4 of the canon applies unchanged.
- A large package cut by file puts related code in two slices. Mitigation: the manifest names neighbouring
  slices of the same package, and a slice may read a neighbour's file for context without auditing it.

---

## 8. User-visible changes

None - developer tooling and a findings register. The tickets it files carry their own user-facing
changes.

---

## 9. Testing strategy

- Slicing, on a fixture tree in `tests/`: every file in exactly one slice; limits held; platform twins and
  a declared parity pair never split; the same tree gives the same manifest; a package over the limit is
  cut, small neighbours are merged; the class-B flag changes the membership only.
- Summary, on a fixture ticket folder with phases in different states: counts and exit code; an uncovered
  file in the fixture is reported.
- Live run: slicing on the current tree, `INDEX.md` with one phase per slice, summary with open slices and
  zero uncovered files.

---

## 10. Related

- [`DEV/research/audit_2026-09-24/`](../../research/audit_2026-09-24/README.md) - the previous register, its
  method and its "checked and found sound" list.
- [`DEV/research/CODE_QUALITY.md`](../../research/CODE_QUALITY.md), [`DEV/research/VALIDATION.md`](../../research/VALIDATION.md)
  - the patterns and the evidence rules the read applies.
- [`docs/PARITY.md`](../../../docs/PARITY.md) - the port map that decides parity pairs.
- [`DEV/RELEASE.md`](../../RELEASE.md) - the release path the release-path slice reads.
- [33 bundled-binaries-notices](33_2026-09-25_bundled-binaries-notices.md) - already-filed release-path
  finding.

---

## 11. Acceptance criteria

1. The summary shows 0 class-A files outside a slice and 0 files in two slices.
2. `INDEX.md` has one phase per slice in risk order, and each phase can be executed with `/spec-dev` on
   its own: its files are listed in `slices.json`, and its procedure does not send the reader here.
3. Re-running the slicing on the same tree prints the same manifest.
4. Every 2026-09-24 finding id whose code is in class A has a re-check line in `FINDINGS.md`.
5. Every `crit` and `high` finding has a ticket in package 1 of the queue or an inline fix with its test
   run cited.
6. The summary exits with "open" while any slice is open, and this ticket reaches `Implemented` only when
   it exits clean.

---

## 12. Implementation log

- **2026-09-26 - approved by the owner's "implement" and built.** `scripts/audit-slices.ps1` (slicing,
  `-Write`, `-Tail`, `-Summary`) and its fixture test `tests/audit_slices_test.go`
  (`go test -run TestAuditSlices ./tests/` - exit 0). The live run cut class A into **21 slices over
  237 files, 42 790 lines** (more than §4 estimated: the GUI page, `internal/i18n` and the release path
  were uncounted there); a second `-Write` on the same tree wrote a byte-identical manifest. 144 of the
  148 ids of the previous register are placed on slices; G19, Q5 and Q6 cite only class-B files (tests,
  the lab) and Q7 cites no file at all ("observed in the test log") - the summary lists it for a
  re-check by hand. The release path ranked 4th, above the median, so open question 2's default did not
  pin it.
- **Deviations from §3.2, each deliberate:**
  - Class A also holds `cmd/doc-html-ui/ui.html` (the GUI's code, where G5, G10, G15, G17 and G18 live),
    `cmd/*/versioninfo.json` (version stamping) and `.github/workflows/*.yml` (the tag workflows are the
    one step that publishes). The GUI's string dictionary `cmd/doc-html-ui/i18n.js` is class B: UI
    strings are package 3's, per the non-goals.
  - "The winget manifest source" is `winget/*.yaml`; the per-version copies under `winget/manifests/`
    are submitted snapshots, not sources, and are out of scope.
  - Parity pairs come from `configs/parity-map.json`, the machine-readable twin of the port map that
    `tests/parity_map_test.go` keeps equal to `docs/PARITY.md`, not from parsing the Markdown table.
    A module's page or stylesheet (`options.html`) follows the module.
  - The prescan runs `golangci-lint` (which carries the staticcheck analyzers) because standalone
    `staticcheck` is not installed, and `go test` without `-race`, which windows/386 does not support.
  - The phase procedure lives in `PHASE_TEMPLATE.md` in the tactical folder and is rendered into every
    phase file, so a phase is executable without this spec.
- **Execution:** slices run as parallel read-only auditors, one per phase file; the orchestrating
  session owns `FINDINGS.md`, the final ids, the dedupe and the tickets, so two slices can never take
  the same id or file the same defect twice.
- **2026-09-26 - campaign closed.** All 22 slices read (21 by parallel auditors under the phase
  procedure, the tail slice S22 by the orchestrating session after the S07 auditor showed two
  release-path files in no class - R30, fixed in the slicer and cut with `-Tail`).
  `./scripts/audit-slices.ps1 -Summary` ends in `audit-slices: PASS (campaign closed: 22 slices, 138
  findings)`, exit 0: 239 class-A files, 0 uncovered, 0 held twice; 144 of 144 placed register ids
  re-checked, none regressed, E6 / E12 / E14 fixed in the desktop edition only (their extension halves
  filed as B46); Q7 re-checked by hand.
  - **Findings:** crit 0, high 6, med 31, low 101 - [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).
    The six highs: R1 (a `-Plan` subset turns the release gate green), R18 (`release.yml` dispatch ships
    the branch under a tag), E31 (the default single-page merge drops page styles - flipped PDF images
    render mirrored), T13 (the reader chrome is sent to the paid engine), B49 (page OCR fetches any image
    a page names with the extension's access), X25 (the PDF ligature filter drops real short lines and
    keeps `scripts/test.ps1` red on the owner's machine).
  - **Tickets filed, all package 1:** 36-47. Every crit/high has one; 13 lows and one med were fixed
    inline, each with its test run (G20, G21, G31, R30-R32, P27, P29, P31, P32, P37, P38, E25, E40, T12,
    T14, O13).
  - **Proof of the inline fixes:** `go test -count=1` over the 14 touched packages exit 0;
    `golangci-lint` over them exit 0; `GOOS=linux go vet` exit 0; `gofmt -l cmd internal tests` empty.
    `go test ./tests/` without the full-corpus test exit 0; `TestConvertTestDoc` dies at the 2 GB ceiling
    of the 386 toolchain (known, not a regression) and passes under `GOARCH=amd64` (exit 0, 132 s).
- **Deviations during execution:** severities were raised from the auditor's med to high for E31, T13
  and X25 - each is a user-visible wrong result in the default flow; `docs/PARITY.md` staleness found
  along the way (E29, E39, B42) is filed in 45 rather than fixed here, because ticket 45 decides which side
  is right.
