# Validation Ladder & Post-Change Discipline — "done" means evidence

A step is not done when you *intended* it to work. It is done when a check passed **in this
run**. The gap between those two is where most regressions live. This document is the rule that
closes it: every change ends with the weakest piece of evidence strong enough to prove it, and
not a heavier one.

## The ladder

Match the evidence to the kind of change. Climbing higher than necessary wastes time; climbing
lower than necessary ships a bug.

- **Doc / text only** → grep the file for the content you claim you wrote. If the words are
  there, the change is real.
- **Script** → run it; it exits `0` (or produces the expected artifact). A script you wrote and
  did not run is a guess.
- **Config / build files** → the target build passes. Config that "looks right" but breaks the
  build is the most common self-inflicted wound.
- **Code** → the *narrowest meaningful* check:
  - a compile / type-check for a pure symbol or signature change,
  - a targeted test for changed logic,
  - a full build or run only when packaging, resources, or end-to-end behaviour actually need
    proof.
  Do not run the whole suite to validate a one-line rename; do not claim a behaviour fix with
  only a compile.
- **User-visible behaviour** → run it and observe. Compiling is not behaving.

The rule of thumb: **grep < run-script < compile < targeted test < full build < run-and-observe.**
Pick the lowest rung that actually proves *this* change, and stop there.

## Record expected vs actual

For every check you run, write down what you expected and what you got:

```
expected: exit 0, "3 files written"
actual:   exit 0, "3 files written"   → PASS
```

This is not bureaucracy. Predicting the result *before* reading it is what catches the check
that exited `0` while printing the wrong thing — the failure that a glance at "it ran" would
have missed. A check whose `actual` you did not read did not happen.

`FAIL` on any check means the step is not done: record the blocker and stop. Never auto-revert
and never paper over a failure by moving on — a human decides from the evidence.

## Post-change discipline

A change is not just its diff. Before calling a step complete, run the housekeeping the project
needs so it is never "remembered later":

- **Changelog / dev log** — append the entry if the project keeps one.
- **User-facing docs** — update them for any new user-visible capability, in every language the
  docs ship in. Do this *before* marking the step done, not in a cleanup pass that never comes.
- **Code index** — regenerate it if you changed code (see `RESEARCH_INDEX.md`); a stale index
  misdirects the next search.
- **Ticket status** — move the spec's status to match reality (see `SPEC_LIFECYCLE.md`).

Bundle these into one routine (a script, a hook, a checklist) so they ride along with the change
instead of depending on memory.

## A lightweight progress journal (optional)

For multi-step work it helps to keep a human-readable journal — one concise entry per step, with
raw output kept *out* of it:

```
[STEP 4.2] add RetryPolicy to UploadService
changed:    upload/UploadService.ext, upload/RetryPolicy.ext
validation: <test command> → PASS
evidence:   scratch/sessions/20260314_upload_test.txt
blocker:    none
next:       4.3
```

`validation` must name the actual command or predicate, not "verified" or "checked". Full build
logs and grep dumps go to a scratch directory and are *referenced* from `evidence:`, never pasted
into the journal — the journal stays scannable, the raw proof stays available.

## Why this is a document, not a vibe

Each rung of the ladder is a concrete, repeatable action a reviewer — human or agent — can demand
and re-run. "Done" stops being a feeling and becomes a line you can point at: *which check, what
did it print, was it PASS.* The ladder also keeps the ceremony proportional: a typo does not earn
a full build, and a migration does not get waved through on a compile. Make the check part of
"done", not part of "someday".

## Adapting it

- Map each rung to your stack's real commands (your compiler, your test runner, your build, your
  run/launch command).
- If your CI already gates merges, the local ladder is your *fast feedback* before CI — it should
  be a subset you can run in seconds-to-a-minute, not a duplicate of the full pipeline.
- Drop the journal if your tasks are small; keep it the moment a task spans more than a handful of
  steps or more than one session.
