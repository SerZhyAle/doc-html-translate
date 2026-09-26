// Shared Go/JS fixtures for the EPUB reader's DOM path: the same cases run through internal/epub,
// so the two editions open the same package, resolve the same targets and keep the same chapter
// text (docs/PARITY.md, "EPUB container", "EPUB href resolution", "EPUB and HTML content
// fidelity"). The DOM comes from linkedom via ./_dom.mjs, imported first.

import "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { Buffer } from "node:buffer";

import { parseContainer, renderChapter, resolveTocAnchor, rewriteAnchor } from "../src/epub.js";
import { decodeChapter, decodeHtml } from "../src/charset.js";
import { xhtmlToHtmlSyntax } from "../src/epub-normalize.js";
import { el, fragHtml } from "./_dom.mjs";

const fixture = (name) => JSON.parse(readFileSync(new URL(`../../tests/testdata/${name}`, import.meta.url), "utf8"));

const escAttr = (v) => v.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;");

// internal/epub TestContainerSharedCases builds the same container.xml from each case.
test("parseContainer: shared Go/JS fixture", () => {
  for (const c of fixture("epub_container_cases.json").cases) {
    const rootfiles = c.rootfiles.map((rf) => `<rootfile${["full-path", "media-type"]
      .filter((k) => k in rf).map((k) => ` ${k}="${escAttr(rf[k])}"`).join("")}/>`).join("");
    const xml = `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles>${rootfiles}</rootfiles></container>`;
    if (c.want === null) assert.throws(() => parseContainer(xml), /no rootfile/, c.name);
    else assert.equal(parseContainer(xml), c.want, c.name);
  }
});

const dirOf = (p) => p.slice(0, Math.max(0, p.lastIndexOf("/")));

// The anchor a target to chapter want#frag becomes in the one-document viewer.
function anchorFor(fx, c) {
  const idx = fx.chapters.indexOf(c.want);
  return c.frag ? `d${idx}-${c.frag}` : `epub-sec-${idx}`;
}

// internal/epub TestTOCTargetSharedCases runs the same TOC cases through resolveTOCHref.
test("resolveTocAnchor: shared Go/JS target fixture", () => {
  const fx = fixture("epub_target_cases.json");
  const pathToIndex = new Map(fx.chapters.map((p, i) => [p, i]));
  for (const c of fx.toc) {
    const got = resolveTocAnchor(c.href, dirOf(fx.tocFile), pathToIndex);
    assert.equal(got, c.want === null ? null : anchorFor(fx, c), `${c.href} ${c.about || ""}`);
  }
});

// internal/epub TestLinkTargetSharedCases runs the same link cases through rewriteLinks.
test("rewriteAnchor: shared Go/JS target fixture", () => {
  const fx = fixture("epub_target_cases.json");
  const pathToIndex = new Map(fx.chapters.map((p, i) => [p, i]));
  for (const c of fx.links) {
    const a = el(`<a href="${escAttr(c.href)}">x</a>`, "a");
    rewriteAnchor(a, 0, dirOf(fx.linkFile), pathToIndex);
    const want = c.want === null ? null : `#${anchorFor(fx, c)}`;
    assert.equal(a.getAttribute("href"), want, `${c.href} ${c.about || ""}`);
  }
});

// internal/epub TestContentFidelitySharedCases and internal/htmlconv TestCharsetSharedCases run
// the same cases (audit finding B46).
test("content fidelity: shared Go/JS charset cases", () => {
  for (const c of fixture("content_fidelity_cases.json").charset) {
    const bytes = new Uint8Array(Buffer.from(c.bytes, "base64"));
    const text = c.reader === "chapter" ? decodeChapter(bytes) : decodeHtml(bytes);
    assert.ok(text.includes(c.want), `${c.name}: ${JSON.stringify(text)}`);
  }
});

test("content fidelity: shared Go/JS XHTML syntax cases", () => {
  for (const c of fixture("content_fidelity_cases.json").xhtml) {
    assert.equal(xhtmlToHtmlSyntax(c.in), c.want, c.name);
  }
});

test("content fidelity: shared Go/JS cover SVG cases", () => {
  for (const c of fixture("content_fidelity_cases.json").cover) {
    const { frag } = renderChapter(`<html><body>${c.body}</body></html>`, 0, "OEBPS", new Map(), (p) => `blob:${p}`);
    const html = fragHtml(frag);
    const isCover = !html.includes("<svg") && html.includes("<img");
    assert.equal(isCover, c.cover, `${c.name}: ${html}`);
    if (c.text) assert.ok(html.includes(c.text), `${c.name} keeps ${c.text}: ${html}`);
  }
});

test("renderChapter: an XHTML self-closing anchor does not swallow the chapter", () => {
  const xhtml = `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><body><p><a id="n1"/>First</p><p>Second</p></body></html>`;
  const { frag } = renderChapter(xhtml, 0, "OEBPS", new Map(), () => null);
  const html = fragHtml(frag);
  assert.match(html, /<a id="d0-n1"><\/a>First/);
  assert.ok(!/<a[^>]*>[^<]*First[\s\S]*Second[\s\S]*<\/a>/.test(html), html);
});
