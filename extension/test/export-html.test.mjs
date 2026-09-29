// Image encoding of the "save as HTML" export (export-html.js), driven through a real 2D canvas.
// The completeness planner runs next to it: the same pure module, no canvas needed.

import { test } from "node:test";
import assert from "node:assert/strict";
import { createCanvas } from "@napi-rs/canvas";

import {
  exportImageEncoding,
  exportPlan,
  estimateExportBytes,
  EXPORT_PREPARE_MAX_PAGES,
  EXPORT_PREPARE_MAX_FILE_BYTES,
} from "../src/export-html.js";

// B40: every image was re-encoded as JPEG, so a transparent PNG exported as a black box.
test("the export keeps a transparent image lossless and alpha-capable, an opaque one compact", () => {
  // A line-art diagram: one stroke on a transparent background, the stroke far below the first
  // band of rows the scan reads at once, so a band-by-band scan has to reach it.
  const w = 40, h = 600;
  const diagram = createCanvas(w, h);
  const dctx = diagram.getContext("2d");
  dctx.fillStyle = "#000";
  dctx.fillRect(0, 0, w, h);
  dctx.clearRect(10, 590, 5, 5);
  assert.equal(exportImageEncoding(dctx, w, h).type, "image/png");

  const blank = createCanvas(w, h).getContext("2d"); // nothing drawn: fully transparent
  assert.equal(exportImageEncoding(blank, w, h).type, "image/png");

  const photo = createCanvas(w, h);
  const pctx = photo.getContext("2d");
  pctx.fillStyle = "#6a8";
  pctx.fillRect(0, 0, w, h);
  const enc = exportImageEncoding(pctx, w, h);
  assert.equal(enc.type, "image/jpeg");
  assert.ok(enc.quality > 0 && enc.quality <= 1);
});

// Ticket 55: the export must say what it holds before it writes anything, offer a bounded
// "prepare all pages" where that stays safe, and give a concrete reason where it does not.
test("a fully rendered or unpaged document exports through the direct path", () => {
  assert.equal(exportPlan({ total: 0, rendered: 0 }).action, "direct"); // no page dimension
  assert.equal(exportPlan({}).action, "direct");
  assert.equal(exportPlan({ total: 300, rendered: 300, imageBytes: 900 }).action, "direct");
  assert.equal(exportPlan({ total: 3, rendered: 4 }).action, "direct"); // overshoot stays direct
});

test("a partial export within the budget offers to prepare the remaining pages", () => {
  const plan = exportPlan({ total: 300, rendered: 120, imageBytes: 0 });
  assert.equal(plan.action, "offer-prepare");
  assert.equal(plan.remaining, 180);
  // Known image bytes below the file budget do not change the offer.
  assert.equal(exportPlan({ total: 2, rendered: 1, imageBytes: 1024 }).action, "offer-prepare");
  // The last page short of the page bound is still preparable.
  const edge = exportPlan({ total: EXPORT_PREPARE_MAX_PAGES + 5, rendered: 5, imageBytes: 0 });
  assert.equal(edge.action, "offer-prepare");
});

test("a document too large to prepare is refused with a concrete reason, before any file", () => {
  const pages = exportPlan({ total: EXPORT_PREPARE_MAX_PAGES + 6, rendered: 5, imageBytes: 0 });
  assert.equal(pages.action, "partial-only");
  assert.equal(pages.reason.key, "vExportReasonPages");
  assert.deepEqual(pages.reason.args, [EXPORT_PREPARE_MAX_PAGES + 1, EXPORT_PREPARE_MAX_PAGES]);

  const bytes = exportPlan({ total: 300, rendered: 1, imageBytes: EXPORT_PREPARE_MAX_FILE_BYTES });
  assert.equal(bytes.action, "partial-only");
  assert.equal(bytes.reason.key, "vExportReasonBytes");
  assert.match(String(bytes.reason.args[0]), /GB|MB$/);

  // Exactly at the file budget is still preparable; unknown bytes (0) never refuse on their own.
  assert.equal(exportPlan({ total: 300, rendered: 1, imageBytes: Math.floor(EXPORT_PREPARE_MAX_FILE_BYTES / 1.37) }).action, "offer-prepare");
  assert.equal(exportPlan({ total: 300, rendered: 1, imageBytes: 0 }).action, "offer-prepare");
});

test("estimateExportBytes prices the base64 growth the data: URIs cost", () => {
  assert.equal(estimateExportBytes(0), 0);
  assert.equal(estimateExportBytes(1024 * 1024), Math.ceil(1024 * 1024 * 1.37));
  assert.ok(estimateExportBytes(1000) > 1000);
});
