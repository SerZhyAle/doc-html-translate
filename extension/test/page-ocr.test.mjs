// Behaviour tests for page-ocr.js - the whole-page OCR broker in the background service worker.
// It talks to three parties only through chrome.* messages (the page agent via tabs.sendMessage,
// the recognizer host via runtime.sendMessage, and replies on runtime.onMessage), so a fake
// `chrome` that records what was sent and can answer as the agent or the host is enough to drive a
// whole run without a browser, an engine or a page.

import { test } from "node:test";
import assert from "node:assert/strict";

// Each test swaps in its own agent and host behaviour; the listeners the module registers on
// import stay bound to this one fake for the life of the process.
const listeners = { message: [], removed: [], updated: [] };
const sentToTab = [];
let agent = () => null;          // (tabId, msg) -> reply of the page agent
let host = () => {};             // (msg) -> the host's reaction to a broadcast
let offscreen;                   // set per test: present = offscreen host, undefined = frame host
const offscreenCalls = { created: 0, closed: 0 };

// deliver hands a message to the broker as if it came from the given tab (or from the extension
// itself when tabId is null), which is how the host and the agent both reach it.
function deliver(msg, tabId = null) {
  const sender = tabId == null ? {} : { tab: { id: tabId } };
  for (const fn of listeners.message) fn({ dht: "page-ocr", ...msg }, sender, () => {});
}

globalThis.chrome = {
  runtime: {
    getURL: (p) => `chrome-extension://test/${p}`,
    sendMessage: async (msg) => { host(msg); },
    onMessage: { addListener: (fn) => listeners.message.push(fn) },
  },
  tabs: {
    sendMessage: async (tabId, msg) => {
      sentToTab.push({ tabId, ...msg });
      return agent(tabId, msg);
    },
    onRemoved: { addListener: (fn) => listeners.removed.push(fn) },
    onUpdated: { addListener: (fn) => listeners.updated.push(fn) },
  },
  scripting: {
    insertCSS: async () => {},
    executeScript: async () => {},
    removeCSS: async () => {},
  },
  storage: { local: { get: async () => ({ options: { ocrLang: "deu" } }) } },
  i18n: { getMessage: () => "" },
};
Object.defineProperty(globalThis.chrome, "offscreen", { get: () => offscreen, configurable: true });

const { startRun, removeLayer } = await import("../src/page-ocr.js");

const forTab = (tabId, t) => sentToTab.filter((m) => m.tabId === tabId && m.t === t);
const lastStatus = (tabId) => forTab(tabId, "status").at(-1);

function useOffscreenHost() {
  offscreen = {
    createDocument: async ({ url }) => {
      offscreenCalls.created++;
      // The real host announces itself once its script has loaded; answer the same way.
      const hostId = new URL(url, "chrome-extension://test/").searchParams.get("host");
      queueMicrotask(() => deliver({ t: "host-ready", hostId }));
    },
    closeDocument: async () => { offscreenCalls.closed++; },
  };
}

test("a run recognizes every picture in order and forwards the plates to the page", async () => {
  useOffscreenHost();
  const recognized = [];
  agent = (tabId, msg) => (msg.t === "collect"
    ? { images: [{ id: "a", src: "https://x.test/a.png" }, { id: "b", src: "https://x.test/b.png" }] }
    : null);
  host = (msg) => {
    if (msg.t !== "recognize") return;
    recognized.push({ src: msg.src, lang: msg.lang, hostId: msg.hostId });
    queueMicrotask(() => deliver({ t: "job-done", jobId: msg.jobId, ok: true, specs: [{ text: msg.src }], htmlLang: "de" }));
  };

  await startRun(11);

  // The reader's stored OCR language reaches the host, and pictures go one at a time in page order.
  assert.deepEqual(recognized, [
    { src: "https://x.test/a.png", lang: "deu", hostId: "offscreen" },
    { src: "https://x.test/b.png", lang: "deu", hostId: "offscreen" },
  ]);
  const plates = forTab(11, "plates");
  assert.deepEqual(plates.map((p) => [p.id, p.specs[0].text, p.htmlLang]), [
    ["a", "https://x.test/a.png", "de"],
    ["b", "https://x.test/b.png", "de"],
  ]);
  const st = lastStatus(11);
  assert.equal(st.running, false);
  assert.equal(st.done, 2);
  assert.equal(st.failed, 0);
  assert.equal(st.total, 2);
  // Nothing else uses the shared offscreen host, so the finished run closes it.
  assert.equal(offscreenCalls.created, 1);
  assert.equal(offscreenCalls.closed, 1);
});

