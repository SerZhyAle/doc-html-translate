# Phase 05 - Extension runner

**Strategic spec:** [`../16_2026-08-11_ocr-visual-fidelity-lab.md`](../16_2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 04
**Steps done:** 5 / 5

## Objective

The browser edition producing the identical evidence record, so both editions are scored by one
metrics package against one set of annotations.

## Prerequisites

- [ ] Phase 04 is ✅ Done (the evidence schema exists).
- [ ] `extension/vendor/` populated (`npm run vendor`).

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/scripts/ocrlab.mjs` | New | ≤ 480 |
| `extension/scripts/_ocrlab-evidence.mjs` | New | ≤ 200 |
| `extension/test/ocrlab-evidence.test.mjs` | New | ≤ 180 |
| `extension/package.json` | Modified | ≤ 40 |
| `tests/parity_test.go` | Modified | ≤ 900 |

## Steps

### Step 05.1 - Mirror the evidence schema in JS

**Files:** `extension/scripts/_ocrlab-evidence.mjs`
**Depends on:** - start of phase

**Prompt for developer:**
> Export `SCHEMA_VERSION`, a `makeRun({...})` factory and a `makePlate({...})` factory producing objects
> with exactly the Phase 04 field names, plus `validateRun(run)` returning a list of field-level
> problems. Field order in the emitted JSON must match the Go encoder's so the two files diff cleanly.

**Verification:**
- `export const SCHEMA_VERSION` matches exactly once and equals the Go `evidence.SchemaVersion`.
- `makePlate` returns an object whose keys are exactly the Go `Plate` JSON tags.

**Status:** `[x]` done - `SCHEMA_VERSION = 1` (once, matching `evidence.SchemaVersion`);
`Object.keys(makePlate())` = the eleven Go `Plate` tags in declaration order.

---

### Step 05.2 - Drive the loaded extension over CDP

**Files:** `extension/scripts/ocrlab.mjs`
**Depends on:** Step 05.1

**Prompt for developer:**
> Following the dependency-free CDP pattern already used by `scripts/make-screenshots.mjs` (Node's
> global `WebSocket`, no Puppeteer), launch Chrome with the extension loaded, open the viewer on each
> corpus scene as a local image input, wait for the lazy OCR to complete, and read back every
> `.ocr-plate`'s rect (mapped to natural image coordinates), text, computed background/colour,
> `scrollHeight` and `clientHeight`. Reuse the same three pinned viewports as Phase 04. Serve the
> scenes over the same local `node:http` server pattern rather than `file://`, so lazy loading and the
> IntersectionObserver behave as they do in production.

**Verification:**
- `extension/scripts/ocrlab.mjs` imports no package outside `node:` builtins and the repo's own `scripts/_*.mjs`.
- The viewport list matches Phase 04's three sizes.
- Running it writes `evidence.json` with `"edition": "extension"`.

**Status:** `[x]` done - imports are `node:` builtins plus `_lib.mjs` / `_ocrlab-cdp.mjs` /
`_ocrlab-evidence.mjs`; `VIEWPORTS` equals `runner.Viewports`; a smoke run over
`synth-two-columns` wrote `"edition": "extension"` with three plate records and a 720x360
rendered capture in the source image's own pixels.

---

### Step 05.3 - Apply the same stress cases

**Files:** `extension/scripts/ocrlab.mjs`
**Depends on:** Step 05.2

**Prompt for developer:**
> Port the Phase 04 stress constants verbatim into the runner (same six names, same literals) and apply
> each by mutating plate text nodes, letting the extension's own `MutationObserver` re-fit fire, then
> capturing. Record one plate set per (viewport, stress case) pair, exactly as the desktop runner does.

**Verification:**
- The six stress-case names appear in `ocrlab.mjs` and match `tools/ocrlab/runner/stress.go`.
- The emitted evidence contains a plate set for every (viewport, stress) pair.

**Status:** `[x]` done - all six names, texts, factors and directions compare equal to
`stress.go` field by field; the smoke run emitted 18 plate records over the 3x6 pairs and one
screenshot per case.

---

### Step 05.4 - Add the npm entry point

**Files:** `extension/package.json`
**Depends on:** Step 05.3

**Prompt for developer:**
> Add `"ocrlab": "node scripts/ocrlab.mjs"` to `scripts`. Do not add a dependency.

**Verification:**
- `npm run ocrlab -- --help` prints usage from `extension/`.
- `devDependencies` is unchanged.

**Status:** `[x]` done - `npm run ocrlab -- --help` prints the usage; the diff is the one
`scripts` line and nothing else.

---

### Step 05.5 - Guard the schema against drift

**Files:** `extension/test/ocrlab-evidence.test.mjs`, `tests/parity_test.go`
**Depends on:** Step 05.4

**Prompt for developer:**
> Add a Node test asserting `validateRun` accepts a fixture emitted by the Go runner, and a Go parity
> test `TestParityOCRLabEvidenceSchema` that parses `_ocrlab-evidence.mjs`, extracts `SCHEMA_VERSION`
> and the `makePlate` key list, and fails when either diverges from the Go `evidence` package. Also
> assert the six stress-case names match across `stress.go` and `ocrlab.mjs`.

**Verification:**
- `TestParityOCRLabEvidenceSchema` matches exactly once in `tests/parity_test.go`.
- `npm test` in `extension/` includes the new test file and passes.
- Changing `SCHEMA_VERSION` on one side alone makes `go test ./tests/...` fail.

**Status:** `[x]` done - `TestParityOCRLabEvidenceSchema` defined once and passing; `npm test`
115/115 including the 13 new cases; bumping `SCHEMA_VERSION` to 2 on the JS side alone failed the
Go test with "evidence schema version drift", and it passed again once restored.

## Phase done criteria

- [x] Every `Step 05.*` is `[x] done`.
- [x] `npm test` green in `extension/` (115/115); `go test ./tests/...` green (`-short`, and the
      full run passes when `TestConvertTestDoc` is run in its own process - see Environment traps).
- [x] Both editions' `evidence.json` for the same synthetic scene load into `ocrlab score`:
      `loop4` (desktop) and `ext-smoke2` (extension) both grade `synth-two-columns`, and both
      report the same defect.
- [x] Grep for `TODO(phase-05)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".
- [x] `./scripts/test.ps1` and `./scripts/lint.ps1` both pass, after the Phase 04 transport
      regression found here was fixed. See Environment traps.

## Deviations from the written plan

- **`extension/scripts/_ocrlab-cdp.mjs` (new, ~85 lines) is not in "Files touched".** The DevTools
  client, `evaluate`, `waitFor` and `findChrome` were written inline first and put `ocrlab.mjs` at
  483 lines before Step 05.3 had added anything - past its 480 budget. They are a self-contained
  block duplicated from `make-screenshots.mjs`, so they moved to their own `scripts/_*.mjs`, which
  is the shape Step 05.2's own verification predicate allows ("`node:` builtins and the repo's own
  `scripts/_*.mjs`"). `ocrlab.mjs` ends at 447. `make-screenshots.mjs` is untouched: refactoring it
  onto the shared client is out of this phase's scope.
- **Screenshots are produced, though no step asks for one.** The Phase 04 evidence record has a
  `screenshots` field and the objective is "the identical evidence record"; an extension run that
  left it empty would be gradable on geometry only, and every pixel measurement (residual ink,
  halo, rendered contrast) would report a zero sample. The capture is one CDP call per stress case
  with `clip.scale = naturalWidth / cssWidth`, so it lands in the source image's own pixels
  directly - the JS equivalent of `runner.CropToImage`, verified at 720x360 for a 720x360 scene.
- **`docs/PARITY.md` gains an "OCR lab evidence schema" section**, which no step names either. The
  new parity test's comment points at it, and INDEX's completion gate requires every new shared
  constant to be recorded there with a drift guard. Written now rather than left to Phase 08 so the
  test does not cite a section that does not exist.
- **`peakRssBytes` is the browser's `JSHeapUsedSize`**, read through the CDP Performance domain.
  Like the desktop runner's `peakRSS` (the Go runtime's `Sys`) it is an honest in-process proxy and
  not the operating system's RSS; the two are not comparable to each other and neither claims to be.

## Environment traps found by running it

- **`--dump-dom` and `--screenshot` produce nothing on this machine's browsers, so the *desktop*
  runner was broken.** `tools/ocrlab/runner.TestProbeCollectsPlatesAndExits` failed with "browser
  returned an empty DOM" after ~30 ms. Reproduced outside Go, with output redirected to a real file,
  on Chrome 151.0.7922.76 and Edge, under `--headless=new`, `--headless=old` and `--headless`: zero
  bytes of stdout and no screenshot file, every time. Phase 04 chose those switches deliberately
  ("it needs no dependency and no open port"), and a browser update expired that premise. Nothing in
  Phase 05 caused it and nothing in Phase 05 depends on it - the extension runner reads the DOM over
  CDP - but Phase 06 could not have produced a desktop baseline, so **it was fixed here on the
  owner's instruction**: `tools/ocrlab/runner/cdp.go` drives the browser over the DevTools protocol
  using the WebSocket client in `golang.org/x/net`, which the module already required, so `go.mod`
  is unchanged. The probe, its fragment protocol and `extractProbeResult` are untouched. Written up
  in [`PHASE_04__desktop-runner.md`](PHASE_04__desktop-runner.md) under "The transport had to
  change", since that is the phase that owns the code.
- **`go test ./tests/...` OOMs on this machine's 386 toolchain** at the 2 GB ceiling, inside
  `TestConvertTestDoc`. Not a regression and not new: `scripts/test.ps1` already pins
  `GOARCH=amd64` for exactly this reason, and the test passes in its own process (171 s).

## Handoff notes

Exact OCR text will differ between the Tesseract CLI and tesseract.js (strategic §6 accepts this). The
gate compares each edition's evidence against the same annotations - never against each other's text.

## Rollback plan

Revert the phase commit(s). The extension's shipped `src/` is untouched; only `scripts/`, `test/` and
`package.json` change.
