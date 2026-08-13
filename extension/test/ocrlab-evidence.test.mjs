// Tests for the cross-edition evidence contract (scripts/_ocrlab-evidence.mjs).
//
// The point of the schema is that one metrics package grades both editions, so the check that
// matters is not "the JS validates its own output" - it is that the JS accepts what the Go runner
// actually wrote. GO_RUN below is a real desktop run, trimmed to one scene and two plates and
// otherwise byte-for-byte what tools/ocrlab wrote on 2026-08-11. A field the Go encoder renames
// or drops fails here. TestParityOCRLabEvidenceSchema guards the same contract from the Go side.

import { test } from "node:test";
import assert from "node:assert/strict";
import {
  SCHEMA_VERSION, EDITION_DESKTOP, EDITION_EXTENSION, MODE_FILL,
  clipped, makePlate, makeRun, makeScene, validateRun,
} from "../scripts/_ocrlab-evidence.mjs";

const GO_RUN = {
  schemaVersion: 1,
  runId: "loop4",
  startedAt: "2026-08-11T22:29:10Z",
  edition: "desktop",
  engine: { tesseract: "tesseract v5.4.0.20240606", tessdataVersion: "", lang: "eng" },
  browser: { name: "edge", version: "edge 151.0.4129.72" },
  viewports: [
    { name: "desktop", width: 1280, height: 800, deviceScaleFactor: 1 },
    { name: "tablet", width: 768, height: 1024, deviceScaleFactor: 1 },
    { name: "phone", width: 390, height: 844, deviceScaleFactor: 2 },
  ],
  scenes: [
    {
      sceneId: "synth-two-columns",
      imageWidth: 720,
      imageHeight: 360,
      plates: [
        {
          text: "Column one begins here and carries a second line and then a third.",
          rect: { x0: 49, y0: 60, x1: 578, y1: 145 },
          viewport: "desktop",
          stressCase: "none",
          fontPx: 25.7274,
          background: "rgb(250, 249, 246)",
          ink: "rgb(95, 95, 96)",
          mode: "fill",
          modeConfidence: 0,
          scrollHeight: 140,
          clientHeight: 140,
        },
        {
          text: "Yes.Yes.Yes.Yes.Yes.Yes.Yes",
          rect: { x0: 49, y0: 60, x1: 578, y1: 145 },
          viewport: "desktop",
          stressCase: "short",
          fontPx: 25.7274,
          background: "rgb(250, 249, 246)",
          ink: "rgb(95, 95, 96)",
          mode: "fill",
          modeConfidence: 0,
          scrollHeight: 140,
          clientHeight: 140,
        },
      ],
      screenshots: {
        source: "shots/synth-two-columns/source.png",
        rendered: "shots/synth-two-columns/none.png",
        stress: { none: "shots/synth-two-columns/none.png", short: "shots/synth-two-columns/short.png" },
      },
      ocrMs: 521,
      renderMs: 8037,
      peakRssBytes: 469934080,
    },
  ],
};

const clone = (v) => JSON.parse(JSON.stringify(v));

test("validateRun accepts a run the Go runner emitted", () => {
  assert.deepEqual(validateRun(GO_RUN), []);
});

test("the schema version is the one the Go evidence package declares", () => {
  assert.equal(SCHEMA_VERSION, 1);
  assert.equal(GO_RUN.schemaVersion, SCHEMA_VERSION);
});

test("makePlate emits exactly the Go Plate fields, in the Go declaration order", () => {
  assert.deepEqual(Object.keys(makePlate()), [
    "text", "rect", "viewport", "stressCase", "fontPx", "background", "ink",
    "mode", "modeConfidence", "scrollHeight", "clientHeight",
  ]);
  assert.deepEqual(Object.keys(makePlate(GO_RUN.scenes[0].plates[0])), Object.keys(GO_RUN.scenes[0].plates[0]));
});

test("makeRun and makeScene emit the Go Run and Scene fields, in order", () => {
  assert.deepEqual(Object.keys(makeRun()), [
    "schemaVersion", "runId", "startedAt", "edition", "engine", "browser", "viewports", "scenes",
  ]);
  assert.deepEqual(Object.keys(makeScene()), [
    "sceneId", "imageWidth", "imageHeight", "plates", "screenshots", "ocrMs", "renderMs", "peakRssBytes",
  ]);
  // error is omitempty on the Go side, so it appears only when there is one.
  assert.ok(!("error" in makeScene({ sceneId: "x" })));
  assert.equal(makeScene({ sceneId: "x", error: "no media" }).error, "no media");
});

test("a rebuilt run round-trips the Go run unchanged", () => {
  const rebuilt = makeRun({ ...GO_RUN, edition: EDITION_DESKTOP });
  assert.deepEqual(rebuilt, GO_RUN);
});

test("clipped mirrors the Go Plate.Clipped slack, which absorbs layout rounding", () => {
  assert.equal(clipped({ scrollHeight: 143, clientHeight: 140 }), false);
  assert.equal(clipped({ scrollHeight: 144, clientHeight: 140 }), false);
  assert.equal(clipped({ scrollHeight: 145, clientHeight: 140 }), true);
});

test("validateRun names a wrong schema version rather than passing it on", () => {
  const bad = clone(GO_RUN);
  bad.schemaVersion = 2;
  assert.match(validateRun(bad).join("\n"), /schemaVersion/);
});

test("validateRun rejects an unknown edition and an unknown mode", () => {
  const bad = clone(GO_RUN);
  bad.edition = "mobile";
  bad.scenes[0].plates[0].mode = "paint";
  const problems = validateRun(bad).join("\n");
  assert.match(problems, /edition/);
  assert.match(problems, /paint/);
});

test("validateRun rejects a plate whose viewport the run never declared", () => {
  const bad = clone(GO_RUN);
  bad.scenes[0].plates[0].viewport = "watch";
  assert.match(validateRun(bad).join("\n"), /watch is not one of the run's declared viewports/);
});

test("validateRun rejects a plate missing a field the scorer reads", () => {
  const bad = clone(GO_RUN);
  delete bad.scenes[0].plates[0].scrollHeight;
  assert.match(validateRun(bad).join("\n"), /scrollHeight: missing/);
});

test("validateRun rejects an inverted rect and a duplicate scene id", () => {
  const bad = clone(GO_RUN);
  bad.scenes[0].plates[0].rect = { x0: 578, y0: 60, x1: 49, y1: 145 };
  bad.scenes.push(clone(bad.scenes[0]));
  const problems = validateRun(bad).join("\n");
  assert.match(problems, /x1\/y1 must not be smaller/);
  assert.match(problems, /duplicate synth-two-columns/);
});

test("a failed scene carries its reason and is not graded on geometry it never had", () => {
  const run = clone(GO_RUN);
  run.edition = EDITION_EXTENSION;
  run.scenes = [{ sceneId: "broken", error: "convert: no page produced", plates: [], screenshots: {}, imageWidth: 0, imageHeight: 0, ocrMs: 0, renderMs: 0, peakRssBytes: 0 }];
  assert.deepEqual(validateRun(run), []);
});

test("the default mode is the one the Go encoder writes today", () => {
  assert.equal(makePlate().mode, MODE_FILL);
});
