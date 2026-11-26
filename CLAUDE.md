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

Trace from [internal/pipeline/pipeline.go](internal/pipeline/pipeline.go) into the format package; the stage order
carries meaning ([conversion-pipeline](.claude/rules/conversion-pipeline.md)). The Go app and the JS extension share
no code: read [docs/PARITY.md](docs/PARITY.md) before adding a user-facing feature.

## Commands

PowerShell, repository root; the commands and the 5 s limit are in AGENTS.md ("Build, Test, Lint").
`./a.ps1 <alias>` is the launcher over the scripts (its header lists every alias).

## Environment (this machine)

- **PowerShell is the shell: `go`, `node` and `npm` work in the PowerShell tool, not in the Bash tool.**
- The Go toolchain here is `GOARCH=386`, so `go test ./tests/` can die with "out of memory" at the 2 GB
  ceiling and pass on a rerun - not a regression; if it persists, set `$env:GOARCH = 'amd64'` in the same call.
- `.gitattributes` pins `*.go` to LF; there is no "normalize to LF first" step.

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

Pick the cheapest path among `.claude/commands/` (`/quick`, `/fix`, `/research`, the `/spec*` chain, `/build` -
local, never publishes - and `/release` - published and paid). The full list:
[agent-commands](.claude/rules/agent-commands.md). Where a method reference conflicts with
[AGENTS.md](AGENTS.md), **AGENTS.md wins**.

## Spec / plan tickets

Read [tickets](.claude/rules/tickets.md) before creating, moving or queueing a ticket.

- One ticket = `DEV/plan/D2HTML-<NNNN>_<slug>.md` (older names stay). The ledger allocates the id; it is never
  reused or changed. Open tickets live in `DEV/plan/`; an `Implemented` or `Verified` one moves to `DEV/plan/done/`.
- Status lives in the ledger `DEV/plan/spec-catalog.jsonl` and changes only through the ticket tool
  (`update.ps1 -Id <id> -Status <s>`, from the `sza` plugin); never edit the Status line by hand.
- `DEV/plan/` is gitignored: search it with the path given explicitly.
- [DEV/plan/RELEASE_QUEUE.md](DEV/plan/RELEASE_QUEUE.md) is the queue: it wins on order, the ledger on status.
  The ticket tool updates it on every status change; check the row and its package after each one.
- Lifecycle: `Draft -> Approved -> Tactical -> In Progress -> Implemented -> Verified` (plus `Partial` / `Broken`
  and `Block*`), from code + checks, never from the filename. No time or effort estimates.
  Full flow: [docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md).

## Research notes

A new note is `DEV/research/RESEARCH_<topic-slug>_<YYYY-MM-DD>.md` (`tests/research_naming_test.go` enforces it);
findings of one ticket go to `DEV/plan/<ticket>/research/`. Detail:
[research-notes](.claude/rules/research-notes.md).

## Persistent memory

Claude Code's native per-user memory is active for this repo (under `~/.claude/projects/.../memory/`
with a `MEMORY.md` index). The discipline - what is worth recording, the entry types, and re-verifying a
remembered path against the live tree before acting on it - is the canon's (`AI_USAGE.md` §4).
