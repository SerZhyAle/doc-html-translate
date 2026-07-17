import { test } from "node:test";
import assert from "node:assert/strict";
import { splitParagraphs, decodeText } from "../src/txt.js";

// Bytes as Notepad's "Unicode" / "Unicode big endian" write them: a BOM, then 2-byte units.
function utf16Bytes(s, littleEndian) {
  const buf = new ArrayBuffer((s.length + 1) * 2);
  const view = new DataView(buf);
  view.setUint16(0, 0xfeff, littleEndian);
  for (let i = 0; i < s.length; i++) view.setUint16((i + 1) * 2, s.charCodeAt(i), littleEndian);
  return buf;
}

// Decoding every text file as UTF-8 turned a UTF-16 save into mojibake - the same defect the
// Go side had, measured on both before the fix.
test("decodeText: UTF-16LE Cyrillic, the case that was mojibake", () => {
  const want = "Это обычное предложение на русском языке.";
  assert.equal(decodeText(utf16Bytes(want, true)), want);
});

test("decodeText: UTF-16BE", () => {
  const want = "The Project Gutenberg eBook of something.";
  assert.equal(decodeText(utf16Bytes(want, false)), want);
});

test("decodeText: a UTF-8 BOM does not reach the text", () => {
  const bytes = new Uint8Array([0xef, 0xbb, 0xbf, ...new TextEncoder().encode("Первый абзац.")]);
  assert.equal(decodeText(bytes.buffer), "Первый абзац.");
});

test("decodeText: plain UTF-8 is untouched", () => {
  const want = "Обычный UTF-8 без BOM.";
  assert.equal(decodeText(new TextEncoder().encode(want).buffer), want);
});

// A pre-Unicode Cyrillic code page must be detected and decoded, not read as UTF-8. cp1251 and
// koi8-r remap the same bytes, so the frequency-weighted fit has to pick the right one.
test("decodeText: Windows-1251 Cyrillic is detected", () => {
  const want = "Лицензионное соглашение на использование программы.";
  // Encode to cp1251 by hand: each Cyrillic letter is one byte in the 0xC0.. range.
  const map = new Map();
  const cyr = "абвгдежзийклмнопрстуфхцчшщъыьэюя";
  for (let i = 0; i < cyr.length; i++) map.set(cyr[i], 0xe0 + i);
  const CYR = "АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ";
  for (let i = 0; i < CYR.length; i++) map.set(CYR[i], 0xc0 + i);
  const bytes = Uint8Array.from([...want].map((c) => (map.has(c) ? map.get(c) : c.charCodeAt(0))));
  assert.equal(decodeText(bytes.buffer), want);
});

// A non-Russian legacy file (French Latin-1) is not valid UTF-8 either, so it reaches the same
// path - but no Cyrillic candidate is right, so it must not be Cyrillized. The sentence is long
// enough to carry real signal (fraction ~0.17, well below the floor); a very short accent-dense
// string can exceed the floor, which is the "short files don't carry enough signal" limit the
// ticket calls out and the shared floor accepts.
test("decodeText: non-Russian legacy bytes are not forced into Cyrillic", () => {
  // Every character here is Latin-1 representable, so its code point IS its Latin-1 byte; the
  // accented bytes make it invalid UTF-8, so it reaches the legacy path.
  const src = "Élément très cher, à côté de l'hôtel où nous étions cet été, très reconnaissant.";
  const bytes = Uint8Array.from([...src].map((c) => c.charCodeAt(0) & 0xff));
  const out = decodeText(bytes.buffer);
  assert.ok(!/[а-яА-Я]/.test(out), `should not invent Cyrillic, got: ${out}`);
});

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
