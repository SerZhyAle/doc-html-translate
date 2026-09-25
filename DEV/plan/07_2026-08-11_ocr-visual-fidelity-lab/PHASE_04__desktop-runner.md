# Phase 04 - Desktop runner

**Strategic spec:** [`../07_2026-08-11_ocr-visual-fidelity-lab.md`](../07_2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 6 / 6

## Objective

One command that turns the corpus into comparable evidence for the CLI/GUI edition: recognized text,
plate geometry per viewport, before/after screenshots, translation-stress captures, scores and a report
a reviewer can open.

## Prerequisites

- [x] Phase 03 is ✅ Done (the `evidence` schema and every metric exist).
- [x] `tesseract` resolvable by `ocr.Locate()`; Chrome or Edge installed.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/diag.go` | New | ≤ 160 |
| `internal/ocr/diag_test.go` | New | ≤ 160 |
| `internal/ocr/overlay.go` | Modified | ≤ 720 |
| `tools/ocrlab/runner/stress.go` | New | ≤ 160 |
| `tools/ocrlab/runner/browser.go` | New | ≤ 360 |
| `tools/ocrlab/runner/run.go` | New | ≤ 400 |
| `tools/ocrlab/report/report.go` | New | ≤ 400 |
| `tools/ocrlab/main.go` | Modified | ≤ 400 |

## Steps

### Step 04.1 - Add a diagnostics export to `internal/ocr`

**Files:** `internal/ocr/diag.go`, `internal/ocr/diag_test.go`, `internal/ocr/overlay.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add an opt-in diagnostic sidecar so the runner reads the app's real geometry instead of
> re-implementing it. Export `func DiagnosticsPath() string` reading the `DOCHT_OCR_DIAG` environment
> variable (empty = disabled) and, when set, have `OverlayBook` append one JSON line per overlaid image
> to that file: image path, `Result.Width`/`Height`, and for each block its text, box, `LineH`, the
> computed percent style and the sampled `bg`/`ink`. Normal user output must be byte-identical when the
> variable is unset - assert that in the test by running the overlay twice, once with and once without,
> and comparing the rewritten HTML.

**Verification:**
- `func DiagnosticsPath() string` matches exactly once.
- `DOCHT_OCR_DIAG` appears in `internal/ocr/diag.go` exactly once.
- `internal/ocr/diag_test.go` asserts identical HTML output with and without the variable set.

**Status:** `[x]` done

---

### Step 04.2 - Define the deterministic stress cases

**Files:** `tools/ocrlab/runner/stress.go`
**Depends on:** Step 04.1

**Prompt for developer:**
> Declare the stress set as constants: `none` (the recognized text), `short`, `long-latin`,
> `long-cyrillic`, `rtl-arabic` and `cjk`, each a fixed string plus a length multiplier applied by
> repeating the string until it reaches roughly `N x` the source text length. No translation service is
> called; these are literals, which is what makes the geometry check reproducible.

**Verification:**
- `var StressCases = []StressCase{...}` declares exactly the six named cases.
- No file under `tools/ocrlab/` imports `internal/translator` or performs an outbound request other
  than `corpus.Fetch`.

**Status:** `[x]` done

---

### Step 04.3 - Drive headless Chrome and capture

**Files:** `tools/ocrlab/runner/browser.go`
**Depends on:** Step 04.2

**Prompt for developer:**
> Implement the capture step against the same headless browser `scripts/verify-html.ps1` locates (Edge
> first, Chrome fallback, `CHROME`/`DOCHT_BROWSER` override). For each scene and each pinned viewport
> (`1280x800 @1`, `768x1024 @1`, `390x844 @2`): load the converted page, wait for the overlay script's
> re-fit to settle, read back every `.ocr-box`'s bounding rect mapped into natural image coordinates
> plus its `scrollHeight`/`clientHeight`, screenshot, then apply each stress case by replacing every
> plate's text node and screenshot again. A missing browser is a hard error naming the variable to set,
> never a skipped scene.

**Verification:**
- `func Capture(...) ([]evidence.Plate, map[string]string, error)` matches exactly once.
- The viewport list declares all three sizes with their device scale factors.
- A missing browser returns an error whose message contains `DOCHT_BROWSER`.

**Status:** `[x]` done

---

### Step 04.4 - Wire `ocrlab run`

**Files:** `tools/ocrlab/runner/run.go`, `tools/ocrlab/main.go`
**Depends on:** Step 04.3

**Prompt for developer:**
> Add the `run` subcommand: `-split dev|holdout|all`, `-scene <id>` (repeatable), `-out temp/ocrlab/<run-id>`.
> For each selected scene, convert the image through the real pipeline with the OCR overlay forced and
> `DOCHT_OCR_DIAG` pointed into the run directory, capture per 04.3, and write `evidence.json`. Record
> `PeakRSSBytes` from the conversion process. Refuse to start when `ocrlab verify` reports a hash
> mismatch or a missing asset for a selected scene - an incomplete corpus must fail the run, not shrink
> it.

**Verification:**
- `go run ./tools/ocrlab run -h` prints `-split`, `-scene` and `-out`.
- `run` exits non-zero with a named reason when a selected scene's media is absent.
- A completed run writes `evidence.json` parseable by `evidence.LoadRun`.

**Status:** `[x]` done

---

### Step 04.5 - Wire `ocrlab score`

**Files:** `tools/ocrlab/main.go`
**Depends on:** Step 04.4

**Prompt for developer:**
> Add the `score` subcommand taking a run directory, loading `evidence.json` plus the annotations, and
> writing `scores.json` (per-scene `metrics.SceneScore`) and `summary.json` (`metrics.Summary`, by
> category and split). It must run entirely offline from the saved run - no browser, no tesseract - so
> a scoring change can be re-applied to yesterday's evidence.

**Verification:**
- `go run ./tools/ocrlab score <dir>` produces `scores.json` and `summary.json`.
- `score` succeeds with no browser installed (assert by unsetting the browser override in a test).
- Scenes whose annotation is not `IsTruth()` appear in `summary.json` under a `skipped` list with a reason.

**Status:** `[x]` done

---

### Step 04.6 - Render the reviewable report

**Files:** `tools/ocrlab/report/report.go`, `tools/ocrlab/main.go`
**Depends on:** Step 04.5

**Prompt for developer:**
> Add the `report` subcommand producing `report.md` (per-category tables for all eight §3.2 dimensions,
> by split and edition) and `report.html` - a self-contained page with, per scene, the source and the
> rendered result side by side, the stress captures, every plate's geometry drawn as an outline over
> the image, and the named failure reason for each failed dimension. No external assets: images are
> referenced relative to the run directory so the page opens offline.

**Verification:**
- `go run ./tools/ocrlab report <dir>` writes `report.md` and `report.html`.
- `report.html` contains no `http://` or `https://` asset reference.
- Every scene section in `report.html` names at least one of: `pass`, or a failure reason string.

**Status:** `[x]` done

## Phase done criteria

- [x] Every `Step 04.*` is `[x] done`.
- [x] `go test ./tools/ocrlab/... ./internal/ocr/...` passes.
- [x] `go run ./tools/ocrlab run -split dev` completes over all 10 scenes; report.md + report.html written.
- [x] Grep for `TODO(phase-04)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Deviations from the written plan

**Capture is a DOM probe, not a DevTools session.** Step 04.3 said "read back every `.ocr-box`'s
bounding rect"; the plan did not say how. The implementation injects a probe script into a *copy* of
the converted page and reads its JSON out of `--dump-dom`, which is the pattern
`scripts/verify-html.ps1` already uses and needs no WebSocket client and no dependency. The cost is
one browser launch per state rather than one session for all of them.

> **Revised 2026-08-12: the probe stayed, the transport did not.** `--dump-dom` and `--screenshot`
> both stopped doing anything - see "The transport had to change" below. The injected probe, the
> fragment protocol and `extractProbeResult` are unchanged; only how the page is opened and read
> back moved to the DevTools protocol.

**A scene with no plates now scores instead of being skipped.** The first real run exposed the flaw:
four scenes where the recognizer found nothing produced no plates, `Score` refused them, and they
landed in the skipped list - out of every aggregate. That is precisely the result the benchmark
exists to surface, so `primaryViewport` now falls back to the run's first viewport and
`hardFailures` names the case in words: "no plates at all over N annotated group(s)".

## Environment traps found by running it

Four days of "the runner hangs" compressed into four lines, all measured, all now guarded or
commented in the code:

1. **`msedge.exe --version` never exits on Windows.** It launches a full browser instead of printing
   a version, so the version probe hung the whole run before a single page was rendered. The version
   now comes from the installer's sibling version directory; `--version` is only used off Windows and
   only under a deadline.
2. **`cmd.Stdout` as a `bytes.Buffer` makes `cmd.Wait` hang forever.** `os/exec` builds a pipe, and
   the browser's helper processes inherit the write end, so the pipe never closes after our own child
   exits. Output goes to real files now.
3. **A relative `--user-data-dir` hangs headless Edge** rather than failing. Absolute paths only;
   `TestBrowserProfileDirIsAbsolute` guards it.
4. **A relative `--screenshot` path** is resolved against the browser's working directory, so the
   file lands somewhere else and the run reports "no screenshot". Absolute paths only.

Also: headless helper processes outlive the launch and share the run's profile, so `Browser.Close`
kills exactly the processes whose command line names this run's profile directory - never by image
name, which would take the user's own browser with it.

## The transport had to change (2026-08-12)

Found while closing Phase 05, on `TestProbeCollectsPlatesAndExits`: **the browser's one-shot
switches stopped working.** Measured on Chrome 151.0.7922.76 and the matching Edge, under
`--headless=new`, `--headless=old` and plain `--headless`, with output redirected to a real file:
`--dump-dom` writes zero bytes and `--screenshot` writes no file. Neither errors - they silently do
nothing, so the symptom was "browser returned an empty DOM" after 30 ms and looked like a rendering
fault. Traps 2 and 4 above are now moot; trap 1 and 3 still hold.

The runner therefore speaks the DevTools protocol ([`cdp.go`](../../../tools/ocrlab/runner/cdp.go)),
the same protocol and for the same reason as the extension edition's `_ocrlab-cdp.mjs`. **No new
dependency:** `golang.org/x/net` was already a direct requirement and carries a WebSocket client, so
`go.mod` is untouched. What the old design cost is also recovered: one browser for the whole run
instead of one launch per page state, and viewports emulated rather than relaunched.

Two traps of its own, both now commented in the code:

5. **Chrome refuses a DevTools WebSocket that carries an `Origin` header** (its DNS-rebinding
   defence, since Chrome 111) and says only "bad status"; `x/net/websocket` refuses to dial
   *without* one. Both are satisfied by sending exactly `http://127.0.0.1` and launching with
   `--remote-allow-origins=http://127.0.0.1` - not `*`, which is what most tooling reaches for.
6. **A fragment-only navigation does not reload**, and every page state here is selected by the
   fragment, so the probe would never re-run and each state would report the previous one's
   numbers. Every open goes through `about:blank` first.

`--virtual-time-budget` is gone with the rest: it was never a statement that the page was ready,
only a guess at how long things take. The session now waits for the probe's own output, which is
exactly the thing the caller is about to read.

## First baseline (2026-08-11, 10 scenes)

Recorded here because it is what the phase produced, not as the Phase 06 report:

- 8 scenes scored, 2 skipped (the two Commons images await human annotation).
- **5 of 8 carry a hard failure.** Four - `synth-adjacent-balloons`, `synth-balloon-on-panel`,
  `synth-caption-on-gradient`, `synth-text-on-halftone` - produce **no plates at all**: comic
  balloons, a caption over a gradient and lettering on a halftone are simply not recognized, so the
  reader gets the original text untouched.
- `synth-two-columns` merges its two columns into one plate and that plate crosses the other column
  at every one of the six stress cases.
- `ocrlab score` re-ran over the saved evidence with no browser and no recognizer, which is the
  property that lets a scoring change be told apart from a product change.

## Handoff notes

`DOCHT_OCR_DIAG` is the only new surface on the shipped app, is off unless set, and is covered by an
output-identity test - this is the strategic §6 "diagnostic output without changing normal user output"
row. It adds no writable-directory requirement to the app itself (the path comes from the caller), so
the MSIX row of §6 is satisfied by construction.

## Rollback plan

Revert the phase commit(s). The only shipped-code change is `internal/ocr/diag.go` plus its call site
in `OverlayBook`; reverting removes the sidecar and leaves the overlay untouched.
