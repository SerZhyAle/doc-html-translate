# CLI diagnostics report things that are not true, and one path hangs

**Status:** Implemented - 2026-07-17; all three items landed and verified by running the CLI, not by reasoning
**Priority:** 1
**Date:** 2026-07-17

> **What the implementation found that this ticket had wrong.** Recorded because both corrections
> matter more than the items as written.
>
> 1. **Item 1 named one hang; there are four.** The pause is not specific to "the flag package's
>    error path" - it is every unconditional `Scanln`. The worst is `internal/pipeline`, which pauses
>    identically after a **translation failure**: a scripted conversion whose translation broke hung
>    forever. This sweep could not have seen it - every run used `-notranslate`. Also gated: the
>    `internal/app` splash. Deliberately **not** gated: `promptSetDefault`, which asks a question
>    rather than pausing. A pause serves a console (gate on stdout); a question needs an answer
>    (depends on stdin) and already takes the safe "no" on EOF. Gating it on stdout would have made
>    `echo y | app > out.txt` stop registering.
> 2. **Fixing the hang did not fix the reported command.** `-h` also exited **1** with the help on
>    **stderr** under an `Error: flag: help requested` banner, so `-h | Select-Object -First 60`
>    returned *empty*. Owner-approved as a semantic change (2026-07-17): `-version` in the same binary
>    already exits 0 via a sentinel, so `-h` was the outlier, not the convention. Now: `-h` -> usage on
>    stdout, exit 0; a real flag error -> stderr, exit 1, unchanged.
>
> **The harness lied first, and that is worth remembering.** A `Start-Job` wrapper reported "no hang"
> for the *pre-fix* binary: a background job hands the child an already-closed stdin, so `Scanln` gets
> EOF and returns even when the bug is present. Reproducing it needs stdout redirected **and** a stdin
> pipe held open and never written - an idle console, in other words. Under that harness the pre-fix
> binary hangs on `-h`, `-bogus` and "no input file" alike.
>
> Filed on the way: [`2026-07-17_output-typography`](../2026-07-17_output-typography.md) - the sweep for
> the `..`/long-dash rule turned up ~42 violations in user-visible strings across 11 files, including
> `<title>%s — Page %d</title>` in five extractors. Out of scope here; not fixed by stealth.

## What / why

Defects found while converting the new test corpus (2026-07-17). Small individually; together they
mean the CLI's own output cannot be trusted to say what happened, which is exactly what a user falls
back on when something looks wrong. Desktop-only - none has an extension analogue.

### 1. The error path blocks forever when the output is piped

Any invocation that ends in the flag package's error path prints usage and then waits on
`Press Enter to close...`. That prompt is right for a double-clicked exe, but it fires regardless of
whether stdout is a console. Piped or redirected - CI, a script, a wrapper - the process hangs until
killed. Found by running `doc-html-translate.exe -h | Select-Object -First 60`, which never returned.

`logging.Progress` already makes exactly this distinction (`\r` on a TTY, `\n` off it), so the
convention exists and this path just does not follow it.

### 2. "Pages: N (with text: N)" counts pages that have no text

Converting a scan with no text layer at all logs, in the same run:

```
WARNING: pdftotext failed, falling back to pdflib: pdftotext extracted no content
Images: 6 extracted across 6 pages
Pages: 6 (with text: 6)
```

Six pages "with text" immediately after reporting that no text was extracted. Whatever the counter
means (pages carrying *any* extracted content, images included), it is not what it says, and it says
it directly under the line that contradicts it. The reader learns to distrust the log.

### 3. Images that fail to overlay disappear silently

The same run: `Images: 6 extracted` then `OCR overlay: 5 image(s) overlaid`. One image produced no
overlay and nothing said which, or why. The count is the only evidence, and only if someone
subtracts. This is the same shape as the `8/9` seen on `Kupní smlouva ..pdf` (2026-07-16), where the
missing images were the *whole* failure and the count was the only clue - see the non-ASCII path
note in `DEV/CHANGELOG.md` and `test_doc/CORPUS.md`.

