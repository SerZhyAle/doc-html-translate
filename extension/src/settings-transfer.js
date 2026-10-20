// A deliberately small, versioned settings file. Never serialize the whole storage area:
// it also holds reading positions, diagnostics and OCR cache metadata.
import { DEFAULT_OPTIONS } from "./defaults.js";
import {
  UI_SIZES, TEXT_SIZES, INPUT_PROFILES, CHROME_THEMES, DEFAULT_UI_STATE, normalizeUiState, pick,
} from "./ui-settings.js";

export const SETTINGS_FORMAT = "doc-html-translate-extension-settings";
// Version 2 added the interface settings (size, text size, controls, interface theme) and the
// remembered page and groups. A version 1 file still imports: what it lacks takes its default.
export const SETTINGS_VERSION = 2;
const READABLE_VERSIONS = [1, 2];
const OPTION_KEYS = Object.keys(DEFAULT_OPTIONS);
// What a version 1 file carried. Anything beyond these is optional on import.
const INTERFACE_KEYS = ["uiSize", "textSize", "inputProfile", "chromeTheme"];
const REQUIRED_KEYS = OPTION_KEYS.filter((key) => !INTERFACE_KEYS.includes(key));
const UI_LANGS = new Set(["", "en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh"]);
const SOURCE_LANGS = new Set(["auto", "en", "ru", "uk", "fr", "de", "es", "it", "pt", "zh", "ja", "ko", "ar"]);
const THEMES = new Set(["light", "sepia", "dark", "night"]);
const OCR_LANGS = new Set(["eng", "rus", "ukr", "jpn", "jpn_vert", "deu", "fra", "spa", "ita", "por", "pol", "chi_sim", "chi_tra", "kor"]);
const HOST = /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

function object(value, name) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`${name} must be an object`);
}

// The interface settings are display preferences: an out-of-range value (a newer file, a hand
// edit) falls back to the default instead of refusing the whole file, unlike the behavioural
// options, whose wrong value would change what the extension does.
function interfaceSettings(options) {
  return {
    uiSize: pick(options.uiSize, UI_SIZES, DEFAULT_OPTIONS.uiSize),
    textSize: pick(options.textSize, TEXT_SIZES, DEFAULT_OPTIONS.textSize),
    inputProfile: pick(options.inputProfile, INPUT_PROFILES, DEFAULT_OPTIONS.inputProfile),
    chromeTheme: pick(options.chromeTheme, CHROME_THEMES, DEFAULT_OPTIONS.chromeTheme),
  };
}

// The part of the remembered state a file carries: the page and the group states. Viewports are
// per-device scroll positions and stay out, like reading positions.
function exportedUiState(state) {
  const { page, groups } = normalizeUiState(state);
  return { page, groups };
}

export function validateSettingsFile(value) {
  object(value, "File");
  if (value.format !== SETTINGS_FORMAT) throw new Error("Unrecognized settings format");
  if (!READABLE_VERSIONS.includes(value.version)) throw new Error(`Unsupported settings version: ${String(value.version)}`);
  const allowedTop = ["format", "options", "uiLang", "version", "ui"];
  const unexpected = Object.keys(value).filter((key) => !allowedTop.includes(key));
  if (unexpected.length || ["format", "options", "uiLang", "version"].some((key) => !(key in value))) {
    throw new Error("Unexpected fields in settings file");
  }
  object(value.options, "Options");
  const options = value.options;
  const keys = Object.keys(options);
  if (keys.some((key) => !OPTION_KEYS.includes(key)) || REQUIRED_KEYS.some((key) => !(key in options))) {
    throw new Error("Settings fields are missing or unrecognized");
  }
  for (const key of ["enabledByDefault", "ocrImages", "allowRemoteContent"]) {
    if (typeof options[key] !== "boolean") throw new Error(`${key} must be true or false`);
  }
  if (!["all", "allowlist"].includes(options.siteMode)) throw new Error("Invalid site mode");
  if (!SOURCE_LANGS.has(options.sourceLang)) throw new Error("Invalid source language");
  if (!THEMES.has(options.theme)) throw new Error("Invalid reading theme");
  if (typeof options.ocrLang !== "string" || !OCR_LANGS.has(options.ocrLang)) {
    throw new Error("Invalid OCR language");
  }
  for (const key of ["disabledHosts", "allowedHosts"]) {
    if (!Array.isArray(options[key]) || options[key].length > 10000 ||
        options[key].some((host) => typeof host !== "string" || !HOST.test(host) || host !== host.toLowerCase()) ||
        new Set(options[key]).size !== options[key].length) throw new Error(`Invalid ${key} list`);
  }
  if (!UI_LANGS.has(value.uiLang)) throw new Error("Invalid interface language");
  if ("ui" in value) object(value.ui, "Interface state");
  return {
    options: { ...structuredClone(options), ...interfaceSettings(options) },
    uiLang: value.uiLang,
    ui: exportedUiState(value.ui),
  };
}

export function makeSettingsFile(storedOptions, uiLang = "", uiState = DEFAULT_UI_STATE) {
  const options = Object.fromEntries(OPTION_KEYS.map((key) => [key, storedOptions?.[key] ?? DEFAULT_OPTIONS[key]]));
  const file = {
    format: SETTINGS_FORMAT, version: SETTINGS_VERSION,
    options: { ...options, ...interfaceSettings(options) },
    uiLang,
    ui: exportedUiState(uiState),
  };
  validateSettingsFile(file);
  return file;
}

export function changedSettings(before, after) {
  const changed = OPTION_KEYS.filter((key) => JSON.stringify(before.options[key]) !== JSON.stringify(after.options[key]));
  if (before.uiLang !== after.uiLang) changed.push("uiLang");
  if (JSON.stringify(before.ui) !== JSON.stringify(after.ui)) changed.push("ui");
  return changed;
}
