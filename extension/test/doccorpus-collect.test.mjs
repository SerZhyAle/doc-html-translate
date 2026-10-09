// Tests for the document collector's completion logic (tools/doccorpus/collect.mjs).
//
// The defect they pin (ticket 110, F4): the collector returned "settled" after three identical
// polls at the bottom of the page while the viewer still showed "Rendering page 13 / 184", and the
// judge then graded 12 of 184 pages as page loss. Every case drives collect() with a scripted
// probe and a fake clock, so no browser and no real waiting is involved.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { classify, collect, VIEWER_STATUS, OCR_COUNTER, PAGE_TOTAL, STAGES } from "../../tools/doccorpus/collect.mjs";

const here = (rel) => new URL(rel, import.meta.url);
const read = (rel) => readFileSync(here(rel), "utf8");
const catalog = JSON.parse(read("../_locales/en/messages.json"));

// msg renders a viewer string the way the en catalog would, so the table uses real wording.
function msg(key, ...args) {
  return catalog[key].message.replace(/\{(\d+)\}/g, (_, n) => String(args[Number(n) - 1]));
}

const ready = (over = {}) => ({
  state: "ready", scrollHeight: 2000, textLen: 500, images: 0, imagesLoaded: 0, imagesPending: 0,
  canvases: 0, plates: 0, overlays: 0, pageUnits: 0, pageTotal: null, status: "", ocrDone: null,
  ocrTotal: null, ocrPending: 0, extractionPending: 0, notice: "", ...over,
});

// A fake clock: sleeping advances time and nothing else.
function clock() {
  let t = 0;
  return { now: () => t, sleep: async (ms) => { t += ms; } };
}

// scripted builds a probe() that answers from the poll index.
function scripted(answer) {
  let n = 0;
  const fn = async () => answer(n++);
  fn.calls = () => n;
  return fn;
}

function run(probe, over = {}) {
  const c = clock();
  const scrolls = [];
  const out = collect({ probe, scroll: async (y) => { scrolls.push(y); }, ...c, budgetMs: 120_000, ...over });
  return out.then((r) => ({ ...r, scrolls }));
}

// ---- classify ---------------------------------------------------------------------------

test("classify names every state from the viewer's real status strings", () => {
  const table = [
    ["starting: reader not ready", { state: "pending" }, "starting"],
    ["starting: downloading, page empty", ready({ textLen: 0, status: msg("vStatusDownloading") }), "starting"],
    ["starting: stale reading status with content is not starting", ready({ status: msg("vStatusReadingEpub") }), "settled"],
    ["rendering-active: status", ready({ pageUnits: 12, pageTotal: 184, status: msg("vStatusRendering", 13, 184) }), "rendering-active"],
    ["rendering-active: fewer sections than the total, status overwritten", ready({ pageUnits: 12, pageTotal: 184, status: msg("vOcrStatus", 12, 12) }), "rendering-active"],
    ["rendering-active: pictures still decoding", ready({ images: 4, imagesLoaded: 2, imagesPending: 2 }), "rendering-active"],
    ["awaiting-scroll", ready({ pageUnits: 100, pageTotal: 184, status: msg("vStatusPagesSoFar", 100, 184) }), "awaiting-scroll"],
    ["extracting: pending page images", ready({ pageUnits: 184, pageTotal: 184, extractionPending: 30, status: msg("vStatusDonePages", 184) }), "extracting"],
    ["ocr-active: counter", ready({ pageUnits: 184, pageTotal: 184, status: msg("vOcrStatus", 55, 184), ocrDone: 55, ocrTotal: 184 }), "ocr-active"],
    ["ocr-active: pending wrapper", ready({ ocrPending: 1 }), "ocr-active"],
    ["ocr-active: standalone image", ready({ status: msg("ocrProgress") }), "ocr-active"],
    ["settled", ready({ pageUnits: 184, pageTotal: 184, status: msg("vStatusDonePages", 184), ocrDone: 184, ocrTotal: 184 }), "settled"],
    ["settled: standalone picture has a total of 1 and no section", ready({ pageTotal: 1, pageUnits: 0, status: "Done" }), "settled"],
    ["settled: desktop page, no viewer chrome", ready({ pageUnits: 40 }), "settled"],
    ["settled: no-text banner is a result, not an error", ready({ notice: "Little or no text found" }), "settled"],
    ["producer-error: notice", ready({ notice: msg("vEpubFailTitle") }), "producer-error"],
  ];
  for (const [name, probe, want] of table) assert.equal(classify(probe), want, name);
  assert.deepEqual([...new Set(table.map((r) => r[2]))].sort(), STAGES.slice().sort(), "the table covers every stage");
});

