import { test } from "node:test";
import assert from "node:assert/strict";
import { isMobiBytes } from "../src/ebook.js";

const withMagic = (magic, len = 68) => {
  const b = new Uint8Array(len);
  for (let i = 0; i < magic.length && i < 8; i++) b[60 + i] = magic.charCodeAt(i);
  return b.buffer;
};

test("isMobiBytes: BOOKMOBI at offset 60 -> true (MOBI/AZW3)", () => {
  assert.equal(isMobiBytes(withMagic("BOOKMOBI")), true);
});

test("isMobiBytes: other PDB type/creator -> false", () => {
  assert.equal(isMobiBytes(withMagic("BOOKTEXt")), false);
});

test("isMobiBytes: buffer too short -> false", () => {
  assert.equal(isMobiBytes(new Uint8Array(10).buffer), false);
});
