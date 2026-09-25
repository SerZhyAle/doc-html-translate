# Strategic spec: 18_2026-09-24_bugfix-extension-lifecycle-leaks - Extension: release every resource, never render a stale document

**Ticket:** 18_2026-09-24_bugfix-extension-lifecycle-leaks
**Status:** Implemented (2026-09-25; heap-snapshot and live two-tab checks of §13 still to run)
**Priority:** 60
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** none - implemented directly from this spec (see §13)
**Findings:** B1-B13 B25 B27 B28 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The extension viewer and the page-OCR feature accumulate resources over a session:
- **PDF documents:** never released.
- **OCR image fitting:** its listeners and observers pile up for every OCR'd image.
- **Offscreen OCR host:** never closes once two tabs have used it.
- **Refused page-OCR host frame:** when the page refuses the host frame (`host-refused`), the `waitForHost` entry and its 10 s timer are left pending (found by ticket 25's `page-ocr.test.mjs`, which steps over it with fake timers).
- **OCR engine errors:** a failed engine start is cached forever.
- **Positioning loop:** the page agent runs a frame loop for as long as plated images exist.
- **Memory:** detached images stay pinned, decoded bitmaps are not closed, big files are held twice, and page rasterization has no size cap.

Concurrency is also unsafe. A slow earlier load can overwrite a newer document and leak its
resources, and old work corrupts the new document's OCR progress. Stop in one tab kills another
tab's job. Navigating mid-run leaves a job hanging for minutes and a service worker kept awake.
Rule syncing can apply stale options and swallow errors. Diagnostics keep stale errors, and the
"Original PDF" button opens the wrong file after another file is picked.

## 2. Goals
1. Closing or replacing a document releases everything it allocated: document handles, blob URLs, bitmaps, listeners, observers and workers.
2. The OCR engine and its host exist only while some tab is actively using them.
3. A transient engine failure is retried on the next request.
4. The page agent does no per-frame work while nothing moves, and releases images removed from the page.
5. Only the most recent load can render; superseded loads stop and clean up.
6. Stop and navigation affect only their own tab's job, and pending work settles promptly.
7. Rule updates apply in order, report failures, and skip host entries the browser cannot accept.
8. Diagnostics describe the latest run accurately, and document-specific buttons act on the current document.
9. Memory stays bounded when exporting or rasterizing large documents.

**Non-goals:**
- Content security (ticket `bugfix-extension-content-security`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** MV3 on Chrome and Edge; the service worker can be terminated at any time.
- **Performance:** no regression in the time to first page.
- **Localization:** n/a.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-extension-content-security`, `bugfix-resource-budgets` (the extension caps).
- **Platform constraints:** extension-only. The desktop edition has no equivalent, so it is declined for Go with that rationale; record in docs/PARITY.md if any shared constant changes.
- **Validation level:** a scripted open-10-documents check with a heap snapshot comparison; two-tab page-OCR stop and navigate scenarios; a stale-load race test.

## 4. Current architecture context
The viewer is a single long-lived page that swaps documents, with a generation counter checked in
only one renderer. The page-OCR broker in the service worker tracks runs per tab, talks to one
shared offscreen host that has a module-wide stop flag, and injects an agent into the page that
positions plates every animation frame.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Document scope:** each document gets a scope that registers every resource it creates and disposes all of them on teardown.
- **Load tokens:** a token is captured at load start and checked after every await; a superseded load disposes its own scope.
- **Engine lifecycle:** reference counting over active runs, and the failure path resets the cached engine.
- **Event-driven positioning:** plates reposition on scroll, resize and observed layout changes instead of every frame, with pruning of disconnected anchors.
- **Job-scoped control:** stop and cancel messages carry job and tab identity; navigation settles the job; the agent's pings are bounded.
- **Serialized rule sync:** one queue, error reporting, host validation.
- **Accurate diagnostics:** a new run resets its error, and writes are serialized.
- **Bounded export and rasterization:** asynchronous encoding, a size warning, and a canvas cap.

## 6. Open questions / research items
1. **Heap-regression check in CI**
   - **Question:** can a headless heap check run in the extension's test harness?
   - **Status:** Open - not attempted. The harness is `node --test` over linkedom, with no pdf.js worker
     and no engine, so a heap number there would not measure the leaks that matter. It needs the real
     extension under a browser (the `ocrlab` CDP harness is the nearest start).

## 7. Risks
- **Event-driven positioning misses layout shifts that only a frame loop catches.** Likelihood: medium. Impact: misaligned plates. Mitigation: a resize/mutation observer plus a slow fallback poll.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
**ADR-1: a per-document disposal scope.** Why: leaks here came from resources created in many places with no single owner.

## 10. Links to other specs
`bugfix-extension-content-security`, `bugfix-resource-budgets`.

## 11. Done criteria (strategic)
1. After opening and closing 10 PDFs in one viewer tab, memory returns near the baseline.
2. Two tabs finish page OCR, and the offscreen host closes.
3. Stop in tab A does not affect tab B's result.
4. Picking a local file during a slow URL load shows the local file, and it stays shown.

## 12. Next step
`/spec-tech 18_2026-09-24_bugfix-extension-lifecycle-leaks`

## 13. Implementation record (2026-09-25)

| Finding | Change |
|---|---|
| B1 | `teardownCurrent` destroys the pdf.js document and any loading task still in flight. |
| B2 | `buildOverlay` registers each fit; `releaseOverlays(root)` stops them (viewer teardown), and a fit stops itself once its container is placed and then detached. |
| B3 | The offscreen host stays only while another run is *running* on it; `releaseHost` clears the run's host fields. Also a failed start releases its host. |
| host-refused | the ready waiter and its timer are dropped on refusal. |
| B4 | A rejected engine start is forgotten, so the next request retries; the engine is also terminated after 60 s of an idle queue. |
| B5, B6 | The page agent places plates per change signal (scroll, resize, ResizeObserver, mutations) plus a 1 s poll while layers exist, instead of every frame; anchors whose image left the page are pruned with their layers. |
| B7 | The size-probe ImageBitmap is closed at once. |
| B8 | Export encodes images asynchronously one at a time (`toBlob` + FileReader), empties each canvas, and warns in the status line past 100 MB (`vSavedLarge`, 13 locales). Blob URLs are still held until teardown - they back OCR re-fetches of those images. |
| B9 | `rasterizePage` caps the canvas at 16 MP / 8192 px per side (`rasterScale`). |
| B10 | Load tokens: `beginLoad` tears down and returns the generation; `loadUrl`, the file picker and every loader check it after each await and release what a superseded load made. |
| B11 | A queued OCR image of a replaced document is skipped (`isCancelled`), and a finished one does not touch the new document's counters. |
| B12 | Stop names the job (`jobId`); the host cancels that job only. |
| B13 | Navigation settles the job in flight and tells the host; the agent stops pinging after 180 s without a status. SPA content swaps are covered by the pruning of B6. |
| B25 | Rule syncs are serialized, report failure (`sync-rules` answers `{ok:false,error}`), and disabled hosts DNR would reject are dropped (`ruleDomains`). |
| B27 | A run that names a format starts clean; writes are serialized. |
| B28 | "Original" acts on the URL of the document on screen (`currentUrl`), and is hidden for picked files. |

Tests: `viewer.test.mjs` (stale-load race - fails on the old code), `page-ocr.test.mjs` (two tabs close
the host, stop scoped to a job, navigation settles at once), `background.test.mjs`, `diagnostics.test.mjs`,
`lifecycle.test.mjs`, `pdf-images.test.mjs`.

Not done / open:
- Done criterion 1 (memory back near baseline after 10 PDFs) and the live two-tab page-OCR runs need a
  real browser; not run.
- The event-driven placement (spec §7 risk) needs a manual check on a page with sticky or animated
  pictures.
