# Research Order & the Code Index — never guess

The most expensive thing an AI assistant does is *guess*: invent a file path, assume a function
signature, recall an API that changed two versions ago. A guess looks like progress and costs a
full wrong implementation. The cure is a fixed order for finding things, and — for a large
codebase — a maintained index so finding is one query instead of a blind grep.

The one rule under all of it: **if you state a path, a symbol, or an API, you have verified it.**

## The research order

Read in this order and stop as soon as a source answers the question:

1. **The map.** The repo's index/overview doc — `README`, `ARCHITECTURE.md`, an operations
   index, a feature-to-path map. This tells you *where* to look before you look.
2. **The spec / plan.** If the work is ticket-bound, the spec under your plan directory holds
   the decisions and constraints already made. Don't re-derive them.
3. **The code index, then the code.** Locate symbols with your code index or grep *before*
   reading whole trees. Find the file, then read that file — not the directory.
4. **External docs.** Official framework docs, changelogs, issue trackers — when the answer is
   version-specific or about third-party behaviour. Use them freely; this is not cheating and
   needs no permission.

Each rung is cheaper to consult than the one below it is to get wrong. The discipline is to
climb, not to jump straight into reading source or — worse — straight into writing it.

## The code index (for codebases big enough to get lost in)

Grep is fine until the tree is large, names collide, or "where does X live" takes five searches.
At that point, maintain an **index**: a generated list of the codebase's units (classes,
modules, files) with, for each, its path and a short role — and optionally what it depends on or
is injected with. The agent queries the index ("show me everything matching `*Repository`",
"what plays the role `data-source`") and gets the location in one shot.

Two rules keep an index trustworthy:

- **Query it before you grep.** For "where is class/module X", the index answers faster and more
  precisely than a content search.
- **Regenerate it after you change code.** A stale index is worse than none — it sends you to a
  path that moved. Make regeneration a step in your post-change routine, or a hook, so it is
  never skipped. The index is a derived artifact: regenerate, don't hand-edit, and it can be
  git-ignored.

This is optional. A small or flat repo does not need it — the research *order* above still
applies. Adopt the index when "find where this lives" has become a tax.

## Work in parallel

Independent lookups should run concurrently, not in sequence. A local symbol search and an
external docs fetch answering the same question start in the same breath — never wait for one
before kicking off the other. If your runtime has sub-agents, fan out: one reads the local code,
another reads the framework docs, a third checks open issues. You spend wall-clock once instead
of three times.

## Persist what you find

Research you did and then dropped on the floor gets done again next session. For anything beyond
a trivial lookup, write the findings to a scratch file (or into the spec, for ticket-bound work):
the files and symbols touched, exact locations, the relevant external references, and the open
questions. The next step — and the next session — reads the notes instead of re-grepping.

## Why this is a document, not a vibe

"Don't guess" is easy to nod at and hard to keep under time pressure. Making it a written order
turns it into something a reviewer can check: *did the change cite a verified path, or assume
one?* And making the index a generated artifact turns "I think it's over there" into a query
with an answer. The goal is that every claim in a change is one the agent could point at, not one
it hoped was true.

## Research artifacts in this repo

Findings already written down, so the next session reads them instead of re-deriving them. Each
answers one question and unblocks one piece of work.

| Date | Artifact | Question it answers | What it unblocks |
|---|---|---|---|
| 2026-08-12 | [`ocrlab/2026-08-11__baseline.md`](ocrlab/2026-08-11__baseline.md) | What is the first measured state of OCR visual fidelity, in both editions, across all eight of the strategic table's dimensions - and what may honestly be turned into an acceptance bound? | Ticket [`2026-08-11_ocr-visual-fidelity-lab`](../plan/2026-08-11_ocr-visual-fidelity-lab.md) **Phase 06** (it *is* steps 06.1-06.2, and `DEV/ocrlab/thresholds.json` is derived from it) and **Phase 07**, which must cite one of its tables for every change it makes. |
| 2026-08-12 | [`ocr_halftone_2026-08-12.md`](ocr_halftone_2026-08-12.md) | Lettering printed over a halftone screen produces no plates at all and the grey rescue ladder does not recover it. What does? | Ticket [`2026-08-11_ocr-halftone-defeats-recognition`](../plan/done/2026-08-11_ocr-halftone-defeats-recognition.md), now done. |
| 2026-08-11 | [`ocr_grey_rescue_2026-08-11.md`](ocr_grey_rescue_2026-08-11.md) | 14 of 40 scenes produced no plates at all. Why, and what is the smallest correction that recovers them without regressing anything? | The `ocrRescueLineConf` ladder in `internal/ocr/tesseract.go`. **Its number is to be re-derived, not inherited** - it rests on 8 annotated dev scenes, not on the corpus. |

Add a row when you write an artifact. An artifact nobody can find is one that will be written twice.

## Adapting it

- Replace the named docs with whatever your repo actually has as its map.
- No index tooling yet? Start with grep and the order above; add an index only when the codebase
  has outgrown search.
- If your stack has a language server or symbol index already, that *is* your code index — query
  it first and skip the custom one.
