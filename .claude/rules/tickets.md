---
paths:
  - "DEV/plan/**"
  - ".sza-profile.json"
  - "docs/SPEC_LIFECYCLE.md"
  - ".claude/commands/spec*.md"
  - "scripts/doc-registry-local.ps1"
---

# Spec / plan tickets

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Ticket store and queue (from CLAUDE.md)

- One ticket = one Markdown file `DEV/plan/D2HTML-<NNNN>_<slug>.md` (the grammar is in `.sza-profile.json`; a
  ticket closed before the 2026-11-19 prefix migration keeps its `E<NNNN>_<slug>.md` name, one written before the
  ledger its `NN_<YYYY-MM-DD>_<slug>.md` name); a ticket with a tactical breakdown gets
  a sibling directory of the same name whose `INDEX.md` is the authority on phase state. The id is the ticket's
  permanent id: the next unused number (the ledger allocates it), never reused and never
  changed - a reorder moves lines in the queue and renames nothing, and a move to `done/` keeps the prefix. The
  one exception is the migration itself: an open ticket kept its number under `D2HTML-` and the old `E<NNNN>` id
  stays a name of it through the ledger field `formerIds`.
  Every open ticket lives in `DEV/plan/`, nowhere else. The status lives in the ledger
  (`DEV/plan/spec-catalog.jsonl`) and changes only through the ticket tool (`update.ps1 -Id <id> -Status <s>`),
  which mirrors it into the spec header; never edit the Status line by hand. The ticket tool is the canon
  harness shipped with the `sza` plugin (`tools/harness/spec_catalog/` there: `update.ps1`, `complete.ps1`,
  `next-id.ps1`, ..), configured by `.sza-profile.json`; nothing of it lives in this repo.
- `DEV/plan/` is gitignored - tickets, the ledger and the queue are local-only - so `git grep` and a
  repo-wide Grep skip it: search it with the path given explicitly.
- **[DEV/plan/RELEASE_QUEUE.md](../../DEV/plan/RELEASE_QUEUE.md) is the queue** - the unfinished work before the
  next release, in execution order, grouped into release packages (a package is an ordinal, never a
  version). The queue wins on order, the ledger on status. A ticket that reaches `Implemented` or `Verified`
  moves to `DEV/plan/done/` (fix relative links when you move one).
- **Every status change updates the queue, through the ticket tool.** `.sza-profile.json` leaves the queue at
  the harness default (`paths.releaseQueue`), so `insert.ps1`, `update.ps1`, `complete.ps1` and the other ledger
  writers rewrite the rows themselves: a new status and stamp; a new or reopened ticket added at the end of its
  package; an `Implemented`, `Verified`, `BlockNeedUserTest` or archived one dropped. The agent that changes a
  status therefore:
  - changes it only with the ticket tool, never in the ledger, the spec header or a queue row by hand;
  - reads the queue after the write, moves a newly added row to the package it belongs to, and fixes the line
    order if the new status breaks it;
  - on a "release queue not updated" / "not re-rendered" warning runs `release-queue.ps1 -Reconcile` before it
    finishes, and checks with `release-queue.ps1 -Validate`;
  - for a pre-ledger numbered ticket (no ledger row, kept as a `#` line under its package) edits that line in
    the same change as the ticket's Status, and deletes it when the ticket is closed.
- Finished tickets: `release-queue.ps1 -Show Ready` and `list-blockneedusertest.ps1` (the hands-on checks owed).
  - Shape - the portfolio harness's: `current-next-release: N` in the header; a package heading is a line
    holding the number alone (`--` = not scheduled), with its definition as the `#` line under it; a row is
    `ticket  changed  status` padded to columns 63 and 79. No prose: a note is a `#` line. A hand-written row for
    a pre-ledger ticket makes `pq` reject the whole file, hence the `#` line. History:
    `DEV/plan/done/2026-11-17_release-queue-narrative.md` and
    `DEV/plan/done/2026-11-26_release-queue-before-cleanup.md`.
  - Line order inside a package, top to bottom: work in flight; what the rest is validated against (the
    instrument before what it measures); ready-to-build work, Tactical before Approved before Draft; a
    blocker above what waits on it; strings- or docs-only work after every change that adds keys; anything
    waiting on an owner answer; below that, anything waiting on a human pass or a third party. A deviation
    from the order is stated, never silent.
- Lifecycle: `Draft -> Approved -> Tactical -> In Progress -> Implemented -> Verified` (plus
  `Partial` / `Broken` from an audit, and explicit `Block*` states). Status comes from the code +
  checks, never inferred from the filename. No time/effort estimates in spec files.
- Full flow: [docs/SPEC_LIFECYCLE.md](../../docs/SPEC_LIFECYCLE.md).

## Local engineering corpus

- Local engineering corpus (`DEV/plan` and `DEV/private`, git-ignored): `./scripts/doc-registry-local.ps1` holds its own registry, `DEV/DOCUMENT_REGISTRY.jsonl` (every record resolves, every `.md` of the trees is claimed, every link resolves; hand-run, COULD NOT VERIFY on a clone). Run it after a ticket, research note or phase file is added, moved or renamed; the tracked registry names nothing in those trees (DOC-INTERNAL-QUALITY 0.10).
