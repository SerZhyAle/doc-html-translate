// A deliberately small, versioned settings file. Never serialize the whole storage area:
// it also holds reading positions, diagnostics and OCR cache metadata.
import { DEFAULT_OPTIONS } from "./defaults.js";

export const SETTINGS_FORMAT = "doc-html-translate-extension-settings";
export const SETTINGS_VERSION = 1;
const OPTION_KEYS = Object.keys(DEFAULT_OPTIONS);
const UI_LANGS = new Set(["", "en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh"]);
const SOURCE_LANGS = new Set(["auto", "en", "ru", "uk", "fr", "de", "es", "it", "pt", "zh", "ja", "ko", "ar"]);
const THEMES = new Set(["light", "sepia", "dark", "night"]);
const OCR_LANGS = new Set(["eng", "rus", "ukr", "jpn", "jpn_vert", "deu", "fra", "spa", "ita", "por", "pol", "chi_sim", "kor"]);
const HOST = /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

function object(value, name) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`${name} must be an object`);
}

export function validateSettingsFile(value) {
  object(value, "File");
  if (value.format !== SETTINGS_FORMAT) throw new Error("Unrecognized settings format");
  if (value.version !== SETTINGS_VERSION) throw new Error(`Unsupported settings version: ${String(value.version)}`);
  if (Object.keys(value).sort().join() !== ["format", "options", "uiLang", "version"].sort().join()) {
    throw new Error("Unexpected fields in settings file");
  }
  object(value.options, "Options");
  const options = value.options;
  if (Object.keys(options).sort().join() !== [...OPTION_KEYS].sort().join()) {
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
  return { options: structuredClone(options), uiLang: value.uiLang };
}

export function makeSettingsFile(storedOptions, uiLang = "") {
  const options = Object.fromEntries(OPTION_KEYS.map((key) => [key, storedOptions?.[key] ?? DEFAULT_OPTIONS[key]]));
  const file = { format: SETTINGS_FORMAT, version: SETTINGS_VERSION, options, uiLang };
  validateSettingsFile(file);
  return file;
}

export function changedSettings(before, after) {
  const changed = OPTION_KEYS.filter((key) => JSON.stringify(before.options[key]) !== JSON.stringify(after.options[key]));
  if (before.uiLang !== after.uiLang) changed.push("uiLang");
  return changed;
}
