# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Canonical agent guide: [AGENTS.md](AGENTS.md)** - read it first
(architecture map, build/test/lint, conventions, pitfalls). This file only adds Claude-Code-specific notes,
so the two don't drift. Other docs: [README.md](README.md) (user-facing), [DEV/README.md](DEV/README.md) (dev notes).

The portfolio-wide **SZA Unified Rules** canon (the `sza` plugin, from `github.com/SerZhyAle/sza-unified-rules`) and
this repo's overlay record are pinned in AGENTS.md ("SZA Unified Rules" section) - this repo consumes
them by reference; the delta record is `<canon>/contrib/epub_2_html.md`. The universal rules - evidence
discipline, commit policy, house text style, chat/artifact language, shell safety, build-is-not-a-release -
have one home in the canon and are deliberately not restated here. The canon also ships its own enforcement
hooks with the plugin; this repo registers none of its own and must not re-implement them.

## What this is

`doc-html-translate` - a Go 1.26 Windows app that converts EPUB / PDF / MOBI / AZW3 / FB2 / RTF / TXT /
Markdown / HTML, CBZ / CBR / CB7 / CBT comic archives, standalone images and FileDO secret files (`.fd-sec`)
into clean local HTML with generated navigation and a real multi-level TOC, plus an optional OCR overlay
(Tesseract; forced for images and comics) and optional translation (Google Cloud or local Ollama).
Binaries: `cmd/doc-html-translate` (CLI) and `cmd/doc-html-ui` (GUI: a loopback HTTP server that shows
`ui.html` in a browser app window and runs the CLI as a child process for each conversion). A second
edition, the MV3 browser extension under `extension/`, re-implements the conversion in JS and shares no
code with the Go app. Signature "free" flow: convert, open `index.html` in Chrome, then use Chrome's
built-in page translation (no API key needed).

## Build vs Release ("сборка" vs "релиз")

The two flows, their triggers, their scripts and their cost tags are defined once in
[AGENTS.md](AGENTS.md) ("Build vs Release") and [DEV/RELEASE.md](DEV/RELEASE.md). The Claude-Code-specific
part is only the routing: a "сборка"/"build" ask runs `/build`, a "релиз"/"release" ask runs `/release`,
and one never silently becomes the other.

## Where to start for a change

Trace from [internal/pipeline/pipeline.go](internal/pipeline/pipeline.go) into the format package
(`internal/epub`, `internal/pdf`, ..). Entry points: [cmd/doc-html-translate/main.go](cmd/doc-html-translate/main.go),
app wiring in [internal/app/app.go](internal/app/app.go) and [internal/config/flags.go](internal/config/flags.go).

One run, in order (`Runner.run` / `Runner.build` in the pipeline file): reuse check against the completion
record -> `.fd-sec` unwrap -> format extractor -> JPEG XL normalize -> optional page split -> navigation and
reader injection (`internal/htmlgen`) -> OCR overlay -> translation -> TOC, index and search index ->
completion record -> open the browser. Every extractor, whatever the format, returns the same `*epub.Book`
(spine + TOC), and every later stage works on that one model. The order carries meaning: navigation and the
OCR overlay go in before translation so that their text is translated too, the TOC is built after it so that
its labels are translated, and the completion record is the last write - an interrupted run leaves none and
is rebuilt.

The verb-style command line (`convert`, `doctor`, `input check`, `output status`, ..) is `internal/commands`,
entered through `commands.Front` by both binaries before the classic flags; its manifest is
`docs/cli-commands.json` and [docs/CLI.md](docs/CLI.md) is the render, never edited by hand.

## Cross-edition parity

Two independent codebases that share no code: the Go app (CLI/GUI/MSIX) and the JS browser extension
(`extension/src`). Logic is hand-ported Go -> JS, so it drifts. The port map, the invariants that must
match and the intentional differences live in [docs/PARITY.md](docs/PARITY.md), with the process rules in
[AGENTS.md](AGENTS.md) ("Cross-Edition Parity") - read both before adding a user-facing feature.
`scripts/parity-check.ps1` is the advisory structural gate; `tests/parity_test.go` hard-fails on value drift.

## Commands

From the repository root, in PowerShell. Between releases only what takes 5 s or less is run; the rule and the
measured list are in AGENTS.md ("Build, Test, Lint").

- Fast gate: `./scripts/check.ps1 -Fast`; for the touched area only, `-Fast -Plan scripts/<child>.ps1,..`.
- One Go test: `go test -tags nodynamic ./internal/<pkg>/ -run <TestName>`. The cross-package suite is the
  `tests/` package: `go test -tags nodynamic ./tests/ -run <TestName>`.
