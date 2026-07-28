// popup.js - global on/off and per-site disable. Writes the shared `options`
// object to storage; background.js rebuilds the DNR rules on the storage change.

import { LANGS, getInstalledLangs, downloadLang } from "./ocr-lang.js";
import { t, initI18n, applyI18n, loadMessages, uiLang } from "./i18n.js";
import { DEFAULT_OPTIONS } from "./defaults.js";

const globalEl = document.getElementById("global");
const siteEl = document.getElementById("site");
const hostEl = document.getElementById("host");
const ocrImagesEl = document.getElementById("ocr-images");
const ocrLangsDetailsEl = document.getElementById("ocr-langs-details");
const ocrLangsEl = document.getElementById("ocr-langs");

// Show the build's date-time version (yy.MMdd.HHmm) so you can tell what you're testing.
const verEl = document.getElementById("ver");
if (verEl) verEl.textContent = "v" + chrome.runtime.getManifest().version;

// Routed through i18n.js so the options page's interface-language override reaches these too.
const msg = (key, fallback) => t(key, fallback);

// Render the nested OCR-language list: installed languages are selectable (radio),
// others show a Download button that fetches + caches the language, then re-renders.
// While "Use OCR for images" is off we show a call-to-action pointing at the switch
// above instead of a greyed-out, dead-looking list (which reads as "unavailable").
async function renderOcrLangs() {
  const o = await getOptions();
  ocrLangsEl.replaceChildren();
  ocrLangsEl.classList.remove("disabled");

  if (!o.ocrImages) {
    const off = document.createElement("div");
    off.className = "hint";
    off.textContent = msg("ocrOffHint", "Turn on to recognize text in images - then pick or download a language (English is built-in).");
    ocrLangsEl.append(off);
    return;
  }

  const installed = await getInstalledLangs();
  const hint = document.createElement("div");
  hint.className = "hint";
  hint.textContent = msg("ocrLangsHint", "Recognition language (English is built-in).");
  ocrLangsEl.append(hint);

  for (const lang of LANGS) {
    const row = document.createElement("div");
    row.className = "ocr-lang";
    if (installed.includes(lang.code)) {
      const id = `ocrlang-${lang.code}`;
      const radio = document.createElement("input");
      radio.type = "radio";
      radio.name = "ocrLang";
      radio.id = id;
      radio.checked = o.ocrLang === lang.code;
      radio.addEventListener("change", async () => {
        const oo = await getOptions();
        oo.ocrLang = lang.code;
        await setOptions(oo);
      });
      const label = document.createElement("label");
      label.htmlFor = id;
      label.textContent = `${lang.name} (${msg("ocrInstalled", "installed")})`;
      row.append(radio, label);
    } else {
      const name = document.createElement("span");
      name.textContent = lang.name;
      const btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = msg("ocrDownload", "Download");
      btn.addEventListener("click", async () => {
        btn.disabled = true;
        try {
          await downloadLang(lang.code, (m) => {
            if (m && typeof m.progress === "number") {
              btn.textContent = `${msg("ocrDownloading", "Downloading")} ${Math.round(m.progress * 100)}%`;
            }
          });
          await renderOcrLangs();
        } catch (e) {
          console.error("language download failed", e);
          btn.textContent = msg("ocrDownload", "Download");
          btn.disabled = false;
        }
      });
      row.append(name, btn);
    }
    ocrLangsEl.append(row);
  }
}

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}
async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

async function activeHost() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (tab && tab.url) return new URL(tab.url).hostname;
  } catch { /* no host access */ }
  return "";
}

async function init() {
  await initI18n();
  await loadMessages(uiLang());
  applyI18n(document);
  const opts = await getOptions();
  const host = await activeHost();

  globalEl.checked = opts.enabledByDefault;

  ocrImagesEl.checked = opts.ocrImages;
  ocrImagesEl.addEventListener("change", async () => {
    const o = await getOptions();
    o.ocrImages = ocrImagesEl.checked;
    await setOptions(o);
    // Keep the popup compact when it opens, but reveal the next OCR step when
    // the user has just opted in.
    if (ocrImagesEl.checked) ocrLangsDetailsEl.open = true;
    renderOcrLangs();
  });
  renderOcrLangs();

  if (host) {
    hostEl.textContent = host;
    siteEl.disabled = !opts.enabledByDefault;
    siteEl.checked = opts.enabledByDefault && !opts.disabledHosts.includes(host);
  } else {
    hostEl.textContent = "(not a website)";
    siteEl.disabled = true;
  }

  globalEl.addEventListener("change", async () => {
    const o = await getOptions();
    o.enabledByDefault = globalEl.checked;
    await setOptions(o);
    siteEl.disabled = !o.enabledByDefault || !host;
    siteEl.checked = o.enabledByDefault && host && !o.disabledHosts.includes(host);
  });

  siteEl.addEventListener("change", async () => {
    if (!host) return;
    const o = await getOptions();
    const set = new Set(o.disabledHosts);
    if (siteEl.checked) set.delete(host);
    else set.add(host);
    o.disabledHosts = [...set];
    await setOptions(o);
  });

  document.getElementById("open-pdf").addEventListener("click", () => {
    // Open the viewer's empty state in a new tab; the user clicks "Open a PDF file"
    // there (the OS file dialog needs a user gesture on the viewer page itself).
    chrome.tabs.create({ url: chrome.runtime.getURL("src/viewer.html") });
    window.close();
  });

  document.getElementById("opts").addEventListener("click", (e) => {
    e.preventDefault();
    chrome.runtime.openOptionsPage();
  });
}

init();
