// ocr-host.js - the recognizer host for the whole-page OCR run. It owns no UI and no DOM of the
// reader's: it takes an image URL, runs the shipped recognition unit on it, and answers with plate
// specs (text plus geometry in percent of the picture). Nothing else crosses the boundary - the
// page never receives the engine, and the engine never receives the page.
//
// The same document is used two ways, decided by the broker (page-ocr.js):
//   - as an offscreen document, where the browser has one;
//   - as an extension-owned frame parked off-screen inside the reader's page, where it does not.
// Either way the code runs on the extension's origin and under the extension's own content
// security policy, which is what lets the WebAssembly module compile at all. The reader's page
// cannot reach in: an extension frame is a separate origin from the document that hosts it.
//
// ?host=<id> names this host so a broadcast meant for one tab's frame is ignored by another's.
// The broker mints ids nobody can guess: this page is web-accessible (it must be, to be framed
// inside the reader's page), so a site can load a copy of it, and a guessable id would hand that
// copy another tab's pictures.

import { recognize, ocrLangToHtmlLang } from "./ocr-overlay.js";
import { plateSpecs } from "./ocr-plates.js";

const HOST_ID = new URLSearchParams(location.search).get("host") || "";

function send(msg) {
  try {
    const p = chrome.runtime.sendMessage({ ...msg, dht: "page-ocr", hostId: HOST_ID });
    if (p && typeof p.catch === "function") p.catch(() => {});
  } catch { /* the broker went away; nothing to report to */ }
}

// One job at a time is not a policy choice here - the recognition unit already funnels every
// call through a single shared worker. A stop names the job it cancels: the offscreen host
// serves every tab, so one tab's stop must not reach another tab's picture. A cancelled job
// that is still queued is skipped before recognition starts; one already running is not
// answered.
const cancelled = new Set();
const MAX_CANCELLED = 256; // a stop for a job this host never saw must not grow the set forever

async function runJob(job) {
  const isCancelled = () => cancelled.has(job.jobId);
  if (isCancelled()) { cancelled.delete(job.jobId); return; }
  try {
    const { blocks, width, height } = await recognize(job.src, {
      lang: job.lang,
      isCancelled,
      onProgress: (m) => {
        if (m && typeof m.progress === "number") send({ t: "job-progress", jobId: job.jobId, p: m.progress });
      },
    });
    if (isCancelled()) return;
    send({
      t: "job-done",
      jobId: job.jobId,
      ok: true,
      specs: plateSpecs({ blocks, width, height }),
      width,
      height,
      count: blocks.length,
      // The BCP-47 tag for the language the words were read in. The agent puts it on the plates
      // themselves, never on the page: the page's own lang attribute belongs to the page, and this
      // is what tells the browser's translator these particular words are not in it.
      htmlLang: ocrLangToHtmlLang(job.lang),
    });
  } catch (e) {
    if (isCancelled()) return;
    // A picture that cannot be fetched or cannot be read fails alone. The broker moves to the next
    // one; the reader is told how many were skipped, not left with a run that stopped silently.
    send({ t: "job-done", jobId: job.jobId, ok: false, error: String((e && e.message) || e) });
  }
}

// Jobs come from the broker in the service worker, which has no tab. Anything with a tab is a
// page agent or another extension page, and none of them hands out work.
function fromBroker(sender) {
  return !!sender && sender.id === chrome.runtime.id && !sender.tab;
}

chrome.runtime.onMessage.addListener((msg, sender) => {
  if (!HOST_ID || !msg || msg.dht !== "page-ocr" || msg.hostId !== HOST_ID || !fromBroker(sender)) return;
  if (msg.t === "recognize") { runJob(msg).finally(() => cancelled.delete(msg.jobId)); return; }
  if (msg.t === "stop" && msg.jobId) {
    if (cancelled.size >= MAX_CANCELLED) cancelled.clear();
    cancelled.add(msg.jobId);
  }
});

// Announce readiness last, so the broker never sends a job before the listener is attached. The
// broker waits for this with a timeout: on a site whose policy refuses an extension frame the
// message never arrives, and that silence is what the reader is told about.
send({ t: "host-ready" });
