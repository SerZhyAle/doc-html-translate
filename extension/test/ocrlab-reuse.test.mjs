// Tests for the extension producer's side of scene reuse (scripts/_ocrlab-reuse.mjs).
//
// The rule itself (what counts as an identical, complete earlier scene) lives in Go and is tested
// there; what is pinned here is the contract with it: the arguments `ocrlab reuse` receives, what a
// hit and every kind of miss look like to the producer, and that a broken lookup can only ever
// cost the saving, never turn into a wrong record. The helper is stubbed, so no Go and no browser.

import { test } from "node:test";
import assert from "node:assert/strict";
import { reuseArgv, reuseScene } from "../scripts/_ocrlab-reuse.mjs";
import { makeReusedFrom, makeScene, validateRun, makeRun } from "../scripts/_ocrlab-evidence.mjs";

const scene = { id: "synth-uniform-paper", lang: "eng", langSource: "default" };
const engine = { tesseract: "tesseract.js 7.0.0", tessdataVersion: "eng=sha256:aa", lang: "per-scene" };
const browser = { name: "chrome", version: "Chrome/151.0.1" };
const base = { repoRoot: "/repo", outDir: "/repo/temp/ocrlab/ext-1", scene, engine, browser };

const REUSED_FROM = {
  bundle: "temp/ocrlab/ext-0", declarationSha256: "d".repeat(64), producerDigest: "p".repeat(64),
  collectedAt: "2026-10-09T10:00:00Z", reusedAt: "2026-10-09T11:00:00Z", integrity: "sizes",
};
const record = (over = {}) => ({
  sceneId: scene.id, imageWidth: 20, imageHeight: 10, plates: null, screenshots: { source: "shots/x/source.png", rendered: "shots/x/desktop-none.png" },
  ocrMs: 4000, renderMs: 900, peakRssBytes: 0, lang: "eng", langSource: "flag", reusedFrom: REUSED_FROM, ...over,
});

// stub returns a spawnSync-shaped result and remembers how it was called.
function stub(result) {
  const calls = [];
  const run = (cmd, argv, opts) => { calls.push({ cmd, argv, opts }); return result; };
  run.calls = calls;
  return run;
}

test("the lookup asks the Go lab for exactly this scene, edition and engine", () => {
  const run = stub({ status: 0, stdout: "null\n", stderr: "no reuse for x: producer digest differs\n" });
  reuseScene({ ...base, reuseFrom: "/repo/temp/ocrlab/ext-0", run });
  const [{ cmd, argv, opts }] = run.calls;
  assert.equal(cmd, "go");
  assert.equal(opts.cwd, "/repo");
  assert.deepEqual(argv, [
    "run", "./tools/ocrlab", "reuse",
    "-run", "/repo/temp/ocrlab/ext-1", "-edition", "extension", "-scene", "synth-uniform-paper",
    "-tesseract", "tesseract.js 7.0.0", "-tessdata", "eng=sha256:aa",
    "-browser-name", "chrome", "-browser-version", "Chrome/151.0.1",
    "-reuse-from", "/repo/temp/ocrlab/ext-0",
  ]);
  assert.ok(!reuseArgv({ outDir: "o", sceneId: "s", engine, browser }).includes("-reuse-from"));
});

test("a hit returns the Go record with provenance and this run's own language choice", () => {
  const run = stub({ status: 0, stdout: `${JSON.stringify(record())}\n`, stderr: "reused synth-uniform-paper from temp/ocrlab/ext-0\n" });
  const hit = reuseScene({ ...base, run });
  assert.equal(hit.bundle, "temp/ocrlab/ext-0");
  assert.equal(hit.scene.reusedFrom.producerDigest, "p".repeat(64));
  assert.equal(hit.scene.langSource, "default", "the earlier record's langSource must not leak in");
  assert.equal(hit.scene.ocrMs, 4000);
});

test("a miss carries the Go reason without its prefix", () => {
  const run = stub({ status: 0, stdout: "null\n", stderr: "no reuse for synth-uniform-paper: temp/ocrlab/ext-0: language data differs\n" });
  assert.deepEqual(reuseScene({ ...base, run }), { miss: "temp/ocrlab/ext-0: language data differs" });
});

test("every way the lookup can break is a miss, never a record", () => {
  const broken = {
    "non-zero exit": { status: 1, stdout: "", stderr: "ocrlab reuse: the new run is not declared\n" },
    "spawn error": { status: null, error: new Error("spawn go ENOENT"), stdout: "", stderr: "" },
    "not JSON": { status: 0, stdout: "{oops\n", stderr: "" },
    "another scene": { status: 0, stdout: `${JSON.stringify(record({ sceneId: "other" }))}\n`, stderr: "" },
    "no provenance": { status: 0, stdout: `${JSON.stringify(record({ reusedFrom: undefined }))}\n`, stderr: "" },
  };
  for (const [name, result] of Object.entries(broken)) {
    const out = reuseScene({ ...base, run: stub(result) });
    assert.ok(out.miss && !out.scene, `${name} must be a miss`);
  }
});

test("--fresh never calls the helper", () => {
  const run = stub({ status: 0, stdout: `${JSON.stringify(record())}\n`, stderr: "" });
  assert.deepEqual(reuseScene({ ...base, fresh: true, run }), { miss: "--fresh" });
  assert.equal(run.calls.length, 0);
});

test("makeReusedFrom emits exactly the Go ReusedFrom fields, in order", () => {
  assert.deepEqual(Object.keys(makeReusedFrom()), [
    "bundle", "declarationSha256", "producerDigest", "collectedAt", "reusedAt", "integrity",
  ]);
  assert.deepEqual(makeReusedFrom(REUSED_FROM), REUSED_FROM);
});

test("makeScene keeps provenance on a reused scene and writes none on a collected one", () => {
  assert.deepEqual(makeScene(record()).reusedFrom, REUSED_FROM);
  assert.ok(!("reusedFrom" in makeScene({ sceneId: "x" })), "reusedFrom is omitempty on the Go side");
});

test("a run holding a reused scene is still valid evidence", () => {
  const viewports = [{ name: "desktop", width: 1280, height: 800, deviceScaleFactor: 1 }];
  const run = makeRun({
    runId: "ext-1", startedAt: "2026-10-09T11:00:00Z", edition: "extension",
    engine: engine, browser, viewports, scenes: [record({ plates: [] })],
  });
  assert.deepEqual(validateRun(run), []);
  assert.deepEqual(run.scenes[0].reusedFrom, REUSED_FROM);
});
