// Unit tests for reading-position.js - the resume-reading rules (ticket 53). The module is
// pure and takes its document/window as parameters, so the geometry runs against small
// fakes instead of a DOM engine: linkedom reports every offsetTop as 0, which would make
// position math untestable.
//
// The readerKey goldens are the desktop's own values: internal/htmlgen.ReaderKey produced
// them (go test ./internal/htmlgen -run TestReaderKeyGoldenValues asserts the same hex
// literals), so one identity rule holds verbatim in both editions.

import test from "node:test";
import assert from "node:assert/strict";
import {
  fnv1a64,
  readerKey,
  currentPosition,
  resolvePositionTarget,
  mergePosition,
  removePosition,
  POSITIONS_MAX,
} from "../src/reading-position.js";

test("readerKey matches the desktop ReaderKey byte for byte", () => {
  // Goldens from `go run` over internal/htmlgen.ReaderKey, 2026-09-29. The derivation is
  // frozen: changing either side invalidates every saved position that edition holds.
  assert.equal(readerKey("a.epub", 100, "Title", 3), "b47b207ce648c694");
  assert.equal(readerKey("book.epub", 100, "War and Peace", 3), "8f6d8fb2fe4caf87");
  assert.equal(readerKey("1984.txt", 100, "1984", 3), "adf39f75ec5d86d1");
});

test("readerKey separates documents that share a title", () => {
  const a = readerKey("a.epub", 100, "Title", 3);
  assert.notEqual(a, readerKey("b.epub", 100, "Title", 3), "different file, same title");
  assert.notEqual(a, readerKey("a.epub", 101, "Title", 3), "different size");
  assert.notEqual(a, readerKey("a.epub", 100, "Title", 4), "different page count");
  assert.notEqual(a, readerKey("a.epub", 100, "", 3), "empty title");
});

test("readerKey refuses a document it cannot identify", () => {
  assert.equal(readerKey("", 100, "Title", 3), "");
  assert.equal(readerKey("a.epub", 0, "Title", 3), "");
  assert.equal(readerKey("a.epub", 100, "", 3), "");
  assert.equal(readerKey("a.epub", 100, "Title", 0), "");
});

test("fnv1a64 is the byte-level FNV-1a the Go hash/fnv writes", () => {
  // "a" hashes to the standard test vector 0xaf63dc4c8601ec8c.
  assert.equal(fnv1a64(new TextEncoder().encode("a")), "af63dc4c8601ec8c");
  assert.equal(fnv1a64(new TextEncoder().encode("")), "cbf29ce484222325");
});

// fakeDoc builds the two shapes currentPosition and resolvePositionTarget touch: the
// section list under "#content section[data-page]" and per-section [id] children.
function fakeDoc(sections) {
  const byId = new Map();
  for (const s of sections) {
    byId.set(s.id, s);
    for (const child of s.children) byId.set(child.id, child);
  }
  return {
    querySelectorAll: (sel) => (sel === "#content section[data-page]" ? sections : []),
    getElementById: (id) => byId.get(id) || null,
  };
}

function section(page, top, children) {
  const sec = {
    id: `page-${page}`,
    dataset: { page: String(page) },
    offsetTop: top,
    children,
    querySelectorAll: (sel) => (sel === "[id]" ? [sec, ...children] : []),
    contains: (n) => n === sec || children.includes(n),
  };
  return sec;
}

test("currentPosition reads the deepest anchor above the reading line", () => {
  const p1 = { id: "d0-p1", offsetTop: 100 };
  const h2 = { id: "d0-h2", offsetTop: 900 };
  const doc = fakeDoc([section(1, 0, [p1, h2]), section(2, 5000, [])]);
  // Reading line: 1000 + 900/3 = 1300 - inside section 1, past h2.
  const pos = currentPosition(doc, { scrollY: 1000, innerHeight: 900 });
  assert.deepEqual(pos, { page: 1, frag: "d0-h2", off: 400, secoff: 1300 });
});

test("currentPosition falls back to the section start and survives a viewportless window", () => {
  const doc = fakeDoc([section(1, 0, []), section(2, 400, [])]);
  const pos = currentPosition(doc, { scrollY: 0, innerHeight: undefined });
  assert.deepEqual(pos, { page: 1, frag: "", off: 0, secoff: 0 });
});

test("currentPosition is null when nothing is rendered", () => {
  assert.equal(currentPosition(fakeDoc([]), { scrollY: 0, innerHeight: 900 }), null);
});

test("resolvePositionTarget prefers the saved anchor and falls back to the section", () => {
  const h2 = { id: "d0-h2", offsetTop: 900 };
  const sec = section(1, 0, [h2]);
  const doc = fakeDoc([sec]);

  const atAnchor = resolvePositionTarget({ page: 1, frag: "d0-h2", off: 400, secoff: 1300 }, doc);
  assert.equal(atAnchor.el, h2);
  assert.equal(atAnchor.off, 400, "the anchor's own offset wins");

  const fragGone = resolvePositionTarget({ page: 1, frag: "d0-gone", off: 400, secoff: 1300 }, doc);
  assert.equal(fragGone.el, sec, "a vanished anchor falls back to the section");
  assert.equal(fragGone.off, 1300);

  assert.equal(resolvePositionTarget({ page: 9, frag: "", off: 0, secoff: 0 }, doc), null,
    "a section that is not rendered resolves to nothing");
  assert.equal(resolvePositionTarget(null, doc), null);
});

test("mergePosition keeps one entry per document and prunes the oldest past the cap", () => {
  let store = {};
  store = mergePosition(store, "a", { page: 1, frag: "", off: 10, secoff: 10 }, 100);
  store = mergePosition(store, "b", { page: 2, frag: "", off: 20, secoff: 20 }, 200);
  store = mergePosition(store, "a", { page: 1, frag: "", off: 30, secoff: 30 }, 300);
  assert.equal(store.a.at, 300, "re-saving a document updates its entry, not a second one");
  assert.deepEqual(Object.keys(store).sort(), ["a", "b"]);

  for (let i = 0; i < POSITIONS_MAX; i++) {
    store = mergePosition(store, `doc${i}`, { page: 1, frag: "", off: 0, secoff: 0 }, 1000 + i);
  }
  store = mergePosition(store, "a", { page: 1, frag: "", off: 30, secoff: 30 }, 5000);
  assert.equal(Object.keys(store).length, POSITIONS_MAX, "the map never grows past the cap");
  assert.equal(store.b, undefined, "the oldest entry fell off");
  assert.ok(store.a, "the just-saved entry stays");
});

test("removePosition drops exactly one document", () => {
  const store = mergePosition({}, "a", { page: 1, frag: "", off: 0, secoff: 0 }, 1);
  const two = mergePosition(store, "b", { page: 1, frag: "", off: 0, secoff: 0 }, 2);
  const after = removePosition(two, "a");
  assert.deepEqual(Object.keys(after), ["b"]);
  assert.deepEqual(removePosition(after, "missing"), after);
});
