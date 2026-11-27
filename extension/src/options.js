// options.js - the options page: default on/off, interface language and theme, reading theme, source-language
// hint, image OCR + language downloads, the site mode and both site lists, the interface size and controls, the
// settings file and About. Persists the shared `options` object; the page and group memory is `uiState`.
//
// The page is a rail of destinations and a pane (options-nav.js). This file wires the editors of the first
// four and the interface page, and starts the rest.

import { renderOcrLangs } from "./ocr-lang-ui.js";
import { glyph, applyGlyphs } from "./glyphs.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { getOptions, setOptions } from "./options-store.js";
import { t, initI18n, applyI18n, loadMessages, setUiLang, uiLang } from "./i18n.js";
import { applyUiProfile, revealPage } from "./ui-profile.js";
import { initOptionHelp, refreshOptionHelp } from "./ui-help.js";
import { bindGroups } from "./ui-groups.js";
import { initNav } from "./options-nav.js";
import { initSettingsTransfer } from "./options-data.js";
import { renderAbout, initDiagnostics } from "./options-about.js";

// UI languages the extension ships, by endonym - the only label that helps a reader who cannot
// read the language currently on screen. Mirrors internal/i18n.Codes on the desktop side.
const UI_LANGUAGES = [
  ["en", "English"], ["ru", "Русский"], ["uk", "Українська"], ["de", "Deutsch"],
  ["it", "Italiano"], ["es", "Español"], ["fr", "Français"], ["pt", "Português"],
  ["ar", "العربية"], ["hi", "हिन्दी"], ["bn", "বাংলা"], ["ur", "اردو"], ["zh", "中文"],
];

const $ = (id) => document.getElementById(id);
const enabledEl = $("enabled");
const themeEl = $("theme");
const langEl = $("lang");
const hostsEl = $("hosts");
const allowedHostsEl = $("allowed-hosts");
const siteModeEl = $("site-mode");
const ocrImagesEl = $("ocr-images");
const allowRemoteEl = $("allow-remote");
const ocrLangsEl = $("ocr-langs");
const INTERFACE_EDITORS = [["ui-size", "uiSize"], ["text-size", "textSize"], ["input-profile", "inputProfile"], ["chrome-theme", "chromeTheme"]];

// Show the build's date-time version (yy.MMdd.HHmm) so you can tell what you're testing.
$("ver").textContent = `v${chrome.runtime.getManifest().version}`;

// Routed through i18n.js so the interface-language override below applies to these strings too,
// not only to the ones Chrome serves by browser language.
const msg = (key, fallback, ...args) => t(key, fallback, ...args);

// renderLangList is the shared OCR-language renderer (ocr-lang-ui.js); the options page
// labels the block in its own markup, so no extra hint line is needed here.
const renderLangList = () => renderOcrLangs(ocrLangsEl);

// flash shows a ".saved" confirmation. The span is a role="status" live region that is empty
// until it flashes: a status message is announced when its text appears (WCAG 4.1.3), and an
// always-present invisible string would be neither announced again nor readable as absent.
// The wording comes from the span's data-saved-key with an English data-saved-fallback, the
// same degrade-to-English rule every t() call site follows.
function flash(id) {
  const e = $(id);
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
    li.className = "none";
    li.textContent = msg("optNoSites", "None");
    element.append(li);
    return;
  }
  for (const h of hosts) {
    const li = document.createElement("li");
    const name = document.createElement("span");
    name.textContent = h;
    const btn = document.createElement("button");
    btn.type = "button";
    const caption = document.createElement("span");
    caption.textContent = msg("optHostRemove", "Remove");
    btn.append(glyph("action.remove"), caption);
    // The visible word stays first and the site follows it, so the accessible name still contains the
    // visible label and tells apart the rows of one list.
    btn.setAttribute("aria-label", `${caption.textContent}: ${h}`);
    btn.addEventListener("click", async () => {
      const o = await getOptions();
      o[key] = (o[key] || []).filter((x) => x !== h);
      await setOptions(o);
      renderHosts(o[key], element, key);
    });
    li.append(name, btn);
    element.append(li);
  }
}

// renderSourceLangHint fills the source-language hint, whose message marks with {1} where the
// attribute name goes: it stays a <code> literal inside the translated sentence.
function renderSourceLangHint() {
  const code = document.createElement("code");
  code.textContent = "<html lang>";
  const [before, after = ""] = msg("optSourceLangHint",
    "Sets {1} so the browser offers the right translation. Auto-detect works for most documents (EPUBs also use their declared language) - and guesses right more often than not.").split("{1}");
  $("source-lang-hint").replaceChildren(before, code, after);
}

