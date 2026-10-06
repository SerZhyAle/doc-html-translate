// popup.js - global on/off and the active per-site mode. Writes the shared `options`
// object to storage; background.js rebuilds the DNR rules on the storage change.
// The popup also reflects the active tab: a supported document gets a one-click
// open in the reader, and a site whose documents are left alone says so.

import { renderOcrLangs } from "./ocr-lang-ui.js";
import { t, initI18n, applyI18n, loadMessages, uiLang } from "./i18n.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { applyGlyphs } from "./glyphs.js";
import { siteHost } from "./site-host.js";
import { allowlistMode, hostInList, siteEnabled, setSiteEnabled } from "./site-mode.js";
import { isInterceptableUrl } from "./intercept.js";

const globalEl = document.getElementById("global");
const siteEl = document.getElementById("site");
const hostEl = document.getElementById("host");
const modeEl = document.getElementById("site-mode");
const membershipEl = document.getElementById("site-membership");
const ocrImagesEl = document.getElementById("ocr-images");
const ocrLangsDetailsEl = document.getElementById("ocr-langs-details");
const ocrLangsEl = document.getElementById("ocr-langs");
const tabStateEl = document.getElementById("tab-state");
const ctaEl = document.getElementById("open-pdf");

// Show the build's date-time version (yy.MMdd.HHmm) so you can tell what you're testing.
const verEl = document.getElementById("ver");
if (verEl) verEl.textContent = "v" + chrome.runtime.getManifest().version;

// Routed through i18n.js so the options page's interface-language override reaches these too.
const msg = (key, fallback, ...args) => t(key, fallback, ...args);

// renderLangList is the shared OCR-language renderer (ocr-lang-ui.js); the popup keeps its
// compact heading line above the rows.
const renderLangList = () => renderOcrLangs(ocrLangsEl, { hint: msg("ocrLangsHint", "Recognition language (English is built-in).") });

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}
async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

// activeTab fetches the tab the popup is acting on; everything the popup says about "this
// site" and "this document" is derived from its URL alone.
async function activeTab() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    return tab || null;
  } catch { /* no tab access */ }
  return null;
}

async function init() {
  await initI18n();
  await loadMessages(uiLang());
  applyI18n(document);
  applyGlyphs(document);
  document.getElementById("convert-note").textContent = msg("popupConvertNote",
    "Off by default. Or right-click a document link and choose “{1}”.",
    msg("convertDocMenu", "Convert with doc-html-translate"));
  const opts = await getOptions();
  const tab = await activeTab();
  const tabUrl = (tab && tab.url) || "";
  // activeHost is the site the per-site switch acts on (site-host.js): on the viewer's own
  // tab, the host of the document it shows.
  const host = tabUrl ? siteHost(tabUrl, chrome.runtime.getURL("src/viewer.html")) : "";

  globalEl.checked = opts.enabledByDefault;
  modeEl.textContent = allowlistMode(opts)
    ? msg("siteModeAllowlist", "Only listed sites")
    : msg("siteModeAll", "All sites except disabled sites");
  const showMembership = (o) => {
    membershipEl.textContent = !host ? "" : allowlistMode(o)
      ? hostInList(o.allowedHosts, host)
        ? msg("popupInAllowlist", "In the allowlist") : msg("popupNotInAllowlist", "Not in the allowlist")
      : hostInList(o.disabledHosts, host)
        ? msg("popupInDisablelist", "In the disable list") : msg("popupNotInDisablelist", "Not in the disable list");
  };
  showMembership(opts);

  ocrImagesEl.checked = opts.ocrImages;
  ocrImagesEl.addEventListener("change", async () => {
    const o = await getOptions();
    o.ocrImages = ocrImagesEl.checked;
    await setOptions(o);
    // Keep the popup compact when it opens, but reveal the next OCR step when
    // the user has just opted in.
    if (ocrImagesEl.checked) ocrLangsDetailsEl.open = true;
    renderLangList();
  });
  renderLangList();

  // The tab's own story first: a supported document gets the one-click open (instead of
  // routing its reader through the empty viewer state), a site whose documents are left
  // alone is told so where the switch that controls it sits right below.
  const docUrl = isInterceptableUrl(tabUrl) ? tabUrl : "";
  if (docUrl) {
    tabStateEl.hidden = false;
    tabStateEl.textContent = msg("popupTabDocument", "The current tab is a supported document.");
    ctaEl.textContent = msg("popupOpenInReader", "Open it in the reader");
  } else if (host && opts.enabledByDefault && !siteEnabled(opts, host)) {
    tabStateEl.hidden = false;
    tabStateEl.textContent = msg("popupTabOff", "Reflow is off on {1} - documents open as usual.", host);
  }

  ctaEl.addEventListener("click", () => {
    // A document tab opens in the reader directly; anything else opens the viewer's empty
    // state, where "Open a PDF file" lives (the OS file dialog needs a user gesture on the
    // viewer page itself).
    const url = docUrl
      ? `${chrome.runtime.getURL("src/viewer.html")}?file=${encodeURIComponent(docUrl)}`
      : chrome.runtime.getURL("src/viewer.html");
    chrome.tabs.create({ url });
    window.close();
  });

  if (host) {
    hostEl.textContent = host;
    siteEl.disabled = !opts.enabledByDefault;
    siteEl.checked = siteEnabled(opts, host);
  } else {
    hostEl.textContent = msg("popupNoSite", "(not a website)");
    siteEl.disabled = true;
  }

  globalEl.addEventListener("change", async () => {
    const o = await getOptions();
    o.enabledByDefault = globalEl.checked;
    await setOptions(o);
    siteEl.disabled = !o.enabledByDefault || !host;
    siteEl.checked = siteEnabled(o, host);
  });

  siteEl.addEventListener("change", async () => {
    if (!host) return;
    const o = await getOptions();
    setSiteEnabled(o, host, siteEl.checked);
    await setOptions(o);
    showMembership(o);
  });

  document.getElementById("opts").addEventListener("click", (e) => {
    e.preventDefault();
    chrome.runtime.openOptionsPage();
  });
}

init();
