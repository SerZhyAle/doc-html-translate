// Unit tests for the pure reflow/toc/lang modules. Run: npm test (node --test).
// These cover the heuristic port without needing a browser or PDF.js.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

import {
  classifyBlock,
  countWords,
  isLigaturesArtifact,
  extractRows,
  reflowPage,
} from "../src/reflow.js";
import { resolveOutline, destToPageIndex, buildToc } from "../src/toc.js";
import { detectLang, normalizeLangTag, declarationContradicted } from "../src/lang.js";

const fixture = (name) => JSON.parse(readFileSync(new URL(`../../tests/testdata/${name}`, import.meta.url), "utf8"));

// transform = [fontSize,0,0,fontSize, x, y]; one item per word/line.
const item = (str, x, y, width, fontSize = 12) => ({
  str,
  transform: [fontSize, 0, 0, fontSize, x, y],
  width,
  height: fontSize,
});

test("classifyBlock: ALL-CAPS short -> h2", () => {
  assert.equal(classifyBlock("CHAPTER ONE", { centered: false, fontRatio: 1 }), "h2");
});

test("classifyBlock: centered short -> h2", () => {
  assert.equal(classifyBlock("A Quiet Place", { centered: true, fontRatio: 1 }), "h2");
});

test("classifyBlock: centered medium (9-14 words) -> h3", () => {
  assert.equal(
    classifyBlock("a slightly longer centered subtitle that keeps going well past eight words", { centered: true, fontRatio: 1 }),
    "h3",
  );
});

test("classifyBlock: big font short + notWide -> h2 via veryBig", () => {
  assert.equal(classifyBlock("Introduction", { centered: false, fontRatio: 1.8, notWide: true }), "h2");
});

test("classifyBlock: big font but full-width body stays p (notWide gate)", () => {
  assert.equal(classifyBlock("Introduction", { centered: false, fontRatio: 1.8, notWide: false }), "p");
});

test("classifyBlock: plain body -> p", () => {
  assert.equal(
    classifyBlock("This is an ordinary paragraph of body text that runs on for a while.", {}),
    "p",
  );
});

test("isLigaturesArtifact: short garbage rows flagged", () => {
  assert.equal(isLigaturesArtifact("if lf if if if if if"), true);
  assert.equal(isLigaturesArtifact("This is normal prose text"), false);
  assert.equal(isLigaturesArtifact("a b c"), false); // < 4 words
});

// The same fixture drives internal/pdf TestIsLigaturesArtifactSharedCases, so the two
// editions drop exactly the same rows (docs/PARITY.md, "PDF reflow heuristics").
test("isLigaturesArtifact: shared Go/JS fixture", () => {
  const fixture = JSON.parse(readFileSync(new URL("../../tests/testdata/ligature_artifact_cases.json", import.meta.url), "utf8"));
  assert.ok(fixture.cases.length > 0);
  for (const c of fixture.cases) {
    assert.equal(isLigaturesArtifact(c.text), c.artifact, `isLigaturesArtifact(${JSON.stringify(c.text)}): ${c.why}`);
  }
});

test("countWords: CJK runs halve, other runs count one, Hangul is not halved", () => {
  assert.equal(countWords("人类"), 1);
  assert.equal(countWords("个健康的环境中充分发挥自己的潜能。"), 9);
  assert.equal(countWords("2015年9月可持续发展议程与宣言"), 9);
  assert.equal(countWords("제1조 ① 대한민국은 민주공화국이다"), 4);
  assert.equal(countWords("We are determined to end poverty and hunger"), 8);
  assert.equal(countWords("   "), 0);
});

// The same fixture drives internal/pdf TestClassifyBlockSharedCases, so the two editions classify
// the same blocks alike (docs/PARITY.md, "PDF reflow heuristics"; ticket 73). centered is derived
// from the pdftotext geometry the desktop sees - leadingSpaces - pageMargin > 8 - which is the
// spaces proxy of this edition's geometric test.
test("classifyBlock: shared Go/JS heading fixture", () => {
  const fixture = JSON.parse(readFileSync(new URL("../../tests/testdata/pdf_heading_cases.json", import.meta.url), "utf8"));
  assert.ok(fixture.cases.length > 0);
  for (const c of fixture.cases) {
    const centered = c.leadingSpaces - c.pageMargin > 8;
    assert.equal(classifyBlock(c.text, { centered }), c.tag,
      `classifyBlock(${JSON.stringify(c.text)}): ${c.why}`);
  }
});

