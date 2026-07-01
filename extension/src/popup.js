// popup.js - global on/off and per-site disable. Writes the shared `options`
// object to storage; background.js rebuilds the DNR rules on the storage change.

import { LANGS, getInstalledLangs, downloadLang } from "./ocr-lang.js";
import { DEFAULT_OPTIONS } from "./defaults.js";

const globalEl = document.getElementById("global");
const siteEl = document.getElementById("site");
const hostEl = document.getElementById("host");
const ocrImagesEl = document.getElementById("ocr-images");
const ocrLangsEl = document.getElementById("ocr-langs");

const msg = (key, fallback) => {
  try { return chrome.i18n.getMessage(key) || fallback; } catch { return fallback; }
};

// Render the nested OCR-language list: installed languages are selectable (radio),
// others show a Download button that fetches + caches the language, then re-renders.
// The whole block is disabled while "Use OCR for images" is off.
async function renderOcrLangs() {
  const o = await getOptions();
  const installed = await getInstalledLangs();
  ocrLangsEl.replaceChildren();
  ocrLangsEl.classList.toggle("disabled", !o.ocrImages);

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
      radio.disabled = !o.ocrImages;
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
      btn.disabled = !o.ocrImages;
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
  const opts = await getOptions();
  const host = await activeHost();

  globalEl.checked = opts.enabledByDefault;

  ocrImagesEl.checked = opts.ocrImages;
  ocrImagesEl.addEventListener("change", async () => {
    const o = await getOptions();
    o.ocrImages = ocrImagesEl.checked;
    await setOptions(o);
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
