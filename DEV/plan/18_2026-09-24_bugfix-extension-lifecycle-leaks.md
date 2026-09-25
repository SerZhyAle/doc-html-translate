# Strategic spec: 18_2026-09-24_bugfix-extension-lifecycle-leaks - Extension: release every resource, never render a stale document

**Ticket:** 18_2026-09-24_bugfix-extension-lifecycle-leaks
**Status:** Draft
**Priority:** 60
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/18_2026-09-24_bugfix-extension-lifecycle-leaks/` (created by /spec-tech)
**Findings:** B1-B13 B25 B27 B28 (see the [findings register](../research/audit_2026-09-24/README.md))

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
   - **Status:** Open.

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
