// Unit tests for the pure pieces of epub.js: the central-directory ZIP reader
// (stored + deflate) and EPUB href path resolution. The XHTML sanitize/rewrite
// path needs a DOM and is covered by the manual acceptance gates, not here.

import { test } from "node:test";
import assert from "node:assert/strict";
import { deflateRawSync } from "node:zlib";
import { Buffer } from "node:buffer";
import { readFileSync } from "node:fs";

import { unzip, resolveBookPath } from "../src/epub.js";

// makeZip builds a minimal ZIP (no CRC - the reader ignores it) with the given
// entries, exercising both stored (method 0) and deflate (method 8) paths.
function makeZip(entries) {
  const local = [];
  const central = [];
  let offset = 0;
  for (const e of entries) {
    const name = Buffer.from(e.name, "utf8");
    const raw = typeof e.data === "string" ? Buffer.from(e.data, "utf8") : Buffer.from(e.data);
    const method = e.store ? 0 : 8;
    const comp = method === 0 ? raw : deflateRawSync(raw);

    const lh = Buffer.alloc(30);
    lh.writeUInt32LE(0x04034b50, 0);
    lh.writeUInt16LE(20, 4);
    lh.writeUInt16LE(method, 8);
    lh.writeUInt32LE(comp.length, 18);
    lh.writeUInt32LE(raw.length, 22);
    lh.writeUInt16LE(name.length, 26);
    local.push(lh, name, comp);

    const ch = Buffer.alloc(46);
    ch.writeUInt32LE(0x02014b50, 0);
    ch.writeUInt16LE(20, 4);
    ch.writeUInt16LE(20, 6);
    ch.writeUInt16LE(method, 10);
    ch.writeUInt32LE(comp.length, 20);
    ch.writeUInt32LE(raw.length, 24);
    ch.writeUInt16LE(name.length, 28);
    ch.writeUInt32LE(offset, 42);
    central.push(ch, name);

    offset += lh.length + name.length + comp.length;
  }
  const cd = Buffer.concat(central);
  const eocd = Buffer.alloc(22);
  eocd.writeUInt32LE(0x06054b50, 0);
  eocd.writeUInt16LE(entries.length, 8);
  eocd.writeUInt16LE(entries.length, 10);
  eocd.writeUInt32LE(cd.length, 12);
  eocd.writeUInt32LE(offset, 16);

  const buf = Buffer.concat([...local, cd, eocd]);
  return buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength);
}

const dec = (bytes) => new TextDecoder("utf-8").decode(bytes);

test("unzip: reads stored and deflated entries, skips directories", async () => {
  const longText = "<html><body>" + "<p>Hello world</p>".repeat(200) + "</body></html>";
  const ab = makeZip([
    { name: "mimetype", data: "application/epub+zip", store: true },
    { name: "META-INF/", data: "", store: true },          // directory: skipped
    { name: "META-INF/container.xml", data: "<container/>" },
    { name: "OEBPS/ch1.xhtml", data: longText },           // deflated (compresses well)
  ]);

  const files = await unzip(ab);

  assert.equal(dec(files.get("mimetype")), "application/epub+zip");           // stored
  assert.equal(dec(files.get("META-INF/container.xml")), "<container/>");
  assert.equal(dec(files.get("OEBPS/ch1.xhtml")), longText);                  // deflate round-trip
  assert.ok(!files.has("META-INF/"), "directory entries are not stored");
  assert.equal(files.size, 3);
});

test("unzip: rejects a non-ZIP buffer", async () => {
  const notZip = new TextEncoder().encode("%PDF-1.7 not a zip at all ........").buffer;
  await assert.rejects(() => unzip(notZip), /not a ZIP archive/);
});

// The same fixture drives internal/epub TestResolveBookPathSharedCases, so the two
// editions agree on which book-supplied names resolve and where (docs/PARITY.md,
// "EPUB href resolution").
test("resolveBookPath: shared Go/JS fixture", () => {
  const fixture = JSON.parse(readFileSync(new URL("../../tests/testdata/epub_href_cases.json", import.meta.url), "utf8"));
  assert.ok(fixture.cases.length > 0);
  for (const c of fixture.cases) {
    assert.equal(resolveBookPath(c.base, c.href), c.want, `resolveBookPath(${JSON.stringify(c.base)}, ${JSON.stringify(c.href)})`);
  }
});
