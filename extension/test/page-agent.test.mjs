// Behaviour tests for page-agent.js - the whole-page OCR code that runs in the reader's document.
// It is a classic script injected into a page's isolated world, so it is evaluated here the same
// way: as script text, over a linkedom document, with a fake `chrome` that records listeners and
// messages. Ticket 42 (B49, B50).

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { pathToFileURL, fileURLToPath } from "node:url";
import { parseHTML } from "linkedom";

const SRC = new URL("../src/", import.meta.url);
const agentSrc = readFileSync(new URL("page-agent.js", SRC), "utf8");

class FakeObserver {
  static live = new Set();
  constructor(cb) { this.cb = cb; FakeObserver.live.add(this); }
  observe() {}
  unobserve() {}
  disconnect() { FakeObserver.live.delete(this); }
}

// page builds a fresh document and chrome fake, and returns helpers to run the injected script
// against them the way the broker's executeScript does.
function page(body = "") {
  const { document } = parseHTML(`<!doctype html><html><body>${body}</body></html>`);
  Object.defineProperty(document, "images", { get: () => document.querySelectorAll("img") });

  const docListeners = [];
  const addDoc = document.addEventListener.bind(document);
  const removeDoc = document.removeEventListener.bind(document);
  document.addEventListener = (type, fn, opts) => { docListeners.push({ type, fn }); addDoc(type, fn, opts); };
  document.removeEventListener = (type, fn, opts) => {
    const i = docListeners.findIndex((l) => l.type === type && l.fn === fn);
    if (i >= 0) docListeners.splice(i, 1);
    removeDoc(type, fn, opts);
  };

  const messageListeners = [];
  const sent = [];
  const chrome = {
    runtime: {
      getURL: (p) => pathToFileURL(fileURLToPath(new URL(`../${p}`, import.meta.url))).href,
      sendMessage: (m) => { sent.push(m); return Promise.resolve(); },
      onMessage: {
        addListener: (fn) => messageListeners.push(fn),
        removeListener: (fn) => {
          const i = messageListeners.indexOf(fn);
          if (i >= 0) messageListeners.splice(i, 1);
        },
      },
    },
    i18n: { getMessage: () => "" },
  };
  const window = { scrollX: 0, scrollY: 0, addEventListener() {}, removeEventListener() {} };
  const scope = {
    window, document, chrome,
    MutationObserver: FakeObserver, IntersectionObserver: FakeObserver, ResizeObserver: FakeObserver,
    requestAnimationFrame: () => 0, cancelAnimationFrame: () => {},
    getComputedStyle: () => ({}),
  };
  const run = (src) => new Function(...Object.keys(scope), src)(...Object.values(scope));
  const inject = () => run(agentSrc);
  const ask = (m) => {
    let reply;
    for (const fn of [...messageListeners]) fn({ dht: "page-ocr", ...m }, {}, (r) => { if (reply === undefined) reply = r; });
    return reply;
  };
  return { document, window, chrome, inject, ask, messageListeners, docListeners, sent };
}

// picture gives an <img> the loading state and layout linkedom does not model.
function picture(img, { src, complete = true, w = 400, h = 300 }) {
  Object.defineProperties(img, {
    currentSrc: { value: src, configurable: true },
    complete: { value: complete, configurable: true },
    naturalWidth: { value: complete ? w : 0, configurable: true },
    naturalHeight: { value: complete ? h : 0, configurable: true },
    getBoundingClientRect: { value: () => ({ left: 0, top: 0, right: 200, bottom: 200, width: 200, height: 200 }), configurable: true },
  });
  return img;
}

test("collect skips file: pictures and pictures that never loaded", () => {
  const p = page(`<img id="file"><img id="pending"><img id="broken"><img id="ok"><img id="blob">`);
  const $ = (id) => p.document.getElementById(id);
  picture($("file"), { src: "file:///C:/Users/me/secret.png" });
  picture($("pending"), { src: "https://x.test/lazy.png", complete: false });
  picture($("broken"), { src: "https://x.test/404.png", w: 0, h: 0 });
  picture($("ok"), { src: "https://x.test/art.png" });
  picture($("blob"), { src: "blob:https://x.test/0f1e" });
  p.inject();

  const res = p.ask({ t: "collect" });

  assert.deepEqual(res.images.map((i) => i.src), ["https://x.test/art.png", "blob:https://x.test/0f1e"]);
});

test("a page-dispatched click on the bar's buttons starts nothing; the reader's click does", () => {
  const p = page();
  p.inject();
  p.ask({ t: "status", running: false, done: 0, total: 0 });
  const rescan = [...p.document.querySelectorAll("#dht-ocr-bar button")].find((b) => b.textContent === "Scan new pictures");
  assert.ok(rescan, "the bar carries the rescan button");

  // What a page script can do: click() or dispatchEvent - neither is a trusted event.
  rescan.dispatchEvent(new p.document.defaultView.Event("click"));
  if (typeof rescan.click === "function") rescan.click();
  assert.deepEqual(p.sent.filter((m) => m.t === "rescan"), []);

  const trusted = new p.document.defaultView.Event("click");
  Object.defineProperty(trusted, "isTrusted", { value: true });
  rescan.dispatchEvent(trusted);
  assert.equal(p.sent.filter((m) => m.t === "rescan").length, 1);
});

test("Remove then Start leaves one message listener and no stale observers", () => {
  FakeObserver.live.clear();
  const p = page();
  p.inject();
  p.ask({ t: "collect" }); // builds the bar, which starts watching for new pictures
  assert.equal(p.messageListeners.length, 1);
  assert.equal(p.docListeners.filter((l) => l.type === "load").length, 1);

  p.ask({ t: "teardown" });
  assert.equal(p.messageListeners.length, 0, "teardown removes the agent's message listener");
  assert.equal(p.docListeners.length, 0, "teardown removes the agent's document listeners");
  assert.equal(FakeObserver.live.size, 0, "teardown disconnects every observer");

  p.inject();
  p.ask({ t: "collect" });
  assert.equal(p.messageListeners.length, 1, "a second Start answers with one agent, not two");
  assert.equal(p.document.querySelectorAll("#dht-ocr-bar").length, 1);
  p.ask({ t: "teardown" });
});

test("pixels come from the picture as the page shows it, and a tainted canvas says so", () => {
  const p = page(`<img id="a">`);
  const img = picture(p.document.getElementById("a"), { src: "https://x.test/a.png" });
  let taint = false;
  const createElement = p.document.createElement.bind(p.document);
  p.document.createElement = (tag) => {
    if (tag !== "canvas") return createElement(tag);
    return {
      width: 0,
      height: 0,
      getContext: () => ({ drawImage: (src) => assert.equal(src, img) }),
      toDataURL: () => {
        if (taint) { const e = new Error("tainted"); e.name = "SecurityError"; throw e; }
        return "data:image/png;base64,AAAA";
      },
    };
  };
  p.inject();
  const [{ id }] = p.ask({ t: "collect" }).images;

  assert.deepEqual(p.ask({ t: "pixels", id }), { ok: true, src: "data:image/png;base64,AAAA" });
  taint = true;
  assert.deepEqual(p.ask({ t: "pixels", id }), { ok: false, error: "tainted" });
  assert.deepEqual(p.ask({ t: "pixels", id: "nope" }), { ok: false, error: "unloaded" });
  p.ask({ t: "teardown" });
});
