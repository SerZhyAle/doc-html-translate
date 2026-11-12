// diagnostics.js - the extension's whole diagnostics surface: what the last document was, and
// the JSON diagnostic envelope the user can paste into a mail to the author.
//
// The browser sandbox hosts no run-log store worth archiving, so this edition deliberately has
// a different shape from the desktop app's log archive - see docs/PARITY.md ("Intentional
// divergences") and the ticket's ADR-4. What it does have is the useful part of a report:
// version, browser, format, page count, options and the last error, in the envelope of
// DIAGNOSTIC-REPORT 8 A with the readable lines beside it.
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
//
// A call naming a format starts a new run, so the previous document's page count and error do not
// carry over into it - a clean open used to keep reporting the last failure. Writes are
// read-modify-write, so they run one after another; unawaited callers used to overwrite each
// other's fields.
let writeTail = Promise.resolve();

export function recordRun(patch = {}) {
  const run = writeTail.then(() => writeRun(patch));
  writeTail = run;
  return run;
}

async function writeRun({ format, pages, error } = {}) {
  try {
    const prev = format != null ? {} : (await chrome.storage.local.get(KEY))[KEY] || {};
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

// SCHEMA_VERSION is the version of the diagnostic envelope (DIAGNOSTIC-REPORT 8 B: `schemaVersion`
// in the JSON carrier, `schema_version` in the desktop edition's environment.txt). The two editions
// move it together - tests/parity_test.go pins it.
export const SCHEMA_VERSION = 1;

// reportFields are the facts of the report as [label, value] pairs. The labels are English
// whatever the interface language is, and the same shared labels (version, platform, interface
// language, ocr) as the desktop archive's report.txt.
async function reportFields(manifestVersion, options) {
  const o = { ...DEFAULT_OPTIONS, ...(options || {}) };
  const run = (await readRun()) || {};
  const nav = typeof navigator !== "undefined" ? navigator : {};

  return [
    ["edition", "browser extension"],
    ["version", manifestVersion || ""],
    ["platform", nav.platform || ""],
    ["user agent", nav.userAgent || ""],
    ["interface language", uiLang()],
    ["auto reflow", o.enabledByDefault ? "on" : "off"],
    ["site mode", o.siteMode === "allowlist" ? "allowlist" : "all"],
    ["theme", o.theme],
    ["source language", o.sourceLang],
    ["ocr", o.ocrImages ? "on" : "off"],
    ["ocr language", o.ocrLang],
    // The count, never the hosts: which sites someone reads is their business.
    ["disabled hosts", String((o.disabledHosts || []).length)],
    ["allowed hosts", String((o.allowedHosts || []).length)],
    ["last format", run.format || "none"],
    ["last pages", String(run.pages || 0)],
    ["last error", run.error || "none"],
    ["last run at", run.at || "never"],
  ];
}

// reportText is the readable summary, one `label: value` line per fact. It is **English whatever
// the interface language is**, so the author reads one format.
export async function reportText(manifestVersion, options) {
  const fields = await reportFields(manifestVersion, options);
  return fields.map(([k, v]) => `${k}: ${v}`).join("\n") + "\n";
}

// reportEnvelope is the report as the single JSON diagnostic envelope of DIAGNOSTIC-REPORT 8 A:
// what a program reads (`app`, `environment`, `options`, `lastRun`) and, beside it, the readable
// lines in `report`. It is what the clipboard and the saved file carry. Count-only like the text:
// no document name, text or URL, and the host lists are counts.
export async function reportEnvelope(manifestVersion, options) {
  const report = (await reportText(manifestVersion, options)).trimEnd().split("\n");
  const o = { ...DEFAULT_OPTIONS, ...(options || {}) };
  const run = (await readRun()) || {};
  const nav = typeof navigator !== "undefined" ? navigator : {};
  const envelope = {
    schemaVersion: SCHEMA_VERSION,
    timestamp: new Date().toISOString(),
    app: { id: "doc-html-translate", version: manifestVersion || "", edition: "browser extension" },
    environment: { platform: nav.platform || "", userAgent: nav.userAgent || "", locale: uiLang() },
    options: {
      autoReflow: !!o.enabledByDefault,
      siteMode: o.siteMode === "allowlist" ? "allowlist" : "all",
      theme: o.theme,
      sourceLang: o.sourceLang,
      ocr: !!o.ocrImages,
      ocrLang: o.ocrLang,
      disabledHosts: (o.disabledHosts || []).length,
      allowedHosts: (o.allowedHosts || []).length,
    },
    lastRun: { format: run.format || "", pages: run.pages || 0, error: run.error || "", at: run.at || "" },
    report,
  };
  return JSON.stringify(envelope, null, 2) + "\n";
}
