// collect.mjs - the document collector's completion logic, free of any browser.
//
// run.mjs hands it a probe (one in-page read of the reader), a scroll and a clock; it decides when
// the reader has finished producing the document. "The DOM stopped changing" is not that moment:
// the extension renders a PDF in chunks of 100 pages built off-DOM, so the tree is frozen at the
// chunk boundary while the status bar still says "Rendering page 13 / 184". A run that stopped
// there graded 12 of 184 pages as page loss (ticket 110, F4). An unsettled run is incomplete
// evidence about the collector, never a product failure - the judge reports it that way.
//
// Nothing here talks to the page, so the logic is tested with scripted probes and a fake clock
// (extension/test/doccorpus-collect.test.mjs).

// The viewer's own status strings (extension/_locales/en/messages.json; the run pins uiLang to en).
// tests drift-check every entry against viewer.js and the catalog, so a reworded status fails a
// test instead of silently turning the collector back into a "DOM stopped changing" detector.
export const VIEWER_STATUS = [
  { key: "vStatusRendering", text: "Rendering page {1} / {2}", re: /^Rendering page \d+ \/ \d+/ },
  { key: "vStatusPagesSoFar", text: "Pages 1-{1} of {2} - keep scrolling to load more", re: /^Pages 1-\d+ of \d+ - keep scrolling/ },
  { key: "vStatusDownloading", text: "Downloading document..", re: /^Downloading document/ },
  { key: "vStatusDetecting", text: "Detecting language..", re: /^Detecting language/ },
  { key: "vStatusReadingFile", text: "Reading file..", re: /^Reading file/ },
  { key: "vStatusReadingEpub", text: "Reading EPUB..", re: /^Reading EPUB/ },
  { key: "vStatusReadingComic", text: "Reading comic..", re: /^Reading comic/ },
  { key: "vStatusUnlocking", text: "Unlocking..", re: /^Unlocking/ },
  { key: "ocrProgress", text: "Recognizing text..", re: /^Recognizing text/ },
  { key: "vOcrStatus", text: "OCR: {1}/{2} images", re: /OCR: (\d+)\/(\d+)/ },
];
const status = (key) => VIEWER_STATUS.find((s) => s.key === key).re;

const RENDERING = status("vStatusRendering");
const KEEP_SCROLLING = status("vStatusPagesSoFar");
const RECOGNIZING = status("ocrProgress");
const STARTING = ["vStatusDownloading", "vStatusDetecting", "vStatusReadingFile", "vStatusReadingEpub", "vStatusReadingComic", "vStatusUnlocking"].map(status);

// The viewer's overall OCR counter and its page-count chrome, read in-page by run.mjs's PROBE.
export const OCR_COUNTER = status("vOcrStatus");
export const PAGE_TOTAL = /\/\s*(\d+)/;

// The viewer shows this banner for a scan opened with OCR off. It sits beside a rendered document,
// so it is a result for the judge to grade, not a failure to produce one.
const NO_TEXT_NOTICE = /^Little or no text/;

export const STAGES = ["starting", "rendering-active", "awaiting-scroll", "extracting", "ocr-active", "settled", "producer-error"];

// classify names what the reader is doing right now from one probe. The order is the precedence:
// the first match wins, and only `settled` means nothing is still being produced.
export function classify(p) {
  if (!p || p.state !== "ready") return "starting";
  if (p.notice && !NO_TEXT_NOTICE.test(p.notice)) return "producer-error";

  const status = p.status || "";
  const empty = !p.textLen && !p.pageUnits && !p.images && !p.canvases;
  // A loader's status can outlive its work, so a "Reading.." line counts only while there is
  // still nothing on the page.
  if (empty && STARTING.some((re) => re.test(status))) return "starting";

  if (RENDERING.test(status)) return "rendering-active";
  if (KEEP_SCROLLING.test(status)) return "awaiting-scroll";
  // pageUnits > 0: a standalone picture reports a total of 1 yet never gets a page section, so a
  // total with no section at all is not evidence of unrendered pages.
  if (p.pageTotal > 0 && p.pageUnits > 0 && p.pageUnits < p.pageTotal) return "rendering-active";
  if (p.imagesPending > 0) return "rendering-active";

  if (p.extractionPending > 0) return "extracting";

  const counted = p.ocrTotal !== null && p.ocrTotal !== undefined && p.ocrDone < p.ocrTotal;
  if (counted || p.ocrPending > 0 || RECOGNIZING.test(status)) return "ocr-active";
  return "settled";
}

