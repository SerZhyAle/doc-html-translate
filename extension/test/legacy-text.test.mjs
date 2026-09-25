// Shared legacy-text fixtures (ticket 10): the same inputs and expected texts are read by
// tests/legacy_text_test.go, so the two editions are pinned to one result.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import "./_dom.mjs";
import { decodeText, splitParagraphs } from "../src/txt.js";
import { parseRtf } from "../src/rtf.js";
import { parseFb2 } from "../src/fb2.js";

const dir = new URL("../../tests/testdata/legacy-text/", import.meta.url);
const cases = JSON.parse(readFileSync(new URL("cases.json", dir), "utf8"));

// blockText reads a rendered paragraph the way the expected files are written: the verse lines
// of a stanza are separated by a newline.
function blockText(el) {
  let s = "";
  for (const n of el.childNodes) s += n.nodeName === "BR" ? "\n" : n.textContent;
  return s;
}

// bookText joins a book's headings and paragraphs in document order. The Go edition writes a
// section title as a paragraph and the extension as a heading; the text is the same.
function bookText(book) {
  const blocks = [];
  for (const s of book.sections) {
    const div = document.createElement("div");
    div.appendChild(s.frag);
    for (const el of div.querySelectorAll("h2, h3, h4, p")) {
      const t = blockText(el);
      if (t) blocks.push(t);
    }
  }
  return blocks.join("\n\n");
}

async function extract(format, bytes) {
  switch (format) {
    case "txt": return splitParagraphs(decodeText(bytes)).join("\n\n");
    case "rtf": return bookText(await parseRtf(bytes));
    case "fb2": return bookText(await parseFb2(bytes));
    default: throw new Error(`unknown format ${format}`);
  }
}

test("legacy-text fixtures: the full set is present", () => {
  assert.ok(cases.length >= 7, `got ${cases.length} cases`);
});

for (const c of cases) {
  test(`legacy-text fixture: ${c.name}`, async () => {
    const bytes = new Uint8Array(readFileSync(new URL(c.input, dir)));
    const want = readFileSync(new URL(c.expected, dir), "utf8").replace(/\n+$/, "");
    assert.equal(await extract(c.format, bytes), want);
  });
}
