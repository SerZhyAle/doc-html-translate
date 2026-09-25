// page-ocr.js - the broker for the whole-page OCR run, running in the background service worker.
// It owns the reader's intent and nothing else: it injects the page agent, asks it for the page's
// pictures, hands them one at a time to the recognizer host, forwards the resulting plates back to
// the page, and can stop and undo the whole thing.
//
// Three roles, and this is the middle one (see DEV/plan/done/2026-09-19_page-ocr-overlay.md):
//   page-agent.js  - the only code in the reader's document; draws and keeps the plates
//   page-ocr.js    - this file; intent, ordering, host lifecycle, stopping
//   ocr-host.js    - an extension-owned document where the engine actually runs
// Only geometry and text cross between them.
//
// Where the recognizer host lives is decided at runtime, not in the manifest. The browser's own
// offscreen document is the right host and is used wherever it exists; where it does not - the
// extension's declared minimum browser version is older than that API, and raising the minimum
// would drop installed readers, which this project does not do - the host is an extension-owned
// frame parked inside the page instead. The frame is on the extension's origin, so the engine
// still compiles under the extension's policy; what the page's policy can do is refuse to load the
// frame at all, and that refusal is reported to the reader rather than swallowed.

const NS = "page-ocr";

// A picture gets this long before the run gives up on it and moves to the next one. Recognition is
// seconds per picture on ordinary art and can be a minute on a dense scan; the ceiling exists so a
// host that died takes one picture down with it and not the run.
const JOB_TIMEOUT_MS = 150000;

// How long to wait for a freshly created host to announce itself. A page whose security policy
// refuses an extension frame never answers, and this is how long before the reader is told so.
const HOST_READY_TIMEOUT_MS = 10000;

const runs = new Map();        // tabId -> run
const pendingJobs = new Map(); // jobId -> { resolve, timer, hostId }
const hostReady = new Map();   // hostId -> { resolve, timer }
let jobSeq = 0;
let offscreenPromise = null;
let offscreenHostId = "";

// ocr-host.html has to stay web-accessible (the frame host is parked inside the reader's page),
// so any site can load its own copy in a frame. Two things keep such a copy useless:
//   - host ids are unguessable, so a copy never matches the id a job is broadcast to;
//   - a host message counts only when it comes from the host page carrying that same id.
// Without both, a site could embed a host named after another tab's and receive that tab's
// pictures, or answer jobs it was never given.
const HOST_PAGE = chrome.runtime.getURL("src/ocr-host.html");

function newHostId(prefix) {
  return `${prefix}-${crypto.randomUUID()}`;
}

function isHostSender(sender, hostId) {
  if (!hostId || !sender || typeof sender.url !== "string") return false;
  // Split by hand: URL.origin is "null" for a non-special scheme in some engines.
  const [page, query = ""] = sender.url.split("#")[0].split("?");
  return page === HOST_PAGE && new URLSearchParams(query).get("host") === hostId;
}

function broadcast(msg) {
  try {
    const p = chrome.runtime.sendMessage({ dht: NS, ...msg });
    if (p && typeof p.catch === "function") p.catch(() => {});
  } catch { /* no listener yet */ }
}

function toTab(tabId, msg) {
  return chrome.tabs.sendMessage(tabId, { dht: NS, ...msg }).catch(() => null);
}

// ---- The recognizer host ---------------------------------------------------

function offscreenAvailable() {
  return typeof chrome.offscreen !== "undefined" && typeof chrome.offscreen.createDocument === "function";
}

function dropWaiter(hostId) {
  const waiter = hostReady.get(hostId);
  if (waiter) { clearTimeout(waiter.timer); hostReady.delete(hostId); }
}

function waitForHost(hostId) {
  return new Promise((resolve) => {
    const timer = setTimeout(() => { hostReady.delete(hostId); resolve(false); }, HOST_READY_TIMEOUT_MS);
    hostReady.set(hostId, { resolve, timer });
  });
}

async function ensureOffscreenHost() {
  if (offscreenPromise) return offscreenPromise;
  offscreenPromise = (async () => {
    const create = async () => {
      const hostId = newHostId("off");
      const ready = waitForHost(hostId);
      try {
        await chrome.offscreen.createDocument({
          url: `src/ocr-host.html?host=${hostId}`,
          reasons: ["BLOBS", "DOM_SCRAPING"],
          justification: "Runs the text-recognition engine outside the web page, so the page's own security policy cannot block it and the page's scripts cannot reach it.",
        });
      } catch (e) {
        dropWaiter(hostId);
        throw e;
      }
      if (!(await ready)) throw new Error("host-silent");
      offscreenHostId = hostId;
      return hostId;
    };
    try {
      return await create();
    } catch (e) {
      // "Only a single offscreen document may be created" means one is already there - from an
      // earlier run in this or another tab. It is the host we want when this worker knows its
      // id; after a worker restart the id is gone with the worker, so it is replaced.
      if (!/single offscreen document/i.test(String(e && e.message))) throw e;
      if (offscreenHostId) return offscreenHostId;
      try { await chrome.offscreen.closeDocument(); } catch { /* raced another close */ }
      return create();
    }
  })();
  try {
    return await offscreenPromise;
  } catch (e) {
    offscreenPromise = null;
    throw e;
  }
}