A skipped image is normal (no text found in it). A *failed* image is not. The log does not
distinguish them, so the normal case hides the broken one.

**How expensive that is, measured 2026-07-17.** `Aphrodite's Mirror (1).pdf` logs:

```
OCR overlay: 2304 images in 54.2s (42.5/s)
OCR overlay: 1711 image(s) overlaid
```

593 pages - a quarter of the book - produced nothing. From the app's own output there is no way to
tell whether that is a catastrophe or a Tuesday. Establishing that it is **correct** (the book is a
rendered graphic novel and those 593 pages are art panels with no dialogue) took extracting the
per-image plate counts, measuring the file-size distribution of the two groups, and finally **opening
the images and looking at them**. The size distributions are no help - zero-plate pages have a 152 KB
median against 173 KB for the rest, i.e. they are ordinary full-detail artwork, not blank leaves.

That is the cost of the missing distinction: on this book the gap is entirely benign, on
`Kupní smlouva ..pdf` an identically-shaped gap was the entire bug, and the log renders the two
indistinguishable. Naming the reason per image ("no text found" vs "recognition failed: ..") is what
turns a number a reader must audit by hand into a number they can trust at a glance.

### 4. An unreadable input is reported as a successful 100-page book

**Moved to its own ticket:**
[`2026-07-17_binary-input-becomes-a-garbage-document`](2026-07-17_binary-input-becomes-a-garbage-document.md).

It started here as one CBZ observation, but the 2026-07-17 sweep measured the same failure across
eight extensions (`.docx` `.odt` `.pptx` `.xlsx` `.djvu` `.cbz` `.cbr` `.cb7` `.cbt`), and the fix is
input validation rather than reporting - a different change, in a different file, with its own parity
story (the extension already gets it right). Items 1-3 below stay here: they are about the log telling
the truth.

## Notes

- The corpus and repro steps: `test_doc/CORPUS.md`. Item 2 and 3 both reproduce on
  `test_doc/comic-scan-tiny_First-Earthman-on-Mars-1944.pdf` in about a second.
- Item 3 overlaps the non-ASCII path failure only in symptom, not cause. Keep them separate: this
  ticket is about *reporting*, not about the underlying staging gap in `internal/ocr`.

## Done criteria

- [x] `-h` and every flag-error path exit without prompting when stdout is not a console. Went wider:
      all four unconditional pauses, including the translation-failure one this sweep could not see.
- [x] The page counter says what it counts, or stops being printed when it cannot. Now
      `Pages: 6 (with text: 0, image-only: 6)`, covered by `TestPageCountSummary`.
- [x] An image that fails to overlay is named with its reason; one that legitimately has no text is
      distinguished from one that errored. `classifyRecognition` + `ocr.OverlayResult`, covered by
      `TestClassifyRecognition`.
- [x] Verified by running the CLI piped and reading the output, not by reasoning about it - after the
      first harness turned out to be reporting a false pass (see the status note).
- [x] Tests / gate green (`scripts/test.ps1`) - lint green; `go test ./...` green except the
      pre-existing `TestExtract_NoCalibre`, which asserts Calibre is *absent* and fails identically on
      a clean tree (this machine has it installed).
- [x] Changelog entry in `DEV/CHANGELOG.md`.

## Left open by this ticket

- **`TestExtract_NoCalibre` fails on any machine that has Calibre installed.** Unrelated to these
  items, but it means `go test ./...` is not green out of the box here, which is exactly the kind of
  untrustworthy signal this ticket is about. Worth its own small ticket.
- **The duplicate error line on a bad flag.** `-bogus` prints the flag package's own message *and*
  main's `Error: ..` line, so the same sentence appears twice. Pre-existing, cosmetic, and left alone
  to keep this diff reviewable.
