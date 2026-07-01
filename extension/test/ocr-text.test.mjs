// Tests for the OCR-noise filter (isTranslatable). Mirrors internal/ocr/text.go's
// TestIsTranslatable - the two must agree (see docs/PARITY.md).
import { test } from "node:test";
import assert from "node:assert/strict";
import { isTranslatable } from "../src/ocr-text.js";

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
