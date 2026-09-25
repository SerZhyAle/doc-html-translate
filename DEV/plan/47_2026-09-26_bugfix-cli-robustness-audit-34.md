# CLI robustness: overwritten chapters, blocking dialogs, silent failures

**Status:** Draft
**Priority:** 60
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **E34** - the XHTML-to-HTML rename has no collision check: `a.xhtml` beside `a.html` overwrites a
  chapter (proved with a throwaway test).
- **P34** - on a console run the GUI does not host, a warning is a modal MessageBox, so a PDF with
  JPEG2000 images or a blocked pdftotext halts a `-noopen` batch until someone clicks.
- **P36** - a failed run's final error goes only to stderr; the run log and the `-report` bundle do not
  say why the run failed.
- **P25** - `askYes` leaves unread words for the next prompt: "n yy" registers the default handler.
- **X29** - on the 386 build, RTF `\bin2147483647` overflows an int and panics (executed).
- **P30** - helper processes run under `context.Background()`; a cancel waits for their deadline, and an
  extraction killed by Ctrl+C is reported as a parse failure (the deferral recorded in done/11).
- **E27** - Markdown and the other whole-file text inputs have no size budget on the 386 build.

## 2. Goals

1. No generated name overwrites a source file; a collision gets a unique name.
2. Warnings never block an unattended run.
3. The run log records the error that ended the run.
4. Prompts read a whole line.
5. RTF parameter arithmetic is bounded on 32-bit builds, and the JS twin checked for the same.
6. The run's context reaches every helper process.
7. Whole-file text inputs have a size budget, pinned in both editions' limits.

## 3. Constraints

- The prompt in P25 gates a registry write: the change is proven on Windows by the owner.
- Limits are a parity invariant (docs/PARITY.md "Input limits").

## 4. Acceptance

- One Go test per item, failing before and passing after, output cited.
