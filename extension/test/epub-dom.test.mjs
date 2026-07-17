// Unit tests for the DOM-driven half of epub.js: TOC parsing (parseNavToc /
// parseNcxToc) and the chapter rewriters (rewriteImg / convertSvgImage /
// rewriteAnchor / renderChapter). These paths need a DOM and were previously only
// covered by manual acceptance gates (see the note atop the pure-piece test in
// epub.test.mjs). Closes the epub side of the cross-edition parity ticket's item
// #22 (DEV/plan/2026-07-01_cross-edition-parity.md).
//
// The DOM is provided by linkedom via ./_dom.mjs, which must be imported first so
// globalThis.document / DOMParser exist before epub.js's functions run.

import "./_dom.mjs";
import { fragHtml, fragText, el } from "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";

import {
  parseNavToc,
  parseNcxToc,
  rewriteImg,
  convertSvgImage,
  rewriteAnchor,
  renderChapter,
} from "../src/epub.js";

// A chapter document is full XHTML; wrap fixtures so linkedom's parser (which does
// not synthesize <html>/<body>) sees a proper document.
const doc = (body) => `<!doctype html><html><body>${body}</body></html>`;

// ---- parseNavToc (EPUB3 nav.xhtml) -----------------------------------------

test("parseNavToc: prefers <nav epub:type=toc> and reads nested lists", () => {
  const html = doc(`
    <nav epub:type="landmarks"><ol><li><a href="cover.xhtml">Cover</a></li></ol></nav>
    <nav epub:type="toc">
      <ol>
        <li><a href="ch1.xhtml">One</a>
          <ol><li><a href="ch1.xhtml#s1">One.1</a></li></ol>
        </li>
        <li><a href="ch2.xhtml">Two</a></li>
      </ol>
    </nav>`);
  const toc = parseNavToc(html);
  assert.equal(toc.length, 2);
  assert.deepEqual(
    toc.map((e) => e.title),
    ["One", "Two"],
  );
  assert.equal(toc[0].href, "ch1.xhtml");
  assert.equal(toc[0].children.length, 1);
  assert.deepEqual(toc[0].children[0], { title: "One.1", href: "ch1.xhtml#s1", children: [] });
});

test("parseNavToc: falls back to role=doc-toc, then the first nav", () => {
  const roleDoc = doc(`
    <nav><ol><li><a href="x.xhtml">First</a></li></ol></nav>
    <nav role="doc-toc"><ol><li><a href="y.xhtml">Real</a></li></ol></nav>`);
  assert.equal(parseNavToc(roleDoc)[0].title, "Real");

  const firstDoc = doc(`<nav><ol><li><a href="only.xhtml">Only</a></li></ol></nav>`);
  assert.equal(parseNavToc(firstDoc)[0].title, "Only");
});

test("parseNavToc: collapses whitespace in titles and keeps list-less span entries", () => {
  const html = doc(`
    <nav epub:type="toc"><ol>
      <li><a href="a.xhtml">  Spaced\n   Title  </a></li>
      <li><span>Part heading</span><ol><li><a href="b.xhtml">Leaf</a></li></ol></li>
    </ol></nav>`);
  const toc = parseNavToc(html);
  assert.equal(toc[0].title, "Spaced Title");
  assert.equal(toc[1].title, "Part heading");
  assert.equal(toc[1].href, "");
  assert.equal(toc[1].children[0].title, "Leaf");
});

test("parseNavToc: returns [] when there is no nav or no list", () => {
  assert.deepEqual(parseNavToc(doc(`<p>no toc here</p>`)), []);
  assert.deepEqual(parseNavToc(doc(`<nav epub:type="toc"><p>empty</p></nav>`)), []);
});

// ---- parseNcxToc (EPUB2 toc.ncx) -------------------------------------------

const ncx = (navMap) =>
  `<?xml version="1.0"?><ncx xmlns="http://www.daisy.org/z3986/2005/ncx/">${navMap}</ncx>`;