test("rendering outranks awaiting-scroll, awaiting-scroll outranks extraction and OCR", () => {
  const busy = { extractionPending: 5, ocrPending: 2 };
  assert.equal(classify(ready({ ...busy, status: msg("vStatusRendering", 3, 9) })), "rendering-active");
  assert.equal(classify(ready({ ...busy, status: msg("vStatusPagesSoFar", 3, 9) })), "awaiting-scroll");
  assert.equal(classify(ready({ ...busy })), "extracting");
});

// ---- collect ----------------------------------------------------------------------------

test("12 pages stable for many polls under 'Rendering page 13 / 184' never settle; 184 pages do", async () => {
  const STUCK_POLLS = 40;
  const probe = scripted((n) => (n < STUCK_POLLS
    ? ready({ pageUnits: 12, pageTotal: 184, scrollHeight: 2000, status: msg("vStatusRendering", 13, 184) })
    : ready({ pageUnits: 184, pageTotal: 184, scrollHeight: 2000, status: msg("vStatusDonePages", 184) })));
  const r = await run(probe);
  assert.equal(r.outcome, "settled");
  assert.equal(r.stage, "settled");
  assert.equal(r.rendered, 184);
  assert.equal(r.total, 184);
  assert.ok(r.polls > STUCK_POLLS, `settled after ${r.polls} polls, before the page finished rendering`);
});

test("a stalled run walks back to the top once, as before", async () => {
  const probe = scripted(() => ready({ pageUnits: 12, pageTotal: 184, scrollHeight: 1000, status: msg("vStatusRendering", 13, 184) }));
  const r = await run(probe, { budgetMs: 60_000 });
  assert.equal(r.outcome, "timeout");
  assert.equal(r.scrolls.filter((y) => y === 0).length, 1, "exactly one rewalk from the top");
});

test("OCR stuck at 55/184 times out in ocr-active", async () => {
  const probe = scripted(() => ready({
    pageUnits: 184, pageTotal: 184, status: msg("vOcrStatus", 55, 184), ocrDone: 55, ocrTotal: 184,
  }));
  const r = await run(probe, { budgetMs: 60_000 });
  assert.equal(r.outcome, "timeout");
  assert.equal(r.stage, "ocr-active");
  assert.equal(r.ocrDone, 55);
  assert.equal(r.ocrTotal, 184);
  assert.equal(r.rendered, 184);
});

test("a timeout while rendering reports rendering-active with the counts", async () => {
  const probe = scripted(() => ready({ pageUnits: 12, pageTotal: 184, status: msg("vStatusRendering", 13, 184), extractionPending: 12 }));
  const r = await run(probe, { budgetMs: 30_000 });
  assert.deepEqual(
    { outcome: r.outcome, stage: r.stage, rendered: r.rendered, total: r.total, extractionPending: r.extractionPending },
    { outcome: "timeout", stage: "rendering-active", rendered: 12, total: 184, extractionPending: 12 },
  );
  assert.ok(r.polls > 0);
});

test("a timeout while the next chunk is awaited reports awaiting-scroll", async () => {
  const probe = scripted(() => ready({ pageUnits: 100, pageTotal: 184, status: msg("vStatusPagesSoFar", 100, 184) }));
  const r = await run(probe, { budgetMs: 30_000 });
  assert.equal(r.outcome, "timeout");
  assert.equal(r.stage, "awaiting-scroll");
});

test("a timeout before the reader ever became ready reports starting", async () => {
  const r = await run(scripted(() => ({ state: "pending" })), { budgetMs: 10_000 });
  assert.equal(r.outcome, "timeout");
  assert.equal(r.stage, "starting");
  assert.equal(r.probe, null);
});

