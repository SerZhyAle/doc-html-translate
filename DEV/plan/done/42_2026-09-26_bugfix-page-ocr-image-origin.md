# Page OCR recognizes any image a web page names, with the extension's access

**Status:** BlockNeedUserTest - implemented 2026-09-26; left: real Chrome: cross-origin image via the public fallback, same-origin from pixels, `file:` image not recognized
**Priority:** 85
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

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

## Implementation (2026-09-26)

### B49 - collect only what the page could read; only the reader's click starts a scan

- `extension/src/page-agent.js`
  - Collection takes only `http:`, `https:`, `data:` and `blob:` pictures (`file:` is gone from the
    allowed schemes) and only a loaded picture: `complete` with a nonzero natural width and height.
  - New `pixels` message: the agent copies the picture as the page shows it onto a canvas and answers
    with a PNG `data:` URL, or `tainted` when the page's origin rules forbid reading it, or
    `unloaded` / `unreadable`.
  - Every bar button acts only on `event.isTrusted`, so `click()` or `dispatchEvent` from the page's
    scripts starts, stops or removes nothing.
  - Pictures already in the page that finish loading later are offered through the rescan button (a
    capture-phase `load` listener), so lazy-loaded pages do not lose them now that unloaded pictures
    are skipped.
- `extension/src/page-ocr.js` (broker): before each picture it asks the agent for the page's pixels
  and sends the `data:` URL to the recognizer host. The original URL is used only when the canvas was
  tainted **and** `fallbackSource` admits it: `http(s)` on a public host name. That excludes IP
  literals (the URL parser normalizes shorthand forms such as `0x7f.1`), `localhost`, single-label
  names and the `.localhost` / `.local` / `.internal` / `.lan` / `.home.arpa` / `.intranet` / `.corp`
  suffixes. Anything else counts as a failed picture and is never fetched.
- `extension/src/ocr-host.js`: a job whose source is not `http(s):` or `data:image/` is refused
  (defense in depth, so a `file:` source can never reach `fetchToBlob` from the page run).
- Tests: `extension/test/page-agent.test.mjs` - "collect skips file: pictures and pictures that never
  loaded", "a page-dispatched click on the bar's buttons starts nothing; the reader's click does",
  "pixels come from the picture as the page shows it, and a tainted canvas says so".
  `extension/test/page-ocr.test.mjs` - "a picture the page can read is recognized from its own
  pixels, not fetched again", "a tainted picture on the reader's own machine or network is never
  fetched by the extension", "a picture whose pixels the agent could not take for any other reason is
  not fetched", "fallbackSource admits only http(s) on a public host name". All four agent tests fail
  against the pre-fix `page-agent.js`. The broker fake answers an unhandled `pixels` with `tainted`,
  so the earlier tests keep their public `https://x.test` sources.
- Declined: moving the bar out of page-reachable DOM (closed shadow root). `isTrusted` already closes
  the page-started scan. A shadow root would also need `page-overlay.css` loaded into it, which means
  either a new web-accessible resource or a style copy, and both widen what this change touches.
- Residual risk: a public host name that resolves to a private address still passes
  `fallbackSource`, because the URL alone cannot tell. The fetch then happens only for a picture that
  the page did load and show.
- Permissions: unchanged (`manifest.json` untouched, no new web-accessible resource, still one
  injected script and its stylesheet). The network surface narrows. `docs/security-posture.json`
  `net-ext-images` gets a new `leaves` text and a new evidence line (`page-ocr.js`
  `export function fallbackSource`). The matching table row in `docs/SECURITY_POSTURE.md` is
  hand-mirrored. **Owner:** run `scripts/security-posture.ps1 -Render`, then the check, on Windows to
  confirm the render matches (no pwsh here). Public en/ru/uk texts are unchanged, because they stay
  true for a narrower surface.

### B50 - teardown removes every listener and observer

- `extension/src/page-agent.js`: the `runtime.onMessage` listener is kept in `state.onMessage` and
  removed in `teardown()`. So is the new document `load` listener. The existing observers, the
  scroll and resize listeners, the poll and the ping were already released.
- Test: `extension/test/page-agent.test.mjs` "Remove then Start leaves one message listener and no
  stale observers" checks for 1 listener, 0 after teardown, 1 after the second Start, 0 live
  observers and 0 document listeners.

### Evidence and open items

- `go build ./... && go vet ./... && go test ./...`: all packages ok. `cd extension && npm test`:
  279 tests pass and 0 fail, against the 271 baseline plus 8 new tests.
- `page-agent.js` is now 579 lines, over the ~500 budget. The picture rules were first extracted
  into a second injected file, then folded back: the posture, the privacy pages and the store listing
  all promise "one script and its stylesheet". A split needs those texts changed first.
- **Owner to verify in a real browser:** a cross-origin, non-CORS picture on an ordinary site still
  gets plates (canvas tainted, public fallback). A same-origin picture is read from its pixels. A
  `file:` picture on a local page gets none.
- No Go twin: this is extension-only (docs/PARITY.md intentional divergence), so PARITY is unchanged.
