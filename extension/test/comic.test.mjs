// Unit tests for the pure pieces of comic.js: the natural page ordering, the
// page-entry filter (both shared invariants with the Go internal/comic package),
// and the lazy CBZ (ZIP) / CBT (TAR) readers. The DOM rendering path is covered by
// the manual acceptance gates, not here.

import { test } from "node:test";
import assert from "node:assert/strict";
import { deflateRawSync } from "node:zlib";
import { Buffer } from "node:buffer";

import {
  naturalCompare,
  isPageEntry,
  detectContainer,
  parseComic,
  DesktopOnlyError,
  PAGE_EXTS,
} from "../src/comic.js";

test("naturalCompare orders pages numeric-aware", () => {
  const in_ = ["page10.jpg", "page2.jpg", "page1.jpg", "page100.jpg", "cover.png", "page02.jpg"];
  const want = ["cover.png", "page1.jpg", "page2.jpg", "page02.jpg", "page10.jpg", "page100.jpg"];
  assert.deepEqual([...in_].sort(naturalCompare), want);
  assert.ok(naturalCompare("p2", "p10") < 0, "2 before 10");
  assert.ok(naturalCompare("p10", "p2") > 0);
  assert.ok(naturalCompare("7", "07") < 0, "fewer leading zeros first");
});

test("isPageEntry filters non-page entries", () => {
  for (const y of ["page01.jpg", "deep/dir/page01.JPG", "art.PNG", "scan.webp"]) {
    assert.ok(isPageEntry(y), `${y} should be a page`);
  }
  for (const n of [
    "ComicInfo.xml", "Thumbs.db", "folder/", "__MACOSX/page01.jpg",
    "sub/__MACOSX/._p.jpg", ".DS_Store", "dir/._page01.jpg", "notes.txt", "scan.tif",
  ]) {
    assert.ok(!isPageEntry(n), `${n} should not be a page`);
  }
});

test("PAGE_EXTS excludes TIFF (browser cannot display it)", () => {
  assert.ok(!PAGE_EXTS.includes("tif") && !PAGE_EXTS.includes("tiff"));
});

// makeZip builds a minimal ZIP (no CRC - the reader ignores it) with stored or
// deflated entries, matching the epub.test.mjs helper.
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
  const cdOffset = offset;
  const eocd = Buffer.alloc(22);
  eocd.writeUInt32LE(0x06054b50, 0);
  eocd.writeUInt16LE(entries.length, 8);
  eocd.writeUInt16LE(entries.length, 10);
  eocd.writeUInt32LE(cd.length, 12);
  eocd.writeUInt32LE(cdOffset, 16);
  return Buffer.concat([...local, cd, eocd]);
}

// makeTar builds a minimal USTAR archive; the reader ignores the checksum so it is
// left blank (spaces).
function makeTar(entries) {
  const blocks = [];
  for (const e of entries) {
    const raw = typeof e.data === "string" ? Buffer.from(e.data, "utf8") : Buffer.from(e.data);
    const hdr = Buffer.alloc(512);
    hdr.write(e.name, 0, "utf8");
    hdr.write(raw.length.toString(8).padStart(11, "0") + "\0", 124, "ascii"); // size, octal
    hdr.write("        ", 148, "ascii"); // checksum field blank
    hdr[156] = 0x30; // typeflag '0' = regular file
    blocks.push(hdr);
    const dataLen = Math.ceil(raw.length / 512) * 512;
    const data = Buffer.alloc(dataLen);
    raw.copy(data);
    blocks.push(data);
  }
  blocks.push(Buffer.alloc(1024)); // two zero blocks terminate the archive
  return Buffer.concat(blocks);
}

const sample = [
  { name: "page10.jpg", data: "TEN" },
  { name: "page2.jpg", data: "TWO" },
  { name: "page1.jpg", data: "ONE" },
  { name: "ComicInfo.xml", data: "<meta/>" },
  { name: "Thumbs.db", data: "junk" },
  { name: "__MACOSX/x.jpg", data: "resource" },
  { name: "notes.txt", data: "ignore" },
];

async function assertPages(buf) {
  const pages = await parseComic(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
  assert.equal(pages.length, 3, "three page images");
  assert.deepEqual(pages.map((p) => p.name), ["page1.jpg", "page2.jpg", "page10.jpg"]);
  const bytes = await pages[0].load();
  assert.equal(Buffer.from(bytes).toString("utf8"), "ONE", "first page is page1's data");
  assert.equal(pages[0].mime, "image/jpeg");
}

test("parseComic reads a CBZ in natural order, filtering non-pages", async () => {
  await assertPages(makeZip(sample));
});

test("parseComic reads a CBZ with deflated entries", async () => {
  await assertPages(makeZip(sample.map((e) => ({ ...e, store: false }))));
});

test("parseComic reads a CBT (tar) in natural order", async () => {
  await assertPages(makeTar(sample));
});

test("parseComic rejects CBR/CB7 with DesktopOnlyError", async () => {
  const rar = Buffer.from([0x52, 0x61, 0x72, 0x21, 0x1a, 0x07, 0x00]);
  await assert.rejects(() => parseComic(rar.buffer.slice(rar.byteOffset, rar.byteOffset + rar.byteLength)), DesktopOnlyError);
  const sevenz = Buffer.from([0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c, 0x00]);
  await assert.rejects(() => parseComic(sevenz.buffer.slice(sevenz.byteOffset, sevenz.byteOffset + sevenz.byteLength)), DesktopOnlyError);
});

test("parseComic throws when no page images are present", async () => {
  const buf = makeZip([{ name: "ComicInfo.xml", data: "<x/>" }, { name: "readme.txt", data: "hi" }]);
  await assert.rejects(() => parseComic(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength)));
});

test("detectContainer recognizes signatures", () => {
  assert.equal(detectContainer(new Uint8Array([0x50, 0x4b, 0x03, 0x04])), "zip");
  assert.equal(detectContainer(new Uint8Array([0x52, 0x61, 0x72, 0x21, 0x1a, 0x07])), "rar");
  assert.equal(detectContainer(new Uint8Array([0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c])), "7z");
  assert.equal(detectContainer(new Uint8Array([0x00, 0x01, 0x02, 0x03])), "tar");
});
