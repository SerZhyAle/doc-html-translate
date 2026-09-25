import { test } from "node:test";
import assert from "node:assert/strict";
import { stripRtf } from "../src/rtf.js";

// Bytes read as latin1 (one byte per char), so a test can write raw high bytes as \xNN.
const bytes = (s) => Uint8Array.from(Buffer.from(s, "latin1"));

test("stripRtf: drops control words and braces, keeps text", () => {
  const out = stripRtf(bytes("{\\rtf1\\ansi Hello world}"));
  assert.match(out, /Hello world/);
});

test("stripRtf: \\par becomes a paragraph break", () => {
  const out = stripRtf(bytes("{\\rtf1 One\\par Two}"));
  assert.match(out, /One\n\nTwo/);
});

// The same cases as internal/rtf/parse_test.go TestStripRTF: both editions must agree.
const cases = [
  ["unicode escape with fallback", "{\\rtf1\\uc1\\u1055?\\u1088?}", "Пр"],
  ["unicode escape with hex fallback (B21: no doubled letter)", "{\\rtf1\\ansicpg1251\\uc1\\u1055\\'cf\\u1088\\'f0}", "Пр"],
  ["uc2 skips two fallback bytes", "{\\rtf1{\\uc2\\u1055\\'cf\\'cf}x}", "Пx"],
  ["uc scoped to its group", "{\\rtf1{\\uc0\\u1055}\\u1088?}", "Пр"],
  ["negative value", "{\\rtf1\\u-3913?}", "\u{F0B7}"],
  ["surrogate pair", "{\\rtf1\\u-10179?\\u-8704?}", "\u{1F600}"],
  ["lone surrogate", "{\\rtf1\\u-10179?x}", "\u{FFFD}x"],
  ["font table skipped", "{\\rtf1{\\fonttbl{\\f0\\fnil Calibri;}}Body}", "Body"],
  ["colour table and info skipped", "{\\rtf1{\\colortbl;\\red0\\green0\\blue0;}{\\info{\\author Me}}Body}", "Body"],
  ["starred destination skipped", "{\\rtf1{\\*\\generator Riched20;}Body}", "Body"],
  ["picture skipped", "{\\rtf1{\\pict\\wmetafile8 0102abcdef}Body}", "Body"],
  ["bin data skipped even with braces", "{\\rtf1{\\pict\\bin4 {}\\}}Body}", "Body"],
  ["control symbols", "{\\rtf1 a\\~b\\_c\\-d\\{\\}\\\\}", "a\u{A0}b\u{2011}cd{}\\"],
  ["symbol words", "{\\rtf1\\ldblquote x\\rdblquote\\emdash}", "\u{201C}x\u{201D}\u{2014}"],
  ["paragraph and tab", "{\\rtf1 a\\par b\\tab c}", "a\n\nb\tc"],
  ["source line breaks are not text", "{\\rtf1 ab\r\ncd}", "abcd"],
  ["default code page is 1252", "{\\rtf1\\ansi caf\\'e9}", "café"],
  ["ansicpg honoured", "{\\rtf1\\ansi\\ansicpg1251 \\'cf\\'f0}", "Пр"],
  ["font charset beats ansicpg", "{\\rtf1\\ansi\\ansicpg1252{\\fonttbl{\\f0\\fcharset0 A;}{\\f1\\fcharset204 B;}}\\f1\\'cf\\f0\\'e9}", "Пé"],
  ["font change inside a group is scoped", "{\\rtf1{\\fonttbl{\\f1\\fcharset204 B;}}{\\f1\\'cf}\\'cf}", "ПÏ"],
  ["raw high byte uses the code page", "{\\rtf1\\ansicpg1252 Z\xfcrich}", "Zürich"],
  ["double-byte code page", "{\\rtf1\\ansicpg932 \\'82\\'a0}", "あ"],
  ["field instruction hidden, result kept", "{\\rtf1{\\field{\\*\\fldinst HYPERLINK \"x\"}{\\fldrslt link}}}", "link"],
  ["unknown code page falls back", "{\\rtf1\\ansicpg437 \\'e9}", "é"],
];

for (const [name, input, want] of cases) {
  test(`stripRtf: ${name}`, () => {
    assert.equal(stripRtf(bytes(input)), want);
  });
}

// A decoder used to be created per byte; a multi-megabyte Russian RTF must stay fast.
test("stripRtf: large input decodes quickly and completely", () => {
  const line = "\\'cf\\'f0\\'e8\\'e2\\'e5\\'f2, \\'ec\\'e8\\'f0! \\u1055\\'cf\\u1088\\'f0\\par\r\n";
  const n = 50000;
  const src = `{\\rtf1\\ansi\\ansicpg1251{\\fonttbl{\\f0\\fcharset204 Times;}}\\f0 ${line.repeat(n)}}`;
  const start = Date.now();
  const out = stripRtf(bytes(src));
  const elapsed = Date.now() - start;
  assert.ok(elapsed < 5000, `took ${elapsed} ms`);
  assert.equal(out.split("Привет, мир! Пр").length - 1, n);
});