// fillFields puts the stored options into the editors. It sets values only, so no change event fires.
function fillFields(o) {
  enabledEl.checked = o.enabledByDefault;
  siteModeEl.value = o.siteMode === "allowlist" ? "allowlist" : "all";
  themeEl.value = o.theme;
  langEl.value = o.sourceLang;
  ocrImagesEl.checked = o.ocrImages;
  allowRemoteEl.checked = o.allowRemoteContent === true;
  for (const [id, key] of INTERFACE_EDITORS) {
    $(id).value = o[key];
    // A stored value this build does not offer leaves the select blank; show the default it falls back to.
    if ($(id).value !== o[key]) $(id).value = DEFAULT_OPTIONS[key];
  }
  renderHosts(o.disabledHosts || [], hostsEl, "disabledHosts");
  renderHosts(o.allowedHosts || [], allowedHostsEl, "allowedHosts");
}

// update merges one change into the stored options. Each editor commits on its own change, as it always has.
async function update(patch) {
  const opts = await getOptions();
  Object.assign(opts, patch);
  await setOptions(opts);
  return opts;
}

function bindEditors() {
  siteModeEl.addEventListener("change", () => update({ siteMode: siteModeEl.value }));
  ocrImagesEl.addEventListener("change", async () => {
    await update({ ocrImages: ocrImagesEl.checked });
    renderLangList();
  });
  allowRemoteEl.addEventListener("change", () => update({ allowRemoteContent: allowRemoteEl.checked }));
  enabledEl.addEventListener("change", () => update({ enabledByDefault: enabledEl.checked }));
  themeEl.addEventListener("change", async () => {
    await update({ theme: themeEl.value });
    flash("saved-theme");
  });
  langEl.addEventListener("change", () => update({ sourceLang: langEl.value }));
  // The interface settings apply at once, without a restart: the root's attributes change and
  // ui-sizing.css does the rest, so every open component takes the new size or theme together.
  for (const [id, key] of INTERFACE_EDITORS) {
    $(id).addEventListener("change", async () => {
      applyUiProfile(await update({ [key]: $(id).value }));
      if (id === "chrome-theme") flash("saved-chrome-theme");
    });
  }
}

// A change made elsewhere - the popup writes the same options - keeps this page coherent (SETTINGS-NAVIGATION
// 3 rule 1): the editors follow, and the lists and the OCR rows are redrawn only when their data changed.
function followStorage() {
  chrome.storage.onChanged.addListener((changes, area) => {
    if (area !== "local" || !changes.options) return;
    const next = { ...changes.options.newValue };
    const ocrChanged = next.ocrImages !== undefined && next.ocrImages !== ocrImagesEl.checked;
    getOptions().then((o) => {
      fillFields(o);
      applyUiProfile(o);
      if (ocrChanged) renderLangList();
    });
  });
}

// renderSettings redraws what applyI18n cannot reach - the editors' values, the site rows, the OCR rows, the
// source-language hint, the About tables and the help labels - after a language change, an import or a reset.
async function renderSettings() {
  const o = await getOptions();
  fillFields(o);
  applyUiProfile(o);
  $("ui-lang").value = (await chrome.storage.local.get("uiLang")).uiLang || "";
  renderLangList();
  renderSourceLangHint();
  renderAbout();
  refreshOptionHelp();
}

// initUiLanguage fills the interface-language selector and applies the stored choice. The first
// entry means "follow the browser", which is the default: the browser's language is a good guess,
// just not the only reasonable one - someone on an English work browser still reads in their own
// language.
async function initUiLanguage() {
  const sel = $("ui-lang");
  const stored = await initI18n();
  if (stored) await loadMessages(stored);

  const follow = document.createElement("option");
  follow.value = "";
  follow.dataset.i18n = "optFollowBrowser";
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
    // applyI18n reaches only the static markup; the lists built in script are rebuilt so their
    // buttons and labels switch language too.
    await renderSettings();
    flash("saved-ui-lang");
  });
}

async function init() {
  // The interface language and size first: the site lists, the OCR rows, the About tables and the
  // source-language hint are built in script, and under a stored override they would otherwise speak the
  // browser's language - and the window stays hidden until all of it is in place (no flash of English).
  await initUiLanguage();
  const o = await getOptions();
  applyUiProfile(o);
  applyGlyphs(document);
  applyI18n(document);
  fillFields(o);
  renderSourceLangHint();
  renderLangList();
  renderAbout();
  bindEditors();
  followStorage();
  initOptionHelp();
  initDiagnostics();
  const groups = await bindGroups();
  initSettingsTransfer({
    groups,
    onApplied: async () => {
      await initI18n();
      await loadMessages(uiLang());
      applyI18n(document);
      await renderSettings();
    },
  });
  await initNav();
  revealPage();
}

init();
