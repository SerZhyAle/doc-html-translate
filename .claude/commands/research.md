# Research Guide

> **GLOBAL DIRECTIVES (anti-bureaucracy):**
> 1. Dry technical prose, no filler.
> 2. Autonomy: silently fix minor inaccuracies; block only for critical decisions.
> 3. Terse report: one dry statement with the dossier path + next high-value reads.

A repeatable research pass before dev, docs, or cross-surface debugging. Build a scratch
dossier first, then narrow into code. The point: read deliberately once instead of
grepping the same things five times mid-implementation.

## Usage

```text
/research <topic or question>
/research <ticket-id> <topic>     # ticket-bound: persist findings under the ticket
```

- `/research player startup`
- `/research where does auth-token refresh happen?`
- `/research T0123 best strategy for full-text search over filenames`

## Process

**Step 1 — Anchor the topic.**
- Use the user's explicit target when given.
- A token shaped like a ticket id (e.g. `^[A-Z]\d{4}$`) → ticket-bound run: findings will
  be persisted (Step 5). The rest of the text is the topic.
- Infer the affected area/module from the request and the currently open file.

**Step 2 — Build a dossier first** (before broad reading). Write a Markdown scratch file to
`temp/research_<topic-slug>.md` collecting, from a *quick* sweep:
- the most relevant docs to read first;
- the files/symbols that match the topic (one grep/code-index pass, not exhaustive);
- existing specs/plans touching the area;
- a short list of suggested next reads.

This is scratch — it is allowed to be rough. Its job is to stop repeated global greps.

**Step 3 — Follow the routing stack in order** (unless the dossier shows a tighter first
read):
1. Repo map / index doc.
2. Architecture doc.
3. Build / ops doc.
4. Dependency / stack doc.
5. Your code index or `DEV/plan/` for prior decisions.

**Step 4 — Drill into implementation files.**
- The smallest set of follow-up reads that answers the question.
- Use the dossier to avoid re-grepping.
- Keep cross-surface questions grounded in the dossier sections.

**Step 5 — Persist findings (ticket-bound runs only).**
- Write curated findings — conclusions, chosen option, rejected options *with reasons*,
  affected areas — to `DEV/plan/YYYY-MM-DD_<slug>_<slug>/research/<NN>__<topic-slug>.md`. Create the
  folder if missing.
- The scratch dossier stays scratch; the artifact is the durable result a tactical plan
  will consume. Raw grep dumps stay out of it.

## Output

- Report the dossier path in `temp/`.
- Ticket-bound: report the artifact path under `DEV/plan/`.
- List the next 3–6 high-value reads.
- Answer the direct question after the dossier-backed reads.
- If a section had no matches, say so and continue — absence is a finding.