// Short real rows used to vanish from the page with the old average-word-length test.
test("reflowPage: short dialogue rows survive, fragment rows do not", () => {
  const row = (str, y) => ({ str, transform: [12, 0, 0, 12, 72, y], width: str.length * 6 });
  const blocks = reflowPage(
    { items: [row("Text of page 2", 700), row("fi fl fi fi fl", 686), row("Is it so? I do.", 672)] },
    { width: 612 },
  );
  const text = blocks.map((b) => b.text).join(" | ");
  assert.match(text, /Text of page 2/);
  assert.match(text, /Is it so\? I do\./);
  assert.doesNotMatch(text, /fi fl/);
});

test("extractRows: reconstructs word spacing from x-gaps", () => {
  const rows = extractRows([
    item("Hello", 72, 700, 30),
    item("World", 140, 700, 30),
  ]);
  assert.equal(rows.length, 1);
  assert.equal(rows[0].text, "Hello World");
});

test("extractRows: groups by baseline, orders top-to-bottom", () => {
  const rows = extractRows([
    item("second", 72, 680, 50),
    item("first", 72, 700, 40),
  ]);
  assert.deepEqual(rows.map((r) => r.text), ["first", "second"]);
});

test("reflowPage: heading + two paragraphs", () => {
  const viewport = { width: 612, height: 792 };
  const items = [
    // centered ALL-CAPS heading
    item("CHAPTER ONE", 246, 720, 120),
    // paragraph 1 (3 flush lines)
    item("Lorem ipsum dolor sit amet consectetur a", 72, 690, 460),
    item("continuation line of the first paragraph", 72, 674, 460),
    item("and a third line wrapping onward here", 72, 658, 460),
    // paragraph 2 (indented first line -> new block)
    item("Second paragraph opens with an indent", 90, 630, 450),
    item("and continues flush on the next line", 72, 614, 450),
  ];
  const blocks = reflowPage({ items }, viewport);
  assert.equal(blocks[0].tag, "h2");
  assert.equal(blocks[0].text, "CHAPTER ONE");
  assert.equal(blocks.length, 3);
  assert.equal(blocks[1].tag, "p");
  assert.equal(blocks[2].tag, "p");
  assert.match(blocks[1].text, /Lorem ipsum/);
  assert.match(blocks[2].text, /Second paragraph/);
});

test("reflowPage: empty input -> no blocks", () => {
  assert.deepEqual(reflowPage({ items: [] }, { width: 612 }), []);
});

test("reflowPage: indented single-line opener is NOT mis-tagged as a heading", () => {
  // A short, single-line paragraph that opens with a 30pt first-line indent. It
  // still reaches the right margin, so it must stay <p> (regression guard).
  const viewport = { width: 612, height: 792 };
  const items = [
    item("body line one runs the full width of the column here now", 72, 700, 468),
    item("body line two also full width across the text column area", 72, 684, 468),
    item("A short indented opener that reaches the right edge fully", 102, 660, 438), // indent 30, rightX 540
    item("more full width body text following along on this line ok", 72, 644, 468),
  ];
  const blocks = reflowPage({ items }, viewport);
  for (const b of blocks) assert.equal(b.tag, "p", `block mis-tagged: ${b.text}`);
});

test("reflowPage: short multi-line centered heading -> h2", () => {
  // Two centered lines, both inset symmetrically, set in a larger font.
  const viewport = { width: 612, height: 792 };
  const items = [
    // body to establish margins (left 72, right ~540)
    item("ordinary body text line filling the column width here now", 72, 700, 468),
    item("another ordinary body text line filling the column fully", 72, 684, 468),
    item("more ordinary body text to anchor the page left margin ok", 72, 668, 468),
    // centered two-line title (inset both sides), bigger font (h=18)
    item("A Symmetrically Centered", 200, 620, 212, 18),
    item("Two Line Heading Block", 210, 598, 192, 18),
  ];
  const blocks = reflowPage({ items }, viewport);
  const heading = blocks.find((b) => /Symmetrically Centered/.test(b.text));
  assert.ok(heading, "centered heading block missing");
  assert.equal(heading.tag, "h2");
});