test("a picture the host cannot read is counted as failed, and the run carries on", async () => {
  useOffscreenHost();
  agent = (tabId, msg) => (msg.t === "collect"
    ? { images: [{ id: "bad", src: "https://x.test/bad.png" }, { id: "ok", src: "https://x.test/ok.png" }] }
    : null);
  host = (msg) => {
    if (msg.t !== "recognize") return;
    const ok = !msg.src.includes("bad");
    queueMicrotask(() => deliver({ t: "job-done", jobId: msg.jobId, ok, specs: [], error: ok ? undefined : "fetch" }));
  };
  const warn = console.warn;
  console.warn = () => {};
  try {
    await startRun(12);
  } finally {
    console.warn = warn;
  }

  assert.deepEqual(forTab(12, "plates").map((p) => p.id), ["ok"]);
  const st = lastStatus(12);
  assert.equal(st.done, 1);
  assert.equal(st.failed, 1);
  assert.equal(st.running, false);
});

test("a page that refuses the recognizer frame is told so instead of left waiting", async (t) => {
  // No offscreen API: the broker has to park a host frame in the page, and the page says no.
  offscreen = undefined;
  // The host-ready wait armed before the frame request outlives the refusal; fake timers let the
  // test run it out instead of holding the process open for the real ten seconds.
  t.mock.timers.enable({ apis: ["setTimeout"] });
  let recognizeSent = false;
  agent = (tabId, msg) => {
    if (msg.t === "collect") return { images: [{ id: "a", src: "https://x.test/a.png" }] };
    if (msg.t === "host-frame") return { ok: false };
    return null;
  };
  host = (msg) => { if (msg.t === "recognize") recognizeSent = true; };
  const warn = console.warn;
  console.warn = () => {};
  try {
    await startRun(13);
  } finally {
    console.warn = warn;
  }

  const frameReq = forTab(13, "host-frame")[0];
  assert.match(frameReq.url, /^chrome-extension:\/\/test\/src\/ocr-host\.html\?host=tab13-\d+$/);
  const st = lastStatus(13);
  assert.equal(st.running, false);
  assert.match(st.error, /does not let the extension start its text recognizer/);
  assert.equal(recognizeSent, false);
  t.mock.timers.tick(10000);
});

test("a page with no pictures ends the run without starting a host", async () => {
  useOffscreenHost();
  const created = offscreenCalls.created;
  agent = (tabId, msg) => (msg.t === "collect" ? { images: [] } : null);

  await startRun(14);

  assert.equal(offscreenCalls.created, created);
  const st = lastStatus(14);
  assert.equal(st.running, false);
  assert.equal(st.total, 0);
  assert.equal(st.error, undefined);
});

test("stop from the page settles the picture in flight and removal tears the layer down", async () => {
  useOffscreenHost();
  agent = (tabId, msg) => (msg.t === "collect"
    ? { images: [{ id: "a", src: "https://x.test/a.png" }, { id: "b", src: "https://x.test/b.png" }] }
    : null);
  // The host never answers: only the reader's stop can release the run.
  let recognizeCount = 0;
  host = (msg) => {
    if (msg.t !== "recognize") return;
    recognizeCount++;
    queueMicrotask(() => deliver({ t: "stop" }, 15));
  };

  await startRun(15);

  assert.equal(recognizeCount, 1, "the second picture must not be sent after a stop");
  assert.equal(forTab(15, "plates").length, 0);
  assert.equal(lastStatus(15).running, false);

  await removeLayer(15);
  assert.equal(forTab(15, "teardown").length, 1);
});
