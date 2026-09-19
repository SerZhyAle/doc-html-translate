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
// call through a single shared worker - but the host tracks it anyway so a stop can be honoured
// between jobs rather than only after the whole queue drains.
let stopped = false;

async function runJob(job) {
  if (stopped) return;
  try {
    const { blocks, width, height } = await recognize(job.src, {
      lang: job.lang,
      onProgress: (m) => {
        if (m && typeof m.progress === "number") send({ t: "job-progress", jobId: job.jobId, p: m.progress });
      },
    });
    if (stopped) return;
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
    // A picture that cannot be fetched or cannot be read fails alone. The broker moves to the next
    // one; the reader is told how many were skipped, not left with a run that stopped silently.
    send({ t: "job-done", jobId: job.jobId, ok: false, error: String((e && e.message) || e) });
  }
}

chrome.runtime.onMessage.addListener((msg) => {
  if (!msg || msg.dht !== "page-ocr" || msg.hostId !== HOST_ID) return;
  if (msg.t === "recognize") { stopped = false; runJob(msg); return; }
  if (msg.t === "stop") { stopped = true; return; }
});

// Announce readiness last, so the broker never sends a job before the listener is attached. The
// broker waits for this with a timeout: on a site whose policy refuses an extension frame the
// message never arrives, and that silence is what the reader is told about.
send({ t: "host-ready" });
