// options.js - default on/off, reading theme, source-language hint, image OCR + language
// downloads, and a view of the per-site disable list. Persists the shared `options` object.

import { LANGS, getInstalledLangs, downloadLang } from "./ocr-lang.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { t, initI18n, applyI18n, loadMessages, setUiLang, uiLang } from "./i18n.js";

// UI languages the extension ships, by endonym - the only label that helps a reader who cannot
// read the language currently on screen. Mirrors internal/i18n.Codes on the desktop side.
const UI_LANGUAGES = [
  ["en", "English"], ["ru", "Русский"], ["uk", "Українська"], ["de", "Deutsch"],
  ["it", "Italiano"], ["es", "Español"], ["fr", "Français"], ["pt", "Português"],
  ["ar", "العربية"], ["hi", "हिन्दी"], ["bn", "বাংলা"], ["ur", "اردو"], ["zh", "中文"],
];

const enabledEl = document.getElementById("enabled");
const themeEl = document.getElementById("theme");
const langEl = document.getElementById("lang");
const hostsEl = document.getElementById("hosts");
const ocrImagesEl = document.getElementById("ocr-images");
const ocrLangsEl = document.getElementById("ocr-langs");

// Show the build's date-time version (yy.MMdd.HHmm) so you can tell what you're testing.
const verEl = document.getElementById("ver");
if (verEl) verEl.textContent = "v" + chrome.runtime.getManifest().version;

// Routed through i18n.js so the interface-language override below applies to these strings too,
// not only to the ones Chrome serves by browser language.
const msg = (key, fallback) => t(key, fallback);

// Render the OCR-language manager: installed languages are selectable as the default
// recognition language; others show a Download button (fetch + cache, then re-render).
// While OCR is off we show a call-to-action pointing at the checkbox above instead of a
// greyed-out list (kept in sync with popup.js's renderOcrLangs).
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

function flash(id) {
  const e = document.getElementById(id);
  if (!e) return;
  e.classList.add("show");
  setTimeout(() => e.classList.remove("show"), 900);
}

function renderHosts(hosts) {
  hostsEl.replaceChildren();
  if (!hosts.length) {
    const li = document.createElement("li");
    li.className = "hint";
    li.textContent = "None";
    hostsEl.append(li);
    return;
  }
  for (const h of hosts) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.textContent = "remove";
    btn.style.marginLeft = "0.6rem";
    btn.addEventListener("click", async () => {
      const o = await getOptions();
      o.disabledHosts = o.disabledHosts.filter((x) => x !== h);
      await setOptions(o);
      renderHosts(o.disabledHosts);
    });
    li.textContent = h;
    li.append(btn);
    hostsEl.append(li);
  }
}

async function init() {
  const o = await getOptions();
  enabledEl.checked = o.enabledByDefault;
  themeEl.value = o.theme;
  langEl.value = o.sourceLang;
  renderHosts(o.disabledHosts);

  ocrImagesEl.checked = o.ocrImages;
  ocrImagesEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.ocrImages = ocrImagesEl.checked;
    await setOptions(opts);
    renderOcrLangs();
  });
  renderOcrLangs();

  enabledEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.enabledByDefault = enabledEl.checked;
    await setOptions(opts);
  });
  themeEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.theme = themeEl.value;
    await setOptions(opts);
    flash("saved-theme");
  });
  langEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.sourceLang = langEl.value;
    await setOptions(opts);
  });

  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === "local" && changes.options) renderHosts((changes.options.newValue || {}).disabledHosts || []);
  });

  await initUiLanguage();
}

// initUiLanguage fills the interface-language selector and applies the stored choice. The first
// entry means "follow the browser", which is the default: the browser's language is a good guess,
// just not the only reasonable one - someone on an English work browser still reads in their own
// language.
async function initUiLanguage() {
  const sel = document.getElementById("ui-lang");
  if (!sel) return;

  const stored = await initI18n();
  if (stored) await loadMessages(stored);

  const follow = document.createElement("option");
  follow.value = "";
  follow.textContent = msg("optFollowBrowser", "Follow the browser");
  sel.appendChild(follow);
  for (const [code, name] of UI_LANGUAGES) {
    const o = document.createElement("option");
    o.value = code;
    o.textContent = name;
    sel.appendChild(o);
  }
  sel.value = stored || "";

  sel.addEventListener("change", async () => {
    await setUiLang(sel.value);
    applyI18n(document);
    flash("saved-theme");
  });

  applyI18n(document);
}

init();
