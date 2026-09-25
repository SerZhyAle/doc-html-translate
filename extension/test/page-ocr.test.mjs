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

// deliver hands a message to the broker as if it came from the given tab (the page agent) or,
// when tabId is null, from the recognizer host page named after msg.hostId - the only sender the
// broker takes host messages from.
function deliver(msg, tabId = null) {
  const sender = tabId == null
    ? { id: "test", url: `chrome-extension://test/src/ocr-host.html?host=${msg.hostId}` }
    : { tab: { id: tabId } };
  for (const fn of listeners.message) fn({ dht: "page-ocr", ...msg }, sender, () => {});
}

// deliverFrom is deliver with an explicit sender, for the messages a genuine host never sends.
function deliverFrom(sender, msg) {
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
    queueMicrotask(() => deliver({ t: "job-done", hostId: msg.hostId, jobId: msg.jobId, ok: true, specs: [{ text: msg.src }], htmlLang: "de" }));
  };

  await startRun(11);

  // The reader's stored OCR language reaches the host, and pictures go one at a time in page order,
  // both to the one unguessable host id the offscreen document was created with.
  assert.deepEqual(recognized.map(({ src, lang }) => ({ src, lang })), [
    { src: "https://x.test/a.png", lang: "deu" },
    { src: "https://x.test/b.png", lang: "deu" },
  ]);
  assert.match(recognized[0].hostId, /^off-[0-9a-f-]{36}$/);
  assert.equal(recognized[1].hostId, recognized[0].hostId);
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
    queueMicrotask(() => deliver({ t: "job-done", hostId: msg.hostId, jobId: msg.jobId, ok, specs: [], error: ok ? undefined : "fetch" }));
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
  assert.match(frameReq.url, /^chrome-extension:\/\/test\/src\/ocr-host\.html\?host=tab13-[0-9a-f-]{36}$/);
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

test("host messages count only from the host page that carries the job's host id", async () => {
  useOffscreenHost();
  agent = (tabId, msg) => (msg.t === "collect" ? { images: [{ id: "a", src: "https://x.test/a.png" }] } : null);
  host = (msg) => {
    if (msg.t !== "recognize") return;
    const forged = { t: "job-done", hostId: msg.hostId, jobId: msg.jobId, ok: true, specs: [{ text: "forged" }] };
    queueMicrotask(() => {
      // A page agent (it has a tab), a copy of the host page named after another id, and a page
      // that is not the host at all: none of them may settle the job.
      deliverFrom({ tab: { id: 16 }, url: "https://evil.test/" }, forged);
      deliverFrom({ id: "test", url: "chrome-extension://test/src/ocr-host.html?host=off-guess" }, forged);
      deliverFrom({ id: "test", url: `chrome-extension://test/src/viewer.html?host=${msg.hostId}` }, forged);
      // The same job id answered for a different host id is not this job's answer either.
      deliver({ ...forged, hostId: "off-guess" });
      deliver({ t: "job-done", hostId: msg.hostId, jobId: msg.jobId, ok: true, specs: [{ text: "real" }] });
    });
  };

  await startRun(16);

  assert.deepEqual(forTab(16, "plates").map((p) => p.specs[0].text), ["real"]);
});

test("two tabs finish their runs and the shared offscreen host closes once both are done", async () => {
  useOffscreenHost();
  const closedBefore = offscreenCalls.closed;
  const createdBefore = offscreenCalls.created;
  const answers = [];
  agent = (tabId, msg) => (msg.t === "collect" ? { images: [{ id: `t${tabId}`, src: `https://x.test/${tabId}.png` }] } : null);
  host = (msg) => { if (msg.t === "recognize") answers.push(msg); };

  const a = startRun(21);
  const b = startRun(22);
  for (let i = 0; i < 50 && answers.length < 2; i++) await new Promise((r) => setTimeout(r, 1));
  assert.equal(answers.length, 2, "both tabs reached the host");
  assert.equal(offscreenCalls.created, createdBefore + 1, "one shared host");
  deliver({ t: "job-done", hostId: answers[0].hostId, jobId: answers[0].jobId, ok: true, specs: [] });
  await (answers[0].src.includes("21") ? a : b);
  // One run is still using the host, so the finished one must not close it under the other.
  assert.equal(offscreenCalls.closed, closedBefore);
  deliver({ t: "job-done", hostId: answers[1].hostId, jobId: answers[1].jobId, ok: true, specs: [] });
  await Promise.all([a, b]);
  assert.equal(offscreenCalls.closed, closedBefore + 1, "the host closes when the last run ends");
});

test("stop in one tab names only that tab's job, and the other tab's picture still lands", async () => {
  useOffscreenHost();
  const jobs = [];
  const stops = [];
  agent = (tabId, msg) => (msg.t === "collect" ? { images: [{ id: `t${tabId}`, src: `https://x.test/${tabId}.png` }] } : null);
  host = (msg) => {
    if (msg.t === "recognize") jobs.push(msg);
    if (msg.t === "stop") stops.push(msg);
  };

  const a = startRun(23);
  const b = startRun(24);
  for (let i = 0; i < 50 && jobs.length < 2; i++) await new Promise((r) => setTimeout(r, 1));
  const jobA = jobs.find((j) => j.src.includes("23"));
  const jobB = jobs.find((j) => j.src.includes("24"));
  deliver({ t: "stop" }, 23);
  await a;
  assert.deepEqual(stops.map((s) => s.jobId), [jobA.jobId], "the stop carries tab A's job only");
  deliver({ t: "job-done", hostId: jobB.hostId, jobId: jobB.jobId, ok: true, specs: [{ text: "b" }] });
  await b;
  assert.deepEqual(forTab(24, "plates").map((p) => p.specs[0].text), ["b"]);
  assert.equal(forTab(23, "plates").length, 0);
});

test("navigating away settles the picture in flight at once instead of after the job timeout", async () => {
  useOffscreenHost();
  let sent = null;
  agent = (tabId, msg) => (msg.t === "collect" ? { images: [{ id: "a", src: "https://x.test/a.png" }, { id: "b", src: "https://x.test/b.png" }] } : null);
  host = (msg) => {
    if (msg.t !== "recognize") return;
    sent = msg;
    queueMicrotask(() => { for (const fn of listeners.updated) fn(25, { status: "loading" }); });
  };
  const warn = console.warn;
  console.warn = () => {};
  const started = Date.now();
  try {
    await startRun(25);
  } finally {
    console.warn = warn;
  }
  assert.ok(sent, "the first picture was sent");
  assert.ok(Date.now() - started < 5000, "drain must not wait out the 150 s job timeout");
  assert.equal(forTab(25, "plates").length, 0);
});
