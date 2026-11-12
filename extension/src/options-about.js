// options-about.js - the options page's About destination (SETTINGS-NAVIGATION 2.0 section 3 rules 3 and 4):
// the running version, the links, the resource index, the related projects and the diagnostics.
//
// Everything here is drawn from product-inventory.js and the manifest. Opening the page requests nothing,
// and the diagnostics leave the page only when the reader presses a button: the clipboard summary is a
// local write, the file is a local download. No document text is ever in either (diagnostics.js).

import { LINKS, UPDATE_SOURCES, RESOURCES, RELATED } from "./product-inventory.js";
import { reportEnvelope } from "./diagnostics.js";
import { glyph } from "./glyphs.js";
import { getOptions } from "./options-store.js";
import { saveTextFile } from "./save-file.js";
import { t } from "./i18n.js";

const el = (id) => document.getElementById(id);
const msg = (key, fallback, ...args) => t(key, fallback, ...args);

// link builds an anchor for an inventory entry. A web address and a mail address carry the external marker
// (BUTTONS 2.0 section 2 rule 3) and open in a new tab; a file inside the package opens in a new tab too,
// without the marker, because it does not leave the extension.
function link(entry, label) {
  const a = document.createElement("a");
  a.href = entry.href;
  const text = document.createElement("span");
  text.textContent = label;
  a.append(text);
  if (!entry.local) {
    a.className = "ext";
    a.append(glyph("nav.open-external"));
  }
  if (!entry.href.startsWith("mailto:")) {
    a.target = "_blank";
    a.rel = "noopener";
  }
  return a;
}

const labelOf = (entry) => (entry.key ? msg(entry.key, entry.fallback) : entry.name);

function listItem(child) {
  const li = document.createElement("li");
  li.append(child);
  return li;
}

function renderLinks() {
  el("about-links").replaceChildren(...["licence", "notices", "privacy", "help", "site", "repository"]
    .map((id) => listItem(link(LINKS[id], labelOf(LINKS[id])))));
  const note = document.createElement("li");
  note.className = "label";
  note.textContent = msg("optUpdatesNote", "Updates arrive through your browser's extension store; the extension never checks by itself:");
  el("about-updates").replaceChildren(note, ...UPDATE_SOURCES.map((s) => listItem(link(s, labelOf(s)))));
  el("about-support").replaceChildren(listItem(link(LINKS.support, labelOf(LINKS.support))));
}

function cell(className, child) {
  const span = document.createElement("span");
  span.className = className;
  if (typeof child === "string") span.textContent = child; else if (child) span.append(child);
  return span;
}

function renderTables() {
  const res = [];
  for (const r of RESOURCES) {
    const licence = r.licence ? link({ href: r.licence, local: true }, msg("aboutLicence", "Licence")) : null;
    res.push(cell("res-name", r.name), cell("res-what", msg(r.key, r.fallback)), cell("res-link", licence));
  }
  el("res-table").replaceChildren(...res);
  const rel = [];
  for (const r of RELATED) {
    rel.push(cell("res-name", r.name), cell("res-what", msg(r.key, r.fallback)), cell("res-link", link(r, msg("relOpen", "Open"))));
  }
  el("rel-table").replaceChildren(...rel);
}

// renderAbout (re)draws everything the language shows: it runs once at start and again after a language
// change, because the labels are built here and applyI18n reaches only static markup.
export function renderAbout() {
  const version = `v${chrome.runtime.getManifest().version}`;
  el("about-ver").textContent = version;
  renderLinks();
  renderTables();
}

function setDiagStatus(text) { el("diag-status").textContent = text; }

// The summary is the JSON diagnostic envelope (DIAGNOSTIC-REPORT 8 A) with the readable lines inside it,
// English whatever the interface language is, so the author reads one format.
async function summary() {
  return reportEnvelope(chrome.runtime.getManifest().version, await getOptions());
}

// initDiagnostics wires the two actions. Copy puts the summary on the clipboard - a write from the reader's own
// click needs no permission. Save hands the same envelope over as a file for a mail attachment or a bug report.
export function initDiagnostics() {
  el("copy-diag").addEventListener("click", async () => {
    const text = await summary();
    let ok = false;
    try {
      await navigator.clipboard.writeText(text);
      ok = true;
    } catch {
      // The clipboard API can be refused; fall back to a hidden textarea + execCommand.
      try {
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        ok = document.execCommand("copy");
        document.body.removeChild(ta);
      } catch { ok = false; }
    }
    setDiagStatus(ok ? msg("optCopied", "Copied") : "");
  });
  el("save-diag").addEventListener("click", async () => {
    try {
      saveTextFile("doc-html-translate-diagnostics.json", await summary(), "application/json;charset=utf-8");
      setDiagStatus(msg("optDiagSaved", "Diagnostics file downloaded."));
    } catch (error) { setDiagStatus(error.message); }
  });
}