async function ensureHost(run) {
  if (offscreenAvailable()) {
    run.hostKind = "offscreen";
    run.hostId = await ensureOffscreenHost();
    return;
  }
  const hostId = newHostId(`tab${run.tabId}`);
  const ready = waitForHost(hostId);
  const res = await toTab(run.tabId, {
    t: "host-frame",
    url: chrome.runtime.getURL(`src/ocr-host.html?host=${hostId}`),
  });
  // The ready wait was armed before the request so a fast host cannot answer into nothing; a
  // refusal means no host will ever answer, so the wait and its timer go now.
  if (!res || !res.ok) { dropWaiter(hostId); throw new Error("host-refused"); }
  if (!(await ready)) throw new Error("host-blocked");
  run.hostKind = "frame";
  run.hostId = hostId;
}

// The offscreen document is shared, so it stays only while another run is actually using it. A
// finished run keeps its entry in `runs` (the page's counts live there), which is why the test is
// "running", not "present": counting finished runs kept the host open for the whole session.
function offscreenInUse(except) {
  return [...runs.values()].some((r) => r !== except && r.running && r.hostKind === "offscreen");
}

function closeOffscreen() {
  offscreenPromise = null;
  offscreenHostId = "";
  try {
    const p = chrome.offscreen.closeDocument();
    if (p && typeof p.catch === "function") return p.catch(() => {});
  } catch { /* already gone */ }
  return Promise.resolve();
}

async function releaseHost(run) {
  const kind = run.hostKind;
  run.hostKind = "";
  run.hostId = "";
  if (kind === "frame") { await toTab(run.tabId, { t: "drop-host-frame" }); return; }
  if (kind === "offscreen" && !offscreenInUse(run)) await closeOffscreen();
}

// ---- One picture -----------------------------------------------------------

function recognizeOne(run, picture) {
  const jobId = `j${++jobSeq}`;
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      pendingJobs.delete(jobId);
      resolve({ ok: false, error: "timeout" });
    }, JOB_TIMEOUT_MS);
    pendingJobs.set(jobId, { resolve, timer, hostId: run.hostId });
    run.jobId = jobId;
    broadcast({ hostId: run.hostId, t: "recognize", jobId, src: picture.src, lang: run.lang });
  });
}

// settleJob: hostId, when given, must be the host the job was sent to.
function settleJob(jobId, result, hostId) {
  const pending = pendingJobs.get(jobId);
  if (!pending || (hostId !== undefined && pending.hostId !== hostId)) return;
  clearTimeout(pending.timer);
  pendingJobs.delete(jobId);
  pending.resolve(result);
}

// ---- The run ---------------------------------------------------------------

function status(run, extra) {
  return toTab(run.tabId, {
    t: "status",
    running: run.running,
    stopped: run.stopped,
    done: run.done,
    failed: run.failed,
    total: run.total,
    ...extra,
  });
}

async function getOcrLang() {
  try {
    const got = await chrome.storage.local.get("options");
    return (got.options && got.options.ocrLang) || "eng";
  } catch {
    return "eng";
  }
}

async function injectAgent(tabId) {
  // The plates are styled by the shared stylesheet, not by a copy written for this surface.
  await chrome.scripting.insertCSS({ target: { tabId }, files: ["src/ocr-overlay.css", "src/page-overlay.css"] });
  await chrome.scripting.executeScript({ target: { tabId }, files: ["src/page-agent.js"] });
}

function hostFailureMessage(err) {
  const code = String((err && err.message) || err);
  if (code === "host-blocked" || code === "host-refused" || code === "host-silent") {
    return chrome.i18n.getMessage("pageOcrHostBlocked")
      || "This site does not let the extension start its text recognizer. Use the right-click OCR on a single image instead.";
  }
  return chrome.i18n.getMessage("pageOcrFailedStart") || "Could not start reading this page.";
}

async function drain(run) {
  while (run.queue.length && !run.stopped) {
    const picture = run.queue.shift();
    await toTab(run.tabId, { t: "busy", id: picture.id, on: true });
    const res = await recognizeOne(run, picture);
    await toTab(run.tabId, { t: "busy", id: picture.id, on: false });
    if (run.stopped) break;
    if (res.ok) {
      run.done++;
      await toTab(run.tabId, { t: "plates", id: picture.id, specs: res.specs, htmlLang: res.htmlLang });
    } else {
      // One picture that cannot be fetched or cannot be read is one picture. The count is shown to
      // the reader so a page that mostly failed does not read as a page that mostly had no text.
      run.failed++;
      console.warn("page OCR: picture failed", picture.src, res.error);
    }
    await status(run);
  }
  run.running = false;
  await status(run);
  await releaseHost(run);
}

