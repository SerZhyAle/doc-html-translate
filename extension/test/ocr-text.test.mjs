// Tests for the OCR-noise filter (isTranslatable). Mirrors internal/ocr/text.go's
// TestIsTranslatable - the two must agree (see docs/PARITY.md).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { isTranslatable, joinLineWords, joinPlateLines, repairPipeMisreads, OCR_HANGUL_JOIN_GAP_RATIO } from "../src/ocr-text.js";

test("keeps real translatable text", () => {
  for (const s of [
    "Get your food, drink & duty free delivered before the trolley.",
    "AUTO MIETEN. BIS ZU 25% SPAREN.",
    "CHAPTER ONE",
    "Section 12.3 - Results and Summary",
    "Привет мир, это тест",
    "日本語", // short CJK phrase
  ]) {
    assert.equal(isTranslatable(s), true, `should keep: ${s}`);
  }
});

test("drops OCR noise with nothing to translate", () => {
  for (const s of [
    "",
    "   ",
    "XS",                    // < 5 letters
    "Se of",                 // < 5 letters
    "25",                    // digits only
    "25%",
    "1234 5678",
    "————— ——— ~~ :",        // symbols only
    "BCDFG",                 // no vowels
    "https://example.com/x", // address
    "www.sixt.de",
    "user@example.com",
    "sixt.de",
    "C:\\Users\\serzh",      // path
    "gr [u : &o Se A JETZT MIETEN", // mishmash
  ]) {
    assert.equal(isTranslatable(s), false, `should drop: ${JSON.stringify(s)}`);
  }
});

// internal/ocr TestIsTranslatableSharedCases runs the same cases (audit finding B38).
test("isTranslatable: shared Go/JS fixture", () => {
  const fx = JSON.parse(readFileSync(new URL("../../tests/testdata/ocr_translatable_cases.json", import.meta.url), "utf8"));
  for (const c of fx.cases) {
    assert.equal(isTranslatable(c.text), c.want, `${JSON.stringify(c.text)} ${c.about || ""}`);
  }
});

// internal/ocr TestRepairPipeMisreadsSharedCases runs the same cases (OCR-PIPELINE amendment 1.5).
test("repairPipeMisreads: shared Go/JS fixture", () => {
  const fx = JSON.parse(readFileSync(new URL("../../tests/testdata/ocr_pipe_repair_cases.json", import.meta.url), "utf8"));
  for (const c of fx.cases) {
    assert.equal(repairPipeMisreads(c.line), c.want, `${JSON.stringify(c.line)} ${c.about || ""}`);
  }
});

// internal/ocr TestCJKJoinSharedCases runs the same cases (OCR-PIPELINE amendment 1.8).
test("joinLineWords / joinPlateLines: shared Go/JS fixture", () => {
  const fx = JSON.parse(readFileSync(new URL("../../tests/testdata/ocr_cjk_join_cases.json", import.meta.url), "utf8"));
  assert.ok(fx.words.length && fx.lines.length, "the fixture carries no cases");
  for (const c of fx.words) {
    const words = c.words.map(([text, x0, y0, x1, y1]) => ({ text, bbox: { x0, y0, x1, y1 } }));
    assert.equal(joinLineWords(words), c.want, c.about);
  }
  for (const c of fx.lines) {
    assert.equal(joinPlateLines(c.lines), c.want, c.about);
  }
});

test("joinLineWords reads the boxes through the recognizer scale, and a boxless word keeps its space", () => {
  // The 2x-upscaled Korean pair: 2 px apart at the recognizer's scale is 1 px on the page, still
  // a syllable gap; a word with no box has no gap to measure.
  const up = [{ text: "시", bbox: { x0: 0, y0: 0, x1: 48, y1: 60 } }, { text: "장", bbox: { x0: 52, y0: 0, x1: 100, y1: 60 } }];
  assert.equal(joinLineWords(up, 2), "시장");
  assert.equal(joinLineWords([{ text: "시", bbox: null }, { text: "장", bbox: null }]), "시 장");
  assert.equal(joinLineWords([{ text: "市", bbox: null }, { text: "長", bbox: null }]), "市長", "Han needs no box");
  assert.equal(OCR_HANGUL_JOIN_GAP_RATIO, 0.33, "the geometric middle of the 0.30-0.36 plateau");
});