// ---- TOC -------------------------------------------------------------------
const refA = { num: 3, gen: 0 };
const refB = { num: 9, gen: 0 };
const mockPdf = {
  async getOutline() {
    return [
      { title: "Chapter 1", dest: "d1", items: [{ title: "1.1 Intro", dest: [refB, { name: "XYZ" }], items: [] }] },
      { title: "  ", dest: null, items: [] }, // pruned: blank title, no dest, no kids
    ];
  },
  async getDestination(name) {
    return name === "d1" ? [refA, { name: "XYZ" }] : null;
  },
  async getPageIndex(ref) {
    if (ref === refA) return 0; // page 1
    if (ref === refB) return 4; // page 5
    throw new Error("unknown ref");
  },
};

test("destToPageIndex: named + explicit dests resolve; junk -> null", async () => {
  assert.equal(await destToPageIndex(mockPdf, "d1"), 0);
  assert.equal(await destToPageIndex(mockPdf, [refB]), 4);
  assert.equal(await destToPageIndex(mockPdf, null), null);
  assert.equal(await destToPageIndex(mockPdf, []), null);
});

test("resolveOutline: nested, 1-based pages, prunes empties", async () => {
  const entries = await resolveOutline(mockPdf, await mockPdf.getOutline());
  assert.equal(entries.length, 1);
  assert.equal(entries[0].title, "Chapter 1");
  assert.equal(entries[0].page, 1);
  assert.equal(entries[0].children.length, 1);
  assert.equal(entries[0].children[0].title, "1.1 Intro");
  assert.equal(entries[0].children[0].page, 5);
});

test("buildToc: empty outline -> []", async () => {
  assert.deepEqual(await buildToc({ async getOutline() { return null; } }), []);
});

// ---- Language detection ----------------------------------------------------
test("detectLang: scripts", () => {
  assert.equal(detectLang("the quick brown fox and the lazy dog in a field"), "en");
  assert.equal(detectLang("это пример русского текста для определения языка"), "ru");
  assert.equal(detectLang("це приклад українського тексту з літерами і ї є ґ"), "uk");
  assert.equal(detectLang("これは日本語のテキストのサンプルです"), "ja");
  assert.equal(detectLang("这是一段用于语言检测的中文文本示例内容"), "zh");
  assert.equal(detectLang("der die das und ist ein mit nicht auch auf eine"), "de");
});

test("detectLang: nothing to speak from states no language (ticket 76)", () => {
  assert.equal(detectLang(""), "");
  assert.equal(detectLang("   \n\t  "), "");
  // An image-only scan's "sample" is page separators, not English.
  assert.equal(detectLang("     "), "");
  // Letters of no counted block say nothing either - never an invented "en".
  assert.equal(detectLang("12345 ... ??? _"), "");
  // One pre-1918 Russian і does not make Cyrillic Ukrainian; the Ukrainian-only
  // letters ї є ґ do.
  assert.equal(detectLang("сіи сказанія были записаны въ старинныхъ книгахъ"), "ru");
  assert.equal(detectLang("у цього оповідання є їжачок і ґудзик"), "uk");
});

// The viewer's pdfDocumentLang order: options hint -> /Lang (guarded) -> heuristic. The
// desktop runs the guard half of the same cases in internal/textutil
// TestDeclarationContradicted.
test("pdfDocumentLang: declaration guard, shared Go/JS fixture", () => {
  for (const c of fixture("pdf_lang_cases.json").cases) {
    const declared = normalizeLangTag(c.declared);
    const lang = declared && !declarationContradicted(c.declared, c.text) ? declared : detectLang(c.text);
    assert.equal(lang, c.want, c.name);
  }
});

test("normalizeLangTag", () => {
  assert.equal(normalizeLangTag("en-US"), "en-US");
  assert.equal(normalizeLangTag("RU"), "ru");
  assert.equal(normalizeLangTag("fr_FR"), "fr-FR");
  assert.equal(normalizeLangTag(""), "");
  assert.equal(normalizeLangTag(null), "");
  // Same cases as internal/textutil TestNormalizeLangTag.
  assert.equal(normalizeLangTag(" ru "), "ru");
  assert.equal(normalizeLangTag("zh-Hans"), "zh");
  assert.equal(normalizeLangTag("russian"), "");
  assert.equal(normalizeLangTag("uk-UA-x"), "uk-UA");
  assert.equal(normalizeLangTag("123"), "");
});
