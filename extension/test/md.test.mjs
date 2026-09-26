// md.js against the shared Go/JS section fixture internal/md runs too (docs/PARITY.md, "Markdown
// sections"). The DOM comes from linkedom via ./_dom.mjs.

import "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

import { parseMarkdown } from "../src/md.js";

// linkedom's DOMParser does not wrap a bare fragment in <html><body> the way a browser's HTML
// parser does (see _dom.mjs), and md.js hands it marked's fragment output.
const LinkedomParser = globalThis.DOMParser;
globalThis.DOMParser = class extends LinkedomParser {
  parseFromString(s, type) {
    if (type === "text/html" && !/<html[\s>]/i.test(s)) s = `<!doctype html><html><body>${s}</body></html>`;
    return super.parseFromString(s, type);
  }
};

// internal/md TestSectionsSharedCases splits goldmark's rendering of the same Markdown.
test("parseMarkdown: sections, shared Go/JS fixture", async () => {
  const fx = JSON.parse(readFileSync(new URL("../../tests/testdata/md_section_cases.json", import.meta.url), "utf8"));
  for (const c of fx.cases) {
    const book = await parseMarkdown(new TextEncoder().encode(c.md));
    assert.deepEqual(book.sections.map((s) => s.label), c.sections, c.name);
  }
});
