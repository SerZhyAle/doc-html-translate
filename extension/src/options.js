// options.js - default on/off, reading theme, source-language hint, image OCR + language
// downloads, the site mode and both site lists. Persists the shared `options` object.

import { renderOcrLangs } from "./ocr-lang-ui.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { reportText } from "./diagnostics.js";
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
const allowedHostsEl = document.getElementById("allowed-hosts");
const siteModeEl = document.getElementById("site-mode");
const ocrImagesEl = document.getElementById("ocr-images");
const allowRemoteEl = document.getElementById("allow-remote");
const ocrLangsEl = document.getElementById("ocr-langs");

// Show the build's date-time version (yy.MMdd.HHmm) so you can tell what you're testing.
const verEl = document.getElementById("ver");
if (verEl) verEl.textContent = "v" + chrome.runtime.getManifest().version;

// Routed through i18n.js so the interface-language override below applies to these strings too,
// not only to the ones Chrome serves by browser language.
const msg = (key, fallback) => t(key, fallback);

// renderLangList is the shared OCR-language renderer (ocr-lang-ui.js); the options page
// labels the block in its own markup, so no extra hint line is needed here.
const renderLangList = () => renderOcrLangs(ocrLangsEl);

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}
async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

// flash shows a ".saved" confirmation. The span is a role="status" live region that is empty
// until it flashes: a status message is announced when its text appears (WCAG 4.1.3), and an
// always-present invisible string would be neither announced again nor readable as absent.
// The wording comes from the span's data-saved-key with an English data-saved-fallback, the
// same degrade-to-English rule every t() call site follows.
function flash(id) {
  const e = document.getElementById(id);
  if (!e) return;
  e.textContent = msg(e.dataset.savedKey, e.dataset.savedFallback || "");
  e.classList.add("show");
  clearTimeout(e._hideTimer);
  e._hideTimer = setTimeout(() => { e.classList.remove("show"); e.textContent = ""; }, 900);
}

function renderHosts(hosts, element, key) {
  element.replaceChildren();
  if (!hosts.length) {
    const li = document.createElement("li");
    li.className = "hint";
    li.textContent = "None";
    element.append(li);
    return;
  }
  for (const h of hosts) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.textContent = msg("optHostRemove", "Remove");
    btn.addEventListener("click", async () => {
      const o = await getOptions();
      o[key] = (o[key] || []).filter((x) => x !== h);
      await setOptions(o);
      renderHosts(o[key], element, key);
    });
    li.textContent = h;
    li.append(btn);
    element.append(li);
  }
}

async function init() {
  const o = await getOptions();
  enabledEl.checked = o.enabledByDefault;
  siteModeEl.value = o.siteMode === "allowlist" ? "allowlist" : "all";
  themeEl.value = o.theme;
  langEl.value = o.sourceLang;
  renderHosts(o.disabledHosts || [], hostsEl, "disabledHosts");
  renderHosts(o.allowedHosts || [], allowedHostsEl, "allowedHosts");
  siteModeEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.siteMode = siteModeEl.value;
    await setOptions(opts);
  });

  ocrImagesEl.checked = o.ocrImages;
  ocrImagesEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.ocrImages = ocrImagesEl.checked;
    await setOptions(opts);
    renderLangList();
  });
  renderLangList();

  allowRemoteEl.checked = o.allowRemoteContent === true;
  allowRemoteEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.allowRemoteContent = allowRemoteEl.checked;
    await setOptions(opts);
  });

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
    if (area === "local" && changes.options) {
      const next = changes.options.newValue || {};
      siteModeEl.value = next.siteMode === "allowlist" ? "allowlist" : "all";
      renderHosts(next.disabledHosts || [], hostsEl, "disabledHosts");
      renderHosts(next.allowedHosts || [], allowedHostsEl, "allowedHosts");
    }
  });

  initDiagnostics();
  await initUiLanguage();
}

// initDiagnostics wires the About block's one action: put a short English summary on the
// clipboard so a bug report can carry facts instead of a memory. Nothing is uploaded and no
// permission is needed - a clipboard write from the user's own click requires none.
function initDiagnostics() {
  const btn = document.getElementById("copy-diag");
  if (!btn) return;
  btn.addEventListener("click", async () => {
    const text = await reportText(chrome.runtime.getManifest().version, await getOptions());
    let ok = false;
    try {
      await navigator.clipboard.writeText(text);
      ok = true;
    } catch (e) {
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
      } catch (e2) { ok = false; }
    }
    if (ok) flash("saved-diag");
  });
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
