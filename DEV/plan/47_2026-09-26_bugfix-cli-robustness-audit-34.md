# CLI robustness: overwritten chapters, blocking dialogs, silent failures

**Status:** Implemented
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

## Implementation (2026-09-26)

Every test below was run against the pre-fix code first and failed as quoted, then passed after
the fix. Verified on Linux: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`,
`GOOS=windows GOARCH=386 go build ./...`, `go test ./...` (42 packages ok), `GOARCH=386 go test`
over every touched package, and the extension suite (273 pass, 0 fail).

- **E34** - `internal/epub/normalize.go`: `planContentRenames` claims every manifest href up front
  (renamed sources stay on disk) and gives a rename target that is already claimed, planned or on
  disk the next free `name_N.html` (`unclaimedHref`, case-folded like NTFS). A collision skips the
  intermediate href of the rewrite chain, so links to the untouched `a.html` still reach it.
  Test: `TestExtract_XHTMLRenameAvoidsCollision` (before: "both chapters landed on one href:
  [a.html a.html c.html]").
- **P34** - `internal/dialog/unattended.go` (`SetUnattended`, `logWarning`), both `ShowWarning`
  sides, `internal/app/app.go` (`unattendedRun`). The signal is `-noopen` (the flag's own help says
  "batch mode") or stdout that is not a console (piped into a script or a file). An unattended
  warning goes to the console and the run log instead of a modal MessageBox; a console run with a
  human in front of it keeps the box, and the GUI-hosted path is unchanged. Questions (the cost
  dialog) are deliberately unchanged: they already default to "do not spend", and a batch sets
  `-max-cost`. Tests: `TestUnattendedWarningIsLogged` (before: run log empty),
  `TestUnattendedWarningShowsNoBox` (Windows-only, compiled with `GOOS=windows go test -c`),
  `TestUnattendedRun`. README (EN/RU/UK) `-noopen` row updated. **Owner, on Windows:** run
  `doc-html-translate -noopen <a PDF with JPEG2000 images>` with no ImageMagick/ffmpeg and confirm
  no box opens and the warning is in the console and run log; without `-noopen` the box still
  opens.
- **P36** - `cmd/doc-html-translate/main.go` `reportFailure`: the final error keeps its exact
  stderr text and is also written to the run log as `Run failed (exit code N): ..` (through the
  report redaction filter), so a `-report` bundle says why the run failed. Test:
  `TestReportFailureRecordsRunLog`.
- **P25** - `internal/app/prompt.go`: one buffered stdin reader shared by every prompt and the
  closing pause; `askYes` consumes a whole line. Test: `TestAskYesReadsWholeLine` (before, with
  the old `fmt.Scanln` body: "leftover words of the first answer answered the second prompt"),
  `TestReadLineAnswers`. **Owner, on Windows:** the prompt gates the registry write - run the
  no-argument first-run flow, answer `n yy` to the first question and confirm the second question
  still waits for its own answer and nothing is registered.
- **X29** - `internal/rtf/parse.go`: `readParam` accumulates in int64 and saturates at 2^31-1
  (`maxParamValue`); the `\bin` skip is bounded by what is left instead of computing
  `r.pos+param`. JS twin (`extension/src/rtf.js`) checked: its numbers are doubles, ten digits
  stay far below 2^53 and the skip already uses `Math.min`, so it has no overflow and is not
  changed; the difference (values above 2^31-1) is recorded in docs/PARITY.md "RTF and FB2 text
  decoding". A shared case "huge bin parameter skips to the end" runs on both sides. Test:
  `TestReadParamSaturates`; under `GOARCH=386` the pre-fix code panicked with "index out of range
  [-2147483624]" in `stripRTF`.
- **P30** - the pipeline's run context now reaches every `procrun.Run`: `pdf.Extract`
  (pdftotext, the JPX converter, the Windows blocked-pdftotext retry), `mobi.Extract` (Calibre),
  `comic.Extract` (7-Zip list and extract) and `ocr` (`Recognize` and its rescue passes,
  `DetectScript`, `EngineLangs`/`MissingLangs`). A pdftotext failure on a cancelled run no longer
  falls back to the pure-Go reader, and `pipeline.extractFailed` reports an extraction that failed
  on a cancelled run as interrupted (exit 130). No shipped `procrun.Run` call passes
  `context.Background()` any more; the ocrlab dev tool and `ocr.OverlayFile` (used only by it)
  keep one, having no run to cancel. AGENTS.md's procrun line states the rule. Test:
  `TestCancelStopsHelperAndReportsInterrupted` (before: "code = 3 (extract mobi: ebook-convert did
  not finish within 5s ..)" and "the run took 5.006s after a cancel at 300ms").
- **E27** - `internal/limits`: `MaxTextInputBytes = 100 << 20` and `ReadTextInput` (refuses from
  the size on disk before the read, and caps the read against a file that grows). TXT, Markdown,
  FB2, RTF and HTML read through it. Extension: `TEXT_MAX_INPUT_BYTES` and `checkTextInput` in
  `limits.js`, applied in `viewer.js` `loadFromData` before the parser, with a `vLimitText` message
  in all 13 locales (Go message in `i18n_limits.go`). Pinned by `TestParityInputLimits`, listed in
  docs/PARITY.md "Input limits" and in README (EN/RU/UK) and extension/README. Tests:
  `TestTextInputsRefuseOversizedFiles` (sparse file one byte over the budget; before: `.txt` and
  `.rtf` converted it, `.fb2` failed as an XML syntax error), JS `checkTextInput refuses a
  document over the text budget`. **Owner decision:** 100 MB was chosen to equal the EPUB
  per-file cap, since an EPUB chapter goes through the same whole-file HTML path; change it in both
  editions and the parity test together if a different number is wanted.