test("parseNcxToc: reads navPoints, nesting, src and collapsed labels", () => {
  const xml = ncx(`<navMap>
    <navPoint>
      <navLabel><text>  Chapter\n  One  </text></navLabel>
      <content src="ch1.xhtml"/>
      <navPoint>
        <navLabel><text>Section</text></navLabel>
        <content src="ch1.xhtml#sec"/>
      </navPoint>
    </navPoint>
    <navPoint>
      <navLabel><text>Chapter Two</text></navLabel>
      <content src="ch2.xhtml"/>
    </navPoint>
  </navMap>`);
  const toc = parseNcxToc(xml);
  assert.equal(toc.length, 2);
  assert.equal(toc[0].title, "Chapter One");
  assert.equal(toc[0].href, "ch1.xhtml");
  assert.equal(toc[0].children.length, 1);
  assert.deepEqual(toc[0].children[0], { title: "Section", href: "ch1.xhtml#sec", children: [] });
  assert.equal(toc[1].title, "Chapter Two");
});

test("parseNcxToc: returns [] when there is no navMap", () => {
  assert.deepEqual(parseNcxToc(`<?xml version="1.0"?><ncx xmlns="x"><head/></ncx>`), []);
});

// Note: epub.js uses getElementsByTagNameNS("*", "navMap") so an explicit prefix
// (<ncx:navMap>) resolves the same as the default namespace in a real browser DOM.
// The test DOM (linkedom) does not resolve XML namespace prefixes - it treats
// "ncx:navMap" as a literal tag name - so prefix-independence is verified by the
// manual acceptance gates, not here. The default-namespace path is covered above.

// ---- rewriteImg -------------------------------------------------------------

test("rewriteImg: relative src is pointed at the blob URL, srcset dropped", () => {
  const img = el(`<img src="images/p.png" srcset="images/p@2x.png 2x">`, "img");
  const blobFor = (p) => (p === "OEBPS/images/p.png" ? "blob:abc" : null);
  rewriteImg(img, "OEBPS", blobFor);
  assert.equal(img.getAttribute("src"), "blob:abc");
  assert.equal(img.getAttribute("srcset"), null);
});

test("rewriteImg: percent-encoded relative src is decoded before lookup", () => {
  const img = el(`<img src="im%20ages/a%20b.png">`, "img");
  let asked = "";
  rewriteImg(img, "OEBPS", (p) => { asked = p; return "blob:z"; });
  assert.equal(asked, "OEBPS/im ages/a b.png");
  assert.equal(img.getAttribute("src"), "blob:z");
});

test("rewriteImg: http(s)/data sources are left untouched", () => {
  for (const src of ["https://cdn/x.png", "data:image/png;base64,AAAA"]) {
    const img = el(`<img src="${src}">`, "img");
    rewriteImg(img, "OEBPS", () => "blob:should-not-be-used");
    assert.equal(img.getAttribute("src"), src);
  }
});

test("rewriteImg: missing target (or no src) removes the image", () => {
  const missing = el(`<div><img src="gone.png"></div>`, "img");
  rewriteImg(missing, "OEBPS", () => null);
  assert.equal(missing.parentNode, null);

  const noSrc = el(`<div><img alt="x"></div>`, "img");
  rewriteImg(noSrc, "OEBPS", () => "blob:x");
  assert.equal(noSrc.parentNode, null);
});

// ---- convertSvgImage --------------------------------------------------------

test("convertSvgImage: single-image svg is replaced by an <img> with the blob src", () => {
  const im = el(`<div><svg><image xlink:href="cover.jpg" alt="Cover"/></svg></div>`, "image");
  const container = im.closest("div");
  convertSvgImage(im, "OEBPS", (p) => (p === "OEBPS/cover.jpg" ? "blob:cover" : null));
  const img = container.querySelector("img");
  assert.ok(img, "an <img> replaced the svg");
  assert.equal(container.querySelector("svg"), null, "the wrapping svg is gone");
  assert.equal(img.getAttribute("src"), "blob:cover");
  assert.equal(img.getAttribute("alt"), "Cover");
});

test("convertSvgImage: multi-image svg replaces only the <image> node", () => {
  const svgHtml = `<div><svg><image xlink:href="a.jpg"/><image xlink:href="b.jpg"/></svg></div>`;
  const container = el(svgHtml, "div");
  const first = container.querySelector("image");
  convertSvgImage(first, "OEBPS", () => "blob:a");
  assert.ok(container.querySelector("svg"), "the svg survives (more than one image)");
  assert.equal(container.querySelector("img").getAttribute("src"), "blob:a");
  assert.ok(container.querySelector("image"), "the second <image> is still present");
});

