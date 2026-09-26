// Image encoding of the "save as HTML" export (export-html.js), driven through a real 2D canvas.

import { test } from "node:test";
import assert from "node:assert/strict";
import { createCanvas } from "@napi-rs/canvas";

import { exportImageEncoding } from "../src/export-html.js";

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
