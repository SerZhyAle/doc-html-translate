// badge.js - the toolbar icon's per-tab state, owned by the service worker.
//
// A conversion or an OCR pass runs in a viewer tab (or, for whole-page OCR, through this very
// worker), and the only chrome that survives the tab being in the background is the toolbar
// badge. State is derived from what the extension already knows - the job counts the viewer
// and the page-OCR broker report, and the interception options - never from document content.
//
// The worker is event-driven and its globals die with it; every state change arrives with a
// message or a browser event, so the map is always rebuilt by the next event. A badge left
// over from a dead worker clears on the tab's next navigation (navigated() is wired to
// tabs.onUpdated in background.js).

import { siteHost } from "./site-host.js";
import { siteEnabled } from "./site-mode.js";

// One badge cell, in the popup's own palette: the product blue while work runs, green for a
// finished run, red for a failure, grey for a site whose documents are left alone.
const COLORS = { work: "#2563eb", ok: "#188038", err: "#d93025", off: "#5f6368" };

// How long a finished job leaves its check mark before the badge hands control back to the
// passive state (or nothing). Settles the ticket's open decision: a job leaves a transient
// result, and it always clears on the next navigation regardless of this timer.
const FLASH_MS = 4000;

// tabId -> { convert: job|null, ocr: job|null, error: bool, off: bool }
const tabs = new Map();
// tabId -> timer of a running success flash
const flashes = new Map();

function job() { return { done: 0, total: 0 }; }

function state(tabId) {
  let e = tabs.get(tabId);
  if (!e) { e = { convert: null, ocr: null, error: false, off: false }; tabs.set(tabId, e); }
  return e;
}

function paint(tabId, text, color) {
  try {
    chrome.action.setBadgeText({ tabId, text });
    chrome.action.setBadgeBackgroundColor({ tabId, color });
  } catch { /* no action API in this context (tests) */ }
}

function refresh(tabId) {
  const e = tabs.get(tabId);
  if (!e) { paint(tabId, ""); return; }
  if (e.error) return paint(tabId, "!", COLORS.err);
  const active = e.convert || e.ocr;
  if (active) {
    // Counts when the job knows them ("3/7"), the house ellipsis while it is still
    // downloading or parsing and has nothing to count yet.
    return paint(tabId, active.total > 0 ? `${active.done}/${active.total}` : "..", COLORS.work);
  }
  if (flashes.has(tabId)) return paint(tabId, "✓", COLORS.ok);
  if (e.off) return paint(tabId, "off", COLORS.off);
  paint(tabId, "");
}

function dropFlash(tabId) {
  const timer = flashes.get(tabId);
  if (timer) { clearTimeout(timer); flashes.delete(tabId); }
}

// job reports one viewer/broker job's phase. kind is "convert" or "ocr"; phase is
// "begin" | "progress" | "end"; done/total carry the counts when the job has them.
export function tabJob(tabId, kind, phase, done = 0, total = 0) {
  if (!tabs.has(tabId)) state(tabId);
  const e = state(tabId);
  const slot = kind === "ocr" ? "ocr" : "convert";
  if (phase === "begin") {
    dropFlash(tabId);
    e[slot] = job();
  } else if (phase === "end") {
    e[slot] = null;
  } else if (e[slot]) {
    e[slot].done = done;
    e[slot].total = total;
  }
  if (phase === "end" && !e.convert && !e.ocr && !e.error) {
    // A transient done mark, unless another job took over the cell meanwhile. The flash is
    // recorded before the repaint: this refresh is the one that has to show the check mark.
    flashes.set(tabId, setTimeout(() => { flashes.delete(tabId); refresh(tabId); }, FLASH_MS));
  }
  refresh(tabId);
}

// tabError marks a failed job (a document that would not load or run) with the error cell.
export function tabError(tabId) {
  dropFlash(tabId);
  const e = state(tabId);
  e.convert = null;
  e.ocr = null;
  e.error = true;
  refresh(tabId);
}

// tabReset says the tab started loading a fresh document: its previous document's jobs and
// error are gone; the passive off-state is kept until the new URL says otherwise.
export function tabReset(tabId) {
  dropFlash(tabId);
  const e = tabs.get(tabId);
  if (!e) return;
  e.convert = null;
  e.ocr = null;
  e.error = false;
  refresh(tabId);
}

// tabBase recomputes the passive state from what interception knows about the tab's URL:
// a site whose documents are left alone shows a grey "off" until the reader re-enables it.
export function tabBase(tabId, url, options) {
  const e = state(tabId);
  const host = siteHost(String(url || ""), chrome.runtime.getURL("src/viewer.html"));
  e.off = !!(host && options.enabledByDefault && !siteEnabled(options, host));
  refresh(tabId);
}

// tabNavigated drops everything for a tab whose page went away or started loading: a job has
// nothing to report on, an error is last page's news, and the off-state belongs to the URL.
export function tabNavigated(tabId) {
  dropFlash(tabId);
  tabs.delete(tabId);
  paint(tabId, "");
}

export function tabGone(tabId) {
  tabNavigated(tabId);
}
