// html.js against the shared Go/JS fixtures internal/htmlconv runs too: the language a page
// declares, and the text of a page in a legacy encoding (docs/PARITY.md, "HTML input language",
// "EPUB and HTML content fidelity"). The DOM comes from linkedom via ./_dom.mjs, imported first.

import "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { Buffer } from "node:buffer";

import { parseHtml } from "../src/html.js";

const fixture = (name) => JSON.parse(readFileSync(new URL(`../../tests/testdata/${name}`, import.meta.url), "utf8"));

// internal/htmlconv TestLangSharedCases runs the same pages through rootAttrs.
test("parseHtml: declared language, shared Go/JS fixture", async () => {
  for (const c of fixture("html_lang_cases.json").cases) {
    const { lang } = await parseHtml(new TextEncoder().encode(c.html));
    assert.equal(lang, c.want, c.name);
  }
});

// The decode itself is pinned by the shared charset cases (epub-parity.test.mjs); this holds
// parseHtml to using it. "Привет" in windows-1251 is cf f0 e8 e2 e5 f2.
test("parseHtml: a windows-1251 page reads as Cyrillic, not mojibake", async () => {
  const head = Buffer.from('<html><head><meta charset="windows-1251"><title>', "latin1");
  const tail = Buffer.from("</title></head><body><p>x</p></body></html>", "latin1");
  const bytes = Buffer.concat([head, Buffer.from("cff0e8e2e5f2", "hex"), tail]);
  const book = await parseHtml(new Uint8Array(bytes));
  assert.equal(book.title, "Привет");
});
