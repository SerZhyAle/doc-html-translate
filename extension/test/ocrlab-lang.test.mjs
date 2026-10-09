// Tests for the per-scene OCR language of the extension lab (scripts/_ocrlab-lang.mjs).
// tools/ocrlab/evidence/lang_test.go pins the same behaviour on the Go side, and
// TestParityOCRLabLangTable keeps the two mapping tables identical.

import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  DEFAULT_LANG, LANG_PER_SCENE, declarationProblems, loadAnnotation, missingLangData,
  runLangs, sceneLang, tessCode,
} from "../scripts/_ocrlab-lang.mjs";
import { makeRun, makeScene, validateRun } from "../scripts/_ocrlab-evidence.mjs";

const human = (...langs) => ({
  origin: "human", review: { annotatedBy: "alice" }, groups: langs.map((language) => ({ language })),
});

test("Cyrillic truth reads as rus, and the truth beats the corpus field", () => {
  assert.deepEqual(sceneLang({ annotation: human("ru"), scene: { languages: ["en"] } }), { lang: "rus", source: "truth" });
});

test("several declared languages keep first-appearance order", () => {
  assert.equal(sceneLang({ annotation: human("ru", "en", "ru") }).lang, "rus+eng");
});

test("an explicit --lang wins over everything", () => {
  assert.deepEqual(sceneLang({ explicit: "eng", annotation: human("ru") }), { lang: "eng", source: "flag" });
  assert.equal(sceneLang({ explicit: "rus+eng", annotation: human("ru") }).lang, "rus+eng");
});

test("no group language falls to the corpus field, then to the default", () => {
  assert.deepEqual(sceneLang({ annotation: human(), scene: { languages: ["fr"] } }), { lang: "fra", source: "corpus" });
  assert.deepEqual(sceneLang({ scene: { languages: ["uk", "pt-BR"] } }), { lang: "ukr+por", source: "corpus" });
  assert.deepEqual(sceneLang({}), { lang: DEFAULT_LANG, source: "default" });
});

test("an OCR-seeded draft declares nothing", () => {
  const seed = { ...human("ru"), origin: "ocr-seed" };
  assert.deepEqual(sceneLang({ annotation: seed }), { lang: DEFAULT_LANG, source: "default" });
});

test("a language the catalogue cannot read reads as the default; the readable part of a mix is kept", () => {
  assert.equal(sceneLang({ scene: { languages: ["ar"] } }).lang, DEFAULT_LANG);
  assert.deepEqual(sceneLang({ scene: { languages: ["ar", "ru"] } }), { lang: "rus", source: "corpus" });
});

test("region and script subtags select the pack the app selects", () => {
  for (const [declared, pack] of Object.entries({
    EN: "eng", " ru ": "rus", "pt-BR": "por", pt_BR: "por", "zh-CN": "chi_sim", "zh-Hans": "chi_sim",
    "zh-TW": "chi_tra", "zh-Hant": "chi_tra", ja: "jpn",
  })) assert.equal(tessCode(declared), pack, declared);
  for (const declared of ["", "ar", "hi", "xx", "nl", "constructor", "__proto__"]) assert.equal(tessCode(declared), null, declared);
});

test("a missing annotation file is not an error", () => {
  const dir = mkdtempSync(join(tmpdir(), "ocrlab-lang-"));
  assert.equal(loadAnnotation(dir, "absent"), null);
  writeFileSync(join(dir, "broken.json"), "{not json");
  assert.equal(loadAnnotation(dir, "broken"), null);
  writeFileSync(join(dir, "poster.json"), JSON.stringify(human("ru")));
  assert.equal(sceneLang({ annotation: loadAnnotation(dir, "poster") }).lang, "rus");
});

test("language data that is not vendored is named, so the scene is unmeasured", () => {
  const dir = mkdtempSync(join(tmpdir(), "ocrlab-pack-"));
  writeFileSync(join(dir, "eng.traineddata"), "x");
  assert.deepEqual(missingLangData("eng", dir), []);
  assert.deepEqual(missingLangData("rus", dir), ["rus"]);
  assert.deepEqual(missingLangData("rus+eng+fra", dir), ["rus", "fra"]);
});

test("the run is labelled per-scene, naming only packs it can read", () => {
  const dir = mkdtempSync(join(tmpdir(), "ocrlab-run-"));
  writeFileSync(join(dir, "eng.traineddata"), "x");
  const scenes = [{ lang: "rus" }, { lang: "eng" }, { lang: "rus+eng" }];
  assert.deepEqual(runLangs(scenes, "", dir), { label: LANG_PER_SCENE, packs: "eng" });
  assert.deepEqual(runLangs(scenes, "rus+eng", dir), { label: "rus+eng", packs: "rus+eng" });
});

test("a language that drifted from the Go declaration is reported", () => {
  const declaration = { scenes: { a: { lang: "rus" }, b: { lang: "eng" } } };
  assert.deepEqual(declarationProblems([{ id: "a", lang: "rus" }, { id: "b", lang: "eng" }], declaration), []);
  assert.deepEqual(declarationProblems([{ id: "a", lang: "eng" }], declaration), [
    "a: this runner reads eng, the declaration says rus",
  ]);
});

test("the scene record carries the language and the unmeasured reason additively", () => {
  const plain = makeScene({ sceneId: "a", imageWidth: 1, imageHeight: 1 });
  for (const key of ["lang", "langSource", "unmeasured"]) assert.ok(!(key in plain), key);

  const skipped = makeScene({ sceneId: "a", lang: "rus", langSource: "truth", unmeasured: "language data unavailable: rus" });
  assert.equal(skipped.lang, "rus");
  assert.equal(skipped.langSource, "truth");
  assert.equal(skipped.unmeasured, "language data unavailable: rus");
  assert.ok(!("error" in skipped), "unmeasured is not a failure");
});

test("validateRun accepts an unmeasured scene without plates or size", () => {
  const run = makeRun({
    runId: "r", startedAt: "2026-10-09T00:00:00Z", edition: "extension",
    engine: { tesseract: "t", tessdataVersion: "", lang: LANG_PER_SCENE },
    browser: { name: "chrome", version: "1" },
    viewports: [{ name: "desktop", width: 1280, height: 800, deviceScaleFactor: 1 }],
    scenes: [{ sceneId: "a", lang: "rus", unmeasured: "language data unavailable: rus" }],
  });
  assert.deepEqual(validateRun(run), []);
});
