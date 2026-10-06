import { test } from "node:test";
import assert from "node:assert/strict";
import { mergeScreenBlocks } from "../src/ocr-screen.js";
import { clusterLines, trimOutlierWords } from "../src/ocr-cluster.js";

const line = (text, y) => ({ text, bbox: { x0: 100, y0: y, x1: 300, y1: y + 20 },
  lineHeight: 20, typeHeight: 18, conf: 95, tokens: 2,
  lines: [{ x0: 100, y0: y, x1: 300, y1: y + 20 }],
  lineContent: [{ text, typeHeight: 18, conf: 95, tokens: 2 }] });
const part = (start, end) => {
  const a = ["first line", "second line", "third line", "fourth line", "fifth line"];
  const ls = a.slice(start, end).map((t, i) => line(t, 100 + (start + i) * 40));
  return { ...ls[0], text: ls.map((l) => l.text).join(" "),
    bbox: { ...ls[0].bbox, y1: ls.at(-1).bbox.y1 },
    lines: ls.flatMap((l) => l.lines), lineContent: ls.flatMap((l) => l.lineContent) };
};

for (const [name, kept, expected, drop] of [
  ["missing tail", [part(0, 3)], ["first line second line third line", "fourth line fifth line"], ["first line second line third line"]],
  ["interior gap", [part(0, 1), part(2, 5)], ["first line", "second line", "third line fourth line fifth line"], ["first line", "third line fourth line fifth line"]],
  ["full duplicate", [part(0, 5)], [part(0, 5).text], [part(0, 5).text]],
  ["unread prefix", [part(2, 5)], ["first line second line", "third line fourth line fifth line"], ["third line fourth line fifth line"]],
]) {
  test(`rescue merge preserves ${name}`, () => {
    const rejected = [], got = mergeScreenBlocks(kept, [part(0, 5)], rejected);
    assert.deepEqual(got.map((b) => b.text), expected);
    assert.deepEqual(rejected.map((b) => b.text), drop);
    for (const old of kept) assert.ok(got.includes(old), "existing plates stay unchanged");
    for (const b of got) assert.equal(b.lines.length, b.lineContent.length);
  });
}

test("missing line association keeps conservative duplicate fallback", () => {
  const full = part(0, 5); delete full.lineContent;
  const dropped = [];
  assert.equal(mergeScreenBlocks([part(0, 3)], [full], dropped).length, 1);
  assert.deepEqual(dropped, [full]);
});

test("rescue additions preserve engine column order", () => {
  const left = line("left", 300), right = line("right", 100);
  right.bbox = { ...right.bbox, x0: 500, x1: 700 };
  const before = line("left prefix", 200);
  assert.deepEqual(mergeScreenBlocks([left, right], [before]).map((b) => b.text), ["left prefix", "left", "right"]);
});

test("large paragraph whitespace cannot admit a duplicated source line", () => {
  const dropped = [], got = mergeScreenBlocks([part(0, 1)], [part(0, 5)], dropped);
  assert.equal(got.length, 2); assert.equal(dropped.length, 1);
  assert.equal(got[1].text.includes("first"), false);
});

test("clustering retains transcripts alongside the source line boxes", () => {
  const blocks = clusterLines([100, 125].map((y, i) => ({ text: `line ${i}`, bbox: line("", y).bbox,
    conf: 95, tokens: 2, wordH: [18] })), 50);
  assert.equal(blocks.length, 1);
  assert.deepEqual(blocks[0].lineContent.map((c) => c.text), ["line 0", "line 1"]);
});

test("recognized word union excludes an inflated engine line header", () => {
  const words = [{ text: "Hello", bbox: { x0: 100, y0: 100, x1: 150, y1: 120 } }];
  assert.deepEqual(trimOutlierWords({ x0: 50, y0: 50, x1: 200, y1: 200 }, words), words[0].bbox);
});

test("same transcript on a stronger pass corroborates tighter geometry", () => {
  const good = line("same transcript", 100), bad = line("same transcript", 100);
  bad.bbox.x0 = 40; bad.lines[0].x0 = 40; bad.lineContent[0].conf = 80;
  const got = mergeScreenBlocks([bad], [good]);
  assert.equal(got.length, 1); assert.equal(got[0].bbox.x0, 100);
  assert.equal(got[0].text, bad.text); assert.equal(bad.lines[0].x0, 40);
  good.lineContent[0].text = "different transcript";
  assert.equal(mergeScreenBlocks([bad], [good])[0].bbox.x0, 40);
});