- One extension test: `node --test extension/test/<file>.test.mjs`.
- Vet, lint, spelling: `go vet -tags nodynamic ./internal/<pkg>/`, `./scripts/lint.ps1`, `./scripts/typo.ps1`.
- Release tier, not run between releases: `./scripts/test.ps1` (the whole Go suite), `./scripts/test-extension.ps1`,
  `./scripts/check.ps1` with no switch, `./scripts/build-local.ps1 -Release`.
- `./a.ps1 <alias>` is the launcher over the scripts (`fc` fast gate, `l` lint, `ty` spelling, `cli` / `ui` one
  build each); the header of [a.ps1](a.ps1) lists every alias.

## Environment (this machine)

- **PowerShell is the shell: `go`, `node` and `npm` work in the PowerShell tool, not in the Bash tool.**
- The Go toolchain here is `GOARCH=386`, so `go test ./tests/` can die with "out of memory" at the 2 GB
  ceiling and pass on a rerun - not a regression; if it persists, set `$env:GOARCH = 'amd64'` in the same call.
- **An ad-hoc go command needs `-tags nodynamic`** (or `$env:GOFLAGS = '-tags=nodynamic'` in the same call) once
  it reaches `internal/jxl`; without it the compile stops on the guard (ticket 115, AGENTS.md "Pitfalls").
- **Line endings are settled, don't re-litigate them:** `.gitattributes` pins `*.go text eol=lf`, so `gofmt`
  and the gofmt linter agree with the committed source. `gofmt -l .` comes back clean apart from throwaway
  files under the gitignored `temp/`. The old "normalize to LF first" dance is obsolete.
- `scripts/build.ps1` / `scripts/build-ui.ps1` copy artifacts to `C:/GD/tc/SZA/_APP` and need `goversioninfo`;
  this is environment-specific and may fail on other machines.

## Content conventions

- **Typography:** the house text style has one home in the canon (`DOCUMENTATION_CONCEPT.md` §5) and is
  not restated here; it applies to generated output, docs and commit messages, and
  `tests/typography_test.go` holds it in every Markdown file of the repo, this one included.
  **One deliberate exception:** the converted
  page `<title>Book — Page N</title>` keeps an em dash - that is conventional book typography, not chatter,
  and the drift guard (`tests/typography_test.go`) allowlists an em dash only inside a `<title>` literal.
- Platform-specific code is split via `*_windows.go` / `*_nonwindows.go` - keep both sides in sync.

## High-value invariants (full list in AGENTS.md)

- No-arg CLI enters the **registration** flow, not conversion.
- Existing output is reused only when its completion record says it finished, from the same source, with the
  same result-affecting options; otherwise it is rebuilt. `-force` always rebuilds.
- Don't change public CLI flag semantics unless explicitly asked.

## Skill routing (slash commands)

Imported from the Universal Agent Kit, plus the `sza` plugin's ticket-queue forwards; the live set is
whatever `.claude/commands/` holds. Pick the cheapest path that fits:

- `/quick` - trivial edit (typo, one constant, one string). No spec, no build gate.
- `/fix` - narrow bug/behaviour fix. Local validation only (a targeted `go test` of the touched package, 5 s or less; `scripts/test.ps1` is the release tier).
- `/research` - investigate before any non-trivial change; persist findings under `DEV/research/`.
- `/spec` - write the strategic *what/why* for a feature (no file paths, no symbols).
- `/spec-tech` - break an approved spec into a phased, verifiable plan.
- `/spec-dev` - execute a tactical plan one step at a time, checking each.
- `/spec-check` - audit implementation against the spec; set status from reality.
- `/spec-fix` - apply the audit's mechanical action items, then re-audit.
- `/git` - branch model, staging, commit grouping.
- `/changelog` - record the "What's new" entry in the engineering ledger.
- `/docs-sync` - propagate a finished change across every documentation surface at once.
- `/verify-view` - headless-check converted HTML output.
- `/build` - "сборка": local+free flow (gate, build CLI+UI, changelog entry, commit). Never publishes.
- `/release` - "релиз": published+paid flow. Curates "What's new in version XXX" and drives the full
  checklist (GitHub release, winget, Store, extension) with per-step confirmation. See [DEV/RELEASE.md](DEV/RELEASE.md).
- `/spec-draft` (`/sd`), `/spec-quiz` (`/sq`), `/spec-next` (`/sn`), `/spec-all` (`/sa`) - the ticket-queue
  commands: capture a Draft, clear a `BlockQuestions` ticket, work the next ticket of the queue, take one
  ticket to its audit. Each file is a thin forward to the `sza` plugin skill of the same name and carries no
  behaviour of its own.

