# CLAUDE.md

Claude Code guidance for this repo. **Canonical agent guide: [AGENTS.md](AGENTS.md)** - read it first
(architecture map, build/test/lint, conventions, pitfalls). This file only adds Claude-Code-specific notes,
so the two don't drift. Other docs: [README.md](README.md) (user-facing), [DEV/README.md](DEV/README.md) (dev notes).

The portfolio-wide **SZA Unified Rules** canon (the `sza` plugin, from `github.com/SerZhyAle/sza-unified-rules`) and
this repo's overlay record are pinned in AGENTS.md ("SZA Unified Rules" section) - this repo consumes
them by reference; the delta record is `<canon>/contrib/epub_2_html.md`.

## What this is

`doc-html-translate` - a Go 1.25 Windows app that converts EPUB / PDF / MOBI / AZW3 / FB2 / RTF / TXT /
Markdown / HTML into clean local HTML with generated navigation and a real multi-level TOC, plus optional
translation (Google Cloud or local Ollama). Binaries: `cmd/doc-html-translate` (CLI) and `cmd/doc-html-ui`
(GUI). Conversion runs through [internal/pipeline/pipeline.go](internal/pipeline/pipeline.go) into per-format
`internal/<fmt>` extractors. Signature "free" flow: convert, open `index.html` in Chrome, then use Chrome's
built-in page translation (no API key needed).

## Build vs Release ("сборка" vs "релиз")

Canonical: [DEV/RELEASE.md](DEV/RELEASE.md). **Build** = local, free: `./scripts/build-local.ps1
-Message "..."` (gate + build CLI + UI + commit; never touches GitHub). **Release** = published,
paid: `./scripts/release.ps1` prints the full checklist and runs nothing; `v*` / `ext-cws-v*` /
`ext-edge-v*` tags trigger paid CI. Treat any "сборка"/"build" request as the local flow - never push a tag, submit to
winget, upload to the Store, or publish the extension unless the request is explicitly a release.

## Where to start for a change

Trace from [internal/pipeline/pipeline.go](internal/pipeline/pipeline.go) into the format package
(`internal/epub`, `internal/pdf`, ..). Entry points: [cmd/doc-html-translate/main.go](cmd/doc-html-translate/main.go),
app wiring in [internal/app/app.go](internal/app/app.go) and [internal/config/flags.go](internal/config/flags.go).

## Cross-edition parity

Two independent codebases, no shared code: the Go app (CLI/GUI/MSIX) and the JS browser extension
(`extension/src`). Logic is hand-ported Go -> JS, so it drifts. **[docs/PARITY.md](docs/PARITY.md) is
the source of truth** for the port map, the invariants that must match (theme palette, PDF reflow
constants, EPUB TOC rules, OCR host/catalog/classes, defaults), and the intentional differences (don't
"fix" them). Any new user-facing feature is **one cross-edition ticket** (template
[DEV/plan/_TEMPLATE_cross-edition.md](DEV/plan/_TEMPLATE_cross-edition.md)) covering every edition;
update docs/PARITY.md when you change a shared invariant. Open gaps:
[DEV/plan/2026-07-01_cross-edition-parity.md](DEV/plan/2026-07-01_cross-edition-parity.md).

## Environment (this machine)

- **PowerShell is the shell, and `go` is on PATH in PowerShell - not in the Bash tool.** Run Go via PowerShell.
- Prefer the `scripts/` flow over ad-hoc commands: `scripts/build.ps1`, `scripts/test.ps1`, `scripts/lint.ps1`,
  `scripts/check.ps1`.
- **CRLF gotcha:** the working tree is CRLF, so `gofmt -l .` flags every file. Don't trust it as a signal of
  "unformatted" - normalize to LF first, or rely on `scripts/lint.ps1`.
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

Imported from the Universal Agent Kit. Pick the cheapest path that fits:

- `/quick` - trivial edit (typo, one constant, one string). No spec, no build gate.
- `/fix` - narrow bug/behaviour fix. Local validation only (`scripts/test.ps1` on the touched package).
- `/research` - investigate before any non-trivial change; persist findings under `DEV/research/`.
- `/spec` - write the strategic *what/why* for a feature (no file paths, no symbols).
- `/spec-tech` - break an approved spec into a phased, verifiable plan.
- `/spec-dev` - execute a tactical plan one step at a time, checking each.
- `/spec-check` - audit implementation against the spec; set status from reality.
- `/spec-fix` - apply the audit's mechanical action items, then re-audit.
- `/git` - branch model, staging, commit grouping (honour the CRLF + typography rules above).
- `/build` - "сборка": local+free flow (gate, build CLI+UI, changelog entry, commit). Never publishes.
- `/release` - "релиз": published+paid flow. Curates "What's new in version XXX" and drives the full
  checklist (GitHub release, winget, Store, extension) with per-step confirmation. See [DEV/RELEASE.md](DEV/RELEASE.md).

Subagents: `solution-researcher` (read-only investigator), `implementer` (focused code writer),
`rd-lead-kit` (senior orchestrator - opt-in, not the default agent). Methodology references:
[docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md), [DEV/research/CODE_QUALITY.md](DEV/research/CODE_QUALITY.md),
[DEV/research/VALIDATION.md](DEV/research/VALIDATION.md), [DEV/research/RESEARCH_INDEX.md](DEV/research/RESEARCH_INDEX.md).
Where these conflict with [AGENTS.md](AGENTS.md), **AGENTS.md wins**.

## Spec / plan tickets

- One ticket = one Markdown file `DEV/plan/<YYYY-MM-DD>_<slug>.md`. The status is the first
  `**Status:**` line in the file; keep it honest by hand.
- **[DEV/plan/ROADMAP.md](DEV/plan/ROADMAP.md) is the queue.** `**Priority:** N` is a position in it -
  lower runs first. Finished tickets move to `DEV/plan/done/` (fix relative links when you move one).
  When priorities and a ticket's own prose disagree, the prose usually wins - it was written against
  evidence.
- Lifecycle: `Draft -> Approved -> Tactical -> In Progress -> Implemented -> Verified` (plus
  `Partial` / `Broken` from an audit, and explicit `Block*` states). Status comes from the code +
  checks, never inferred from the filename. No time/effort estimates in spec files.
- Full flow: [docs/SPEC_LIFECYCLE.md](docs/SPEC_LIFECYCLE.md).

## Persistent memory

Claude Code's native per-user memory is already active for this repo (under
`~/.claude/projects/.../memory/` with a `MEMORY.md` index). Record only durable, non-obvious context
(work preferences, decision rationale, corrections **and** confirmations) - never anything derivable
from the repo, `git log`, or this file. Verify a remembered path/symbol against the repo before acting
on it. Discipline and the four entry types: the kit's `docs/AGENT_MEMORY.md` (not committed here).
