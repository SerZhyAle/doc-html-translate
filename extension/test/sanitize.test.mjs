// Unit tests for sanitize.js (sanitizeToFragment) - the sanitize half shared by
// the non-EPUB formats (html.js, md.js, ebook.js). Like the epub DOM path, this
// needs a DOM and was previously only exercised by manual gates; part of the
// cross-edition parity ticket's item #22.

import "./_dom.mjs";
import { fragHtml } from "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";

import { sanitizeToFragment } from "../src/sanitize.js";

const doc = (body) => `<!doctype html><html><body>${body}</body></html>`;

test("sanitizeToFragment: drops unsafe/irrelevant nodes", () => {
  const { frag } = sanitizeToFragment(
    doc(`<p>keep</p><script>bad()</script><style>x{}</style><iframe></iframe><form></form>`),
    0,
  );
  const html = fragHtml(frag);
  assert.ok(html.includes("keep"));
  assert.ok(!/script|style|iframe|form/.test(html), `unsafe tags removed: ${html}`);
});

test("sanitizeToFragment: strips on* handlers and inline styles, namespaces ids", () => {
  const { frag } = sanitizeToFragment(
    doc(`<p id="lead" style="color:red" onmouseover="x()">Body</p>`),
    7,
  );
  const html = fragHtml(frag);
  assert.ok(html.includes('id="d7-lead"'), "id is namespaced");
  assert.ok(!/onmouseover|style=/.test(html), `handlers and styles stripped: ${html}`);
  assert.ok(html.includes("Body"));
});

test("sanitizeToFragment: label is the first h1..h4, whitespace-collapsed", () => {
  const { label } = sanitizeToFragment(doc(`<h3>  Section\n   Title </h3><p>x</p>`), 0);
  assert.equal(label, "Section Title");
});

test("sanitizeToFragment: no heading yields an empty label", () => {
  const { label } = sanitizeToFragment(doc(`<p>no heading</p>`), 0);
  assert.equal(label, "");
});