Subagents: `solution-researcher` (read-only investigator), `implementer` (focused code writer),
`rd-lead-kit` (senior orchestrator - opt-in, not the default agent). Methodology references:
[docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md), [DEV/research/CODE_QUALITY.md](DEV/research/CODE_QUALITY.md),
[DEV/research/VALIDATION.md](DEV/research/VALIDATION.md), [DEV/research/RESEARCH_INDEX.md](DEV/research/RESEARCH_INDEX.md).
Where these conflict with [AGENTS.md](AGENTS.md), **AGENTS.md wins**.

## Spec / plan tickets

- One ticket = one Markdown file `DEV/plan/E<NNNN>_<slug>.md` (the grammar is in `.sza-profile.json`; a ticket
  written before the ledger keeps its `NN_<YYYY-MM-DD>_<slug>.md` name); a ticket with a tactical breakdown gets
  a sibling directory of the same name whose `INDEX.md` is the authority on phase state. The id is the ticket's
  permanent id: the next unused number (the ledger allocates it), never reused and never
  changed - a reorder moves lines in the queue and renames nothing, and a move to `done/` keeps the prefix.
  Every open ticket lives in `DEV/plan/`, nowhere else. The status lives in the ledger
  (`DEV/plan/spec-catalog.jsonl`) and changes only through the ticket tool (`update.ps1 -Id <id> -Status <s>`),
  which mirrors it into the spec header; never edit the Status line by hand. The ticket tool is the canon
  harness shipped with the `sza` plugin (`tools/harness/spec_catalog/` there: `update.ps1`, `complete.ps1`,
  `next-id.ps1`, ..), configured by `.sza-profile.json`; nothing of it lives in this repo.
- `DEV/plan/` is gitignored - tickets, the ledger and the queue are local-only - so `git grep` and a
  repo-wide Grep skip it: search it with the path given explicitly.
- **[DEV/plan/RELEASE_QUEUE.md](DEV/plan/RELEASE_QUEUE.md) is the queue** - what is left to do before the
  next release, in execution order, grouped into release packages (a package is an ordinal, never a
  version). The queue wins on order, the ledger on status. A ticket that reaches `Implemented` or `Verified`
  moves to `DEV/plan/done/` and its line leaves its package (fix relative links when you move one); one still
  owed a hands-on check stays listed once, in the "implemented, waiting on a hands-on check" block, until it is
  `Verified`.
  - Shape - the portfolio harness's: `current-next-release: N` in the header; a package heading is a line
    holding the number alone (`--` = not scheduled), with its definition as the `#` line under it; a row is
    `ticket  changed  status` padded to columns 63 and 79 - the spec file name without extension, the last
    status move as `yy-MM-dd HH:mm`, the status copied from the ledger (from the ticket file for a pre-ledger
    numbered ticket). No prose: a note is a `#` line. The history it carried until 2026-11-17 is
    `DEV/plan/done/2026-11-17_release-queue-narrative.md`.
  - Line order inside a package, top to bottom: work in flight; what the rest is validated against (the
    instrument before what it measures); ready-to-build work, Tactical before Approved before Draft; a
    blocker above what waits on it; strings- or docs-only work after every change that adds keys; anything
    waiting on an owner answer; below that, anything waiting on a human pass or a third party. A deviation
    from the order is stated, never silent.
- Lifecycle: `Draft -> Approved -> Tactical -> In Progress -> Implemented -> Verified` (plus
  `Partial` / `Broken` from an audit, and explicit `Block*` states). Status comes from the code +
  checks, never inferred from the filename. No time/effort estimates in spec files.
- Full flow: [docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md).

## Research notes

- A standalone research note is `DEV/research/RESEARCH_<topic-slug>_<YYYY-MM-DD>.md`, or a folder of that
  name holding a `README.md` when it carries fixtures or test pages. The `RESEARCH_` prefix is the interface
  (`REPO-LAYOUT` rule 3): a glob or an index finds the notes by it, whatever folder they end up in.
  Findings that belong to one ticket go into that ticket's own folder, `DEV/plan/<ticket>/research/`.
- The notes named before 2026-09-25 (lowercase, no prefix) keep their names: tickets and the frozen
  `DEV/plan/done/` archive link to them. `tests/research_naming_test.go` holds that list, which only
  shrinks, and fails on any new name without the prefix.
- `CODE_QUALITY.md`, `VALIDATION.md` and `RESEARCH_INDEX.md` beside them are method references, not notes.
- A new note is covered by the `research-notes` record of `docs/DOCUMENT_REGISTRY.jsonl` by its name alone;
  `scripts/doc-registry.ps1` fails on a file there that no record claims.

## Persistent memory

Claude Code's native per-user memory is active for this repo (under `~/.claude/projects/.../memory/`
with a `MEMORY.md` index). The discipline - what is worth recording, the entry types, and re-verifying a
remembered path against the live tree before acting on it - is the canon's (`AI_USAGE.md` §4).
