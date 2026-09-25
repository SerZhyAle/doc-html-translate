# Phase 08 - Docs cleanup

**Strategic spec:** [`../16_2026-08-11_ocr-visual-fidelity-lab.md`](../16_2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** 🚧 In Progress (5 of 6; 08.6 waits on phases 05-07)
**Depends on:** all phases
**Steps done:** 5 / 6

## Objective

Every surface that must know about the lab knows about it, and no public quality claim moves.

## Prerequisites

- [ ] Phases 01-07 are ✅ Done.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `DEV/CHANGELOG.md` | Modified | - |
| `AGENTS.md` | Modified | - |
| `test_doc/CORPUS.md` | Modified | - |
| `DEV/DOCS_SURFACES.md` | Modified | - |
| `DEV/plan/ROADMAP.md` | Modified | - |
| `tools/ocrlab/README.md` | New | ≤ 160 |

## Steps

### Step 08.1 - Write the lab's own usage doc

**Files:** `tools/ocrlab/README.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Document the command surface (`verify`, `fetch`, `synth`, `seed`, `run`, `score`, `report`, `gate`),
> the environment it needs (tesseract, a Chromium browser, `DOCHT_BROWSER`), the directory layout, and
> the one rule a contributor must not break: OCR output never becomes truth, and no threshold is
> changed outside a dated report.

**Verification:**
- `tools/ocrlab/README.md` names all eight subcommands.
- It states the corpus root and the manifest path.

**Status:** `[x]` done

---

### Step 08.2 - Register the lab in the agent guide

**Files:** `AGENTS.md`
**Depends on:** Step 08.1

**Prompt for developer:**
> Add `tools/ocrlab` to the repo map with one line on what it is, and add the `ocrlab verify` / `gate`
> commands to the commands list. State plainly that it is not built by `scripts/build.ps1` and ships in
> nothing.

**Verification:**
- `AGENTS.md` mentions `tools/ocrlab` in the repo map and in the commands section.

**Status:** `[x]` done

---

### Step 08.3 - Describe the visual corpus

**Files:** `test_doc/CORPUS.md`
**Depends on:** Step 08.2

**Prompt for developer:**
> Add an `ocrlab/` section: what the scene categories are, where the manifest and annotations live,
> that the media is gitignored and never ships, and the licence rule. Cross-link the manifest.

**Verification:**
- `test_doc/CORPUS.md` contains an `ocrlab` section linking `DEV/ocrlab/corpus.json`.

**Status:** `[x]` done

---

### Step 08.4 - Record the surface decision

**Files:** `DEV/DOCS_SURFACES.md`
**Depends on:** Step 08.3

**Prompt for developer:**
> Record that the lab is a developer surface: no README, site, store-listing or `_locales` change, per
> strategic §6 ("no public quality claim moves until the benchmark has a stable, reproducible result").
> If Phase 07 made a user-visible behaviour change, list the user-facing surfaces it does touch.

**Verification:**
- `DEV/DOCS_SURFACES.md` names this ticket and its surface decision.

**Status:** `[x]` done

---

### Step 08.5 - Place the ticket in the queue

**Files:** `DEV/plan/ROADMAP.md`
**Depends on:** Step 08.4

**Prompt for developer:**
> Add the ticket to the queue at priority 40 with a one-line "what a user sees today", and note the two
> human-owned gates (corpus acquisition, holdout review) so the queue does not imply the work is
> agent-completable.

**Verification:**
- `DEV/plan/ROADMAP.md` lists the ticket with its priority and the human gates.

**Status:** `[x]` done

---

### Step 08.6 - Changelog

**Files:** `DEV/CHANGELOG.md`
**Depends on:** Step 08.5

**Prompt for developer:**
> Add one entry per modified file across all phases, in the house format, with the honest scope: what
> the lab measures, which editions it covers, and what remains human-gated.

**Verification:**
- Every file listed in any phase's "Files touched" appears in the changelog.
- No entry claims a corpus size or a quality result that the baseline report does not show.

**Status:** `[~]` in progress - every phase 01-06 and 08 file is in `DEV/CHANGELOG.md` (checked
2026-09-25; phases 01-06 under their brace-list entries of 2026-08-11 - 2026-08-12, phase 08's docs in
the entry of 2026-09-25). `test_doc/CORPUS.md` lives under the gitignored `test_doc/` and was not
checkable from a fresh clone. Step 07.3's files are in the entry of 2026-09-25 15:43:25; the rest of
phase 07's wait on 07.1 / 07.2, which are ⛔ Blocked.

## Phase done criteria

- [ ] Every `Step 08.*` is `[x] done`.
- [ ] `./scripts/check.ps1` green.
- [ ] See INDEX.md Completion gate.

## Handoff notes

See INDEX.md Completion gate.

## Rollback plan

Revert the phase commit(s). Documentation only.
