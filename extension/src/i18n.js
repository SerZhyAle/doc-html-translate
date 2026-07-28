// i18n.js - the extension's interface language.
//
// Chrome serves _locales/<code>/messages.json by browser language, but the viewer is a reading
// surface, not browser chrome: someone running an English browser at work still reads books in
// their own language. So the effective language is an explicit option first, the browser's
// language second (options.js persists `uiLang`; "" means follow the browser).
//
// Every call site keeps an English fallback, so a missing key degrades to English rather than to
// a blank button.

const RTL_UI_LANGS = ["ar", "ur"];

// Cached override, filled by initI18n(). Reading it synchronously matters: applyI18n runs during
// page setup, where an await would paint English first and correct itself a frame later.
let uiLangOverride = "";

// t resolves a key, falling back to the English literal at the call site. Trailing arguments fill
// {1}, {2}, .. placeholders. The braces are our own marker, not Chrome's $1 form, so a message
// file stays valid regardless of whether it declares placeholders.
function t(key, fallback, ...args) {
  let text = fallback;
  try {
    const chosen = uiLang();
    if (chosen && MESSAGES[chosen] && MESSAGES[chosen][key]) text = MESSAGES[chosen][key];
    else text = chrome.i18n.getMessage(key) || fallback;
  } catch {
    text = fallback;
  }
  if (args.length === 0) return text;
  return text.replace(/\{(\d)\}/g, (m, i) => (args[i - 1] === undefined ? m : String(args[i - 1])));
}

// uiLang returns the effective interface language: the stored override, else the browser's UI
// language reduced to its base code, else English.
function uiLang() {
  if (uiLangOverride) return uiLangOverride;
  try {
    return (chrome.i18n.getUILanguage() || "en").split("-")[0];
  } catch {
    return "en";
  }
}

// initI18n loads the stored override. Call it before applyI18n where an await is possible.
async function initI18n() {
  try {
    const st = await chrome.storage.local.get("uiLang");
    uiLangOverride = st && typeof st.uiLang === "string" ? st.uiLang : "";
  } catch {
    uiLangOverride = "";
  }
  return uiLangOverride;
}

// applyI18n fills every tagged node under root and mirrors the chrome for right-to-left
// languages. It sets dir on the element it is given - never on the document body of a converted
// document, whose direction belongs to the document itself.
function applyI18n(root) {
  const scope = root || document;
  scope.querySelectorAll("[data-i18n]").forEach((el) => {
    el.textContent = t(el.dataset.i18n, el.textContent);
  });
  scope.querySelectorAll("[data-i18n-title]").forEach((el) => {
    el.title = t(el.dataset.i18nTitle, el.title);
  });
  scope.querySelectorAll("[data-i18n-ph]").forEach((el) => {
    el.placeholder = t(el.dataset.i18nPh, el.placeholder);
  });
  const rtl = RTL_UI_LANGS.includes(uiLang());
  if (scope === document) {
    document.documentElement.lang = uiLang();
    document.documentElement.dir = rtl ? "rtl" : "ltr";
  } else if (scope.setAttribute) {
    scope.setAttribute("dir", rtl ? "rtl" : "ltr");
  }
}

// MESSAGES holds the override translations. chrome.i18n can only serve the browser's own
// language, so an explicit choice needs its own table; it is filled by loadMessages() from the
// same _locales files, keeping one source of truth.
const MESSAGES = {};

async function loadMessages(code) {
  if (!code || MESSAGES[code]) return;
  try {
    const url = chrome.runtime.getURL(`_locales/${localeDir(code)}/messages.json`);
    const raw = await (await fetch(url)).json();
    const flat = {};
    for (const [k, v] of Object.entries(raw)) flat[k] = v.message;
    MESSAGES[code] = flat;
  } catch {
    // A missing file leaves the language untranslated rather than breaking the page.
  }
}

// localeDir maps a language code to its _locales directory name. Chrome names the Chinese
// directory zh_CN; the rest are plain codes.
function localeDir(code) {
  return code === "zh" ? "zh_CN" : code;
}

// setUiLang stores the override and loads its table. "" restores "follow the browser".
async function setUiLang(code) {
  uiLangOverride = code || "";
  if (code) await loadMessages(code);
  try {
    await chrome.storage.local.set({ uiLang: uiLangOverride });
  } catch {
    // storage is optional - the language still applies for this session
  }
}

export { t, uiLang, initI18n, applyI18n, loadMessages, setUiLang, localeDir, RTL_UI_LANGS };
