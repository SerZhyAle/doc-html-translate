import { test } from "node:test";
import assert from "node:assert/strict";
import { stripRtf } from "../src/rtf.js";

// Bytes read as latin1 (one byte per char), matching the reader's decode path.
const bytes = (s) => Uint8Array.from(Buffer.from(s, "latin1"));

test("stripRtf: drops control words and braces, keeps text", () => {
  const out = stripRtf(bytes("{\\rtf1\\ansi Hello world}"));
  assert.match(out, /Hello world/);
});

test("stripRtf: \\par becomes a paragraph break", () => {
  const out = stripRtf(bytes("{\\rtf1 One\\par Two}"));
  assert.match(out, /One\n\nTwo/);
});

test("stripRtf: \\'XX hex escapes decode as Windows-1251 (Cyrillic)", () => {
  // \'e0 -> U+0430 (а), \'e1 -> U+0431 (б) in Windows-1251.
  const out = stripRtf(bytes("{\\rtf1 \\'e0\\'e1}"));
  assert.match(out, /аб/);
});
