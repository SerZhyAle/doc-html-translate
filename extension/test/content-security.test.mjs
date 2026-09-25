// Content-security guards that are not one module's behaviour: the manifest's web-accessible
// surface, the export's own policy and the MOBI link retargeting. See
// DEV/plan/19_2026-09-24_bugfix-extension-content-security.md.

import "./_dom.mjs";
import { fragHtml } from "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

import { buildExportHtml, EXPORT_CSP } from "../src/export-html.js";
import { retargetBookLinks, applyBookLinks } from "../src/ebook.js";
import { sanitizeToFragment } from "../src/sanitize.js";

const manifest = JSON.parse(fs.readFileSync(path.join(import.meta.dirname, "..", "manifest.json"), "utf8"));

// Every web-accessible resource is reachable by every site the entry matches, so each one has to
// name the feature that cannot work without it. Adding a file here means adding its reason.
const WEB_ACCESSIBLE = {
  "src/viewer.html": "declarativeNetRequest can only redirect a document request to a web-accessible page",
  "src/ocr-host.html": "the page-OCR host is framed inside the reader's page where offscreen documents are missing",
  "src/ocr-plates.js": "the page agent (a content script) imports it dynamically",
};

test("only the resources a feature needs are web-accessible", () => {
  const listed = manifest.web_accessible_resources.flatMap((e) => e.resources);
  assert.deepEqual([...listed].sort(), Object.keys(WEB_ACCESSIBLE).sort());
  assert.ok(!listed.some((r) => r.includes("*")), "no wildcard entries");
});

test("the OCR host is not exposed to non-web schemes", () => {
  const entry = manifest.web_accessible_resources.find((e) => e.resources.includes("src/ocr-host.html"));
  assert.ok(!entry.matches.includes("<all_urls>"), entry.matches.join(","));
});

test("the export carries a script-free policy before any content", () => {
  const html = buildExportHtml({ title: "T", theme: "light", lang: "en", styleVars: "", css: "p{}", body: '<a href="#x">x</a>' });
  const head = html.slice(0, html.indexOf("<title>"));
  assert.ok(head.includes(`http-equiv="Content-Security-Policy" content="${EXPORT_CSP}"`));
  for (const directive of ["default-src 'none'", "script-src 'none'", "object-src 'none'", "base-uri 'none'", "form-action 'none'"]) {
    assert.ok(EXPORT_CSP.includes(directive), directive);
  }
  assert.ok(!/unsafe-eval|script-src [^;]*'unsafe-inline'/.test(EXPORT_CSP));
});

test("MOBI filepos and kindle:pos links become in-page anchors through the sanitizer", async () => {
  const book = {
    splitTOCHref: (href) => [2, `filepos${href.split(":")[1]}`],
    resolveHref: async (href) => (href.includes("0001") ? { index: 5 } : undefined),
  };
  const src = `<!doctype html><html><body>` +
    `<a href="filepos:123">a</a><a href="kindle:pos:fid:0001:off:0000">b</a>` +
    `<a href="kindle:pos:fid:9999:off:0000">c</a><a data-dht-target="x" href="javascript:y()">d</a>` +
    `</body></html>`;
  const { frag } = sanitizeToFragment(await retargetBookLinks(src, book), 2);
  applyBookLinks(frag);
  const html = fragHtml(frag);
  assert.ok(html.includes('href="#d2-filepos123"'), html);
  assert.ok(html.includes('href="#ebook-sec-5"'), html);
  assert.ok(!/javascript|data-dht-target|kindle:|filepos:/.test(html), html);
});
