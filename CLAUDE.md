# CLAUDE.md

Claude Code guidance for this repo. **Canonical agent guide: [AGENTS.md](AGENTS.md)** - read it first
(architecture map, build/test/lint, conventions, pitfalls). This file only adds Claude-Code-specific notes,
so the two don't drift. Other docs: [README.md](README.md) (user-facing), [DEV/README.md](DEV/README.md) (dev notes).

The portfolio-wide **SZA Unified Rules** canon (the `sza` plugin, from `github.com/SerZhyAle/sza-unified-rules`) and
this repo's overlay record are pinned in AGENTS.md ("SZA Unified Rules" section) - this repo consumes
them by reference; the delta record is `<canon>/contrib/epub_2_html.md`. The universal rules - evidence
discipline, commit policy, house text style, chat/artifact language, shell safety, build-is-not-a-release -
have one home in the canon and are deliberately not restated here. The canon also ships its own enforcement
hooks with the plugin; this repo registers none of its own and must not re-implement them.

## What this is

`doc-html-translate` - a Go 1.25 Windows app that converts EPUB / PDF / MOBI / AZW3 / FB2 / RTF / TXT /
Markdown / HTML into clean local HTML with generated navigation and a real multi-level TOC, plus optional
translation (Google Cloud or local Ollama). Binaries: `cmd/doc-html-translate` (CLI) and `cmd/doc-html-ui`
(GUI). Conversion runs through [internal/pipeline/pipeline.go](internal/pipeline/pipeline.go) into per-format
`internal/<fmt>` extractors. Signature "free" flow: convert, open `index.html` in Chrome, then use Chrome's
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

## Cross-edition parity

Two independent codebases that share no code: the Go app (CLI/GUI/MSIX) and the JS browser extension
(`extension/src`). Logic is hand-ported Go -> JS, so it drifts. The port map, the invariants that must
match and the intentional differences live in [docs/PARITY.md](docs/PARITY.md), with the process rules in
[AGENTS.md](AGENTS.md) ("Cross-Edition Parity") - read both before adding a user-facing feature.
`scripts/parity-check.ps1` is the advisory structural gate; `tests/parity_test.go` hard-fails on value drift.

## Environment (this machine)

- **PowerShell is the shell, and `go` is on PATH in PowerShell - not in the Bash tool.** Run Go via PowerShell.
- Prefer the `scripts/` flow over ad-hoc commands: `scripts/build.ps1`, `scripts/test.ps1`, `scripts/lint.ps1`,
  `scripts/check.ps1`.
- **Line endings are settled, don't re-litigate them:** `.gitattributes` pins `*.go text eol=lf`, so `gofmt`
  and the gofmt linter agree with the committed source. `gofmt -l .` comes back clean apart from throwaway
  files under the gitignored `temp/`. The old "normalize to LF first" dance is obsolete.
- `scripts/build.ps1` / `scripts/build-ui.ps1` copy artifacts to `C:/GD/tc/SZA/_APP` and need `goversioninfo`;
  this is environment-specific and may fail on other machines.

## Content conventions

- **Typography:** the house text style has one home in the canon (`DOCUMENTATION_CONCEPT.md` §5) and is
  not restated here. The one repo-specific exception is on the next line.
  Applies to generated output, docs, and commit messages. **One deliberate exception:** the converted
  page `<title>Book — Page N</title>` keeps an em dash - that is conventional book typography, not chatter,
  and the drift guard (`tests/typography_test.go`) allowlists an em dash only inside a `<title>` literal.
- Platform-specific code is split via `*_windows.go` / `*_nonwindows.go` - keep both sides in sync.

## High-value invariants (full list in AGENTS.md)

- No-arg CLI enters the **registration** flow, not conversion.
- Existing output is reused when `index.html` exists, unless `-force` is passed.
- Don't change public CLI flag semantics unless explicitly asked.

## Skill routing (slash commands)

Imported from the Universal Agent Kit; the live set is whatever `.claude/commands/` holds. Pick the
cheapest path that fits:

- `/quick` - trivial edit (typo, one constant, one string). No spec, no build gate.
- `/fix` - narrow bug/behaviour fix. Local validation only (`scripts/test.ps1` on the touched package).
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

Subagents: `solution-researcher` (read-only investigator), `implementer` (focused code writer),
`rd-lead-kit` (senior orchestrator - opt-in, not the default agent). Methodology references:
[docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md), [DEV/research/CODE_QUALITY.md](DEV/research/CODE_QUALITY.md),
[DEV/research/VALIDATION.md](DEV/research/VALIDATION.md), [DEV/research/RESEARCH_INDEX.md](DEV/research/RESEARCH_INDEX.md).
Where these conflict with [AGENTS.md](AGENTS.md), **AGENTS.md wins**.

## Spec / plan tickets

- One ticket = one Markdown file `DEV/plan/<YYYY-MM-DD>_<slug>.md`; a ticket with a tactical breakdown gets
  a sibling directory of the same name whose `INDEX.md` is the authority on phase state. The status is the
  first `**Status:**` line in the file; keep it honest by hand.
- **[DEV/plan/RELEASE_QUEUE.md](DEV/plan/RELEASE_QUEUE.md) is the queue** - what is left to do before the
  next release, in execution order, grouped into release packages (`rel` is a package ordinal, never a
  version). Precedence is stated in that file and it is not symmetric: **the queue wins on order, the
  ticket file wins on status.** A ticket that reaches `Implemented` or `Verified` moves to
  `DEV/plan/done/` and its line leaves the table (fix relative links when you move one).
- The queue is maintained by hand - there is no reconcile command and no single write path into it, so a
  status change means editing the ticket and the queue line together. Expect the queue to go stale and
  re-derive from the ticket files when they disagree.
- Lifecycle: `Draft -> Approved -> Tactical -> In Progress -> Implemented -> Verified` (plus
  `Partial` / `Broken` from an audit, and explicit `Block*` states). Status comes from the code +
  checks, never inferred from the filename. No time/effort estimates in spec files.
- Full flow: [docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md).

## Persistent memory

Claude Code's native per-user memory is active for this repo (under `~/.claude/projects/.../memory/`
with a `MEMORY.md` index). The discipline - what is worth recording, the entry types, and re-verifying a
remembered path against the live tree before acting on it - is the canon's (`AI_USAGE.md` §4).