test("a notice heading is a producer error at once, with its text", async () => {
  const probe = scripted(() => ready({ notice: msg("vEpubFailTitle") }));
  const r = await run(probe);
  assert.equal(r.outcome, "producer-error");
  assert.equal(r.stage, "producer-error");
  assert.equal(r.error, msg("vEpubFailTitle"));
  assert.equal(r.polls, 1);
});

test("a probe that keeps throwing is a producer error, not a hang", async () => {
  const probe = scripted(() => { throw new Error("Execution context was destroyed"); });
  const r = await run(probe, { budgetMs: 300_000, probeErrorMs: 20_000 });
  assert.equal(r.outcome, "producer-error");
  assert.match(r.error, /Execution context was destroyed/);
});

test("a probe that throws only during a navigation is retried", async () => {
  const probe = scripted((n) => {
    if (n < 4) throw new Error("Execution context was destroyed");
    return ready({ pageUnits: 5, pageTotal: 5, status: msg("vStatusDonePages", 5) });
  });
  const r = await run(probe, { probeErrorMs: 20_000 });
  assert.equal(r.outcome, "settled");
});

test("a fatal probe error (dead browser) propagates", async () => {
  const probe = scripted(() => { throw Object.assign(new Error("socket closed"), { fatal: true }); });
  await assert.rejects(run(probe), /socket closed/);
});

test("a settled run needs the scroll at the bottom", async () => {
  const probe = scripted(() => ready({ scrollHeight: 10_000, pageUnits: 3, pageTotal: 3 }));
  const r = await run(probe, { stepPx: 1000 });
  assert.equal(r.outcome, "settled");
  assert.ok(Math.max(...r.scrolls) >= 10_000, "scrolled to the bottom before settling");
});

test("the final probe is returned for the caller's measurements", async () => {
  const probe = scripted(() => ready({ pageUnits: 7, pageTotal: 7, textLen: 321 }));
  const r = await run(probe);
  assert.equal(r.probe.textLen, 321);
});

// ---- drift: the literals the collector matches must still exist in the viewer ----------------

test("every status the collector matches exists in viewer.js and the en catalog", () => {
  const viewer = read("../src/viewer.js");
  for (const s of VIEWER_STATUS) {
    assert.equal(catalog[s.key]?.message, s.text, `${s.key}: the en catalog no longer says "${s.text}"`);
    assert.ok(viewer.includes(`t("${s.key}", "${s.text}"`), `${s.key}: viewer.js no longer uses "${s.text}"`);
    const sample = s.text.replace(/\{(\d+)\}/g, (_, n) => String(Number(n) + 12));
    assert.match(sample, s.re, `${s.key}: the collector's pattern no longer matches "${sample}"`);
  }
});

test("the page chrome and markers the probe reads exist in the viewer", () => {
  const viewer = read("../src/viewer.js");
  const html = read("../src/viewer.html");
  assert.ok(viewer.includes("`/ ${total}`"), "setPageTotal no longer writes '/ N'");
  assert.equal(PAGE_TOTAL.exec("/ 184")[1], "184");
  assert.equal(OCR_COUNTER.exec("OCR: 55/184 images")[2], "184");
  assert.ok(html.includes('id="page-total"'), "viewer.html lost #page-total");
  assert.ok(viewer.includes("section.dataset.pdfPage = String(pageNum)"), "deferred extraction no longer marks data-pdf-page");
  assert.ok(viewer.includes('el("div", "pdf-page-pending")'), "the reserved page box is no longer .pdf-page-pending");
  assert.ok(viewer.includes('el("div", "ocr-pending")'), "the OCR wrapper is no longer .ocr-pending");
  assert.ok(viewer.includes('"Little or no text found"') && catalog.vLittleTextTitle.message === "Little or no text found",
    "the no-text banner heading changed; collect.mjs NO_TEXT_NOTICE must follow");
});

test("run.mjs probes the same markers the classifier reads", () => {
  const run = read("../../tools/doccorpus/run.mjs");
  for (const sel of ['".ocr-pending"', "section[data-pdf-page], .pdf-page-pending", 'getElementById("page-total")']) {
    assert.ok(run.includes(sel), `run.mjs PROBE no longer reads ${sel}`);
  }
});