async function startRun(tabId, { rescan = false } = {}) {
  let run = runs.get(tabId);
  if (run && run.running) return;
  if (!run) {
    run = { tabId, queue: [], done: 0, failed: 0, total: 0, running: false, stopped: false, hostId: "", hostKind: "", jobId: "" };
    runs.set(tabId, run);
  }
  run.stopped = false;
  run.running = true;
  run.lang = await getOcrLang();
  try {
    if (!rescan) await injectAgent(tabId);
    const collected = await toTab(tabId, { t: "collect" });
    const images = (collected && collected.images) || [];
    run.queue = images;
    run.total += images.length;
    if (!images.length) {
      run.running = false;
      await status(run);
      return;
    }
    await status(run);
    await ensureHost(run);
  } catch (e) {
    run.running = false;
    run.queue = [];
    console.warn("page OCR: could not start", e);
    await releaseHost(run);
    await status(run, { error: hostFailureMessage(e) });
    return;
  }
  await drain(run);
}

async function stopRun(tabId) {
  const run = runs.get(tabId);
  if (!run) return;
  run.stopped = true;
  run.queue = [];
  // The stop names this run's job: the offscreen host is shared, and a stop for the host as a
  // whole would cancel another tab's picture too.
  if (run.jobId) broadcast({ hostId: run.hostId, t: "stop", jobId: run.jobId });
  // Settle the picture this run was in the middle of, so drain() is not left waiting on one the
  // reader has already given up on. Only this run's job: another tab's run is not the reader's to
  // stop from here.
  if (run.jobId) settleJob(run.jobId, { ok: false, error: "stopped" });
}

async function removeLayer(tabId) {
  await stopRun(tabId);
  const run = runs.get(tabId);
  if (run) { await releaseHost(run); runs.delete(tabId); }
  await toTab(tabId, { t: "teardown" });
  try {
    await chrome.scripting.removeCSS({ target: { tabId }, files: ["src/ocr-overlay.css", "src/page-overlay.css"] });
  } catch { /* the tab navigated away; the stylesheet went with it */ }
}

// ---- Wiring ----------------------------------------------------------------

chrome.runtime.onMessage.addListener((msg, sender) => {
  if (!msg || msg.dht !== NS) return;
  const tabId = sender.tab && sender.tab.id;
  switch (msg.t) {
    case "host-ready": {
      if (!isHostSender(sender, msg.hostId)) return;
      const waiter = hostReady.get(msg.hostId);
      if (waiter) { clearTimeout(waiter.timer); hostReady.delete(msg.hostId); waiter.resolve(true); }
      return;
    }
    case "job-done":
      if (!isHostSender(sender, msg.hostId)) return;
      settleJob(msg.jobId, msg, msg.hostId);
      return;
    case "job-progress":
      return;
    case "stop":
      if (tabId != null) stopRun(tabId);
      return;
    case "remove":
      if (tabId != null) removeLayer(tabId);
      return;
    case "rescan":
      if (tabId != null) startRun(tabId, { rescan: true });
      return;
    case "ping":
      // The agent pings while a run is going. An incoming message resets the service worker's idle
      // timer, which is what keeps the broker alive across a run longer than the browser's own
      // patience with an idle worker.
      return;
    default:
      return;
  }
});

// A tab that navigates or closes takes its layer with it. The run state must go too, or a later
// run in the same tab starts with the previous page's picture count.
// The picture in flight is settled here too: its page is gone, and leaving it pending held drain()
// - and the worker, through the job timer - for the full job timeout.
function forgetTab(tabId) {
  const run = runs.get(tabId);
  if (!run) return;
  run.stopped = true;
  run.queue = [];
  if (run.jobId) {
    if (run.hostId) broadcast({ hostId: run.hostId, t: "stop", jobId: run.jobId });
    settleJob(run.jobId, { ok: false, error: "navigated" });
  }
  runs.delete(tabId);
  if (run.hostKind === "offscreen" && !offscreenInUse(run)) closeOffscreen();
  run.hostKind = "";
  run.hostId = "";
}

chrome.tabs.onRemoved.addListener((tabId) => forgetTab(tabId));
// tabs.onUpdated rather than webNavigation: it says enough for this, and webNavigation would cost a
// permission the runtime does not otherwise use - which the store listing would then have to
// justify to the reader for nothing.
chrome.tabs.onUpdated.addListener((tabId, info) => { if (info.status === "loading") forgetTab(tabId); });

export { startRun, removeLayer };
