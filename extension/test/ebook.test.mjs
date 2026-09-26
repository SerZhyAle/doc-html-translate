import { test } from "node:test";
import assert from "node:assert/strict";
import "./_dom.mjs";
import { fragHtml } from "./_dom.mjs";
import { isMobiBytes, retargetBookLinks, applyBookLinks } from "../src/ebook.js";
import { sanitizeToFragment } from "../src/sanitize.js";

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

// Audit finding B33: the marker was stripped only from a section that had a book link of its own,
// so in any other section a document-supplied one survived the sanitizer and became a live link.
test("retargetBookLinks: a document-supplied target marker never becomes a link", async () => {
  const html = '<html><body><p><a data-dht-target="d9-evil" href="https://example.com/">t</a></p></body></html>';
  const out = await retargetBookLinks(html, {});
  assert.ok(!out.includes("data-dht-target"), out);
  const { frag } = sanitizeToFragment(out, 0);
  applyBookLinks(frag);
  assert.ok(!fragHtml(frag).includes('href="#d9-evil"'), fragHtml(frag));
});
