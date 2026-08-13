// Tests for the line-to-plate grouping (ocr-cluster.js). Mirrors internal/ocr/cluster_test.go -
// the two implementations are hand-ported, so they are given the same input and must reach the
// same plates (see docs/PARITY.md).

import { test } from "node:test";
import assert from "node:assert/strict";
import { clusterLines, medianLinePitch } from "../src/ocr-cluster.js";

// The line boxes below are not invented: they are what tesseract returned for the lab's two
// grouping scenes on 2026-08-11, in the upscaled space the desktop app clusters in. The extension
// divides by the upscale factor before clustering instead, so it sees these halved - both spaces
// are asserted below, because the decision must not depend on which one it is made in.
const balloonOnPanel = [
  { bbox: { x0: 644, y0: 184, x1: 930, y1: 218 }, conf: 96.0, text: "WELL, THAT IS" },
  { bbox: { x0: 645, y0: 256, x1: 890, y1: 285 }, conf: 96.7, text: "ONE WAY TO" },
  { bbox: { x0: 646, y0: 328, x1: 838, y1: 357 }, conf: 93.6, text: "SOLVE IT!" },
  { bbox: { x0: 156, y0: 126, x1: 1034, y1: 826 }, conf: 0.0, text: "" }, // the panel: no text
];

const adjacentBalloons = [
  { bbox: { x0: 116, y0: 123, x1: 390, y1: 151 }, conf: 96.4, text: "ARE YOU SURE" },
  { bbox: { x0: 116, y0: 175, x1: 362, y1: 203 }, conf: 95.7, text: "ABOUT THIS?" },
  { bbox: { x0: 118, y0: 259, x1: 298, y1: 287 }, conf: 96.4, text: "NOT EVEN" },
  { bbox: { x0: 117, y0: 311, x1: 308, y1: 339 }, conf: 95.9, text: "SLIGHTLY." },
];

const halved = (lines) => lines.map((l) => ({
  ...l,
  bbox: {
    x0: Math.round(l.bbox.x0 / 2), y0: Math.round(l.bbox.y0 / 2),
    x1: Math.round(l.bbox.x1 / 2), y1: Math.round(l.bbox.y1 / 2),
  },
}));

// The splitting half of the pair: all-caps lettering boxes far shorter than the line it came from
// (29-34 px ink for 72 px leading here), so grouping it by ink height tore one balloon into three
// plates - three sentence fragments for a translator that has no sentence to work with.
test("one balloon is one plate", () => {
  for (const lines of [balloonOnPanel, halved(balloonOnPanel)]) {
    const blocks = clusterLines(lines, 80);
    assert.equal(blocks.length, 1);
    assert.equal(blocks[0].text, "WELL, THAT IS ONE WAY TO SOLVE IT!");
  }
});

// The merging half: the fix above must not be paid for with a plate that spans two balloons. The
// pitch inside a balloon is 52 px here and the step across to the next one is 84 px.
test("adjacent balloons stay two plates", () => {
  for (const lines of [adjacentBalloons, halved(adjacentBalloons)]) {
    const blocks = clusterLines(lines, 80);
    assert.equal(blocks.length, 2);
    assert.equal(blocks[0].text, "ARE YOU SURE ABOUT THIS?");
    assert.equal(blocks[1].text, "NOT EVEN SLIGHTLY.");
  }
});

test("a section break is not a line pitch", () => {
  const headingAndDistantBody = [
    { bbox: { x0: 20, y0: 10, x1: 120, y1: 30 }, conf: 90, text: "Heading" },
    { bbox: { x0: 10, y0: 200, x1: 190, y1: 220 }, conf: 88, text: "This body" },
  ];
  assert.equal(medianLinePitch(headingAndDistantBody, 20), 0);
  assert.equal(clusterLines(headingAndDistantBody).length, 2);

  const body = [
    { bbox: { x0: 10, y0: 10, x1: 190, y1: 30 }, conf: 95, text: "Alpha beta" },
    { bbox: { x0: 10, y0: 34, x1: 190, y1: 54 }, conf: 95, text: "gamma delta" },
    { bbox: { x0: 10, y0: 58, x1: 160, y1: 78 }, conf: 95, text: "epsilon" },
  ];
  assert.equal(medianLinePitch(body, 20), 24);
  assert.equal(clusterLines(body).length, 1);
});

test("lines that share no column measure no pitch off each other", () => {
  const columns = [
    { bbox: { x0: 10, y0: 10, x1: 100, y1: 30 }, conf: 95, text: "left one" },
    { bbox: { x0: 300, y0: 20, x1: 390, y1: 40 }, conf: 95, text: "right one" },
  ];
  assert.equal(medianLinePitch(columns, 20), 0);
});
