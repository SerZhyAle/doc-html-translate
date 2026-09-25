// The input limits (limits.js) as the EPUB and comic readers apply them. The bomb fixtures are
// built here, not committed: zeros deflate about a thousandfold, so a 4.4 GB archive costs a few
// megabytes. The EPUB bomb has the same shape as internal/epub TestExtractRefusesTotalBomb, so
// both editions are shown refusing the same file (docs/PARITY.md, "Input limits").

import { test } from "node:test";
import assert from "node:assert/strict";
import { deflateRawSync } from "node:zlib";
import { Buffer } from "node:buffer";

import { unzip } from "../src/epub.js";
import { parseComic } from "../src/comic.js";
import {
  ARCHIVE_MAX_ENTRIES,
  ARCHIVE_MAX_TOTAL_BYTES,
  EPUB_MAX_ENTRY_BYTES,
  COMIC_MAX_PAGE_BYTES,
  InputLimitError,
  formatBytes,
  inflateRawCapped,
} from "../src/limits.js";

const MB = 1024 * 1024;

// makeZip writes entries given either as data (deflated here) or as a precompressed stream
// with the size its header should declare.
function makeZip(entries) {
  const local = [];
  const central = [];
  let offset = 0;
  for (const e of entries) {
    const name = Buffer.from(e.name, "utf8");
    const comp = e.comp ?? deflateRawSync(Buffer.from(e.data ?? "", "utf8"));
    const size = e.size ?? Buffer.byteLength(e.data ?? "", "utf8");

    const lh = Buffer.alloc(30);
    lh.writeUInt32LE(0x04034b50, 0);
    lh.writeUInt16LE(20, 4);
    lh.writeUInt16LE(8, 8);
    lh.writeUInt32LE(comp.length, 18);
    lh.writeUInt32LE(size, 22);
    lh.writeUInt16LE(name.length, 26);
    local.push(lh, name, comp);

    const ch = Buffer.alloc(46);
    ch.writeUInt32LE(0x02014b50, 0);
    ch.writeUInt16LE(20, 4);
    ch.writeUInt16LE(20, 6);
    ch.writeUInt16LE(8, 10);
    ch.writeUInt32LE(comp.length, 20);
    ch.writeUInt32LE(size, 24);
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

const zeros90 = deflateRawSync(Buffer.alloc(90 * MB));

const epubBase = [
  { name: "mimetype", data: "application/epub+zip" },
  { name: "META-INF/container.xml", data: "<container/>" },
  { name: "OEBPS/ch1.xhtml", data: "<html><body><p>Hello</p></body></html>" },
];

test("limits match the desktop's published numbers", () => {
  assert.equal(ARCHIVE_MAX_ENTRIES, 20000);
  assert.equal(formatBytes(ARCHIVE_MAX_TOTAL_BYTES), "4 GB");
  assert.equal(formatBytes(EPUB_MAX_ENTRY_BYTES), "100 MB");
  assert.equal(formatBytes(COMIC_MAX_PAGE_BYTES), "200 MB");
});

// Done criterion 4: the bomb EPUB the desktop refuses is refused here too, from the listing.
test("unzip refuses an EPUB whose listing unpacks past the total limit", async () => {
  const bomb = [...epubBase];
  for (let i = 0; i < 50; i++) bomb.push({ name: `OEBPS/fill${i}.bin`, comp: zeros90, size: 90 * MB });
  await assert.rejects(() => unzip(makeZip(bomb)), (err) => {
    assert.ok(err instanceof InputLimitError, String(err));
    assert.equal(err.key, "vLimitTotal");
    assert.match(err.message, /4 GB/);
    return true;
  });
});

test("unzip refuses an EPUB with more entries than the limit", async () => {
  const many = [...epubBase];
  for (let i = 0; i < ARCHIVE_MAX_ENTRIES; i++) many.push({ name: `x/${i}`, comp: Buffer.alloc(0), size: 0 });
  await assert.rejects(() => unzip(makeZip(many)), (err) => err instanceof InputLimitError && err.key === "vLimitEntries");
});

test("unzip skips by name an entry over the per-file limit, keeping the rest", async () => {
  const big = deflateRawSync(Buffer.alloc(EPUB_MAX_ENTRY_BYTES + 1));
  const files = await unzip(makeZip([...epubBase, { name: "OEBPS/huge.bin", comp: big, size: EPUB_MAX_ENTRY_BYTES + 1 }]));
  assert.ok(!files.has("OEBPS/huge.bin"));
  assert.ok(files.has("OEBPS/ch1.xhtml"));
});

// A header that understates the size is caught while inflating: the entry is dropped, not
// kept cut short and not inflated whole.
test("unzip drops an entry that inflates past its declared size", async () => {
  const files = await unzip(makeZip([...epubBase, { name: "OEBPS/liar.bin", comp: zeros90, size: 10 }]));
  assert.ok(!files.has("OEBPS/liar.bin"));
  assert.ok(files.has("OEBPS/ch1.xhtml"));
});

test("inflateRawCapped stops at the cap", async () => {
  await assert.rejects(() => inflateRawCapped(zeros90, "z.bin", 1 * MB), (err) => err instanceof InputLimitError && err.key === "vLimitEntry");
  const ok = await inflateRawCapped(deflateRawSync(Buffer.from("abc")), "a", 3);
  assert.equal(Buffer.from(ok).toString(), "abc");
});

test("parseComic refuses a CBZ bomb from its listing", async () => {
  const pages = [];
  for (let i = 0; i < 50; i++) pages.push({ name: `page${i}.jpg`, comp: zeros90, size: 90 * MB });
  await assert.rejects(() => parseComic(makeZip(pages)), (err) => err instanceof InputLimitError && err.key === "vLimitTotal");
});

test("parseComic skips an oversize page and refuses a page that inflates past its listing", async () => {
  const big = deflateRawSync(Buffer.alloc(COMIC_MAX_PAGE_BYTES + 1));
  const pages = await parseComic(makeZip([
    { name: "page1.jpg", comp: big, size: COMIC_MAX_PAGE_BYTES + 1 },
    { name: "page2.jpg", data: "TWO" },
    { name: "page3.jpg", comp: zeros90, size: 3 },
  ]));
  assert.deepEqual(pages.map((p) => p.name), ["page2.jpg", "page3.jpg"]);
  assert.equal(Buffer.from(await pages[0].load()).toString(), "TWO");
  await assert.rejects(() => pages[1].load(), (err) => err instanceof InputLimitError);
});
