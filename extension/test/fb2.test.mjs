import { test } from "node:test";
import assert from "node:assert/strict";
import { fragHtml } from "./_dom.mjs";
import { decodeFb2, parseFb2 } from "../src/fb2.js";

const latin1 = (s) => Uint8Array.from(Buffer.from(s, "latin1"));

// A Russian FB2 declared windows-1251 used to be decoded as UTF-8 and came out as U+FFFD.
test("decodeFb2: follows the encoding the XML declaration names", () => {
  // "Мир" in windows-1251 is CC E8 F0.
  const bytes = latin1('<?xml version="1.0" encoding="windows-1251"?><p>\xcc\xe8\xf0</p>');
  assert.match(decodeFb2(bytes), /<p>Мир<\/p>/);
});

test("decodeFb2: an unknown or UTF-16 label without a BOM reads as UTF-8", () => {
  const utf8 = new TextEncoder().encode('<?xml version="1.0" encoding="x-no-such"?><p>Мир</p>');
  assert.match(decodeFb2(utf8), /Мир/);
  const utf16Label = new TextEncoder().encode('<?xml version="1.0" encoding="UTF-16"?><p>Мир</p>');
  assert.match(decodeFb2(utf16Label), /Мир/);
});

test("decodeFb2: damaged UTF-8 shows a replacement character", () => {
  assert.equal(decodeFb2(latin1("<p>ab\xffcd</p>")), "<p>ab\u{FFFD}cd</p>");
});

// Verse, subtitles, epigraph authors, citations and table cells used to be dropped. The element
// set and the classes are the ones internal/fb2 writes.
test("parseFb2: keeps every prose element", async () => {
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"><body><section>
<title><p>Chapter</p></title>
<subtitle>Part</subtitle>
<epigraph><p>Quote</p><text-author>Someone</text-author></epigraph>
<poem><stanza><v>Line one</v><v>Line <emphasis>two</emphasis></v></stanza><text-author>Poet</text-author></poem>
<cite><p>Cited</p></cite>
<table><tr><td>Cell</td></tr></table>
</section></body></FictionBook>`;
  const book = await parseFb2(new TextEncoder().encode(xml));
  const html = fragHtml(book.sections[0].frag);
  for (const want of [
    '<p class="subtitle">Part</p>',
    "<p>Quote</p>",
    '<p class="text-author">Someone</p>',
    '<p class="stanza">Line one<br>Line two</p>',
    '<p class="text-author">Poet</p>',
    "<p>Cited</p>",
    "<p>Cell</p>",
  ]) {
    assert.ok(html.includes(want), `missing ${want} in ${html}`);
  }
});
