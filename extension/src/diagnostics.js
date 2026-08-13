// diagnostics.js - the extension's whole diagnostics surface: what the last document was, and
// a short text the user can paste into a mail to the author.
//
// The browser sandbox hosts no run-log store worth archiving, so this edition deliberately has
// a different shape from the desktop app's log archive - see docs/PARITY.md ("Intentional
// divergences") and the ticket's ADR-4. What it does have is the useful part of a report:
// version, browser, format, page count, options and the last error.
//
// Nothing here records document content: no file name, no page text, no URL, and the host list
// is reported as a count, never as names.

import { DEFAULT_OPTIONS } from "./defaults.js";
import { uiLang } from "./i18n.js";

const KEY = "lastRun";

// An error message is a message, not a stack: 200 characters is enough to name the failure and
// short enough that it cannot smuggle a document into the report.
const MAX_ERROR = 200;

// recordRun remembers what the viewer just did. It swallows storage errors: diagnostics exist
// to explain a broken render, never to cause one.
export async function recordRun({ format, pages, error } = {}) {
  try {
    const prev = (await chrome.storage.local.get(KEY))[KEY] || {};
    const run = {
      format: format != null ? String(format) : prev.format || "",
      pages: pages != null ? Number(pages) : prev.pages || 0,
      error: error != null ? String(error).slice(0, MAX_ERROR) : prev.error || "",
      at: new Date().toISOString(),
    };
    await chrome.storage.local.set({ [KEY]: run });
  } catch (e) {
    /* diagnostics must never break a render */
  }
}

export async function readRun() {
  try {
    return (await chrome.storage.local.get(KEY))[KEY] || null;
  } catch (e) {
    return null;
  }
}

// reportText is the paste-ready summary. It is **English whatever the interface language is**,
// so the author reads one format - the same rule the desktop archive's environment.txt follows,
// and the same shared field labels (version, platform, interface language, ocr).
export async function reportText(manifestVersion, options) {
  const o = { ...DEFAULT_OPTIONS, ...(options || {}) };
  const run = (await readRun()) || {};
  const nav = typeof navigator !== "undefined" ? navigator : {};

  const fields = [
    ["edition", "browser extension"],
    ["version", manifestVersion || ""],
    ["platform", nav.platform || ""],
    ["user agent", nav.userAgent || ""],
    ["interface language", uiLang()],
    ["auto reflow", o.enabledByDefault ? "on" : "off"],
    ["theme", o.theme],
    ["source language", o.sourceLang],
    ["ocr", o.ocrImages ? "on" : "off"],
    ["ocr language", o.ocrLang],
    // The count, never the hosts: which sites someone reads is their business.
    ["disabled hosts", String((o.disabledHosts || []).length)],
    ["last format", run.format || "none"],
    ["last pages", String(run.pages || 0)],
    ["last error", run.error || "none"],
    ["last run at", run.at || "never"],
  ];
  return fields.map(([k, v]) => `${k}: ${v}`).join("\n") + "\n";
}