const signature = (p) => [
  p.scrollHeight, p.textLen, p.images, p.imagesLoaded, p.canvases, p.plates, p.overlays,
  p.status, p.pageUnits, p.pageTotal, p.extractionPending, p.ocrPending, p.ocrDone, p.ocrTotal,
].join("|");

const orNull = (v) => (v === undefined ? null : v);

function summary(outcome, stage, p, polls, extra) {
  return {
    outcome,
    stage,
    rendered: p ? p.pageUnits || 0 : 0,
    total: p ? orNull(p.pageTotal) : null,
    ocrDone: p ? orNull(p.ocrDone) : null,
    ocrTotal: p ? orNull(p.ocrTotal) : null,
    extractionPending: p ? p.extractionPending || 0 : 0,
    polls,
    probe: p,
    ...extra,
  };
}

// collect scrolls the page top to bottom in steps - lazy images, deferred PDF pages and the
// scroll-triggered OCR all start only when they are seen - and polls until the reader is settled.
// Settled needs three things together: the classifier says nothing is being produced, the scroll is
// at the bottom, and the page signature held for `settleAfter` polls. When the signature holds at
// the bottom while the reader is still busy, the walk goes back to the top once: an image that
// landed after the walk passed it stays unprocessed until it is seen again, as for a human reader.
//
// probe() returns the in-page probe object and may throw: a navigation in flight (the desktop's
// EPUB index stub) drops the old context. An error with `fatal` set - a dead browser - is
// rethrown; any other failure is retried, and only one that lasts `probeErrorMs` is reported as a
// producer error. All time comes from now()/sleep(), so a test runs with a fake clock.
export async function collect({
  probe, scroll, sleep, now, budgetMs,
  stepPx = 720, pollMs = 120, idleMs = 1000,
  settleAfter = 3, stallAfter = 5, maxRewalks = 1, probeErrorMs = 30_000,
}) {
  const deadline = now() + budgetMs;
  let y = 0;
  let last = "";
  let still = 0;
  let stalled = 0;
  let rewalks = 0;
  let polls = 0;
  let stage = "starting";
  let p = null; // the last probe that was ready
  let rendered = 0;
  let failingSince = null;
  let failure = null;

  while (now() < deadline) {
    if (p && y < p.scrollHeight) {
      y += stepPx;
      await scroll(y);
      await sleep(pollMs);
    } else {
      await sleep(idleMs);
    }
    polls += 1;

    let next;
    try {
      next = await probe();
      failingSince = null;
    } catch (err) {
      if (err && err.fatal) throw err;
      failure = err;
      if (failingSince === null) failingSince = now();
      if (now() - failingSince >= probeErrorMs) break;
      continue;
    }
    if (!next || next.state !== "ready") { stage = "starting"; continue; }

    p = next;
    stage = classify(p);
    if (stage === "producer-error") return summary("producer-error", stage, p, polls, { error: p.notice });

    const sig = signature(p);
    const atBottom = y >= p.scrollHeight;
    still = atBottom && stage === "settled" && sig === last ? still + 1 : 0;
    stalled = atBottom && stage !== "settled" && sig === last ? stalled + 1 : 0;
    if ((p.pageUnits || 0) > rendered) { rendered = p.pageUnits; rewalks = 0; }
    if (stalled >= stallAfter && rewalks < maxRewalks) {
      rewalks += 1;
      stalled = 0;
      y = -stepPx;
    }
    last = sig;
    if (still >= settleAfter) return summary("settled", "settled", p, polls);
  }

  if (failingSince !== null) {
    return summary("producer-error", "producer-error", p, polls, { error: failure && failure.message ? failure.message : String(failure) });
  }
  return summary("timeout", stage, p, polls);
}
