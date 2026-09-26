import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { splitParagraphs, decodeText, sniffUtf16, measureUtf8, acceptAsUtf8 } from "../src/txt.js";

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
  // The Western fallback reads it as windows-1252, so the accents come out right.
  assert.equal(out, src);
});

function utf16NoBom(s, littleEndian) {
  const buf = new ArrayBuffer(s.length * 2);
  const view = new DataView(buf);
  for (let i = 0; i < s.length; i++) view.setUint16(i * 2, s.charCodeAt(i), littleEndian);
  return buf;
}

// BOM-less UTF-16 is valid UTF-8 for ASCII and Cyrillic, so it used to come through with NULs.
test("decodeText: BOM-less UTF-16 in both byte orders", () => {
  for (const want of ["Plain English line.\nSecond line.", "Привет, мир. Это текст без метки порядка байтов."]) {
    assert.equal(decodeText(utf16NoBom(want, true)), want);
    assert.equal(decodeText(utf16NoBom(want, false)), want);
  }
});

test("sniffUtf16: text without NULs, NULs on both parities and tiny input are not UTF-16", () => {
  assert.equal(sniffUtf16(new TextEncoder().encode("ordinary text")), null);
  assert.equal(sniffUtf16(new Uint8Array(8)), null);
  assert.equal(sniffUtf16(new Uint8Array([0x61, 0])), null);
});

// One damaged byte in a Russian UTF-8 book used to send the whole file to the legacy detector.
test("decodeText: a truncated UTF-8 tail costs one replacement character", () => {
  const full = new TextEncoder().encode("Обычный русский текст. ".repeat(20) + "конец");
  assert.equal(decodeText(full.slice(0, -1)), "Обычный русский текст. ".repeat(20) + "коне\u{FFFD}");
});

test("decodeText: damage mid-text stays UTF-8 below the threshold", () => {
  const text = "Обычный русский текст в кодировке UTF-8. ".repeat(20);
  const b = new TextEncoder().encode(text);
  b[b.indexOf(0x20, b.length >> 1)] = 0xff;
  const out = decodeText(b);
  assert.equal(out.split("\u{FFFD}").length - 1, 1);
  assert.ok(out.startsWith("Обычный"));
});

// The same WHATWG counts as internal/textutil TestMeasureUTF8.
test("measureUtf8 / acceptAsUtf8: threshold and truncated tail", () => {
  const enc = (s) => Uint8Array.from(Buffer.from(s, "latin1"));
  const tail = measureUtf8(new Uint8Array([...new TextEncoder().encode("Привет"), 0xd1]));
  assert.deepEqual(tail, { multi: 6, invalidBytes: 1, errors: 1, truncatedTail: true });
  assert.equal(acceptAsUtf8(new Uint8Array([...new TextEncoder().encode("ПриветПрив"), 0xff, 0xff])), false);
  assert.equal(acceptAsUtf8(enc("abc\xd0")), true);
  assert.equal(acceptAsUtf8(enc("caf\xe9 cr\xe8me")), false);
});

test("splitParagraphs: blank lines join consecutive lines into one paragraph", () => {
  const out = splitParagraphs("Line one\nstill one\n\nPara two\n");
  assert.deepEqual(out, ["Line one still one", "Para two"]);
});

test("splitParagraphs: one paragraph per line when there are no blank lines", () => {
  const out = splitParagraphs("First\nSecond\nThird");
  assert.deepEqual(out, ["First", "Second", "Third"]);
});

test("splitParagraphs: shared Go/JS fixture", () => {
  // internal/txt TestParagraphsSharedCases runs the same cases through parseParagraphs.
  const fixture = JSON.parse(readFileSync(new URL("../../tests/testdata/txt_paragraph_cases.json", import.meta.url), "utf8"));
  for (const c of fixture.cases) {
    assert.deepEqual(splitParagraphs(c.in), c.want, c.name);
  }
});

test("splitParagraphs: normalizes CRLF and CR", () => {
  assert.deepEqual(splitParagraphs("A\r\n\r\nB"), ["A", "B"]);
  assert.deepEqual(splitParagraphs("A\r\rB"), ["A", "B"]);
});
