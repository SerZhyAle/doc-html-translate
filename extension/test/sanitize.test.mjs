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

// ---- Content security (DEV/plan/done/19_2026-09-24_bugfix-extension-content-security.md) ----

import { restoreRemote, urlKind, isSafeLinkHref, resourceVerdict } from "../src/url-policy.js";

test("script-scheme links lose their href, whatever the spelling", () => {
  const hrefs = ["javascript:alert(1)", "JaVaScRiPt:x()", " javascript:x()", "java&#9;script:x()",
    "vbscript:x", "data:text/html,<script>x()</script>", "file:///etc/passwd"];
  const body = hrefs.map((h, i) => `<a id="l${i}" href="${h}">t${i}</a>`).join("");
  const { frag } = sanitizeToFragment(doc(body), 0);
  const html = fragHtml(frag);
  assert.ok(!/href=/.test(html), `every unsafe href removed: ${html}`);
  assert.ok(html.includes("t6"), "the link text stays as inert text");
});

test("safe links survive; same-section fragments follow the namespaced ids", () => {
  const { frag } = sanitizeToFragment(doc(
    `<a href="https://x.test/">w</a><a href="mailto:a@b.c">m</a><a href="#fn1">1</a><p id="fn1">note</p>`,
  ), 3);
  const html = fragHtml(frag);
  assert.ok(html.includes('href="https://x.test/"'));
  assert.ok(html.includes('href="mailto:a@b.c"'));
  assert.ok(html.includes('href="#d3-fn1"') && html.includes('id="d3-fn1"'), `fragment retargeted: ${html}`);
});

test("SVG links and animations cannot smuggle a script URL", () => {
  const { frag } = sanitizeToFragment(doc(
    `<svg><a xlink:href="javascript:x()"><text>a</text></a><a href="javascript:y()"><text>b</text></a>` +
    `<a><set attributeName="href" to="javascript:z()"/><animate attributeName="href" values="javascript:z()"/><text>c</text></a></svg>`,
  ), 0);
  const html = fragHtml(frag);
  assert.ok(!/javascript/i.test(html), `no script URL left: ${html}`);
});

test("name attributes are removed; an anchor's name becomes its namespaced id", () => {
  const { frag } = sanitizeToFragment(doc(
    `<img name="getElementById" src="data:image/png;base64,AA"><form name="x"></form><a name="ch1">One</a><a href="#ch1">go</a>`,
  ), 2);
  const html = fragHtml(frag);
  assert.ok(!/name=/.test(html), `no name left: ${html}`);
  assert.ok(html.includes('id="d2-ch1"') && html.includes('href="#d2-ch1"'), html);
});

test("remote resources are parked until allowed; local ones are kept", () => {
  const { frag, remote } = sanitizeToFragment(doc(
    `<img id="a" src="https://t.test/pixel.gif"><img id="b" src="data:image/png;base64,AA">` +
    `<img id="c" srcset="blob:x 1x, //t.test/b.png 2x"><video id="d" poster="http://t.test/p.jpg"></video>` +
    `<table id="e" background="https://t.test/bg.png"></table><svg><image id="f" href="https://t.test/i.png"/></svg>` +
    `<img id="g" src="chrome-extension://abc/x.png">`,
  ), 0);
  assert.equal(remote, 5);
  const host = document.createElement("div");
  host.appendChild(frag);
  const html = host.innerHTML;
  assert.ok(!/ (src|srcset|poster|background|href)="(https?:)?\/\//.test(html), `no live remote URL: ${html}`);
  assert.equal(host.querySelector("#d0-b").getAttribute("src"), "data:image/png;base64,AA");
  assert.equal(host.querySelector("#d0-g").hasAttribute("src"), false, "an unknown scheme is dropped, not parked");

  restoreRemote(host);
  assert.equal(host.querySelector("#d0-a").getAttribute("src"), "https://t.test/pixel.gif");
  assert.equal(host.querySelector("#d0-c").getAttribute("srcset"), "blob:x 1x, //t.test/b.png 2x");
  assert.equal(host.querySelector("#d0-d").getAttribute("poster"), "http://t.test/p.jpg");
  assert.equal(host.querySelector("#d0-f").getAttribute("href"), "https://t.test/i.png");
  assert.equal(host.querySelectorAll("[data-dht-remote]").length, 0);
});

test("a document cannot pre-park a URL of its own for the restore to trust", () => {
  const { frag, remote } = sanitizeToFragment(doc(
    `<a data-dht-remote="" data-dht-remote-href="javascript:x()">t</a>`,
  ), 0);
  const host = document.createElement("div");
  host.appendChild(frag);
  assert.equal(remote, 0);
  restoreRemote(host);
  assert.ok(!/javascript/.test(host.innerHTML), host.innerHTML);
});

test("urlKind and the verdicts classify the edge cases", () => {
  assert.equal(urlKind("#x"), "fragment");
  assert.equal(urlKind("img/a.png"), "relative");
  assert.equal(urlKind("//cdn/a.png"), "protocol-relative");
  assert.equal(urlKind("\u0001java\nscript:x"), "javascript");
  assert.equal(isSafeLinkHref("tel:+1"), true);
  assert.equal(isSafeLinkHref("blob:x"), false);
  assert.equal(resourceVerdict("blob:x"), "keep");
  assert.equal(resourceVerdict("HTTPS://x/a.png"), "remote");
  assert.equal(resourceVerdict("javascript:x"), "drop");
});
