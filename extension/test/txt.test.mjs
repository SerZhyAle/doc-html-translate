import { test } from "node:test";
import assert from "node:assert/strict";
import { splitParagraphs } from "../src/txt.js";

test("splitParagraphs: blank lines join consecutive lines into one paragraph", () => {
  const out = splitParagraphs("Line one\nstill one\n\nPara two\n");
  assert.deepEqual(out, ["Line one still one", "Para two"]);
});

test("splitParagraphs: one paragraph per line when there are no blank lines", () => {
  const out = splitParagraphs("First\nSecond\nThird");
  assert.deepEqual(out, ["First", "Second", "Third"]);
});

test("splitParagraphs: normalizes CRLF and CR", () => {
  assert.deepEqual(splitParagraphs("A\r\n\r\nB"), ["A", "B"]);
  assert.deepEqual(splitParagraphs("A\r\rB"), ["A", "B"]);
});
