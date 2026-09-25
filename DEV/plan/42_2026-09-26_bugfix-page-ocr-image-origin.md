# Page OCR recognizes any image a web page names, with the extension's access

**Status:** Draft
**Priority:** 85
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **B49 (high, plaus)** - once the reader starts page OCR, the page agent collects every `<img>` src the
  page lists, including `file:`, intranet addresses and images that never loaded (a zero natural size
  skips the size test). The extension fetches each one with its `<all_urls>` access and writes the
  recognized text into the page's own DOM, where the page's scripts can read it; the page can also start
  new scans by clicking the hidden rescan button. A hostile page could read local or intranet images it
  could never read itself. Collection is shown by a repro; whether the browser completes a `file:` fetch
  was not run.
- **B50** - `teardown()` never removes the agent's message listener, so Remove and Start again leaves two
  agents; the stale one answers first (repro: two listeners, two bars).

## 2. Goals

1. Page OCR recognizes only images the page itself loaded and could read: schemes the page can reach, a
   loaded image, and pixels taken from what the page shows rather than a privileged re-fetch where that
   is possible.
2. Only a user gesture in the extension's own UI starts a scan.
3. Teardown removes every listener and observer the agent added.

## 3. Constraints

- Permissions do not grow; `docs/security-posture.json` and the privacy texts are updated if the
  network surface changes.
- Extension-only feature: no Go twin (docs/PARITY.md intentional divergence).

## 4. Acceptance

- Tests: a `file:` or unloaded image is not collected; a page-dispatched click does not start a scan;
  Remove and Start leaves one listener.