test("convertSvgImage: absolute href is kept as-is on the new img", () => {
  const im = el(`<div><svg><image href="https://cdn/x.png"/></svg></div>`, "image");
  const container = im.closest("div");
  convertSvgImage(im, "OEBPS", () => "blob:nope");
  assert.equal(container.querySelector("img").getAttribute("src"), "https://cdn/x.png");
});

// ---- rewriteAnchor ----------------------------------------------------------

const pathIndex = () => new Map([["OEBPS/ch1.xhtml", 0], ["OEBPS/ch2.xhtml", 1]]);

test("rewriteAnchor: same-document fragment is namespaced to this chapter", () => {
  const a = el(`<a href="#intro">Intro</a>`, "a");
  rewriteAnchor(a, 3, "OEBPS", pathIndex());
  assert.equal(a.getAttribute("href"), "#d3-intro");
});

test("rewriteAnchor: cross-chapter link (with and without fragment) maps to the target id", () => {
  const bare = el(`<a href="ch2.xhtml">Two</a>`, "a");
  rewriteAnchor(bare, 0, "OEBPS", pathIndex());
  assert.equal(bare.getAttribute("href"), "#epub-sec-1");

  const frag = el(`<a href="ch2.xhtml#sec">Two.sec</a>`, "a");
  rewriteAnchor(frag, 0, "OEBPS", pathIndex());
  assert.equal(frag.getAttribute("href"), "#d1-sec");
});

test("rewriteAnchor: external links open in a new tab with a safe rel", () => {
  const a = el(`<a href="https://example.com">Ext</a>`, "a");
  rewriteAnchor(a, 0, "OEBPS", pathIndex());
  assert.equal(a.getAttribute("target"), "_blank");
  assert.equal(a.getAttribute("rel"), "noopener noreferrer");
  assert.equal(a.getAttribute("href"), "https://example.com");
});

test("rewriteAnchor: link outside the rendered spine loses its href but keeps its text", () => {
  const a = el(`<a href="notes.xhtml">Notes</a>`, "a");
  rewriteAnchor(a, 0, "OEBPS", pathIndex());
  assert.equal(a.getAttribute("href"), null);
  assert.equal(a.textContent, "Notes");
});

test("rewriteAnchor: a bare <a name> becomes an id target", () => {
  const a = el(`<a name="top"></a>`, "a");
  rewriteAnchor(a, 2, "OEBPS", pathIndex());
  assert.equal(a.id, "top");
});

// ---- renderChapter (the full pipeline) --------------------------------------

test("renderChapter: strips unsafe nodes, rewrites img+links, namespaces ids", () => {
  const xhtml = doc(`
    <h2 id="head">Chapter One</h2>
    <script>alert(1)</script>
    <p id="p1" style="color:red" onclick="bad()">Body <a href="#head">back</a></p>
    <img src="img/pic.png">
    <iframe src="evil"></iframe>`);
  const { frag, label } = renderChapter(xhtml, 5, "OEBPS", pathIndex(), () => "blob:pic");
  const html = fragHtml(frag);

  assert.equal(label, "Chapter One");
  assert.ok(!/script|iframe|onclick|style=/.test(html), `unsafe content removed: ${html}`);
  assert.ok(html.includes('id="d5-head"'), "heading id is namespaced");
  assert.ok(html.includes('id="d5-p1"'), "paragraph id is namespaced");
  assert.ok(html.includes('href="#d5-head"'), "internal link is namespaced");
  assert.ok(html.includes('src="blob:pic"'), "image points at the blob URL");
});

test("renderChapter: exposes a body id as a marker anchor so #id links still resolve", () => {
  const xhtml = `<!doctype html><html><body id="start"><p>Hi <a href="#start">up</a></p></body></html>`;
  const { frag } = renderChapter(xhtml, 1, "OEBPS", pathIndex(), () => null);
  const html = fragHtml(frag);
  assert.ok(html.includes('id="d1-start"'), `marker anchor present: ${html}`);
  assert.ok(html.includes('href="#d1-start"'), "the link resolves to the marker");
});

test("renderChapter: label is empty and text is preserved when there is no heading", () => {
  const { frag, label } = renderChapter(doc(`<p>Just prose.</p>`), 0, "OEBPS", pathIndex(), () => null);
  assert.equal(label, "");
  assert.equal(fragText(frag).trim(), "Just prose.");
});
