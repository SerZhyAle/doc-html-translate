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
| 2026-08-15 | [`ocr_rescue_floor_2026-08-15.md`](ocr_rescue_floor_2026-08-15.md) | `ocrRescueLineConf` was set between two points from one cycle. What does the distribution look like over the corpus the app now has, and what floor does it support? | Ticket [`2026-08-13_ocr-rescue-floor-drops-genuine-lettering`](../plan/2026-08-13_ocr-rescue-floor-drops-genuine-lettering.md). **Refutes the premise that a floor can be re-derived at all:** genuine rescued lettering runs 32.8-69.2 and invented lettering 8.4-73.9 - they overlap, so no single confidence separates them. What does is length, which brackets an empty band (`allie` 36.1 / `KPECTbAHHH!` 58.3) and gives the shipped pair 47 + 4 letters. Also builds the instrument the ticket needed: the floor's own discard record. |
| 2026-08-15 | [`ocrlab/2026-08-15__extension-parity-run.md`](ocrlab/2026-08-15__extension-parity-run.md) | Does the browser edition behave the way the desktop edition it was ported from does, measured rather than pinned by constants? | Closes the open evidence item of [`2026-08-13_ocr-sweep-plate-composition`](../plan/2026-08-13_ocr-sweep-plate-composition.md). Concealment equal or better in the extension on every scene that plates; three scenes differ, two of them extension defects (a protected-outline blocker, a caption missed by tesseract.js). **Also finds a scorer defect**: a run that wrote no post-overlay render scores concealment `0`, so "recognized nothing" reads as "concealed perfectly" on the desktop side. |
| 2026-08-13 | [`ocr_plate_coverage_2026-08-13.md`](ocr_plate_coverage_2026-08-13.md) | What actually separates a plate that has merged unrelated regions from a plate that is simply large? And how good is Tesseract's script detector on real material? | Ticket [`2026-08-13_ocr-sweep-plate-composition`](../plan/2026-08-13_ocr-sweep-plate-composition.md). **Refutes the measure that ticket named** - a plate's box fill by area puts the defect above four scenes that must not be broken - and brackets the pair that does work (`0.52` coverage, `0.72` vertical line fill) plus the script floor (`6.4`) from opposite directions. |
| 2026-08-13 | [`ocr_sweep_2026-08-13.md`](ocr_sweep_2026-08-13.md) | 21 corpus documents converted in the shipping OCR mode and read as rendered pages: what does a reader actually get, on inputs the automatic checks all pass? | Ticket [`2026-08-13_ocr-sweep-plate-composition`](../plan/2026-08-13_ocr-sweep-plate-composition.md). Settles that recognition is not the failing part - composition is: one plate reaches **80.6% of its image**, a plate's text lands over an unrelated region, and plates do not conceal what they replace. Also finds a corrupt PDF raster and a false FAIL in `verify-html.ps1` on EPUB output. |
| 2026-08-12 | [`ocrlab/2026-08-11__baseline.md`](ocrlab/2026-08-11__baseline.md) | What is the first measured state of OCR visual fidelity, in both editions, across all eight of the strategic table's dimensions - and what may honestly be turned into an acceptance bound? | Ticket [`2026-08-11_ocr-visual-fidelity-lab`](../plan/2026-08-11_ocr-visual-fidelity-lab.md) **Phase 06** (it *is* steps 06.1-06.2, and `DEV/ocrlab/thresholds.json` is derived from it) and **Phase 07**, which must cite one of its tables for every change it makes. |
| 2026-08-12 | [`ocr_positioning_exchange_2026-08-12.md`](ocr_positioning_exchange_2026-08-12.md) | An outside project asked how our overlay decides reading order, direction, hyphenation and rotation, and what positioning error we accept. What does the shipped code actually do, and what does the acceptance gate actually bound? | Ticket [`2026-08-12_ocr-exchange-followups`](../plan/2026-08-12_ocr-exchange-followups.md). Settles that reading order is entirely the engine's, that no rotation or dehyphenation exists, and that the gate bounds **drift only** - which is how a defect that put every plate on a 1.9x-wrong picture passed at 0 px drift. Both defects it found are fixed. |
| 2026-08-12 | [`ocr_display_lettering_2026-08-12.md`](ocr_display_lettering_2026-08-12.md) | A poster comes back with one stray word on it. Which part of the shipped path loses the other ten, and does anything already in the code recover them? | Ticket [`2026-08-12_ocr-misses-display-lettering-on-saturated-art`](../plan/done/2026-08-12_ocr-misses-display-lettering-on-saturated-art.md). Settles that the ladder is reached and still fails, that the loss is at the line-mean gate, and that a sparse-text rung recovers six of nine lines; **corrects the ticket's own first cause**. |
| 2026-08-12 | [`ocr_halftone_2026-08-12.md`](ocr_halftone_2026-08-12.md) | Lettering printed over a halftone screen produces no plates at all and the grey rescue ladder does not recover it. What does? | Ticket [`2026-08-11_ocr-halftone-defeats-recognition`](../plan/done/2026-08-11_ocr-halftone-defeats-recognition.md), now done. |
| 2026-08-11 | [`ocr_grey_rescue_2026-08-11.md`](ocr_grey_rescue_2026-08-11.md) | 14 of 40 scenes produced no plates at all. Why, and what is the smallest correction that recovers them without regressing anything? | The `ocrRescueLineConf` ladder in `internal/ocr/tesseract.go`. **Its number is to be re-derived, not inherited** - it rests on 8 annotated dev scenes, not on the corpus. |

Add a row when you write an artifact. An artifact nobody can find is one that will be written twice.

## Adapting it

- Replace the named docs with whatever your repo actually has as its map.
- No index tooling yet? Start with grep and the order above; add an index only when the codebase
  has outgrown search.
- If your stack has a language server or symbol index already, that *is* your code index — query
  it first and skip the custom one.
